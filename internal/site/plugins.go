// SPDX-License-Identifier: AGPL-3.0-or-later

package site

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/Flowfin/site/internal/releases"
	"github.com/Flowfin/site/internal/roster"
)

// RosterFile is the roster this repository builds against. Entry 6 of #7 is
// answered `start here against committed fixtures and vendor the published file
// when it appears`, so this is a copy this repository authors today and a
// vendored one later. Nothing about the build changes on that day: what moves is
// where the bytes come from, which is #24.
//
// What the build reads beside it is internal/releases, which answers both what
// each repository has published and whether it is there at all.
const RosterFile = "data/roster.json"

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

	recorded, err := releases.Load(root)
	if err != nil {
		return nil, "", err
	}

	entries, err := roster.Parse(body, func(repository string) (bool, error) {
		return recorded.Known(repository), nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("reading %s: %w", RosterFile, err)
	}

	rows := make([]plugin, 0, len(entries))
	for _, e := range entries {
		said, means, err := stateInWords(e.State, recorded.Repositories[e.Repository])
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
	return rows, recorded.Taken, nil
}

// stateInWords says what state a plugin is in, in the word the table carries
// and in the sentence its own page carries.
//
// The declared word is the floor and the recorded releases raise it, which is
// decisions/0001's rule: whether something ships is a fact about published
// releases and not an opinion a row may hold, and decisions/0009 fixes which
// releases count. So a row declaring `shell` or `build-up` for a repository
// that has published a finished release is shown as shipping, and the
// disagreement between the two is a thing to repair where it is published
// rather than a thing this build guesses at.
//
// A declared word it does not know is refused rather than printed, so a state
// added to the roster vocabulary reds the build here instead of reaching a
// reader as a bare identifier. The parser refuses one too; this is the second
// half, and the two are different failures: the parser judges the file, and
// this judges whether the page can say what the file declared.
func stateInWords(state string, published releases.Repository) (string, string, error) {
	switch state {
	case roster.BuildUp, roster.Shell:
	default:
		return "", "", fmt.Errorf("declares the state %q, and this page has no words for it", state)
	}

	if published.Ships() {
		return "Ships", "The plugin has a finished release published, so there is something to install." +
			somethingToTest(published), nil
	}

	said := "The plugin is being built. Nothing finished is published for it, so there is nothing to install and installing what is in the repository would add nothing a user can see."
	if state == roster.Shell {
		said = "The repository holds the shape of the plugin and none of what it is for. Nothing finished is published for it, so there is nothing to install and installing what is in the repository would add nothing a user can see."
	}
	return stateWord(state), said + somethingToTest(published), nil
}

// stateWord is the declared state in the words the table carries.
func stateWord(state string) string {
	if state == roster.Shell {
		return "Shell only"
	}
	return "In build-up"
}

// somethingToTest is the news a prerelease is, in words that cannot be mistaken
// for the table's word.
//
// decisions/0009 says a prerelease is not nothing and that dropping it would
// lose the one piece of news a plugin that does not ship can offer, and that it
// belongs on the page rather than in the state. This is that sentence, and it is
// empty where there is nothing to say rather than saying there is none.
func somethingToTest(published releases.Repository) string {
	if published.Prereleases == 0 {
		return ""
	}
	if published.Prereleases == 1 {
		return " One prerelease is published, which is something to test rather than something to run."
	}
	return fmt.Sprintf(" %d prereleases are published, which is something to test rather than something to run.", published.Prereleases)
}

// saidAboutTheRows is the sentence above the table. It is composed here rather
// than written into the template, because it states how many rows there are and
// where they came from, and a number in the template is a number the template
// would have to be edited to keep true.
func saidAboutTheRows(rows []plugin, taken string) string {
	return fmt.Sprintf(
		"%d rows, read from the roster this repository carries. What each plugin has published was read on %s, and the state below is computed from that rather than declared: a row says what is true before anything is published, and a finished release raises it.",
		len(rows), taken)
}
