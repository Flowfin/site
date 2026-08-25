// SPDX-License-Identifier: AGPL-3.0-or-later

package site

import (
	"bytes"
	"flag"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The bytes of every file one build wrote, committed beside the tree it was
// built from, so that the question a reviewer asks about a change to this
// package is which bytes it moved rather than whether the author says it moved
// any.
//
// The tree is a fixture and the template is not. A fixture tree means the
// goldens move when the generator moves and never when the real roster, the
// real prose or the real token file does, so a page whose data changed
// yesterday does not red a change to a template today. The template is copied
// out of the repository instead, because it is the one input every page is
// rendered through: a golden set rendered through a copy of it would go on
// agreeing with itself after the real frame had changed, which is the one
// change these files exist to catch.
//
// The fixture roster carries the cases the real one does not have all of at
// once: the shortest identifier and the longest, a sentence with markup
// characters in it, a sentence outside ASCII, a row in each declared state, a
// repository whose only publication is a single prerelease, and a row declaring
// the shell state for a repository that has published a finished release, which
// is the disagreement the state is computed to settle.
const (
	fixtureTree  = "testdata/tree"
	goldenDir    = "testdata/golden"
	goldenSuffix = ".golden"
)

// rewriteGoldens is the update path, and it is a flag rather than a condition
// the suite decides for itself. Running the suite compares; rewriting is a
// second command somebody types, and what it produces lands in a pull request
// as a diff rather than as a green run nobody looked at.
//
//	go test ./internal/site -run TestEveryFileTheBuildWroteMatchesItsGolden -update
var rewriteGoldens = flag.Bool("update", false,
	"rewrite the golden files from what the build produced, instead of comparing against them")

// buildFixture renders the fixture tree and answers with what was written,
// keyed by the path inside the output directory. edit is applied to the
// assembled tree before the build, which is how a case moves one input and asks
// what moved with it.
func buildFixture(t *testing.T, edit func(t *testing.T, root string)) map[string][]byte {
	t.Helper()

	root := t.TempDir()
	copyTree(t, fixtureTree, root)
	copyFile(t,
		filepath.Join("..", "..", TemplatesDir, "page.html.tmpl"),
		filepath.Join(root, TemplatesDir, "page.html.tmpl"))
	if edit != nil {
		edit(t, root)
	}

	written, err := Build(root, OutputDir, io.Discard)
	if err != nil {
		t.Fatalf("building the fixture tree: %v", err)
	}
	if len(written) == 0 {
		t.Fatal("the build wrote nothing, so there is nothing for a golden file to be about")
	}

	produced := make(map[string][]byte, len(written))
	for _, w := range written {
		rel := strings.TrimPrefix(w, OutputDir+"/")
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(w)))
		if err != nil {
			t.Fatalf("reading what the build wrote at %s: %v", w, err)
		}
		produced[rel] = b
	}
	return produced
}

// differences reports every file that does not match the bytes committed for
// it and every golden with no file behind it, and it writes nothing at all.
// The writing is refresh below, which is reached only through the flag.
func differences(dir string, produced map[string][]byte) []string {
	var found []string

	for _, rel := range sorted(produced) {
		name := goldenPath(dir, rel)
		want, err := os.ReadFile(name)
		switch {
		case os.IsNotExist(err):
			found = append(found, rel+": the build wrote this file and there is no golden for it")
			continue
		case err != nil:
			found = append(found, rel+": the golden beside it could not be read: "+err.Error())
			continue
		}
		if !bytes.Equal(want, produced[rel]) {
			found = append(found, rel+": "+firstDifference(want, produced[rel]))
		}
	}

	held, err := goldensHeld(dir)
	if err != nil {
		return append(found, "the golden files could not be listed: "+err.Error())
	}
	for _, rel := range held {
		if _, ok := produced[rel]; !ok {
			found = append(found, rel+": there is a golden for this file and the build wrote no such file")
		}
	}
	return found
}

// goldensHeld is what is committed, named the way the produced file it is about
// is named. The set is walked rather than listed at the top, because the golden
// files sit at the same paths the output does and a rule that read one level
// would stop seeing every page below the first directory.
func goldensHeld(dir string) ([]string, error) {
	var held []string
	err := filepath.WalkDir(dir, func(name string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir() || !strings.HasSuffix(name, goldenSuffix):
			return nil
		}
		rel, err := filepath.Rel(dir, strings.TrimSuffix(name, goldenSuffix))
		if err != nil {
			return err
		}
		held = append(held, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(held)
	return held, nil
}

// refresh writes what the build produced over the golden files and removes the
// ones nothing produces any more, so that the committed set is what one build
// wrote rather than that plus whatever an earlier one left behind.
func refresh(t *testing.T, dir string, produced map[string][]byte) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", dir, err)
	}
	held, err := goldensHeld(dir)
	if err != nil {
		t.Fatalf("listing %s: %v", dir, err)
	}
	for _, rel := range held {
		if _, ok := produced[rel]; !ok {
			if err := os.Remove(goldenPath(dir, rel)); err != nil {
				t.Fatalf("removing the golden for %s, which nothing produces any more: %v", rel, err)
			}
		}
	}
	for _, rel := range sorted(produced) {
		name := goldenPath(dir, rel)
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatalf("creating the directory for the golden for %s: %v", rel, err)
		}
		if err := os.WriteFile(name, produced[rel], 0o644); err != nil {
			t.Fatalf("writing the golden for %s: %v", rel, err)
		}
	}
}

// goldenPath is where the bytes of one produced file are committed: the path
// the build wrote it to, under the golden directory, with a suffix on the end.
// The committed set has the shape of the output rather than a flattened
// spelling of it, so a name is read as the page it is about and no separator
// has to be invented for a character a path may not carry.
func goldenPath(dir, rel string) string {
	return filepath.Join(dir, filepath.FromSlash(rel)+goldenSuffix)
}

func sorted(produced map[string][]byte) []string {
	names := make([]string, 0, len(produced))
	for rel := range produced {
		names = append(names, rel)
	}
	sort.Strings(names)
	return names
}

// firstDifference says where two versions of one file part company, in one
// line, because a page is thousands of bytes and a failure printing both of
// them is a failure nobody reads.
func firstDifference(want, got []byte) string {
	wantLines := strings.Split(string(want), "\n")
	gotLines := strings.Split(string(got), "\n")
	for i := 0; i < len(wantLines) && i < len(gotLines); i++ {
		if wantLines[i] != gotLines[i] {
			return "line " + itoa(i+1) + " reads " + short(gotLines[i]) +
				" and the golden carries " + short(wantLines[i])
		}
	}
	return "the two agree for " + itoa(min(len(wantLines), len(gotLines))) +
		" line(s) and then one of them ends: the build wrote " + itoa(len(gotLines)) +
		" line(s) and the golden carries " + itoa(len(wantLines))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// copyTree copies a directory into another, which is how the fixture reaches a
// directory a case may write in without a case ever writing into the fixture.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()

	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("reading the fixture at %s: %v", src, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("creating %s: %v", dst, err)
	}
	for _, e := range entries {
		from, to := filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())
		if e.IsDir() {
			copyTree(t, from, to)
			continue
		}
		copyFile(t, from, to)
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()

	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("reading %s: %v", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("creating the directory for %s: %v", dst, err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		t.Fatalf("writing %s: %v", dst, err)
	}
}

// replaceInFile is the one-word edit the near misses are made of. It refuses a
// word it did not find, because a case that changed nothing and then asserted
// that nothing moved would pass for the wrong reason.
func replaceInFile(t *testing.T, name, from, to string) {
	t.Helper()

	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	if !bytes.Contains(b, []byte(from)) {
		t.Fatalf("%s does not carry %q, so changing it would move nothing and this case would prove nothing", name, from)
	}
	if err := os.WriteFile(name, bytes.Replace(b, []byte(from), []byte(to), 1), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

func TestEveryFileTheBuildWroteMatchesItsGolden(t *testing.T) {
	produced := buildFixture(t, nil)

	if *rewriteGoldens {
		refresh(t, goldenDir, produced)
		t.Logf("rewrote %d golden file(s) from what the build produced", len(produced))
		return
	}

	if found := differences(goldenDir, produced); len(found) > 0 {
		t.Fatalf("%d file(s) do not match what is committed beside them:\n  %s\n"+
			"Where the change is intended, rewrite them deliberately and put the diff in the pull request body:\n"+
			"  go test ./internal/site -run TestEveryFileTheBuildWroteMatchesItsGolden -update",
			len(found), strings.Join(found, "\n  "))
	}
}

// Every page is rendered through one template, so a word changed in it moves
// every page and discriminates nothing between them. That is what this asserts
// rather than what it works around: what the assertion is worth is the other
// side of it, that the files which are not rendered through the template do not
// move, so a golden set reddening as a block is a template change and a golden
// set reddening in one place is not.
func TestOneWordChangedInTheTemplateRedsEveryPageAndNothingElse(t *testing.T) {
	before := buildFixture(t, nil)
	after := buildFixture(t, func(t *testing.T, root string) {
		replaceInFile(t, filepath.Join(root, TemplatesDir, "page.html.tmpl"),
			"Skip to the content", "Jump to the content")
	})

	if len(before) != len(after) {
		t.Fatalf("the two builds wrote a different number of files, %d and %d, so what follows would be comparing two different sites",
			len(before), len(after))
	}

	var moved, still []string
	for _, rel := range sorted(before) {
		page := strings.HasSuffix(rel, ".html")
		same := bytes.Equal(before[rel], after[rel])
		if same {
			still = append(still, rel)
		} else {
			moved = append(moved, rel)
		}
		if page && same {
			t.Errorf("%s is a page and one word changed in the template did not move it, so its golden would not catch a template change", rel)
		}
		if !page && !same {
			t.Errorf("%s is not rendered through the template and moved when a word in it changed", rel)
		}
	}
	t.Logf("%d file(s) moved and %d did not: %s", len(moved), len(still), strings.Join(still, ", "))
}

// The change that does discriminate is one to a single page's own input. The
// not-found page is the one whose prose reaches exactly one produced file: it
// is written from one file in the content directory and it is deliberately not
// listed in the sitemap, so nothing else in the output carries a byte of it.
func TestOneWordChangedInOnePagesOwnInputRedsThatPageAlone(t *testing.T) {
	before := buildFixture(t, nil)
	after := buildFixture(t, func(t *testing.T, root string) {
		replaceInFile(t, filepath.Join(root, filepath.FromSlash(NotFoundFile)),
			"page", "leaf")
	})

	var moved []string
	for _, rel := range sorted(before) {
		if !bytes.Equal(before[rel], after[rel]) {
			moved = append(moved, rel)
		}
	}
	if len(moved) != 1 || moved[0] != NotFoundPath {
		t.Fatalf("one word changed in %s moved %v, and the only file it may move is %s",
			NotFoundFile, moved, NotFoundPath)
	}
}

// The update path is a command somebody types. Running the suite compares and
// writes nothing, which is asserted against a copy of the committed set that
// has been made wrong on purpose: the comparison reports it and leaves the
// wrong bytes exactly where they are, and only the rewriting path repairs them.
func TestTheComparisonWritesNothingAndTheUpdatePathIsAskedForByName(t *testing.T) {
	if f := flag.Lookup("update"); f == nil {
		t.Fatal("there is no update flag, so the way to rewrite the golden files is not a thing somebody asks for by name")
	} else if f.DefValue != "false" {
		t.Fatalf("the update flag defaults to %q, so rewriting the golden files is what the suite does when nobody asked", f.DefValue)
	}

	dir := t.TempDir()
	produced := buildFixture(t, nil)
	refresh(t, dir, produced)
	if found := differences(dir, produced); len(found) > 0 {
		t.Fatalf("a set just written does not match what wrote it: %s", strings.Join(found, "\n  "))
	}

	spoiled := goldenPath(dir, IndexPath)
	wrong := append([]byte("<!-- this is not what the build wrote -->\n"), produced[IndexPath]...)
	if err := os.WriteFile(spoiled, wrong, 0o644); err != nil {
		t.Fatalf("making one golden wrong on purpose: %v", err)
	}

	found := differences(dir, produced)
	if len(found) != 1 || !strings.HasPrefix(found[0], IndexPath+":") {
		t.Fatalf("the comparison reported %v, and what is wrong is %s alone", found, IndexPath)
	}
	left, err := os.ReadFile(spoiled)
	if err != nil {
		t.Fatalf("reading the golden the comparison looked at: %v", err)
	}
	if !bytes.Equal(left, wrong) {
		t.Fatal("the comparison rewrote the golden it was comparing against, so a run of the suite is an update nobody asked for")
	}

	refresh(t, dir, produced)
	if found := differences(dir, produced); len(found) > 0 {
		t.Fatalf("the update path did not repair what it was asked to: %s", strings.Join(found, "\n  "))
	}
}
