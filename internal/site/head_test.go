// The suite over what a page says about itself before a reader opens it.
//
// None of this is visible on the page. A description that is missing, empty or
// the same as another page's renders an identical document, and what changes is a
// line in a search result and a card in a chat window, neither of which anybody
// looks at while writing the page. So each case is a mistake that leaves the page
// looking finished.
package site

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// headTemplate renders the three things this file decides and nothing else, so a
// case that reads a produced page is reading what the build put there rather than
// whatever the repository's frame happens to carry today.
const headTemplate = `<title>{{ .Title }}</title>
<meta name="description" content="{{ .Description }}" />
<link rel="canonical" href="{{ .Canonical }}" />
<meta property="og:site_name" content="{{ .SiteName }}" />
<meta property="og:title" content="{{ .Title }}" />
<meta property="og:description" content="{{ .Description }}" />
<h1>{{ .Title }}</h1>
{{- range .Paragraphs }}
<p>{{ . }}</p>
{{- end }}
{{- range .Onward }}
<a href="{{ .Href }}">{{ .Text }}</a>
{{- end }}
`

// headTree is a tree that produces all three pages the build writes today,
// through a frame that renders what this file decides.
func headTree(t *testing.T, index, privacy, notFound string) string {
	t.Helper()
	root := t.TempDir()
	mkdir(t, filepath.Join(root, TemplatesDir))
	mkdir(t, filepath.Join(root, ContentDir))
	write(t, filepath.Join(root, TemplatesDir, "page.html.tmpl"), headTemplate)
	write(t, filepath.Join(root, ContentDir, "index.txt"), index)
	write(t, filepath.Join(root, filepath.FromSlash(PrivacyFile)), privacy)
	write(t, filepath.Join(root, filepath.FromSlash(NotFoundFile)), notFound)
	return root
}

const (
	headIndex = `A title

description: What the landing page says.

One paragraph.
`
	headPrivacy = `Privacy

description: What happens to a request.

One paragraph.

checked: No page fetches a script. [page-fetches-no-script]

residual: The host sees the request.
`
	headNotFound = `This page is not here

description: The address you asked for is not one this site answers at.

One paragraph.

onward: The page this site starts at [/]
`
)

// The address a page states about itself is the one record 0008 gives it, read
// off where the build puts the file. A directory address is served by the index
// document inside it, so the page states the directory rather than the file, and
// a page that stated the file would send every reader who copied the address out
// of a search result to a second spelling of the same page.
func TestTheAddressAPageStatesIsTheOneTheRecordGivesIt(t *testing.T) {
	for produced, want := range map[string]string{
		"index.html":             "/",
		"privacy/index.html":     "/privacy/",
		"404.html":               "/404.html",
		"plugins/sso/index.html": "/plugins/sso/",
		"design-system.html":     "/design-system.html",
	} {
		if got := addressOf(produced); got != want {
			t.Errorf("%s is stated as %q, want %q", produced, got, want)
		}
	}
}

// Every produced page carries its own address, absolute and with the scheme a
// reader arrives over, because a canonical link that is not absolute says
// nothing a search engine can use.
func TestEveryProducedPageStatesItsOwnAddress(t *testing.T) {
	root := headTree(t, headIndex, headPrivacy, headNotFound)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused: %v", err)
	}
	for produced, want := range map[string]string{
		IndexPath:    Origin + "/",
		PrivacyPath:  Origin + "/privacy/",
		NotFoundPath: Origin + "/404.html",
	} {
		body := read(t, filepath.Join(root, OutputDir, filepath.FromSlash(produced)))
		if !strings.Contains(body, `<link rel="canonical" href="`+want+`" />`) {
			t.Errorf("%s does not state %q:\n%s", produced, want, body)
		}
	}
}

// The description comes out of the file the prose does, so changing one value
// changes one page. A description written into the frame would be one description
// for every page, which is worse than none: it makes every result look like every
// other one, and nothing about any of the pages looks wrong.
func TestChangingOneDescriptionChangesExactlyThatPage(t *testing.T) {
	root := headTree(t, headIndex, headPrivacy, headNotFound)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused: %v", err)
	}
	before := map[string]string{}
	for _, p := range []string{IndexPath, PrivacyPath, NotFoundPath} {
		before[p] = read(t, filepath.Join(root, OutputDir, filepath.FromSlash(p)))
	}

	write(t, filepath.Join(root, filepath.FromSlash(PrivacyFile)),
		strings.Replace(headPrivacy, "What happens to a request.", "Something else entirely.", 1))
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused after the value changed: %v", err)
	}

	after := read(t, filepath.Join(root, OutputDir, filepath.FromSlash(PrivacyPath)))
	if after == before[PrivacyPath] {
		t.Error("the privacy page did not change when its description did")
	}
	if !strings.Contains(after, "Something else entirely.") {
		t.Errorf("the new value did not reach the page:\n%s", after)
	}
	for _, p := range []string{IndexPath, NotFoundPath} {
		if read(t, filepath.Join(root, OutputDir, filepath.FromSlash(p))) != before[p] {
			t.Errorf("%s changed when another page's description did", p)
		}
	}
}

// The card a shared link renders carries the site's name as well as the page's,
// so a reader who sees three of them can tell they came from one place.
func TestThePreviewCarriesTheSiteName(t *testing.T) {
	root := headTree(t, headIndex, headPrivacy, headNotFound)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused: %v", err)
	}
	body := read(t, filepath.Join(root, OutputDir, IndexPath))
	for _, want := range []string{
		`<meta property="og:site_name" content="` + SiteName + `" />`,
		`<meta property="og:title" content="A title" />`,
		`<meta property="og:description" content="What the landing page says." />`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the page does not carry %s:\n%s", want, body)
		}
	}
}

// One sentence on two pages, which is what a description copied from the page
// beside it produces. Every page renders, every result reads plausibly, and a
// result for either page reads as a result for the other.
func TestTheBuildRefusesTwoPagesWithOneDescription(t *testing.T) {
	shared := strings.Replace(headPrivacy, "What happens to a request.", "What the landing page says.", 1)
	root := headTree(t, headIndex, shared, headNotFound)

	_, err := Build(root, OutputDir, io.Discard)
	if err == nil {
		t.Fatal("the build accepted one description on two pages")
	}
	for _, want := range []string{"/", "/privacy/", "What the landing page says."} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal reads %q, which does not name %q", err, want)
		}
	}
}

// A page with no description block is refused rather than produced with an empty
// element, and the refusal says which file to repair.
func TestTheBuildRefusesAPageThatSaysNothingAboutItself(t *testing.T) {
	for name, prose := range map[string]struct{ index, privacy, notFound string }{
		"the landing page":  {strings.Replace(headIndex, "description: What the landing page says.\n\n", "", 1), headPrivacy, headNotFound},
		"the privacy page":  {headIndex, strings.Replace(headPrivacy, "description: What happens to a request.\n\n", "", 1), headNotFound},
		"the not-found one": {headIndex, headPrivacy, strings.Replace(headNotFound, "description: The address you asked for is not one this site answers at.\n\n", "", 1)},
	} {
		root := headTree(t, prose.index, prose.privacy, prose.notFound)
		_, err := Build(root, OutputDir, io.Discard)
		if err == nil {
			t.Errorf("the build accepted %s with no description", name)
			continue
		}
		if !strings.Contains(err.Error(), descriptionKeyword) {
			t.Errorf("the refusal for %s reads %q and does not name the missing block", name, err)
		}
	}
}

// A block with the keyword and no sentence, which is what a value that did not
// arrive leaves behind. It is refused with the absent one, because the element it
// would produce reads in the markup as a description and is not one.
func TestTheBuildRefusesADescriptionWithNoSentence(t *testing.T) {
	root := headTree(t,
		strings.Replace(headIndex, "description: What the landing page says.", "description:", 1),
		headPrivacy, headNotFound)

	_, err := Build(root, OutputDir, io.Discard)
	if err == nil {
		t.Fatal("the build accepted a description block with no sentence")
	}
	if !strings.Contains(err.Error(), descriptionKeyword) {
		t.Errorf("the refusal reads %q and does not name the block", err)
	}
}

// What the build writes when a page is produced from a tree with no such file at
// all is nothing, and the run says so. This is here because the description is
// now a required part of every page's prose, and a run that skipped a page must
// still not read like a run that had nothing to skip.
func TestTheAbsentPagesAreStillReported(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, TemplatesDir))
	mkdir(t, filepath.Join(root, ContentDir))
	write(t, filepath.Join(root, TemplatesDir, "page.html.tmpl"), headTemplate)
	write(t, filepath.Join(root, ContentDir, "index.txt"), headIndex)

	var log strings.Builder
	if _, err := Build(root, OutputDir, &log); err != nil {
		t.Fatalf("the build refused a tree with one page in it: %v", err)
	}
	for _, want := range []string{PrivacyFile, NotFoundFile} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run said nothing about the absent %s:\n%s", want, log.String())
		}
	}
	if _, err := os.Stat(filepath.Join(root, OutputDir, NotFoundPath)); !os.IsNotExist(err) {
		t.Error("a page was written from a file that is not there")
	}
}
