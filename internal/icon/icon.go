// SPDX-License-Identifier: AGPL-3.0-or-later

// Package icon renders the mark a browser asks every page for.
//
// A browser asks for an icon on the first visit to any site, whether or not a
// page offers one. Where none is offered it guesses at /favicon.ico, and where
// that is not there either the request is answered by the host's not-found
// page: a round trip the reader pays for that carries nothing, on a site whose
// budget counts requests. Referencing a file and producing it costs one request
// either way, and the reference is what stops the guess.
//
// What the mark is was decided as a typographic icon and no drawn mark, so what
// is here is a letter set in the reader's own font rather than artwork. That
// decision is why this package can exist at all: a drawn mark is a file
// somebody authors and this repository commits, and a letter is a function of
// values the token file already carries.
//
// So the file is produced rather than committed, which is the rule the
// exclusion file and the sitemap are already written under: a committed file
// describing generated output is a second copy of the output that drifts from
// it, and the reproducibility check covers what the build writes rather than
// what somebody remembered to update.
//
// Nothing here defines a colour, a corner or a font stack. Every value is read
// from the pinned token copy, which is the rule the design system page already
// renders under, so the mark cannot go on stating a colour the system has
// stopped using. A value the copy does not carry is a reason this package
// returns rather than a default it substitutes: a mark drawn in a fallback
// colour looks finished, and what it says about the project is wrong.
package icon

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Flowfin/site/internal/tokens"
)

// Path is where the mark lands in the output. It is an SVG rather than one of
// the raster forms a browser also accepts, and that is the means rather than a
// preference: a typographic icon is text, and this is the one image format in
// which it stays text. Nothing has to be rasterised, so nothing has to bundle a
// typeface to rasterise it with, which is the reason the pages download no face
// either. The bytes are a function of the token copy and of nothing else, so
// two builds of one tree produce one file. And the dimensions are declared in
// the file's own root element, in a form read without an image decoder, which
// is what lets anything else compare a number against them.
const Path = "icon.svg"

// MediaType is what a reference states the file is, so a browser knows before
// it fetches whether it can use it.
const MediaType = "image/svg+xml"

// Side is the coordinate space the mark is drawn in, and the size a reader that
// asks for no particular one is given.
//
// It is a square because every surface that shows an icon reserves a square for
// it, and one that is not square is letterboxed into one by somebody else's
// rule. The number is twice the largest a browser asks for in a tab strip, so
// the file states a size no raster it is scaled into is larger than, and a
// scalable file is held to it in neither direction.
const Side = 64

// Alt is what the mark says to a reader who cannot see it. It is the project's
// name, because that is what the mark is: a letter standing for a word, where
// the word is the whole of what it means.
const Alt = "Flowfin"

// letter is what is drawn. It is the initial of the name above rather than a
// glyph chosen for its shape, so a reader who meets the mark beside the name
// once does not have to be told the two belong together.
const letter = "F"

// The values this mark is composed of, by the path each one sits at in the
// token copy. They are named here rather than spelled at the point of use, so
// that a copy which has stopped carrying one is reported as that path rather
// than as a blank in the output.
const (
	groundAt = "surface.tokens.ground."
	inkAt    = "surface.tokens.ink."
	radiusAt = "shape.radius.value"
	// The mark is one letter at the largest role the scale has, which is the
	// role the file describes as the one heading a screen is named by. The
	// weight is taken from that role rather than typed here for the reason
	// every other value is read: a scale that moves moves the mark with it.
	weightAt = "type.distances.television.roles.display.weight"
	// The family list, as the flattened copy spells an ordered list.
	stackAt = "font.sans.stack["
)

// The two brightness schemes, in the order the mark declares them: the light
// one is what the file is drawn in before any query is read, and the dark one
// is what a query switches it to.
var schemes = []string{"light", "dark"}

// what each colour the mark is composed of is called inside the file, against
// the group its value is read from.
var inks = map[string]string{"ground": groundAt, "ink": inkAt}

// Render returns the bytes of the mark, or the reasons it could not be drawn.
//
// It answers with every missing value rather than with the first, because a
// token copy that has lost one value has usually lost the group, and a reader
// repairing them one run at a time learns that the slowest way.
func Render(values tokens.Values) ([]byte, []string) {
	var reasons []string

	colours := map[string]map[string]string{}
	for _, scheme := range schemes {
		colours[scheme] = map[string]string{}
		for name, at := range inks {
			v, ok := values[at+scheme+".srgb"]
			if !ok || strings.TrimSpace(v) == "" {
				reasons = append(reasons, fmt.Sprintf(
					"%s carries no colour, and a mark drawn in a colour this file does not state looks finished and is wrong about the project",
					at+scheme+".srgb"))
				continue
			}
			colours[scheme][name] = v
		}
	}

	radius, why := whole(values, radiusAt, "the corner the mark is cut with")
	reasons = appendReason(reasons, why)
	weight, why := whole(values, weightAt, "the weight the letter is set at")
	reasons = appendReason(reasons, why)

	stack := familyStack(values)
	if stack == "" {
		reasons = append(reasons, fmt.Sprintf(
			"%s0] carries no family, and a letter set in no stated family is set in whatever the reader's software reached for",
			stackAt))
	}

	if len(reasons) > 0 {
		sort.Strings(reasons)
		return nil, reasons
	}

	// The letter is set at two thirds of the square. That is a proportion
	// rather than a size, so it follows the coordinate space above instead of
	// being a second number somebody has to keep in step with it, and it
	// leaves the margin every surface that rounds the corner off an icon eats
	// into.
	size := Side * 2 / 3

	var b strings.Builder
	fmt.Fprintf(&b, "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\" role=\"img\" aria-label=\"%s\">\n",
		Side, Side, Side, Side, Alt)
	fmt.Fprintf(&b, "  <title>%s</title>\n", Alt)
	b.WriteString("  <style>\n")
	// The light scheme is stated first and unconditionally, so a reader whose
	// software answers no question about brightness still gets a mark rather
	// than an unpainted square. The dark one is stated as the answer to a
	// question the reader's machine has already been asked, which is the shape
	// the pages carry: the preference belongs to the reader, and this file
	// follows it rather than choosing for them.
	fmt.Fprintf(&b, "    .ground { fill: %s }\n", colours["light"]["ground"])
	fmt.Fprintf(&b, "    .ink { fill: %s }\n", colours["light"]["ink"])
	b.WriteString("    @media (prefers-color-scheme: dark) {\n")
	fmt.Fprintf(&b, "      .ground { fill: %s }\n", colours["dark"]["ground"])
	fmt.Fprintf(&b, "      .ink { fill: %s }\n", colours["dark"]["ink"])
	b.WriteString("    }\n")
	b.WriteString("  </style>\n")
	fmt.Fprintf(&b, "  <rect class=\"ground\" width=\"%d\" height=\"%d\" rx=\"%s\" />\n", Side, Side, radius)
	fmt.Fprintf(&b, "  <text class=\"ink\" x=\"%d\" y=\"%d\" text-anchor=\"middle\" dominant-baseline=\"central\" font-family=\"%s\" font-size=\"%d\" font-weight=\"%s\">%s</text>\n",
		Side/2, Side/2, stack, size, weight, letter)
	b.WriteString("</svg>\n")
	return []byte(b.String()), nil
}

// familyStack joins the ordered list of family names the copy carries into the
// one declaration a font-family attribute takes, quoting a name that carries a
// space so the list is read as the names it holds rather than as more of them.
//
// It walks the list by index rather than asking the copy how long it is,
// because a flattened list has no length: what it has is a run of paths, and
// the run ends where the next index is not there.
func familyStack(values tokens.Values) string {
	var families []string
	for i := 0; ; i++ {
		v, ok := values[stackAt+strconv.Itoa(i)+"]"]
		if !ok {
			break
		}
		if strings.ContainsAny(v, " \t") {
			v = "'" + v + "'"
		}
		families = append(families, v)
	}
	return strings.Join(families, ", ")
}

// whole reads a value the mark writes as a bare number, and refuses one that is
// not, rather than writing it out and producing a mark whose renderer discards
// an attribute. what names the value in the reason, because a path on its own
// says where the value was and not what was missing.
func whole(values tokens.Values, at, what string) (string, string) {
	v, ok := values[at]
	if !ok {
		return "", fmt.Sprintf("%s is not in the copy, and it is %s", at, what)
	}
	if v == "" || strings.TrimLeft(v, "0123456789") != "" {
		return "", fmt.Sprintf("%s is %q, and %s has to be a whole number", at, v, what)
	}
	return v, ""
}

func appendReason(reasons []string, why string) []string {
	if why == "" {
		return reasons
	}
	return append(reasons, why)
}
