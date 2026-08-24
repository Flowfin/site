// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the legal notice.
//
// The page's whole difficulty is that most of what it exists to say is not
// decided yet, and a page in that state has two ways of looking finished while
// saying nothing. A value nobody has taken and a value somebody dropped are the
// same empty string, and they render as the same empty element. So every case
// below is a file that produces a page a reader would read as complete, and each
// one is refused for the reason it is wrong rather than for being empty.
package site

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// legalTemplate renders the page's words, the way onward and the entries, so a
// case reads what the build put on the page rather than whatever the
// repository's frame carries today.
const legalTemplate = `<title>{{ .Title }}</title>
<meta name="description" content="{{ .Description }}" />
<h1>{{ .Title }}</h1>
{{- range .Paragraphs }}
<p>{{ . }}</p>
{{- end }}
{{- range .Notices }}
<dt>{{ .Asks }}</dt>
{{- if .Answer }}
<dd>{{ .Answer }}</dd>
{{- else }}
<dd>Not decided yet. {{ .Waiting }} is where that answer is taken.</dd>
{{- end }}
{{- end }}
{{- range .Onward }}
<a href="{{ .Href }}">{{ .Text }}</a>
{{- end }}
`

const legalProse = `Who publishes this site

description: Who publishes this site and how to reach them.

One paragraph.

onward: What happens to a request for a page here [/privacy/]
`

// The smallest file of values that is a legal notice: one question, undecided,
// naming what it waits on.
const legalValues = `{
  "publisher": {
    "asks": "Who publishes this site",
    "state": "undecided",
    "waiting": "an open question"
  }
}
`

// legalTree is a tree that produces the landing page and the legal notice.
func legalTree(t *testing.T, prose, values string) string {
	t.Helper()
	root := t.TempDir()
	mkdir(t, filepath.Join(root, TemplatesDir))
	mkdir(t, filepath.Join(root, ContentDir))
	mkdir(t, filepath.Join(root, filepath.Dir(filepath.FromSlash(LegalFile))))
	write(t, filepath.Join(root, TemplatesDir, "page.html.tmpl"), legalTemplate)
	write(t, filepath.Join(root, ContentDir, "index.txt"), headIndex)
	write(t, filepath.Join(root, filepath.FromSlash(LegalProse)), prose)
	write(t, filepath.Join(root, filepath.FromSlash(LegalFile)), values)
	return root
}

// legalRefusal builds a tree that must be refused and returns what the refusal
// said.
func legalRefusal(t *testing.T, prose, values string) string {
	t.Helper()
	_, err := Build(legalTree(t, prose, values), OutputDir, io.Discard)
	if err == nil {
		t.Fatalf("the build was accepted, and it carries the mistake this case is about:\n%s", values)
	}
	return err.Error()
}

// The page is produced, at the address the record gives it, and what it shows
// comes out of the file rather than out of the template.
func TestTheLegalNoticeIsAProducedPage(t *testing.T) {
	root := legalTree(t, legalProse, legalValues)
	written, err := Build(root, OutputDir, io.Discard)
	if err != nil {
		t.Fatalf("the build refused: %v", err)
	}
	want := OutputDir + "/" + LegalPath
	var found bool
	for _, w := range written {
		if w == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("the build wrote %q and none of them is %s", written, want)
	}
	body := read(t, filepath.Join(root, OutputDir, filepath.FromSlash(LegalPath)))
	if !strings.Contains(body, "<dt>Who publishes this site</dt>") {
		t.Errorf("the question in the file did not reach the page:\n%s", body)
	}
}

// An answer that has been taken renders as itself, and changing the value in the
// file changes the page. That is what makes the answer a data change rather than
// an edit to prose, which matters because the answer is the part of this page
// most likely to move.
func TestChangingAValueChangesThePage(t *testing.T) {
	answered := `{
  "publisher": {
    "asks": "Who publishes this site",
    "state": "answered",
    "value": "A name in the tree"
  }
}
`
	root := legalTree(t, legalProse, answered)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused: %v", err)
	}
	body := read(t, filepath.Join(root, OutputDir, filepath.FromSlash(LegalPath)))
	if !strings.Contains(body, "<dd>A name in the tree</dd>") {
		t.Fatalf("the answer did not reach the page:\n%s", body)
	}

	write(t, filepath.Join(root, filepath.FromSlash(LegalFile)),
		strings.Replace(answered, "A name in the tree", "A different name", 1))
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused after the value changed: %v", err)
	}
	body = read(t, filepath.Join(root, OutputDir, filepath.FromSlash(LegalPath)))
	if strings.Contains(body, "A name in the tree") {
		t.Errorf("the old value is still on the page:\n%s", body)
	}
	if !strings.Contains(body, "<dd>A different name</dd>") {
		t.Errorf("the new value did not reach the page:\n%s", body)
	}
}

// An answer nobody has taken renders as a sentence saying so and naming what it
// waits on. This is the whole point of the state: an empty element and an open
// question look identical to a reader, and only one of them is honest.
func TestAnUndecidedAnswerRendersAsASentence(t *testing.T) {
	root := legalTree(t, legalProse, legalValues)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused: %v", err)
	}
	body := read(t, filepath.Join(root, OutputDir, filepath.FromSlash(LegalPath)))
	if strings.Contains(body, "<dd></dd>") {
		t.Errorf("an undecided answer rendered as an empty element:\n%s", body)
	}
	for _, want := range []string{"Not decided yet.", "an open question"} {
		if !strings.Contains(body, want) {
			t.Errorf("the page does not carry %q:\n%s", want, body)
		}
	}
}

// The four shapes that render as a finished page and are wrong, each named by
// what a reader would take from it.
func TestTheBuildRefusesAnEntryThatWouldRenderAsFinished(t *testing.T) {
	for name, values := range map[string]string{
		"answered with nothing in it":  `{"publisher":{"asks":"Who","state":"answered"}}`,
		"answered and still waiting":   `{"publisher":{"asks":"Who","state":"answered","value":"A name","waiting":"an open question"}}`,
		"undecided with a value":       `{"publisher":{"asks":"Who","state":"undecided","value":"A name","waiting":"an open question"}}`,
		"undecided waiting on nothing": `{"publisher":{"asks":"Who","state":"undecided"}}`,
	} {
		got := legalRefusal(t, legalProse, values)
		if !strings.Contains(got, "publisher") {
			t.Errorf("the refusal for an entry %s reads %q and does not name the entry", name, got)
		}
	}
}

// A state outside the two, and a state left off. Both are refused by name rather
// than read as one of the two, because guessing which was meant is how a value
// ends up in the register that does not check it.
func TestTheBuildRefusesAStateOutsideTheTwo(t *testing.T) {
	for name, values := range map[string]string{
		"a third state": `{"publisher":{"asks":"Who","state":"pending","value":"A name"}}`,
		"no state":      `{"publisher":{"asks":"Who","value":"A name"}}`,
	} {
		got := legalRefusal(t, legalProse, values)
		for _, want := range []string{Answered, Undecided} {
			if !strings.Contains(got, want) {
				t.Errorf("the refusal for %s reads %q and does not name %q as one of the two", name, got, want)
			}
		}
	}
}

// An entry with no question renders as an answer to something the page never
// asked, which is a value with no meaning attached to it.
func TestTheBuildRefusesAnEntryThatAsksNothing(t *testing.T) {
	got := legalRefusal(t, legalProse, `{"publisher":{"state":"answered","value":"A name"}}`)
	if !strings.Contains(got, "asks nothing") {
		t.Errorf("the refusal reads %q", got)
	}
}

// A field nothing reads is refused rather than ignored, for the reason the roster
// parser gives: a field the build passes over is a field somebody added believing
// it did something, and the day they notice is the day the page has been wrong
// for a month.
func TestTheBuildRefusesAFieldNothingReads(t *testing.T) {
	got := legalRefusal(t, legalProse,
		`{"publisher":{"asks":"Who","state":"answered","value":"A name","address":"A street"}}`)
	if !strings.Contains(got, "address") {
		t.Errorf("the refusal reads %q and does not name the field", got)
	}
}

// A file with no entry at all, which is a legal notice that answers who publishes
// this site by saying nothing.
func TestTheBuildRefusesAFileWithNoEntry(t *testing.T) {
	got := legalRefusal(t, legalProse, `{}`)
	if !strings.Contains(got, "no entry") {
		t.Errorf("the refusal reads %q", got)
	}
}

// The page cites the one describing what happens to a request rather than
// summarising it, and a file that does not is refused. A summary is a second copy
// of those statements, and the copy is what goes out of step.
func TestTheBuildRefusesAPageThatDoesNotCiteThePrivacyPage(t *testing.T) {
	got := legalRefusal(t,
		strings.Replace(legalProse, "onward: What happens to a request for a page here [/privacy/]\n", "", 1),
		legalValues)
	if !strings.Contains(got, addressOf(PrivacyPath)) {
		t.Errorf("the refusal reads %q and does not name the page that was not cited", got)
	}
}

// Every reason is collected rather than the first, because a file with three
// mistakes in it is three repairs and reporting one of them costs three runs.
func TestTheBuildReportsEveryEntryItRefused(t *testing.T) {
	got := legalRefusal(t, legalProse, `{
  "publisher": {"asks":"Who","state":"answered"},
  "contact": {"asks":"How","state":"pending"},
  "postal": {"asks":"Whether","state":"undecided"}
}`)
	for _, want := range []string{"publisher", "contact", "postal", "3 reason(s)"} {
		if !strings.Contains(got, want) {
			t.Errorf("the refusal reads %q and does not carry %q", got, want)
		}
	}
}

// The questions are read as a sequence rather than as a set, because who
// publishes a site comes before how to reach them and a page that shuffled them
// on every build would not reproduce either.
func TestTheEntriesKeepTheOrderTheFileGivesThem(t *testing.T) {
	root := legalTree(t, legalProse, `{
  "publisher": {"asks":"First question","state":"undecided","waiting":"an open question"},
  "contact": {"asks":"Second question","state":"undecided","waiting":"an open question"},
  "postal": {"asks":"Third question","state":"undecided","waiting":"an open question"}
}`)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused: %v", err)
	}
	body := read(t, filepath.Join(root, OutputDir, filepath.FromSlash(LegalPath)))
	first := strings.Index(body, "First question")
	second := strings.Index(body, "Second question")
	third := strings.Index(body, "Third question")
	if first < 0 || second < 0 || third < 0 {
		t.Fatalf("not every question reached the page:\n%s", body)
	}
	if !(first < second && second < third) {
		t.Errorf("the questions are on the page in another order:\n%s", body)
	}
}

// A value out of the file is escaped on the way into the page by the path every
// other page takes, so a name containing a bracket cannot become markup.
func TestTheLegalNoticeEscapesTheValuesItRenders(t *testing.T) {
	root := legalTree(t, legalProse,
		`{"publisher":{"asks":"Who","state":"answered","value":"A <b>name</b>"}}`)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused: %v", err)
	}
	body := read(t, filepath.Join(root, OutputDir, filepath.FromSlash(LegalPath)))
	if strings.Contains(body, "<b>name</b>") {
		t.Errorf("the markup in the value reached the page as markup:\n%s", body)
	}
	if !strings.Contains(body, "&lt;b&gt;name&lt;/b&gt;") {
		t.Errorf("the markup in the value did not reach the page as text:\n%s", body)
	}
}

// Either source absent is reported rather than passed over, and the report names
// which of the two was missing, because a page with words and no values and a
// page with values and no words are different repairs.
func TestTheBuildSaysWhichOfTheTwoSourcesIsAbsent(t *testing.T) {
	for _, absent := range []string{LegalProse, LegalFile} {
		root := legalTree(t, legalProse, legalValues)
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(absent))); err != nil {
			t.Fatalf("preparing the tree: %v", err)
		}
		var log strings.Builder
		if _, err := Build(root, OutputDir, &log); err != nil {
			t.Fatalf("the build refused a tree with no %s: %v", absent, err)
		}
		if !strings.Contains(log.String(), absent) {
			t.Errorf("the run said nothing about the absent %s:\n%s", absent, log.String())
		}
		if _, err := os.Stat(filepath.Join(root, OutputDir, filepath.FromSlash(LegalPath))); !os.IsNotExist(err) {
			t.Errorf("a page was written with no %s in the tree", absent)
		}
	}
}
