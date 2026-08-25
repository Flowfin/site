// SPDX-License-Identifier: AGPL-3.0-or-later

// Package releases holds what the build knows about published releases, and it
// knows it from a file rather than from a request.
//
// Whether a plugin ships is a fact about published releases and not an opinion
// a roster row may hold, so the build has to read the release lists somehow.
// Reading them at build time would put a request off the machine inside a build
// that has to be reproducible and has to work with no network, so the answer is
// taken once by a verb somebody runs, written down with the command that took
// it, and committed. A build renders what was last recorded and says when it
// was recorded, rather than guessing or failing.
//
// Which releases count is decisions/0009-what-counts-as-shipping.md and is not
// restated here: this package reads the two counts that record's rule needs and
// applies it in one place.
//
// The record is also what answers whether a roster row's repository is there at
// all. That is the same question asked once: a repository that answered with its
// release list is a repository that exists, and a second file carrying the same
// set would be a second thing to keep current.
package releases

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// File is where the record lives. It sits with the other files the build reads
// rather than in a register of its own, because the build reads it on every run
// and a reader looking for what the pages are made of looks there.
const File = "data/releases.json"

// Repository is what was read about one repository. Two counts rather than a
// verdict: the rule that turns them into a word is a decision that can move, and
// a record carrying the word instead of the counts would have to be retaken the
// day it moves.
//
// Finished is releases that are published, not drafts and not prereleases, which
// is what decisions/0009 makes the signal. Prereleases is the rest of the
// published ones, and it is here because that record says a plugin whose only
// releases are prereleases has something to test and that this news belongs on
// the page rather than in the state.
type Repository struct {
	Finished    int `json:"finished"`
	Prereleases int `json:"prereleases"`
}

// Record is the file.
type Record struct {
	Taken        string                `json:"taken"`
	Command      string                `json:"command"`
	Repositories map[string]Repository `json:"repositories"`
}

// Ships says whether what was recorded about a repository means the plugin in
// it ships. It is the one place decisions/0009's rule is applied, so a build,
// a page and anything else that asks get the same answer.
func (r Repository) Ships() bool { return r.Finished > 0 }

// Known says whether the record carries a repository, which is the same
// question as whether the repository is there: the record is made by asking
// each one for its releases, and a repository that answered exists.
func (rec Record) Known(repository string) bool {
	_, ok := rec.Repositories[repository]
	return ok
}

// Load reads the record and fails closed on every shape a build must not carry
// on past.
//
// Each refusal is written apart rather than collapsed into one message about a
// bad file, because they are different repairs. A record that carries no
// repository would answer `not there` for every roster row, which reds a build
// for twelve reasons that are all one reason. A record saying nothing about when
// it was taken is an answer nobody can judge the age of, and the page states
// that moment, so a build that carried on would put a page in front of a reader
// with nothing on it saying how old the claim is.
func Load(root string) (Record, error) {
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(File)))
	if err != nil {
		return Record{}, fmt.Errorf("reading %s, which is what the shipping state and the roster's repositories are read from: %w", File, err)
	}
	return Read(body)
}

// Read is Load without the filesystem, so a case can hand it bytes.
func Read(body []byte) (Record, error) {
	var rec Record
	if err := json.Unmarshal(body, &rec); err != nil {
		return Record{}, fmt.Errorf("reading %s: %w", File, err)
	}
	if len(rec.Repositories) == 0 {
		return Record{}, fmt.Errorf("%s carries no repository, and a record that answers `not there` for every row is a record nobody took rather than a set of repositories that are gone", File)
	}
	if strings.TrimSpace(rec.Taken) == "" {
		return Record{}, fmt.Errorf("%s says nothing about when it was taken, and the page states that moment, so a build carrying on would put a claim in front of a reader with nothing saying how old it is", File)
	}
	for name, r := range rec.Repositories {
		if r.Finished < 0 || r.Prereleases < 0 {
			return Record{}, fmt.Errorf("%s records %s with a negative count, which is not something a release list can answer", File, name)
		}
	}
	return rec, nil
}

// Fetcher answers what one repository has published. It is a parameter so that
// the suite over this package reaches no network: the one implementation that
// does is below and is reached only by the verb.
type Fetcher func(repository string) (Repository, error)

// Refresh asks about each repository and returns the record to write. It takes
// the moment rather than reading a clock, so a caller decides what the record
// says it was taken at and a case can assert it.
//
// It fails on the first repository it could not read rather than writing a
// record with a hole in it. A record missing a repository is a record that says
// the repository is not there, and a refresh that produced one would turn a
// network failure into a claim about somebody's repository.
func Refresh(repositories []string, taken, command string, fetch Fetcher) (Record, error) {
	if len(repositories) == 0 {
		return Record{}, fmt.Errorf("nothing was named to ask about, and a record built from no question is one that answers `not there` for every row")
	}
	rec := Record{Taken: taken, Command: command, Repositories: map[string]Repository{}}
	names := append([]string(nil), repositories...)
	sort.Strings(names)
	for _, name := range names {
		r, err := fetch(name)
		if err != nil {
			return Record{}, fmt.Errorf("asking %s for its releases: %w", name, err)
		}
		rec.Repositories[name] = r
	}
	return rec, nil
}

// Marshal writes the record the way the file carries it: indented, sorted by
// the encoder because a JSON object's keys are written in order, and with the
// trailing newline a text file ends with. A record written any other way would
// produce a diff on every refresh that changed nothing.
func Marshal(rec Record) ([]byte, error) {
	body, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

// Run refreshes the record and writes it, reporting every repository it asked
// about and what moved.
//
// It writes rather than reporting a difference, which is the opposite of what
// the token verb does with its copy and is deliberate. The token file has an
// authority somewhere else and a difference is evidence about that file; this
// record has no authority anywhere, it is the answer itself, and a run that
// only reported it would leave somebody typing counts into a file by hand.
func Run(root string, repositories []string, taken, command string, fetch Fetcher, out io.Writer) error {
	before, err := Load(root)
	if err != nil {
		// A tree with no record yet, or one this verb is about to
		// replace, is not a reason to refuse a refresh. What was there is
		// reported as unreadable and the run says so rather than
		// pretending it compared something.
		fmt.Fprintf(out, "releases: nothing readable was there before this run (%v)\n", err)
		before = Record{Repositories: map[string]Repository{}}
	}

	rec, err := Refresh(repositories, taken, command, fetch)
	if err != nil {
		return fmt.Errorf("releases: %w", err)
	}

	body, err := Marshal(rec)
	if err != nil {
		return err
	}
	name := filepath.Join(root, filepath.FromSlash(File))
	if err := os.WriteFile(name, body, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", File, err)
	}

	moved := 0
	names := make([]string, 0, len(rec.Repositories))
	for n := range rec.Repositories {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		now := rec.Repositories[n]
		was, had := before.Repositories[n]
		switch {
		case !had:
			moved++
			fmt.Fprintf(out, "  %s: NEW, %d finished, %d prerelease(s)\n", n, now.Finished, now.Prereleases)
		case was != now:
			moved++
			fmt.Fprintf(out, "  %s: MOVED, was %d finished and %d prerelease(s), now %d and %d\n",
				n, was.Finished, was.Prereleases, now.Finished, now.Prereleases)
		default:
			fmt.Fprintf(out, "  %s: unchanged, %d finished, %d prerelease(s)\n", n, now.Finished, now.Prereleases)
		}
	}
	for n := range before.Repositories {
		if _, still := rec.Repositories[n]; !still {
			moved++
			fmt.Fprintf(out, "  %s: GONE, the roster no longer names it and this run did not ask about it\n", n)
		}
	}

	fmt.Fprintf(out, "releases: %d repository(s) recorded as taken on %s, %d entry(s) moved, written to %s\n",
		len(rec.Repositories), rec.Taken, moved, File)
	return nil
}
