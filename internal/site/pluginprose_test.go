// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the per-plugin prose and the join between it and the roster.
//
// Every case builds a tree it wrote itself, for the reason the rest of this
// package's suites give: a case reading the prose this repository carries would
// pass or fail on what those twelve files happen to say today.
package site

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// proseForEveryRow writes one prose file per row the roster carries, so a tree
// a case assembled is a tree the build accepts. A roster that does not decode
// gets no files, because the case that wrote it is about the parser refusing it
// and the read never reaches this far.
func proseForEveryRow(t *testing.T, root, roster string) {
	t.Helper()
	var rows []struct {
		ID string `json:"id"`
	}
	if json.Unmarshal([]byte(roster), &rows) != nil {
		return
	}
	for _, r := range rows {
		writeProse(t, root, r.ID, "What "+r.ID+" is for, at more length than one sentence.")
	}
}

// writeProse puts one prose file in the tree, and proseFile is where it goes.
func writeProse(t *testing.T, root, id, body string) {
	t.Helper()
	mkdir(t, filepath.Join(root, filepath.FromSlash(PluginProseDir)))
	write(t, proseFile(root, id), body+"\n")
}

func proseFile(root, id string) string {
	return filepath.Join(root, filepath.FromSlash(PluginProseDir), id+proseSuffix)
}

// proseRefusal builds a tree and answers with the reason the build gave, failing the
// case where the build accepted it. Every case using it is about a refusal, so
// the acceptance is the failure.
func proseRefusal(t *testing.T, root string) string {
	t.Helper()
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		return err.Error()
	}
	t.Fatal("the build accepted a tree this case expects it to refuse")
	return ""
}

// A row whose file is gone reds the build, and the reason names the identifier
// rather than the path. The identifier is what the two halves share; a reader
// given the path has been told where the file was rather than which plugin the
// site has gone thin about.
func TestARowWithNoProseFileRedsTheBuildAndNamesTheIdentifier(t *testing.T) {
	root := pageTree(t, twoRows, bothRecorded)
	if err := os.Remove(proseFile(root, "beta")); err != nil {
		t.Fatalf("removing the prose this case is about: %v", err)
	}

	said := proseRefusal(t, root)
	if !strings.Contains(said, "beta") {
		t.Fatalf("the refusal does not name the identifier: %s", said)
	}
	if strings.Contains(said, "alpha") {
		t.Fatalf("the refusal names a row whose file is there: %s", said)
	}
}

// A file matching no row reds the build, and the reason names the identifier.
// This is the other direction, and it is a separate case because the two
// failures have opposite causes: a row with no file is a plugin the site is
// silently thin about, and a file with no row is a page nobody can reach.
func TestAProseFileWithNoRowRedsTheBuildAndNamesTheIdentifier(t *testing.T) {
	root := pageTree(t, twoRows, bothRecorded)
	writeProse(t, root, "gamma", "Prose for a plugin the roster does not carry.")

	said := proseRefusal(t, root)
	if !strings.Contains(said, "gamma") {
		t.Fatalf("the refusal does not name the identifier: %s", said)
	}
}

// Both directions at once, which is what a renamed row actually produces. A
// build reporting only the first half sends a reader to repair one side and
// meet the other.
func TestARenamedRowIsRefusedFromBothSidesInOneRun(t *testing.T) {
	root := pageTree(t, twoRows, bothRecorded)
	if err := os.Rename(proseFile(root, "beta"), proseFile(root, "gamma")); err != nil {
		t.Fatalf("renaming the prose this case is about: %v", err)
	}

	said := proseRefusal(t, root)
	for _, want := range []string{"beta", "gamma", "2 reason(s)"} {
		if !strings.Contains(said, want) {
			t.Fatalf("the refusal does not carry %q: %s", want, said)
		}
	}
}

// A tree with no prose directory at all is refused once per row, naming every
// plugin, rather than once naming a path. A list of identifiers is the
// work-list; one path is the observation that the work-list is missing.
func TestATreeWithNoProseAtAllIsRefusedOncePerRow(t *testing.T) {
	root := pageTree(t, twoRows, bothRecorded)
	if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(PluginProseDir))); err != nil {
		t.Fatalf("removing the prose directory: %v", err)
	}

	said := proseRefusal(t, root)
	for _, want := range []string{"alpha", "beta", "2 reason(s)"} {
		if !strings.Contains(said, want) {
			t.Fatalf("the refusal does not carry %q: %s", want, said)
		}
	}
}

// A file that exists and says nothing is refused. It passes the same walk a
// written one does, and the page it produces is the table row with more space
// around it, which is the state the pairing rule exists to make impossible.
func TestAProseFileWithNoParagraphRedsTheBuild(t *testing.T) {
	root := pageTree(t, twoRows, bothRecorded)
	write(t, proseFile(root, "beta"), "\n\n   \n")

	said := proseRefusal(t, root)
	if !strings.Contains(said, "beta") || !strings.Contains(said, "carries no paragraph") {
		t.Fatalf("the refusal is not the one this case is about: %s", said)
	}
}

// A block opening like a keyword is refused rather than rendered as a
// paragraph. These files carry no keyword, so a line that looks like one is
// somebody expecting a field to exist, and a page that renders it looks
// finished either way.
func TestAProseBlockOpeningLikeAKeywordRedsTheBuild(t *testing.T) {
	root := pageTree(t, twoRows, bothRecorded)
	write(t, proseFile(root, "beta"), "A paragraph.\n\ndescription: A sentence somebody expected to be read.\n")

	said := proseRefusal(t, root)
	if !strings.Contains(said, "beta") || !strings.Contains(said, "description:") {
		t.Fatalf("the refusal is not the one this case is about: %s", said)
	}
}

// Something in the directory that is not a prose file is refused rather than
// passed over. A name the walk skips is bytes in the tree that no page renders
// and no row is missing for, which is the shape both halves of the pairing rule
// exist to have no room for.
func TestSomethingThatIsNotAProseFileRedsTheBuild(t *testing.T) {
	root := pageTree(t, twoRows, bothRecorded)
	write(t, filepath.Join(root, filepath.FromSlash(PluginProseDir), "notes.md"), "Not prose.\n")

	said := proseRefusal(t, root)
	if !strings.Contains(said, "notes.md") {
		t.Fatalf("the refusal does not name what it refused: %s", said)
	}
	// The reason is asserted rather than only the name, because the walk that
	// pairs identifiers refuses this file too, as prose for a row nobody
	// carries. A case satisfied by either would pass with this rule deleted and
	// would be asserting that something went red rather than that this did.
	if !strings.Contains(said, "is not one") {
		t.Fatalf("the refusal is the pairing one rather than the one this case is about: %s", said)
	}
}

// Each page renders its own file and no other. Asserting only that a page
// carries some prose would pass a build handing every page the first file it
// read, which is the mistake a join keyed by an identifier is for.
func TestEachPluginPageRendersItsOwnProse(t *testing.T) {
	root := pageTree(t, twoRows, bothRecorded)
	writeProse(t, root, "alpha", "The paragraph that belongs to alpha alone.")
	writeProse(t, root, "beta", "The paragraph that belongs to beta alone.")
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused a tree this case expects it to accept: %v", err)
	}

	for _, c := range []struct{ id, mine, theirs string }{
		{"alpha", "belongs to alpha alone", "belongs to beta alone"},
		{"beta", "belongs to beta alone", "belongs to alpha alone"},
	} {
		page := read(t, filepath.Join(root, OutputDir, PluginsDir, c.id, indexDocument))
		if !strings.Contains(page, c.mine) {
			t.Errorf("the page for %s does not render its own prose", c.id)
		}
		if strings.Contains(page, c.theirs) {
			t.Errorf("the page for %s renders another row's prose", c.id)
		}
	}
}

// The page carries more than the roster sentence, which is the whole reason
// these files exist. The sentence is still there and the paragraphs are around
// it, so the assertion is about both rather than about a replacement.
func TestAPluginPageCarriesItsProseAsWellAsTheRosterSentence(t *testing.T) {
	root := pageTree(t, twoRows, bothRecorded)
	writeProse(t, root, "alpha", "A paragraph the roster does not carry.")
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused a tree this case expects it to accept: %v", err)
	}

	page := read(t, filepath.Join(root, OutputDir, PluginsDir, "alpha", indexDocument))
	for _, want := range []string{"What alpha does", "A paragraph the roster does not carry."} {
		if !strings.Contains(page, want) {
			t.Fatalf("the page does not carry %q", want)
		}
	}
}

// The paragraphs arrive in the order the file carries them, between the roster
// sentence and the computed state sentence. A page that reordered prose would
// read as a different page and pass every containment check.
func TestTheProseArrivesInTheOrderTheFileCarriesIt(t *testing.T) {
	root := pageTree(t, twoRows, bothRecorded)
	writeProse(t, root, "alpha", "The first paragraph.\n\nThe second paragraph.")
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused a tree this case expects it to accept: %v", err)
	}

	page := read(t, filepath.Join(root, OutputDir, PluginsDir, "alpha", indexDocument))
	summary := strings.Index(page, "What alpha does")
	first := strings.Index(page, "The first paragraph.")
	second := strings.Index(page, "The second paragraph.")
	state := strings.Index(page, "The plugin is being built.")
	if summary < 0 || first < 0 || second < 0 || state < 0 {
		t.Fatalf("the page is missing one of the four parts this case orders: %s", page)
	}
	if summary > first || first > second || second > state {
		t.Fatalf("the page does not carry the four parts in the order it should: %s", page)
	}
}

// Prose carrying markup characters renders as text. The roster sentence is
// already fuzzed through this path; this is the second entrance to it, and a
// page that escaped one and not the other is a page nothing else would catch.
func TestProseCarryingMarkupRendersAsTextOnItsPage(t *testing.T) {
	root := pageTree(t, twoRows, bothRecorded)
	writeProse(t, root, "alpha",
		"A paragraph with <b>markup</b>, an & ampersand and a \"quotation\" in it.")
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused a tree this case expects it to accept: %v", err)
	}

	page := read(t, filepath.Join(root, OutputDir, PluginsDir, "alpha", indexDocument))
	if strings.Contains(page, "<b>markup</b>") {
		t.Fatalf("prose reached the page as markup: %s", page)
	}
	for _, want := range []string{"&lt;b&gt;markup&lt;/b&gt;", "&amp; ampersand"} {
		if !strings.Contains(page, want) {
			t.Fatalf("the page does not carry %q as text: %s", want, page)
		}
	}
}
