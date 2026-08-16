// The suite over the design system page.
//
// The page renders a file this repository does not author, so the cases below
// are about what happens when that file moves rather than about what it holds
// today. A case asserting the real copy's 154 values would be a statement about
// the copy on the day it ran, and it would go red the next time somebody
// upstream adds a colour, which is the one event this page is supposed to
// survive without anybody editing it.
//
// So every fixture is a token file this suite wrote, small enough to read, and
// each case moves one thing in it and asks what the page did.
package site

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Flowfin/site/internal/budget"
	"github.com/Flowfin/site/internal/tokens"
)

// designTemplate renders the parts of the page this suite is about, so a case
// reads what the build put there rather than whatever the repository's frame
// carries today.
const designTemplate = `<title>{{ .Title }}</title>
<meta name="description" content="{{ .Description }}" />
<h1>{{ .Title }}</h1>
{{- range .Paragraphs }}
<p>{{ . }}</p>
{{- end }}
{{- range .Shows }}
<h2>{{ .Shows }}</h2>
{{- range .Samples }}
<li><span style="{{ .Drawn }}">{{ .Text }}</span> {{ .Says }}</li>
{{- end }}
{{- end }}
{{- range .Budgets }}
<h2>{{ .Whose }}</h2>
{{- range .Rows }}
<tr><td>{{ .Name }}</td><td>{{ .Limit }}</td><td>{{ .Means }}</td></tr>
{{- end }}
{{- end }}
{{- if .Values }}
<p>{{ .Reading }}</p>
{{- range .Values }}
<dt>{{ .Path }}</dt><dd>{{ .Value }}{{ if .Drawn }} <span style="{{ .Drawn }}">{{ .Value }}</span>{{ end }}</dd>
{{- end }}
{{- end }}
`

const designProse = `The design system

description: Every value this site is built from.

One paragraph.
`

// designTokens is the smallest token file that reaches every group the page
// draws: a colour pair for the corner, a preset with a ring and an accent in
// both schemes, one role at one distance, one radius and one budget number.
// Four of its leaves are the file's own sentences about itself and twelve are
// values.
const designTokens = `{
  "what": "This file is what the suite builds a page from.",
  "surface": {
    "tokens": {
      "raise-2": { "light": { "srgb": "#F0F0F2" } },
      "ink": { "light": { "srgb": "#0E0E11" } }
    }
  },
  "focus": {
    "presets": {
      "standard": {
        "missing": "nothing",
        "ring-width": 3,
        "dark": { "accent": { "srgb": "#5B9CFF" } },
        "light": { "accent": { "srgb": "#0A6CE8" } }
      }
    }
  },
  "type": {
    "weight": "The CSS numeric weight, 100 to 900. A platform rounds to the nearest it has.",
    "distances": {
      "telephone": { "roles": { "tile": { "size": 14, "weight": 540 } } }
    }
  },
  "shape": { "radius": { "value": 12 } },
  "budget": {
    "numbers": {
      "focus-change": {
        "what": "Key press to the first moved scanline.",
        "limit": 80,
        "unit": "ms",
        "comparison": "below",
        "why": "Above that the remote feels sluggish."
      }
    }
  }
}
`

// designTree writes a tree that produces the landing page and the design system
// page, from the token file a case hands it.
func designTree(t *testing.T, tokenFile string) string {
	t.Helper()
	root := t.TempDir()
	mkdir(t, filepath.Join(root, TemplatesDir))
	mkdir(t, filepath.Join(root, ContentDir))
	mkdir(t, filepath.Join(root, filepath.Dir(filepath.FromSlash(tokens.File))))
	write(t, filepath.Join(root, TemplatesDir, "page.html.tmpl"), designTemplate)
	write(t, filepath.Join(root, ContentDir, "index.txt"), headIndex)
	write(t, filepath.Join(root, filepath.FromSlash(DesignSystemProse)), designProse)
	write(t, filepath.Join(root, filepath.FromSlash(tokens.File)), tokenFile)
	return root
}

// designPage builds a tree and returns the design system page it produced.
func designPage(t *testing.T, tokenFile string) string {
	t.Helper()
	root := designTree(t, tokenFile)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("building: %v", err)
	}
	return read(t, filepath.Join(root, OutputDir, filepath.FromSlash(DesignSystemPath)))
}

// designRefusal builds a tree that must be refused and returns what the refusal
// said.
func designRefusal(t *testing.T, tokenFile string) string {
	t.Helper()
	_, err := Build(designTree(t, tokenFile), OutputDir, io.Discard)
	if err == nil {
		t.Fatal("the build accepted a token file the page cannot be drawn from")
	}
	return err.Error()
}

// listsEntry answers whether the page listed a path.
func listsEntry(listed []string, want string) bool {
	for _, l := range listed {
		if l == want {
			return true
		}
	}
	return false
}

// entries returns the paths the page listed, in the order it listed them.
func entries(page string) []string {
	var out []string
	rest := page
	for {
		at := strings.Index(rest, "<dt>")
		if at < 0 {
			return out
		}
		rest = rest[at+len("<dt>"):]
		end := strings.Index(rest, "</dt>")
		if end < 0 {
			return out
		}
		out = append(out, rest[:end])
		rest = rest[end:]
	}
}

// The rule is about the value and never about the key, and the file that forces
// that is the real one: it carries a `weight` that is a sentence and a `weight`
// that is a number, so a rule reading the last segment gets one of the two
// wrong whichever side it picks.
func TestTheValueListKeepsTheNumberAndDropsTheSentenceUnderOneKey(t *testing.T) {
	page := designPage(t, designTokens)

	listed := entries(page)
	for _, want := range []string{"type.distances.telephone.roles.tile.weight", "type.distances.telephone.roles.tile.size"} {
		if !listsEntry(listed, want) {
			t.Errorf("the page lists no entry for %s, and it is a value", want)
		}
	}
	for _, unwanted := range []string{"type.weight", "what", "budget.numbers.focus-change.what"} {
		if listsEntry(listed, unwanted) {
			t.Errorf("the page lists %s, and it is the file's own prose rather than a value", unwanted)
		}
	}
}

// The three counts on the page add up to what the file holds. That is what lets
// a reader audit the split without reading the source that takes it, so a rule
// that quietly dropped a leaf would show up here as a total that does not add
// up.
func TestThePageStatesHowMuchOfTheFileItListedAndHowMuchItDidNot(t *testing.T) {
	page := designPage(t, designTokens)

	for _, want := range []string{
		"holds 16 leaves",
		"lists the 12 that are values",
		"Another 4 are sentences the file writes about itself",
		"no leaf holds nothing at all",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page does not say %q, and the counts are what make the rule auditable", want)
		}
	}
	if got := len(entries(page)); got != 12 {
		t.Errorf("the page says it listed 12 values and listed %d", got)
	}
}

// A leaf holding nothing at all is counted as carrying no value rather than as a
// value spelled `null`. The pinned copy carries none today, which is exactly why
// the case is written against a fixture: the day the published file gains one,
// the page has already decided what to do with it.
func TestALeafHoldingNothingIsCountedAsCarryingNoValue(t *testing.T) {
	page := designPage(t, strings.Replace(designTokens,
		`"shape": { "radius": { "value": 12 } },`,
		`"shape": { "radius": { "value": 12, "note": null } },`, 1))

	if !strings.Contains(page, "one leaf holds nothing at all") {
		t.Error("the page does not count the leaf holding nothing, so it was read as a value or dropped in silence")
	}
	if listsEntry(entries(page), "shape.radius.note") {
		t.Error("the page lists the leaf holding nothing as though it were a value")
	}
}

// The other half of the rule, which catches nothing in the pinned copy today. A
// leaf whose prose does not end at the leaf's end is still prose, and a rule
// asking only whether the value ends in a full stop reads it as a value and puts
// a paragraph in the value list.
func TestASentenceThatDoesNotEndTheLeafIsStillProse(t *testing.T) {
	page := designPage(t, strings.Replace(designTokens,
		`"missing": "nothing",`,
		`"missing": "nothing", "note": "Two sentences. And no full stop at the end",`, 1))

	if listsEntry(entries(page), "focus.presets.standard.note") {
		t.Error("the page lists a leaf carrying prose, so the rule read only the end of the value")
	}
	if !strings.Contains(page, "Another 5 are sentences") {
		t.Error("the page did not count the leaf carrying prose as one")
	}
}

// The event this page has to survive without anybody editing it. A value added
// upstream appears, a sentence added upstream does not, and both counts move by
// one.
func TestAValueAddedUpstreamAppearsAndASentenceAddedUpstreamDoesNot(t *testing.T) {
	before := designPage(t, designTokens)
	after := designPage(t, strings.Replace(designTokens,
		`"shape": { "radius": { "value": 12 } },`,
		`"shape": { "radius": { "value": 12 }, "gutter": { "value": 16, "role": "What a new group would say about itself." } },`, 1))

	was, now := len(entries(before)), len(entries(after))
	if now != was+1 {
		t.Fatalf("the file gained one value and one sentence and the page went from %d entries to %d", was, now)
	}
	if !listsEntry(entries(after), "shape.gutter.value") {
		t.Error("the page does not list the value the file gained")
	}
	if listsEntry(entries(after), "shape.gutter.role") {
		t.Error("the page lists the sentence the file gained")
	}
	for _, want := range []string{"holds 18 leaves", "lists the 13 that are values", "Another 5 are sentences"} {
		if !strings.Contains(after, want) {
			t.Errorf("the page does not say %q after the file grew", want)
		}
	}
}

// The value on the page and the value the file carries are one string. A page
// rendering a token from anywhere but the copy would pass every other case here
// and be wrong in the one way this page exists to prevent.
func TestAValueChangedInTheCopyChangesExactlyThatEntryOnThePage(t *testing.T) {
	before := designPage(t, designTokens)
	after := designPage(t, strings.Replace(designTokens, `"value": 12`, `"value": 13`, 1))

	if !strings.Contains(before, "<dt>shape.radius.value</dt><dd>12</dd>") {
		t.Fatal("the page did not carry the radius the file gives it")
	}
	if !strings.Contains(after, "<dt>shape.radius.value</dt><dd>13</dd>") {
		t.Error("the radius moved in the copy and did not move on the page")
	}
	if strings.Contains(after, "<dd>12</dd>") {
		t.Error("the page still carries the value the copy no longer has")
	}
	if len(entries(before)) != len(entries(after)) {
		t.Error("changing one value changed how many entries the page carries")
	}
}

// A colour is drawn with itself. The swatch and the value beside it are the same
// string, so a swatch cannot disagree with the number next to it.
func TestAColourIsDrawnWithTheColourItStates(t *testing.T) {
	page := designPage(t, designTokens)

	want := `<dd>#5B9CFF <span style="background:#5B9CFF;color:#5B9CFF">#5B9CFF</span></dd>`
	if !strings.Contains(page, want) {
		t.Errorf("the page does not draw the accent with itself, and a swatch built from a second copy is the failure this page exists against")
	}
	if strings.Contains(page, `<dd>ms <span`) {
		t.Error("the page draws a swatch for a value that is not a colour")
	}
}

// Every group is drawn out of values rather than out of anything written here,
// which is what a case can check by moving a value and watching one drawing
// move with it.
func TestEveryGroupIsDrawnOutOfTheValueItNames(t *testing.T) {
	page := designPage(t, designTokens)

	for _, want := range []string{
		`<span style="font-size:14px;font-weight:540">tile</span>`,
		`<span style="border-radius:12px;background:#F0F0F2;color:#0E0E11">radius</span>`,
		`<span style="outline:3px solid #5B9CFF">standard</span>`,
		`<span style="outline:3px solid #0A6CE8">standard</span>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page does not carry %s", want)
		}
	}
}

// A group that has emptied is a refusal rather than a heading with nothing under
// it. The file is published elsewhere, so a renamed key is how this arrives, and
// a page that silently stopped demonstrating what it says it demonstrates is the
// failure the refusal exists for.
func TestAGroupThatCanNoLongerBeDrawnRedsTheBuild(t *testing.T) {
	cases := []struct {
		name    string
		renamed string
		says    string
	}{
		{"the ring width", `"ring-width"`, "The focus ring, one per colour vision preset"},
		{"the radius", `"shape": { "radius"`, "The corner a tile is drawn with"},
		{"the role size", `"size"`, "Type, at both viewing distances"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			said := designRefusal(t, strings.Replace(designTokens, c.renamed,
				strings.Replace(c.renamed, `"`, `"was-`, 1), 1))
			if !strings.Contains(said, c.says) {
				t.Errorf("the refusal does not name the group that emptied, it said: %s", said)
			}
		})
	}
}

// The two budgets are about different things, and the page says whose each one
// is rather than heading both of them "budget" and leaving a reader to tell them
// apart.
func TestThePageStatesBothBudgetsAndSaysWhichIsWhich(t *testing.T) {
	page := designPage(t, designTokens)

	client := strings.Index(page, "What a native client has to meet")
	site := strings.Index(page, "What this page itself has to fit inside")
	if client < 0 || site < 0 {
		t.Fatal("the page does not carry both budgets under headings saying whose each one is")
	}
	if !strings.Contains(page, "<td>focus-change</td><td>under 80 ms</td>") {
		t.Error("the client budget line is not the limit and the comparison the file gives")
	}
	if !strings.Contains(page, "under "+strconv.Itoa(budget.HTMLBytes)+" bytes") {
		t.Error("the site budget line is not the constant the check that refuses a page reads")
	}
	if strings.Index(page, "under "+strconv.Itoa(budget.HTMLBytes)+" bytes") < client {
		t.Error("the site budget is stated above the client budget's heading, so a reader meets a number before it says whose it is")
	}
}

// A limit whose comparison the file does not carry is a number that states
// nothing: a ceiling and a required value are opposite claims and the page would
// print them in the same words.
func TestABudgetNumberMissingWhatItMeansIsRefused(t *testing.T) {
	for _, dropped := range []string{`"unit": "ms",`, `"comparison": "below",`} {
		said := designRefusal(t, strings.Replace(designTokens, dropped, "", 1))
		if !strings.Contains(said, "budget.numbers.focus-change") {
			t.Errorf("dropping %s was refused without naming the number, it said: %s", dropped, said)
		}
	}
}

// An absent source is reported rather than passed over, so a run that produced
// no design system page cannot be read as a run that had nothing to produce.
func TestTheBuildSaysWhenThereIsNoSourceForTheDesignSystemPage(t *testing.T) {
	root := designTree(t, designTokens)
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(DesignSystemProse))); err != nil {
		t.Fatalf("removing the prose: %v", err)
	}
	var log strings.Builder
	written, err := Build(root, OutputDir, &log)
	if err != nil {
		t.Fatalf("building: %v", err)
	}
	if wrote(written, path.Join(OutputDir, DesignSystemPath)) {
		t.Error("the build wrote a design system page out of a tree that carries no prose for one")
	}
	if !strings.Contains(log.String(), "no "+DesignSystemProse+" in the tree") {
		t.Errorf("the build did not say which source was missing, it said: %s", log.String())
	}
}
