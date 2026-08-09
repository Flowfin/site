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
  </body>
</html>
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
		// One element, of the shape a copied snippet arrives in.
		"page-fetches-no-script": []byte(strings.Replace(cleanPage, `</head>`,
			`  <script src="https://example.invalid/a.js"></script>`+"\n  </head>", 1)),
		// A note to the author that reached the output.
		"output-carries-no-unfinished-marker": []byte(strings.Replace(cleanPage, `<h1>A title</h1>`,
			`<h1>A title</h1><!-- TODO: the real heading -->`, 1)),
		"tracked-text-names-no-tool": b64(t, "QSBub3RlIGFib3ZlLgpHZW5lcmF0ZWQgYnkgQ2hhdEdQVCBhbmQgbGVmdCBpbi4K"),
	}

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
		if got := r.decide([]byte(cleanPage)); len(got) != 0 {
			t.Errorf("row %s refused a page that breaks nothing: %v", r.ID, got)
		}
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
	wr(filepath.Join("templates", "page.html.tmpl"), template)
	wr(filepath.Join("content", "index.txt"), "A title\n\nOne paragraph.\n")

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
