// SPDX-License-Identifier: AGPL-3.0-or-later

package site

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/Flowfin/site/internal/roster"
)

// The two files the plugin rows are read from.
//
// RosterFile is the roster this repository builds against. Entry 6 of #7 is
// answered `start here against committed fixtures and vendor the published file
// when it appears`, so this is a copy this repository authors today and a
// vendored one later. Nothing about the build changes on that day: what moves is
// where the bytes come from, which is #24.
//
// RepositoriesFile is the recorded answer to the one question the roster cannot
// answer about itself. The parser asks whether each row's repository is there
// and refuses a read that skipped the question, and asking a host is a request
// off this machine, which a build may not make: it would produce different bytes
// on different days and fail with no network. So the answer is taken once,
// written down with the command that took it, and read from the tree.
const (
	RosterFile       = "data/roster.json"
	RepositoriesFile = "data/repositories.json"
)

// plugin is one row as a page renders it. It carries the words rather than the
// row, so the template holds no knowledge of what a state word means and no
// count of anything.
type plugin struct {
	ID string
	// Href is where this plugin's own page is on this site, which is what
	// the row on the landing page sends a reader to. Repository is where the
	// code is, and it is on the plugin's page rather than in the table: a
	// table cell offering both is a cell where a reader has to guess which
	// of two links is the one they want.
	Href       string
	Repository string
	Summary    string
	State      string
	// Means is what the state word says in a sentence. It is composed where
	// the rows are read, so no template and no per-plugin file carries it,
	// and a state word gaining a meaning changes one place.
	Means string
	// Produced is the path the build writes this plugin's page to. It is not
	// rendered; it is what the writer and the address derivation use.
	Produced string
}

// PluginsDir is the container the plugin pages go in, which is the address
// decisions/0008-the-url-shape.md gives them. The twelve pages need one or they
// collide with everything else at the root.
const PluginsDir = "plugins"

// repositoryRecord is what RepositoriesFile holds. The moment and the command
// are part of it rather than beside it: a recorded answer with nothing saying
// when it was taken is an answer nobody can judge the age of.
type repositoryRecord struct {
	Taken        string   `json:"taken"`
	Command      string   `json:"command"`
	Repositories []string `json:"repositories"`
}

// readPlugins reads the roster and returns the rows a page renders, or every
// reason it would not.
//
// It fails closed at each step, which is the position the parser already takes
// and the reason this build reads a roster at all. An absent roster is not a
// site with no plugins in it, an unreadable record is not a set of repositories
// that are all there, and a row naming a repository the record does not carry is
// refused rather than rendered with a link to nothing.
func readPlugins(root string, log io.Writer) ([]plugin, string, error) {
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(RosterFile)))
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(log, "no %s in the tree, so the page carries no plugin row\n", RosterFile)
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("reading %s: %w", RosterFile, err)
	}

	known, taken, err := readRepositories(root)
	if err != nil {
		return nil, "", err
	}

	entries, err := roster.Parse(body, func(repository string) (bool, error) {
		return known[repository], nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("reading %s: %w", RosterFile, err)
	}

	rows := make([]plugin, 0, len(entries))
	for _, e := range entries {
		said, means, err := stateInWords(e.State)
		if err != nil {
			return nil, "", fmt.Errorf("%s, row %s: %w", RosterFile, e.ID, err)
		}
		// The identifier reduced to one segment before it reaches a path.
		// The parser already refuses an identifier that is not one, so this
		// is the second of two readings rather than the only one, and it is
		// here because the line that joins a value out of a data file into a
		// path is where a reader decides whether a row can choose where the
		// build writes.
		segment := filepath.Base(e.ID)
		rows = append(rows, plugin{
			ID:         e.ID,
			Href:       "/" + path.Join(PluginsDir, segment) + "/",
			Repository: e.Repository,
			Summary:    e.Summary,
			State:      said,
			Means:      means,
			Produced:   path.Join(PluginsDir, segment, indexDocument),
		})
	}

	fmt.Fprintf(log, "read %s (%d row(s))\n", RosterFile, len(rows))
	return rows, taken, nil
}

// readRepositories reads the recorded answer and returns the set it carries with
// the day it was taken.
//
// An empty set is refused rather than returned. A record carrying no repository
// would answer `not there` for every row, which reds the build for twelve
// reasons that are all one reason, and the message a reader gets has to be that
// one rather than the twelve.
func readRepositories(root string) (map[string]bool, string, error) {
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(RepositoriesFile)))
	if err != nil {
		return nil, "", fmt.Errorf("reading %s, which is what answers whether a roster row's repository is there: %w", RepositoriesFile, err)
	}
	var rec repositoryRecord
	if err := json.Unmarshal(body, &rec); err != nil {
		return nil, "", fmt.Errorf("reading %s: %w", RepositoriesFile, err)
	}
	if len(rec.Repositories) == 0 {
		return nil, "", fmt.Errorf("%s carries no repository, and a record that answers `not there` for every row is a record nobody took rather than a set of repositories that are gone", RepositoriesFile)
	}
	if rec.Taken == "" {
		return nil, "", fmt.Errorf("%s says nothing about when it was taken, and a recorded answer nobody can judge the age of is one a reader has to trust", RepositoriesFile)
	}
	known := make(map[string]bool, len(rec.Repositories))
	for _, r := range rec.Repositories {
		known[r] = true
	}
	return known, rec.Taken, nil
}

// stateInWords says what a declared state word means on the page.
//
// It refuses a word it does not know rather than printing it, so a state added
// to the roster vocabulary reds the build here instead of reaching a reader as a
// bare identifier. The parser refuses one too; this is the second half, and the
// two are different failures: the parser judges the file, and this judges
// whether the page can say what the file declared.
func stateInWords(state string) (string, string, error) {
	switch state {
	case roster.BuildUp:
		return "In build-up", "The plugin is being built. Nothing is published for it, so there is nothing to install and installing what is in the repository would add nothing a user can see.", nil
	case roster.Shell:
		return "Shell only", "The repository holds the shape of the plugin and none of what it is for. Nothing is published for it, so there is nothing to install and installing what is in the repository would add nothing a user can see.", nil
	default:
		return "", "", fmt.Errorf("declares the state %q, and this page has no words for it", state)
	}
}

// saidAboutTheRows is the sentence above the table. It is composed here rather
// than written into the template, because it states how many rows there are and
// where they came from, and a number in the template is a number the template
// would have to be edited to keep true.
func saidAboutTheRows(rows []plugin, taken string) string {
	return fmt.Sprintf(
		"%d rows, read from the roster this repository carries. Each repository was confirmed to be there on %s, and what a state word declares is what is true before anything is published.",
		len(rows), taken)
}
