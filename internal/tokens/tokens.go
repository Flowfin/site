// Package tokens holds the pinned copy of the design token file this
// repository reads, and the run that says when the copy has fallen behind the
// published one.
//
// Nothing here defines a value. The token file is published with the
// machine-readable data rather than here, which is
// decisions/0007-where-the-design-tokens-live.md, and what this repository
// keeps is a copy: the build reads it and needs no network, and the copy is
// the same bytes twice however the day is going for somebody else's host.
//
// The comparison is the same shape as the one over the pins, for the same
// reason, and it reads differently in one way worth stating. A pin that has
// fallen behind shows up as a version somebody can read. A token that has
// fallen behind shows up as a colour or a spacing step that no longer matches
// what the clients are held to, and nobody notices, because a page rendering a
// wrong value renders it perfectly.
//
// So the run compares values and names them. It reads both files as a set of
// paths to values rather than as bytes, and a difference is reported as the
// path that differs with both sides beside it. Two files whose values agree and
// whose formatting does not compare equal, which is deliberate: what this
// repository vendors is the values, and a run reddening because somebody
// upstream re-indented a file would be a run people learn to ignore.
//
// Four states stay distinguishable, because collapsing them is what makes a
// stale copy invisible. A copy whose values match the published file is
// current. A copy whose values differ is behind, and the run names every value
// and prints both sides. A published file that could not be read is
// unresolved, which is a failure rather than a pass. And a published file that
// is not there at all is not a comparison at all: the run says it compared
// nothing and what would end that, and it does not pass, because a freshness
// check that passed having fetched nothing is the exact failure it exists to
// prevent.
package tokens

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
	"strconv"
	"strings"
	"time"
)

// File is the pinned copy, under the directory that holds what this repository
// vendors rather than authors.
const File = "data/design-tokens.json"

// Published is where the copy comes from. It is the repository that holds the
// machine-readable data rather than the address the site is served at: on the
// day this repository becomes that origin, a comparison against the served
// address would be a file compared with itself.
const Published = "https://raw.githubusercontent.com/Flowfin/hub/HEAD/docs/design-tokens.json"

// ErrNotPublished is what a fetcher answers with when there is no file at the
// published location. It is separate from every other failure because the two
// mean opposite things about what to do next: an unreadable file is a fetch to
// repeat, and an absent one is a file somebody has to land.
var ErrNotPublished = errors.New("no file is published at that address")

// Values is a token file read as one value per path. The path is the name a
// difference is reported under, so a reader is told which value moved rather
// than that the file did.
type Values map[string]string

// Difference is one value that does not agree. Whether a side carries the
// value is its own field rather than being read off an empty string: a value
// somebody set to nothing and a value that is not there are different states,
// and a token removed upstream is exactly the kind of difference that would go
// unreported if they were collapsed.
type Difference struct {
	Name         string
	Pinned       string
	HasPinned    bool
	Published    string
	HasPublished bool
}

// A Fetcher answers with the published file. It is a parameter so the suite can
// drive every answer this run has to tell apart, including the ones a network
// cannot be asked to produce on demand.
type Fetcher func() ([]byte, error)

// Load reads the pinned copy.
func Load(root string) (Values, error) {
	name := filepath.Join(root, filepath.FromSlash(File))
	body, err := os.ReadFile(filepath.Clean(name))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", File, err)
	}
	values, err := Read(body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", File, err)
	}
	return values, nil
}

// Read parses a token file and flattens it to one value per path. Numbers keep
// the digits the file was written with rather than being taken through a float,
// because a limit of 1200 reported back as 1.2e+03 is a difference a reader has
// to decode before they can see there is none.
func Read(body []byte) (Values, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var top any
	if err := dec.Decode(&top); err != nil {
		return nil, fmt.Errorf("is not JSON: %w", err)
	}
	if _, ok := top.(map[string]any); !ok {
		return nil, errors.New("is not the object of groups a token file has to be")
	}
	values := Values{}
	flatten("", top, values)
	if len(values) == 0 {
		return nil, errors.New("carries no value, and a token file with nothing in it is not a copy of one")
	}
	return values, nil
}

// budgetPrefix is where the numbers a client is held to sit in the file. It is
// a prefix rather than a list of the five, because the set is the file's to
// decide and a list here would be a second declaration of it.
const budgetPrefix = "budget.numbers."

// Number is one of the numbers a client has to meet, read out of the copy. It
// carries the limit as the digits the file was written with rather than as an
// integer, for the reason Read keeps them: a limit reported back in another
// spelling is a difference a reader has to decode before they can see there is
// none.
type Number struct {
	Name       string
	Limit      string
	Unit       string
	Comparison string
	// What and Why are the file's own sentences beside the number. They are
	// prose about the value rather than the value, which is why nothing
	// compares them and why a page prints them beside the limit.
	What string
	Why  string
}

// Stated is how the number is written where somebody reads it. The file carries
// the limit and the comparison apart, and printing the number alone would state
// a ceiling and a required value in the same words, which are opposite claims.
//
// This is the one place that mapping is made. The page states a number by asking
// here, and the row that refuses a second copy of one asks here too, so a
// spelling the page uses and a spelling the row looks for cannot part company.
func (n Number) Stated() string {
	switch n.Comparison {
	case "below":
		return "under " + n.Limit + " " + n.Unit
	case "equal":
		return "exactly " + n.Limit + " " + n.Unit
	default:
		return n.Comparison + " " + n.Limit + " " + n.Unit
	}
}

// Bare is the number with its unit and nothing in front of it. It is the other
// spelling the same value arrives in, and it is the one somebody types when they
// are quoting a limit inside a sentence rather than stating it in a table.
func (n Number) Bare() string {
	return n.Limit + " " + n.Unit
}

// Numbers reads the client budget out of a file that has been flattened. It
// returns one reason per number it could not make sense of rather than the
// first, because a file with three broken numbers is three repairs.
//
// An empty result carries no reason. Whether a file with no such number is a
// failure depends on what the caller was going to do with them, and the two
// callers answer it differently: a page with no budget table and a rule with
// nothing to compare against are refused in different words.
func Numbers(values Values) ([]Number, []string) {
	var at []string
	for p := range values {
		if strings.HasPrefix(p, budgetPrefix) && strings.HasSuffix(p, ".limit") {
			at = append(at, strings.TrimSuffix(p, ".limit"))
		}
	}
	sort.Strings(at)

	var out []Number
	var reasons []string
	for _, a := range at {
		unit, ok := values[a+".unit"]
		if !ok {
			reasons = append(reasons, fmt.Sprintf("%s carries a limit and no unit, so the number states nothing", a))
			continue
		}
		comparison, ok := values[a+".comparison"]
		if !ok {
			reasons = append(reasons, fmt.Sprintf("%s carries a limit and no comparison, so whether it is a ceiling or a value is not stated", a))
			continue
		}
		out = append(out, Number{
			Name:       strings.TrimPrefix(a, budgetPrefix),
			Limit:      values[a+".limit"],
			Unit:       unit,
			Comparison: comparison,
			What:       values[a+".what"],
			Why:        values[a+".why"],
		})
	}
	return out, reasons
}

// flatten walks the document. An array is indexed rather than joined, because
// the order of a font stack is part of what it says and a reordering has to
// read as a difference.
func flatten(prefix string, node any, into Values) {
	switch v := node.(type) {
	case map[string]any:
		for key, child := range v {
			flatten(join(prefix, key), child, into)
		}
	case []any:
		for i, child := range v {
			flatten(prefix+"["+strconv.Itoa(i)+"]", child, into)
		}
	case json.Number:
		into[prefix] = v.String()
	case string:
		into[prefix] = v
	case bool:
		into[prefix] = strconv.FormatBool(v)
	case nil:
		into[prefix] = "null"
	}
}

func join(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// Compare returns every value the two files disagree about, in path order so
// that two runs over one pair of files report in the same sequence.
func Compare(pinned, published Values) []Difference {
	names := map[string]bool{}
	for name := range pinned {
		names[name] = true
	}
	for name := range published {
		names[name] = true
	}
	var sorted []string
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)

	var out []Difference
	for _, name := range sorted {
		here, inHere := pinned[name]
		there, inThere := published[name]
		if inHere == inThere && here == there {
			continue
		}
		out = append(out, Difference{
			Name: name, Pinned: here, HasPinned: inHere, Published: there, HasPublished: inThere,
		})
	}
	return out
}

// Run compares the pinned copy against the published file and reports what it
// found. It writes nothing: the difference is the evidence for a change
// somebody makes, and a run that resolved it quietly would destroy that.
func Run(root string, fetch Fetcher, out io.Writer) error {
	pinned, err := Load(root)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "design tokens: %d value(s) pinned in %s\n", len(pinned), File)

	body, err := fetch()
	switch {
	case errors.Is(err, ErrNotPublished):
		fmt.Fprintf(out, "  nothing answered at %s, so this run compared nothing\n", Published)
		fmt.Fprintf(out, "  what ends this state is that file being published; until then the copy in this tree is held to nothing\n")
		return fmt.Errorf("design tokens: no file is published at %s, so this run compared nothing and is not a run that found the copy current", Published)
	case err != nil:
		fmt.Fprintf(out, "  UNRESOLVED, %s could not be read: %v\n", Published, err)
		return fmt.Errorf("design tokens: the published file could not be read, and a copy that was not compared is not a current one")
	}

	published, err := Read(body)
	if err != nil {
		fmt.Fprintf(out, "  UNRESOLVED, what %s answered %v\n", Published, err)
		return fmt.Errorf("design tokens: the published file could not be read, and a copy that was not compared is not a current one")
	}

	differing := Compare(pinned, published)
	for _, d := range differing {
		switch {
		case !d.HasPinned:
			fmt.Fprintf(out, "  %s: BEHIND, this tree carries no such value, the published file says %q\n", d.Name, d.Published)
		case !d.HasPublished:
			fmt.Fprintf(out, "  %s: BEHIND, this tree says %q, the published file carries no such value\n", d.Name, d.Pinned)
		default:
			fmt.Fprintf(out, "  %s: BEHIND, this tree says %q, the published file says %q\n", d.Name, d.Pinned, d.Published)
		}
	}

	fmt.Fprintf(out, "%d value(s) pinned, %d published, %d differing.\n", len(pinned), len(published), len(differing))
	if len(differing) > 0 {
		return fmt.Errorf("design tokens: %d value(s) differ from what is published, and %s is the authority for them", len(differing), Published)
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
