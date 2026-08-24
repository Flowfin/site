// SPDX-License-Identifier: AGPL-3.0-or-later

// Package link walks what the build produced and refuses a reference that goes
// nowhere.
//
// A generated site fails silently in one specific way: a template renames a
// page, every reference to the old path stays where it was, and nothing notices
// because the build succeeded. The bytes are valid, the run is green, and the
// only party who finds out is a reader who followed the link.
//
// It reads the output rather than the templates, for the same reason the origin
// row does: a reference assembled from a data value exists in the page and not in
// the template.
//
// It needs no network and never makes a request. A reference that leaves this
// site is somebody else's uptime, and fetching one would turn an unrelated outage
// into a red gate; those are compared against the allowlist by the invariant
// gate instead. What this decides is the half that is entirely a property of what
// the build just wrote.
package link

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Flowfin/site/internal/site"
)

var (
	reference = regexp.MustCompile(`(?is)\b(href|src)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	// An id is what a fragment points at. The name attribute on an anchor is
	// the older spelling of the same thing and browsers still honour it, so a
	// page using it is not a page with a dead fragment.
	target = regexp.MustCompile(`(?is)\b(?:id|name)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	// A reference with an authority, or one whose scheme carries no path into
	// this site. Neither is this check's business.
	elsewhere = regexp.MustCompile(`(?i)^(?:[a-z][a-z0-9+.-]*:)?//|^(?:mailto|tel|data|javascript|sms|news):`)
)

// Run builds the site at root into a directory it throws away and refuses a
// reference in the result that resolves to nothing.
func Run(root string, log io.Writer) error {
	tmp, err := os.MkdirTemp("", "site-links-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	out := filepath.Join(tmp, site.OutputDir)
	written, err := site.Build(root, out, io.Discard)
	if err != nil {
		return fmt.Errorf("the build refused, so there is nothing to walk: %w", err)
	}

	pages, produced, err := gather(out, written)
	if err != nil {
		return err
	}

	details := Decide(pages, produced)

	refs := countReferences(pages)
	fmt.Fprintf(log, "links: %d reference(s) over %d page(s), against %d produced file(s)\n",
		refs, len(pages), len(produced))
	if len(pages) == 0 {
		fmt.Fprintf(log, "  the build produced no page, so this leg examined nothing\n")
		return nil
	}
	if refs == 0 {
		fmt.Fprintf(log, "  no page carries a reference, so this leg examined nothing\n")
		return nil
	}
	if len(details) > 0 {
		for _, d := range details {
			fmt.Fprintf(log, "  %s\n", d)
		}
		return fmt.Errorf("links: %d reference(s) resolve to nothing the build wrote", len(details))
	}
	fmt.Fprintf(log, "  every reference that stays inside this site resolves to a file the build wrote\n")
	return nil
}

// Page is one produced page, named by the address it will be served at.
type Page struct {
	Name string
	Body []byte
}

// Decide returns one detail per reference that goes nowhere, in the order the
// pages were given and the references were written. produced is the set of paths
// the build wrote, relative to the output directory.
func Decide(pages []Page, produced map[string]bool) []string {
	ids := map[string]map[string]bool{}
	for _, p := range pages {
		ids[p.Name] = identifiers(p.Body)
	}

	var details []string
	for _, p := range pages {
		for _, r := range referencesIn(p.Body) {
			if r.value == "" || elsewhere.MatchString(r.value) {
				continue
			}
			address, fragment := split(r.value)

			file := p.Name
			if address != "" {
				file = resolve(p.Name, address)
				if !produced[file] {
					details = append(details, fmt.Sprintf(
						"%s: line %d %s=%q resolves to %s, which the build did not write",
						page(p.Name), r.line, r.attribute, r.value, page(file)))
					continue
				}
			}
			if fragment == "" {
				continue
			}
			known, ok := ids[file]
			if !ok {
				details = append(details, fmt.Sprintf(
					"%s: line %d %s=%q points into %s, which is not a page and carries no target",
					page(p.Name), r.line, r.attribute, r.value, page(file)))
				continue
			}
			if !known[fragment] {
				details = append(details, fmt.Sprintf(
					"%s: line %d %s=%q points at #%s, and %s carries no element with that id",
					page(p.Name), r.line, r.attribute, r.value, fragment, page(file)))
			}
		}
	}
	return details
}

type found struct {
	attribute string
	value     string
	line      int
}

func referencesIn(body []byte) []found {
	var out []found
	for _, m := range reference.FindAllSubmatchIndex(body, -1) {
		value := ""
		switch {
		case m[4] >= 0:
			value = string(body[m[4]:m[5]])
		case m[6] >= 0:
			value = string(body[m[6]:m[7]])
		}
		out = append(out, found{
			attribute: strings.ToLower(string(body[m[2]:m[3]])),
			value:     strings.TrimSpace(value),
			line:      lineOf(body, m[0]),
		})
	}
	return out
}

func identifiers(body []byte) map[string]bool {
	out := map[string]bool{}
	for _, m := range target.FindAllSubmatch(body, -1) {
		for _, g := range m[1:] {
			if v := strings.TrimSpace(string(g)); v != "" {
				out[v] = true
			}
		}
	}
	return out
}

// split takes the address and the fragment apart, and drops a query, which the
// static host this site is served from does nothing with.
func split(ref string) (address, fragment string) {
	address = ref
	if i := strings.Index(address, "#"); i >= 0 {
		fragment, address = address[i+1:], address[:i]
	}
	if i := strings.Index(address, "?"); i >= 0 {
		address = address[:i]
	}
	return address, fragment
}

// resolve turns a reference written on a page into the path the host would look
// for. An address absolute from the site root is what every reference in a
// produced page is supposed to be, and a relative one is resolved against the
// page it was written on so that the failure is reported rather than assumed
// away. A directory address is served by the index document inside it.
func resolve(from, address string) string {
	target := address
	if !strings.HasPrefix(address, "/") {
		target = path.Join(path.Dir(from), address)
	}
	target = strings.TrimPrefix(path.Clean("/"+target), "/")
	if strings.HasSuffix(address, "/") || address == "" {
		target = path.Join(target, "index.html")
	}
	return target
}

func gather(out string, written []string) ([]Page, map[string]bool, error) {
	label := filepath.ToSlash(out) + "/"

	produced := map[string]bool{}
	var pages []Page
	for _, w := range written {
		name := strings.TrimPrefix(filepath.ToSlash(w), label)
		produced[name] = true
		if !strings.HasSuffix(name, ".html") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(name)))
		if err != nil {
			return nil, nil, err
		}
		pages = append(pages, Page{Name: name, Body: body})
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Name < pages[j].Name })
	return pages, produced, nil
}

func countReferences(pages []Page) int {
	n := 0
	for _, p := range pages {
		n += len(referencesIn(p.Body))
	}
	return n
}

// page names a produced path the way a reader opens it.
func page(name string) string {
	return path.Join(site.OutputDir, name)
}

func lineOf(body []byte, at int) int {
	return strings.Count(string(body[:at]), "\n") + 1
}
