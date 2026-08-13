// The legal notice, which is the page saying who publishes this site and how to
// reach them.
//
// Two obligations meet on it and the page is clearer for keeping them apart.
// Publishing a site under a name of one's own carries a provider identification
// duty in some jurisdictions, and which one applies turns on where the publisher
// sits and on whether the site is commercial rather than on anything in this
// repository. Separately, the privacy page describes what happens to a reader's
// request, and a description of that kind is expected to name who is answerable
// for it. This page cites that one rather than repeating it, so there is one copy
// of the data statement and not two.
//
// What this file does not hold is the words that identify anybody. The identity,
// the contact route and whether an address is published at all are open on the
// tracker, because the cost of each answer is paid by a person rather than by the
// build. So every value the page shows is read out of the tree, and an answer
// that has not been taken is a state in the data rather than a blank.
//
// That is the whole reason for the closed set below. A value nobody has decided
// and a value somebody forgot are the same empty string, and they render as the
// same empty element on a page that looks finished. Here they cannot be the same
// thing: an entry says which of the two states it is in, an answered one with
// nothing in it is refused, and an undecided one renders as a sentence naming
// what it waits on. A reader of the page can then tell a question that is open
// from an answer that went missing, which is the distinction the page exists to
// keep.
package site

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// LegalFile is what the page's values are read from, LegalProse is its words, and
// LegalPath is where the produced page lands. The address is the one
// decisions/0008-the-url-shape.md gives it.
const (
	LegalFile  = "data/publisher.json"
	LegalProse = ContentDir + "/legal.txt"
	LegalPath  = "legal/index.html"
)

// LegalAddress is the address the frame links every page to. It is derived from
// the path the page is written to rather than written twice, because the row that
// refuses a page missing the link compares against this.
var LegalAddress = addressOf(LegalPath)

// The two states an entry may be in, and there is no third. A page cannot show a
// value it does not have, so the only question is whether it says why.
const (
	Answered  = "answered"
	Undecided = "undecided"
)

// notice is one thing the page answers, or says it cannot. Asks is the question
// in the words a reader reads, Answer is what it is answered with, and Waiting is
// what an unanswered one is waiting on. Exactly one of the last two carries
// anything, which is what the loader refuses everything else for.
type notice struct {
	Asks    string
	Answer  string
	Waiting string
}

// entry is one row of the file.
type entry struct {
	Asks    string `json:"asks"`
	State   string `json:"state"`
	Value   string `json:"value"`
	Waiting string `json:"waiting"`
}

var entryFields = map[string]bool{"asks": true, "state": true, "value": true, "waiting": true}

// readPublisher reads what the tree says about who publishes this site, in the
// order the file gives, and returns every reason it will not rather than the
// first.
//
// The order is the file's rather than sorted, because the questions are read as a
// sequence: who publishes comes before how to reach them. A map would lose that,
// so the keys are read out of the raw object and put back in the order they were
// written.
func readPublisher(name string) ([]notice, error) {
	body, err := os.ReadFile(filepath.Clean(name))
	if err != nil {
		return nil, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%s is not the object this file has to be: %w", filepath.ToSlash(name), err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf(
			"%s carries no entry, and a legal notice with nothing on it is a page that says who publishes this site by saying nothing",
			filepath.ToSlash(name))
	}

	var reasons []string
	var out []notice
	for _, key := range keysInFileOrder(body, raw) {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw[key], &fields); err != nil {
			reasons = append(reasons, fmt.Sprintf("%q is not an object: %v", key, err))
			continue
		}
		for field := range fields {
			if !entryFields[field] {
				reasons = append(reasons, fmt.Sprintf("%q carries the field %q, which is not a field of an entry", key, field))
			}
		}
		var e entry
		if err := json.Unmarshal(raw[key], &e); err != nil {
			reasons = append(reasons, fmt.Sprintf("%q does not carry the fields an entry carries: %v", key, err))
			continue
		}
		n, reason := readEntry(key, e)
		if reason != "" {
			reasons = append(reasons, reason)
			continue
		}
		out = append(out, n)
	}

	if len(reasons) > 0 {
		return nil, fmt.Errorf("%s was refused, %d reason(s):\n  %s",
			filepath.ToSlash(name), len(reasons), strings.Join(reasons, "\n  "))
	}
	return out, nil
}

// readEntry decides one entry, or says why it will not.
//
// Every refusal here is a shape that renders as a finished page. An answered
// entry with nothing in it is the blank this file exists to prevent. An undecided
// one carrying a value is a value somebody believes is published and no reader
// will ever see, which is worse than a blank because whoever wrote it has stopped
// looking. An undecided one naming nothing to wait on reads as a question nobody
// is holding. And an answered one still naming something reads as open while
// showing an answer, which is the pair a reader cannot resolve.
func readEntry(key string, e entry) (notice, string) {
	if strings.TrimSpace(e.Asks) == "" {
		return notice{}, fmt.Sprintf("%q asks nothing, and an entry with no question renders as an answer to a question the page never puts", key)
	}
	switch e.State {
	case Answered:
		if strings.TrimSpace(e.Value) == "" {
			return notice{}, fmt.Sprintf("%q is %s and carries no value, which renders as an empty answer on a page that looks finished", key, Answered)
		}
		if strings.TrimSpace(e.Waiting) != "" {
			return notice{}, fmt.Sprintf("%q is %s and still names %q as what it waits on, which reads as open and shows an answer at the same time", key, Answered, e.Waiting)
		}
		return notice{Asks: e.Asks, Answer: e.Value}, ""
	case Undecided:
		if strings.TrimSpace(e.Value) != "" {
			return notice{}, fmt.Sprintf("%q is %s and carries the value %q, which no reader will see and which whoever wrote it has stopped looking at", key, Undecided, e.Value)
		}
		if strings.TrimSpace(e.Waiting) == "" {
			return notice{}, fmt.Sprintf("%q is %s and names nothing it waits on, which reads on the page as a question nobody is holding", key, Undecided)
		}
		return notice{Asks: e.Asks, Waiting: e.Waiting}, ""
	case "":
		return notice{}, fmt.Sprintf("%q declares no state, and %s and %s are the only two an entry may be in", key, Answered, Undecided)
	default:
		return notice{}, fmt.Sprintf("%q declares the state %q, and %s and %s are the only two an entry may be in", key, e.State, Answered, Undecided)
	}
}

// keysInFileOrder returns the object's keys in the order they were written. The
// standard library hands them back as a map, which has no order, and the page
// reads as a sequence.
func keysInFileOrder(body []byte, raw map[string]json.RawMessage) []string {
	at := make(map[string]int, len(raw))
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
		// The quoted key as it appears in the object. Two keys cannot
		// share an index, so the first occurrence orders them.
		at[key] = strings.Index(string(body), `"`+key+`"`)
	}
	sort.Slice(keys, func(i, j int) bool { return at[keys[i]] < at[keys[j]] })
	return keys
}

// writeLegal renders the legal notice out of its prose and its values, through
// the same template every other page goes through.
//
// Either source being absent is reported rather than passed over, the way the
// assets walk and the other pages report an absent one, and the report names
// which of the two was missing: a page with words and no values and a page with
// values and no words are different repairs.
func writeLegal(root, out, label string, tmpl *template.Template, said descriptions, log io.Writer) ([]string, error) {
	prose := filepath.Join(root, filepath.FromSlash(LegalProse))
	values := filepath.Join(root, filepath.FromSlash(LegalFile))
	for _, source := range []struct{ name, path string }{{LegalProse, prose}, {LegalFile, values}} {
		if _, err := os.Stat(source.path); os.IsNotExist(err) {
			fmt.Fprintf(log, "no %s in the tree, so no legal notice was written\n", source.name)
			return nil, nil
		}
	}

	p, err := readLegal(prose)
	if err != nil {
		return nil, fmt.Errorf("reading the legal notice prose: %w", err)
	}
	p.Notices, err = readPublisher(values)
	if err != nil {
		return nil, fmt.Errorf("reading who publishes this site: %w", err)
	}
	p.locate(LegalPath)
	if err := said.add(LegalPath, p.Description); err != nil {
		return nil, err
	}

	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, p); err != nil {
		return nil, fmt.Errorf("rendering the legal notice: %w", err)
	}
	name := filepath.Join(out, filepath.FromSlash(LegalPath))
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return nil, fmt.Errorf("creating the directory for %s: %w", LegalPath, err)
	}
	if err := os.WriteFile(name, []byte(rendered.String()), 0o644); err != nil {
		return nil, fmt.Errorf("writing %s: %w", LegalPath, err)
	}
	slashed := path.Join(label, LegalPath)

	answered := 0
	for _, n := range p.Notices {
		if n.Answer != "" {
			answered++
		}
	}
	fmt.Fprintf(log, "wrote %s (%d bytes, %d of %d answered)\n",
		slashed, rendered.Len(), answered, len(p.Notices))
	return []string{slashed}, nil
}

// readLegal reads the page's words. It carries no values, so what it refuses is a
// page with nothing to read and a page that does not send a reader to the one it
// is not allowed to repeat.
func readLegal(name string) (page, error) {
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
	if !citesPrivacy(p.Onward) {
		reasons = append(reasons, fmt.Sprintf(
			"the file sends no reader to %s, and a legal notice that does not is a page somebody will answer by repeating what that page says, which is the second copy this page exists to avoid",
			addressOf(PrivacyPath)))
	}
	if len(reasons) > 0 {
		return page{}, fmt.Errorf("%s was refused, %d reason(s):\n  %s",
			filepath.ToSlash(name), len(reasons), strings.Join(reasons, "\n  "))
	}
	return p, nil
}

// citesPrivacy answers whether the page sends a reader to the page describing
// what happens to a request. The address is derived from where the build writes
// that page, so a page that moves does not leave this comparison pointing at an
// address nothing answers at.
func citesPrivacy(onward []link) bool {
	want := addressOf(PrivacyPath)
	for _, l := range onward {
		if l.Href == want {
			return true
		}
	}
	return false
}
