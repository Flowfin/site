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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Flowfin/site/internal/tokens"
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

const cleanPage = `<!DOCTYPE html>
<html lang="en">
  <head>
    <title>A title</title>
  </head>
  <body>
    <main><h1>A title</h1></main>
    <footer>
      <p>
        Flowfin is not affiliated with the Jellyfin project. Other projects are
        named here to say what this software works with.
      </p>
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
		// The footer dropped, which is what a second template or a page
		// written by hand looks like. The sentence lives in one file so
		// that no page can ship without it, and this is the row that
		// makes that a property of the pages rather than a habit.
		"page-carries-the-affiliation-notice": []byte(strings.Replace(cleanPage,
			"Flowfin is not affiliated with the Jellyfin project.", "", 1)),
		// One element, of the shape a copied snippet arrives in.
		"page-fetches-no-script": []byte(strings.Replace(cleanPage, `</head>`,
			`  <script src="https://example.invalid/a.js"></script>`+"\n  </head>", 1)),
		// An image written the way somebody writes one, with the source
		// and the alternative text and nothing about how much room it
		// needs. It is the first thing anybody adds to a page and it is
		// the whole of the layout shift the budget puts at zero.
		"image-carries-its-own-dimensions": []byte(strings.Replace(cleanPage, `<main>`,
			`<main><img src="/icon.png" alt="The mark" />`, 1)),
		// A note to the author that reached the output.
		"output-carries-no-unfinished-marker": []byte(strings.Replace(cleanPage, `<h1>A title</h1>`,
			`<h1>A title</h1><!-- TODO: the real heading -->`, 1)),
		// A stylesheet from somebody else's domain, which is the shape a
		// page picks up the moment anybody reaches for a font or an icon
		// set. It trips this row and no other, so a red run says which
		// repair it wants.
		"output-references-no-domain-outside-the-allowlist": []byte(strings.Replace(cleanPage, `</head>`,
			`  <link rel="stylesheet" href="https://cdn.example.invalid/a.css" />`+"\n  </head>", 1)),
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
	}

	// The neighbour a row is given is of the population it reads. A page rule
	// judged against a Go file, or the other way round, would pass for the
	// wrong reason.
	cleanTest := b64(t, "cGFja2FnZSBzYW1wbGUKCmltcG9ydCAoCgkib3MiCgkidGVzdGluZyIKKQoKZnVuYyBUZXN0U29tZXRoaW5nKHQgKnRlc3RpbmcuVCkgewoJaWYgb3MuR2V0ZW52KCJIT01FIikgPT0gIiIgewoJCXQuU2tpcCgibm8gaG9tZSIpCgl9Cn0K")

	rules := Rules()
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
		}
		if got := r.decide(neighbour); len(got) != 0 {
			t.Errorf("row %s refused a %s that breaks nothing: %v", r.ID, r.Subject, got)
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

// A violation reds exactly one row. A fixture that trips two of them would make
// a red run unreadable, and a row that quietly also refuses its neighbour's case
// is a row whose message points at the wrong repair.
func TestAViolationRedsExactlyOneRow(t *testing.T) {
	noLang := []byte(strings.Replace(cleanPage, `<html lang="en">`, `<html>`, 1))

	var refused []string
	for _, r := range Rules() {
		if len(r.decide(noLang)) > 0 {
			refused = append(refused, r.ID)
		}
	}
	if len(refused) != 1 || refused[0] != "page-declares-its-language" {
		t.Errorf("the missing language attribute refused %v, want only page-declares-its-language", refused)
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
		body := []byte(strings.Replace(cleanPage, `<main>`, ref.markup+`<main>`, 1))
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
		body := []byte(strings.Replace(cleanPage, `<main>`, markup+`<main>`, 1))
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
	wr(filepath.Join("content", "index.txt"), "A title\n\nOne paragraph.\n")
	// The copy the build reads. It carries a colour, because the row about
	// where a colour is read from is about there being one file that may
	// carry one, and a fixture whose copy held none would prove nothing
	// about which file that is.
	wr(filepath.FromSlash(tokens.File), `{"surface":{"ground":{"dark":{"srgb":"#121216","alpha":1}}}}`)

	git(t, root, "init", "-q")
	git(t, root, "add", "-A")
	return root
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
    <title>{{ .Title }}</title>
  </head>
  <body>
    <main>
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
		body := []byte(strings.Replace(cleanPage, `<main>`, `<main>`+c.markup, 1))
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
	body := []byte(strings.Replace(cleanPage, `<main>`,
		`<main><img src="/assets/icon.png" alt="The mark" />`, 1))

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
	body := []byte(strings.Replace(cleanPage, `<main>`,
		`<main><img src="/icon.png" alt="The mark" />`, 1))

	var refused []string
	for _, r := range Rules() {
		if len(r.decide(body)) > 0 {
			refused = append(refused, r.ID)
		}
	}
	if len(refused) != 1 || refused[0] != "image-carries-its-own-dimensions" {
		t.Errorf("the image refused %v, want only image-carries-its-own-dimensions", refused)
	}
}

// The colour row, at the level of the decision rather than through a whole run,
// because what it has to get right is which bytes are a colour and which are an
// address that looks like one.
func TestTheColourRowRefusesATypedColourAndLeavesAFragmentAlone(t *testing.T) {
	refused := map[string]string{
		"a full hex value in a style attribute": `<p style="color: #ECECEF">Read this</p>`,
		"the shorthand for a published value":   `<style>a { color: #fff }</style>`,
		"a value with alpha on it":              `<style>hr { background: #FFFFFF12 }</style>`,
		"a colour in the prose the build reads": `The accent is #5B9CFF on a dark ground.`,
	}
	for name, line := range refused {
		got := decideTypedColour([]byte(line))
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
		if got := decideTypedColour([]byte(line)); len(got) != 0 {
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
