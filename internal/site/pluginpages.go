// SPDX-License-Identifier: AGPL-3.0-or-later

// One page per roster row, all of them rendered through the frame every other
// page is rendered through.
//
// The template is the reason this repository has a generator at all. Nothing
// about a plugin is typed twice and nothing here knows how many plugins there
// are: a thirteenth is a row in the roster, and neither this file nor the frame
// nor any check learns a number when it arrives.
//
// A plugin whose row declares the shell state gets the same page as any other.
// Hiding it would make the site disagree with the list it is generated from,
// and the reader arriving at a shell is the reader most likely to have been
// sent by the table and least likely to know what they are looking at.
//
// Every value out of the roster reaches the page through the same engine the
// prose does, so a sentence carrying a bracket renders as text rather than as
// markup. That is a property of this path rather than of the rows this tree
// carries today, and the suite exercises it with data that tries.
package site

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// writePluginPages renders one page per row and reports what it wrote.
//
// It is handed the rows rather than reading them again, so the table on the
// landing page and the pages it links cannot disagree about what the roster
// says. A row with no page and a page with no row are the same failure seen
// from two sides, and there is one read behind both.
func writePluginPages(rows []plugin, out, label string, tmpl *template.Template, said descriptions, log io.Writer) ([]string, error) {
	var written []string
	for _, r := range rows {
		p := page{
			Title:       r.ID,
			Description: r.Summary,
			Paragraphs:  paragraphsFor(r),
			Onward: []link{
				{Text: "The repository this plugin is built in", Href: "https://github.com/" + r.Repository},
				{Text: "Every plugin and what state it is in", Href: "/"},
			},
		}
		p.locate(r.Produced)
		if err := said.add(r.Produced, p.Description); err != nil {
			return nil, err
		}

		var rendered strings.Builder
		if err := tmpl.Execute(&rendered, p); err != nil {
			return nil, fmt.Errorf("rendering the page for %s: %w", r.ID, err)
		}
		name := filepath.Join(out, filepath.FromSlash(r.Produced))
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			return nil, fmt.Errorf("creating the directory for %s: %w", r.ID, err)
		}
		if err := os.WriteFile(name, []byte(rendered.String()), 0o644); err != nil {
			return nil, fmt.Errorf("writing the page for %s: %w", r.ID, err)
		}
		slashed := path.Join(label, r.Produced)
		written = append(written, slashed)
		fmt.Fprintf(log, "wrote %s (%d bytes)\n", slashed, rendered.Len())
	}
	if len(written) > 0 {
		fmt.Fprintf(log, "%d plugin page(s) written, one per roster row\n", len(written))
	}
	return written, nil
}

// paragraphsFor is what a plugin's page says, in the order it says it.
//
// The roster sentence opens, because it is what the table sent the reader here
// with and a page that opens on something else reads as a different plugin. The
// file's paragraphs follow it, which is what the page is for: what the plugin
// does in more than one sentence, what an operator needs before it is any use,
// and what it does with data about a person.
//
// The state sentence stays after them and stays computed. What state a plugin
// is in is a fact about published releases rather than an opinion a file may
// hold, which is decisions/0001's rule, and a second copy of it in prose is a
// copy that goes stale the day something is published and says so to a reader
// who has no way to tell which half is current.
func paragraphsFor(r plugin) []string {
	said := make([]string, 0, len(r.Prose)+len(r.Generations)+3)
	said = append(said, r.Summary)
	said = append(said, r.Prose...)
	said = append(said, r.Means)
	// The generation lines sit under the state sentence, because which
	// server a build is for is only a question once there is a build, and
	// the sentence above them is what says whether there is one.
	said = append(said, generationLines(r)...)
	return append(said, whereItIs(r))
}

// whereItIs is the sentence naming the repository. It is composed rather than
// carried on the row, because it is a sentence about a value the row already
// holds and a second copy of that value on the row is a second thing to keep.
func whereItIs(r plugin) string {
	return fmt.Sprintf(
		"The plugin is built in the repository %s, and what this page says about it is read from the roster rather than written here.",
		r.Repository)
}
