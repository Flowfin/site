// SPDX-License-Identifier: AGPL-3.0-or-later

// Package freshness says when the pinned roster in this tree has fallen behind
// the published one.
//
// The build reads a copy so that it is reproducible and works with no network,
// which is the same reason the release data is recorded rather than fetched. A
// copy nobody checks is a copy that silently goes stale, so this compares it
// against what is published and reds on a difference. It writes nothing: the
// difference is the evidence for the change somebody makes, and a run that
// quietly resolved it would destroy that.
//
// It is a package of its own rather than a function in internal/roster, and the
// reason is the dependency graph rather than tidiness. The parser is on the path
// every build takes, this reaches the network, and a build that linked a client
// it never calls is a build whose offline property rests on nobody calling it.
//
// It fails closed in both directions a comparison can be absent. A fetch that
// could not be made is reported as unresolved and reds, because a copy that was
// not compared is not a current one. And nothing published at the address is
// reported as OFF and reds too, rather than passing: a freshness check that
// passed having compared nothing is the exact failure it exists to prevent.
package freshness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Flowfin/site/internal/roster"
	"github.com/Flowfin/site/internal/site"
)

// Published is where the roster is expected to be published.
//
// The address is derived rather than decided here, and what it is derived from
// is where the two files already published for machines sit: the catalogue
// manifest and the design token file are both in the served directory of the
// repository that holds machine-readable data, and
// decisions/0001-where-the-plugin-list-comes-from.md puts the roster beside the
// manifest. The file name is settled nowhere, which is why the comparison below
// reports OFF rather than green when nothing answers here: a name guessed wrong
// costs a corrected constant and can never cost a false pass.
const Published = "https://raw.githubusercontent.com/Flowfin/hub/HEAD/docs/roster.json"

// ErrNotPublished is what a fetcher answers with when there is no file at the
// address. It is a state of its own rather than an error, because a file that
// has not been published yet and a file that could not be read are different
// things to report and only one of them will end on its own.
var ErrNotPublished = errors.New("no file is published at that address")

// Fetcher reads the published file. It is a parameter so that the suite over
// this package reaches no network.
type Fetcher func() ([]byte, error)

// Difference is one row the two files disagree about. It carries the identifier
// rather than a position, because the identifier is the thing the two sides
// share and a row's position is not a fact about the roster.
type Difference struct {
	ID string
	// Says is what differs, in words, so a run names the repair rather than
	// leaving somebody to diff two files to find it.
	Says string
}

// Rows reads a roster leniently, applying none of the rules the parser applies.
//
// The comparison is about two files agreeing, not about either being valid: a
// published file this tree would refuse is still a published file the copy has
// fallen behind, and reporting it as unreadable would hide the difference behind
// a second problem. What refuses a roster the build may not use is the parser,
// on the build's own path.
func Rows(body []byte) ([]roster.Entry, error) {
	var rows []roster.Entry
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("the bytes are not the array of rows a roster is: %w", err)
	}
	return rows, nil
}

// Compare returns every row the two files disagree about, in identifier order so
// that two runs over one pair report in the same sequence.
func Compare(pinned, published []roster.Entry) []Difference {
	here := index(pinned)
	there := index(published)

	names := map[string]bool{}
	for id := range here {
		names[id] = true
	}
	for id := range there {
		names[id] = true
	}
	var sorted []string
	for id := range names {
		sorted = append(sorted, id)
	}
	sort.Strings(sorted)

	var out []Difference
	for _, id := range sorted {
		a, inHere := here[id]
		b, inThere := there[id]
		switch {
		case !inHere:
			out = append(out, Difference{id, "the published file carries this row and this tree does not"})
		case !inThere:
			out = append(out, Difference{id, "this tree carries this row and the published file does not"})
		case a != b:
			out = append(out, Difference{id, saysWhatDiffers(a, b)})
		}
	}
	return out
}

// index keys the rows by identifier. A row with no identifier is keyed by the
// empty string rather than dropped, so a file carrying one is reported as a
// difference instead of quietly matching nothing.
func index(rows []roster.Entry) map[string]roster.Entry {
	out := make(map[string]roster.Entry, len(rows))
	for _, r := range rows {
		out[r.ID] = r
	}
	return out
}

// saysWhatDiffers names each field the two rows disagree about, with both
// values. A message saying only that a row differs sends the next reader back
// to the two files to find out which field it was.
func saysWhatDiffers(a, b roster.Entry) string {
	var said []string
	if a.Repository != b.Repository {
		said = append(said, fmt.Sprintf("this tree says the repository is %q and the published file says %q", a.Repository, b.Repository))
	}
	if a.Summary != b.Summary {
		said = append(said, fmt.Sprintf("this tree says the sentence is %q and the published file says %q", a.Summary, b.Summary))
	}
	if a.State != b.State {
		said = append(said, fmt.Sprintf("this tree says the state is %q and the published file says %q", a.State, b.State))
	}
	return strings.Join(said, ", ")
}

// Run compares the pinned copy against the published file and reports what it
// found.
func Run(root string, fetch Fetcher, out io.Writer) error {
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(site.RosterFile)))
	if err != nil {
		return fmt.Errorf("roster: reading the pinned copy at %s: %w", site.RosterFile, err)
	}
	pinned, err := Rows(body)
	if err != nil {
		return fmt.Errorf("roster: the pinned copy at %s could not be read: %w", site.RosterFile, err)
	}
	fmt.Fprintf(out, "roster: %d row(s) pinned in %s\n", len(pinned), site.RosterFile)

	fetched, err := fetch()
	switch {
	case errors.Is(err, ErrNotPublished):
		fmt.Fprintf(out, "  OFF, nothing answered at %s, so this run compared nothing\n", Published)
		fmt.Fprintf(out, "  what ends this state is that file being published; until then the copy in this tree is held to nothing\n")
		return fmt.Errorf("roster: no file is published at %s, so this run compared nothing and is not a run that found the copy current", Published)
	case err != nil:
		fmt.Fprintf(out, "  UNRESOLVED, %s could not be read: %v\n", Published, err)
		return fmt.Errorf("roster: the published file could not be read, and a copy that was not compared is not a current one")
	}

	published, err := Rows(fetched)
	if err != nil {
		fmt.Fprintf(out, "  UNRESOLVED, what %s answered %v\n", Published, err)
		return fmt.Errorf("roster: the published file could not be read, and a copy that was not compared is not a current one")
	}

	differing := Compare(pinned, published)
	for _, d := range differing {
		fmt.Fprintf(out, "  %s: BEHIND, %s\n", d.ID, d.Says)
	}

	fmt.Fprintf(out, "%d row(s) pinned, %d published, %d differing.\n", len(pinned), len(published), len(differing))
	if len(differing) > 0 {
		return fmt.Errorf("roster: %d row(s) differ from what is published, and %s is the authority for them", len(differing), Published)
	}
	return nil
}

// Publisher is the fetcher the scheduled run uses. A plain GET over a file
// somebody else publishes, which is why this needs no client library and no
// dependency.
func Publisher() ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(Published)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotPublished
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %s", Published, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, fmt.Errorf("%s answered with an empty body", Published)
	}
	return body, nil
}
