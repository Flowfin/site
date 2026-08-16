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

// colourValue is an sRGB value as the token file writes one. It is anchored at
// both ends so that a sentence mentioning a colour is not read as being one,
// which cannot happen under the rule above and is cheap to hold anyway.
var colourValue = regexp.MustCompile(`^#(?:[0-9A-Fa-f]{8}|[0-9A-Fa-f]{6}|[0-9A-Fa-f]{4}|[0-9A-Fa-f]{3})$`)

// nothing is what the reader flattens a leaf holding nothing to. It is named
// here rather than written into the comparison so that the bound the package
// comment states has a name in the source.
const nothing = "null"

// tokenValue is one leaf of the pinned copy as the page states it. Drawn is the
// declaration that renders the value where the value is something that can be
// rendered, and is empty otherwise, so the template asks whether there is
// something to draw rather than deciding what a colour is.
type tokenValue struct {
	Path  string
	Value string
	Drawn template.CSS
}

// sample is one thing the page shows by drawing it with the value it names
// rather than by describing it.
type sample struct {
	// Text is what is drawn. It is short because what a reader is looking
	// at is the drawing rather than the words.
	Text string
	// Says is what the reader is looking at, beside the drawing rather than
	// inside it, so the sample is not carrying its own caption in the size
	// it is demonstrating.
	Says  string
	Drawn template.CSS
}

// demonstration is a group of samples under what they are about.
type demonstration struct {
	Shows   string
	Says    string
	Samples []sample
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

	shows, reasons := demonstrations(values)
	if len(reasons) > 0 {
		return nil, fmt.Errorf("%s cannot be drawn from %s, %d reason(s):\n  %s",
			DesignSystemPath, tokens.File, len(reasons), strings.Join(reasons, "\n  "))
	}
	p.Shows = shows

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
		if v.Drawn != "" {
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
			listed = append(listed, tokenValue{Path: p, Value: v, Drawn: draw(v)})
		}
	}
	return listed, sentences, empty
}

// draw returns the declaration that renders a value with itself, or nothing
// where the value is not something a page can draw.
//
// A colour is drawn as its own foreground and its own background at once, which
// is a swatch that needs no size written anywhere: the box is as large as the
// text inside it, and the text is invisible because it is the colour it sits on.
// A size stated here would be a length this repository defined, and the one file
// a length is read from is the one this page is rendering.
func draw(value string) template.CSS {
	if !colourValue.MatchString(value) {
		return ""
	}
	return template.CSS("background:" + value + ";color:" + value)
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

// demonstrations builds the groups the page shows by drawing rather than by
// describing, and returns one reason per group it could not build.
//
// A group that came out empty is a refusal rather than a heading with nothing
// under it. The file this reads is published elsewhere, so a group emptying is
// how a renamed key would arrive here, and the failure that names it is worth
// more than a page that silently stops demonstrating what it says it does.
func demonstrations(values tokens.Values) ([]demonstration, []string) {
	var out []demonstration
	var reasons []string

	add := func(d demonstration, about string) {
		if len(d.Samples) == 0 {
			reasons = append(reasons, fmt.Sprintf(
				"nothing in the file answers to %s, so the group %q would be a heading with nothing under it",
				about, d.Shows))
			return
		}
		out = append(out, d)
	}

	add(typeScale(values), "a size and a weight under type.distances.<distance>.roles.<role>")
	add(corners(values), "a value under shape.radius<...>.value")
	add(focusRings(values), "a ring width and an accent under focus.presets.<preset>")

	return out, reasons
}

// typeScale draws each role at each viewing distance, in the size and the weight
// the file gives it. The two distances are one scale read from two distances, so
// the samples are listed together rather than as two scales.
func typeScale(values tokens.Values) demonstration {
	d := demonstration{
		Shows: "Type, at both viewing distances",
		Says: "Each role drawn at the size and the weight the file gives it. " +
			"The two distances are one scale read from two distances rather than two scales.",
	}
	for _, size := range endingIn(values, "size") {
		if !strings.HasPrefix(size, "type.distances.") {
			continue
		}
		weight, ok := values[strings.TrimSuffix(size, "size")+"weight"]
		if !ok {
			continue
		}
		parts := strings.Split(size, ".")
		if len(parts) < 5 {
			continue
		}
		distance, role := parts[2], parts[len(parts)-2]
		d.Samples = append(d.Samples, sample{
			Text:  role,
			Says:  fmt.Sprintf("%s, %s at weight %s", distance, values[size], weight),
			Drawn: template.CSS(fmt.Sprintf("font-size:%spx;font-weight:%s", values[size], weight)),
		})
	}
	return d
}

// corners draws each radius as the corner it produces, on a box that is as large
// as the words inside it. Which radius is which is the file's own reading: a
// radius is a value under a key that names one, and the widest a column of text
// may get is a length under the same group that is not a corner at all.
func corners(values tokens.Values) demonstration {
	d := demonstration{
		Shows: "The corner a tile is drawn with",
		Says: "Each radius drawn as the corner it makes. The surface and the ink are " +
			"the light scheme's, so the corner is visible against the box rather than against the page.",
	}
	ground, ink := values["surface.tokens.raise-2.light.srgb"], values["surface.tokens.ink.light.srgb"]
	if ground == "" || ink == "" {
		return d
	}
	for _, radius := range endingIn(values, "value") {
		if !strings.HasPrefix(radius, "shape.radius") {
			continue
		}
		d.Samples = append(d.Samples, sample{
			Text: strings.TrimSuffix(strings.TrimPrefix(radius, "shape."), ".value"),
			Says: values[radius],
			Drawn: template.CSS(fmt.Sprintf("border-radius:%spx;background:%s;color:%s",
				values[radius], ground, ink)),
		})
	}
	return d
}

// focusRings draws the accent of every colour vision preset as the ring it
// produces, in both schemes, because which hue works depends on which cone type
// is missing and a reader on a dark machine and a reader on a light one are not
// being shown the same colour.
func focusRings(values tokens.Values) demonstration {
	d := demonstration{
		Shows: "The focus ring, one per colour vision preset",
		Says: "Each preset's accent drawn as the ring it makes, at the width that " +
			"preset asks for, in both schemes. What each preset is for is what a reader is missing.",
	}
	for _, width := range endingIn(values, "ring-width") {
		if !strings.HasPrefix(width, "focus.presets.") {
			continue
		}
		preset := strings.TrimSuffix(strings.TrimPrefix(width, "focus.presets."), ".ring-width")
		missing := values["focus.presets."+preset+".missing"]
		for _, scheme := range []string{"dark", "light"} {
			accent, ok := values["focus.presets."+preset+"."+scheme+".accent.srgb"]
			if !ok {
				continue
			}
			d.Samples = append(d.Samples, sample{
				Text:  preset,
				Says:  fmt.Sprintf("%s, missing %s", scheme, missing),
				Drawn: template.CSS(fmt.Sprintf("outline:%spx solid %s", values[width], accent)),
			})
		}
	}
	return d
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
func clientBudget(values tokens.Values) (budgetTable, []string) {
	t := budgetTable{
		Whose: "What a native client has to meet",
		Says: "Five numbers, each a release condition rather than a target. They travel " +
			"with the values above because they are the same class of fact, and a client " +
			"that has not been measured has not met one.",
	}
	var reasons []string
	for _, limit := range endingIn(values, "limit") {
		if !strings.HasPrefix(limit, "budget.numbers.") {
			continue
		}
		at := strings.TrimSuffix(limit, ".limit")
		name := strings.TrimPrefix(at, "budget.numbers.")
		unit, ok := values[at+".unit"]
		if !ok {
			reasons = append(reasons, fmt.Sprintf("%s carries a limit and no unit, so the number states nothing", at))
			continue
		}
		comparison, ok := values[at+".comparison"]
		if !ok {
			reasons = append(reasons, fmt.Sprintf("%s carries a limit and no comparison, so whether it is a ceiling or a value is not stated", at))
			continue
		}
		t.Rows = append(t.Rows, budgetRow{
			Name:  name,
			Limit: stated(comparison, values[limit], unit),
			Means: strings.TrimSpace(values[at+".what"] + " " + values[at+".why"]),
		})
	}
	if len(t.Rows) == 0 {
		reasons = append(reasons, "nothing under budget.numbers carries a limit, so the client budget would be a table with no line in it")
	}
	return t, reasons
}

// stated turns a limit and its comparison into what the page says. The file
// carries the two apart, and a page printing the number alone would state a
// ceiling and a required value in the same words, which are opposite claims.
func stated(comparison, limit, unit string) string {
	switch comparison {
	case "below":
		return "under " + limit + " " + unit
	case "equal":
		return "exactly " + limit + " " + unit
	default:
		return comparison + " " + limit + " " + unit
	}
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
