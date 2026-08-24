// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the pins file, the consumer check and the run.
//
// Every refusal is proved by a fixture that trips exactly it, and every state
// the run can report is proved by a resolver that produces that state on
// demand. A network cannot be asked for an unreachable registry when a test
// wants one, which is why the resolver is a parameter: the three states this
// run exists to keep apart are the three that are hardest to observe by waiting
// for them to happen.
package pins

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const goodFile = `[
  {
    "id": "prettier",
    "registry": "npm",
    "name": "prettier",
    "version": "3.9.6",
    "checksum": "",
    "reads": [".github/workflows/prettier.yml"],
    "repeats": [],
    "why": "The formatter is fetched at run time and no updater sees it."
  },
  {
    "id": "golang-image",
    "registry": "docker-hub",
    "name": "library/golang",
    "version": "1.26.5-bookworm",
    "checksum": "sha256:6c5605ab3a9a9fb3c4eafe5b3d63cdbf3881caf113262b67862547b54a9db599",
    "reads": [],
    "repeats": ["Dockerfile"],
    "why": "The container builds from this image and the digest is what makes it the same build twice."
  }
]
`

const readerThatDoesNotRepeat = `name: Formatting
jobs:
  prettier:
    steps:
      - run: |
          version="$(jq -er '.[] | select(.id == "prettier") | .version' pins.json)"
          echo "PRETTIER_VERSION=${version}" >> "$GITHUB_ENV"
`

const dockerfileThatAgrees = `FROM golang:1.26.5-bookworm@sha256:6c5605ab3a9a9fb3c4eafe5b3d63cdbf3881caf113262b67862547b54a9db599
WORKDIR /src
`

// tree writes a root carrying the file and the two consumers it declares.
func tree(t *testing.T, file, reader, dockerfile string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".github", "workflows"), 0o755); err != nil {
		t.Fatalf("preparing the workflow directory: %v", err)
	}
	for name, body := range map[string]string{
		File: file,
		filepath.Join(".github", "workflows", "prettier.yml"): reader,
		"Dockerfile": dockerfile,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	return root
}

func good(t *testing.T) string {
	t.Helper()
	return tree(t, goodFile, readerThatDoesNotRepeat, dockerfileThatAgrees)
}

// Each thing the file may not be, in the smallest form somebody would actually
// write, with the fragment the message has to carry so that a reader knows
// which row to open rather than bisecting the file by hand.
func TestLoadRefusesEachThingItDoesNotUnderstand(t *testing.T) {
	for name, c := range map[string]struct{ file, says string }{
		"not JSON at all": {
			`[{"id": "prettier",]`, "not the array of objects"},
		"a field nothing reads": {
			`[{"id":"a","registry":"npm","name":"a","version":"1.0.0","reads":["x"],"why":"w","channel":"stable"}]`,
			`carries the field "channel"`},
		"a required field missing": {
			`[{"id":"a","registry":"npm","name":"a","reads":["x"],"why":"w"}]`,
			"has no version"},
		"a registry nothing resolves": {
			`[{"id":"a","registry":"crates","name":"a","version":"1.0.0","reads":["x"],"why":"w"}]`,
			`the registry "crates"`},
		"the same pin twice": {
			`[{"id":"a","registry":"npm","name":"a","version":"1.0.0","reads":["x"],"why":"w"},` +
				`{"id":"a","registry":"npm","name":"a","version":"2.0.0","reads":["y"],"why":"w"}]`,
			"declared twice"},
		"a checksum that is not a digest": {
			`[{"id":"a","registry":"npm","name":"a","version":"1.0.0","checksum":"deadbeef","reads":["x"],"why":"w"}]`,
			"is not a sha256 digest"},
		"an image pinned by tag alone": {
			`[{"id":"a","registry":"docker-hub","name":"library/a","version":"1","repeats":["Dockerfile"],"why":"w"}]`,
			"carries no checksum"},
		"a pin nothing consumes": {
			`[{"id":"a","registry":"npm","name":"a","version":"1.0.0","reads":[],"repeats":[],"why":"w"}]`,
			"names no file that consumes it"},
		"a file declaring nothing": {
			`[]`, "declares no pin"},
	} {
		root := tree(t, c.file, readerThatDoesNotRepeat, dockerfileThatAgrees)
		_, err := Load(root)
		if err == nil {
			t.Errorf("%s was accepted", name)
			continue
		}
		if !strings.Contains(err.Error(), c.says) {
			t.Errorf("%s was refused with %q, which does not say %q", name, err, c.says)
		}
	}
}

// The neighbour of every refusal above: a file that breaks none of them parses
// into the rows it declares.
func TestLoadReadsAFileThatBreaksNothing(t *testing.T) {
	declared, err := Load(good(t))
	if err != nil {
		t.Fatalf("Load refused a file that breaks nothing: %v", err)
	}
	if len(declared) != 2 {
		t.Fatalf("Load read %d pin(s), want 2", len(declared))
	}
	if got, want := declared[1].Reference(),
		"golang:1.26.5-bookworm@sha256:6c5605ab3a9a9fb3c4eafe5b3d63cdbf3881caf113262b67862547b54a9db599"; got != want {
		t.Errorf("the image reference is %q, want %q", got, want)
	}
}

// A pin declared as read and also written into the file that reads it is two
// copies of one fact, which is the state this file exists to remove.
func TestAgreeRefusesAReaderThatAlsoCarriesTheVersion(t *testing.T) {
	root := tree(t, goodFile,
		strings.Replace(readerThatDoesNotRepeat, "${version}", "3.9.6", 1),
		dockerfileThatAgrees)
	declared, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	wrong := Agree(root, declared)
	if len(wrong) != 1 {
		t.Fatalf("Agree reported %v, want exactly the reader carrying the version", wrong)
	}
	for _, want := range []string{"prettier.yml", "3.9.6"} {
		if !strings.Contains(wrong[0], want) {
			t.Errorf("the message %q does not name %q", wrong[0], want)
		}
	}
}

// The one-character version of the mistake on the other side: a digest edited
// in the file that repeats it, which is what a hand-applied update looks like
// when only one of the two places is changed.
func TestAgreeRefusesARepeaterThatCarriesADifferentReference(t *testing.T) {
	root := tree(t, goodFile, readerThatDoesNotRepeat,
		strings.Replace(dockerfileThatAgrees, "sha256:6c56", "sha256:6c57", 1))
	declared, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	wrong := Agree(root, declared)
	if len(wrong) != 1 {
		t.Fatalf("Agree reported %v, want exactly the Dockerfile disagreeing", wrong)
	}
	if !strings.Contains(wrong[0], "Dockerfile") {
		t.Errorf("the message %q does not name the file", wrong[0])
	}
}

func TestAgreePassesConsumersThatAgree(t *testing.T) {
	root := good(t)
	declared, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if wrong := Agree(root, declared); len(wrong) != 0 {
		t.Errorf("Agree refused consumers that agree: %v", wrong)
	}
}

// current answers every pin with what it already says, which is the resolver a
// run over an unchanged upstream sees.
func current(p Pin) (Upstream, error) {
	return Upstream{Version: p.Version, Checksum: p.Checksum}, nil
}

func TestRunPassesWhenEveryUpstreamAgrees(t *testing.T) {
	var log bytes.Buffer
	if err := Run(good(t), current, &log); err != nil {
		t.Fatalf("Run refused a tree whose pins are all current: %v\n%s", err, log.String())
	}
	if !strings.Contains(log.String(), "2 pin(s) declared, 2 compared, 0 behind, 0 unresolved.") {
		t.Errorf("the run does not say what it compared; it said:\n%s", log.String())
	}
}

// A pin that has fallen behind is named with both values, because the point of
// the run is to be the evidence for a change somebody makes and a message
// saying only that something moved does not carry it.
func TestRunNamesAPinThatIsBehindAndPrintsBothValues(t *testing.T) {
	behind := func(p Pin) (Upstream, error) {
		if p.ID != "prettier" {
			return current(p)
		}
		return Upstream{Version: "3.10.0"}, nil
	}

	var log bytes.Buffer
	err := Run(good(t), behind, &log)
	if err == nil {
		t.Fatalf("Run passed a tree carrying a pin that is behind:\n%s", log.String())
	}
	for _, want := range []string{"prettier: BEHIND", "pinned 3.9.6", "npm says 3.10.0", "1 behind"} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q; it said:\n%s", want, log.String())
		}
	}
	if !strings.Contains(err.Error(), "behind") {
		t.Errorf("the error reads %q, which does not say what was wrong", err)
	}
}

// A tag rebuilt under the same name is the drift a version string cannot see,
// so the image is compared by digest and the digest is what the report prints.
func TestRunComparesAnImageByItsDigest(t *testing.T) {
	moved := func(p Pin) (Upstream, error) {
		if p.Registry != DockerHub {
			return current(p)
		}
		return Upstream{Version: p.Version, Checksum: "sha256:" + strings.Repeat("a", 64)}, nil
	}

	var log bytes.Buffer
	if err := Run(good(t), moved, &log); err == nil {
		t.Fatalf("Run passed an image whose tag now resolves elsewhere:\n%s", log.String())
	}
	for _, want := range []string{"golang-image: BEHIND", "sha256:6c5605ab", "docker-hub says sha256:aaaa"} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q; it said:\n%s", want, log.String())
		}
	}
}

// An upstream that could not be read is not an upstream that agreed. This is
// the state a comparison collapses into a pass when nobody separates them, and
// it is the one that makes a stale pin invisible for as long as the registry is
// unreachable.
func TestRunReportsAnUnreadableUpstreamAsUnresolvedRatherThanCurrent(t *testing.T) {
	oneDown := func(p Pin) (Upstream, error) {
		if p.ID != "prettier" {
			return current(p)
		}
		return Upstream{}, fmt.Errorf("the name does not resolve")
	}

	var log bytes.Buffer
	err := Run(good(t), oneDown, &log)
	if err == nil {
		t.Fatalf("Run passed with an upstream it could not read:\n%s", log.String())
	}
	if !strings.Contains(log.String(), "prettier: UNRESOLVED") {
		t.Errorf("the run does not report the pin as unresolved; it said:\n%s", log.String())
	}
	if strings.Contains(log.String(), "prettier: current") {
		t.Errorf("the run reported an unread upstream as current; it said:\n%s", log.String())
	}
	if !strings.Contains(err.Error(), "an unresolved pin is not a current one") {
		t.Errorf("the error reads %q, which does not say why an unresolved pin fails", err)
	}
}

// A run that reached nothing says so. Reporting the whole set as current
// because none of it could be read is the failure every freshness comparison in
// this repository is written against.
func TestRunRefusesToPassARunThatComparedNothing(t *testing.T) {
	allDown := func(Pin) (Upstream, error) { return Upstream{}, fmt.Errorf("no route to the registry") }

	var log bytes.Buffer
	err := Run(good(t), allDown, &log)
	if err == nil {
		t.Fatalf("Run passed having compared nothing:\n%s", log.String())
	}
	if !strings.Contains(err.Error(), "compared nothing") {
		t.Errorf("the error reads %q, which does not say that nothing was compared", err)
	}
	if !strings.Contains(log.String(), "2 pin(s) declared, 0 compared") {
		t.Errorf("the run does not print what it compared; it said:\n%s", log.String())
	}
}

// The run reports and does not bump, on every verdict it can reach. A run that
// rewrote a version on its way to a red result is the failure this is for, and
// a red result is exactly when nobody would look.
func TestRunWritesNothingWhateverItFinds(t *testing.T) {
	for name, resolve := range map[string]Resolver{
		"everything current": current,
		"something behind": func(p Pin) (Upstream, error) {
			return Upstream{Version: "9.9.9", Checksum: "sha256:" + strings.Repeat("b", 64)}, nil
		},
		"nothing reachable": func(Pin) (Upstream, error) { return Upstream{}, fmt.Errorf("down") },
	} {
		root := good(t)
		before := snapshot(t, root)
		_ = Run(root, resolve, &bytes.Buffer{})
		if after := snapshot(t, root); after != before {
			t.Errorf("the run with %s changed the tree:\nbefore\n%s\nafter\n%s", name, before, after)
		}
	}
}

// snapshot is every path under root with the bytes at it, so a file added,
// removed or edited moves the result.
func snapshot(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		lines = append(lines, fmt.Sprintf("%s %d %x", filepath.ToSlash(rel), len(body), body))
		return nil
	})
	if err != nil {
		t.Fatalf("reading the tree: %v", err)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// This repository's own file against this repository's own consumers. The
// fixtures above prove the guard; this proves the tree, which is the other
// question and is not answered by any of them.
func TestThisRepositoryAgreesWithWhatItDeclares(t *testing.T) {
	const root = "../.."

	declared, err := Load(root)
	if err != nil {
		t.Fatalf("this repository's %s does not load: %v", File, err)
	}
	if wrong := Agree(root, declared); len(wrong) != 0 {
		t.Errorf("this repository disagrees with what it declares:\n%s", strings.Join(wrong, "\n"))
	}
}
