// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the invariant table.
//
// Every row is proved twice: the smallest violation somebody would actually
// write reds exactly that row, and a tree that breaks nothing reds none of them.
// A guard that has never refused anything is a guard nobody has tested, and a
// guard that refuses everything is as useless as one that refuses nothing, so
// only the pair distinguishes them.
//
// The marker fixture is base64 in source. A test file carrying that literal
// would be a tracked file naming a tool that produced it, which is the thing the
// row exists to refuse, so the suite would fail on itself.
package invariant

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Flowfin/site/internal/budget"
	"github.com/Flowfin/site/internal/licence"
	"github.com/Flowfin/site/internal/releases"
	"github.com/Flowfin/site/internal/security"
	"github.com/Flowfin/site/internal/site"
	"github.com/Flowfin/site/internal/tokens"
	"github.com/Flowfin/site/internal/version"
)

// b64 decodes a fixture whose bytes are the point.
func b64(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("decoding the fixture: %v", err)
	}
	return b
}

// What the table is handed where a case is about the rows rather than about a
// tree. Two numbers rather than the five the real copy carries, and neither of
// them one of those five: a case asserting against the published limits would
// pass by reading the same file the row reads, which is the one thing this row
// cannot be allowed to prove about itself. One is a ceiling and one is an exact
// value, because the two are spelled differently and a fixture carrying only a
// ceiling would leave the other spelling untried.
var fixtureNumbers = []tokens.Number{
	{Name: "first-frame", Limit: "37", Unit: "ms", Comparison: "below"},
	{Name: "torn-frames", Limit: "4", Unit: "frames", Comparison: "equal"},
}

// The opening tag of the content, written once because most fixtures below put
// their markup next to it and a second spelling of it here would drift against
// the page silently: the replacement would find nothing, the fixture would be
// the clean page, and the row it exists for would report that nothing refused
// it.
const contentOpen = `<main id="content" tabindex="-1">`

const cleanPage = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta name="color-scheme" content="light dark" />
    <title>A title</title>
    <meta name="description" content="What this page is, in one sentence." />
  </head>
  <body>
    <a href="#content">Skip to the content</a>
    ` + contentOpen + `<h1>A title</h1></main>
    <footer>
      <p>
        Flowfin is not affiliated with the Jellyfin project. Other projects are
        named here to say what this software works with.
      </p>
      <p><a href="/legal/">Who publishes this site</a></p>
      <p><a href="https://example.invalid/notice">The intended-use notice</a></p>
    </footer>
  </body>
</html>
`

// A workflow that pins nothing of its own: the action reference carries the
// version in the comment an updater writes, and the fetched tool takes its
// version out of the file that declares it.
const cleanWorkflow = `name: Formatting
jobs:
  prettier:
    runs-on: ubuntu-latest
    timeout-minutes: 5
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
      - run: |
          version="$(jq -er '.[] | select(.id == "prettier") | .version' pins.json)"
          echo "PRETTIER_VERSION=${version}" >> "$GITHUB_ENV"
      - run: npx --yes "prettier@${PRETTIER_VERSION}" --check .
`

// Each row against the one-line mistake it exists for, and against a body that
// makes the same kind of mistake nowhere.
func TestEveryRowRefusesItsOwnViolationAndPassesTheNeighbour(t *testing.T) {
	violations := map[string][]byte{
		// The attribute deleted, which is what somebody tidying the
		// opening tag does.
		"page-declares-its-language": []byte(strings.Replace(cleanPage, `<html lang="en">`, `<html>`, 1)),
		// The element emptied rather than removed, which is what a
		// template producing a title from an absent value writes.
		"page-carries-a-title": []byte(strings.Replace(cleanPage, `<title>A title</title>`, `<title></title>`, 1)),
		// The declaration gone from the head, which is what a head
		// rewritten by hand carries and what this generator already
		// produced against served pages that declare it. Nothing on the
		// page looks wrong afterwards to anybody whose machine is set
		// the way the page happens to be drawn.
		"page-declares-the-schemes-it-supports": []byte(strings.Replace(cleanPage,
			`    <meta name="color-scheme" content="light dark" />`+"\n", "", 1)),
		// A stylesheet written the way one is written, with the motion in
		// the cascade and no query anywhere asking the reader whether
		// they want it. Nothing about the page looks wrong to anybody
		// who has not set the preference, which is why this is the
		// version of the mistake that ships.
		"page-motion-answers-the-reader": []byte(strings.Replace(cleanPage, `</head>`,
			"  <style>a { transition: color 120ms }</style>\n  </head>", 1)),
		// The footer dropped, which is what a second template or a page
		// written by hand looks like. The sentence lives in one file so
		// that no page can ship without it, and this is the row that
		// makes that a property of the pages rather than a habit.
		"page-carries-the-affiliation-notice": []byte(strings.Replace(cleanPage,
			"Flowfin is not affiliated with the Jellyfin project.", "", 1)),
		// The one line deleted from the top of the frame, which is the
		// whole of this mistake: the page looks identical, reads
		// identically and is one key press further from its own text for
		// every link the frame puts above it. What a reader reaches
		// first afterwards is the footer, from the bottom of the frame.
		"page-reaches-the-content-first": []byte(strings.Replace(cleanPage,
			`    <a href="#content">Skip to the content</a>`+"\n", "", 1)),
		// One element, of the shape a copied snippet arrives in.
		"page-fetches-no-script": []byte(strings.Replace(cleanPage, `</head>`,
			`  <script src="https://example.invalid/a.js"></script>`+"\n  </head>", 1)),
		// An image written the way somebody writes one, with the source
		// and the alternative text and nothing about how much room it
		// needs. It is the first thing anybody adds to a page and it is
		// the whole of the layout shift the budget puts at zero.
		"image-carries-its-own-dimensions": []byte(strings.Replace(cleanPage, contentOpen,
			contentOpen+`<img src="/icon.png" alt="The mark" />`, 1)),
		// The name of a row with one letter more than the row has, which
		// is what a page ends up citing after somebody renames a row and
		// repairs the page from memory. The sentence beside it goes on
		// saying a check refuses this, and no check does.
		"page-cites-only-checks-that-exist": []byte(strings.Replace(cleanPage, contentOpen,
			contentOpen+`<ul><li data-refused-by="page-fetches-no-scripts">No page fetches a script.</li></ul>`, 1)),
		// The element rendered from a value that did not arrive, which
		// leaves the attribute in place and empty. The page renders
		// identically and the row is the only thing between it and a
		// search result showing its address.
		"page-carries-a-description": []byte(strings.Replace(cleanPage,
			`content="What this page is, in one sentence."`, `content=""`, 1)),
		// A document grown past the limit, which is what a page does when
		// its prose keeps arriving and nobody measures. The padding is a
		// comment so that nothing else on the page changes with it, and it
		// is assembled rather than written out because a test source
		// carrying twenty kilobytes of anything is unreadable.
		"page-fits-the-markup-budget": []byte(strings.Replace(cleanPage, contentOpen,
			contentOpen+"<!--"+strings.Repeat("p", budget.HTMLBytes)+"-->", 1)),
		// A stylesheet that outgrew what inlining bought, which is what
		// happens to an inlined one: the request it replaced is cheap
		// again and nobody is watching the size.
		"page-fits-the-stylesheet-budget": []byte(strings.Replace(cleanPage, `</head>`,
			"  <style>"+strings.Repeat("a{color:red}", budget.InlineCSSBytes/12+1)+"</style>\n  </head>", 1)),
		// One face, of the shape a copied stylesheet arrives in, and
		// served from this site so that the row about a foreign domain
		// says nothing about it.
		"page-downloads-no-web-font": []byte(strings.Replace(cleanPage, `</head>`,
			"  <style>@font-face { font-family: Body; src: url(/body.woff2) }</style>\n  </head>", 1)),
		// One image more than the landing page may ask for, each carrying
		// everything else the page rows require, so this row is the only
		// one the fixture trips.
		"landing-page-asks-for-at-most-two-images": []byte(strings.Replace(cleanPage, contentOpen,
			contentOpen+strings.Repeat(`<img src="/a.png" alt="A picture of something" width="8" height="8" />`, budget.LandingImages+1), 1)),
		// The link taken out of the frame's footer, which is what
		// tidying a footer looks like. Every page still renders, and
		// whoever publishes the site is reachable from whichever pages
		// happened to keep the link.
		"page-links-the-legal-notice": []byte(strings.Replace(cleanPage,
			`      <p><a href="/legal/">Who publishes this site</a></p>`+"\n", "", 1)),
		// The leading slash left off, which is how anybody writes a link
		// to a sibling page and is correct from the page it was written
		// on. From the not-found document, served in answer to an address
		// of any depth, it points at whatever directory the reader
		// happened to ask for.
		"page-references-everything-from-the-site-root": []byte(strings.Replace(cleanPage, contentOpen,
			contentOpen+`<a href="privacy/">What happens to a request</a>`, 1)),
		// A note to the author that reached the output.
		"output-carries-no-unfinished-marker": []byte(strings.Replace(cleanPage, `<h1>A title</h1>`,
			`<h1>A title</h1><!-- TODO: the real heading -->`, 1)),
		// A stylesheet from somebody else's domain, which is the shape a
		// page picks up the moment anybody reaches for a font or an icon
		// set. It trips this row and no other, so a red run says which
		// repair it wants.
		"output-references-no-domain-outside-the-allowlist": []byte(strings.Replace(cleanPage, `</head>`,
			`  <link rel="stylesheet" href="https://cdn.example.invalid/a.css" />`+"\n  </head>", 1)),
		// The day the route was last confirmed left where it was for a
		// year, which is the whole failure: nobody edits a file that is
		// still there and still looks right.
		"output-carries-no-expired-reporting-route": []byte(
			"Contact: https://example.invalid/report\nExpires: 2020-01-01T00:00:00Z\n"),
		// One end tag left out, which is the mistake this row exists for
		// and the one a browser hides.
		"page-parses": []byte(strings.Replace(cleanPage, contentOpen+"<h1>A title</h1></main>",
			contentOpen+"<h1>A title</h1></section></main>", 1)),
		// The same identifier on two elements, which is what a template
		// rendering a name into an id does the moment two rows share one.
		"page-uses-no-identifier-twice": []byte(strings.Replace(cleanPage, contentOpen,
			contentOpen+`<p id="sso">One</p><p id="sso">Two</p>`, 1)),
		// The level under a heading chosen because it looked right rather
		// than because it was next.
		"page-skips-no-heading-level": []byte(strings.Replace(cleanPage, "<h1>A title</h1>",
			"<h1>A title</h1><h3>A section</h3>", 1)),
		// An image written with its source and its size and nothing for
		// anybody who cannot see it.
		"page-image-carries-alternative-text": []byte(strings.Replace(cleanPage, contentOpen,
			contentOpen+`<img src="/icon.png" width="32" height="32" />`, 1)),
		// A field somebody is asked to fill in with nothing saying what
		// goes in it.
		"page-names-every-control": []byte(strings.Replace(cleanPage, contentOpen,
			contentOpen+`<input type="search" />`, 1)),
		// A cookie written as a meta element, which needs nothing running
		// on the page: a host serves the document and a browser acts on it.
		// It is the one way a site whose scripting budget is zero bytes
		// still sets one.
		"page-touches-no-browser-storage": []byte(strings.Replace(cleanPage, `</head>`,
			`  <meta http-equiv="set-cookie" content="seen=1" />`+"\n  </head>", 1)),
		// The handler that arrives with a control somebody added. It carries
		// no source anywhere, so the row about a script element passes it,
		// and it runs in a reader's browser all the same.
		"page-carries-no-inline-handler": []byte(strings.Replace(cleanPage, contentOpen,
			contentOpen+`<button onclick="alert(1)">Somewhere</button>`, 1)),
		"tracked-text-names-no-tool": b64(t, "QSBub3RlIGFib3ZlLgpHZW5lcmF0ZWQgYnkgQ2hhdEdQVCBhbmQgbGVmdCBpbi4K"),
		// The version put back where it is convenient, which is what
		// somebody does who is adding a step and does not know the file
		// exists. It is one line, and it is the line that takes the pin
		// back out of the set anything watches.
		"workflow-step-carries-no-version-literal": []byte(strings.Replace(cleanWorkflow,
			`prettier@${PRETTIER_VERSION}`, `prettier@3.9.6`, 1)),
		// The colour typed into the markup that needs it, which is what
		// somebody does with the value already on the screen in front of
		// them. It is the second definition of a value published
		// elsewhere, and the page goes on rendering it perfectly after the
		// published one moves.
		"design-tokens-live-in-exactly-one-file": []byte(strings.Replace(cleanPage, `<body>`,
			`<body style="background: #121216">`, 1)),
		// The limit quoted in a sentence, which is how a number a client
		// is held to gets into prose: somebody explains what the software
		// promises and writes the figure down beside it. It is the same
		// second definition as the colour above and it looks like an
		// ordinary sentence afterwards, which is why nobody finds it.
		"client-budget-numbers-live-in-exactly-one-file": []byte(strings.Replace(cleanPage,
			`<h1>A title</h1>`,
			`<h1>A title</h1><p>A key press answers `+fixtureNumbers[0].Stated()+`.</p>`, 1)),
		// The sentence a document gains the day somebody wants a reader to
		// know which release they are looking at. It is assembled from the
		// constant rather than written out, because a test source carrying
		// the version a second time is what this row refuses and the suite
		// would fail the tree it judges.
		"version-lives-in-exactly-one-file": []byte(
			"The bundle to take is " + version.Number + ", and it is the one this page describes.\n"),
		// The sentence somebody writes the day a reader asks which server
		// the plugin needs. It is one clause in ordinary prose, it is
		// true when it is typed, and it goes on reading as true after the
		// release it was copied from has moved to the next generation.
		"build-input-carries-no-server-generation": []byte(strings.Replace(cleanPage,
			`<h1>A title</h1>`,
			`<h1>A title</h1><p>It needs Jellyfin 10.11 or newer.</p>`, 1)),
		// The six shapes the headless rule refuses, each in the smallest
		// test somebody would actually write. Base64 for the same reason
		// the marker fixture is: a test source carrying these literally is
		// the thing these rows refuse, so the suite would fail on itself.
		"test-opens-no-window":                     b64(t, "cGFja2FnZSBzYW1wbGUKCmltcG9ydCAoCgkidGVzdGluZyIKCgkiZ2l0aHViLmNvbS9jaHJvbWVkcC9jaHJvbWVkcCIKKQoKZnVuYyBUZXN0UmVuZGVyKHQgKnRlc3RpbmcuVCkgeyBfID0gY2hyb21lZHAuTmV3Q29udGV4dCB9Cg=="),
		"test-needs-no-display-server":             b64(t, "cGFja2FnZSBzYW1wbGUKCmltcG9ydCAoCgkib3MvZXhlYyIKCSJ0ZXN0aW5nIgopCgpmdW5jIFRlc3RSZW5kZXIodCAqdGVzdGluZy5UKSB7IF8gPSBleGVjLkNvbW1hbmQoInh2ZmItcnVuIiwgImdvIiwgInRlc3QiKSB9Cg=="),
		"test-binds-only-loopback":                 b64(t, "cGFja2FnZSBzYW1wbGUKCmltcG9ydCAoCgkibmV0IgoJInRlc3RpbmciCikKCmZ1bmMgVGVzdFNlcnZlKHQgKnRlc3RpbmcuVCkgeyBfLCBfID0gbmV0Lkxpc3RlbigidGNwIiwgIjAuMC4wLjA6ODA4MCIpIH0K"),
		"test-writes-no-certificate-store":         b64(t, "cGFja2FnZSBzYW1wbGUKCmltcG9ydCAoCgkib3MvZXhlYyIKCSJ0ZXN0aW5nIgopCgpmdW5jIFRlc3RUcnVzdCh0ICp0ZXN0aW5nLlQpIHsgXyA9IGV4ZWMuQ29tbWFuZCgiY2VydHV0aWwiLCAiLWFkZHN0b3JlIiwgInJvb3QiLCAiYS5jZXIiKSB9Cg=="),
		"test-asks-for-no-elevation":               b64(t, "cGFja2FnZSBzYW1wbGUKCmltcG9ydCAoCgkib3MvZXhlYyIKCSJ0ZXN0aW5nIgopCgpmdW5jIFRlc3RCaW5kKHQgKnRlc3RpbmcuVCkgeyBfID0gZXhlYy5Db21tYW5kKCJzdWRvIiwgInRydWUiKSB9Cg=="),
		"test-needs-nothing-outside-the-toolchain": b64(t, "cGFja2FnZSBzYW1wbGUKCmltcG9ydCAoCgkidGVzdGluZyIKCgkiZ2l0aHViLmNvbS9zdHJldGNoci90ZXN0aWZ5L3JlcXVpcmUiCikKCmZ1bmMgVGVzdEFzc2VydCh0ICp0ZXN0aW5nLlQpIHsgcmVxdWlyZS5UcnVlKHQsIHRydWUpIH0K"),
		// The file somebody adds. It opens with the paragraph saying what
		// it is for, which is what a reader of this tree learns to write,
		// and nothing about it looks unfinished afterwards. The header is
		// the one line a new file does not get by copying the shape of
		// the ones beside it.
		"source-carries-its-licence-header": []byte(sourceWithNoHeader),
	}

	// The neighbour a row is given is of the population it reads. A page rule
	// judged against a Go file, or the other way round, would pass for the
	// wrong reason.
	cleanTest := b64(t, "cGFja2FnZSBzYW1wbGUKCmltcG9ydCAoCgkib3MiCgkidGVzdGluZyIKKQoKZnVuYyBUZXN0U29tZXRoaW5nKHQgKnRlc3RpbmcuVCkgewoJaWYgb3MuR2V0ZW52KCJIT01FIikgPT0gIiIgewoJCXQuU2tpcCgibm8gaG9tZSIpCgl9Cn0K")

	rules := Rules(fixtureNumbers)
	if len(rules) != len(violations) {
		t.Fatalf("the table holds %d row(s) and this test carries %d violation(s); a row without one proves nothing",
			len(rules), len(violations))
	}

	for _, r := range rules {
		body, ok := violations[r.ID]
		if !ok {
			t.Errorf("no violation is written for row %s", r.ID)
			continue
		}
		if got := r.decide(body); len(got) == 0 {
			t.Errorf("row %s passed its own violation", r.ID)
		}
		neighbour := []byte(cleanPage)
		switch r.Subject {
		case TestSources:
			neighbour = cleanTest
		case Workflows:
			neighbour = []byte(cleanWorkflow)
		case SourceFiles:
			neighbour = []byte(cleanSource)
		}
		if got := r.decide(neighbour); len(got) != 0 {
			t.Errorf("row %s refused a %s that breaks nothing: %v", r.ID, r.Subject, got)
		}
	}
}

// The two source fixtures the licence row is judged against. They are ordinary
// literals rather than base64: what makes a fixture base64 in this suite is
// that the tree's own rows would refuse the source carrying it, and a header
// this repository wants on every file is the opposite case. The header is
// assembled from the constants rather than written out, so a repository that
// later publishes under something else does not leave a case here passing
// against the identifier it used to carry.
const (
	sourceWithNoHeader = "// What this file is for, in the paragraph a reader of this tree learns to write." + "\n" +
		"package sample\n\nfunc Do() {}\n"
	cleanSource = "// " + licence.Header + "\n" + "\n" +
		"// What this file is for, in the paragraph a reader of this tree learns to write." + "\n" +
		"package sample\n\nfunc Do() {}\n"
)

// The other half of the licence row, and the half a presence check misses. The
// header is there, it names a real identifier, and it is the wrong one: the
// bare form the platform reports and the deprecated one on the published list,
// and the `only` spelling, which is a different permission rather than a
// different way of writing the same one. The refusal has to carry what it
// found, because a message saying only that the header was wrong sends the next
// reader back to the file to see which of the three it was.
func TestTheLicenceRowRefusesAnIdentifierThisRepositoryDoesNotPublishUnder(t *testing.T) {
	for _, wrong := range []string{"AGPL-3.0", "AGPL-3.0-only", "MIT"} {
		body := []byte(strings.Replace(cleanSource, licence.Header, licence.Tag+" "+wrong, 1))
		got := decideLicenceHeader(body)
		if len(got) != 1 {
			t.Fatalf("the row reported %d violation(s) for a file declaring %s: %v", len(got), wrong, got)
		}
		if !strings.Contains(got[0], wrong) {
			t.Errorf("the refusal reads %q, which does not name the identifier it found", got[0])
		}
		if !strings.Contains(got[0], licence.Identifier) {
			t.Errorf("the refusal reads %q, which does not say what this repository publishes under", got[0])
		}
	}
}

// What the row leaves alone, and it is the direction that costs a reader if it
// is wrong. A row that refused a body of another kind would refuse every page
// the build wrote and every copy the tree carries, which is a row somebody
// switches off rather than repairs.
func TestTheLicenceRowLeavesWhatIsNotSourceAlone(t *testing.T) {
	for name, body := range map[string]string{
		"a produced page":   cleanPage,
		"a workflow":        cleanWorkflow,
		"a copy of a value": `{"surface":{"ground":{"dark":{"srgb":"#121216","alpha":1}}}}`,
		"a document":        "# A heading\n\nA paragraph carrying no header.\n",
	} {
		if got := decideLicenceHeader([]byte(body)); len(got) != 0 {
			t.Errorf("the row refused %s: %v", name, got)
		}
	}
}

// What this case can claim and what it cannot. Inside the population the row
// reads, a source file with no header reds this row and no other, and putting
// the header back clears it.
//
// The table-wide form the page fixtures above use is not available here, and
// the reason is worth stating rather than working around. Several rows refuse
// the absence of something every produced page carries, and a Go file carries
// none of it, so a source body reds them whatever its header says. That is a
// statement about applying a page row to a body that is not a page, which the
// run never does, rather than about this row. What keeps this row out of the
// opposite mistake - refusing bodies that are not source - is the case above,
// and that is the direction the run could actually reach.
func TestTheLicenceRowMovesOnTheHeaderAndNothingElse(t *testing.T) {
	var refused []string
	for _, r := range Rules(fixtureNumbers) {
		if r.Subject != SourceFiles {
			continue
		}
		if len(r.decide([]byte(sourceWithNoHeader))) > 0 {
			refused = append(refused, r.ID)
		}
	}
	if len(refused) != 1 || refused[0] != "source-carries-its-licence-header" {
		t.Errorf("the file with no header refused %v of the rows reading %s, want only source-carries-its-licence-header",
			refused, SourceFiles)
	}

	for _, r := range Rules(fixtureNumbers) {
		if r.Subject != SourceFiles {
			continue
		}
		if got := r.decide([]byte(cleanSource)); len(got) > 0 {
			t.Errorf("%s refused the same file carrying the header: %v", r.ID, got)
		}
	}
}

// Loopback is what the bind row permits, in the three spellings a test actually
// uses. A row that refused these would be a row somebody turns off.
func TestTheBindRowPermitsLoopback(t *testing.T) {
	// Base64 again, and this time the reason is the row directly above: a
	// probe written out as a literal is a listen address in a tracked test
	// source, and the row refuses its own suite.
	for name, encoded := range map[string]string{
		"127.0.0.1:0": "bmV0Lkxpc3RlbigidGNwIiwgIjEyNy4wLjAuMTowIik=",
		"localhost:0": "bmV0Lkxpc3RlbigidGNwIiwgImxvY2FsaG9zdDowIik=",
		"[::1]:0":     "bmV0Lkxpc3RlbigidGNwIiwgIls6OjFdOjAiKQ==",
	} {
		if got := decideBind(b64(t, encoded)); len(got) != 0 {
			t.Errorf("the bind row refused %s: %v", name, got)
		}
	}
	// An address with no host is every interface on the machine, which is
	// the shape that raises the prompt, and it is the one somebody writes
	// without meaning to.
	if got := decideBind(b64(t, "bmV0Lkxpc3RlbigidGNwIiwgIjo4MDgwIik=")); len(got) == 0 {
		t.Error("the bind row passed an address with no host, which is every interface on the machine")
	}
}

// What the version row leaves alone, and it is most of a workflow file. A row
// that refused these is a row somebody removes, and removing it is how a
// fetched version gets written back into a step.
func TestTheVersionRowLeavesTheUpdaterAndTheCommentsAlone(t *testing.T) {
	for name, line := range map[string]string{
		"an action pinned by commit with its version in the comment": `      - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0`,
		"a version named in a comment above a step":                  `        # The formatter moved to 3.10.0 upstream and the file was not updated.`,
		"a whole line of comment":                                    `# Run 2026-08-10 against 1.26.1, which is what this used to carry.`,
		"a number that is not a version":                             `    timeout-minutes: 10`,
		"a cron expression":                                          `    - cron: "23 5 * * 1"`,
		"a date":                                                     `        run: echo "taken 2026-08-10"`,
	} {
		if got := decideWorkflowVersions([]byte(line + "\n")); len(got) != 0 {
			t.Errorf("the version row refused %s: %v", name, got)
		}
	}
}

// The two shapes a fetched version actually arrives in, each named with the
// literal, because a message saying only that a version was written into a step
// leaves the next person reading the file line by line.
func TestTheVersionRowNamesWhatItRefused(t *testing.T) {
	for name, line := range map[string]string{
		"an environment value":         `          ZIZMOR_VERSION: "1.26.1"`,
		"a version inside the command": `        run: uvx --no-build "zizmor@1.26.1" --min-severity=low .`,
	} {
		got := decideWorkflowVersions([]byte(line + "\n"))
		if len(got) == 0 {
			t.Errorf("%s was not refused", name)
			continue
		}
		if !strings.Contains(got[0], "1.26.1") {
			t.Errorf("%s was refused without naming the version: %v", name, got)
		}
		if !strings.Contains(got[0], "pins.json") {
			t.Errorf("%s was refused without naming where the version belongs: %v", name, got)
		}
	}
}

// What the row about a second copy of the release version leaves alone. Every
// line here is a version-shaped number this tree legitimately carries, and a row
// that refused one of them would red a clean tree for a version that is not this
// repository's, which is how a row gets taken out.
func TestTheReleaseVersionRowTellsAnotherVersionApart(t *testing.T) {
	for name, line := range map[string]string{
		"a longer version spelled around this one": "the plugin published " + version.Number + ".1 as its first build",
		"a version that ends the same way":         "1" + version.Number + " is a server generation and not this",
		"a pinned tool":                            `    "version": "3.9.6",`,
		"the toolchain":                            "toolchain go1.26.5",
		"a date":                                   "Run 2026-08-12 against the default branch.",
	} {
		if got := decideSecondVersion([]byte(line + "\n")); len(got) != 0 {
			t.Errorf("the row refused %s: %v", name, got)
		}
	}
}

// Every group of values the token file carries, in the shape it reaches a build
// input in. The row is named for the file rather than for one of its groups, so
// a group it walks past is a value somebody may type next to the thing it
// describes with nothing to stop them, and the page that describes all four is
// the page where that happens.
//
// Each fixture is one line of a stylesheet somebody writes while looking at the
// published value, which is the whole of this mistake: it renders correctly, it
// keeps rendering correctly after the published value moves, and the only party
// who finds out is a client built from the file.
func TestTheTokenRowRefusesEveryShapeTheFileCarries(t *testing.T) {
	for name, fixture := range map[string]struct{ line, names string }{
		"a colour out of the surface group": {`  <style>body { background: #121216 }</style>`, "#121216"},
		"a type size":                       {`  <style>h1 { font-size: 56px }</style>`, "56px"},
		"a corner radius":                   {`  <style>.tile { border-radius: 12px }</style>`, "12px"},
		"the width a column stops at":       {`  <style>main { max-width: 1080px }</style>`, "1080px"},
		"a font stack":                      {`  <style>body { font-family: ui-sans-serif, sans-serif }</style>`, "font stack"},
		"a type weight":                     {`  <style>h1 { font-weight: 700 }</style>`, "weight"},
	} {
		got := decideTypedTokenValue([]byte(fixture.line + "\n"))
		if len(got) == 0 {
			t.Errorf("%s was not refused", name)
			continue
		}
		if !strings.Contains(got[0], fixture.names) {
			t.Errorf("%s was refused without naming %q: %v", name, fixture.names, got)
		}
		if !strings.Contains(got[0], tokens.File) {
			t.Errorf("%s was refused without naming where the value is read from: %v", name, got)
		}
	}
}

// What the row leaves alone, and every line here is one this tree carries or
// would carry. A row that refused one of them reds a clean build input for
// something that is not a token, which is how a row gets taken back out.
func TestTheTokenRowTellsAValueApartFromASentence(t *testing.T) {
	for name, line := range map[string]string{
		"the link the frame is built around":  `    <a href="#content">Skip to the content</a>`,
		"a figure with its unit spelled out":  "The published page is read from 35 cm and a television from 3 m.",
		"a word that ends in a unit":          "Nothing here is a problem anybody has to solve.",
		"a number in a sentence":              "Twelve plugins, and 1080 of the lines below are prose.",
		"a date":                              "Run 2026-08-13 against the default branch.",
		"a property that only ends in one":    `  <style>.tile { scroll-padding: var(--r) }</style>`,
		"a value taken from the token itself": `  <style>h1 { font-size: var(--display) }</style>`,
	} {
		if got := decideTypedTokenValue([]byte(line + "\n")); len(got) != 0 {
			t.Errorf("the row refused %s: %v", name, got)
		}
	}
}

// The copy at the end of a sentence, which is how a document actually writes a
// version, and the refusal has to say where the version belongs or the next
// person deletes the sentence instead of reading it from the one file.
func TestTheReleaseVersionRowNamesWhereTheVersionBelongs(t *testing.T) {
	got := decideSecondVersion([]byte("This bundle is " + version.Number + ".\n"))
	if len(got) == 0 {
		t.Fatal("the row passed the version written at the end of a sentence")
	}
	if !strings.Contains(got[0], version.Number) {
		t.Errorf("the refusal reads %q and does not name the version it found", got[0])
	}
	if !strings.Contains(got[0], version.SourceFile) {
		t.Errorf("the refusal reads %q and does not name the file the version is read from", got[0])
	}
}

// Both spellings a generation arrives in, and the refusal names the number so
// that a reader repairs the clause rather than deleting the paragraph around it.
func TestTheGenerationRowRefusesBothSpellingsAndNamesTheNumber(t *testing.T) {
	for _, c := range []struct {
		name string
		body string
		want string
	}{
		{"stated in a sentence", "Install it on Jellyfin 10.11 or later.\n", "10.11"},
		{"stated with words between", "It runs on a Jellyfin server of 12.0 and up.\n", "12.0"},
		{"declared under the manifest key", "  \"targetAbi\": \"10.11.0.0\",\n", "10.11.0.0"},
		{"declared with the key hyphenated", "target-abi: 12.0.0.0\n", "12.0.0.0"},
	} {
		got := decideTypedServerGeneration([]byte(c.body))
		if len(got) != 1 {
			t.Errorf("%s: the row reported %d violation(s): %v", c.name, len(got), got)
			continue
		}
		if !strings.Contains(got[0], c.want) {
			t.Errorf("%s: the refusal reads %q and does not name %s", c.name, got[0], c.want)
		}
		if !strings.Contains(got[0], "line 1") {
			t.Errorf("%s: the refusal reads %q and does not name the line", c.name, got[0])
		}
	}
}

// What the row has to walk past, and each of these is in the tree today. The
// notice names the server with no number anywhere near it, the token file states
// lengths as bare decimals, and a number that belongs to something else is not a
// generation because it is shaped like one. A row that refused any of the three
// would be a row somebody switches off, and then the clause it exists for is
// unguarded rather than loosened.
func TestTheGenerationRowLeavesANumberThatIsNotAGenerationAlone(t *testing.T) {
	for _, c := range []struct {
		name string
		body string
	}{
		{"the server named with no number", affiliationNotice + "\n"},
		{"a line height", `  "line-height": 1.5,` + "\n"},
		{"a version of something else", "Formatted with prettier 3.9.6.\n"},
		{"the number on the line above the name", "10.11\nJellyfin\n"},
		{"a whole clean page", cleanPage},
	} {
		if got := decideTypedServerGeneration([]byte(c.body)); len(got) != 0 {
			t.Errorf("%s: the row refused it: %v", c.name, got)
		}
	}
}

// The sentence reds this row and no other. A clause that also tripped the
// version row or the budget row would send the next reader to the wrong repair,
// and the repair here is to take the value from what was published rather than
// to move it to another file.
func TestATypedServerGenerationRedsExactlyOneRow(t *testing.T) {
	page := []byte(strings.Replace(cleanPage, `<h1>A title</h1>`,
		`<h1>A title</h1><p>It needs Jellyfin 10.11 or newer.</p>`, 1))

	var refused []string
	for _, r := range Rules(fixtureNumbers) {
		if len(r.decide(page)) > 0 {
			refused = append(refused, r.ID)
		}
	}
	if len(refused) != 1 || refused[0] != "build-input-carries-no-server-generation" {
		t.Errorf("the typed generation refused %v, want only build-input-carries-no-server-generation", refused)
	}
}

// The notice is one sentence in one file and a template wraps its prose, so the
// row reads the sentence rather than the line breaks around it. A row that
// decided on whitespace would go red the first time somebody reformatted the
// template, which is how a rule gets removed.
func TestTheNoticeRowReadsTheSentenceRatherThanTheLineBreaks(t *testing.T) {
	wrapped := strings.Replace(cleanPage,
		"Flowfin is not affiliated with the Jellyfin project.",
		"Flowfin is not\n        affiliated with\n        the Jellyfin project.", 1)
	if got := decideAffiliation([]byte(wrapped)); len(got) != 0 {
		t.Errorf("the notice row refused a page whose notice is wrapped: %v", got)
	}

	// A page carrying most of the sentence is a page carrying a different
	// sentence, and the row says which one it wanted.
	got := decideAffiliation([]byte(strings.Replace(cleanPage,
		"is not affiliated with the Jellyfin project", "is affiliated with the Jellyfin project", 1)))
	if len(got) == 0 {
		t.Fatal("the notice row passed a page saying the opposite of the notice")
	}
	if !strings.Contains(got[0], "Flowfin is not affiliated with the Jellyfin project.") {
		t.Errorf("the refusal reads %q, which does not say what the page has to carry", got[0])
	}
}

// The four ways the link at the top of a frame stops doing its job, each named
// with what the reader actually reaches. A message saying only that the order
// was wrong leaves somebody reading a produced page tag by tag to work out
// which of the four they have, and the repairs are different: one is a missing
// element, one is a renamed identifier, one is an element written above the
// link, and one is a link to somewhere else entirely.
func TestTheContentLinkRowNamesWhatAReaderReachesInstead(t *testing.T) {
	cases := map[string]struct {
		page  string
		names string
	}{
		// The identifier renamed on the content and not in the link,
		// which is the shape this rots into: both halves are still
		// there and the key press moves nobody.
		"a link to an identifier nothing carries": {
			page:  strings.Replace(cleanPage, contentOpen, `<main id="body" tabindex="-1">`, 1),
			names: "no element on this page answers to that identifier",
		},
		// The link reaching a heading rather than the landmark, which
		// is what somebody writes when the content has no identifier
		// and the first thing inside it does.
		"a link to something that is not the content": {
			page: strings.Replace(cleanPage, `<a href="#content">`, `<a href="#a-title">`, 1) +
				`<p id="a-title">A paragraph</p>`,
			names: "rather than the main element",
		},
		// A link written above it, which is how the order is lost
		// without anybody touching the link itself.
		"a link in front of it": {
			page:  strings.Replace(cleanPage, `<a href="#content">`, `<a href="/install">Install</a><a href="#content">`, 1),
			names: "a link to /install",
		},
		// A control in front of it. It is not a link at all, so the
		// message says what the element is rather than where it goes.
		"a control in front of it": {
			page:  strings.Replace(cleanPage, `<a href="#content">`, `<button>Search</button><a href="#content">`, 1),
			names: "a button element rather than a link to the content",
		},
		// A control a page only calls disabled. The word is in the class
		// rather than in the attribute, so a browser stops on it, and a
		// row reading the word anywhere in the tag would agree with the
		// class and disagree with the reader.
		"a control the page only calls disabled": {
			page:  strings.Replace(cleanPage, `<a href="#content">`, `<button class="disabled">Search</button><a href="#content">`, 1),
			names: "a button element rather than a link to the content",
		},
	}

	for name, c := range cases {
		got := decideContentFirst([]byte(c.page))
		if len(got) == 0 {
			t.Errorf("%s was not refused", name)
			continue
		}
		if !strings.Contains(got[0], c.names) {
			t.Errorf("%s was refused without saying %q: %s", name, c.names, got[0])
		}
	}
}

// A page offering nothing to focus is the same failure one step further on, and
// it is what the fixture in the table would produce if the frame carried no
// footer link either. It is refused for its own reason rather than passing
// because the walk found no first element to judge.
func TestTheContentLinkRowRefusesAPageWithNothingToFocus(t *testing.T) {
	page := strings.Replace(cleanPage, `    <a href="#content">Skip to the content</a>`+"\n", "", 1)
	page = strings.Replace(page, `<p><a href="https://example.invalid/notice">The intended-use notice</a></p>`, "", 1)
	page = strings.Replace(page, `<p><a href="/legal/">Who publishes this site</a></p>`, "", 1)

	got := decideContentFirst([]byte(page))
	if len(got) == 0 {
		t.Fatal("the row passed a page carrying nothing a keyboard reader can focus")
	}
	if !strings.Contains(got[0], "nothing to focus at all") {
		t.Errorf("the refusal reads %q, which does not say what the page is missing", got[0])
	}
}

// What is in front of the link and is not in the tab order. A row that refused
// these would go red on the first page that hides a field or writes a landmark
// a script can move focus to, and a row somebody has to switch off to do
// ordinary work is a row that gets switched off.
func TestTheContentLinkRowLeavesWhatIsOutOfTheOrderAlone(t *testing.T) {
	ahead := map[string]string{
		"an element taken out of the order": `<div tabindex="-1">A region</div>`,
		"a hidden field":                    `<input type="hidden" name="page" value="2" />`,
		"a control that cannot be used":     `<button disabled>Search</button>`,
		"a link with no address":            `<a>Nowhere</a>`,
	}

	for name, markup := range ahead {
		page := strings.Replace(cleanPage, `<a href="#content">`, markup+`<a href="#content">`, 1)
		if got := decideContentFirst([]byte(page)); len(got) != 0 {
			t.Errorf("the row refused a page with %s in front of the link: %v", name, got)
		}
	}
}

// What the citation row leaves alone, which is a page naming a row this gate
// actually decides. A row that refused a correct citation would be a row
// somebody removes, and removing it is how the names on the privacy page stop
// meaning anything. The name is taken out of the table rather than typed here,
// so this test cannot go on passing against a row that was renamed.
func TestTheCitationRowLeavesARealCheckAlone(t *testing.T) {
	real := Rules(fixtureNumbers)[0].ID
	page := []byte(strings.Replace(cleanPage, contentOpen,
		contentOpen+`<ul><li data-refused-by="`+real+`">A statement.</li></ul>`, 1))
	if got := decideCitedChecks(page); len(got) != 0 {
		t.Errorf("the row refused a page citing %s, which it decides: %v", real, got)
	}
}

// The empty citation, which is what a template renders from a value nobody
// supplied. It reads on the page as a name and is not one, so it is refused for
// its own reason rather than falling through the comparison against the table.
func TestTheCitationRowRefusesANameThatIsNotThere(t *testing.T) {
	page := []byte(strings.Replace(cleanPage, contentOpen,
		contentOpen+`<ul><li data-refused-by="">A statement.</li></ul>`, 1))
	got := decideCitedChecks(page)
	if len(got) != 1 {
		t.Fatalf("the row reported %d violation(s) for a citation naming nothing: %v", len(got), got)
	}
	if !strings.Contains(got[0], "no name") {
		t.Errorf("the message does not say what was wrong: %s", got[0])
	}
}

// A row this gate is owed and does not decide is not a check a page may cite.
// The two lists are separate for a reason the run prints on every line, and a
// page citing an owed row would be naming something the run itself reports as
// not decided.
func TestTheCitationRowRefusesAnOwedRow(t *testing.T) {
	owed := Owing()
	if len(owed) == 0 {
		t.Skip("nothing is owed today, so there is no owed name to cite")
	}
	page := []byte(strings.Replace(cleanPage, contentOpen,
		contentOpen+`<ul><li data-refused-by="`+owed[0].ID+`">A statement.</li></ul>`, 1))
	if got := decideCitedChecks(page); len(got) == 0 {
		t.Errorf("the row passed a page citing %s, which this gate does not decide", owed[0].ID)
	}
}

// A violation reds exactly one row. A fixture that trips two of them would make
// a red run unreadable, and a row that quietly also refuses its neighbour's case
// is a row whose message points at the wrong repair.
func TestAViolationRedsExactlyOneRow(t *testing.T) {
	noLang := []byte(strings.Replace(cleanPage, `<html lang="en">`, `<html>`, 1))

	var refused []string
	for _, r := range Rules(fixtureNumbers) {
		if len(r.decide(noLang)) > 0 {
			refused = append(refused, r.ID)
		}
	}
	if len(refused) != 1 || refused[0] != "page-declares-its-language" {
		t.Errorf("the missing language attribute refused %v, want only page-declares-its-language", refused)
	}
}

// The other half of the same row, and the half a presence check misses. The
// attribute is there, it holds a real tag, and it is the wrong one, which is
// what a page left behind in the language this site used to be written in looks
// like. The refusal has to carry the value, because a message saying only that
// something about the language was wrong sends the next reader back to the page
// to find out which of the three things it was.
func TestTheLanguageRowRefusesALanguageTheSiteDoesNotPublishIn(t *testing.T) {
	page := []byte(strings.Replace(cleanPage, `<html lang="en">`, `<html lang="de">`, 1))

	got := decideLang(page)
	if len(got) != 1 {
		t.Fatalf("the row reported %d violation(s) for a page declaring de: %v", len(got), got)
	}
	if !strings.Contains(got[0], `"de"`) {
		t.Errorf("the refusal reads %q, which does not name the value it found", got[0])
	}
	for _, l := range publishedLanguages {
		if !strings.Contains(got[0], l) {
			t.Errorf("the refusal reads %q, which does not say the site publishes in %s", got[0], l)
		}
	}
}

// What the row leaves alone, and it is the direction that costs a reader if it
// is wrong. A tag more specific about the same language is a page in that
// language, and a row refusing it would be a row somebody satisfies by writing
// the vaguest tag available. The case is spelled out of the declared set rather
// than typed, so a set that later publishes a second language does not leave
// this case passing against a language nobody publishes in.
func TestTheLanguageRowLeavesAMoreSpecificTagAlone(t *testing.T) {
	for _, l := range publishedLanguages {
		for name, tag := range map[string]string{
			"the tag itself":          l,
			"a region of it":          l + "-GB",
			"the tag in capitals":     strings.ToUpper(l),
			"a region of it, spaced ": " " + l + "-419 ",
		} {
			page := []byte(strings.Replace(cleanPage, `<html lang="en">`,
				`<html lang="`+tag+`">`, 1))
			if got := decideLang(page); len(got) != 0 {
				t.Errorf("the row refused %s (%q): %v", name, tag, got)
			}
		}
	}
}

// The wrong language reds this row and moves no other, in either direction. A
// page that is otherwise clean carries every other row's subject, so a fixture
// that tripped a second row would mean a red run naming a repair the page does
// not need.
func TestALanguageTheSiteDoesNotPublishInRedsExactlyOneRow(t *testing.T) {
	page := []byte(strings.Replace(cleanPage, `<html lang="en">`, `<html lang="de">`, 1))

	var refused []string
	for _, r := range Rules(fixtureNumbers) {
		if len(r.decide(page)) > 0 {
			refused = append(refused, r.ID)
		}
	}
	if len(refused) != 1 || refused[0] != "page-declares-its-language" {
		t.Errorf("a page declaring de refused %v, want only page-declares-its-language", refused)
	}

	// The same page with the published tag back, so this case says the row
	// moved on the value rather than on anything else the replacement did.
	clean := []byte(strings.Replace(cleanPage, `<html lang="en">`,
		`<html lang="`+publishedLanguages[0]+`">`, 1))
	for _, r := range Rules(fixtureNumbers) {
		if got := r.decide(clean); len(got) > 0 {
			t.Errorf("%s refused the same page carrying %s: %v", r.ID, publishedLanguages[0], got)
		}
	}
}

// The four shapes a reference to somebody else's domain arrives in, each named
// in #37 and each refused with the address written out. A message that said only
// that the page reached another origin would leave the next person grepping the
// output for which line to repair.
func TestTheOriginRowNamesEveryForeignReferenceItRefuses(t *testing.T) {
	for name, ref := range map[string]struct{ markup, want string }{
		"a stylesheet": {`<link rel="stylesheet" href="https://cdn.example.invalid/a.css" />`,
			"https://cdn.example.invalid/a.css"},
		"a font": {`<link rel="preload" as="font" href="https://fonts.example.invalid/f.woff2" />`,
			"https://fonts.example.invalid/f.woff2"},
		"an image": {`<img src="https://images.example.invalid/a.png" alt="" width="1" height="1" />`,
			"https://images.example.invalid/a.png"},
		"a script": {`<script src="https://cdn.example.invalid/a.js"></script>`,
			"https://cdn.example.invalid/a.js"},
		"a candidate in a set": {`<img src="/a.png" srcset="/a.png 1x, https://images.example.invalid/a2.png 2x" alt="" />`,
			"https://images.example.invalid/a2.png"},
		"a form action": {`<form action="https://forms.example.invalid/post"></form>`,
			"https://forms.example.invalid/post"},
		"a stylesheet reached without a scheme": {`<link rel="stylesheet" href="//cdn.example.invalid/a.css" />`,
			"//cdn.example.invalid/a.css"},
		"a background in a style block": {`<style>body { background: url("https://images.example.invalid/b.png"); }</style>`,
			"https://images.example.invalid/b.png"},
		"an imported stylesheet": {`<style>@import "https://cdn.example.invalid/b.css";</style>`,
			"https://cdn.example.invalid/b.css"},
	} {
		body := []byte(strings.Replace(cleanPage, contentOpen, ref.markup+contentOpen, 1))
		got := decideForeignOrigin(body)
		if len(got) == 0 {
			t.Errorf("%s was not refused", name)
			continue
		}
		if !strings.Contains(strings.Join(got, "\n"), ref.want) {
			t.Errorf("%s was refused without naming %s: %v", name, ref.want, got)
		}
		if !strings.Contains(strings.Join(got, "\n"), "example.invalid") {
			t.Errorf("%s was refused without naming the host: %v", name, got)
		}
	}
}

// The one exception, and the references that reach whatever served the page. A
// row that refused these is a row somebody switches off, and switching it off is
// how a font arrives.
func TestTheOriginRowLeavesALinkAndTheSiteItselfAlone(t *testing.T) {
	for name, markup := range map[string]string{
		"a link in running text":     `<p>See <a href="https://jellyfin.org/">the server</a>.</p>`,
		"a link to the project":      `<p><a href="https://flowfin.dev/install/">Install</a></p>`,
		"an absolute path":           `<link rel="stylesheet" href="/style.css" />`,
		"a relative path":            `<img src="a.png" alt="" width="1" height="1" />`,
		"the site's own stylesheet":  `<link rel="stylesheet" href="https://flowfin.dev/style.css" />`,
		"an address that is no host": `<p><a href="mailto:nobody@example.invalid">Write</a></p>`,
		"an inline image":            `<img src="data:image/gif;base64,R0lGODlhAQABAAAAACw=" alt="" width="1" height="1" />`,
		"a fragment":                 `<p><a href="#content">Skip</a></p>`,
	} {
		body := []byte(strings.Replace(cleanPage, contentOpen, markup+contentOpen, 1))
		if got := decideForeignOrigin(body); len(got) != 0 {
			t.Errorf("the origin row refused %s: %v", name, got)
		}
	}
}

// tree writes a buildable repository under a temporary root and puts it in a git
// index, because the tracked-file rule asks git what this repository carries
// rather than walking whatever is sitting in the directory.
func tree(t *testing.T, template string) string {
	t.Helper()

	root := t.TempDir()
	mk := func(dir string) {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("preparing %s: %v", dir, err)
		}
	}
	wr := func(name, body string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	mk("templates")
	mk("content")
	mk(filepath.Dir(filepath.FromSlash(tokens.File)))
	wr(filepath.Join("templates", "page.html.tmpl"), template)
	// The landing page's prose carries no paragraph of its own. The one
	// paragraph on it is the sentence the build composes from the claim
	// about the clients written below, so a fixture that lost that claim
	// produces a landing page with nothing on it rather than one that merely
	// says less.
	wr(filepath.Join("content", "index.txt"),
		"A title\n\ndescription: What the first fixture page is.\n")
	// The second page, and it is here for the frame rather than for
	// anything the privacy register decides. A property of the one file
	// every page is rendered through is a statement about all of them, and a
	// fixture producing a single page cannot tell that apart from a property
	// of that page. The name behind its one checked statement is read out of
	// the table rather than typed, because a page citing a check nothing
	// answers to is refused, and a fixture holding the name would red this
	// whole suite the day a row is renamed.
	wr(filepath.FromSlash(site.PrivacyFile),
		"A second title\n\ndescription: What the second fixture page is.\n\nOne paragraph.\n\n"+
			"checked: One statement. ["+Rules(fixtureNumbers)[0].ID+"]\n\n"+
			"residual: What a host sees is true whatever this site does.\n")
	// The copy the build reads. It carries a colour, because the row about
	// where a colour is read from is about there being one file that may
	// carry one, and a fixture whose copy held none would prove nothing
	// about which file that is. It carries the client budget for the same
	// reason one row further on, and the numbers are the ones the cases
	// above are written against rather than a second set: what the run
	// refuses and what a row refuses have to be the same values, or a case
	// that passed here would say nothing about the run.
	wr(filepath.FromSlash(tokens.File),
		`{"surface":{"ground":{"dark":{"srgb":"#121216","alpha":1}}},"budget":{"numbers":{`+
			strings.Join(asJSON(fixtureNumbers), ",")+`}}}`)
	// The source of the produced reporting route. The day is far enough
	// ahead that the fixture does not expire while nobody is looking at it,
	// which is the one thing in this tree that would go red on a date rather
	// than on a change.
	wr(filepath.FromSlash(security.File), `{"route":"https://example.invalid/report",
		"policy":"https://example.invalid/policy","confirmed":"2099-01-01"}`)
	// What the landing page says about the clients. It is a value rather
	// than a paragraph, so a fixture with no such file is a fixture whose
	// landing page has quietly stopped making the statement, which is what
	// the refusal below is about.
	wr(filepath.FromSlash(site.ClientsFile), `{"intent":"a client per platform","availability":"none-released"}`)
	// The roster and the record that answers whether its one repository is
	// there. One row rather than twelve: what this tree is for is deciding
	// the rows against a page, and the rows the roster produces are the
	// build suite's subject rather than this one's.
	wr(filepath.FromSlash(site.RosterFile),
		`[{"id":"alpha","repository":"Flowfin/jellyfin-plugin-alpha",`+
			`"summary":"What alpha does","state":"build-up"}]`)
	wr(filepath.FromSlash(releases.File),
		`{"taken":"2026-01-02","command":"a command","repositories":`+
			`{"Flowfin/jellyfin-plugin-alpha":{"finished":0,"prereleases":0}}}`)

	git(t, root, "init", "-q")
	git(t, root, "add", "-A")
	return root
}

// producedPages is how many pages a build of root writes.
//
// The cases about a property lost in the frame assert that every produced page
// is named in the refusal, and that is a count of what the fixture tree happens
// to produce. A number typed into those cases is a number the fixture tree
// moves: it moved the day the tree gained a roster, and seven cases that had
// nothing to do with rosters went red naming a repair none of them needed.
func producedPages(t *testing.T, root string) int {
	t.Helper()
	written, err := site.Build(root, filepath.Join(t.TempDir(), site.OutputDir), io.Discard)
	if err != nil {
		t.Fatalf("building the fixture tree to count its pages: %v", err)
	}
	n := 0
	for _, w := range written {
		if strings.HasSuffix(w, ".html") {
			n++
		}
	}
	return n
}

// asJSON writes the fixture numbers the way the token file carries them, so a
// tree and the cases about a single row are held to one set of values rather
// than to two that agree today.
func asJSON(numbers []tokens.Number) []string {
	var out []string
	for _, n := range numbers {
		out = append(out, fmt.Sprintf(`%q:{"limit":%s,"unit":%q,"comparison":%q}`,
			n.Name, n.Limit, n.Unit, n.Comparison))
	}
	return out
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// A temporary repository, so nothing here reads or writes the caller's
	// configuration, and no hook can run in it.
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=", "GIT_CONFIG_SYSTEM=")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

const goodTemplate = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="color-scheme" content="light dark" />
    <title>{{ .Title }}</title>
    <meta name="description" content="{{ .Description }}" />
  </head>
  <body>
    <a href="#content">Skip to the content</a>
    ` + contentOpen + `
      <h1>{{ .Title }}</h1>
      {{- range .Paragraphs }}
      <p>{{ . }}</p>
      {{- end }}
    </main>
    <footer>
      <p>
        Flowfin is not affiliated with the Jellyfin project. Other projects are
        named here to say what this software works with.
      </p>
      <p><a href="/legal/">Who publishes this site</a></p>
    </footer>
  </body>
</html>
`

// The whole run over a tree that breaks nothing. This is the neighbour for the
// run itself rather than for one row.
func TestRunAcceptsATreeThatBreaksNoRule(t *testing.T) {
	var log bytes.Buffer
	if err := Run(tree(t, goodTemplate), &log); err != nil {
		t.Fatalf("Run refused a tree that breaks nothing: %v\n%s", err, log.String())
	}
	if !strings.Contains(log.String(), "page-declares-its-language: ok") {
		t.Errorf("the run did not report the row as examined; it said:\n%s", log.String())
	}
}

// The one-character version of the mistake, in the template rather than in a
// page, because a template is where a page property is actually lost.
func TestRunRefusesATemplateThatDroppedTheLanguage(t *testing.T) {
	root := tree(t, strings.Replace(goodTemplate, `<html lang="en">`, `<html>`, 1))

	var log bytes.Buffer
	err := Run(root, &log)
	if err == nil {
		t.Fatalf("Run accepted a template with no language attribute:\n%s", log.String())
	}
	for _, want := range []string{
		"page-declares-its-language: REFUSED",
		"dist/index.html: the html element carries no lang attribute",
		"it refuses",
		"because",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q; it said:\n%s", want, log.String())
		}
	}
	if !strings.Contains(err.Error(), "1 rule(s) refused") {
		t.Errorf("the error reads %q, which does not say how many rules refused", err)
	}
}

// The same mistake made in the value rather than in the attribute, which is the
// shape a page left behind in another language arrives in. It is put in the
// template because that is where a page property is lost, and every page the
// build produced is named rather than the one somebody happened to open.
func TestRunRefusesATemplateDeclaringALanguageTheSiteDoesNotPublishIn(t *testing.T) {
	root := tree(t, strings.Replace(goodTemplate, `<html lang="en">`, `<html lang="de">`, 1))

	var log bytes.Buffer
	err := Run(root, &log)
	if err == nil {
		t.Fatalf("Run accepted a template declaring a language this site does not publish in:\n%s", log.String())
	}
	for _, want := range []string{
		fmt.Sprintf("page-declares-its-language: REFUSED, %d violation(s)", producedPages(t, root)),
		`dist/index.html: the html element declares lang="de"`,
		`dist/privacy/index.html: the html element declares lang="de"`,
		"it refuses",
		"because",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q; it said:\n%s", want, log.String())
		}
	}
	if !strings.Contains(err.Error(), "1 rule(s) refused") {
		t.Errorf("the error reads %q, which does not say how many rules refused", err)
	}
}

// The declaration lives in the head every page is rendered through, so losing
// it there loses it from every page at once. The run is what shows that, and
// what makes it a statement about the frame is that both produced pages are
// named rather than the one somebody happened to open.
func TestRunRefusesATemplateThatDroppedTheSchemeOnEveryPageItProduced(t *testing.T) {
	root := tree(t, strings.Replace(goodTemplate,
		`    <meta name="color-scheme" content="light dark" />`+"\n", "", 1))

	var log bytes.Buffer
	if err := Run(root, &log); err == nil {
		t.Fatalf("Run accepted a template declaring no colour scheme:\n%s", log.String())
	}
	for _, want := range []string{
		fmt.Sprintf("page-declares-the-schemes-it-supports: REFUSED, %d violation(s)", producedPages(t, root)),
		"dist/index.html: this page declares no colour scheme",
		"dist/privacy/index.html: this page declares no colour scheme",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q; it said:\n%s", want, log.String())
		}
	}
}

// The notice is in the frame so that no page can ship without it, and the row
// over it already refuses a page that dropped the sentence. This is the other
// statement: the sentence removed from the one file every page goes through
// reds every produced page rather than one of them, which is what catches the
// frame being bypassed rather than the sentence being edited.
func TestRunRefusesAFrameThatDroppedTheAffiliationNoticeOnEveryPageItProduced(t *testing.T) {
	root := tree(t, strings.Replace(goodTemplate,
		"Flowfin is not affiliated with the Jellyfin project.", "", 1))

	var log bytes.Buffer
	if err := Run(root, &log); err == nil {
		t.Fatalf("Run accepted a frame carrying no affiliation notice:\n%s", log.String())
	}
	for _, want := range []string{
		fmt.Sprintf("page-carries-the-affiliation-notice: REFUSED, %d violation(s)", producedPages(t, root)),
		"dist/index.html: this page carries no affiliation notice",
		"dist/privacy/index.html: this page carries no affiliation notice",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q; it said:\n%s", want, log.String())
		}
	}
}

// The whole run over a tree whose frame quotes a limit the copy is the
// authority for. It is the shape somebody writes when they are explaining what
// the software promises with the number in front of them, and the run names the
// input it was typed into rather than the pages it came out on, because the
// input is where the repair is.
func TestRunRefusesALimitTypedIntoWhatTheBuildReads(t *testing.T) {
	root := tree(t, strings.Replace(goodTemplate, `      <h1>{{ .Title }}</h1>`,
		"      <h1>{{ .Title }}</h1>\n      <p>A key press answers "+fixtureNumbers[0].Stated()+".</p>", 1))

	var log bytes.Buffer
	if err := Run(root, &log); err == nil {
		t.Fatalf("Run accepted a build input quoting a client budget limit:\n%s", log.String())
	}
	for _, want := range []string{
		"client-budget-numbers-live-in-exactly-one-file: REFUSED, 1 violation(s)",
		"templates/page.html.tmpl: line 13",
		`"under 37 ms"`,
		tokens.File,
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q; it said:\n%s", want, log.String())
		}
	}
}

// The same tree with the sentence taken back out. Without this the case above
// would pass over a run that refuses every tree, which proves the opposite of
// what it is for.
func TestRunAcceptsAFrameThatQuotesNoLimit(t *testing.T) {
	var log bytes.Buffer
	if err := Run(tree(t, goodTemplate), &log); err != nil {
		t.Fatalf("Run refused a tree quoting no limit: %v\n%s", err, log.String())
	}
	if !strings.Contains(log.String(), "client-budget-numbers-live-in-exactly-one-file: ok") {
		t.Errorf("the run did not report the row as examined; it said:\n%s", log.String())
	}
}

// A copy carrying no client budget number is refused rather than decided
// against. The row would report ok having compared nothing, which reads as a
// tree holding no second copy of a limit and means that nobody said what the
// limits are.
func TestRunRefusesACopyThatCarriesNoClientBudget(t *testing.T) {
	root := tree(t, goodTemplate)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(tokens.File)),
		[]byte(`{"surface":{"ground":{"dark":{"srgb":"#121216","alpha":1}}}}`), 0o644); err != nil {
		t.Fatalf("rewriting the copy: %v", err)
	}
	git(t, root, "add", "-A")

	var log bytes.Buffer
	err := Run(root, &log)
	if err == nil {
		t.Fatalf("Run decided the table against a copy carrying no client budget:\n%s", log.String())
	}
	if !strings.Contains(err.Error(), "carries no client budget number") {
		t.Errorf("the refusal reads %q, which does not say what was missing", err)
	}
	if strings.Contains(log.String(), "client-budget-numbers-live-in-exactly-one-file: ok") {
		t.Errorf("the run reported the row as examined against a copy with nothing in it:\n%s", log.String())
	}
}

// A number the copy carries with no unit beside it is refused before any row is
// decided, and the refusal names the number rather than the file alone. A limit
// with no unit states nothing, so a row comparing against it would be comparing
// against a bare figure and would refuse every line that happened to carry it.
func TestRunRefusesACopyWhoseLimitCarriesNoUnit(t *testing.T) {
	root := tree(t, goodTemplate)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(tokens.File)),
		[]byte(`{"budget":{"numbers":{"first-frame":{"limit":37,"comparison":"below"}}}}`), 0o644); err != nil {
		t.Fatalf("rewriting the copy: %v", err)
	}
	git(t, root, "add", "-A")

	var log bytes.Buffer
	err := Run(root, &log)
	if err == nil {
		t.Fatalf("Run decided the table against a limit that states nothing:\n%s", log.String())
	}
	if !strings.Contains(err.Error(), "budget.numbers.first-frame") {
		t.Errorf("the refusal reads %q, which does not name the number", err)
	}
}

// The link to the content is the first thing in the frame, so the one line
// deleted takes it off every page at once. The row over a page says that page
// is wrong; this says the frame is, and the two are told apart by both produced
// pages being named rather than the one somebody happened to open.
func TestRunRefusesAFrameThatDroppedTheContentLinkOnEveryPageItProduced(t *testing.T) {
	root := tree(t, strings.Replace(goodTemplate,
		`    <a href="#content">Skip to the content</a>`+"\n", "", 1))

	var log bytes.Buffer
	if err := Run(root, &log); err == nil {
		t.Fatalf("Run accepted a frame carrying no link to the content:\n%s", log.String())
	}
	for _, want := range []string{
		fmt.Sprintf("page-reaches-the-content-first: REFUSED, %d violation(s)", producedPages(t, root)),
		"dist/index.html: line 19: the first thing a keyboard reader reaches is a link to /legal/",
		"dist/privacy/index.html: line 19: the first thing a keyboard reader reaches is a link to /legal/",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q; it said:\n%s", want, log.String())
		}
	}
}

// A page added to the build brings no frame of its own, which is the last of
// the frame's properties and the cheapest to lose. The fixture writes a second
// source and nothing else, and the run over it decides the head rules on two
// pages rather than on one.
func TestASecondPageIsRenderedThroughTheSameFrame(t *testing.T) {
	root := tree(t, goodTemplate)

	var log bytes.Buffer
	if err := Run(root, &log); err != nil {
		t.Fatalf("Run refused a tree that breaks nothing: %v\n%s", err, log.String())
	}
	pages := producedPages(t, root)
	if pages < 2 {
		t.Fatalf("the fixture tree produces %d page(s), and a frame property over one page is a property of that page", pages)
	}
	for _, want := range []string{
		fmt.Sprintf("page-declares-its-language: ok, %d file(s)", pages),
		fmt.Sprintf("page-carries-a-title: ok, %d file(s)", pages),
		fmt.Sprintf("page-declares-the-schemes-it-supports: ok, %d file(s)", pages),
		fmt.Sprintf("page-carries-the-affiliation-notice: ok, %d file(s)", pages),
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q; it said:\n%s", want, log.String())
		}
	}
}

// The rules with nothing in the tree to read are printed by every run, passing
// or not. A run listing only what it could decide reads as a tree that satisfies
// the rest.
func TestRunNamesWhatItCouldNotDecide(t *testing.T) {
	var log bytes.Buffer
	if err := Run(tree(t, goodTemplate), &log); err != nil {
		t.Fatalf("Run: %v", err)
	}
	owing := Owing()
	if len(owing) == 0 {
		t.Fatal("nothing is recorded as owed, so this test proves nothing")
	}
	for _, o := range owing {
		if !strings.Contains(log.String(), o.ID+": not decided, waiting on") {
			t.Errorf("the run does not name %s as owed; it said:\n%s", o.ID, log.String())
		}
	}
}

// A row that judges a thing a page may carry, over pages carrying none of it,
// prints what it read rather than ok. Nothing this build writes moves, so the
// motion row is the row in that state, and an ok beside it would read as a set
// of pages that answer the reader when it is a set that never asks.
func TestRunSaysARowFoundNothingToDecide(t *testing.T) {
	root := tree(t, goodTemplate)

	var log bytes.Buffer
	if err := Run(root, &log); err != nil {
		t.Fatalf("Run refused a tree that breaks nothing: %v\n%s", err, log.String())
	}
	want := fmt.Sprintf("page-motion-answers-the-reader: %d file(s) of every page the build produced "+
		"carried no declarations that move something, so this rule decided nothing", producedPages(t, root))
	if !strings.Contains(log.String(), want) {
		t.Errorf("the run does not say the row decided nothing; it said:\n%s", log.String())
	}
	if strings.Contains(log.String(), "page-motion-answers-the-reader: ok") {
		t.Errorf("the run reported the row as ok over pages carrying nothing it judges; it said:\n%s", log.String())
	}
}

// The other side of the same line, which is what makes it a statement about the
// pages rather than a sentence the row always prints. A frame that moves
// something and answers for it reports the count it read.
func TestRunCountsWhatAMovingPageCarried(t *testing.T) {
	answered := strings.Replace(goodTemplate, "  </head>",
		"    <style>\n"+
			"      a { transition: color 120ms }\n"+
			"      @media (prefers-reduced-motion: reduce) { a { transition: none } }\n"+
			"    </style>\n  </head>", 1)

	root := tree(t, answered)

	var log bytes.Buffer
	if err := Run(root, &log); err != nil {
		t.Fatalf("Run refused a tree whose motion is answered for: %v\n%s", err, log.String())
	}
	// One declaration per page, because the frame carries one and every page
	// is rendered through it, which is what makes the count a reading of the
	// pages rather than of the file they came from.
	pages := producedPages(t, root)
	want := fmt.Sprintf("page-motion-answers-the-reader: ok, %d declarations that move something "+
		"in %d file(s) of every page the build produced", pages, pages)
	if !strings.Contains(log.String(), want) {
		t.Errorf("the run does not report what the row read; it said:\n%s", log.String())
	}
}

// The mistake in the place it is actually made. The frame is one file and every
// page is rendered through it, so a stylesheet written there with no query in it
// takes the answer away from the reader on every page at once, and the run names
// each of them rather than one.
func TestRunRefusesAFrameThatMovesSomethingWithNothingAskingTheReader(t *testing.T) {
	root := tree(t, strings.Replace(goodTemplate, "  </head>",
		"    <style>a { transition: color 120ms }</style>\n  </head>", 1))

	var log bytes.Buffer
	err := Run(root, &log)
	if err == nil {
		t.Fatalf("Run passed a frame that moves something with nothing asking the reader:\n%s", log.String())
	}
	for _, want := range []string{
		fmt.Sprintf("page-motion-answers-the-reader: REFUSED, %d violation(s)", producedPages(t, root)),
		"index.html: line",
		filepath.ToSlash(filepath.Join("privacy", "index.html")) + ": line",
		"no query asking the reader about motion switches transition off",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not carry %q; it said:\n%s", want, log.String())
		}
	}
}

// A tree whose build refuses has no output, and a rule about produced pages
// cannot pass over nothing. The run says which of the two happened.
func TestRunRefusesATreeItCannotBuild(t *testing.T) {
	root := tree(t, goodTemplate)
	if err := os.Remove(filepath.Join(root, "templates", "page.html.tmpl")); err != nil {
		t.Fatalf("removing the template: %v", err)
	}

	err := Run(root, io.Discard)
	if err == nil {
		t.Fatal("Run passed a tree that does not build")
	}
	if !strings.Contains(err.Error(), "the build refused") {
		t.Errorf("the error reads %q, which does not say the build was the problem", err)
	}
}

// The image row against the shapes it has to tell apart, and against the ones it
// may not touch. A row that refused an image carrying its size would be a row
// somebody removes the first time they add a picture.
func TestTheImageRowReadsTheValueRatherThanTheAttributeName(t *testing.T) {
	for name, c := range map[string]struct {
		markup  string
		refused bool
		names   string
	}{
		"both dimensions, written out": {
			`<img src="/icon.png" alt="The mark" width="64" height="64" />`, false, ""},
		"no width": {
			`<img src="/icon.png" alt="The mark" height="64" />`, true, "carries no width attribute"},
		"no height": {
			`<img src="/icon.png" alt="The mark" width="64" />`, true, "carries no height attribute"},
		// The one a rule reading for the name of the attribute passes:
		// the template wrote the attribute and the value behind it was
		// not there. A grep finds the word and a browser reserves
		// nothing.
		"an attribute the template left empty": {
			`<img src="/icon.png" alt="The mark" width="" height="64" />`, true, `carries the width ""`},
		// The same mistake one step along, where somebody writes the
		// unit a stylesheet takes on an attribute that does not take
		// one.
		"a unit on the attribute": {
			`<img src="/icon.png" alt="The mark" width="64px" height="64" />`, true, `carries the width "64px"`},
		"an image with no source at all": {
			`<img alt="The mark" />`, true, "with no source on it"},
	} {
		body := []byte(strings.Replace(cleanPage, contentOpen, contentOpen+c.markup, 1))
		got := decideImageDimensions(body)
		switch {
		case c.refused && len(got) == 0:
			t.Errorf("%s was not refused", name)
		case !c.refused && len(got) != 0:
			t.Errorf("%s was refused: %v", name, got)
		case c.refused && !strings.Contains(strings.Join(got, "\n"), c.names):
			t.Errorf("%s was refused without saying %q: %v", name, c.names, got)
		}
	}
}

// The failure names the page, the file and the attribute. A message saying only
// that something about the markup was wrong leaves the next person opening every
// produced page to find which image it meant.
func TestTheImageRowNamesTheFileAndBothMissingAttributes(t *testing.T) {
	body := []byte(strings.Replace(cleanPage, contentOpen,
		contentOpen+`<img src="/assets/icon.png" alt="The mark" />`, 1))

	got := strings.Join(decideImageDimensions(body), "\n")
	for _, want := range []string{"/assets/icon.png", "no width attribute", "no height attribute"} {
		if !strings.Contains(got, want) {
			t.Errorf("the refusal does not name %q: %s", want, got)
		}
	}
}

// An image with nothing wrong but its dimensions reds this row and no other, so
// a red run says which repair it wants rather than which area to look in.
func TestAnImageWithoutItsDimensionsRedsExactlyOneRow(t *testing.T) {
	body := []byte(strings.Replace(cleanPage, contentOpen,
		contentOpen+`<img src="/icon.png" alt="The mark" />`, 1))

	var refused []string
	for _, r := range Rules(fixtureNumbers) {
		if len(r.decide(body)) > 0 {
			refused = append(refused, r.ID)
		}
	}
	if len(refused) != 1 || refused[0] != "image-carries-its-own-dimensions" {
		t.Errorf("the image refused %v, want only image-carries-its-own-dimensions", refused)
	}
}

// The scheme row against the shapes it has to tell apart. Two of these are the
// reason it reads the value rather than the name: a declaration a template
// wrote against nothing, and one naming a word no browser acts on, both of
// which a rule searching for the attribute would pass.
func TestTheSchemeRowReadsTheValueRatherThanTheAttribute(t *testing.T) {
	for name, c := range map[string]struct {
		head    string
		refused bool
		names   string
	}{
		"both schemes on the element": {
			`<meta name="color-scheme" content="light dark" />`, false, ""},
		"one scheme on the element": {
			`<meta name="color-scheme" content="dark" />`, false, ""},
		"the scheme forced to one": {
			`<meta name="color-scheme" content="only light" />`, false, ""},
		"the answer given in a stylesheet": {
			`<style>:root { color-scheme: light dark }</style>`, false, ""},
		"an unknown identifier beside a real one": {
			`<meta name="color-scheme" content="ligth dark" />`, false, ""},
		// What a template renders from a value that was not supplied.
		// The word is on the page and a browser reads nothing.
		"the value the template left empty": {
			`<meta name="color-scheme" content="" />`, true, "no value on it"},
		// The word somebody reaches for from another setting, which
		// declares nothing here.
		"a word no browser acts on": {
			`<meta name="color-scheme" content="auto" />`, true, `"auto"`},
		// The question asked and never answered. A page may carry the
		// query and still owe the declaration, so reading the query as
		// one would let the row pass a page that says nothing.
		"the reader's preference asked about": {
			`<style>@media (prefers-color-scheme: dark) { body { color: inherit } }</style>`,
			true, "declares no colour scheme"},
	} {
		body := []byte(strings.Replace(cleanPage,
			`    <meta name="color-scheme" content="light dark" />`+"\n", "    "+c.head+"\n", 1))
		got := decideColourScheme(body)
		switch {
		case c.refused && len(got) == 0:
			t.Errorf("%s was not refused", name)
		case !c.refused && len(got) != 0:
			t.Errorf("%s was refused: %v", name, got)
		case c.refused && !strings.Contains(strings.Join(got, "\n"), c.names):
			t.Errorf("%s was refused without saying %q: %v", name, c.names, got)
		}
	}
}

// A head that dropped the declaration reds this row and no other, so a red run
// says which repair it wants rather than which area to look in.
func TestAHeadWithNoSchemeRedsExactlyOneRow(t *testing.T) {
	body := []byte(strings.Replace(cleanPage,
		`    <meta name="color-scheme" content="light dark" />`+"\n", "", 1))

	var refused []string
	for _, r := range Rules(fixtureNumbers) {
		if len(r.decide(body)) > 0 {
			refused = append(refused, r.ID)
		}
	}
	if len(refused) != 1 || refused[0] != "page-declares-the-schemes-it-supports" {
		t.Errorf("the head with no declaration refused %v, want only page-declares-the-schemes-it-supports", refused)
	}
}

// The colour row, at the level of the decision rather than through a whole run,
// because what it has to get right is which bytes are a colour and which are an
// address that looks like one.
// The motion row against the shapes it has to tell apart. Two of them answer the
// reader and are the two a stylesheet is actually written with, so a row
// refusing either would be a row somebody takes out the first time a page moves.
// The near miss is the last one: the query is there, the property is there, and
// what the reader gets for asking for less motion is the motion.
func TestTheMotionRowTellsTheQueryFromTheCascade(t *testing.T) {
	for name, c := range map[string]struct {
		style   string
		refused bool
		names   string
	}{
		"motion in the cascade, with nothing asking the reader": {
			`a { transition: color 120ms }`, true, "no query asking the reader about motion switches transition off"},
		"motion only for a reader who has not asked for less": {
			`@media (prefers-reduced-motion: no-preference) { a { transition: color 120ms } }`, false, ""},
		"motion in the cascade, switched off by the blanket override": {
			`a { transition: color 120ms }
			 @media (prefers-reduced-motion: reduce) { a { transition: none } }`, false, ""},
		// The override written before the rule it answers for, which is
		// where a stylesheet that resets first puts it.
		"the override written above the motion it answers for": {
			`@media (prefers-reduced-motion: reduce) { * { animation: none } }
			 a { animation: pulse 1s infinite }`, false, ""},
		// A second family is a second question. The override says
		// nothing about the one it does not name.
		"one family switched off and another left in the cascade": {
			`a { transition: color 120ms; animation: pulse 1s infinite }
			 @media (prefers-reduced-motion: reduce) { a { transition: none } }`, true,
			"switches animation off"},
		// The duration is what the override is written against as often
		// as the shorthand, and it stops the same thing.
		"the override written as a duration rather than as none": {
			`a { transition: color 120ms }
			 @media (prefers-reduced-motion: reduce) { a { transition-duration: 0s } }`, false, ""},
		"the override that has to win over an author rule": {
			`a { transition: color 120ms }
			 @media (prefers-reduced-motion: reduce) { a { transition: none !important } }`, false, ""},
		// A value with a duration in it moves something however short
		// the duration is, so reading the first word would pass this.
		"a value whose first word is a property and not a stopping word": {
			`a { transition: color 0.01s }`, true, "declares transition: color 0.01s"},
		"the query naming the feature with no answer": {
			`a { transition: color 120ms }
			 @media (prefers-reduced-motion) { a { transition: none } }`, false, ""},
		// The near miss. It is one word away from the shape above it and
		// it is the one a reader who set the preference actually meets.
		"motion written inside the query asking for reduced motion": {
			`@media (prefers-reduced-motion: reduce) { a { animation: shake 1s } }`, true,
			"what a reader gets for asking is the motion"},
		"scrolling that jumps rather than sliding": {
			`html { scroll-behavior: auto }`, false, ""},
		"scrolling that slides with nothing asking the reader": {
			`html { scroll-behavior: smooth }`, true, "switches scroll-behavior off"},
	} {
		page := []byte(strings.Replace(cleanPage, `</head>`,
			"  <style>"+c.style+"</style>\n  </head>", 1))
		got := decideMotion(page)
		if c.refused && len(got) == 0 {
			t.Errorf("%s: the row passed it", name)
			continue
		}
		if !c.refused && len(got) != 0 {
			t.Errorf("%s: the row refused it: %v", name, got)
			continue
		}
		if c.refused && !strings.Contains(strings.Join(got, "\n"), c.names) {
			t.Errorf("%s: the failure reads %v, which does not say %q", name, got, c.names)
		}
	}
}

// The failure names the line and the declaration, because a message saying only
// that the page moves something leaves the next person reading the whole
// stylesheet to find out which rule it meant.
func TestTheMotionRowNamesTheLineAndTheDeclaration(t *testing.T) {
	page := []byte(strings.Replace(cleanPage, `</head>`,
		"  <style>\n    a { transition: color 120ms }\n  </style>\n  </head>", 1))

	got := decideMotion(page)
	if len(got) != 1 {
		t.Fatalf("the row produced %d detail(s), want 1: %v", len(got), got)
	}
	for _, want := range []string{"line 8", "transition: color 120ms"} {
		if !strings.Contains(got[0], want) {
			t.Errorf("the failure reads %q, which does not carry %q", got[0], want)
		}
	}
}

// The two spellings one limit arrives in, and the longer one reported as itself
// rather than as the bare number inside it. A failure naming "37 ms" where the
// line says "under 37 ms" sends the next person looking for a string that is not
// there.
func TestTheBudgetRowRefusesBothSpellingsAndReportsTheLongerOne(t *testing.T) {
	decide := decideTypedBudgetNumber(fixtureNumbers)

	for _, c := range []struct{ line, want string }{
		{"A key press answers " + fixtureNumbers[0].Stated() + ".", `"under 37 ms"`},
		{"The ceiling is 37 ms and nothing may exceed it.", `"37 ms"`},
		{"It drops exactly 4 frames on the worst row.", `"exactly 4 frames"`},
	} {
		got := decide([]byte("A first line.\n" + c.line + "\n"))
		if len(got) != 1 {
			t.Errorf("%q produced %d detail(s), want 1: %v", c.line, len(got), got)
			continue
		}
		if !strings.Contains(got[0], c.want) {
			t.Errorf("the failure for %q reads %q, which does not carry %s", c.line, got[0], c.want)
		}
		if !strings.Contains(got[0], "line 2") {
			t.Errorf("the failure for %q reads %q, which does not name the line", c.line, got[0])
		}
		if !strings.Contains(got[0], tokens.File) {
			t.Errorf("the failure for %q reads %q, which does not name the file it is read from", c.line, got[0])
		}
	}
}

// What the row leaves alone. It compares against the values it was handed rather
// than against the shape of a number, so a figure that is not one of them walks
// through, and a limit in another unit walks through as well. Both bounds are
// deliberate: a row that refused every number with a time unit on it would
// refuse a transition duration, and the file rather than this row is what the
// set of budget numbers is decided by.
func TestTheBudgetRowLeavesANumberThatIsNotABudgetNumberAlone(t *testing.T) {
	decide := decideTypedBudgetNumber(fixtureNumbers)

	for name, line := range map[string]string{
		"a duration that is not a limit": "a { transition: color 120ms }",
		"a neighbouring figure":          "It answers under 38 ms on the machine it was measured on.",
		"the same limit in another unit": "It answers under 0.037 s.",
		"the digits with no unit":        "Thirty seven is 37 and nothing follows it.",
	} {
		if got := decide([]byte(line + "\n")); len(got) != 0 {
			t.Errorf("the row refused %s: %v", name, got)
		}
	}
}

// The row follows the copy rather than carrying the numbers. A row holding the
// five values in its own source would be the second definition of the file it
// exists to keep as the only one, and the way that shows is here: the same line
// is refused or left alone depending on what the copy says, and nothing in this
// package decides which.
func TestTheBudgetRowFollowsTheCopyRatherThanCarryingTheNumbers(t *testing.T) {
	line := []byte("A key press answers under 37 ms.\n")

	if got := decideTypedBudgetNumber(fixtureNumbers)(line); len(got) != 1 {
		t.Fatalf("the row produced %d detail(s) against the copy that carries that limit, want 1: %v", len(got), got)
	}

	moved := []tokens.Number{{Name: "first-frame", Limit: "24", Unit: "ms", Comparison: "below"}}
	if got := decideTypedBudgetNumber(moved)(line); len(got) != 0 {
		t.Errorf("the limit moved in the copy and the row went on refusing the old one: %v", got)
	}
	if got := decideTypedBudgetNumber(moved)([]byte("A key press answers under 24 ms.\n")); len(got) != 1 {
		t.Errorf("the limit moved in the copy and the row did not refuse the new one: %v", got)
	}
}

// A row handed nothing refuses nothing, which is why the run reads the copy
// before it builds the table. This is that half stated where somebody changing
// the table will meet it, and the run's own refusal is the case below.
func TestTheBudgetRowHandedNothingRefusesNothing(t *testing.T) {
	if got := decideTypedBudgetNumber(nil)([]byte("A key press answers under 37 ms.\n")); len(got) != 0 {
		t.Errorf("a row that was told no numbers refused a line anyway: %v", got)
	}
}

// The table is the same table however it is handed, which is what lets the count
// printed by the gate be taken from a call that supplies nothing. A row that
// appeared only when values arrived would make that count a different number
// from the one the run decided.
func TestTheTableHoldsTheSameRowsHoweverItIsHanded(t *testing.T) {
	var with, without []string
	for _, r := range Rules(fixtureNumbers) {
		with = append(with, r.ID)
	}
	for _, r := range Rules(nil) {
		without = append(without, r.ID)
	}
	if strings.Join(with, ",") != strings.Join(without, ",") {
		t.Errorf("the table handed values holds %v and the table handed nothing holds %v", with, without)
	}
}

// The limit typed into a page reds this row and no other, so a red run says
// which repair it wants rather than which area to look in. The colour row is the
// near neighbour: both are about a value that belongs in the copy, and a fixture
// tripping the two of them would leave neither proved.
func TestABudgetNumberTypedIntoAPageRedsExactlyOneRow(t *testing.T) {
	body := []byte(strings.Replace(cleanPage, `<h1>A title</h1>`,
		`<h1>A title</h1><p>A key press answers `+fixtureNumbers[0].Stated()+`.</p>`, 1))

	var refused []string
	for _, r := range Rules(fixtureNumbers) {
		if len(r.decide(body)) > 0 {
			refused = append(refused, r.ID)
		}
	}
	if len(refused) != 1 || refused[0] != "client-budget-numbers-live-in-exactly-one-file" {
		t.Errorf("the typed limit refused %v, want only client-budget-numbers-live-in-exactly-one-file", refused)
	}
}

func TestMotionInTheCascadeRedsExactlyOneRow(t *testing.T) {
	moving := []byte(strings.Replace(cleanPage, `</head>`,
		"  <style>a { transition: color 120ms }</style>\n  </head>", 1))

	var refused []string
	for _, r := range Rules(fixtureNumbers) {
		if len(r.decide(moving)) > 0 {
			refused = append(refused, r.ID)
		}
	}
	if len(refused) != 1 || refused[0] != "page-motion-answers-the-reader" {
		t.Errorf("motion in the cascade refused %v, want only page-motion-answers-the-reader", refused)
	}
}

func TestTheColourRowRefusesATypedColourAndLeavesAFragmentAlone(t *testing.T) {
	refused := map[string]string{
		"a full hex value in a style attribute": `<p style="color: #ECECEF">Read this</p>`,
		"the shorthand for a published value":   `<style>a { color: #fff }</style>`,
		"a value with alpha on it":              `<style>hr { background: #FFFFFF12 }</style>`,
		"a colour in the prose the build reads": `The accent is #5B9CFF on a dark ground.`,
	}
	for name, line := range refused {
		got := decideTypedTokenValue([]byte(line))
		if len(got) != 1 {
			t.Errorf("the colour row did not refuse %s: %v", name, got)
			continue
		}
		if !strings.Contains(got[0], tokens.File) {
			t.Errorf("refusing %s reads %q, which does not say where a colour is read from", name, got[0])
		}
	}

	spared := map[string]string{
		"a fragment reference":         `<a href="#content">Skip to the content</a>`,
		"a fragment on an image":       `<img src="#a1b2c3" alt="" width="1" height="1" />`,
		"a word that is not a colour":  `<a href="/install/">#install the plugin</a>`,
		"a run of digits that is five": `<p>Case #12345 is closed.</p>`,
	}
	for name, line := range spared {
		if got := decideTypedTokenValue([]byte(line)); len(got) != 0 {
			t.Errorf("the colour row refused %s: %v", name, got)
		}
	}
}

// The one-character version of the mistake: the value is on the screen in front
// of somebody writing the markup that needs it, so they type it.
func TestRunRefusesAColourTypedIntoTheTemplate(t *testing.T) {
	root := tree(t, strings.Replace(goodTemplate, `<body>`, `<body style="background: #121216">`, 1))

	var log bytes.Buffer
	err := Run(root, &log)
	if err == nil {
		t.Fatalf("Run accepted a colour typed into the template:\n%s", log.String())
	}
	if !strings.Contains(log.String(), "design-tokens-live-in-exactly-one-file: REFUSED") {
		t.Errorf("the run does not name the row that refused:\n%s", log.String())
	}
	if !strings.Contains(log.String(), "writes the colour #121216") {
		t.Errorf("the run does not say which colour was typed:\n%s", log.String())
	}
}

// Exactly one file, and zero is not one. A run over a tree carrying no copy
// would otherwise report that it found no second definition of a value that has
// no first one, which is a green run over the state the row exists to refuse.
func TestRunRefusesATreeCarryingNoCopyOfTheDesignTokens(t *testing.T) {
	root := tree(t, goodTemplate)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(tokens.File))); err != nil {
		t.Fatalf("removing the copy: %v", err)
	}
	git(t, root, "rm", "-q", "--cached", tokens.File)

	var log bytes.Buffer
	err := Run(root, &log)
	if err == nil {
		t.Fatalf("Run accepted a tree with no copy of the design tokens:\n%s", log.String())
	}
	if !strings.Contains(err.Error(), tokens.File) {
		t.Errorf("the refusal reads %q, which does not name the file that is missing", err)
	}
}

// The expiry row, at the level of the decision. It is the one row that reads
// the clock, so the three answers it has to tell apart are the point: a file
// that is not its subject, one whose expiry is still ahead, and one whose
// expiry has passed.
func TestTheExpiryRowRefusesAPassedDateAndPassesTheRest(t *testing.T) {
	ahead := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	if got := decideExpiry([]byte("Expires: " + ahead + "\n")); len(got) != 0 {
		t.Errorf("the expiry row refused a file that has not expired: %v", got)
	}
	if got := decideExpiry([]byte(cleanPage)); len(got) != 0 {
		t.Errorf("the expiry row refused a file carrying no expiry: %v", got)
	}

	got := decideExpiry([]byte("Expires: 2020-01-01T00:00:00Z\n"))
	if len(got) != 1 {
		t.Fatalf("the expiry row did not refuse a date that has passed: %v", got)
	}
	if !strings.Contains(got[0], "2020-01-01T00:00:00Z") {
		t.Errorf("the refusal reads %q and does not name the date", got[0])
	}
	if !strings.Contains(got[0], security.File) {
		t.Errorf("the refusal reads %q and does not say where the date is moved forward", got[0])
	}

	if got := decideExpiry([]byte("Expires: next August\n")); len(got) != 1 {
		t.Errorf("the expiry row passed a file whose expiry is not a moment: %v", got)
	}
}

// A tree carrying no source for the produced file is refused by name. Without
// it the row above would examine an output that has no such file in it and
// report that nothing had expired.
func TestRunRefusesATreeCarryingNoSourceForTheReportingRoute(t *testing.T) {
	root := tree(t, goodTemplate)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(security.File))); err != nil {
		t.Fatalf("removing the source: %v", err)
	}
	git(t, root, "rm", "-q", "--cached", security.File)

	var log bytes.Buffer
	err := Run(root, &log)
	if err == nil {
		t.Fatalf("Run accepted a tree with no source for the reporting route:\n%s", log.String())
	}
	if !strings.Contains(err.Error(), security.File) {
		t.Errorf("the refusal reads %q, which does not name the file that is missing", err)
	}
}

// The measured number and the limit are both in the refusal. A budget failure
// that says only that the build failed makes the next person measure by hand, and
// the number they would reach for is the one the row already has.
func TestTheBudgetRowsNameTheMeasurementAndTheLimit(t *testing.T) {
	oversize := []byte(strings.Replace(cleanPage, contentOpen,
		contentOpen+"<!--"+strings.Repeat("p", budget.HTMLBytes)+"-->", 1))
	got := decideMarkupBudget(oversize)
	if len(got) != 1 {
		t.Fatalf("the row produced %d detail(s), want 1: %v", len(got), got)
	}
	for _, want := range []string{
		strconv.Itoa(len(oversize)),
		strconv.Itoa(budget.HTMLBytes),
		budget.Record,
	} {
		if !strings.Contains(got[0], want) {
			t.Errorf("the refusal reads %q, which does not carry %q", got[0], want)
		}
	}

	heavy := []byte(strings.Replace(cleanPage, `</head>`,
		"  <style>"+strings.Repeat("a{color:red}", budget.InlineCSSBytes/12+1)+"</style>\n  </head>", 1))
	got = decideStylesheetBudget(heavy)
	if len(got) != 1 {
		t.Fatalf("the stylesheet row produced %d detail(s), want 1: %v", len(got), got)
	}
	for _, want := range []string{strconv.Itoa(budget.InlineCSSBytes), budget.Record} {
		if !strings.Contains(got[0], want) {
			t.Errorf("the refusal reads %q, which does not carry %q", got[0], want)
		}
	}
}

// A page exactly at the limit is refused and a page one byte under is not, because
// the record writes each line as a limit rather than as a target and the byte at
// the boundary is the one nobody tests.
func TestTheMarkupBudgetRowRefusesTheBoundary(t *testing.T) {
	at := make([]byte, budget.HTMLBytes)
	if got := decideMarkupBudget(at); len(got) == 0 {
		t.Errorf("the row passed a page of exactly %d bytes", budget.HTMLBytes)
	}
	under := make([]byte, budget.HTMLBytes-1)
	if got := decideMarkupBudget(under); len(got) != 0 {
		t.Errorf("the row refused a page one byte under the limit: %v", got)
	}
}

// The image line is written for the landing page and says nothing about the
// others, so a frame putting three images on every page reds this row once. A row
// applying it everywhere would refuse more than the record does, which is a rule
// nobody argued.
func TestTheLandingImageRowReadsTheLandingPageAndNoOther(t *testing.T) {
	images := strings.Repeat(`<img src="/a.png" alt="A picture of something" width="8" height="8" />`,
		budget.LandingImages+1)
	root := tree(t, strings.Replace(goodTemplate, `      <h1>{{ .Title }}</h1>`,
		`      <h1>{{ .Title }}</h1>`+images, 1))

	var log bytes.Buffer
	if err := Run(root, &log); err == nil {
		t.Fatalf("Run accepted a frame asking for more images than the budget allows:\n%s", log.String())
	}
	if !strings.Contains(log.String(), "landing-page-asks-for-at-most-two-images: REFUSED, 1 violation(s)") {
		t.Errorf("the run does not refuse exactly the landing page; it said:\n%s", log.String())
	}
	if !strings.Contains(log.String(), "dist/index.html: this page asks for 3 image(s)") {
		t.Errorf("the run does not name the landing page and its count; it said:\n%s", log.String())
	}
	if strings.Contains(log.String(), "dist/privacy/index.html: this page asks for") {
		t.Errorf("the run judged a page this row is not about; it said:\n%s", log.String())
	}
}

// A narrowed row whose page the build did not write is left with nothing to read.
// It reports that rather than passing, so a page renamed out from under one of
// these does not leave a green row behind it.
func TestANarrowedRowFindsNothingWhenThePageIsNotThere(t *testing.T) {
	files := []file{{name: "dist/index.html", body: []byte("a")}}
	if got := onlyNamed(files, "dist/index.html"); len(got) != 1 {
		t.Errorf("the narrowing dropped the page it was asked for: %v", got)
	}
	if got := onlyNamed(files, "dist/install/index.html"); len(got) != 0 {
		t.Errorf("the narrowing returned %d file(s) for a page the build did not write", len(got))
	}
}

// A tree that lost the claim about the clients is refused before any row is
// decided, and the refusal names the file. The landing page still builds
// without it, and what it builds is a page leading with what this project is
// and saying nothing about the clients at all, which reads as a project that
// has none. No reading of the produced bytes separates a sentence that was
// never composed from one nobody ever asked for, so the absence is refused here
// rather than left to a row over the output.
// A tree that lost its roster. The page it produces is a page about a set of
// plugins listing none of them, and nothing in the produced bytes tells that
// apart from a project that has none, which is why the refusal is here rather
// than in a row over the output.
func TestRunRefusesATreeThatLostItsRoster(t *testing.T) {
	root := tree(t, goodTemplate)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(site.RosterFile))); err != nil {
		t.Fatalf("removing the roster: %v", err)
	}
	git(t, root, "add", "-A")

	var log bytes.Buffer
	err := Run(root, &log)
	if err == nil {
		t.Fatalf("Run accepted a tree carrying no roster:\n%s", log.String())
	}
	if !strings.Contains(err.Error(), site.RosterFile) {
		t.Errorf("the refusal reads %q, which does not name the file that is missing", err)
	}
}

func TestRunRefusesATreeThatLostTheClaimAboutTheClients(t *testing.T) {
	root := tree(t, goodTemplate)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(site.ClientsFile))); err != nil {
		t.Fatalf("removing %s: %v", site.ClientsFile, err)
	}
	git(t, root, "rm", "-q", "--cached", filepath.FromSlash(site.ClientsFile))

	var log bytes.Buffer
	err := Run(root, &log)
	if err == nil {
		t.Fatalf("Run accepted a tree with no claim about the clients:\n%s", log.String())
	}
	if !strings.Contains(err.Error(), site.ClientsFile) {
		t.Errorf("the refusal reads %q, which does not name the file that was missing", err)
	}
	if !strings.Contains(err.Error(), "says nothing about the clients") {
		t.Errorf("the refusal reads %q, which does not say what the tree lost", err)
	}
}

// The mistake in the place it is actually made. A cookie written as a meta
// element needs nothing running on the page: a host serves the document and the
// browser acts on it, so it is the one way a site with a zero-byte scripting
// budget still sets one. The frame is one file, so it reds every page at once
// and the run names each of them.
func TestRunRefusesAFrameThatSetsACookie(t *testing.T) {
	root := tree(t, strings.Replace(goodTemplate, "  </head>",
		`    <meta http-equiv="set-cookie" content="seen=1" />`+"\n  </head>", 1))

	var log bytes.Buffer
	if err := Run(root, &log); err == nil {
		t.Fatalf("Run passed a frame that sets a cookie:\n%s", log.String())
	}
	for _, want := range []string{
		fmt.Sprintf("page-touches-no-browser-storage: REFUSED, %d violation(s)", producedPages(t, root)),
		"carries a meta element that sets a cookie",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not carry %q; it said:\n%s", want, log.String())
		}
	}
}

// The other half of the same row, and it is a different repair. A name is code
// that would have to run; the element above is a header the browser acts on with
// nothing running. A refusal naming only one of the two sends the next person
// looking in the wrong half of the page.
//
// The fixture is the near miss rather than a page with the word typed into a
// sentence. A script element carrying its code inside the page has no src
// attribute, so the row about a script element passes it, and what it does is
// write into the storage area this site says it leaves alone.
func TestRunRefusesAPageThatNamesAStorageInterface(t *testing.T) {
	root := tree(t, strings.Replace(goodTemplate, "      <h1>{{ .Title }}</h1>",
		"      <h1>{{ .Title }}</h1>\n      <script>localStorage.setItem(\"seen\", \"1\")</script>", 1))

	var log bytes.Buffer
	if err := Run(root, &log); err == nil {
		t.Fatalf("Run passed a page naming a storage interface:\n%s", log.String())
	}
	if !strings.Contains(log.String(), "names localStorage") {
		t.Errorf("the run does not name what it found; it said:\n%s", log.String())
	}
}

// The one-character version of the mistake this row exists for. The row about a
// script element reads the src attribute, so a page with no source anywhere and
// one handler on one element runs code in a reader's browser and passes every
// row this gate had before this one.
func TestRunRefusesAPageCarryingAHandlerRatherThanAScriptSource(t *testing.T) {
	handler := `<a href="/" onclick="alert(1)">Somewhere</a>`
	root := tree(t, strings.Replace(goodTemplate, "      <h1>{{ .Title }}</h1>",
		"      <h1>{{ .Title }}</h1>\n      "+handler, 1))

	var log bytes.Buffer
	if err := Run(root, &log); err == nil {
		t.Fatalf("Run passed a page carrying a handler:\n%s", log.String())
	}
	if !strings.Contains(log.String(), "page-fetches-no-script: ok") {
		t.Errorf("the row about a script source judged this page rather than passing it; it said:\n%s", log.String())
	}
	for _, want := range []string{
		fmt.Sprintf("page-carries-no-inline-handler: REFUSED, %d violation(s)", producedPages(t, root)),
		"the a element carries the handler attribute onclick",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not carry %q; it said:\n%s", want, log.String())
		}
	}
}

// An address whose scheme runs code rather than fetching anything. It carries no
// attribute name a handler pattern would find, and the row that reads what a
// page references reads the host, which an address of this shape does not have.
func TestRunRefusesAnAddressWhoseSchemeIsAScript(t *testing.T) {
	root := tree(t, strings.Replace(goodTemplate, `<p><a href="/legal/">Who publishes this site</a></p>`,
		`<p><a href="/legal/">Who publishes this site</a></p>`+
			"\n      "+`<p><a href="javascript:alert(1)">Somewhere</a></p>`, 1))

	var log bytes.Buffer
	if err := Run(root, &log); err == nil {
		t.Fatalf("Run passed a page whose address is a script:\n%s", log.String())
	}
	if !strings.Contains(log.String(), "carries an address whose scheme is a script") {
		t.Errorf("the run does not say what it found; it said:\n%s", log.String())
	}
}

// The neighbour that has to stay green, because both rows read names over the
// whole document rather than inside anything. A page carrying none of what
// either judges is passed by both, so a red run over the cases above is about
// what was put on the page and not about the pages themselves.
func TestTheTwoBrowserRowsPassAPageThatCarriesNeither(t *testing.T) {
	var log bytes.Buffer
	if err := Run(tree(t, goodTemplate), &log); err != nil {
		t.Fatalf("Run refused a tree carrying neither: %v\n%s", err, log.String())
	}
	for _, want := range []string{
		"page-touches-no-browser-storage: ok",
		"page-carries-no-inline-handler: ok",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not report %q; it said:\n%s", want, log.String())
		}
	}
}
