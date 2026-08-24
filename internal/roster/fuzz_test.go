// SPDX-License-Identifier: AGPL-3.0-or-later

// The fuzz target over the door the site's content comes through.
//
// The suite beside this file covers the malformed rosters somebody thought of.
// This covers the ones nobody did, and it asks a different question: not whether
// a named mistake is refused, but whether anything Parse accepts is a roster the
// schema allows. A parser that refuses the fixtures and accepts a row with no
// identifier in it would pass every case in that suite.
//
// The property is written out of docs/roster-schema.md rather than out of
// roster.go, because a property derived from the code under test is satisfied by
// definition. Where the two disagree, the document is the authority and the code
// is what moves.
//
// Whether a repository exists is answered yes for every row, always. That
// question is the one thing the file cannot answer about itself, a fuzzer has no
// way to explore it, and a target that reached a real host would be a target
// that needs the network. What it costs is that the refusal for a repository
// that is not there is unreached here; the suite beside this file is where that
// case is.
package roster

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// alwaysThere is the answer described above, and it is the same answer every
// time so that a failing input replays to the same verdict.
func alwaysThere(string) (bool, error) { return true, nil }

// FuzzParse refuses a body that Parse accepted and the schema does not allow,
// and a refusal that arrives as something other than a Refusal carrying at least
// one reason.
func FuzzParse(f *testing.F) {
	// The seed corpus is the awkward shapes that already matter, and it is
	// committed here rather than under testdata/ so that each entry is read
	// beside the reason it exists. `go test` replays every one of them
	// without fuzzing, so the gate's test leg carries the corpus on every
	// run and an input that crashed once cannot come back quietly.
	seeds := []string{
		// A row the schema allows, so the target explores from something
		// that gets past the first refusal rather than from noise.
		`[{"id":"watchlist","repository":"Flowfin/jellyfin-plugin-watchlist","summary":"A private per-user watchlist","state":"build-up"}]`,
		// Brackets and quotes in the sentence.
		`[{"id":"a","repository":"o/jellyfin-plugin-a","summary":"Uses <b> and \"quotes\" and 'ticks' & an ampersand","state":"shell"}]`,
		// An empty field, per field.
		`[{"id":"","repository":"o/jellyfin-plugin-","summary":"x","state":"shell"}]`,
		`[{"id":"a","repository":"","summary":"x","state":"shell"}]`,
		`[{"id":"a","repository":"o/jellyfin-plugin-a","summary":"","state":"shell"}]`,
		`[{"id":"a","repository":"o/jellyfin-plugin-a","summary":"x","state":""}]`,
		// A repository name with path separators in it.
		`[{"id":"a","repository":"o/sub/jellyfin-plugin-a","summary":"x","state":"shell"}]`,
		`[{"id":"a","repository":"jellyfin-plugin-a","summary":"x","state":"shell"}]`,
		// Characters outside ASCII in the sentence and in the identifier.
		`[{"id":"a","repository":"o/jellyfin-plugin-a","summary":"Grüße 日本語","state":"shell"}]`,
		`[{"id":"ä","repository":"o/jellyfin-plugin-ä","summary":"x","state":"shell"}]`,
		// Two rows under one identifier.
		`[{"id":"a","repository":"o/jellyfin-plugin-a","summary":"x","state":"shell"},{"id":"a","repository":"o/jellyfin-plugin-a","summary":"y","state":"shell"}]`,
		// A field the roster does not have, and a field missing.
		`[{"id":"a","repository":"o/jellyfin-plugin-a","summary":"x","state":"shell","ships":"yes"}]`,
		`[{"id":"a","repository":"o/jellyfin-plugin-a","summary":"x"}]`,
		// Shapes that are not the array of objects this file has to be.
		`[]`,
		`{}`,
		`null`,
		`[null]`,
		`["a"]`,
		`[{"id":1,"repository":"o/jellyfin-plugin-a","summary":"x","state":"shell"}]`,
		`not json at all`,
		``,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	// A very long field, kept out of the list above so the list stays
	// readable.
	f.Add([]byte(`[{"id":"a","repository":"o/jellyfin-plugin-a","summary":"` +
		strings.Repeat("long ", 4096) + `","state":"shell"}]`))

	f.Fuzz(func(t *testing.T, body []byte) {
		entries, err := Parse(body, alwaysThere)
		if err != nil {
			var refusal *Refusal
			if !errors.As(err, &refusal) {
				t.Fatalf("refused with %T, and every refusal from this package is a *Refusal a caller can read the reasons out of: %v", err, err)
			}
			if len(refusal.Reasons) == 0 {
				t.Fatalf("refused with no reason on it, and a refusal naming nothing leaves the next person to bisect the file by hand")
			}
			if entries != nil {
				t.Fatalf("refused and returned %d row(s) as well, and a caller taking the rows of a refused file is the hole a door that fails closed may not have", len(entries))
			}
			return
		}

		// Accepted. Everything below is what docs/roster-schema.md says is
		// true of a file that gets this far.
		var rows []map[string]json.RawMessage
		if err := json.Unmarshal(body, &rows); err != nil {
			t.Fatalf("accepted a body that is not the array of rows the schema describes: %v", err)
		}
		if len(rows) != len(entries) {
			t.Fatalf("accepted %d row(s) and returned %d, so a row went missing between the two reads", len(rows), len(entries))
		}
		if len(entries) == 0 {
			t.Fatalf("accepted a file holding no row, and a site with no plugin in it is not what an empty roster was meant to say")
		}

		seen := map[string]int{}
		for i, e := range entries {
			for name := range rows[i] {
				if !fields[name] {
					t.Errorf("row %d was accepted carrying the field %q, which is not a field of this roster", i+1, name)
				}
			}
			if len(rows[i]) != len(fields) {
				t.Errorf("row %d was accepted carrying %d field(s), and every one of the %d is required", i+1, len(rows[i]), len(fields))
			}
			if e.ID == "" {
				t.Errorf("row %d was accepted with no identifier, and the identifier is what a page address and the per-plugin prose are keyed by", i+1)
			}
			if e.Summary == "" {
				t.Errorf("row %d was accepted with no sentence, and a plugin the site cannot say anything about is a page nobody can read", i+1)
			}
			if e.State != BuildUp && e.State != Shell {
				t.Errorf("row %d was accepted declaring the state %q, and a row may only say %q or %q", i+1, e.State, BuildUp, Shell)
			}
			owner, name, ok := strings.Cut(e.Repository, "/")
			switch {
			case !ok || owner == "":
				t.Errorf("row %d was accepted declaring the repository %q, and the schema writes one as owner/name", i+1, e.Repository)
			case strings.Contains(name, "/"):
				t.Errorf("row %d was accepted declaring the repository %q, which carries a path rather than the name after an owner", i+1, e.Repository)
			case name != repositoryPrefix+e.ID:
				t.Errorf("row %d was accepted declaring the repository %q against the identifier %q, and the name after the owner has to be %s", i+1, e.Repository, e.ID, repositoryPrefix+e.ID)
			}
			if first, ok := seen[e.ID]; ok {
				t.Errorf("row %d was accepted repeating the identifier declared on row %d", i+1, first)
			} else {
				seen[e.ID] = i + 1
			}
		}
	})
}
