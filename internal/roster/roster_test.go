// The suite over the door the site's content comes through.
//
// Every refusal is tripped by a fixture that trips exactly it, because a
// fixture breaking two rules proves neither: a red run over it would be
// satisfied by whichever check happened to fire first, and the check the
// fixture was written for could be deleted without the run noticing.
package roster

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

const valid = `[
  {
    "id": "watchlist",
    "repository": "Flowfin/jellyfin-plugin-watchlist",
    "summary": "A private per-user watchlist kept on the server, shown by clients that were never changed",
    "state": "build-up"
  },
  {
    "id": "sso",
    "repository": "Flowfin/jellyfin-plugin-sso",
    "summary": "Sign in with the identity provider the operator already runs",
    "state": "build-up"
  },
  {
    "id": "requests",
    "repository": "Flowfin/jellyfin-plugin-requests",
    "summary": "Ask for something that is not in the library yet",
    "state": "shell"
  }
]
`

// there answers for the repositories the valid fixture names and for nothing
// else, so a fixture that invents one is refused for that reason rather than
// passing because the suite was generous.
func there(repository string) (bool, error) {
	switch repository {
	case "Flowfin/jellyfin-plugin-watchlist",
		"Flowfin/jellyfin-plugin-sso",
		"Flowfin/jellyfin-plugin-requests":
		return true, nil
	}
	return false, nil
}

// reasons pulls the list out of a refusal, and fails the test if what came back
// was not one.
func reasons(t *testing.T, err error) []string {
	t.Helper()
	var refusal *Refusal
	if !errors.As(err, &refusal) {
		t.Fatalf("the error is %v, which is not a refusal carrying its reasons", err)
	}
	return refusal.Reasons
}

// The neighbour of every fixture below: a file that breaks none of the rules
// parses into the rows it declares.
func TestAValidRosterParsesIntoItsRows(t *testing.T) {
	entries, err := Parse([]byte(valid), there)
	if err != nil {
		t.Fatalf("a roster that breaks nothing was refused: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("the roster parsed into %d row(s), and it declares 3", len(entries))
	}
	first := entries[0]
	if first.ID != "watchlist" || first.Repository != "Flowfin/jellyfin-plugin-watchlist" || first.State != BuildUp {
		t.Errorf("the first row read as %+v, which is not what the file says", first)
	}
	if entries[2].State != Shell {
		t.Errorf("the third row reads state %q, and the file declares %q", entries[2].State, Shell)
	}
}

// Each thing the parser refuses, in the smallest form somebody would actually
// write, each tripping exactly one reason, and each reason opening with the row
// it is about so a reader opens the file at a line rather than bisecting it.
//
// `locator` is the whole of what the reason opens with and not a fragment of
// it, because a fragment does not hold the number. A reason saying `row 1`
// where the fixture broke the third row still contains the identifier, still
// contains the word row, and sends somebody to the wrong line; the two ways of
// getting that number wrong are counting from zero and counting the wrong row,
// and neither survives a comparison against the whole opening. Where the row
// carries no identifier the parser names the number alone, and that form has
// its own fixture below rather than being reached by accident.
func TestEachRefusalIsTrippedByItsOwnFixture(t *testing.T) {
	for name, c := range map[string]struct{ file, says, locator string }{
		"a file that is not JSON": {
			`[{"id": "watchlist",]`, "not the array of rows", ""},
		"a file that is an object rather than an array": {
			`{"id": "watchlist"}`, "not the array of rows", ""},
		"a field the schema does not name": {
			strings.Replace(valid, `"state": "shell"`, `"state": "shell",
    "ships": "yes"`, 1),
			`the field "ships"`, "row 3, requests"},
		"a required field missing": {
			strings.Replace(valid, `,
    "state": "shell"`, "", 1),
			"has no state", "row 3, requests"},
		"a sentence with nothing in it": {
			strings.Replace(valid, `"Ask for something that is not in the library yet"`, `""`, 1),
			"empty sentence", "row 3, requests"},
		"an identifier that is there and empty": {
			strings.Replace(valid, `"id": "requests"`, `"id": ""`, 1),
			"carries an empty identifier", "row 3"},
		"a repository that is there and empty": {
			strings.Replace(valid, `"repository": "Flowfin/jellyfin-plugin-requests"`, `"repository": ""`, 1),
			"carries an empty repository", "row 3, requests"},
		"the same identifier twice": {
			strings.Replace(valid, `"id": "requests",
    "repository": "Flowfin/jellyfin-plugin-requests"`, `"id": "sso",
    "repository": "Flowfin/jellyfin-plugin-sso"`, 1),
			"repeats the identifier declared on row 2", "row 3, sso"},
		"an identifier that is not the repository name": {
			strings.Replace(valid, `"id": "requests"`, `"id": "wishes"`, 1),
			"jellyfin-plugin-wishes", "row 3, wishes"},
		"a state outside the two words": {
			strings.Replace(valid, `"state": "shell"`, `"state": "ships"`, 1),
			`the state "ships"`, "row 3, requests"},
		"a repository that is not there": {
			strings.Replace(strings.Replace(valid,
				`"id": "requests"`, `"id": "wishes"`, 1),
				`"repository": "Flowfin/jellyfin-plugin-requests"`, `"repository": "Flowfin/jellyfin-plugin-wishes"`, 1),
			"which is not there", "row 3, wishes"},
		"a file holding no row": {
			`[]`, "holds no row", ""},
	} {
		entries, err := Parse([]byte(c.file), there)
		if err == nil {
			t.Errorf("%s parsed into %d row(s) rather than being refused", name, len(entries))
			continue
		}
		got := reasons(t, err)
		if len(got) != 1 {
			t.Errorf("%s was refused for %d reason(s), and a fixture tripping more than one proves none of them: %v", name, len(got), got)
			continue
		}
		if !strings.Contains(got[0], c.says) {
			t.Errorf("%s was refused with %q, which does not say %q", name, got[0], c.says)
		}
		if c.locator != "" && !strings.HasPrefix(got[0], c.locator+": ") {
			t.Errorf("%s was refused with %q, and the row it is about is %q", name, got[0], c.locator)
		}
	}
}

// A read with no way to ask the question refuses rather than answering it by
// leaving it out. This is the shape a check takes when the thing it depends on
// is absent, and reporting a roster as read when one of its rules was never
// applied is the failure the rest of this repository is written against.
func TestAReadWithNothingToAskRefusesRatherThanSkipping(t *testing.T) {
	_, err := Parse([]byte(valid), nil)
	if err == nil {
		t.Fatal("a read with nothing to ask about a repository parsed the file")
	}
	if got := reasons(t, err); !strings.Contains(got[0], "would report a roster it had not checked") {
		t.Errorf("the refusal reads %q, which does not say why the read stopped", got[0])
	}
}

// A question that could not be answered is not a repository that is there. The
// two collapse into one the moment an error is treated as a false, and what
// that produces is a roster reported as checked against a host that was down.
func TestAnUnansweredQuestionIsNotAnExistingRepository(t *testing.T) {
	down := func(string) (bool, error) { return false, fmt.Errorf("the host did not answer") }

	_, err := Parse([]byte(valid), down)
	if err == nil {
		t.Fatal("a roster was parsed with nothing able to answer whether its repositories exist")
	}
	got := reasons(t, err)
	if len(got) != 3 {
		t.Fatalf("%d row(s) were refused, and the file has 3: %v", len(got), got)
	}
	for _, r := range got {
		if !strings.Contains(r, "could not be answered") {
			t.Errorf("the refusal reads %q, which reports an unanswered question as something else", r)
		}
		if strings.Contains(r, "which is not there") {
			t.Errorf("the refusal reads %q, which reports an unanswered question as a missing repository", r)
		}
	}
}

// Every reason, not the first. A file with two broken rows is two repairs, and
// reporting one of them at a time costs a run each.
func TestEveryReasonIsReportedRatherThanTheFirst(t *testing.T) {
	twice := strings.Replace(strings.Replace(valid,
		`"state": "build-up"`, `"state": "ships"`, 1),
		`"Ask for something that is not in the library yet"`, `""`, 1)

	_, err := Parse([]byte(twice), there)
	if err == nil {
		t.Fatal("a roster with two broken rows was parsed")
	}
	got := reasons(t, err)
	if len(got) != 2 {
		t.Fatalf("a file with two broken rows was refused for %d reason(s): %v", len(got), got)
	}
	joined := strings.Join(got, "\n")
	for _, want := range []string{"watchlist", "requests"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the refusal does not name the row %s; it said:\n%s", want, joined)
		}
	}
}

// The error a caller prints says how many things were wrong and what each of
// them was, because that message is the whole of what somebody repairing the
// file has to work from.
func TestTheRefusalReadsAsSomethingSomebodyCanActOn(t *testing.T) {
	_, err := Parse([]byte(strings.Replace(valid, `"state": "shell"`, `"state": "ships"`, 1)), there)
	if err == nil {
		t.Fatal("a roster with a state outside the two words was parsed")
	}
	for _, want := range []string{"1 reason(s)", "row 3, requests"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error reads %q, which does not carry %q", err, want)
		}
	}
}
