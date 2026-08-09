// The suite over the two-build comparison.
//
// The real build reads only what is committed, so it will not produce a
// difference on request. What the suite can prove is that the comparison notices
// one and names it, so the cases that need a difference supply their own builder.
// The end-to-end proof is a generator that actually reads the clock, and that is
// a run on a pull request rather than a case here.
package reproduce

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const template = `<!DOCTYPE html>
<html lang="en">
  <head>
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

func tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, filepath.Join("templates", "page.html.tmpl"), template)
	write(t, root, filepath.Join("content", "index.txt"), "A title\n\nOne paragraph.\n")
	return root
}

func write(t *testing.T, root, name, body string) {
	t.Helper()
	full := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("preparing %s: %v", name, err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

// The neighbour, and it is the real build rather than a stand-in: a tree whose
// build reads only what is committed produces the same bytes twice, and the run
// says how many files it compared rather than passing silently.
func TestTwoBuildsOfTheRealGeneratorAgree(t *testing.T) {
	var log bytes.Buffer
	if err := Run(tree(t), &log); err != nil {
		t.Fatalf("Run refused two builds of one source: %v\n%s", err, log.String())
	}
	if !strings.Contains(log.String(), "1 file(s), identical in both builds") {
		t.Errorf("the run did not say what it compared; it said:\n%s", log.String())
	}
}

// A generator that reads the clock produces a page that differs in the middle.
// The failure has to name the file, because one saying only that the build is
// not reproducible makes the next person diff by hand.
func TestTheComparisonNamesTheFileWhoseBytesMoved(t *testing.T) {
	var n int
	drifting := func(root, which string) (map[string][]byte, error) {
		n++
		return map[string][]byte{
			"dist/index.html": []byte(fmt.Sprintf("<html lang=\"en\">\n<!-- built at %d -->\n", n)),
		}, nil
	}

	var log bytes.Buffer
	err := run(tree(t), &log, drifting)
	if err == nil {
		t.Fatalf("the comparison accepted two builds that produced different bytes:\n%s", log.String())
	}
	for _, want := range []string{
		"dist/index.html",
		"they are not the same bytes",
		"first at line 2",
		"a time read from the clock",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q; it said:\n%s", want, log.String())
		}
	}
	if !strings.Contains(err.Error(), "1 of 1 produced file(s) differ") {
		t.Errorf("the error reads %q, which does not say how much moved", err)
	}
}

// A file only one of the builds wrote is a difference too, and it reads
// differently from a file whose bytes moved. A build that emits a page on every
// second run is not the same defect as one that emits a changing page.
func TestTheComparisonNoticesAFileOnlyOneBuildWrote(t *testing.T) {
	var n int
	extra := func(root, which string) (map[string][]byte, error) {
		n++
		out := map[string][]byte{"dist/index.html": []byte("<html lang=\"en\">\n")}
		if n == 2 {
			out["dist/extra.html"] = []byte("<html lang=\"en\">\n")
		}
		return out, nil
	}

	var log bytes.Buffer
	if err := run(tree(t), &log, extra); err == nil {
		t.Fatalf("the comparison accepted a file only one build wrote:\n%s", log.String())
	}
	if !strings.Contains(log.String(), "dist/extra.html: only the second build wrote it") {
		t.Errorf("the run does not name the file that appeared; it said:\n%s", log.String())
	}
}

// Two builds that wrote nothing are not a match. A comparison that passed over
// an empty pair would be green on a build that had stopped producing anything.
func TestTheComparisonRefusesTwoEmptyBuilds(t *testing.T) {
	empty := func(root, which string) (map[string][]byte, error) {
		return map[string][]byte{}, nil
	}

	err := run(tree(t), &bytes.Buffer{}, empty)
	if err == nil {
		t.Fatal("the comparison passed two builds that wrote nothing")
	}
	if !strings.Contains(err.Error(), "not a match") {
		t.Errorf("the error reads %q, which does not say why two empty directories are refused", err)
	}
}

// A build that refuses is a different failure from a build that differs, and the
// message says which one happened.
func TestRunSaysWhichBuildRefused(t *testing.T) {
	root := tree(t)
	if err := os.Remove(filepath.Join(root, "templates", "page.html.tmpl")); err != nil {
		t.Fatalf("removing the template: %v", err)
	}

	err := Run(root, &bytes.Buffer{})
	if err == nil {
		t.Fatal("Run passed a tree that does not build")
	}
	if !strings.Contains(err.Error(), "the first build refused") {
		t.Errorf("the error reads %q, which does not say the build was the problem", err)
	}
}
