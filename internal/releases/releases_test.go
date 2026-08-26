// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the record the shipping state is read from.
//
// No case here reaches the network, and neither does this package: what asks a
// repository anything is internal/releases/github, which imports this one and
// is imported by nothing a build reads. The door it comes through is a
// parameter, so every case below hands the refresh something it wrote itself.
package releases

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The rule decisions/0009 fixes, read off the two counts rather than off a
// verdict somebody recorded. The prerelease-only case is the one the record
// exists to get right: it is the shape the plugin that ships arrives in, and a
// rule reading `has published anything` would call it shipping.
func TestOnlyAFinishedReleaseMeansAPluginShips(t *testing.T) {
	for name, tc := range map[string]struct {
		r    Repository
		want bool
	}{
		"nothing published":        {Repository{}, false},
		"prereleases only":         {Repository{Prereleases: 19}, false},
		"one finished release":     {Repository{Finished: 1}, true},
		"finished and prereleases": {Repository{Finished: 11, Prereleases: 19}, true},
	} {
		if got := tc.r.Ships(); got != tc.want {
			t.Errorf("%s: Ships reported %v, want %v", name, got, tc.want)
		}
	}
}

// The record answers the existence question as well, and it answers it from the
// set it carries rather than from a second file. A repository that answered with
// its release list is a repository that is there.
func TestTheRecordAnswersWhetherARepositoryIsThere(t *testing.T) {
	rec := Record{Taken: "2026-01-02", Repositories: map[string]Repository{
		"Flowfin/jellyfin-plugin-alpha": {},
	}}
	if !rec.Known("Flowfin/jellyfin-plugin-alpha") {
		t.Error("a repository the record carries was reported as not there")
	}
	if rec.Known("Flowfin/jellyfin-plugin-beta") {
		t.Error("a repository the record does not carry was reported as there")
	}
}

// Every shape a build must not carry on past, each refused in its own words
// because each is a different repair.
func TestTheReadFailsClosed(t *testing.T) {
	for name, tc := range map[string]struct{ body, says string }{
		"bytes that are not the record": {
			`{"taken":`, "reading " + File},
		"a record carrying no repository": {
			`{"taken":"2026-01-02","repositories":{}}`, "carries no repository"},
		"a record with no repositories field at all": {
			`{"taken":"2026-01-02"}`, "carries no repository"},
		"a record saying nothing about when it was taken": {
			`{"repositories":{"a/b":{"finished":0,"prereleases":0}}}`, "says nothing about when it was taken"},
		"a record taken at whitespace": {
			`{"taken":"  ","repositories":{"a/b":{"finished":0,"prereleases":0}}}`, "says nothing about when it was taken"},
		"a count a release list cannot answer": {
			`{"taken":"2026-01-02","repositories":{"a/b":{"finished":-1,"prereleases":0}}}`, "negative count"},
		"a finished release with no server generation beside it": {
			`{"taken":"2026-01-02","repositories":{"a/b":{"finished":1,"prereleases":0}}}`,
			"no server generation"},
		"a server generation for a plugin nobody can install": {
			`{"taken":"2026-01-02","repositories":{"a/b":{"finished":0,"prereleases":2,"generations":["10.11"]}}}`,
			"no finished release and the generation(s) 10.11"},
		"a finished release stating no generation, counted as fewer than were published": {
			`{"taken":"2026-01-02","repositories":{"a/b":{"finished":3,"prereleases":0,"generations-unstated":2}}}`,
			"only 2 of them stating none"},
		"more releases stating no generation than were published": {
			`{"taken":"2026-01-02","repositories":{"a/b":{"finished":1,"prereleases":0,"generations":["10.11"],"generations-unstated":2}}}`,
			"more releases than were published"},
		"releases stating no generation where none were published": {
			`{"taken":"2026-01-02","repositories":{"a/b":{"finished":0,"prereleases":0,"generations-unstated":1}}}`,
			"more releases than were published"},
		"a negative count of releases stating no generation": {
			`{"taken":"2026-01-02","repositories":{"a/b":{"finished":1,"prereleases":0,"generations":["10.11"],"generations-unstated":-1}}}`,
			"negative count of releases stating no generation"},
		"a server generation that names nothing": {
			`{"taken":"2026-01-02","repositories":{"a/b":{"finished":1,"prereleases":0,"generations":["  "]}}}`,
			"empty server generation"},
	} {
		_, err := Read([]byte(tc.body))
		if err == nil {
			t.Errorf("%s was read rather than refused", name)
			continue
		}
		if !strings.Contains(err.Error(), tc.says) {
			t.Errorf("%s was refused with %q, which does not say %q", name, err, tc.says)
		}
	}
}

// The neighbour of every case above: a record breaking none of the rules reads
// into what it declares.
func TestARecordThatBreaksNothingReadsIntoWhatItDeclares(t *testing.T) {
	rec, err := Read([]byte(`{"taken":"2026-01-02","command":"a command",
		"repositories":{"a/b":{"finished":2,"prereleases":3,"generations":["10.11","12.0"]}}}`))
	if err != nil {
		t.Fatalf("a record breaking nothing was refused: %v", err)
	}
	if rec.Taken != "2026-01-02" || rec.Command != "a command" {
		t.Errorf("the record read as taken %q by %q", rec.Taken, rec.Command)
	}
	if got := rec.Repositories["a/b"]; got.Finished != 2 || got.Prereleases != 3 {
		t.Errorf("the record read a/b as %+v", got)
	}
	if got := rec.Repositories["a/b"].Generations; len(got) != 2 || got[0] != "10.11" || got[1] != "12.0" {
		t.Errorf("the record read the generations as %q, and the order is the file's", got)
	}
}

// The third state reads rather than being refused. A repository whose finished
// releases all publish nothing about which server they are for is a real state
// and not a hole: the plugin that ships today published four such releases
// before it published the metadata a generation is read out of, and a rule
// refusing it could not record that repository at all.
func TestEveryFinishedReleaseStatingNoGenerationIsAState(t *testing.T) {
	rec, err := Read([]byte(`{"taken":"2026-01-02","command":"a command",
		"repositories":{"a/b":{"finished":4,"prereleases":0,"generations-unstated":4}}}`))
	if err != nil {
		t.Fatalf("a record in the third state was refused: %v", err)
	}
	got := rec.Repositories["a/b"]
	if !got.Ships() {
		t.Error("a repository with four finished releases was read as not shipping")
	}
	if len(got.Generations) != 0 || got.Unstated != 4 {
		t.Errorf("the record read a/b as %+v", got)
	}
}

// A refresh that could not read one repository writes nothing, because a record
// missing a repository is a record that says the repository is not there. A
// network failure turned into a claim about somebody's repository is the one
// failure a recorded answer must not have.
func TestARefreshThatCouldNotAskOneRepositoryProducesNoRecord(t *testing.T) {
	boom := errors.New("the service answered 503")
	_, err := Refresh([]string{"a/b", "c/d"}, "2026-01-02", "a command",
		func(repository string) (Repository, error) {
			if repository == "c/d" {
				return Repository{}, boom
			}
			return Repository{Finished: 1}, nil
		})
	if err == nil {
		t.Fatal("a refresh that could not read a repository produced a record")
	}
	if !errors.Is(err, boom) {
		t.Errorf("the failure reads %q, which does not carry what went wrong", err)
	}
	if !strings.Contains(err.Error(), "c/d") {
		t.Errorf("the failure reads %q, which does not name the repository it could not ask", err)
	}
}

// A refresh with nothing to ask about is refused rather than producing an empty
// record, which the read above would then refuse one step later with a message
// about a file rather than about the run that wrote it.
func TestARefreshWithNothingToAskAboutIsRefused(t *testing.T) {
	if _, err := Refresh(nil, "2026-01-02", "a command",
		func(string) (Repository, error) { return Repository{}, nil }); err == nil {
		t.Fatal("a refresh with nothing named produced a record")
	}
}

// What a refresh asks and what it records, in one pass. The order the caller
// names the repositories in does not reach the record, because the file is an
// object and a caller's order would produce a diff on every run that changed
// nothing.
func TestARefreshRecordsWhatEachRepositoryAnswered(t *testing.T) {
	asked := map[string]int{}
	rec, err := Refresh([]string{"c/d", "a/b"}, "2026-01-02", "a command",
		func(repository string) (Repository, error) {
			asked[repository]++
			if repository == "a/b" {
				return Repository{Finished: 1, Prereleases: 2}, nil
			}
			return Repository{}, nil
		})
	if err != nil {
		t.Fatalf("a refresh that could read everything was refused: %v", err)
	}
	for _, name := range []string{"a/b", "c/d"} {
		if asked[name] != 1 {
			t.Errorf("%s was asked %d time(s), want once", name, asked[name])
		}
	}
	if got := rec.Repositories["a/b"]; got.Finished != 1 || got.Prereleases != 2 {
		t.Errorf("the record holds a/b as %+v", got)
	}
	if rec.Taken != "2026-01-02" || rec.Command != "a command" {
		t.Errorf("the record says it was taken %q by %q", rec.Taken, rec.Command)
	}
}

// The bytes a refresh writes are the bytes a second refresh over the same
// answers writes. A record that reordered itself would produce a diff on every
// run, and a diff that is always there is a diff nobody reads.
func TestTwoRefreshesOverTheSameAnswersWriteTheSameBytes(t *testing.T) {
	answers := func(repository string) (Repository, error) {
		return Repository{Finished: len(repository)}, nil
	}
	names := []string{"c/d", "a/b", "e/f"}

	first, err := Refresh(names, "2026-01-02", "a command", answers)
	if err != nil {
		t.Fatalf("the first refresh was refused: %v", err)
	}
	second, err := Refresh([]string{"e/f", "c/d", "a/b"}, "2026-01-02", "a command", answers)
	if err != nil {
		t.Fatalf("the second refresh was refused: %v", err)
	}

	a, err := Marshal(first)
	if err != nil {
		t.Fatalf("writing the first record: %v", err)
	}
	b, err := Marshal(second)
	if err != nil {
		t.Fatalf("writing the second record: %v", err)
	}
	if string(a) != string(b) {
		t.Errorf("two refreshes over the same answers wrote different bytes:\n%s\n%s", a, b)
	}
	if !strings.HasSuffix(string(a), "\n") {
		t.Error("the record does not end with a newline, which every text file in this tree does")
	}
}

// Every field an entry carries is compared, one field at a time, and an entry
// is not different from itself. The verb's case below moves two fields at once,
// so a comparison that had stopped reading one of them would still report that
// entry as MOVED and nothing in the suite would say otherwise. What a run of
// this file reports is what somebody reads instead of diffing the record, so a
// field that dropped out of the comparison is a change that gets recorded and
// announced as unchanged.
func TestOneFieldIsEnoughToMakeTwoEntriesDifferent(t *testing.T) {
	base := Repository{Finished: 2, Prereleases: 3, Unstated: 1, Generations: []string{"10.11", "12.0"}}
	if !same(base, base) {
		t.Error("an entry was reported as saying something other than itself")
	}
	for name, other := range map[string]Repository{
		"one more finished release":              {Finished: 3, Prereleases: 3, Unstated: 1, Generations: []string{"10.11", "12.0"}},
		"one more prerelease":                    {Finished: 2, Prereleases: 4, Unstated: 1, Generations: []string{"10.11", "12.0"}},
		"one more stating no server generation":  {Finished: 2, Prereleases: 3, Unstated: 2, Generations: []string{"10.11", "12.0"}},
		"one generation fewer":                   {Finished: 2, Prereleases: 3, Unstated: 1, Generations: []string{"10.11"}},
		"as many generations, named differently": {Finished: 2, Prereleases: 3, Unstated: 1, Generations: []string{"10.11", "12.1"}},
	} {
		if same(base, other) {
			t.Errorf("%s: two entries that differ were reported as saying the same thing", name)
		}
	}
}

// The verb over a tree, end to end, and what it says about what moved. The
// report is the whole point of the run: a refresh that wrote a file and said
// nothing would leave somebody diffing to find out whether anything changed.
func TestTheVerbWritesTheRecordAndSaysWhatMoved(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(filepath.FromSlash(File))), 0o755); err != nil {
		t.Fatalf("preparing the tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(File)),
		[]byte(`{"taken":"2026-01-01","command":"a command","repositories":{
			"a/b":{"finished":0,"prereleases":0},
			"gone/away":{"finished":0,"prereleases":0}}}`), 0o644); err != nil {
		t.Fatalf("writing what was there before: %v", err)
	}

	var log strings.Builder
	err := Run(root, []string{"a/b", "c/d"}, "2026-01-02", "a command",
		func(repository string) (Repository, error) {
			if repository == "a/b" {
				return Repository{Finished: 1, Generations: []string{"10.11"}}, nil
			}
			return Repository{Prereleases: 4}, nil
		}, &log)
	if err != nil {
		t.Fatalf("the verb refused a tree it should have written: %v", err)
	}

	for _, want := range []string{
		"a/b: MOVED, was 0 finished, 0 prerelease(s), now 1 finished, 0 prerelease(s), for 10.11",
		"c/d: NEW, 0 finished, 4 prerelease(s)",
		"gone/away: GONE",
		"2 repository(s) recorded as taken on 2026-01-02, 3 entry(s) moved",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q; it said:\n%s", want, log.String())
		}
	}

	written, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(File)))
	if err != nil {
		t.Fatalf("reading what the verb wrote: %v", err)
	}
	rec, err := Read(written)
	if err != nil {
		t.Fatalf("what the verb wrote is not a record this package reads: %v", err)
	}
	if rec.Known("gone/away") {
		t.Error("the verb kept a repository the roster no longer names")
	}
	if got := rec.Repositories["c/d"]; got.Prereleases != 4 {
		t.Errorf("the verb recorded c/d as %+v", got)
	}
}

// A tree with no record yet is refreshed rather than refused, and the run says
// there was nothing to compare against rather than reporting every repository
// as unchanged.
func TestTheVerbOverATreeWithNoRecordSaysThereWasNothingThere(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(filepath.FromSlash(File))), 0o755); err != nil {
		t.Fatalf("preparing the tree: %v", err)
	}

	var log strings.Builder
	if err := Run(root, []string{"a/b"}, "2026-01-02", "a command",
		func(string) (Repository, error) { return Repository{}, nil }, &log); err != nil {
		t.Fatalf("the verb refused a tree with no record: %v", err)
	}
	if !strings.Contains(log.String(), "nothing readable was there before this run") {
		t.Errorf("the run does not say there was nothing there; it said:\n%s", log.String())
	}
	if !strings.Contains(log.String(), "a/b: NEW") {
		t.Errorf("the run does not report the repository as new; it said:\n%s", log.String())
	}
}
