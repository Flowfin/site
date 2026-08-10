// The fuzz target over the only place content becomes markup.
//
// A generator's whole attack surface is the door the data comes through and the
// template it is written into. The door is fuzzed beside the parser. This is the
// other half: whatever a value in the data says, it has to arrive on the page as
// text and never as markup.
//
// The property is not "the output contains the input", because the build folds
// wrapped lines back together and a comparison on the bytes would be a
// comparison against that folding. It is about the shape instead: a page
// rendered from arbitrary prose carries the same elements, in the same order,
// carrying the same attribute names as the page the template writes from prose
// of that shape with nothing interesting in it. Attribute values are left out of
// the comparison because those are what legitimately differ. A value that became
// markup adds an element or an attribute, and neither survives that comparison.
//
// The tree is written by this file rather than read from the repository, which
// is the rule the rest of this suite follows and it holds here for the same
// reason: a target reading the real templates/ would be judging what those files
// happen to say today. What it judges instead is the rendering path, and the
// call that escapes is on that path.
package site

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// fuzzTemplate puts a value in the two positions a value can be written into:
// the text of an element and the value of an attribute. They are separate
// contexts with separate escaping, and a page that is safe in one and not the
// other is the failure this target exists to catch.
const fuzzTemplate = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>{{ .Title }}</title>
  </head>
  <body>
    <main data-title="{{ .Title }}">
      <h1>{{ .Title }}</h1>
      {{- range .Paragraphs }}
      <p>{{ . }}</p>
      {{- end }}
    </main>
  </body>
</html>
`

// What a browser reads as the start or the end of an element, the name on it,
// and the attributes hanging off it. A value that escaped its context produces
// an element or an attribute; a value that did not cannot, because every
// character that could open either is written as an entity.
//
// The quoted alternatives in the attribute pattern are what keeps a value from
// being read as an attribute of its own: `data-title="a onclick=1"` is one match
// and not two, because the quoted value is consumed whole. It is only when the
// escaping is gone and a real quote arrives from the data that the second
// attribute becomes visible, which is the case this target is for.
var (
	markup       = regexp.MustCompile(`(?s)<[a-zA-Z/!?][^>]*>`)
	elementName  = regexp.MustCompile(`(?s)^<\s*([a-zA-Z/!?][^\s>]*)`)
	tagAttribute = regexp.MustCompile(`(?s)([a-zA-Z_:][-a-zA-Z0-9_:.]*)\s*=\s*(?:"[^"]*"|'[^']*'|[^\s>]+)`)
)

// FuzzBuildEscapesPageProse refuses a build whose page carries an element the
// template did not write.
func FuzzBuildEscapesPageProse(f *testing.F) {
	// The corpus is the shapes that already matter, committed here so each
	// entry is read beside the reason it exists. `go test` replays all of
	// them without fuzzing, so the gate's test leg carries them on every run.
	seeds := []string{
		// Prose with nothing in it to escape, so the target explores from
		// a page that renders rather than from noise.
		"A title\n\nA paragraph.\n\nA second paragraph.\n",
		// The shape somebody writes on purpose.
		"Title\n\n<script>alert(1)</script>\n",
		"Title\n\n<img src=x onerror=alert(1)>\n",
		// Breaking out of the attribute rather than out of the text.
		"Title\n\n\" onmouseover=\"alert(1)\n",
		"\" onmouseover=\"alert(1)\n\nA paragraph.\n",
		// Brackets, quotes and an ampersand, which is what an ordinary
		// sentence about markup looks like.
		"Uses <b> and \"quotes\" and 'ticks' & an ampersand\n\nAnd a paragraph.\n",
		// A comment and a doctype, which are markup without being an
		// element.
		"Title\n\n<!-- a comment -->\n",
		"<!DOCTYPE html>\n\nA paragraph.\n",
		// An empty field and an empty file.
		"Title\n\n\n\nA paragraph.\n",
		"",
		"\n\n\n",
		// Characters outside ASCII, and a byte sequence that is not
		// valid UTF-8 at all.
		"Grüße\n\n日本語 and a paragraph.\n",
		"Title\n\n\xff\xfe\x00 bytes\n",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	// A very long single field, kept out of the list above so the list stays
	// readable.
	f.Add([]byte("Title\n\n" + strings.Repeat("long ", 4096) + "\n"))

	f.Fuzz(func(t *testing.T, prose []byte) {
		got, ok := renderProse(t, prose)
		if !ok {
			// The build refused. A refusal is not an escape, and which
			// prose is refused is the suite's question rather than
			// this one's.
			return
		}
		shape, ok := renderProse(t, inertProseOfTheSameShape(t, prose))
		if !ok {
			t.Fatalf("the build refused prose of the same shape carrying nothing to escape, so the two pages cannot be compared")
		}

		want, found := skeleton(shape), skeleton(got)
		if len(found) != len(want) {
			t.Fatalf("the page carries %d element(s) and the same shape carries %d, so a value in the prose became markup\nfrom: %q\npage: %s",
				len(found), len(want), prose, got)
		}
		for i := range want {
			if found[i] != want[i] {
				t.Fatalf("element %d of the page is %s where the same shape writes %s, so a value in the prose became markup\nfrom: %q\npage: %s",
					i+1, found[i], want[i], prose, got)
			}
		}
	})
}

// skeleton is the page reduced to what it asks a browser to do: one entry per
// element, carrying its name and the names of its attributes and none of their
// values. The values are left out because they legitimately differ between two
// pages rendered from different prose, and what may not differ is which elements
// and which attributes exist at all.
func skeleton(page string) []string {
	tags := markup.FindAllString(page, -1)
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		name := ""
		if m := elementName.FindStringSubmatch(tag); m != nil {
			name = strings.ToLower(m[1])
		}
		var attributes []string
		for _, a := range tagAttribute.FindAllStringSubmatch(tag, -1) {
			attributes = append(attributes, strings.ToLower(a[1]))
		}
		sort.Strings(attributes)
		out = append(out, "<"+strings.Join(append([]string{name}, attributes...), " ")+">")
	}
	return out
}

// renderProse writes a tree carrying prose as the page's content, builds it, and
// returns the page. The second value is false where the build refused, which is
// a state this target passes over rather than judges.
func renderProse(t *testing.T, prose []byte) (string, bool) {
	t.Helper()

	root := t.TempDir()
	mkdir(t, filepath.Join(root, TemplatesDir))
	write(t, filepath.Join(root, TemplatesDir, "page.html.tmpl"), fuzzTemplate)
	mkdir(t, filepath.Join(root, ContentDir))
	write(t, filepath.Join(root, ContentDir, "index.txt"), string(prose))

	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		return "", false
	}
	b, err := os.ReadFile(filepath.Join(root, OutputDir, "index.html"))
	if err != nil {
		t.Fatalf("the build reported no error and wrote no page: %v", err)
	}
	return string(b), true
}

// inertProseOfTheSameShape is prose the build reads as the same title and the
// same number of paragraphs, carrying nothing that could become markup. Its
// shape is read back out of the build's own reader rather than guessed from the
// bytes, so the two pages are the same shape by construction and not by a second
// implementation of how a blank line is counted.
func inertProseOfTheSameShape(t *testing.T, prose []byte) []byte {
	t.Helper()

	root := t.TempDir()
	name := filepath.Join(root, "index.txt")
	write(t, name, string(prose))

	p, err := readPage(name)
	if err != nil {
		t.Fatalf("the build read this prose and this call did not: %v", err)
	}
	return []byte("x" + strings.Repeat("\n\nx", len(p.Paragraphs)))
}
