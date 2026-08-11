// The suite over the build verb.
//
// Every case builds a tree it wrote itself in a temporary directory rather than
// the tree this file sits in. A case that read the real templates/ and content/
// would pass or fail on what those files happen to say today, which is a
// statement about the tree on the day it ran and not about the code under it.
// The one thing that does judge the real tree is the gate's build leg, and it
// asks a different question: whether this repository still builds.
//
// No case opens a window, binds a socket, reaches the network or needs anything
// that is not in the toolchain, so `go test ./...` is the whole harness.
package site

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Flowfin/site/internal/tokens"
)

// tree writes a minimal buildable tree under a temporary root and returns the
// root. The template is the fixture's own rather than the repository's, for the
// reason in the file comment.
func tree(t *testing.T, prose string) string {
	t.Helper()

	root := t.TempDir()
	mkdir(t, filepath.Join(root, TemplatesDir))
	write(t, filepath.Join(root, TemplatesDir, "page.html.tmpl"),
		"<title>{{ .Title }}</title>\n<h1>{{ .Title }}</h1>\n"+
			"{{- range .Paragraphs }}\n<p>{{ . }}</p>\n{{- end }}\n")
	mkdir(t, filepath.Join(root, ContentDir))
	write(t, filepath.Join(root, ContentDir, "index.txt"), prose)
	return root
}

func mkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("preparing %s: %v", dir, err)
	}
}

func write(t *testing.T, name, body string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(body), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

func read(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(b)
}

// A paragraph wrapped over several source lines is one paragraph, and the words
// at the wrap keep the space between them. Nothing in the compiler or in the
// markup notices when they stop doing so: the page still renders, and it renders
// two words run together in the middle of a sentence.
func TestBuildJoinsAWrappedParagraphIntoOneSentence(t *testing.T) {
	root := tree(t, "Fixture title\n\nA paragraph that the author\nwrapped across two lines.\n")

	written, err := Build(root, OutputDir, io.Discard)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(written) != 1 || written[0] != "dist/index.html" {
		t.Fatalf("Build reported %v, want [dist/index.html]", written)
	}

	got := read(t, filepath.Join(root, OutputDir, "index.html"))
	want := "<p>A paragraph that the author wrapped across two lines.</p>"
	if !strings.Contains(got, want) {
		t.Errorf("the page does not carry %q; it is:\n%s", want, got)
	}
}

// The first block is the title and every block after it is a paragraph, in the
// order they were written.
func TestBuildRendersTheTitleAndEveryParagraphInOrder(t *testing.T) {
	root := tree(t, "Fixture title\n\nFirst.\n\nSecond.\n\nThird.\n")

	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("Build: %v", err)
	}

	got := read(t, filepath.Join(root, OutputDir, "index.html"))
	for _, want := range []string{
		"<title>Fixture title</title>",
		"<h1>Fixture title</h1>",
		"<p>First.</p>",
		"<p>Second.</p>",
		"<p>Third.</p>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the page does not carry %q; it is:\n%s", want, got)
		}
	}
	if first, second := strings.Index(got, "<p>First.</p>"), strings.Index(got, "<p>Second.</p>"); first > second {
		t.Errorf("the paragraphs came out in the wrong order:\n%s", got)
	}
}

// outDir is taken relative to root unless it is absolute. The gate renders into
// a temporary directory it throws away and passes that path absolute; a build a
// person runs passes "dist" and means dist inside the tree being built. Lose the
// distinction and the second one writes wherever the process happens to be
// standing, which is a directory nobody asked about and, on the way, removes
// whatever was already there under that name.
func TestBuildResolvesARelativeOutputDirectoryAgainstTheRoot(t *testing.T) {
	root := tree(t, "Fixture title\n\nOne paragraph.\n")

	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("Build: %v", err)
	}

	inside := filepath.Join(root, OutputDir, "index.html")
	if _, err := os.Stat(inside); err != nil {
		t.Fatalf("nothing at %s: %v", inside, err)
	}
}

// An absolute outDir is used as it is, which is the shape the gate's build leg
// relies on to keep its throwaway output away from a reader's real one.
func TestBuildWritesAnAbsoluteOutputDirectoryWhereItWasAsked(t *testing.T) {
	root := tree(t, "Fixture title\n\nOne paragraph.\n")
	elsewhere := filepath.Join(t.TempDir(), OutputDir)

	written, err := Build(root, elsewhere, io.Discard)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if _, err := os.Stat(filepath.Join(elsewhere, "index.html")); err != nil {
		t.Fatalf("nothing at %s: %v", elsewhere, err)
	}
	if _, err := os.Stat(filepath.Join(root, OutputDir)); !os.IsNotExist(err) {
		t.Errorf("an absolute output directory still produced %s in the tree", filepath.Join(root, OutputDir))
	}
	if len(written) != 1 {
		t.Errorf("Build reported %v, want one file", written)
	}
}

// Prose reaches the page through html/template, so a bracket in a sentence is a
// bracket a reader sees rather than markup a browser runs. The roster sentences
// and the per-plugin prose take this same path later, which is why the property
// is tested with bytes that try rather than with the words that happen to be in
// content/ today.
func TestBuildRendersMarkupInTheProseAsText(t *testing.T) {
	root := tree(t, "Fixture title\n\n<script>alert(1)</script> & \"quoted\"\n")

	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("Build: %v", err)
	}

	got := read(t, filepath.Join(root, OutputDir, "index.html"))
	if strings.Contains(got, "<script>") {
		t.Fatalf("a script element reached the page:\n%s", got)
	}
	if !strings.Contains(got, "&lt;script&gt;alert(1)&lt;/script&gt; &amp; ") {
		t.Errorf("the prose was not escaped as text; the page is:\n%s", got)
	}
}

// The output directory holds what this run produced and nothing else. A file
// left behind by an earlier run is a page that is served and cannot be produced
// again, which is the drift the whole generator exists to prevent.
func TestBuildRemovesWhatAnEarlierRunLeftBehind(t *testing.T) {
	root := tree(t, "Fixture title\n\nOne paragraph.\n")
	stale := filepath.Join(root, OutputDir, "gone.html")
	mkdir(t, filepath.Join(root, OutputDir))
	write(t, stale, "a page from a build nobody can reproduce")

	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("%s survived the build", stale)
	}
}

// Everything under assets/ is served exactly as it was committed, including the
// bytes a text tool would tidy.
func TestBuildCopiesAssetsByteForByte(t *testing.T) {
	root := tree(t, "Fixture title\n\nOne paragraph.\n")
	mkdir(t, filepath.Join(root, AssetsDir, "nested"))
	body := "a:not(b) {\n\tcolor: red;\n}\n"
	write(t, filepath.Join(root, AssetsDir, "nested", "style.css"), body)

	written, err := Build(root, OutputDir, io.Discard)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	copied := filepath.Join(root, OutputDir, "nested", "style.css")
	if got := read(t, copied); got != body {
		t.Errorf("the asset came out as %q, want %q", got, body)
	}
	if len(written) != 2 || written[1] != "dist/nested/style.css" {
		t.Errorf("Build reported %v, want the page and dist/nested/style.css", written)
	}
}

// A run that copied nothing says which of the two reasons it was, because a
// silent run cannot be told from one that had nothing to do.
func TestBuildSaysWhenThereIsNoAssetsDirectory(t *testing.T) {
	root := tree(t, "Fixture title\n\nOne paragraph.\n")

	var log bytes.Buffer
	if _, err := Build(root, OutputDir, &log); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if !strings.Contains(log.String(), "no assets/ in the tree") {
		t.Errorf("the run did not say the directory was absent; it said:\n%s", log.String())
	}
}

// Prose with no title is refused rather than rendered as a page whose heading is
// empty, and the refusal names the file so the repair is obvious.
func TestBuildRefusesProseThatCarriesNoTitle(t *testing.T) {
	root := tree(t, "\n\n\n")

	_, err := Build(root, OutputDir, io.Discard)
	if err == nil {
		t.Fatal("Build accepted prose with no title")
	}
	if !strings.Contains(err.Error(), "carries no title line") {
		t.Errorf("the refusal reads %q, which does not say what was wrong", err)
	}
}

// A tree with no template is refused before anything is written, so a failed
// build does not leave a half-emptied output directory behind it.
func TestBuildRefusesATreeWithNoTemplate(t *testing.T) {
	root := tree(t, "Fixture title\n\nOne paragraph.\n")
	if err := os.Remove(filepath.Join(root, TemplatesDir, "page.html.tmpl")); err != nil {
		t.Fatalf("removing the template: %v", err)
	}
	mkdir(t, filepath.Join(root, OutputDir))
	kept := filepath.Join(root, OutputDir, "index.html")
	write(t, kept, "what the last good build served")

	if _, err := Build(root, OutputDir, io.Discard); err == nil {
		t.Fatal("Build accepted a tree with no page template")
	}
	if _, err := os.Stat(kept); err != nil {
		t.Errorf("a refused build removed %s: %v", kept, err)
	}
}

// The pinned copy of the design token file is a build input, so a malformed one
// is a build that stops rather than a page that renders a value nobody read.
// The tree this repository actually carries is what the gate's build leg reads;
// this case builds its own, so it judges the code rather than the file.
func TestBuildRefusesAMalformedCopyOfTheDesignTokens(t *testing.T) {
	root := tree(t, "Fixture title\n\nOne paragraph.\n")
	mkdir(t, filepath.Join(root, filepath.Dir(filepath.FromSlash(tokens.File))))
	write(t, filepath.Join(root, filepath.FromSlash(tokens.File)), "{\n")

	_, err := Build(root, OutputDir, io.Discard)
	if err == nil {
		t.Fatal("Build accepted a copy of the design tokens that does not parse")
	}
	if !strings.Contains(err.Error(), tokens.File) {
		t.Errorf("the refusal reads %q, which does not name the file that was wrong", err)
	}
}

// What the build reads is the copy in the tree, and the log says so with what
// was in it. A build that read the file and said nothing would be a build a
// reader cannot tell from one that skipped it.
func TestBuildSaysItReadThePinnedDesignTokensAndSaysWhenThereAreNone(t *testing.T) {
	root := tree(t, "Fixture title\n\nOne paragraph.\n")

	var absent bytes.Buffer
	if _, err := Build(root, OutputDir, &absent); err != nil {
		t.Fatalf("Build refused a tree with no pinned copy: %v", err)
	}
	if !strings.Contains(absent.String(), "no "+tokens.File+" in the tree") {
		t.Errorf("the build passed over an absent copy in silence:\n%s", absent.String())
	}

	mkdir(t, filepath.Join(root, filepath.Dir(filepath.FromSlash(tokens.File))))
	write(t, filepath.Join(root, filepath.FromSlash(tokens.File)),
		`{"shape":{"radius":{"value":12},"radius-small":{"value":8}}}`)

	var present bytes.Buffer
	if _, err := Build(root, OutputDir, &present); err != nil {
		t.Fatalf("Build refused a tree carrying a pinned copy: %v", err)
	}
	if !strings.Contains(present.String(), "read "+tokens.File+" (2 value(s))") {
		t.Errorf("the build does not say what it read:\n%s", present.String())
	}
}
