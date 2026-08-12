// The suite over the file the privacy page is made of.
//
// What it proves is the register. Every case below is a statement that would
// render as a finished sentence on a finished page while standing in the wrong
// register, or in none, and a reader has no way to see the difference. So each
// one is refused at the build rather than reported by whoever notices, and each
// refusal is tripped by one mistake so that a red run says which repair it
// wants.
package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The smallest file that is a privacy page: a title, one checked statement and
// one residual. Every case below is this file with one thing changed, so what a
// case proves is that one thing.
const smallest = `Privacy

checked: No page fetches a script. [page-fetches-no-script]

residual: The host sees the request.
`

// registerTemplate is the fixture's own template, and it renders the three
// registers the way the repository's does. It is a fixture rather than the real
// file for the reason the suite beside this one gives: a case that read the
// repository's template would prove the state of the tree on the day it ran
// rather than what the build does with what it is given.
const registerTemplate = `<title>{{ .Title }}</title>
<h1>{{ .Title }}</h1>
{{- range .Paragraphs }}
<p>{{ . }}</p>
{{- end }}
{{- range .Claims }}
<li data-refused-by="{{ .RefusedBy }}">{{ .Text }} Refused by <code>{{ .RefusedBy }}</code>.</li>
{{- end }}
{{- range .Promises }}
<li>{{ .Text }} {{ .Waiting }}</li>
{{- end }}
{{- range .Residuals }}
<p>{{ . }}</p>
{{- end }}
`

// privacyTree is a buildable tree whose template renders the registers, and
// whose privacy prose is the body given.
func privacyTree(t *testing.T, prose string) string {
	t.Helper()
	root := tree(t, "A title\n\nOne paragraph.\n")
	write(t, filepath.Join(root, TemplatesDir, "page.html.tmpl"), registerTemplate)
	write(t, filepath.Join(root, ContentDir, "privacy.txt"), prose)
	return root
}

// privacyFile writes a body under a temporary root and returns the path, so a
// case reads through the same door a build does.
func privacyFile(t *testing.T, body string) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "privacy.txt")
	if err := os.WriteFile(name, []byte(body), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}
	return name
}

// refusal reads a body that must be refused and returns what the refusal said.
func refusal(t *testing.T, body string) string {
	t.Helper()
	_, err := readPrivacy(privacyFile(t, body))
	if err == nil {
		t.Fatalf("the file was accepted, and it carries the mistake this case is about:\n%s", body)
	}
	return err.Error()
}

func TestPrivacyReadsTheThreeRegistersApart(t *testing.T) {
	p, err := readPrivacy(privacyFile(t, `Privacy

An opening paragraph.

checked: No page fetches a script. [page-fetches-no-script]

promised: No cookie is set. [#50]

residual: The host sees the request.
`))
	if err != nil {
		t.Fatalf("a file breaking nothing was refused: %v", err)
	}
	if p.Title != "Privacy" {
		t.Errorf("the title is %q", p.Title)
	}
	if len(p.Paragraphs) != 1 || p.Paragraphs[0] != "An opening paragraph." {
		t.Errorf("the paragraphs are %q", p.Paragraphs)
	}
	if len(p.Claims) != 1 || p.Claims[0].RefusedBy != "page-fetches-no-script" {
		t.Fatalf("the checked statements are %+v", p.Claims)
	}
	if p.Claims[0].Text != "No page fetches a script." {
		t.Errorf("the sentence kept its marker: %q", p.Claims[0].Text)
	}
	if len(p.Promises) != 1 || p.Promises[0].Waiting != "#50" {
		t.Errorf("the promised statements are %+v", p.Promises)
	}
	if len(p.Residuals) != 1 || p.Residuals[0] != "The host sees the request." {
		t.Errorf("the residual statements are %q", p.Residuals)
	}
}

// A statement wrapped over several lines is one statement. The file is prose
// and prose is wrapped, so a parser that read a line as a statement would put
// the marker of a wrapped one on a sentence of its own.
func TestPrivacyJoinsAWrappedStatement(t *testing.T) {
	p, err := readPrivacy(privacyFile(t, `Privacy

checked: No page fetches a script, which is the whole of what
  this row decides.
  [page-fetches-no-script]

residual: The host sees the request.
`))
	if err != nil {
		t.Fatalf("a wrapped statement was refused: %v", err)
	}
	if len(p.Claims) != 1 {
		t.Fatalf("the wrapped statement made %d claim(s)", len(p.Claims))
	}
	want := "No page fetches a script, which is the whole of what this row decides."
	if p.Claims[0].Text != want {
		t.Errorf("the statement reads %q", p.Claims[0].Text)
	}
	if p.Claims[0].RefusedBy != "page-fetches-no-script" {
		t.Errorf("the marker on the last line was not read: %q", p.Claims[0].RefusedBy)
	}
}

// The one-character mistake this file exists for. A keyword with a letter
// missing is not a keyword, and the block behind it becomes a paragraph: the
// sentence still renders, nothing names what stands behind it, and no reader
// can tell.
func TestPrivacyRefusesAMistypedKeyword(t *testing.T) {
	said := refusal(t, strings.Replace(smallest, "checked:", "checkd:", 1))
	if !strings.Contains(said, "checkd:") {
		t.Errorf("the refusal does not name what was written: %s", said)
	}
}

// A checked statement with no marker renders exactly like one that carries a
// check, because what a reader sees is the sentence.
func TestPrivacyRefusesACheckedStatementThatNamesNothing(t *testing.T) {
	said := refusal(t, strings.Replace(smallest, " [page-fetches-no-script]", "", 1))
	if !strings.Contains(said, "names nothing behind it") {
		t.Errorf("the refusal is about something else: %s", said)
	}
}

// Empty brackets are the shape a template with an unset value writes, and on
// the page they read as a name.
func TestPrivacyRefusesEmptyBrackets(t *testing.T) {
	said := refusal(t, strings.Replace(smallest, "[page-fetches-no-script]", "[]", 1))
	if !strings.Contains(said, "empty brackets") {
		t.Errorf("the refusal is about something else: %s", said)
	}
}

// An issue number where a check name goes. This is a promise written into the
// checked register, which is the register mistake that costs the most: the page
// says a machine refuses what nothing refuses.
func TestPrivacyRefusesAnIssueNumberInTheCheckedRegister(t *testing.T) {
	said := refusal(t, strings.Replace(smallest, "[page-fetches-no-script]", "[#50]", 1))
	if !strings.Contains(said, "an issue rather than a check") {
		t.Errorf("the refusal is about something else: %s", said)
	}
}

// The same mistake in the other direction. A promise naming a check name reads
// as though the check exists, and this parser cannot ask whether it does, so it
// refuses the shape rather than the name.
func TestPrivacyRefusesACheckNameInThePromisedRegister(t *testing.T) {
	said := refusal(t, strings.Replace(smallest,
		"checked: No page fetches a script. [page-fetches-no-script]",
		"checked: No page fetches a script. [page-fetches-no-script]\n\npromised: No cookie is set. [page-fetches-no-script]", 1))
	if !strings.Contains(said, "names the issue that would refuse it") {
		t.Errorf("the refusal is about something else: %s", said)
	}
}

// A residual is what nothing refuses and nothing promises. One carrying a
// marker is a claim standing in the register the page uses to say it is not
// claiming.
func TestPrivacyRefusesAResidualThatNamesSomething(t *testing.T) {
	said := refusal(t, strings.Replace(smallest,
		"residual: The host sees the request.",
		"residual: The host sees the request. [page-fetches-no-script]", 1))
	if !strings.Contains(said, "in the wrong register") {
		t.Errorf("the refusal is about something else: %s", said)
	}
}

// A page with nothing checked on it is the page this one exists in order not to
// be, and it is what is left after somebody removes the statement a failing
// check was about.
func TestPrivacyRefusesAFileWithNothingChecked(t *testing.T) {
	said := refusal(t, `Privacy

residual: The host sees the request.
`)
	if !strings.Contains(said, "no checked statement") {
		t.Errorf("the refusal is about something else: %s", said)
	}
}

// What a host sees is true whatever this site does. A page that leaves it out
// reads as though the statements above it covered everything.
func TestPrivacyRefusesAFileWithNoResidual(t *testing.T) {
	said := refusal(t, `Privacy

checked: No page fetches a script. [page-fetches-no-script]
`)
	if !strings.Contains(said, "no residual statement") {
		t.Errorf("the refusal is about something else: %s", said)
	}
}

// Every reason at once rather than the first, because a file with three
// mistakes is three repairs and reporting one of them costs three runs.
func TestPrivacyReportsEveryReasonItRefused(t *testing.T) {
	said := refusal(t, `Privacy

checked: No page fetches a script. [#50]

promised: No cookie is set. [page-fetches-no-script]

residual: The host sees the request. [page-fetches-no-script]
`)
	for _, want := range []string{
		"an issue rather than a check",
		"names the issue that would refuse it",
		"in the wrong register",
	} {
		if !strings.Contains(said, want) {
			t.Errorf("the refusal does not carry the reason %q:\n%s", want, said)
		}
	}
	// Three statements in the wrong register leave the file with nothing in
	// the right one, and both of those are reported as well. A file is
	// refused for everything that is wrong with it, not for the first thing.
	if !strings.Contains(said, "5 reason(s)") {
		t.Errorf("the refusal reports a different number of reasons than the file carries:\n%s", said)
	}
}

// The build writes the page where the address record puts it, and says what it
// wrote there. A run that produced the page silently would leave a reader
// unable to tell it from a run that produced nothing.
func TestBuildWritesThePrivacyPageAndSaysWhatIsOnIt(t *testing.T) {
	root := privacyTree(t, smallest)

	var log strings.Builder
	written, err := Build(root, OutputDir, &log)
	if err != nil {
		t.Fatalf("the build refused: %v\n%s", err, log.String())
	}
	if len(written) != 2 || written[1] != "dist/privacy/index.html" {
		t.Fatalf("the build wrote %q", written)
	}
	if !strings.Contains(log.String(), "1 checked, 0 promised, 1 residual") {
		t.Errorf("the run does not say what the page carries:\n%s", log.String())
	}

	page := read(t, filepath.Join(root, "dist", "privacy", "index.html"))
	if !strings.Contains(page, `data-refused-by="page-fetches-no-script"`) {
		t.Errorf("the produced page cites no check:\n%s", page)
	}
	if !strings.Contains(page, "<code>page-fetches-no-script</code>") {
		t.Errorf("the produced page does not show the reader the name it cites:\n%s", page)
	}
}

// A statement carrying markup renders as text, by the same path the rest of the
// site is escaped by. The privacy page is the page where a sentence about what
// this site does not do would be the worst place to open an element.
func TestBuildRendersMarkupInAStatementAsText(t *testing.T) {
	root := privacyTree(t, `Privacy

checked: No <script> is fetched. [page-fetches-no-script]

residual: The host sees the request.
`)
	if _, err := Build(root, OutputDir, &strings.Builder{}); err != nil {
		t.Fatalf("the build refused: %v", err)
	}
	page := read(t, filepath.Join(root, "dist", "privacy", "index.html"))
	if strings.Contains(page, "<script>") {
		t.Errorf("the statement reached the page as markup:\n%s", page)
	}
	if !strings.Contains(page, "&lt;script&gt;") {
		t.Errorf("the statement did not reach the page as text:\n%s", page)
	}
}

// A tree with no privacy prose says so. The page is one file away from
// existing, and a build that passed over its absence would read like a build
// that had nothing to produce.
func TestBuildSaysWhenThereIsNoPrivacyProse(t *testing.T) {
	root := tree(t, "A title\n\nOne paragraph.\n")
	var log strings.Builder
	written, err := Build(root, OutputDir, &log)
	if err != nil {
		t.Fatalf("the build refused: %v\n%s", err, log.String())
	}
	if len(written) != 1 {
		t.Fatalf("the build wrote %q", written)
	}
	if !strings.Contains(log.String(), "no content/privacy.txt in the tree") {
		t.Errorf("the run passed over the absence:\n%s", log.String())
	}
}

// A source that is there and cannot be read stops the build. Rendering the part
// that parsed would produce a privacy page missing a statement, and a missing
// statement on that page reads as the absence of the thing it was about.
func TestBuildRefusesPrivacyProseItCannotRead(t *testing.T) {
	root := privacyTree(t, "Privacy\n\nchecked: No page fetches a script.\n")

	_, err := Build(root, OutputDir, &strings.Builder{})
	if err == nil {
		t.Fatal("the build accepted prose it could not read")
	}
	if !strings.Contains(err.Error(), "privacy") {
		t.Errorf("the refusal does not say which file it is about: %v", err)
	}
}
