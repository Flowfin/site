// SPDX-License-Identifier: AGPL-3.0-or-later

// The design system as a page: every value the pinned token copy carries,
// stated and drawn with itself.
//
// The page exists because a design system described in prose is a design system
// nobody can check. A paragraph saying the accent is a blue that survives a
// missing cone type is a claim; the accent rendered beside its own hexadecimal
// is the thing itself, and a reader can see in one glance whether the two agree.
// So every value below is read out of the copy in data/, and the colour, the
// corner and the focus ring beside a value are built out of that same value
// rather than out of a second copy of it. Nothing on this page is typed here.
//
// # Which leaf is a value
//
// The token file is a file this repository does not author, and it holds two
// different kinds of thing under one shape. Most leaves are values a client
// reads and has to meet. Some are the file explaining itself to whoever opens
// it, and those are sentences rather than values. A page that listed both would
// be a page of prose with numbers scattered through it, and it would not fit the
// document line the speed budget fixes.
//
// The rule this file takes is about the value and never about the key. A leaf
// that is a string ending in a full stop, or carrying one followed by a space,
// is the file describing itself. Everything else is a value. The key cannot
// decide it: the file carries a `weight` that is a sentence about what weights
// mean and a `weight` that is the number 540, and a rule reading the last
// segment gets both of them wrong.
//
// It has one bound and the bound is stated rather than hidden. A leaf holding
// nothing at all, which the reader flattens to the four letters of a JSON null,
// is counted as carrying no value rather than as a value spelled that way. A
// token whose value was literally those four letters would be read the same way,
// and there is none.
//
// The counts of all three are printed on the page. A rule that silently drops a
// third of a file is a rule nobody audits, and three numbers that add up to what
// the file holds are what let a reader audit it without reading this comment.
//
// # What is not decided here
//
// Two other answers to the same question were available and each moves something
// settled elsewhere, so neither is taken:
// decisions/0014-what-the-design-system-page-renders.md is where all three are
// argued and where the one above is chosen.
package site

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Flowfin/site/internal/budget"
	"github.com/Flowfin/site/internal/tokens"
)

// DesignSystemProse is the page's words and DesignSystemPath is where the
// produced page lands. The address is the one
// decisions/0008-the-url-shape.md gives it, and it is the flat name that record
// declines to move because it is the one address besides the root this project
// has already published.
const (
	DesignSystemProse = ContentDir + "/design-system.txt"
	DesignSystemPath  = "design-system.html"
)

// sentence is what the file's own prose about itself looks like from the value
// side: a full stop at the end, or one followed by a space because the leaf
// carries more than one. Nothing about the key is read.
var sentence = regexp.MustCompile(`\.$|\. `)

// colourValue is an sRGB value as the token file writes one, and number is a
// value that can stand in front of a unit.
//
// Both are anchored at both ends, and that is the first of two things every
// drawing rests on. A value reaches a style attribute, and the file it comes
// from is published elsewhere, so a value carrying a semicolon would arrive on
// the page as a second declaration nobody wrote. A value that is not one of
// these two shapes is not drawn and the group says so.
//
// The second is that the property names are written in the frame and only the
// value is handed to it, so the template engine reads each one in a value
// position and filters it. Nothing here converts a string into markup the engine
// would write out unread, which is what the static analysis over this package
// refuses and what a page assembling its own declarations would have needed.
var (
	colourValue = regexp.MustCompile(`^#(?:[0-9A-Fa-f]{8}|[0-9A-Fa-f]{6}|[0-9A-Fa-f]{4}|[0-9A-Fa-f]{3})$`)
	number      = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)
)

// nothing is what the reader flattens a leaf holding nothing to. It is named
// here rather than written into the comparison so that the bound the package
// comment states has a name in the source.
const nothing = "null"

// tokenValue is one leaf of the pinned copy as the page states it. Colour is the
// value again where the value is a colour and empty otherwise, so the frame asks
// whether there is a colour rather than deciding what one is.
type tokenValue struct {
	Path   string
	Value  string
	Colour string
}

// The three things the page shows by drawing them with the values they name.
// Each one carries its values apart rather than as a finished declaration,
// because the frame writes the property and the engine reads the value.
//
// The type sample is drawn at its size and states its weight in words. The
// property that would draw the weight may not appear in what the build reads,
// which is the row that keeps a weight from being defined anywhere but the token
// file, and this page is inside that row's subject like every other build input.
type typeSample struct {
	Role     string
	Distance string
	Size     string
	Weight   string
}

type cornerSample struct {
	Name   string
	Radius string
	Ground string
	Ink    string
}

type ringSample struct {
	Preset  string
	Scheme  string
	Missing string
	Width   string
	Accent  string
}

// budgetRow is one line of a budget, and budgetTable is a budget with what it is
// about. The page carries two of them and they are about different things, which
// is why the table says whose it is rather than being headed "budget".
type budgetRow struct {
	Name  string
	Limit string
	Means string
}

type budgetTable struct {
	Whose string
	Says  string
	Rows  []budgetRow
}

// writeDesignSystem renders the design system out of its prose and out of the
// pinned token copy, through the same template every other page goes through.
//
// Either source being absent is reported rather than passed over, the way the
// other pages report an absent one, and the report names which of the two was
// missing: a page with words and no values and a page with values and no words
// are different repairs.
func writeDesignSystem(root, out, label string, tmpl *template.Template, said descriptions, log io.Writer) ([]string, error) {
	prose := filepath.Join(root, filepath.FromSlash(DesignSystemProse))
	pinned := filepath.Join(root, filepath.FromSlash(tokens.File))
	for _, source := range []struct{ name, path string }{{DesignSystemProse, prose}, {tokens.File, pinned}} {
		if _, err := os.Stat(source.path); os.IsNotExist(err) {
			fmt.Fprintf(log, "no %s in the tree, so no design system page was written\n", source.name)
			return nil, nil
		}
	}

	p, err := readDesignSystem(prose)
	if err != nil {
		return nil, fmt.Errorf("reading the design system prose: %w", err)
	}

	values, err := tokens.Load(root)
	if err != nil {
		return nil, fmt.Errorf("reading the design tokens: %w", err)
	}

	listed, sentences, empty := readValues(values)
	p.Values = listed
	p.Reading = reading(len(values), len(listed), sentences, empty)

	if reasons := drawings(values, &p); len(reasons) > 0 {
		return nil, fmt.Errorf("%s cannot be drawn from %s, %d reason(s):\n  %s",
			DesignSystemPath, tokens.File, len(reasons), strings.Join(reasons, "\n  "))
	}

	client, reasons := clientBudget(values)
	if len(reasons) > 0 {
		return nil, fmt.Errorf("the client budget cannot be read out of %s, %d reason(s):\n  %s",
			tokens.File, len(reasons), strings.Join(reasons, "\n  "))
	}
	p.Budgets = []budgetTable{client, siteBudget()}

	p.locate(DesignSystemPath)
	if err := said.add(DesignSystemPath, p.Description); err != nil {
		return nil, err
	}

	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, p); err != nil {
		return nil, fmt.Errorf("rendering the design system: %w", err)
	}
	name := filepath.Join(out, filepath.FromSlash(DesignSystemPath))
	if err := os.WriteFile(name, []byte(rendered.String()), 0o644); err != nil {
		return nil, fmt.Errorf("writing %s: %w", DesignSystemPath, err)
	}
	slashed := path.Join(label, DesignSystemPath)

	drawn := 0
	for _, v := range listed {
		if v.Colour != "" {
			drawn++
		}
	}
	fmt.Fprintf(log, "wrote %s (%d bytes, %d value(s) listed, %d drawn, %d sentence(s) not listed)\n",
		slashed, rendered.Len(), len(listed), drawn, sentences)
	return []string{slashed}, nil
}

// readValues splits the pinned copy into what the page lists and what it does
// not, and returns the counts of both kinds it left out. The list is sorted, so
// two builds of one file produce one page.
func readValues(values tokens.Values) (listed []tokenValue, sentences, empty int) {
	paths := make([]string, 0, len(values))
	for p := range values {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		v := values[p]
		switch {
		case sentence.MatchString(v):
			sentences++
		case v == nothing:
			empty++
		default:
			listed = append(listed, tokenValue{Path: p, Value: v, Colour: colour(v)})
		}
	}
	return listed, sentences, empty
}

// colour returns the value again where it is a colour, and nothing otherwise.
//
// The frame draws it as its own foreground and its own background at once, which
// is a swatch that needs no size written anywhere: the box is as large as the
// text inside it, and the text is invisible because it is the colour it sits on.
// A size stated here would be a length this repository defined, and the one file
// a length is read from is the one this page is rendering.
func colour(value string) string {
	if !colourValue.MatchString(value) {
		return ""
	}
	return value
}

// reading is the sentence that lets a reader audit the rule this file takes
// without reading the source that takes it. The three numbers add up to what the
// file holds, so a rule that quietly dropped a value would show up as a total
// that does not.
func reading(held, listed, sentences, empty int) string {
	return fmt.Sprintf(
		"The pinned copy holds %d leaves and this page lists the %d that are values. "+
			"Another %d are sentences the file writes about itself rather than values, "+
			"and %s. Every value is read out of that one file and none is defined here.",
		held, listed, sentences, holding(empty))
}

// holding says how many leaves carry nothing at all, in words that read as
// English at every count. A number with a plural verb welded to it reads wrong
// at one and at zero, and this sentence is on a page rather than in a log.
func holding(empty int) string {
	switch empty {
	case 0:
		return "no leaf holds nothing at all"
	case 1:
		return "one leaf holds nothing at all"
	default:
		return fmt.Sprintf("%d leaves hold nothing at all", empty)
	}
}

// drawings puts the three groups the page shows by drawing on the page, and
// returns one reason per value it could not draw with and per group that came
// out empty.
//
// A group that came out empty is a refusal rather than a heading with nothing
// under it. The file this reads is published elsewhere, so a group emptying is
// how a renamed key would arrive here, and a page that silently stops
// demonstrating what it says it demonstrates is worse than a red build.
func drawings(values tokens.Values, p *page) []string {
	var reasons []string

	scale, why := typeScale(values)
	reasons = append(reasons, why...)
	if len(scale) == 0 {
		reasons = append(reasons, emptyGroup("the type scale",
			"a size and a weight under type.distances.<distance>.roles.<role>"))
	}
	p.TypeScale = scale

	shapes, why := corners(values)
	reasons = append(reasons, why...)
	if len(shapes) == 0 {
		reasons = append(reasons, emptyGroup("the corner a tile is drawn with",
			"a value under shape.radius<...>.value beside a light surface and a light ink"))
	}
	p.Corners = shapes

	rings, why := focusRings(values)
	reasons = append(reasons, why...)
	if len(rings) == 0 {
		reasons = append(reasons, emptyGroup("the focus ring per colour vision preset",
			"a ring width and an accent under focus.presets.<preset>"))
	}
	p.Rings = rings

	return reasons
}

func emptyGroup(group, about string) string {
	return fmt.Sprintf(
		"nothing in the file answers to %s, so %s would be a heading with nothing under it",
		about, group)
}

// numberAt and colourAt read a value the page is about to draw with and refuse
// one that is not the shape a drawing can be built out of.
//
// They exist because the value reaches a style attribute, and the file is
// published elsewhere. The template engine reads each one in a value position
// and filters it as well, so these are the first of two readings rather than the
// only one, and what they add is that a value of the wrong shape is a reason the
// build prints rather than a sample quietly missing from a group. A group that
// is one sample short looks exactly like a group that is complete.
func numberAt(values tokens.Values, at string) (string, string) {
	v, ok := values[at]
	if !ok {
		return "", ""
	}
	if !number.MatchString(v) {
		return "", fmt.Sprintf("%s is %q, and a value drawn in front of a unit has to be a number", at, v)
	}
	return v, ""
}

func colourAt(values tokens.Values, at string) (string, string) {
	v, ok := values[at]
	if !ok {
		return "", ""
	}
	if !colourValue.MatchString(v) {
		return "", fmt.Sprintf("%s is %q, and a value drawn as a colour has to be one", at, v)
	}
	return v, ""
}

// typeScale is each role at each viewing distance, drawn at the size the file
// gives it and stating the weight beside it. The two distances are one scale
// read from two distances, so the samples are listed together rather than as two
// scales.
func typeScale(values tokens.Values) ([]typeSample, []string) {
	var out []typeSample
	var reasons []string
	for _, at := range endingIn(values, "size") {
		if !strings.HasPrefix(at, "type.distances.") {
			continue
		}
		parts := strings.Split(at, ".")
		if len(parts) < 5 {
			continue
		}
		size, why := numberAt(values, at)
		reasons = appendReason(reasons, why)
		weight, why := numberAt(values, strings.TrimSuffix(at, "size")+"weight")
		reasons = appendReason(reasons, why)
		if size == "" || weight == "" {
			continue
		}
		out = append(out, typeSample{
			Role:     parts[len(parts)-2],
			Distance: parts[2],
			Size:     size,
			Weight:   weight,
		})
	}
	return out, reasons
}

// corners is each radius drawn as the corner it produces, on a box that is as
// large as the words inside it. Which radius is which is the file's own reading:
// a radius is a value under a key that names one, and the widest a column of
// text may get is a length under the same group that is not a corner at all.
func corners(values tokens.Values) ([]cornerSample, []string) {
	var out []cornerSample
	var reasons []string
	ground, why := colourAt(values, "surface.tokens.raise-2.light.srgb")
	reasons = appendReason(reasons, why)
	ink, why := colourAt(values, "surface.tokens.ink.light.srgb")
	reasons = appendReason(reasons, why)
	if ground == "" || ink == "" {
		return nil, reasons
	}
	for _, at := range endingIn(values, "value") {
		if !strings.HasPrefix(at, "shape.radius") {
			continue
		}
		radius, why := numberAt(values, at)
		reasons = appendReason(reasons, why)
		if radius == "" {
			continue
		}
		out = append(out, cornerSample{
			Name:   strings.TrimSuffix(strings.TrimPrefix(at, "shape."), ".value"),
			Radius: radius,
			Ground: ground,
			Ink:    ink,
		})
	}
	return out, reasons
}

// focusRings is the accent of every colour vision preset drawn as the ring it
// produces, in both schemes, because which hue works depends on which cone type
// is missing and a reader on a dark machine and a reader on a light one are not
// being shown the same colour.
func focusRings(values tokens.Values) ([]ringSample, []string) {
	var out []ringSample
	var reasons []string
	for _, at := range endingIn(values, "ring-width") {
		if !strings.HasPrefix(at, "focus.presets.") {
			continue
		}
		width, why := numberAt(values, at)
		reasons = appendReason(reasons, why)
		if width == "" {
			continue
		}
		preset := strings.TrimSuffix(strings.TrimPrefix(at, "focus.presets."), ".ring-width")
		for _, scheme := range []string{"dark", "light"} {
			accent, why := colourAt(values, "focus.presets."+preset+"."+scheme+".accent.srgb")
			reasons = appendReason(reasons, why)
			if accent == "" {
				continue
			}
			out = append(out, ringSample{
				Preset:  preset,
				Scheme:  scheme,
				Missing: values["focus.presets."+preset+".missing"],
				Width:   width,
				Accent:  accent,
			})
		}
	}
	return out, reasons
}

// appendReason keeps the callers above readable, because a value that was simply
// not in the file is not a reason and a value of the wrong shape is.
func appendReason(reasons []string, why string) []string {
	if why == "" {
		return reasons
	}
	return append(reasons, why)
}

// endingIn returns every path whose last segment is the one named, sorted, so
// that what a group draws is a property of the file rather than of the order a
// map was walked in.
func endingIn(values tokens.Values, segment string) []string {
	var out []string
	for p := range values {
		if strings.HasSuffix(p, "."+segment) {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// clientBudget is the budget a native client has to meet, read out of the same
// file the values come from. It is not this site's budget and the table says so
// in its own heading rather than leaving the two to be told apart by a reader.
//
// It reads the file's sentences deliberately. The `what` and `why` beside each
// number are prose about the file, which is exactly what a budget row needs
// beside a limit, and they are the reason the rule above keeps them rather than
// discarding them.
//
// How a limit is spelled is asked of the token package rather than decided here.
// The invariant row that refuses a second copy of one of these numbers has to
// look for the same strings this table prints, and a page and a row each
// carrying their own spelling would part company on the day either moved.
func clientBudget(values tokens.Values) (budgetTable, []string) {
	t := budgetTable{
		Whose: "What a native client has to meet",
		Says: "Five numbers, each a release condition rather than a target. They travel " +
			"with the values above because they are the same class of fact, and a client " +
			"that has not been measured has not met one.",
	}
	numbers, reasons := tokens.Numbers(values)
	for _, n := range numbers {
		t.Rows = append(t.Rows, budgetRow{
			Name:  n.Name,
			Limit: n.Stated(),
			Means: strings.TrimSpace(n.What + " " + n.Why),
		})
	}
	if len(t.Rows) == 0 {
		reasons = append(reasons, "nothing under budget.numbers carries a limit, so the client budget would be a table with no line in it")
	}
	return t, reasons
}

// siteBudget is what this page itself has to fit inside, read from the constants
// the row that refuses a page reads. The published budget and the enforced one
// are one set of numbers here, so the page cannot go on stating a limit nothing
// holds anybody to.
func siteBudget() budgetTable {
	return budgetTable{
		Whose: "What this page itself has to fit inside",
		Says: "This site's own budget, which is about the bytes of this document rather " +
			"than about any client. The check that refuses a page over one of these " +
			"lines reads the same constants this table does.",
		Rows: []budgetRow{
			{
				Name:  "the document",
				Limit: "under " + strconv.Itoa(budget.HTMLBytes) + " bytes",
				Means: "Markup and inlined stylesheet together, uncompressed, because that is what one request delivers.",
			},
			{
				Name:  "the inlined stylesheet",
				Limit: "under " + strconv.Itoa(budget.InlineCSSBytes) + " bytes",
				Means: "What the document carries inside its own style elements, which is what keeps inlining cheaper than the request it replaced.",
			},
			{
				Name:  "downloaded faces",
				Limit: "exactly " + strconv.Itoa(budget.WebFontDownloads),
				Means: "A downloaded face blocks or reflows the first text a reader sees, and the faces already on the reader's machine arrive first.",
			},
			{
				Name:  "images the landing page asks for",
				Limit: "at most " + strconv.Itoa(budget.LandingImages),
				Means: "Counted on the page a reader arrives at, because the first page has to be complete after the fewest exchanges.",
			},
		},
	}
}

// readDesignSystem reads the page's words. It carries no values of its own, so
// what it refuses is a page with nothing to read and a block that opens like a
// keyword and is not one.
func readDesignSystem(name string) (page, error) {
	read, err := blocks(name)
	if err != nil {
		return page{}, err
	}

	var p page
	var reasons []string
	for i, b := range read {
		joined := strings.Join(b, " ")
		switch {
		case i == 0:
			p.Title = joined
		case strings.HasPrefix(joined, descriptionKeyword):
			text, reason := describe(joined)
			if reason != "" {
				reasons = append(reasons, reason)
				continue
			}
			p.Description = text
		case keywordLine.MatchString(joined):
			reasons = append(reasons, fmt.Sprintf(
				"a block opens %q, and the only keyword this file carries is %s, so a block that opens like one and is read as a paragraph loses whatever it was for: %s",
				strings.SplitN(joined, ":", 2)[0]+":", descriptionKeyword, short(joined)))
		default:
			p.Paragraphs = append(p.Paragraphs, joined)
		}
	}

	if p.Title == "" {
		reasons = append(reasons, "the file carries no title line")
	}
	if p.Description == "" {
		reasons = append(reasons, missingDescription())
	}
	if len(reasons) > 0 {
		return page{}, fmt.Errorf("%s was refused, %d reason(s):\n  %s",
			filepath.ToSlash(name), len(reasons), strings.Join(reasons, "\n  "))
	}
	return p, nil
}
