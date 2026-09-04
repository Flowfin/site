// SPDX-License-Identifier: AGPL-3.0-or-later

// Package blockers reads this board's own tracker and says whether an issue
// declaring itself blocked still points at anything.
//
// One label on this board means "waits on work tracked by another open issue
// on this board". The label cannot say which issue, so the body says it, and
// for most of the population the body said it in prose: "depends on the roster
// parser issue". A dependency written as a description is a dependency nothing
// can follow. Nobody can tell when it stops being true, so it goes on being
// asserted after the thing it names has closed, and the issue sits on the
// board reading as unavailable work while nothing is holding it. Four on this
// board were in that state when the reading that produced this package was
// taken, two of them for a fortnight.
//
// So the check asks two questions of every issue carrying the label, and they
// fail in opposite directions. An issue whose body names no number at all is
// refused: nothing can be resolved for it and no later run will do better. An
// issue whose named issues have all closed is refused too, and that is the
// half that matters, because that issue is available work every reader walks
// past.
//
// The states stay apart rather than collapsing into one count, for the reason
// the comparison over the pins keeps three: a reference this board does not
// hold is unresolved, which is a failure and never a pass. A run that resolved
// nothing and reported agreement is what this exists to prevent.
//
// There is a third reading, and it is the one an issue passes through both
// questions untouched. Neither of them asks whether the issue an issue waits
// for is waiting back. A set in which every member waits on another member of
// the same set is a set nothing outside it can end, and every member of it
// reads as ordinarily blocked, is subtracted from the count of available work
// by its label, and is walked past by every reader for as long as nobody
// notices. Eight of this board's own issues were in one such set when this was
// measured, and the run's last line called them clean.
//
// That reading is REPORTED and refuses nothing, which is the one place this
// package prints a state in capitals without failing on it. The repair is an
// edit to an issue somebody has to decide on, the same repair the two
// questions above ask for, but unlike them it cannot be made by the issue's own
// author alone: which member of the set gives way is a reading of what those
// issues are for. Refusing here would put a red gate on a condition that stands
// until that reading is taken. Whether it should refuse anyway is the open half
// of #234 and is not decided here.
//
// The reading is the whole tracker rather than one query per issue. A body
// names a number that has closed more often than one that has not, and a
// closed issue is absent from the listing an open-issue query answers with, so
// resolving per reference spends the rate limit on the population this run is
// least able to predict the size of.
package blockers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Label is the one this check reads. It is the string the tracker carries
// rather than a description of it, because what the run compares against is
// the label as declared.
const Label = "blocked-on-another-issue"

// Owner and Repo are the board this reads. It is this repository's own
// tracker: the label's own description says "another open issue on this
// board", so a reference resolves here or it does not resolve at all.
const (
	Owner = "Flowfin"
	Repo  = "site"
)

// An Issue is one row of the tracker. A pull request is one too, because the
// tracker answers with both under one numbering and a body naming a number
// cannot say which of the two it meant.
type Issue struct {
	Number      int
	Title       string
	State       string
	Body        string
	Labels      []string
	PullRequest bool
}

// A Board is everything one reading of the tracker returns.
//
// Labels is read as well as the issues, and it is not decoration. An issue
// listing filtered by a label nobody declares comes back empty, and an empty
// result is indistinguishable from a board on which nothing is blocked. One of
// those is the check working and the other is the check reading a name that no
// longer exists, so the run asks which one it has rather than passing on both.
type Board struct {
	Labels []string
	Issues []Issue
}

// A Reader answers with one whole reading. It is a parameter so that the suite
// can produce every state the run has to keep apart, including the ones a
// tracker cannot be asked for on demand.
type Reader func() (Board, error)

// reference matches a number the way a body means one. The leading class is
// what keeps two other shapes out: an escaped character reference, which is
// how a body carries an apostrophe and would otherwise read as issue 39, and a
// fragment on the end of an address, which names a heading and not an issue.
var reference = regexp.MustCompile(`(^|[^0-9A-Za-z_&/#-])#([0-9]+)`)

// References is every issue number a body names, sorted and each one counted
// once.
//
// What it cannot do is judge whether a reference is the dependency or a
// mention of something else, so an issue naming a closed number in passing
// beside its open dependency reads here as still blocked. That direction is
// the safe one: it under-reports the issues this run asks somebody to look at
// rather than sending them to an issue that is genuinely waiting.
func References(body string) []int {
	seen := map[int]bool{}
	for _, m := range reference.FindAllStringSubmatchIndex(body, -1) {
		start, end := m[4], m[5]
		// What follows the digits decides it as much as what precedes them,
		// and it is read here rather than in the pattern because a trailing
		// class consumes the byte the next reference needs in front of it.
		// A colour literal is the shape this keeps out: the digits of #1a2b3c
		// are a prefix of a number nobody wrote.
		if end < len(body) && isWordByte(body[end]) {
			continue
		}
		n, err := strconv.Atoi(body[start:end])
		if err != nil || n == 0 {
			continue
		}
		seen[n] = true
	}
	out := make([]int, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

func isWordByte(b byte) bool {
	switch {
	case b >= '0' && b <= '9', b >= 'A' && b <= 'Z', b >= 'a' && b <= 'z':
		return true
	}
	return b == '_' || b == '-'
}

// Run takes one reading and reports what every blocked issue points at.
//
// It writes nothing anywhere. The repair for everything it finds is an edit to
// an issue, which is somebody deciding what that issue waits for, and a run
// that stripped a label on its own would be taking that decision from a
// regular expression.
func Run(read Reader, out io.Writer) error {
	board, err := read()
	if err != nil {
		return fmt.Errorf("blockers: the tracker could not be read, so this run resolved nothing: %w", err)
	}

	declared := false
	for _, l := range board.Labels {
		if l == Label {
			declared = true
			break
		}
	}
	if !declared {
		return fmt.Errorf("blockers: %s/%s declares no label %q, so an empty result here would mean the label had been renamed rather than that nothing is blocked", Owner, Repo, Label)
	}

	held := map[int]Issue{}
	for _, i := range board.Issues {
		held[i.Number] = i
	}

	var blocked []Issue
	for _, i := range board.Issues {
		if i.PullRequest || i.State != "open" {
			continue
		}
		for _, l := range i.Labels {
			if l == Label {
				blocked = append(blocked, i)
				break
			}
		}
	}
	sort.Slice(blocked, func(a, b int) bool { return blocked[a].Number < blocked[b].Number })

	fmt.Fprintf(out, "blockers: %d open issue(s) carry %s, out of %d row(s) the tracker holds\n", len(blocked), Label, len(board.Issues))

	nameless, cleared, unresolved := 0, 0, 0
	for _, i := range blocked {
		var open, closed, missing []string
		for _, n := range References(i.Body) {
			if n == i.Number {
				continue
			}
			ref, ok := held[n]
			switch {
			case !ok:
				missing = append(missing, "#"+strconv.Itoa(n))
			case ref.State == "open":
				open = append(open, describe(n, ref))
			default:
				closed = append(closed, describe(n, ref))
			}
		}
		switch {
		case len(open)+len(closed)+len(missing) == 0:
			nameless++
			fmt.Fprintf(out, "  #%d: NAMES NO ISSUE, it carries %s and its body has no number to resolve\n", i.Number, Label)
		case len(missing) > 0:
			unresolved++
			fmt.Fprintf(out, "  #%d: UNRESOLVED, it names %s and this board holds no such row\n", i.Number, strings.Join(missing, ", "))
		case len(open) == 0:
			cleared++
			fmt.Fprintf(out, "  #%d: NO LONGER BLOCKED, everything it names has closed: %s\n", i.Number, strings.Join(closed, ", "))
		default:
			fmt.Fprintf(out, "  #%d: blocked, waiting on %s\n", i.Number, strings.Join(open, ", "))
		}
	}

	sets := reciprocal(blocked)
	for _, set := range sets {
		fmt.Fprintf(out, "  WAITING ON ITSELF: %s each wait on another issue in this set, so nothing closing outside it makes any of them available\n", numbers(set))
	}

	fmt.Fprintf(out, "%d issue(s) read, %d naming no issue, %d no longer blocked, %d unresolved, %d set(s) waiting on themselves.\n",
		len(blocked), nameless, cleared, unresolved, len(sets))

	var wrong []string
	if nameless > 0 {
		wrong = append(wrong, fmt.Sprintf("%d name no issue by number", nameless))
	}
	if cleared > 0 {
		wrong = append(wrong, fmt.Sprintf("%d are no longer blocked and still say they are", cleared))
	}
	if unresolved > 0 {
		wrong = append(wrong, fmt.Sprintf("%d name a number this board does not hold", unresolved))
	}
	if len(wrong) > 0 {
		return fmt.Errorf("blockers: %s", strings.Join(wrong, ", "))
	}
	return nil
}

// reciprocal returns every set of the issues given in which each member waits
// on another member of the same set, each set sorted and the sets ordered by
// their lowest member.
//
// Only an issue carrying the label is a member. An issue that does not carry
// it is claiming to wait for nothing, so a reference into it is not a wait and
// no path runs through it, however many blocked issues name it. That is what
// keeps a popular open issue out of a set it is merely mentioned by.
//
// The edges are the references the two questions above already resolve, so
// this reading inherits their bound: a body naming a number in passing is read
// as a wait. What makes the inherited over-reading survivable here is that a
// set needs the reference to run in both directions, so both bodies have to be
// describing something other than a wait, in opposite directions, at once. A
// one-sided mention produces no set.
//
// The walk is a reachability closure per member rather than a linear
// component algorithm. The population is the issues one board has under one
// label, which is tens rather than thousands, and the cost of the closure is
// paid once per run against a reading that took a paged fetch of the whole
// tracker to produce.
func reciprocal(blocked []Issue) [][]int {
	member := map[int]bool{}
	for _, i := range blocked {
		member[i.Number] = true
	}

	waits := map[int][]int{}
	for _, i := range blocked {
		for _, n := range References(i.Body) {
			if n == i.Number || !member[n] {
				continue
			}
			waits[i.Number] = append(waits[i.Number], n)
		}
	}

	// reach[n] is every member a wait from n arrives at, following waits as
	// far as they go. n is in its own set when it arrives back at itself,
	// which is a path of at least one edge and never the empty one.
	reach := map[int]map[int]bool{}
	for _, i := range blocked {
		seen := map[int]bool{}
		stack := append([]int(nil), waits[i.Number]...)
		for len(stack) > 0 {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if seen[n] {
				continue
			}
			seen[n] = true
			stack = append(stack, waits[n]...)
		}
		reach[i.Number] = seen
	}

	taken := map[int]bool{}
	var sets [][]int
	for _, i := range blocked {
		n := i.Number
		if taken[n] || !reach[n][n] {
			continue
		}
		set := []int{n}
		taken[n] = true
		for _, j := range blocked {
			m := j.Number
			if taken[m] || !reach[n][m] || !reach[m][n] {
				continue
			}
			set = append(set, m)
			taken[m] = true
		}
		sort.Ints(set)
		sets = append(sets, set)
	}
	return sets
}

// numbers writes a set the way the rest of this run writes a reference, so a
// reader can take a number out of either line and put it into the same query.
func numbers(set []int) string {
	out := make([]string, 0, len(set))
	for _, n := range set {
		out = append(out, "#"+strconv.Itoa(n))
	}
	return strings.Join(out, ", ")
}

// describe names a reference the way a reader has to see it, because a body
// naming a merged pull request and a body naming a closed issue are the same
// four characters and are not the same statement.
func describe(n int, ref Issue) string {
	kind := "issue"
	if ref.PullRequest {
		kind = "pull request"
	}
	return fmt.Sprintf("#%d (%s %s)", n, ref.State, kind)
}

// Tracker is the reader the scheduled run uses. It asks for issues and pull
// requests in both states, because a reference this run has to resolve is most
// often to something that has closed.
func Tracker() (Board, error) {
	var board Board

	if err := pages("/labels", func(body []byte) (int, error) {
		var page []struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return 0, err
		}
		for _, l := range page {
			board.Labels = append(board.Labels, l.Name)
		}
		return len(page), nil
	}); err != nil {
		return Board{}, err
	}

	if err := pages("/issues?state=all", func(body []byte) (int, error) {
		var page []struct {
			Number      int    `json:"number"`
			Title       string `json:"title"`
			State       string `json:"state"`
			Body        string `json:"body"`
			PullRequest *struct {
				URL string `json:"url"`
			} `json:"pull_request"`
			Labels []struct {
				Name string `json:"name"`
			} `json:"labels"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return 0, err
		}
		for _, row := range page {
			i := Issue{
				Number:      row.Number,
				Title:       row.Title,
				State:       row.State,
				Body:        row.Body,
				PullRequest: row.PullRequest != nil,
			}
			for _, l := range row.Labels {
				i.Labels = append(i.Labels, l.Name)
			}
			board.Issues = append(board.Issues, i)
		}
		return len(page), nil
	}); err != nil {
		return Board{}, err
	}

	if len(board.Issues) == 0 {
		return Board{}, fmt.Errorf("%s/%s answered with no issue at all, which is not a board this check can read", Owner, Repo)
	}
	return board, nil
}

// pages walks the paged listing at path and stops when a page comes back
// short. The cap is what keeps a tracker answering forever from turning a
// scheduled run into an unbounded one, and reaching it is a failure rather
// than a result, because what it would return is a prefix.
func pages(path string, take func([]byte) (int, error)) error {
	const perPage = 100
	client := &http.Client{Timeout: 30 * time.Second}
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	for page := 1; page <= 50; page++ {
		address := fmt.Sprintf("https://api.github.com/repos/%s/%s%s%sper_page=%d&page=%d",
			Owner, Repo, path, separator, perPage, page)
		req, err := http.NewRequest(http.MethodGet, address, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		// The listing is public, so this run works without a credential and
		// spends the anonymous rate limit when it has none. Where one is in
		// the environment it is used, because the limit that buys is what
		// makes the run survive a board of this size.
		if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("%s answered %s", address, resp.Status)
		}
		n, err := take(body)
		if err != nil {
			return fmt.Errorf("reading %s: %w", address, err)
		}
		if n < perPage {
			return nil
		}
	}
	return fmt.Errorf("%s%s did not stop paging, so this run read a prefix rather than the whole listing", Repo, path)
}
