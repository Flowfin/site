// SPDX-License-Identifier: AGPL-3.0-or-later

// Package pins holds the versions this repository fixes that no updater
// watches, and the run that says which of them has fallen behind.
//
// The updater configuration covers two ecosystems, the actions and the module
// graph. A version fetched at run time inside a workflow step is in neither, and
// neither is the digest a container is built from. Those are declared in one
// file at the root of the tree, so the set is countable rather than scattered
// through steps, and so the question of what this repository pins is answered by
// reading one file instead of five.
//
// The run reports and does not bump. A version and its checksum are one fact,
// and a machine that rewrote both in the same commit has proved nothing about
// the bytes it just trusted. What comes out is the evidence for a change
// somebody makes.
//
// Three states stay distinguishable, because collapsing them is what makes a
// stale pin invisible. A pin that matches its upstream is current. A pin whose
// upstream has moved is behind, and the run prints both values. A pin whose
// upstream could not be read is unresolved, which is a failure rather than a
// pass: a comparison that fetched nothing and reported agreement is the thing
// this run exists to prevent.
package pins

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// File is the one file, at the root of the tree.
const File = "pins.json"

// The registries a pin can be resolved against. A pin naming anything else is
// refused when the file is read rather than at the moment the run reaches it,
// because a file that parses and then fails halfway through a network run is a
// file nobody can check before pushing.
const (
	NPM       = "npm"
	PyPI      = "pypi"
	DockerHub = "docker-hub"
)

// Pin is one declared version.
//
// Reads and Repeats are the two ways a consumer can carry a pin, and they are
// separate fields because they are checked in opposite directions. A file that
// reads this one must not carry the version at all; a file that repeats it must
// carry it exactly. Nothing repeats a version because it would rather: a
// container image reference is fixed in a FROM instruction, which takes no
// indirection, and the alternative to naming that here is a pin the set does not
// know about.
type Pin struct {
	ID       string   `json:"id"`
	Registry string   `json:"registry"`
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Checksum string   `json:"checksum"`
	Reads    []string `json:"reads"`
	Repeats  []string `json:"repeats"`
	Why      string   `json:"why"`
}

// Reference is what a consumer that repeats this pin has to carry. It is
// derived rather than declared, because a fourth copy of the same fact is the
// copy that disagrees with the other three.
func (p Pin) Reference() string {
	if p.Registry != DockerHub {
		return p.Version
	}
	return strings.TrimPrefix(p.Name, "library/") + ":" + p.Version + "@" + p.Checksum
}

// Upstream is what a registry answered.
type Upstream struct {
	Version  string
	Checksum string
}

// A Resolver asks a registry what the current value is. It is a parameter so
// that the suite can prove the run's behaviour on every answer, including the
// answers a network cannot be asked to produce on demand.
type Resolver func(Pin) (Upstream, error)

var (
	known          = map[string]bool{NPM: true, PyPI: true, DockerHub: true}
	fieldNames     = map[string]bool{"id": true, "registry": true, "name": true, "version": true, "checksum": true, "reads": true, "repeats": true, "why": true}
	sha256Checksum = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

// Load reads the file and refuses everything it does not understand. Each
// refusal names the pin it is about, because a message saying only that the file
// is invalid makes the next person bisect it by hand.
func Load(root string) ([]Pin, error) {
	name := filepath.Join(root, File)
	body, err := os.ReadFile(filepath.Clean(name))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", File, err)
	}

	// Read once as raw rows so that an unknown field is reported against the
	// row that carries it, and once as pins.
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("%s is not the array of objects this file has to be: %w", File, err)
	}
	var declared []Pin
	if err := json.Unmarshal(body, &declared); err != nil {
		return nil, fmt.Errorf("%s does not carry the fields this file has to carry: %w", File, err)
	}

	seen := map[string]bool{}
	for i, row := range rows {
		p := declared[i]
		where := p.ID
		if where == "" {
			where = fmt.Sprintf("the pin at position %d", i)
		}
		for field := range row {
			if !fieldNames[field] {
				return nil, fmt.Errorf("%s carries the field %q, which is not a field of this file", where, field)
			}
		}
		for field, value := range map[string]string{"id": p.ID, "registry": p.Registry, "name": p.Name, "version": p.Version, "why": p.Why} {
			if strings.TrimSpace(value) == "" {
				return nil, fmt.Errorf("%s has no %s, and every field but the checksum and the two consumer lists is required", where, field)
			}
		}
		if !known[p.Registry] {
			return nil, fmt.Errorf("%s names the registry %q, which nothing here can resolve", where, p.Registry)
		}
		if seen[p.ID] {
			return nil, fmt.Errorf("%s is declared twice, and a pin declared twice is two answers to one question", where)
		}
		seen[p.ID] = true
		if p.Checksum != "" && !sha256Checksum.MatchString(p.Checksum) {
			return nil, fmt.Errorf("%s carries the checksum %q, which is not a sha256 digest", where, p.Checksum)
		}
		if p.Registry == DockerHub && p.Checksum == "" {
			return nil, fmt.Errorf("%s is an image and carries no checksum, so what it pins is a tag somebody else can move", where)
		}
		if len(p.Reads) == 0 && len(p.Repeats) == 0 {
			return nil, fmt.Errorf("%s names no file that consumes it, so nothing in this tree is held to it", where)
		}
	}
	if len(declared) == 0 {
		return nil, fmt.Errorf("%s declares no pin, and an empty set of pins is not the same statement as a set nobody wrote", File)
	}
	return declared, nil
}

// Agree checks every declared consumer against the pin it consumes, and needs
// no network. A file that reads the pin must not carry the version literal,
// because a version in two places is a version that disagrees with itself; a
// file that repeats it must carry the derived reference exactly.
func Agree(root string, declared []Pin) []string {
	var wrong []string
	for _, p := range declared {
		for _, name := range p.Reads {
			body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
			if err != nil {
				wrong = append(wrong, fmt.Sprintf("%s is declared as reading %s and could not be read: %v", p.ID, name, err))
				continue
			}
			if strings.Contains(string(body), p.Version) {
				wrong = append(wrong, fmt.Sprintf("%s reads %s and also carries the version %s written out, which is a second copy of the fact %s exists to hold", name, File, p.Version, File))
			}
		}
		for _, name := range p.Repeats {
			body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
			if err != nil {
				wrong = append(wrong, fmt.Sprintf("%s is declared as repeating into %s and it could not be read: %v", p.ID, name, err))
				continue
			}
			if !strings.Contains(string(body), p.Reference()) {
				wrong = append(wrong, fmt.Sprintf("%s does not carry %s, which is what %s declares for %s", name, p.Reference(), File, p.ID))
			}
		}
	}
	return wrong
}

// Run reads the file, checks the consumers, resolves every pin and reports what
// it found. It writes nothing anywhere: what it produces is the evidence for a
// change somebody makes.
func Run(root string, resolve Resolver, out io.Writer) error {
	declared, err := Load(root)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "pins: %d declared in %s\n", len(declared), File)

	if wrong := Agree(root, declared); len(wrong) > 0 {
		for _, w := range wrong {
			fmt.Fprintf(out, "  %s\n", w)
		}
		return fmt.Errorf("pins: %d consumer(s) disagree with %s, so there is nothing worth comparing against an upstream", len(wrong), File)
	}

	behind, unresolved := 0, 0
	for _, p := range declared {
		up, err := resolve(p)
		if err != nil {
			unresolved++
			fmt.Fprintf(out, "  %s: UNRESOLVED, pinned %s, %s could not be read: %v\n", p.ID, pinned(p), p.Registry, err)
			continue
		}
		got, want := answer(p, up), pinned(p)
		if got == want {
			fmt.Fprintf(out, "  %s: current at %s, %s says the same\n", p.ID, want, p.Registry)
			continue
		}
		behind++
		fmt.Fprintf(out, "  %s: BEHIND, pinned %s, %s says %s\n", p.ID, want, p.Registry, got)
	}

	compared := len(declared) - unresolved
	fmt.Fprintf(out, "%d pin(s) declared, %d compared, %d behind, %d unresolved.\n",
		len(declared), compared, behind, unresolved)

	switch {
	case compared == 0:
		return fmt.Errorf("pins: no upstream could be read, so this run compared nothing and is not a run that found everything current")
	case unresolved > 0:
		return fmt.Errorf("pins: %d of %d pin(s) could not be resolved, and an unresolved pin is not a current one", unresolved, len(declared))
	case behind > 0:
		return fmt.Errorf("pins: %d pin(s) are behind their upstream", behind)
	}
	return nil
}

// pinned and answer are the two halves of one comparison, and which value it is
// depends on what the pin fixes. An image is pinned by digest, so a tag rebuilt
// under the same name is exactly the drift that matters and a version string
// would not see it.
func pinned(p Pin) string {
	if p.Registry == DockerHub {
		return p.Checksum
	}
	return p.Version
}

func answer(p Pin, up Upstream) string {
	if p.Registry == DockerHub {
		return up.Checksum
	}
	return up.Version
}

// Registries is the resolver the scheduled run uses. Each registry answers a
// plain GET with JSON, which is why this needs no client library and no
// dependency.
func Registries(p Pin) (Upstream, error) {
	switch p.Registry {
	case NPM:
		var body struct {
			Version string `json:"version"`
		}
		if err := get("https://registry.npmjs.org/"+p.Name+"/latest", &body); err != nil {
			return Upstream{}, err
		}
		if body.Version == "" {
			return Upstream{}, fmt.Errorf("the registry answered without a version")
		}
		return Upstream{Version: body.Version}, nil
	case PyPI:
		var body struct {
			Info struct {
				Version string `json:"version"`
			} `json:"info"`
		}
		if err := get("https://pypi.org/pypi/"+p.Name+"/json", &body); err != nil {
			return Upstream{}, err
		}
		if body.Info.Version == "" {
			return Upstream{}, fmt.Errorf("the registry answered without a version")
		}
		return Upstream{Version: body.Info.Version}, nil
	case DockerHub:
		var body struct {
			Digest string `json:"digest"`
		}
		if err := get("https://hub.docker.com/v2/repositories/"+p.Name+"/tags/"+p.Version, &body); err != nil {
			return Upstream{}, err
		}
		if body.Digest == "" {
			return Upstream{}, fmt.Errorf("the registry answered without a digest for tag %s", p.Version)
		}
		return Upstream{Version: p.Version, Checksum: body.Digest}, nil
	}
	return Upstream{}, fmt.Errorf("nothing here resolves the registry %q", p.Registry)
}

func get(address string, into any) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(address)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %s", address, resp.Status)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(into)
}
