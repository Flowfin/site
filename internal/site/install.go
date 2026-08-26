// SPDX-License-Identifier: AGPL-3.0-or-later

// The install page, which is the one page on this site a reader arrives at with
// something to do rather than something to read.
//
// Two values on it are not prose and neither is written here. The catalogue
// address is a commitment published outside this repository, so it is read out
// of the tree and the page states whichever state that value is in; and which
// plugins can actually be installed is computed from what each repository has
// published, so a reader is never given steps for something with no release
// behind it.
//
// The address is the reason this file carries a state at all. An address that
// has not been settled and an address somebody dropped are the same empty
// string, and both render as instructions with a hole where the thing to paste
// should be, on a page that otherwise looks finished. Here they cannot be the
// same thing: an answered entry with nothing in it is refused, and an undecided
// one renders as a sentence naming what it waits on and printing no address at
// all. That is the same distinction the legal notice keeps for the same reason,
// and the two states are the ones that file declares rather than a second pair.
//
// What the page must never do is print steps and leave out which plugins they
// apply to. A reader who follows an install page for a plugin that has published
// nothing spends their time on a server list that will never show it, and the
// page that did that to them looked finished while doing it. So the list is not
// optional: where nothing ships, the page says that in a sentence rather than
// rendering an empty space.
package site

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// InstallFile is where the catalogue address is read from, InstallProse is the
// page's words, and InstallPath is where the produced page lands. The address
// this page is served at is the one decisions/0008-the-url-shape.md gives it.
const (
	InstallFile  = "data/catalogue.json"
	InstallProse = ContentDir + "/install.txt"
	InstallPath  = "install/index.html"
)

// catalogue is what the tree says about the address an operator pastes into a
// server. State is one of the two the legal notice declares, Address is the
// value an answered one carries, and Waiting is what an undecided one is
// waiting on. Exactly one of the last two carries anything.
type catalogue struct {
	State   string `json:"state"`
	Address string `json:"address"`
	Waiting string `json:"waiting"`
}

var catalogueFields = map[string]bool{"state": true, "address": true, "waiting": true}

// readCatalogue reads the address the install page prints, or every reason it
// will not.
//
// Each refusal is a shape that renders as a finished page. An answered entry
// with no address is the hole this file exists to prevent. An undecided one
// carrying an address is an address somebody believes is published and no reader
// will ever see. An undecided one naming nothing to wait on reads as a question
// nobody is holding. An answered one still naming something reads as open while
// showing an answer. And an address that is not one a server can fetch is worse
// than none: it is followed, it fails inside the server, and nothing about the
// page says which of the two ends of it was wrong.
func readCatalogue(name string) (catalogue, error) {
	body, err := os.ReadFile(filepath.Clean(name))
	if err != nil {
		return catalogue{}, err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return catalogue{}, fmt.Errorf("%s is not the object this file has to be: %w", filepath.ToSlash(name), err)
	}

	var reasons []string
	for field := range fields {
		if !catalogueFields[field] {
			reasons = append(reasons, fmt.Sprintf("it carries the field %q, which is not a field of this file", field))
		}
	}
	var c catalogue
	if err := json.Unmarshal(body, &c); err != nil {
		reasons = append(reasons, fmt.Sprintf("it does not carry the fields this file carries: %v", err))
	} else {
		reasons = append(reasons, refusedCatalogue(c)...)
	}

	if len(reasons) > 0 {
		return catalogue{}, fmt.Errorf("%s was refused, %d reason(s):\n  %s",
			filepath.ToSlash(name), len(reasons), strings.Join(reasons, "\n  "))
	}
	return c, nil
}

// refusedCatalogue is every reason the value will not render, rather than the
// first, so a file with two things wrong with it is repaired once.
func refusedCatalogue(c catalogue) []string {
	var reasons []string
	switch c.State {
	case Answered:
		switch {
		case strings.TrimSpace(c.Address) == "":
			reasons = append(reasons, fmt.Sprintf(
				"it is %s and carries no address, which renders as an install page with nothing to paste on a page that looks finished", Answered))
		case !strings.HasPrefix(c.Address, catalogueScheme):
			reasons = append(reasons, fmt.Sprintf(
				"it is %s and carries the address %q, which does not open with %s: a catalogue is fetched by the server rather than opened by the reader, so an address it cannot fetch fails inside the server and says nothing about which of the two ends was wrong",
				Answered, c.Address, catalogueScheme))
		}
		if strings.TrimSpace(c.Waiting) != "" {
			reasons = append(reasons, fmt.Sprintf(
				"it is %s and still names %q as what it waits on, which reads as open and shows an answer at the same time", Answered, c.Waiting))
		}
	case Undecided:
		if strings.TrimSpace(c.Address) != "" {
			reasons = append(reasons, fmt.Sprintf(
				"it is %s and carries the address %q, which no reader will see and which whoever wrote it has stopped looking at", Undecided, c.Address))
		}
		if strings.TrimSpace(c.Waiting) == "" {
			reasons = append(reasons, fmt.Sprintf(
				"it is %s and names nothing it waits on, which reads on the page as a question nobody is holding", Undecided))
		}
	case "":
		reasons = append(reasons, fmt.Sprintf(
			"it declares no state, and %s and %s are the only two this file may be in", Answered, Undecided))
	default:
		reasons = append(reasons, fmt.Sprintf(
			"it declares the state %q, and %s and %s are the only two this file may be in", c.State, Answered, Undecided))
	}
	return reasons
}

// catalogueScheme is what an address a server can fetch opens with. It is
// stated rather than parsed because what is being refused is the address that
// looks right in a page and is not fetchable, and the one spelling this project
// publishes anything under is the one below.
const catalogueScheme = "https://"

// saidAboutTheAddress is the sentence carrying the address, or the sentence
// saying there is none yet. Both are composed here rather than written into the
// prose, because the prose is the same on either answer and a second copy of the
// address in a paragraph is the copy that stays behind when the value moves.
func saidAboutTheAddress(c catalogue) string {
	if c.State == Answered {
		return fmt.Sprintf(
			"The address is %s. It is added once, in the server's administration pages under the plugin catalogues, and it keeps answering afterwards: what is published under it is what the server sees the next time it looks.",
			c.Address)
	}
	return fmt.Sprintf(
		"There is no address to add yet. %s is where that answer is taken, and this page says so rather than printing an address that would fail inside a server for as long as it took somebody to notice.",
		c.Waiting)
}

// installable is the rows a reader can act on, in the order the roster carries
// them, and the sentence above them.
//
// The subset is taken from the computed state rather than from the state word
// the table renders, so this page and that table cannot disagree about what
// ships: both are one read of what each repository has published. The sentence
// is composed here for the reason the table's is composed where its rows are
// read, which is that a count in a template is a count somebody has to edit to
// keep true.
func installable(rows []plugin) ([]plugin, string) {
	out := make([]plugin, 0, len(rows))
	for _, r := range rows {
		if r.Installable {
			out = append(out, r)
		}
	}
	switch {
	case len(rows) == 0:
		return nil, "There is no plugin list to compute this from, so this page says nothing about what can be installed rather than saying that nothing can."
	case len(out) == 0:
		return nil, fmt.Sprintf(
			"None of the %d plugins has a finished release published, so a server given the address today finds nothing in the catalogue to install. The steps above are what to do on the day one appears, and this line is what the page says instead of a list nobody can act on.",
			len(rows))
	case len(out) == 1:
		return out, fmt.Sprintf(
			"One of the %d plugins has a finished release published, so it is the one a server can install today. The rest appear here as they publish, and this list is computed from what each repository has published rather than written on this page.",
			len(rows))
	default:
		return out, fmt.Sprintf(
			"%d of the %d plugins have a finished release published, so those are the ones a server can install today. The rest appear here as they publish, and this list is computed from what each repository has published rather than written on this page.",
			len(out), len(rows))
	}
}

// writeInstall renders the install page out of its prose, the catalogue address
// and the rows the landing page already read.
//
// It is handed the rows rather than reading them again, for the reason the
// plugin pages are: that table and this list are one read of the roster, so they
// cannot disagree about what ships.
//
// Either source being absent is reported rather than passed over, the way the
// other pages report an absent one, and the report names which of the two was
// missing: a page with words and no address and a page with an address and no
// words are different repairs.
func writeInstall(root, out, label string, rows []plugin, tmpl *template.Template, said descriptions, mark string, log io.Writer) ([]string, error) {
	prose := filepath.Join(root, filepath.FromSlash(InstallProse))
	values := filepath.Join(root, filepath.FromSlash(InstallFile))
	for _, source := range []struct{ name, path string }{{InstallProse, prose}, {InstallFile, values}} {
		if _, err := os.Stat(source.path); os.IsNotExist(err) {
			fmt.Fprintf(log, "no %s in the tree, so no install page was written\n", source.name)
			return nil, nil
		}
	}

	p, err := readInstall(prose)
	if err != nil {
		return nil, fmt.Errorf("reading the install prose: %w", err)
	}
	c, err := readCatalogue(values)
	if err != nil {
		return nil, fmt.Errorf("reading the catalogue address: %w", err)
	}
	p.Paragraphs = sayWhatTheClientsAre(p.Paragraphs, saidAboutTheAddress(c))
	p.Installable, p.InstallableRead = installable(rows)
	p.locate(InstallPath, mark)
	if err := said.add(InstallPath, p.Description); err != nil {
		return nil, err
	}

	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, p); err != nil {
		return nil, fmt.Errorf("rendering the install page: %w", err)
	}
	name := filepath.Join(out, filepath.FromSlash(InstallPath))
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return nil, fmt.Errorf("creating the directory for %s: %w", InstallPath, err)
	}
	if err := os.WriteFile(name, []byte(rendered.String()), 0o644); err != nil {
		return nil, fmt.Errorf("writing %s: %w", InstallPath, err)
	}
	slashed := path.Join(label, InstallPath)
	fmt.Fprintf(log, "wrote %s (%d bytes, address %s, %d of %d installable)\n",
		slashed, rendered.Len(), c.State, len(p.Installable), len(rows))
	return []string{slashed}, nil
}

// readInstall reads the page's words. It carries no address and no rows, so what
// it refuses is a page with nothing to read and a page that sends a reader
// nowhere: this is the page a reader arrives at with a server open in front of
// them, and one that offers no way back leaves them with steps and no list.
func readInstall(name string) (page, error) {
	read, err := blocks(name)
	if err != nil {
		return page{}, err
	}

	var p page
	var reasons []string
	for i, b := range read {
		joined := strings.Join(b, " ")
		switch {
		case i == 0:
			p.Title = joined
		case strings.HasPrefix(joined, descriptionKeyword):
			text, reason := describe(joined)
			if reason != "" {
				reasons = append(reasons, reason)
				continue
			}
			p.Description = text
		case strings.HasPrefix(joined, onwardKeyword):
			text, address, reason := splitOnward(joined)
			if reason != "" {
				reasons = append(reasons, reason)
				continue
			}
			p.Onward = append(p.Onward, link{Text: text, Href: address})
		case keywordLine.MatchString(joined):
			reasons = append(reasons, fmt.Sprintf(
				"a block opens %q, which is neither %s nor %s, so a block that opens like one and is read as a paragraph loses whatever it was for: %s",
				strings.SplitN(joined, ":", 2)[0]+":", descriptionKeyword, onwardKeyword, short(joined)))
		default:
			p.Paragraphs = append(p.Paragraphs, joined)
		}
	}

	if p.Title == "" {
		reasons = append(reasons, "the file carries no title line")
	}
	if p.Description == "" {
		reasons = append(reasons, missingDescription())
	}
	if len(p.Onward) == 0 {
		reasons = append(reasons, fmt.Sprintf(
			"the file carries no %s block, and an install page that offers no way back is where a reader who arrived from the table stops, on a page that renders exactly like a finished one",
			onwardKeyword))
	}
	if len(reasons) > 0 {
		return page{}, fmt.Errorf("%s was refused, %d reason(s):\n  %s",
			filepath.ToSlash(name), len(reasons), strings.Join(reasons, "\n  "))
	}
	return p, nil
}
