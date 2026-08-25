// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the comparison between the pinned roster and the published
// one.
//
// No case reaches the network. The door is a parameter, and every case below
// hands the run something it wrote itself, including the two failures that
// cannot be produced on demand from a real address: nothing published there, and
// a fetch that did not come back.
package freshness

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Flowfin/site/internal/roster"
	"github.com/Flowfin/site/internal/site"
)

const twoRows = `[
  {"id":"alpha","repository":"Flowfin/jellyfin-plugin-alpha","summary":"What alpha does","state":"build-up"},
  {"id":"beta","repository":"Flowfin/jellyfin-plugin-beta","summary":"What beta does","state":"shell"}
]`

// tree writes a root carrying the pinned copy and returns it.
func tree(t *testing.T, pinned string) string {
	t.Helper()
	root := t.TempDir()
	name := filepath.Join(root, filepath.FromSlash(site.RosterFile))
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatalf("preparing the tree: %v", err)
	}
	if err := os.WriteFile(name, []byte(pinned), 0o644); err != nil {
		t.Fatalf("writing the pinned copy: %v", err)
	}
	return root
}

func answers(body string) Fetcher {
	return func() ([]byte, error) { return []byte(body), nil }
}

// The neighbour of every case below: two files that agree pass, and the run
// says what it compared rather than only that it was happy.
func TestACopyLevelWithWhatIsPublishedPasses(t *testing.T) {
	var log strings.Builder
	if err := Run(tree(t, twoRows), answers(twoRows), &log); err != nil {
		t.Fatalf("the run refused two files that agree: %v\n%s", err, log.String())
	}
	if !strings.Contains(log.String(), "2 row(s) pinned, 2 published, 0 differing.") {
		t.Errorf("the run does not say what it compared; it said:\n%s", log.String())
	}
}

// Each shape a difference arrives in, one at a time, and the run names the row
// rather than saying the files differ. The three field cases are separate
// because a message that named the row and not the field would send the next
// reader back to diff two files by hand.
func TestADifferenceRedsTheRunAndNamesTheRow(t *testing.T) {
	for name, tc := range map[string]struct{ published, says string }{
		"a sentence somebody edited where it is published": {
			strings.Replace(twoRows, "What alpha does", "What alpha really does", 1),
			`alpha: BEHIND`},
		"a state that moved where it is published": {
			strings.Replace(twoRows, `"state":"shell"`, `"state":"build-up"`, 1),
			`this tree says the state is "shell" and the published file says "build-up"`},
		"a repository that moved where it is published": {
			strings.Replace(twoRows, `"repository":"Flowfin/jellyfin-plugin-beta"`,
				`"repository":"Elsewhere/jellyfin-plugin-beta"`, 1),
			`this tree says the repository is "Flowfin/jellyfin-plugin-beta" and the published file says "Elsewhere/jellyfin-plugin-beta"`},
		"a row published that this tree does not carry": {
			strings.Replace(twoRows, "\n]",
				`,{"id":"gamma","repository":"Flowfin/jellyfin-plugin-gamma","summary":"What gamma does","state":"shell"}]`, 1),
			"the published file carries this row and this tree does not"},
		"a row this tree carries that is not published": {
			`[{"id":"alpha","repository":"Flowfin/jellyfin-plugin-alpha","summary":"What alpha does","state":"build-up"}]`,
			"this tree carries this row and the published file does not"},
	} {
		var log strings.Builder
		err := Run(tree(t, twoRows), answers(tc.published), &log)
		if err == nil {
			t.Errorf("%s: the run passed:\n%s", name, log.String())
			continue
		}
		if !strings.Contains(err.Error(), "differ from what is published") {
			t.Errorf("%s: the run failed with %q, which is not the difference", name, err)
		}
		if !strings.Contains(log.String(), tc.says) {
			t.Errorf("%s: the run does not name what differs; it said:\n%s", name, log.String())
		}
	}
}

// A sentence edited on one side names the row it is about, which the case above
// asserts only loosely for that one shape. It is separated because the sentence
// is the field most likely to be edited where it is published and least likely
// to be noticed here.
func TestAnEditedSentenceNamesTheRowAndBothSentences(t *testing.T) {
	published := strings.Replace(twoRows, "What alpha does", "What alpha really does", 1)

	var log strings.Builder
	if err := Run(tree(t, twoRows), answers(published), &log); err == nil {
		t.Fatalf("the run passed over an edited sentence:\n%s", log.String())
	}
	for _, want := range []string{
		"alpha: BEHIND",
		`this tree says the sentence is "What alpha does"`,
		`the published file says "What alpha really does"`,
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q; it said:\n%s", want, log.String())
		}
	}
}

// A fetch that did not come back reds the run rather than passing it. A copy
// that was not compared is not a current one, and this is the failure the whole
// comparison exists against.
func TestAFetchThatFailedRedsTheRun(t *testing.T) {
	boom := errors.New("dial tcp: lookup failed")

	var log strings.Builder
	err := Run(tree(t, twoRows), func() ([]byte, error) { return nil, boom }, &log)
	if err == nil {
		t.Fatalf("the run passed over a fetch that failed:\n%s", log.String())
	}
	if !strings.Contains(err.Error(), "could not be read") {
		t.Errorf("the run failed with %q, which does not say the file was not read", err)
	}
	if !strings.Contains(log.String(), "UNRESOLVED") || !strings.Contains(log.String(), boom.Error()) {
		t.Errorf("the run does not report the failure it met; it said:\n%s", log.String())
	}
}

// Nothing published at the address reds the run and reports it as off, which is
// the state this comparison is in today and the one it must not report as
// green. It is a state of its own rather than a fetch failure, because it is
// the one of the two that ends by somebody publishing a file.
func TestNothingPublishedIsReportedAsOffRatherThanGreen(t *testing.T) {
	var log strings.Builder
	err := Run(tree(t, twoRows), func() ([]byte, error) { return nil, ErrNotPublished }, &log)
	if err == nil {
		t.Fatalf("the run passed having compared nothing:\n%s", log.String())
	}
	if !strings.Contains(err.Error(), "compared nothing") {
		t.Errorf("the run failed with %q, which does not say it compared nothing", err)
	}
	for _, want := range []string{
		"OFF, nothing answered at " + Published,
		"what ends this state is that file being published",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q; it said:\n%s", want, log.String())
		}
	}
}

// A published file this tree's parser would refuse is still a published file the
// copy has fallen behind, so the comparison reads it leniently and reports the
// difference rather than hiding it behind a second problem. Bytes that are not
// an array of rows at all are the other case, and that one is unresolved.
func TestThePublishedFileIsReadLenientlyAndUnreadableBytesAreUnresolved(t *testing.T) {
	// A row the parser refuses: its identifier does not match its repository
	// name. The comparison still reports it as a row this tree does not
	// carry.
	odd := `[{"id":"alpha","repository":"Flowfin/jellyfin-plugin-alpha","summary":"What alpha does","state":"build-up"},
	         {"id":"beta","repository":"Flowfin/jellyfin-plugin-beta","summary":"What beta does","state":"shell"},
	         {"id":"wishes","repository":"Flowfin/jellyfin-plugin-elsewhere","summary":"What it does","state":"shell"}]`
	var log strings.Builder
	if err := Run(tree(t, twoRows), answers(odd), &log); err == nil {
		t.Fatalf("the run passed over a published row this tree does not carry:\n%s", log.String())
	}
	if !strings.Contains(log.String(), "wishes: BEHIND") {
		t.Errorf("the run does not name the published row; it said:\n%s", log.String())
	}

	var second strings.Builder
	err := Run(tree(t, twoRows), answers(`{"id":"alpha"}`), &second)
	if err == nil {
		t.Fatalf("the run passed over bytes that are not a roster:\n%s", second.String())
	}
	if !strings.Contains(second.String(), "UNRESOLVED") {
		t.Errorf("the run does not report the read as unresolved; it said:\n%s", second.String())
	}
}

// A tree with no pinned copy is refused rather than compared against nothing.
func TestATreeWithNoPinnedCopyIsRefused(t *testing.T) {
	var log strings.Builder
	err := Run(t.TempDir(), answers(twoRows), &log)
	if err == nil {
		t.Fatal("the run passed over a tree carrying no pinned copy")
	}
	if !strings.Contains(err.Error(), site.RosterFile) {
		t.Errorf("the run failed with %q, which does not name the copy it could not read", err)
	}
}

// What the comparison leaves alone, in the direction that costs if it is wrong.
// The order of the rows is what the site presents them in and is not a fact the
// two files have to agree about row by row, so a published file carrying the
// same rows in another order is not behind.
func TestTheOrderOfTheRowsIsNotADifference(t *testing.T) {
	reversed := `[
	  {"id":"beta","repository":"Flowfin/jellyfin-plugin-beta","summary":"What beta does","state":"shell"},
	  {"id":"alpha","repository":"Flowfin/jellyfin-plugin-alpha","summary":"What alpha does","state":"build-up"}
	]`
	var log strings.Builder
	if err := Run(tree(t, twoRows), answers(reversed), &log); err != nil {
		t.Fatalf("the run refused the same rows in another order: %v\n%s", err, log.String())
	}
}

// Compare over the rows themselves, so the shapes above are held to the
// function rather than to the wording of a run.
func TestCompareReportsInIdentifierOrder(t *testing.T) {
	pinned := []roster.Entry{{ID: "zulu"}, {ID: "alpha"}, {ID: "mike"}}
	got := Compare(pinned, nil)
	if len(got) != 3 {
		t.Fatalf("Compare reported %d difference(s) for three rows against none: %v", len(got), got)
	}
	if got[0].ID != "alpha" || got[1].ID != "mike" || got[2].ID != "zulu" {
		t.Errorf("Compare reported %v, which is not identifier order", got)
	}
}
