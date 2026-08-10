// Package roster reads the file everything this site says about the plugins
// comes out of, and refuses one it does not understand.
//
// It is the only door that content comes through, so it fails closed. A row it
// cannot make sense of stops the read rather than rendering a page with a hole
// in it, and every refusal names the row it is about: a parser that says only
// that the file is invalid makes the next person bisect the data by hand.
//
// An unknown field is refused rather than ignored, on purpose. A field the build
// silently passes over is a field somebody will add believing it does something,
// and the day they notice is the day a page has been wrong for a month.
//
// What the file is shaped like is docs/roster-schema.md, and this package is
// that document decided rather than a second statement of it. Two things the
// schema is emphatic about are visible here: every field is a string, so nothing
// depends on a version-shaped value surviving a type guess, and there is no
// field saying whether a plugin ships, because that is a fact about published
// releases rather than an opinion a row may hold.
package roster

import (
	"encoding/json"
	"fmt"
	"strings"
)

// The states a row may declare. A row says what is true when nothing is
// published; the build raises it where the releases say otherwise, which is why
// there is no third word here for a plugin that ships.
const (
	BuildUp = "build-up"
	Shell   = "shell"
)

// repositoryPrefix is what a plugin repository is called before its identifier.
// The identifier and the repository name are one fact written twice in the file,
// and this is what holds the two to each other.
const repositoryPrefix = "jellyfin-plugin-"

// Entry is one row.
type Entry struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Summary    string `json:"summary"`
	State      string `json:"state"`
}

// Exists answers whether a repository is there, written as `owner/name`. It is
// a parameter because it is the one question the file cannot answer about
// itself, and because a suite that had to ask a real host would be a suite that
// needs the network.
type Exists func(repository string) (bool, error)

// Refusal is what a read of a file this package will not accept returns. It
// carries one reason per thing that was wrong rather than the first, because a
// file with three broken rows is three repairs and reporting one of them costs
// three runs.
type Refusal struct {
	Reasons []string
}

func (r *Refusal) Error() string {
	return fmt.Sprintf("the roster was refused, %d reason(s):\n  %s",
		len(r.Reasons), strings.Join(r.Reasons, "\n  "))
}

var fields = map[string]bool{"id": true, "repository": true, "summary": true, "state": true}

// Parse reads the file and returns its rows, or every reason it would not.
//
// exists is asked once per row and may not be nil. A read that skipped the
// question would report a roster it had not checked, which is the one shape a
// door that fails closed may not have.
func Parse(body []byte, exists Exists) ([]Entry, error) {
	if exists == nil {
		return nil, &Refusal{Reasons: []string{
			"nothing was supplied to ask whether a repository exists, and a read that skipped that question would report a roster it had not checked",
		}}
	}

	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, &Refusal{Reasons: []string{
			fmt.Sprintf("the file is not the array of rows this roster has to be: %v", err),
		}}
	}
	var entries []Entry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, &Refusal{Reasons: []string{
			fmt.Sprintf("a row does not carry the fields this roster has to carry: %v", err),
		}}
	}
	if len(entries) == 0 {
		return nil, &Refusal{Reasons: []string{
			"the file holds no row, and a site with no plugin in it is not what an empty roster was meant to say",
		}}
	}

	var reasons []string
	seen := map[string]int{}

	for i, row := range rows {
		e := entries[i]
		where := fmt.Sprintf("row %d", i+1)
		if e.ID != "" {
			where = fmt.Sprintf("row %d, %s", i+1, e.ID)
		}
		refuse := func(format string, args ...any) {
			reasons = append(reasons, where+": "+fmt.Sprintf(format, args...))
		}

		for name := range row {
			if !fields[name] {
				refuse("carries the field %q, which is not a field of this roster, and a field the build passed over is one somebody would add believing it did something", name)
			}
		}
		for _, name := range []string{"id", "repository", "summary", "state"} {
			if _, ok := row[name]; !ok {
				refuse("has no %s, and every field of a row is required", name)
			}
		}

		if e.Summary == "" {
			if _, ok := row["summary"]; ok {
				refuse("carries an empty sentence, and a plugin the site cannot say anything about is a page nobody can read")
			}
		}
		if e.ID != "" {
			if first, ok := seen[e.ID]; ok {
				refuse("repeats the identifier declared on row %d, and an identifier is what a page address and the prose in this repository are keyed by", first)
			} else {
				seen[e.ID] = i + 1
			}
		}
		if e.ID != "" && e.Repository != "" {
			if want := repositoryPrefix + e.ID; !strings.HasSuffix(e.Repository, "/"+want) {
				refuse("declares the repository %s, and an identifier of %s means the name after the owner has to be %s", e.Repository, e.ID, want)
			}
		}
		if _, ok := row["state"]; ok && e.State != BuildUp && e.State != Shell {
			refuse("declares the state %q, and a row may only say %q or %q", e.State, BuildUp, Shell)
		}
		if e.Repository != "" {
			there, err := exists(e.Repository)
			switch {
			case err != nil:
				refuse("whether the repository %s exists could not be answered, and an unanswered question is not a repository that is there: %v", e.Repository, err)
			case !there:
				refuse("names the repository %s, which is not there, so every link this row produces would resolve to nothing", e.Repository)
			}
		}
	}

	if len(reasons) > 0 {
		return nil, &Refusal{Reasons: reasons}
	}
	return entries, nil
}
