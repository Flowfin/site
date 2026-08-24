// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the strict read of a produced page.
//
// Every case is a page of the shape this repository actually produces, because
// what this package has to get right is the markup a template writes rather than
// the markup a browser has to survive. The pair matters as much as the refusal:
// a reader that refused a well-formed page would be worse than none, since the
// repair for it is somebody loosening the rule.
//
// No case opens a window, binds a socket, reaches the network or needs anything
// that is not in the toolchain.
package markup

import (
	"strings"
	"testing"
)

// page wraps a fragment in the smallest document that is otherwise whole, so a
// case is about the line it changed.
func page(body string) []byte {
	return []byte(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>A title</title>
  </head>
  <body>
` + body + `
  </body>
</html>
`)
}

func TestAPageThatBreaksNothingIsRefusedForNothing(t *testing.T) {
	whole := page(`    <main>
      <h1>A title</h1>
      <h2>A section</h2>
      <h3>Under it</h3>
      <p>A paragraph with <a href="/install/">a link</a> and a <br /> in it.</p>
      <img src="/icon.png" alt="The mark" width="32" height="32" />
      <ul>
        <li id="first">One</li>
        <li id="second">Two</li>
      </ul>
      <style>
        /* a rule with a < in it, which is a character and not a tag */
        .narrow { width: calc(100% - 2rem); }
      </style>
    </main>`)

	if got := Read(whole); len(got) != 0 {
		t.Errorf("a page that breaks nothing was refused: %v", got)
	}
}

// The mistake this whole package is for: one end tag left out, which every
// browser recovers from and nothing else notices.
func TestAnElementThatIsNeverClosedIsRefusedAndTheLineIsNamed(t *testing.T) {
	got := Of(Structure, Read(page(`    <main>
      <p>A paragraph nobody closed
      <p>And the next one.</p>
    </main>`)))
	if len(got) == 0 {
		t.Fatal("a page with an unclosed element was not refused")
	}
	// The reader meets the contradiction at the end tag that cannot be
	// right, and what it says is where the element it wanted was opened.
	// Both are in the message, because a line number alone would send
	// somebody to a line with nothing wrong on it.
	if !strings.Contains(got[0].Says, "<p>") || !strings.Contains(got[0].Says, "line 9") {
		t.Errorf("the refusal reads %q and does not name the element and where it was opened", got[0].Says)
	}
	if got[0].Line != 11 {
		t.Errorf("the refusal is on line %d, and the end tag that cannot be right is on line 11", got[0].Line)
	}
}

func TestCrossedElementsAndAStrayEndTagAreRefused(t *testing.T) {
	for name, body := range map[string]string{
		"elements that cross":                 `    <main><p><em>A phrase</p></em></main>`,
		"an end tag for nothing that is open": `    <main><p>A phrase</p></section></main>`,
		"a tag that never ends":               `    <main><p class="wide>A phrase</p></main>`,
	} {
		if got := Of(Structure, Read(page(body))); len(got) == 0 {
			t.Errorf("%s was not refused", name)
		}
	}
}

// A void element closes itself, written either way, and a self-closing tag on
// an element that is not void is not an unclosed element. Both are what a
// template actually writes, so a reader getting either wrong would refuse every
// page this repository produces.
func TestTheShapesAWellFormedTemplateWritesAreLeftAlone(t *testing.T) {
	for name, body := range map[string]string{
		"a void element with a slash":    `    <main><img src="/a.png" alt="" width="1" height="1" /></main>`,
		"a void element without one":     `    <main><br><hr></main>`,
		"an attribute with no value":     `    <main><input aria-label="Search" disabled /></main>`,
		"a value in single quotes":       `    <main><p class='wide'>A phrase</p></main>`,
		"a value in no quotes":           `    <main><p class=wide>A phrase</p></main>`,
		"a comment carrying a tag in it": `    <main><!-- <p>not markup</p> --><p>A phrase</p></main>`,
		"an element closed on itself":    `    <main><span class="x" /><p>A phrase</p></main>`,
	} {
		if got := Read(page(body)); len(got) != 0 {
			t.Errorf("%s was refused: %v", name, got)
		}
	}
}

func TestADuplicateIdentifierIsRefusedAndBothLinesAreNamed(t *testing.T) {
	got := Of(Identity, Read(page(`    <main>
      <section id="install">One</section>
      <section id="install">Two</section>
    </main>`)))
	if len(got) != 1 {
		t.Fatalf("%d problem(s) for one duplicate: %v", len(got), got)
	}
	if !strings.Contains(got[0].Says, `"install"`) || !strings.Contains(got[0].Says, "line 9") {
		t.Errorf("the refusal reads %q and has to name the identifier and where it was first used", got[0].Says)
	}
	if got := Of(Identity, Read(page(`    <main><p id="">A phrase</p></main>`))); len(got) != 1 {
		t.Errorf("an empty identifier was not refused: %v", got)
	}
}

func TestAHeadingLevelThatSkipsIsRefusedAndOneThatFallsBackIsNot(t *testing.T) {
	if got := Of(Heading, Read(page(`    <main><h1>A</h1><h3>B</h3></main>`))); len(got) != 1 {
		t.Errorf("a level skipped from 1 to 3 was not refused: %v", got)
	}
	// Falling back to a level already used is how a document returns to the
	// next section, and a rule refusing it would refuse every page with two
	// sections in it.
	if got := Of(Heading, Read(page(`    <main><h1>A</h1><h2>B</h2><h3>C</h3><h2>D</h2></main>`))); len(got) != 0 {
		t.Errorf("a heading returning to a level already used was refused: %v", got)
	}
}

func TestAnImageWithNoAlternativeTextIsRefusedAndAnEmptyOneIsNot(t *testing.T) {
	if got := Of(Alt, Read(page(`    <main><img src="/a.png" width="1" height="1" /></main>`))); len(got) != 1 {
		t.Errorf("an image with no alt was not refused: %v", got)
	}
	if got := Of(Alt, Read(page(`    <main><img src="/a.png" alt="" width="1" height="1" /></main>`))); len(got) != 0 {
		t.Errorf("an image saying nothing deliberately was refused: %v", got)
	}
}

func TestAControlWithNothingNamingItIsRefused(t *testing.T) {
	if got := Of(Label, Read(page(`    <main><input type="search" /></main>`))); len(got) != 1 {
		t.Errorf("a control with no name was not refused: %v", got)
	}
	for name, body := range map[string]string{
		"a control named by aria-label": `    <main><input type="search" aria-label="Search the site" /></main>`,
		"a hidden field":                `    <main><input type="hidden" name="page" value="2" /></main>`,
	} {
		if got := Of(Label, Read(page(body))); len(got) != 0 {
			t.Errorf("%s was refused: %v", name, got)
		}
	}
}

// A page that does not parse is reported once and not four times. Everything
// after a broken tag is a guess about what the author meant, and a page buried
// under guesses is a page nobody reads the first line of.
func TestABrokenPageIsReportedOnceRatherThanGuessedAt(t *testing.T) {
	got := Read(page(`    <main>
      </section>
      <img src="/a.png" />
      <h1>A</h1><h4>B</h4>
    </main>`))
	if len(got) != 1 {
		t.Fatalf("%d problem(s) reported past the first broken tag: %v", len(got), got)
	}
	if got[0].Kind != Structure {
		t.Errorf("the one problem reported is %s rather than the structure it stopped at", got[0].Kind)
	}
}
