// SPDX-License-Identifier: AGPL-3.0-or-later

package icon

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Flowfin/site/internal/tokens"
)

// complete is a copy carrying every value the mark is composed of and nothing
// else. It is written out here rather than read from the tree, because a
// fixture that reads the real copy proves the state of the file on the day it
// ran rather than the behaviour of this package.
func complete() tokens.Values {
	return tokens.Values{
		"surface.tokens.ground.light.srgb":               "#FAFAFA",
		"surface.tokens.ground.dark.srgb":                "#121216",
		"surface.tokens.ink.light.srgb":                  "#0E0E11",
		"surface.tokens.ink.dark.srgb":                   "#ECECEF",
		"shape.radius.value":                             "12",
		"type.distances.television.roles.display.weight": "700",
		"font.sans.stack[0]":                             "ui-sans-serif",
		"font.sans.stack[1]":                             "Segoe UI",
		"font.sans.stack[2]":                             "sans-serif",
	}
}

// The file states its own size, in the root element and in the coordinate space
// together. That is the whole reason this format was chosen over a raster one:
// the numbers are readable without an image decoder, so anything else can
// compare a number against them.
func TestTheMarkDeclaresItsOwnSize(t *testing.T) {
	body, reasons := Render(complete())
	if len(reasons) > 0 {
		t.Fatalf("a complete copy did not draw a mark: %v", reasons)
	}
	side := strconv.Itoa(Side)
	for _, want := range []string{
		`width="` + side + `"`,
		`height="` + side + `"`,
		`viewBox="0 0 ` + side + ` ` + side + `"`,
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("the mark does not state %s:\n%s", want, body)
		}
	}
}

// Nothing here defines a colour, a corner or a weight. The proof is that moving
// each one in the copy moves it in the file: a value this package held its own
// copy of would go on rendering the old one, and the mark would look finished
// while saying something the system has stopped saying.
func TestEveryValueComesFromTheCopy(t *testing.T) {
	for _, c := range []struct {
		at   string
		to   string
		want string
	}{
		{"surface.tokens.ground.light.srgb", "#010203", "fill: #010203"},
		{"surface.tokens.ink.dark.srgb", "#040506", "fill: #040506"},
		{"shape.radius.value", "3", `rx="3"`},
		{"type.distances.television.roles.display.weight", "500", `font-weight="500"`},
		{"font.sans.stack[0]", "Nothing", `font-family="Nothing`},
	} {
		values := complete()
		values[c.at] = c.to
		body, reasons := Render(values)
		if len(reasons) > 0 {
			t.Fatalf("moving %s stopped the mark being drawn: %v", c.at, reasons)
		}
		if !strings.Contains(string(body), c.want) {
			t.Errorf("moving %s to %q did not reach the file, which wanted %q:\n%s",
				c.at, c.to, c.want, body)
		}
	}
}

// A value the copy does not carry is a reason rather than a default. A mark
// drawn in a fallback colour looks finished and is wrong about the project,
// which is the one failure a substituted value produces and a missing one does
// not.
func TestAMissingValueIsAReasonAndNotADefault(t *testing.T) {
	for _, at := range []string{
		"surface.tokens.ground.light.srgb",
		"surface.tokens.ink.dark.srgb",
		"shape.radius.value",
		"type.distances.television.roles.display.weight",
		"font.sans.stack[0]",
	} {
		values := complete()
		delete(values, at)
		body, reasons := Render(values)
		if body != nil {
			t.Errorf("a copy with no %s still drew a mark:\n%s", at, body)
		}
		if len(reasons) == 0 {
			t.Fatalf("a copy with no %s was refused without a reason", at)
		}
		if !strings.Contains(strings.Join(reasons, "\n"), at) {
			t.Errorf("the reasons for a copy with no %s do not name it: %v", at, reasons)
		}
	}
}

// Every missing value is reported rather than the first. A copy that has lost
// one has usually lost the group, and a reader repairing them one run at a time
// learns that the slowest way.
func TestEveryMissingValueIsReportedAtOnce(t *testing.T) {
	values := complete()
	delete(values, "shape.radius.value")
	delete(values, "type.distances.television.roles.display.weight")
	_, reasons := Render(values)
	if len(reasons) != 2 {
		t.Fatalf("two missing values produced %d reason(s): %v", len(reasons), reasons)
	}
}

// A value of the wrong shape is refused rather than written out. A renderer
// discards an attribute it cannot read, so a corner written as a length with a
// unit on it produces a square mark and no complaint from anything.
func TestAValueOfTheWrongShapeIsRefused(t *testing.T) {
	values := complete()
	values["shape.radius.value"] = "12px"
	body, reasons := Render(values)
	if body != nil {
		t.Fatalf("a corner written with a unit on it still drew a mark:\n%s", body)
	}
	if !strings.Contains(strings.Join(reasons, "\n"), "whole number") {
		t.Errorf("the reason does not say what the value had to be: %v", reasons)
	}
}

// A family name carrying a space is quoted, so the list is read as the names it
// holds rather than as more of them. Unquoted, "Segoe UI" is two families and
// neither exists.
func TestAFamilyNameWithASpaceIsQuoted(t *testing.T) {
	body, reasons := Render(complete())
	if len(reasons) > 0 {
		t.Fatalf("a complete copy did not draw a mark: %v", reasons)
	}
	if !strings.Contains(string(body), `'Segoe UI'`) {
		t.Errorf("a family name with a space in it is not quoted:\n%s", body)
	}
	if !strings.Contains(string(body), `ui-sans-serif, 'Segoe UI', sans-serif`) {
		t.Errorf("the stack is not in the order the copy carries it:\n%s", body)
	}
}

// The stack ends where the copy stops rather than at a length this package
// carries, so a copy listing one family more reaches the file with it.
func TestTheStackIsAsLongAsTheCopySaysAndNoLonger(t *testing.T) {
	values := complete()
	values["font.sans.stack[3]"] = "monospace"
	body, _ := Render(values)
	if !strings.Contains(string(body), `sans-serif, monospace"`) {
		t.Errorf("a fourth family did not reach the file:\n%s", body)
	}

	// A gap in the indices ends the list, because that is where a flattened
	// list ends. A family after a gap is not silently pulled forward into a
	// place the copy does not put it in.
	gapped := complete()
	gapped["font.sans.stack[9]"] = "cursive"
	body, _ = Render(gapped)
	if strings.Contains(string(body), "cursive") {
		t.Errorf("a family after a gap in the indices reached the file:\n%s", body)
	}
}

// Both schemes are stated, and the dark one only inside the question the
// reader's machine has already been asked. A mark that answered the question by
// choosing for the reader would render one of the two wrong on every visit.
func TestTheMarkFollowsTheReadersScheme(t *testing.T) {
	body, _ := Render(complete())
	text := string(body)
	query := strings.Index(text, "@media (prefers-color-scheme: dark)")
	if query < 0 {
		t.Fatalf("the mark states no dark scheme at all:\n%s", text)
	}
	if before, after := text[:query], text[query:]; !strings.Contains(before, "#FAFAFA") ||
		!strings.Contains(after, "#121216") {
		t.Errorf("the light scheme is not stated before the query and the dark one inside it:\n%s", text)
	}
}

// Two renders of one copy produce one set of bytes. The build writes this file
// on every run and the reproducibility check compares two of them, so a mark
// carrying anything that moves would red that check for a reason no change to
// this tree caused.
func TestTwoRendersOfOneCopyAgree(t *testing.T) {
	first, _ := Render(complete())
	second, _ := Render(complete())
	if string(first) != string(second) {
		t.Errorf("two renders of one copy disagree:\n%s\n%s", first, second)
	}
}

// The mark says what it is to a reader who cannot see it. An icon is the one
// image on a page that carries no caption and sits beside no words, so the name
// it answers to is the whole of what it says.
func TestTheMarkSaysWhatItIs(t *testing.T) {
	body, _ := Render(complete())
	if !strings.Contains(string(body), `aria-label="`+Alt+`"`) ||
		!strings.Contains(string(body), "<title>"+Alt+"</title>") {
		t.Errorf("the mark does not name itself:\n%s", body)
	}
}
