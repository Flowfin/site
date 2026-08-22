// The suite over the reference reader and the run.
//
// Every state the run keeps apart is produced by a reader that returns that
// state on demand. A tracker cannot be asked for a board whose label has been
// renamed, or for an issue naming a number nobody ever opened, and those are
// the states this run exists to keep apart, which is why the reading is a
// parameter rather than a call inside the run.
package blockers

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// board is a reading carrying the label and the rows given.
func board(rows ...Issue) Reader {
	return func() (Board, error) {
		return Board{Labels: []string{"documentation", Label, "security"}, Issues: rows}, nil
	}
}

func blockedIssue(number int, body string) Issue {
	return Issue{Number: number, Title: fmt.Sprintf("issue %d", number), State: "open", Body: body, Labels: []string{"enhancement", Label}}
}

func openIssue(number int) Issue {
	return Issue{Number: number, Title: fmt.Sprintf("issue %d", number), State: "open"}
}

func closedIssue(number int) Issue {
	return Issue{Number: number, Title: fmt.Sprintf("issue %d", number), State: "closed"}
}

func run(t *testing.T, read Reader) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Run(read, &out)
	return out.String(), err
}

func TestReferencesReadsEveryNumberOnceAndInOrder(t *testing.T) {
	got := References("Depends on #26, the landing page, and on #5 which has closed. Again #26.")
	want := []int{5, 26}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("References returned %v, want %v", got, want)
	}
}

func TestReferencesKeepsOutTheShapesThatAreNotIssueNumbers(t *testing.T) {
	// An escaped apostrophe, an address fragment and a colour literal are the
	// three ways a body carries a hash followed by digits without meaning an
	// issue. Reading any of them as one turns a blocked issue into an
	// unresolved reference and reds a run for nothing.
	for _, body := range []string{
		"the reader&#39;s browser",
		"https://example.test/docs/parity.md#12",
		"the token is #1a2b3c and nothing else",
		"see decisions/0002-what-docker-is-for.md#3",
	} {
		if got := References(body); len(got) != 0 {
			t.Errorf("References(%q) returned %v, want nothing", body, got)
		}
	}
}

func TestReferencesReadsANumberAtTheStartOfTheBody(t *testing.T) {
	if got := References("#53 decides the version scheme."); fmt.Sprint(got) != fmt.Sprint([]int{53}) {
		t.Fatalf("References returned %v, want [53]", got)
	}
}

func TestRunPassesWhenEveryBlockedIssueNamesSomethingStillOpen(t *testing.T) {
	out, err := run(t, board(
		blockedIssue(36, "Depends on #35, the rendered timing budget."),
		openIssue(35),
	))
	if err != nil {
		t.Fatalf("a board where every blocked issue points at an open one should pass, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "#36: blocked, waiting on #35 (open issue)") {
		t.Fatalf("the run should say what #36 waits on, got:\n%s", out)
	}
	if !strings.Contains(out, "1 issue(s) read, 0 naming no issue, 0 no longer blocked, 0 unresolved.") {
		t.Fatalf("the run should count what it read, got:\n%s", out)
	}
}

func TestRunRefusesABlockedIssueWhoseBodyNamesNoNumber(t *testing.T) {
	out, err := run(t, board(
		blockedIssue(75, "Depends on the roster parser issue."),
		closedIssue(21),
	))
	if err == nil {
		t.Fatalf("an issue naming its dependency only in prose should red the run, got:\n%s", out)
	}
	if !strings.Contains(out, "#75: NAMES NO ISSUE") {
		t.Fatalf("the failure should name #75, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "1 name no issue by number") {
		t.Fatalf("the error should count it, got %v", err)
	}
}

func TestRunRefusesABlockedIssueWhoseNamedIssuesHaveAllClosed(t *testing.T) {
	// The half this check exists for. Nothing is holding #50 and every reader
	// of the board walks past it, because the label says otherwise.
	out, err := run(t, board(
		blockedIssue(50, "Depends on #48, the privacy page, and on #37, the external origin refusal."),
		closedIssue(48),
		closedIssue(37),
	))
	if err == nil {
		t.Fatalf("an issue whose dependencies have all closed should red the run, got:\n%s", out)
	}
	if !strings.Contains(out, "#50: NO LONGER BLOCKED, everything it names has closed: #37 (closed issue), #48 (closed issue)") {
		t.Fatalf("the failure should name #50 and both closed dependencies, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "1 are no longer blocked and still say they are") {
		t.Fatalf("the error should count it, got %v", err)
	}
}

func TestRunKeepsAnIssueBlockedWhenOneOfSeveralIsStillOpen(t *testing.T) {
	// The near miss for the row above. One closed dependency beside one open
	// one is the ordinary state of a blocked issue, and a run reading "some
	// have closed" as "no longer blocked" would red on most of the board.
	out, err := run(t, board(
		blockedIssue(58, "Depends on #53, the release workflow, and on #3 which has closed."),
		openIssue(53),
		closedIssue(3),
	))
	if err != nil {
		t.Fatalf("one open dependency should keep #58 blocked, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "#58: blocked, waiting on #53 (open issue)") {
		t.Fatalf("the run should name the open half only, got:\n%s", out)
	}
}

func TestRunReportsAReferenceThisBoardDoesNotHoldAsUnresolved(t *testing.T) {
	// A number nobody opened resolves to nothing, and reading it as closed
	// would report the issue as available work on the strength of a typo.
	out, err := run(t, board(
		blockedIssue(80, "Depends on #79 and on #4242."),
		closedIssue(79),
	))
	if err == nil {
		t.Fatalf("a reference this board does not hold should red the run, got:\n%s", out)
	}
	if !strings.Contains(out, "#80: UNRESOLVED, it names #4242") {
		t.Fatalf("the failure should name the reference, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "1 name a number this board does not hold") {
		t.Fatalf("the error should count it, got %v", err)
	}
	if strings.Contains(out, "NO LONGER BLOCKED") {
		t.Fatalf("an unresolved reference is not a cleared one, got:\n%s", out)
	}
}

func TestRunIgnoresAnIssueNamingItsOwnNumber(t *testing.T) {
	// An issue that mentions its own number is open by definition, so
	// counting it as a dependency makes it block itself and it can never be
	// reported as cleared. This one waits on nothing: #35 has closed.
	out, err := run(t, board(
		blockedIssue(90, "As #90 says, this depends on #35."),
		closedIssue(35),
	))
	if err == nil {
		t.Fatalf("#90 waits on nothing but itself and should be reported as cleared, got:\n%s", out)
	}
	if !strings.Contains(out, "#90: NO LONGER BLOCKED, everything it names has closed: #35 (closed issue)") {
		t.Fatalf("the run should name only the real dependency, got:\n%s", out)
	}
	if strings.Contains(out, "#90 (open issue)") {
		t.Fatalf("an issue should not appear as its own dependency, got:\n%s", out)
	}
}

func TestRunNamesAPullRequestAsOneRatherThanAsAnIssue(t *testing.T) {
	out, err := run(t, board(
		blockedIssue(72, "Landed by #165."),
		Issue{Number: 165, Title: "a change", State: "closed", PullRequest: true},
	))
	if err == nil {
		t.Fatalf("a merged pull request is a closed reference and should red the run, got:\n%s", out)
	}
	if !strings.Contains(out, "#165 (closed pull request)") {
		t.Fatalf("the run should say the reference is a pull request, got:\n%s", out)
	}
}

func TestRunDoesNotJudgeAPullRequestCarryingTheLabel(t *testing.T) {
	out, err := run(t, board(
		Issue{Number: 179, Title: "a change", State: "open", Body: "no number here", PullRequest: true, Labels: []string{Label}},
		blockedIssue(36, "Depends on #35."),
		openIssue(35),
	))
	if err != nil {
		t.Fatalf("a pull request is not an issue this check judges, got %v\n%s", err, out)
	}
	if strings.Contains(out, "#179") {
		t.Fatalf("the run should not report on a pull request, got:\n%s", out)
	}
}

func TestRunDoesNotJudgeAClosedIssue(t *testing.T) {
	out, err := run(t, board(
		Issue{Number: 25, Title: "done", State: "closed", Body: "Depends on the roster parser issue.", Labels: []string{Label}},
		blockedIssue(36, "Depends on #35."),
		openIssue(35),
	))
	if err != nil {
		t.Fatalf("a closed issue is not work and is not judged, got %v\n%s", err, out)
	}
	if strings.Contains(out, "#25") {
		t.Fatalf("the run should not report on a closed issue, got:\n%s", out)
	}
}

func TestRunPassesABoardOnWhichNothingIsBlocked(t *testing.T) {
	out, err := run(t, board(openIssue(35), closedIssue(3)))
	if err != nil {
		t.Fatalf("a board where nothing carries the label is a real zero, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "0 issue(s) read, 0 naming no issue, 0 no longer blocked, 0 unresolved.") {
		t.Fatalf("the run should say it read nothing, got:\n%s", out)
	}
}

func TestRunRefusesToPassABoardThatDeclaresNoSuchLabel(t *testing.T) {
	// The near miss for the row above, and the reason the reading carries the
	// labels at all. A renamed label answers every issue query with nothing,
	// which is the same empty result as a board on which nothing is blocked.
	read := func() (Board, error) {
		return Board{Labels: []string{"documentation", "security"}, Issues: []Issue{openIssue(35)}}, nil
	}
	out, err := run(t, read)
	if err == nil {
		t.Fatalf("a board declaring no such label should red the run, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "declares no label") {
		t.Fatalf("the error should say the label is absent, got %v", err)
	}
}

func TestRunRefusesToPassAReadingThatFailed(t *testing.T) {
	read := func() (Board, error) { return Board{}, errors.New("the tracker answered 403 Forbidden") }
	out, err := run(t, read)
	if err == nil {
		t.Fatalf("a reading that failed is not a board on which nothing is wrong, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "resolved nothing") {
		t.Fatalf("the error should say the run resolved nothing, got %v", err)
	}
}

func TestRunCountsEveryStateSeparatelyOnOneBoard(t *testing.T) {
	// The three failures are separate counts rather than one, because the
	// repairs are different: a body to edit, a label to remove, and a number
	// that is wrong.
	out, err := run(t, board(
		blockedIssue(75, "Depends on the roster parser issue."),
		blockedIssue(50, "Depends on #48."),
		blockedIssue(80, "Depends on #4242."),
		blockedIssue(36, "Depends on #35."),
		closedIssue(48),
		openIssue(35),
	))
	if err == nil {
		t.Fatalf("three of these four are wrong and the run should be red, got:\n%s", out)
	}
	if !strings.Contains(out, "4 issue(s) read, 1 naming no issue, 1 no longer blocked, 1 unresolved.") {
		t.Fatalf("the run should count the three states apart, got:\n%s", out)
	}
	for _, want := range []string{"1 name no issue by number", "1 are no longer blocked and still say they are", "1 name a number this board does not hold"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the error should carry %q, got %v", want, err)
		}
	}
}
