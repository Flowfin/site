// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the link walk.
//
// Every case is a pair: the reference that has to be refused, and the reference
// one character away from it that has to pass. A walk that refused everything
// would be as useless as one that refused nothing, and only the pair tells them
// apart.
package link

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// site is what a small produced tree looks like: a landing page, a page one level
// down at a directory address, and a file that is not markup.
func fixture(landing, install string) ([]Page, map[string]bool) {
	pages := []Page{
		{Name: "index.html", Body: []byte(landing)},
		{Name: "install/index.html", Body: []byte(install)},
	}
	produced := map[string]bool{
		"index.html":         true,
		"install/index.html": true,
		"style.css":          true,
	}
	return pages, produced
}

const installPage = `<!DOCTYPE html>
<html lang="en">
  <head>
    <title>Install</title>
  </head>
  <body>
    <main id="content"><h1>Install</h1></main>
  </body>
</html>
`

func landing(body string) string {
	return `<!DOCTYPE html>
<html lang="en">
  <head>
    <title>A title</title>
  </head>
  <body>
    <main id="content">
` + body + `
    </main>
  </body>
</html>
`
}

// The failure this leg exists for. A page was renamed and the reference to it
// was not, so the build succeeds and the link goes nowhere.
func TestADeadInternalLinkIsRefusedAndNamed(t *testing.T) {
	pages, produced := fixture(landing(`      <a href="/instal/">Install</a>`), installPage)

	got := Decide(pages, produced)
	if len(got) != 1 {
		t.Fatalf("the walk returned %v, want exactly one refusal", got)
	}
	for _, want := range []string{"dist/index.html", `href="/instal/"`, "dist/instal/index.html", "the build did not write"} {
		if !strings.Contains(got[0], want) {
			t.Errorf("the refusal does not say %q; it says %q", want, got[0])
		}
	}

	pages, produced = fixture(landing(`      <a href="/install/">Install</a>`), installPage)
	if got := Decide(pages, produced); len(got) != 0 {
		t.Errorf("the walk refused the same link spelled correctly: %v", got)
	}
}

// The other half. The page resolves and the place inside it does not, which is
// what a renamed heading leaves behind.
func TestADeadFragmentIsRefusedAndNamed(t *testing.T) {
	pages, produced := fixture(landing(`      <a href="/install/#contents">Install</a>`), installPage)

	got := Decide(pages, produced)
	if len(got) != 1 {
		t.Fatalf("the walk returned %v, want exactly one refusal", got)
	}
	for _, want := range []string{"dist/index.html", "#contents", "dist/install/index.html", "no element with that id"} {
		if !strings.Contains(got[0], want) {
			t.Errorf("the refusal does not say %q; it says %q", want, got[0])
		}
	}

	pages, produced = fixture(landing(`      <a href="/install/#content">Install</a>`), installPage)
	if got := Decide(pages, produced); len(got) != 0 {
		t.Errorf("the walk refused a fragment whose target exists: %v", got)
	}
}

// A fragment with no address before it points into the page it was written on,
// which is what the link past the navigation is.
func TestASamePageFragmentIsResolvedAgainstItsOwnPage(t *testing.T) {
	pages, produced := fixture(landing(`      <a href="#content">Skip</a>`), installPage)
	if got := Decide(pages, produced); len(got) != 0 {
		t.Errorf("the walk refused a fragment into the page carrying it: %v", got)
	}

	pages, produced = fixture(landing(`      <a href="#contents">Skip</a>`), installPage)
	if got := Decide(pages, produced); len(got) != 1 {
		t.Errorf("the walk returned %v for a same-page fragment that points at nothing", got)
	}
}

// What the walk has no business deciding. Every one of these would be a refusal
// somebody switches the leg off over.
func TestTheWalkLeavesWhatIsNotItsBusinessAlone(t *testing.T) {
	for name, markup := range map[string]string{
		"a link to another project":          `      <a href="https://jellyfin.org/">The server</a>`,
		"a reference with no scheme":         `      <a href="//example.invalid/x">Elsewhere</a>`,
		"an address to write to":             `      <a href="mailto:nobody@example.invalid">Write</a>`,
		"an inline image":                    `      <img src="data:image/gif;base64,R0lGODlhAQABAAAAACw=" alt="" />`,
		"a produced file that is not a page": `      <link rel="stylesheet" href="/style.css" />`,
		"a relative reference":               `      <a href="install/">Install</a>`,
		"an address carrying a query":        `      <a href="/install/?from=here">Install</a>`,
		"an empty reference":                 `      <a href="">Nowhere in particular</a>`,
	} {
		pages, produced := fixture(landing(markup), installPage)
		if got := Decide(pages, produced); len(got) != 0 {
			t.Errorf("the walk refused %s: %v", name, got)
		}
	}
}

// A relative reference is resolved against the page it was written on rather
// than against the site root, so a page one level down that points at a sibling
// is judged as the host would serve it.
func TestARelativeReferenceIsResolvedAgainstItsOwnPage(t *testing.T) {
	pages, produced := fixture(landing(`      <a href="/install/">Install</a>`), strings.Replace(installPage,
		`<h1>Install</h1>`, `<h1>Install</h1><a href="../style.css">Style</a>`, 1))
	if got := Decide(pages, produced); len(got) != 0 {
		t.Errorf("the walk refused a sibling reached from one level down: %v", got)
	}

	pages, produced = fixture(landing(`      <a href="/install/">Install</a>`), strings.Replace(installPage,
		`<h1>Install</h1>`, `<h1>Install</h1><a href="style.css">Style</a>`, 1))
	got := Decide(pages, produced)
	if len(got) != 1 {
		t.Fatalf("the walk returned %v for a reference that resolves one level too deep", got)
	}
	if !strings.Contains(got[0], "dist/install/style.css") {
		t.Errorf("the refusal does not name where the reference resolved to; it says %q", got[0])
	}
}

// The whole run, over a tree it builds itself: the walk reads what a build
// wrote rather than what a fixture handed it.
//
// The tree is a temporary one rather than this repository. The leg runs over
// this repository every time the gate verb does, and a suite that also read it
// would red the test leg first and stop the run before the leg it is about was
// reached, so a refusal here would be recorded under the wrong name.
func TestRunWalksWhatABuildWrote(t *testing.T) {
	var log bytes.Buffer
	if err := Run(buildable(t, `      <p><a href="#content">Skip to the content</a></p>`), &log); err != nil {
		t.Fatalf("the walk refused a tree whose references resolve: %v\n%s", err, log.String())
	}
	for _, want := range []string{"1 reference(s) over 1 page(s)", "resolves to a file the build wrote"} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q; it said:\n%s", want, log.String())
		}
	}

	log.Reset()
	err := Run(buildable(t, `      <p><a href="#contents">Skip to the content</a></p>`), &log)
	if err == nil {
		t.Fatalf("the walk passed a tree whose one reference points at nothing:\n%s", log.String())
	}
	if !strings.Contains(log.String(), "dist/index.html: line") || !strings.Contains(log.String(), "#contents") {
		t.Errorf("the run does not name the page and the fragment; it said:\n%s", log.String())
	}
}

// buildable writes the smallest tree the build verb accepts, with body dropped
// into the page template.
func buildable(t *testing.T, body string) string {
	t.Helper()

	root := t.TempDir()
	write := func(name, content string) {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("preparing %s: %v", name, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	write("templates/page.html.tmpl", `<!DOCTYPE html>
<html lang="en">
  <head>
    <title>{{ .Title }}</title>
  </head>
  <body>
    <main id="content">
      <h1>{{ .Title }}</h1>
`+body+`
    </main>
  </body>
</html>
`)
	write("content/index.txt", "A title\n\ndescription: What this fixture page is.\n\nOne paragraph.\n")
	return root
}

// A tree whose build refuses has nothing to walk, and the run says which of the
// two happened rather than reporting a site with no dead links in it.
func TestRunRefusesATreeItCannotBuild(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "content"), 0o755); err != nil {
		t.Fatalf("preparing the tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "content", "index.txt"), []byte("A title\n"), 0o644); err != nil {
		t.Fatalf("writing the prose: %v", err)
	}

	err := Run(root, &bytes.Buffer{})
	if err == nil {
		t.Fatal("the walk passed a tree that does not build")
	}
	if !strings.Contains(err.Error(), "the build refused") {
		t.Errorf("the error reads %q, which does not say the build was the problem", err)
	}
}
