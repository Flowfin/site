// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the pages there is one of per roster row.
//
// Every case builds a tree it wrote itself, for the reason the build suite
// gives. The frame these cases render through is the fixture's rather than the
// repository's, so a case says what the writer produced rather than what the
// real template happens to say today.
package site

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Flowfin/site/internal/releases"
)

// pageTree writes a tree whose frame renders a page's title, its paragraphs and
// its ways onward, which is everything a plugin page is made of.
func pageTree(t *testing.T, roster, record string) string {
	t.Helper()
	root := t.TempDir()
	mkdir(t, filepath.Join(root, TemplatesDir))
	write(t, filepath.Join(root, TemplatesDir, "page.html.tmpl"),
		"<title>{{ .Title }}</title>\n<link rel=\"canonical\" href=\"{{ .Canonical }}\" />\n"+
			"{{- range .Paragraphs }}\n<p>{{ . }}</p>\n{{- end }}\n"+
			"{{- range .Onward }}\n<a href=\"{{ .Href }}\">{{ .Text }}</a>\n{{- end }}\n")
	mkdir(t, filepath.Join(root, ContentDir))
	write(t, filepath.Join(root, ContentDir, "index.txt"),
		"A title\n\ndescription: What the fixture page is.\n\nA paragraph.\n")
	mkdir(t, filepath.Join(root, filepath.Dir(filepath.FromSlash(RosterFile))))
	write(t, filepath.Join(root, filepath.FromSlash(RosterFile)), roster)
	write(t, filepath.Join(root, filepath.FromSlash(releases.File)), record)
	proseForEveryRow(t, root, roster)
	return root
}

// producedPluginPages lists the plugin pages a build wrote, by the address they
// are served at, sorted so a case compares sets rather than an order the writer
// happens to use.
func producedPluginPages(t *testing.T, written []string) []string {
	t.Helper()
	var got []string
	for _, w := range written {
		rel := strings.TrimPrefix(w, OutputDir+"/")
		if strings.HasPrefix(rel, PluginsDir+"/") {
			got = append(got, addressOf(rel))
		}
	}
	sort.Strings(got)
	return got
}

// One page per row and none without one, at the addresses
// decisions/0008-the-url-shape.md gives them. The set is compared both ways: a
// case asserting only that every row got a page would pass a build that also
// wrote a page for something the roster does not carry.
func TestOnePageIsProducedPerRosterRowAndNoneWithoutOne(t *testing.T) {
	written, err := Build(pageTree(t, twoRows, bothRecorded), OutputDir, io.Discard)
	if err != nil {
		t.Fatalf("the build refused a tree this case expects it to accept: %v", err)
	}

	got := producedPluginPages(t, written)
	want := []string{"/plugins/alpha/", "/plugins/beta/"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("the build produced %v, want %v", got, want)
	}
}

// A row added to the roster produces a page with no other edit. Nothing about
// the frame, the writer or any count is touched between the two builds, which
// is the property the whole generator exists for.
func TestARowAddedToTheRosterProducesAPageWithNoOtherEdit(t *testing.T) {
	three := strings.Replace(twoRows, "\n]",
		",\n  {\"id\":\"gamma\",\"repository\":\"Flowfin/jellyfin-plugin-gamma\","+
			"\"summary\":\"What gamma does\",\"state\":\"build-up\"}\n]", 1)
	record := strings.Replace(bothRecorded, `"Flowfin/jellyfin-plugin-beta":{"finished":0,"prereleases":0}}}`,
		`"Flowfin/jellyfin-plugin-beta":{"finished":0,"prereleases":0},`+
			`"Flowfin/jellyfin-plugin-gamma":{"finished":0,"prereleases":0}}}`, 1)

	root := pageTree(t, three, record)
	written, err := Build(root, OutputDir, io.Discard)
	if err != nil {
		t.Fatalf("the build refused a tree this case expects it to accept: %v", err)
	}

	got := producedPluginPages(t, written)
	want := []string{"/plugins/alpha/", "/plugins/beta/", "/plugins/gamma/"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("the build produced %v, want %v", got, want)
	}
	page := read(t, filepath.Join(root, OutputDir, PluginsDir, "gamma", "index.html"))
	if !strings.Contains(page, "What gamma does") {
		t.Errorf("the page for the added row does not say what it does:\n%s", page)
	}
}

// What one page carries: what the plugin does, what state it is in and what
// that state means, and where its repository is. The state sentence is the one
// worth asserting by its content rather than its presence, because a page
// saying a plugin is in build-up and not saying that nothing is installable is
// a page a reader leaves to go looking for a download.
func TestAPluginPageSaysWhatItIsWhatStateItIsInAndWhereItLives(t *testing.T) {
	root := pageTree(t, twoRows, bothRecorded)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused a tree this case expects it to accept: %v", err)
	}

	page := read(t, filepath.Join(root, OutputDir, PluginsDir, "beta", "index.html"))
	for _, want := range []string{
		"<title>beta</title>",
		"What beta does",
		"The repository holds the shape of the plugin and none of what it is for.",
		"there is nothing to install",
		"Flowfin/jellyfin-plugin-beta",
		`href="https://github.com/Flowfin/jellyfin-plugin-beta"`,
		`href="/"`,
		`href="` + Origin + `/plugins/beta/"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page does not carry %q:\n%s", want, page)
		}
	}
}

// A plugin in the shell state gets a page like any other. Hiding it would make
// the site disagree with the list it is generated from, and the reader arriving
// at a shell is the one least likely to know what they are looking at.
func TestAShellGetsAPageLikeAnyOther(t *testing.T) {
	root := pageTree(t, twoRows, bothRecorded)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused a tree this case expects it to accept: %v", err)
	}

	shell := read(t, filepath.Join(root, OutputDir, PluginsDir, "beta", "index.html"))
	building := read(t, filepath.Join(root, OutputDir, PluginsDir, "alpha", "index.html"))
	if n := strings.Count(shell, "<p>"); n != strings.Count(building, "<p>") {
		t.Errorf("the shell's page carries %d paragraph(s) and the other carries %d",
			n, strings.Count(building, "<p>"))
	}
}

// A sentence carrying markup renders as text on the page as well as in the
// table. It is a property of the path the roster takes into a page rather than
// of the rows this tree carries, so the case is written with data that tries.
func TestARowCarryingMarkupRendersAsTextOnItsPage(t *testing.T) {
	trying := strings.Replace(twoRows, "What alpha does",
		`A <b>bold</b> claim & a bracket`, 1)

	root := pageTree(t, trying, bothRecorded)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused a tree this case expects it to accept: %v", err)
	}

	page := read(t, filepath.Join(root, OutputDir, PluginsDir, "alpha", "index.html"))
	if strings.Contains(page, "<b>bold</b>") {
		t.Errorf("the page carries the sentence as markup:\n%s", page)
	}
	if !strings.Contains(page, "&lt;b&gt;bold&lt;/b&gt; claim &amp; a bracket") {
		t.Errorf("the page does not carry the sentence as text:\n%s", page)
	}
}

// A tree with no roster produces no plugin page and no empty container. A build
// that made the directory anyway would leave a served address answering with
// whatever the host does for an empty one.
func TestATreeWithNoRosterProducesNoPluginPage(t *testing.T) {
	root := rosterTree(t, "", bothRecorded)
	written, err := Build(root, OutputDir, io.Discard)
	if err != nil {
		t.Fatalf("the build refused a tree with no roster: %v", err)
	}
	if got := producedPluginPages(t, written); len(got) != 0 {
		t.Errorf("a tree with no roster produced %v", got)
	}
	if _, err := os.Stat(filepath.Join(root, OutputDir, PluginsDir)); !os.IsNotExist(err) {
		t.Errorf("the build made the container for pages it did not write")
	}
}
