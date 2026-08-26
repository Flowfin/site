// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the install page.
//
// The page has two ways of looking finished while being useless, and every case
// below is one of them. It can print steps with nothing to paste, which is the
// address absent or unsettled; and it can print steps without saying which
// plugins they apply to, which is the reader following install instructions for
// something that has published nothing. So the cases are trees that produce a
// page a reader would read as complete, refused for the reason each one is
// wrong rather than for being empty.
//
// Every case writes its own tree. A case reading the catalogue file this
// repository carries would pass or fail on whichever state that value happens
// to be in today, and both states are ones this page has to render.
package site

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Flowfin/site/internal/budget"
	"github.com/Flowfin/site/internal/releases"
)

// installTemplate renders the page's words, its list and the way onward, so a
// case reads what the build put on the page rather than whatever the
// repository's frame carries today.
const installTemplate = `<title>{{ .Title }}</title>
<meta name="description" content="{{ .Description }}" />
<h1>{{ .Title }}</h1>
{{- range .Paragraphs }}
<p>{{ . }}</p>
{{- end }}
{{- if .InstallableRead }}
<p id="installable">{{ .InstallableRead }}</p>
{{- range .Installable }}
<li><a href="{{ .Href }}">{{ .ID }}</a> {{ .Summary }}</li>
{{- end }}
{{- end }}
{{- range .Onward }}
<a href="{{ .Href }}">{{ .Text }}</a>
{{- end }}
`

const installProse = `Installing a plugin

description: What to add to a server so that these plugins appear in it.

One paragraph.

onward: Every plugin and what state it is in [/]
`

// The two states the address may be in, as the smallest file that is each one.
const (
	addressAnswered = `{
  "state": "answered",
  "address": "https://fixture.example/manifest.json"
}
`
	addressUndecided = `{
  "state": "undecided",
  "waiting": "an open question"
}
`
)

// One row that ships and one that does not, so a page rendering the whole
// roster and a page rendering the subset are distinguishable. The shipping row
// is second, so a page taking the first row rather than the computed one is
// caught too.
const (
	oneShipsOneDoesNot = `[
  {"id":"alpha","repository":"Flowfin/jellyfin-plugin-alpha","summary":"What alpha does","state":"build-up"},
  {"id":"beta","repository":"Flowfin/jellyfin-plugin-beta","summary":"What beta does","state":"shell"}
]`
	betaShips = `{"taken":"2026-01-02","command":"a command","repositories":{
  "Flowfin/jellyfin-plugin-alpha":{"finished":0,"prereleases":3},
  "Flowfin/jellyfin-plugin-beta":{"finished":1,"prereleases":0}}}`
	neitherShips = `{"taken":"2026-01-02","command":"a command","repositories":{
  "Flowfin/jellyfin-plugin-alpha":{"finished":0,"prereleases":3},
  "Flowfin/jellyfin-plugin-beta":{"finished":0,"prereleases":0}}}`
)

// installTree writes a buildable tree that produces the landing page and the
// install page. An empty roster or record leaves that file out, which is how a
// case asks what the page says with no rows to compute a list from.
func installTree(t *testing.T, prose, address, roster, record string) string {
	t.Helper()
	root := t.TempDir()
	mkdir(t, filepath.Join(root, TemplatesDir))
	write(t, filepath.Join(root, TemplatesDir, "page.html.tmpl"), installTemplate)
	mkdir(t, filepath.Join(root, ContentDir))
	write(t, filepath.Join(root, ContentDir, "index.txt"), headIndex)
	write(t, filepath.Join(root, filepath.FromSlash(InstallProse)), prose)
	mkdir(t, filepath.Join(root, filepath.Dir(filepath.FromSlash(InstallFile))))
	write(t, filepath.Join(root, filepath.FromSlash(InstallFile)), address)
	if roster != "" {
		write(t, filepath.Join(root, filepath.FromSlash(RosterFile)), roster)
		proseForEveryRow(t, root, roster)
	}
	if record != "" {
		write(t, filepath.Join(root, filepath.FromSlash(releases.File)), record)
	}
	return root
}

// installed builds a tree these cases expect to be accepted and answers with the
// page.
func installed(t *testing.T, prose, address, roster, record string) string {
	t.Helper()
	root := installTree(t, prose, address, roster, record)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused a tree this case expects it to accept: %v", err)
	}
	return read(t, filepath.Join(root, OutputDir, filepath.FromSlash(InstallPath)))
}

// installRefusal builds a tree that must be refused and returns what the
// refusal said.
func installRefusal(t *testing.T, prose, address string) string {
	t.Helper()
	_, err := Build(installTree(t, prose, address, oneShipsOneDoesNot, betaShips), OutputDir, io.Discard)
	if err == nil {
		t.Fatalf("the build was accepted, and it carries the mistake this case is about:\n%s", address)
	}
	return err.Error()
}

// The page is produced, at the address the record gives it, and the address a
// reader pastes comes out of the file rather than out of the prose.
func TestTheInstallPageRendersWithAnAddress(t *testing.T) {
	root := installTree(t, installProse, addressAnswered, oneShipsOneDoesNot, betaShips)
	written, err := Build(root, OutputDir, io.Discard)
	if err != nil {
		t.Fatalf("the build refused: %v", err)
	}
	want := OutputDir + "/" + InstallPath
	var found bool
	for _, w := range written {
		if w == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("the build wrote %q and none of them is %s", written, want)
	}

	page := read(t, filepath.Join(root, OutputDir, filepath.FromSlash(InstallPath)))
	if !strings.Contains(page, "https://fixture.example/manifest.json") {
		t.Errorf("the address in the file did not reach the page:\n%s", page)
	}
	if strings.Contains(page, "no address to add yet") {
		t.Errorf("the page says there is no address while carrying one:\n%s", page)
	}
}

// Changing the address in the file changes the page, which is what makes the
// answer a data change rather than an edit to prose. It is the clause the page
// was built around: whichever address the project settles on, nothing here is
// rewritten for it.
func TestChangingTheAddressChangesThePage(t *testing.T) {
	root := installTree(t, installProse, addressAnswered, oneShipsOneDoesNot, betaShips)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused: %v", err)
	}
	moved := strings.Replace(addressAnswered, "fixture.example", "moved.example", 1)
	write(t, filepath.Join(root, filepath.FromSlash(InstallFile)), moved)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused after the address changed: %v", err)
	}
	page := read(t, filepath.Join(root, OutputDir, filepath.FromSlash(InstallPath)))
	if strings.Contains(page, "fixture.example") {
		t.Errorf("the page still carries the address the file no longer holds:\n%s", page)
	}
	if !strings.Contains(page, "https://moved.example/manifest.json") {
		t.Errorf("the address the file now holds did not reach the page:\n%s", page)
	}
}

// With no address settled the page renders as a sentence naming what the answer
// waits on, and prints no address at all. A broken link or an empty space would
// both be followed; a sentence is not.
func TestTheInstallPageRendersSensiblyWithNoAddress(t *testing.T) {
	page := installed(t, installProse, addressUndecided, oneShipsOneDoesNot, betaShips)
	if strings.Contains(page, "://") {
		t.Errorf("the page carries an address while none is settled:\n%s", page)
	}
	if !strings.Contains(page, "There is no address to add yet.") {
		t.Errorf("the page does not say that there is no address yet:\n%s", page)
	}
	if !strings.Contains(page, "an open question") {
		t.Errorf("the page does not name what the answer waits on:\n%s", page)
	}
	// The rest of the page is unchanged by the address being open, which is
	// what makes the answer, when it comes, a data change.
	if !strings.Contains(page, `id="installable"`) {
		t.Errorf("the page dropped the list of what can be installed:\n%s", page)
	}
}

// The list is the computed shipping state and not the declared word. The row
// that ships declares `shell`, so a page filtering on what the roster says
// carries the wrong one, and a page filtering on what was published carries this
// one.
func TestTheListIsTheComputedShippingState(t *testing.T) {
	page := installed(t, installProse, addressAnswered, oneShipsOneDoesNot, betaShips)
	if !strings.Contains(page, `<a href="/plugins/beta/">beta</a>`) {
		t.Errorf("the row with a finished release is not on the page:\n%s", page)
	}
	if strings.Contains(page, `<a href="/plugins/alpha/">alpha</a>`) {
		t.Errorf("a row with no finished release is offered as installable:\n%s", page)
	}
	if !strings.Contains(page, "One of the 2 plugins has a finished release published") {
		t.Errorf("the sentence above the list does not count what is below it:\n%s", page)
	}
}

// A prerelease is not something to install. It is the one publication a plugin
// that does not ship can have, and a page counting it would send a reader to a
// server list that has nothing in it.
func TestAPrereleaseIsNotInstallable(t *testing.T) {
	page := installed(t, installProse, addressAnswered, oneShipsOneDoesNot, neitherShips)
	if strings.Contains(page, `<a href="/plugins/alpha/">alpha</a>`) {
		t.Errorf("the row whose only publication is a prerelease is offered as installable:\n%s", page)
	}
	if !strings.Contains(page, "None of the 2 plugins has a finished release published") {
		t.Errorf("the page does not say that nothing can be installed today:\n%s", page)
	}
}

// Where nothing ships the page says so in a sentence rather than rendering an
// empty space. This is the failure the clause exists against: a page that prints
// the steps and omits which plugins they apply to looks finished while telling a
// reader nothing.
func TestWithNothingShippingThePageSaysSoRatherThanShowingNothing(t *testing.T) {
	page := installed(t, installProse, addressAnswered, oneShipsOneDoesNot, neitherShips)
	if strings.Contains(page, "<li>") {
		t.Errorf("the page lists something while nothing has a finished release:\n%s", page)
	}
	if !strings.Contains(page, `id="installable"`) {
		t.Errorf("the page went silent about what can be installed rather than saying nothing can:\n%s", page)
	}
}

// The page fits the line of the budget that is decidable by reading it. It is
// asserted here as well as by the row over the produced output, because the row
// reads the pages this repository's tree produces and this reads the page a
// tree of rows and a settled address produces, which is the shape the real one
// grows into.
func TestTheInstallPageFitsTheMarkupBudget(t *testing.T) {
	root := t.TempDir()
	copyTree(t, fixtureTree, root)
	copyFile(t,
		filepath.Join("..", "..", TemplatesDir, "page.html.tmpl"),
		filepath.Join(root, TemplatesDir, "page.html.tmpl"))
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused the fixture tree: %v", err)
	}
	page := read(t, filepath.Join(root, OutputDir, filepath.FromSlash(InstallPath)))
	if len(page) > budget.HTMLBytes {
		t.Fatalf("the install page is %d bytes against a limit of %d, argued in %s",
			len(page), budget.HTMLBytes, budget.Record)
	}
}

// Every shape that renders as a finished page with nothing behind it, refused
// for the reason it is wrong. The two states are the whole vocabulary, so a file
// declaring a third is refused rather than read as one of them.
func TestTheAddressFileIsRefusedForEachWayItLooksFinished(t *testing.T) {
	for name, c := range map[string]struct{ file, says string }{
		"answered with no address": {
			`{"state":"answered"}`,
			"carries no address",
		},
		"answered with an address a server cannot fetch": {
			`{"state":"answered","address":"flowfin.dev/manifest.json"}`,
			"does not open with https://",
		},
		"answered and still waiting": {
			`{"state":"answered","address":"https://fixture.example/manifest.json","waiting":"an open question"}`,
			"reads as open and shows an answer at the same time",
		},
		"undecided with an address": {
			`{"state":"undecided","address":"https://fixture.example/manifest.json","waiting":"an open question"}`,
			"which no reader will see",
		},
		"undecided waiting on nothing": {
			`{"state":"undecided"}`,
			"names nothing it waits on",
		},
		"no state at all": {
			`{"address":"https://fixture.example/manifest.json"}`,
			"declares no state",
		},
		"a state outside the two": {
			`{"state":"pending","address":"https://fixture.example/manifest.json"}`,
			`declares the state "pending"`,
		},
		"a field that is not one of this file's": {
			`{"state":"answered","address":"https://fixture.example/manifest.json","url":"https://fixture.example/"}`,
			`carries the field "url"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			said := installRefusal(t, installProse, c.file)
			if !strings.Contains(said, c.says) {
				t.Fatalf("the refusal does not say why it refused:\nwant it to contain %q\ngot:\n%s", c.says, said)
			}
		})
	}
}

// A page a reader arrived at with a server open in front of them and no way back
// is where they stop, and it renders exactly like a finished one.
func TestTheInstallProseIsRefusedWithNoWayOnward(t *testing.T) {
	said := installRefusal(t,
		"Installing a plugin\n\ndescription: What to add to a server.\n\nOne paragraph.\n",
		addressAnswered)
	if !strings.Contains(said, "offers no way back") {
		t.Fatalf("the refusal does not say what was missing:\n%s", said)
	}
}

// With no rows to compute a list from, the page says nothing about what can be
// installed rather than saying that nothing can. Those are different statements
// and only one of them is true of a tree with no roster in it.
func TestWithNoRosterThePageClaimsNothingAboutWhatShips(t *testing.T) {
	page := installed(t, installProse, addressAnswered, "", "")
	if strings.Contains(page, "None of the") {
		t.Errorf("the page says nothing ships, on a tree with no roster to read that from:\n%s", page)
	}
	if !strings.Contains(page, "no plugin list to compute this from") {
		t.Errorf("the page does not say why it lists nothing:\n%s", page)
	}
}
