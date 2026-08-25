// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the plugin rows.
//
// Every case builds a tree it wrote itself, for the reason the build suite
// gives: a case reading the roster this repository carries would pass or fail
// on what that file happens to say today. What the rows are on the real tree is
// read by the gate's build leg and by the run in the change that landed them.
package site

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Flowfin/site/internal/releases"
	"github.com/Flowfin/site/internal/roster"
)

// A roster of two rows, and the record that says both repositories are there.
// Two rather than one, because a page rendering the first row of a list and a
// page rendering all of it are the same page when the list has one row in it.
const twoRows = `[
  {"id":"alpha","repository":"Flowfin/jellyfin-plugin-alpha","summary":"What alpha does","state":"build-up"},
  {"id":"beta","repository":"Flowfin/jellyfin-plugin-beta","summary":"What beta does","state":"shell"}
]`

const bothRecorded = `{"taken":"2026-01-02","command":"a command","repositories":{
  "Flowfin/jellyfin-plugin-alpha":{"finished":0,"prereleases":0},
  "Flowfin/jellyfin-plugin-beta":{"finished":0,"prereleases":0}}}`

// rosterTree writes a buildable tree whose template renders the rows and
// nothing else, so a case reads what the rows produced rather than the rest of
// a page.
func rosterTree(t *testing.T, roster, record string) string {
	t.Helper()
	root := t.TempDir()
	mkdir(t, filepath.Join(root, TemplatesDir))
	write(t, filepath.Join(root, TemplatesDir, "page.html.tmpl"),
		"<title>{{ .Title }}</title>\n<p>{{ .PluginsRead }}</p>\n"+
			"{{- range .Plugins }}\n<tr><td>{{ .ID }}</td><td>{{ .Href }}</td>"+
			"<td>{{ .Summary }}</td><td>{{ .State }}</td></tr>\n{{- end }}\n")
	mkdir(t, filepath.Join(root, ContentDir))
	write(t, filepath.Join(root, ContentDir, "index.txt"),
		"A title\n\ndescription: What the fixture page is.\n\nA paragraph.\n")
	if roster != "" {
		mkdir(t, filepath.Join(root, filepath.Dir(filepath.FromSlash(RosterFile))))
		write(t, filepath.Join(root, filepath.FromSlash(RosterFile)), roster)
	}
	if record != "" {
		mkdir(t, filepath.Join(root, filepath.Dir(filepath.FromSlash(releases.File))))
		write(t, filepath.Join(root, filepath.FromSlash(releases.File)), record)
	}
	return root
}

func built(t *testing.T, root string) string {
	t.Helper()
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused a tree these cases expect it to accept: %v", err)
	}
	return read(t, filepath.Join(root, OutputDir, IndexPath))
}

// One row on the page per row in the roster, in the order the roster carries
// them, and none the roster does not carry. The order is asserted by position
// rather than by presence, because a page carrying every row in the wrong order
// passes every containment check and is a different table.
func TestThePageCarriesOneRowPerRosterRowInOrder(t *testing.T) {
	page := built(t, rosterTree(t, twoRows, bothRecorded))

	rows := strings.Count(page, "<tr>")
	if rows != 2 {
		t.Fatalf("the page carries %d row(s) for a roster of 2:\n%s", rows, page)
	}
	want := []string{
		"<tr><td>alpha</td><td>/plugins/alpha/</td>" +
			"<td>What alpha does</td><td>In build-up</td></tr>",
		"<tr><td>beta</td><td>/plugins/beta/</td>" +
			"<td>What beta does</td><td>Shell only</td></tr>",
	}
	at := 0
	for _, w := range want {
		i := strings.Index(page[at:], w)
		if i < 0 {
			t.Fatalf("the page does not carry %q in order:\n%s", w, page)
		}
		at += i + len(w)
	}
}

// A row added to the roster produces a row with no other edit, which is the
// whole reason this repository has a generator. Nothing about the template, the
// reader or any count is touched between the two builds.
func TestARowAddedToTheRosterProducesARowWithNoOtherEdit(t *testing.T) {
	three := strings.Replace(twoRows, "\n]",
		",\n  {\"id\":\"gamma\",\"repository\":\"Flowfin/jellyfin-plugin-gamma\","+
			"\"summary\":\"What gamma does\",\"state\":\"build-up\"}\n]", 1)
	record := strings.Replace(bothRecorded, `"Flowfin/jellyfin-plugin-beta":{"finished":0,"prereleases":0}}}`,
		`"Flowfin/jellyfin-plugin-beta":{"finished":0,"prereleases":0},`+
			`"Flowfin/jellyfin-plugin-gamma":{"finished":0,"prereleases":0}}}`, 1)

	page := built(t, rosterTree(t, three, record))
	if got := strings.Count(page, "<tr>"); got != 3 {
		t.Fatalf("the page carries %d row(s) for a roster of 3:\n%s", got, page)
	}
	if !strings.Contains(page, "<td>What gamma does</td>") {
		t.Errorf("the page does not carry the added row:\n%s", page)
	}
	if !strings.Contains(page, "3 rows, read from the roster") {
		t.Errorf("the sentence above the table does not follow the roster:\n%s", page)
	}
}

// A sentence carrying markup renders as text. It is a property of the path the
// roster takes into the page rather than of the file this tree carries today,
// so the case is written with data that tries.
func TestASummaryCarryingMarkupRendersAsText(t *testing.T) {
	trying := strings.Replace(twoRows, "What alpha does",
		`A <b>bold</b> claim & a bracket`, 1)

	page := built(t, rosterTree(t, trying, bothRecorded))
	if strings.Contains(page, "<b>bold</b>") {
		t.Errorf("the page carries the sentence as markup:\n%s", page)
	}
	if !strings.Contains(page, "&lt;b&gt;bold&lt;/b&gt; claim &amp; a bracket") {
		t.Errorf("the page does not carry the sentence as text:\n%s", page)
	}
}

// The four ways this read fails closed. Each one is a state in which a build
// that carried on would put a page in front of a reader saying something nobody
// checked, and each names what it could not do rather than failing as one
// message about a roster.
func TestTheReadFailsClosed(t *testing.T) {
	for name, tc := range map[string]struct {
		roster, record string
		want           string
	}{
		"a row whose repository is not in the record": {
			roster: twoRows,
			record: `{"taken":"2026-01-02","command":"a command","repositories":{"Flowfin/jellyfin-plugin-alpha":{"finished":0,"prereleases":0}}}`,
			want:   "which is not there",
		},
		"a record carrying no repository": {
			roster: twoRows,
			record: `{"taken":"2026-01-02","command":"a command","repositories":{}}`,
			want:   "carries no repository",
		},
		"a record saying nothing about when it was taken": {
			roster: twoRows,
			record: `{"command":"a command","repositories":{"Flowfin/jellyfin-plugin-alpha":{"finished":0,"prereleases":0}}}`,
			want:   "says nothing about when it was taken",
		},
		"a roster with no record beside it": {
			roster: twoRows,
			record: "",
			want:   "which is what the shipping state and the roster's repositories are read from",
		},
	} {
		root := rosterTree(t, tc.roster, tc.record)
		_, err := Build(root, OutputDir, io.Discard)
		if err == nil {
			t.Errorf("%s: the build accepted it", name)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: the build refused with %q, which does not say %q", name, err, tc.want)
		}
	}
}

// A state word the page has no words for reds the build rather than reaching a
// reader as a bare identifier. The parser refuses one too, and this is the
// second half: what the file may declare and what the page can say are two
// questions, and a vocabulary widened in one place without the other is what
// this refuses.
func TestAStateThePageHasNoWordsForRedsTheBuild(t *testing.T) {
	if _, _, err := stateInWords("ships", releases.Repository{}); err == nil {
		t.Fatal("the page accepted a state it has no words for")
	} else if !strings.Contains(err.Error(), "ships") {
		t.Errorf("the refusal reads %q, which does not name the state it found", err)
	}
}

// A tree with no roster builds and says so. The page then carries no table,
// which is a tree somebody is part way through rather than a violation; what
// refuses this repository losing its roster is the invariant over the tree.
func TestATreeWithNoRosterBuildsAndSaysSo(t *testing.T) {
	var log bytes.Buffer
	root := rosterTree(t, "", bothRecorded)
	if _, err := Build(root, OutputDir, &log); err != nil {
		t.Fatalf("the build refused a tree with no roster: %v", err)
	}
	if !strings.Contains(log.String(), "so the page carries no plugin row") {
		t.Errorf("the run does not say the roster was absent; it said:\n%s", log.String())
	}
	if page := read(t, filepath.Join(root, OutputDir, IndexPath)); strings.Contains(page, "<tr>") {
		t.Errorf("a tree with no roster produced a page with rows on it:\n%s", page)
	}
}

// The declared word is the floor and what is published raises it, which is
// decisions/0001's rule read through decisions/0009's counting. The case that
// matters is the shell: a row promising a reader that installing it does
// nothing, for a repository that has published something finished, is shown as
// shipping rather than as what the row claims, and the disagreement between the
// two is a thing to repair where it is published rather than something this
// build hides.
func TestWhatIsPublishedRaisesTheDeclaredState(t *testing.T) {
	for name, tc := range map[string]struct {
		state     string
		published releases.Repository
		word      string
		says      string
	}{
		"a shell with nothing published": {
			roster.Shell, releases.Repository{}, "Shell only", "nothing to install"},
		"a plugin in build-up with nothing published": {
			roster.BuildUp, releases.Repository{}, "In build-up", "nothing to install"},
		"a plugin in build-up with only prereleases": {
			roster.BuildUp, releases.Repository{Prereleases: 19}, "In build-up",
			"19 prereleases are published, which is something to test rather than something to run"},
		"a shell with a finished release": {
			roster.Shell, releases.Repository{Finished: 1}, "Ships", "something to install"},
		"a plugin in build-up with a finished release and prereleases": {
			roster.BuildUp, releases.Repository{Finished: 11, Prereleases: 1}, "Ships",
			"One prerelease is published"},
	} {
		word, means, err := stateInWords(tc.state, tc.published)
		if err != nil {
			t.Errorf("%s was refused: %v", name, err)
			continue
		}
		if word != tc.word {
			t.Errorf("%s is shown as %q, want %q", name, word, tc.word)
		}
		if !strings.Contains(means, tc.says) {
			t.Errorf("%s says %q, which does not carry %q", name, means, tc.says)
		}
	}
}

// The sentence above the table states when the release data was taken, which is
// what makes a page produced from a recorded answer readable as one.
func TestTheSentenceAboveTheTableStatesWhenTheDataWasTaken(t *testing.T) {
	record := strings.Replace(bothRecorded, `"taken":"2026-01-02"`, `"taken":"2019-07-04"`, 1)
	page := built(t, rosterTree(t, twoRows, record))
	if !strings.Contains(page, "was read on 2019-07-04") {
		t.Errorf("the page does not state when the release data was taken:\n%s", page)
	}
}
