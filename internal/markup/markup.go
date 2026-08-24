// SPDX-License-Identifier: AGPL-3.0-or-later

// Package markup reads a produced page strictly and says what is wrong with it.
//
// A browser recovers from broken markup. That is the problem this package is
// about: a template can produce an unclosed element for months and every page
// still renders, so nothing in a review, a build or a reader's visit says a word
// about it, and the first thing that notices is a screen reader announcing a
// heading that swallowed the rest of the document.
//
// So the page is read the way nothing else reads it. Every element is closed, in
// the order it was opened, or the page is refused. That is stricter than HTML,
// which allows a paragraph or a list item to end where the next one begins, and
// it is deliberate: this repository writes every byte of its own markup out of
// templates, so an optional end tag left out is a choice somebody made once, and
// a rule that has to model the browser's recovery is a rule nobody can read.
//
// Four more properties are read off the same walk, because the walk is where
// they are cheap. A duplicate identifier breaks an anchor and every reference to
// it. A heading level that skips leaves a reader who navigates by heading with
// no way to tell what is under what. An image with no alternative text is a hole
// in the page for anybody who cannot see it. A form control with no label is a
// field somebody is asked to fill in without being told what goes in it.
//
// Nothing here is a browser and nothing here reaches the network. The page is
// read as bytes, so this stays inside a suite that runs on a machine with no
// display, and what a browser has to decide is left to the run that has one.
package markup

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// The kinds of problem, which are the properties a rule can be about. They are
// separate because the repairs are: a page that does not parse is a template
// somebody has to fix before anything else on it can be judged, and a heading
// that skips is a page that parses perfectly.
const (
	Structure = "structure"
	Identity  = "identity"
	Heading   = "heading"
	Alt       = "alt"
	Label     = "label"
)

// A Problem is one thing wrong with a page, with the line it is on so that a
// reader can go to it rather than search for it.
type Problem struct {
	Kind string
	Line int
	Says string
}

// Of returns the problems of one kind, which is how a rule about one property
// reads the walk that decides all of them.
func Of(kind string, problems []Problem) []Problem {
	var out []Problem
	for _, p := range problems {
		if p.Kind == kind {
			out = append(out, p)
		}
	}
	return out
}

func (p Problem) String() string { return fmt.Sprintf("line %d: %s", p.Line, p.Says) }

// voidElements close themselves. They are the whole set the standard defines,
// written out rather than derived, because the set is fixed and a rule guessing
// at it would refuse the first page carrying an element it had not met.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

// rawTextElements hold text rather than markup, so what is inside them is not
// read as elements. A `<` inside a stylesheet is a character and not the start
// of a tag.
var rawTextElements = map[string]bool{"script": true, "style": true, "textarea": true, "title": true}

// controls are the elements a person is asked to fill in or press.
var controls = map[string]bool{"input": true, "select": true, "textarea": true}

// element is an open tag waiting to be closed.
type element struct {
	name string
	at   int
}

// tag is one start tag the walk met.
type tag struct {
	name  string
	attrs map[string]string
	at    int
	// closed is true for a tag written as self-closing.
	closed bool
}

// Read walks the page and returns everything wrong with it, in the order the
// problems appear. An unreadable page is refused at the first thing that makes
// no sense rather than reported on twice, because every problem after a broken
// tag is a guess about what the author meant.
func Read(body []byte) []Problem {
	src := string(body)
	var problems []Problem
	var open []element
	ids := map[string]int{}
	lastHeading := 0

	add := func(kind string, at int, format string, args ...any) {
		problems = append(problems, Problem{Kind: kind, Line: lineAt(src, at), Says: fmt.Sprintf(format, args...)})
	}

	i := 0
	for i < len(src) {
		next := strings.IndexByte(src[i:], '<')
		if next < 0 {
			break
		}
		at := i + next
		rest := src[at:]

		switch {
		case strings.HasPrefix(rest, "<!--"):
			end := strings.Index(rest, "-->")
			if end < 0 {
				add(Structure, at, "a comment is opened and never closed")
				return problems
			}
			i = at + end + len("-->")
			continue
		case strings.HasPrefix(rest, "<!"), strings.HasPrefix(rest, "<?"):
			end := strings.IndexByte(rest, '>')
			if end < 0 {
				add(Structure, at, "a declaration is opened and never closed")
				return problems
			}
			i = at + end + 1
			continue
		case strings.HasPrefix(rest, "</"):
			name, after, ok := endTag(rest)
			if !ok {
				add(Structure, at, "an end tag is opened and never closed")
				return problems
			}
			if len(open) == 0 {
				add(Structure, at, "</%s> closes nothing; there is no open element here", name)
				return problems
			}
			last := open[len(open)-1]
			if last.name != name {
				add(Structure, at, "</%s> closes an element that is not open; the innermost open element is <%s>, opened on line %d",
					name, last.name, lineAt(src, last.at))
				return problems
			}
			open = open[:len(open)-1]
			i = at + after
			continue
		}

		t, after, ok := startTag(rest, at)
		if !ok {
			add(Structure, at, "a tag is opened and never closed")
			return problems
		}
		i = at + after

		if id, has := t.attrs["id"]; has {
			if strings.TrimSpace(id) == "" {
				add(Identity, t.at, "<%s> carries an empty id, which no reference can reach", t.name)
			} else if first, seen := ids[id]; seen {
				add(Identity, t.at, "the id %q is used a second time here, and first on line %d; a reference to it reaches one of the two", id, first)
			} else {
				ids[id] = lineAt(src, t.at)
			}
		}

		if level, is := headingLevel(t.name); is {
			if lastHeading > 0 && level > lastHeading+1 {
				add(Heading, t.at, "<%s> follows <h%d>, so a level is skipped and a reader moving by heading cannot tell what is under what",
					t.name, lastHeading)
			}
			lastHeading = level
		}

		if t.name == "img" {
			if _, has := t.attrs["alt"]; !has {
				add(Alt, t.at, "an image carries no alt attribute, so a reader who cannot see it is told nothing; an image that says nothing carries an empty one deliberately")
			}
		}

		if controls[t.name] && !labelled(t) {
			add(Label, t.at, "a <%s> is asked to be filled in with nothing naming it: no label, no aria-label and no aria-labelledby", t.name)
		}

		if voidElements[t.name] || t.closed {
			continue
		}
		open = append(open, element{name: t.name, at: t.at})
		if rawTextElements[t.name] {
			// What is inside holds text rather than markup, so the walk
			// steps over it to the end tag and lets the loop close the
			// element there. A `<` inside a stylesheet is a character,
			// and reading it as a tag is how a strict rule refuses a
			// page that is perfectly well formed.
			closer := "</" + t.name
			end := indexFold(src[i:], closer)
			if end < 0 {
				add(Structure, t.at, "<%s> is opened and never closed", t.name)
				return problems
			}
			i += end
		}
	}

	for _, e := range open {
		problems = append(problems, Problem{
			Kind: Structure,
			Line: lineAt(src, e.at),
			Says: fmt.Sprintf("<%s> is opened and never closed", e.name),
		})
	}
	sort.SliceStable(problems, func(i, j int) bool { return problems[i].Line < problems[j].Line })
	return problems
}

// labelled says whether a control is named by something. A wrapping label is not
// read: what is checked is what the control itself carries, and the page this
// repository produces has no form on it, so the strict answer is the useful one
// on the day a control first appears.
func labelled(t tag) bool {
	if t.attrs["type"] == "hidden" {
		return true
	}
	for _, name := range []string{"aria-label", "aria-labelledby", "title"} {
		if strings.TrimSpace(t.attrs[name]) != "" {
			return true
		}
	}
	return false
}

func headingLevel(name string) (int, bool) {
	if len(name) != 2 || name[0] != 'h' {
		return 0, false
	}
	level, err := strconv.Atoi(name[1:])
	if err != nil || level < 1 || level > 6 {
		return 0, false
	}
	return level, true
}

// endTag reads `</name>` and returns the name and how far it reached.
func endTag(rest string) (string, int, bool) {
	end := strings.IndexByte(rest, '>')
	if end < 0 {
		return "", 0, false
	}
	return strings.ToLower(strings.TrimSpace(rest[2:end])), end + 1, true
}

// startTag reads a start tag and its attributes. A value may be in double
// quotes, in single quotes or in none, because a page is judged as it was
// produced rather than as somebody meant to produce it.
func startTag(rest string, at int) (tag, int, bool) {
	i := 1
	start := i
	for i < len(rest) && !isSpace(rest[i]) && rest[i] != '>' && rest[i] != '/' {
		i++
	}
	if i == start || i >= len(rest) {
		return tag{}, 0, false
	}
	t := tag{name: strings.ToLower(rest[start:i]), attrs: map[string]string{}, at: at}

	for i < len(rest) {
		for i < len(rest) && isSpace(rest[i]) {
			i++
		}
		if i >= len(rest) {
			return tag{}, 0, false
		}
		if rest[i] == '/' {
			t.closed = true
			i++
			continue
		}
		if rest[i] == '>' {
			return t, i + 1, true
		}
		nameStart := i
		for i < len(rest) && !isSpace(rest[i]) && rest[i] != '=' && rest[i] != '>' && rest[i] != '/' {
			i++
		}
		if i >= len(rest) {
			return tag{}, 0, false
		}
		name := strings.ToLower(rest[nameStart:i])
		for i < len(rest) && isSpace(rest[i]) {
			i++
		}
		if i >= len(rest) {
			return tag{}, 0, false
		}
		if rest[i] != '=' {
			t.attrs[name] = ""
			continue
		}
		i++
		for i < len(rest) && isSpace(rest[i]) {
			i++
		}
		if i >= len(rest) {
			return tag{}, 0, false
		}
		var value string
		switch q := rest[i]; q {
		case '"', '\'':
			end := strings.IndexByte(rest[i+1:], q)
			if end < 0 {
				return tag{}, 0, false
			}
			value = rest[i+1 : i+1+end]
			i += end + 2
		default:
			valueStart := i
			for i < len(rest) && !isSpace(rest[i]) && rest[i] != '>' {
				i++
			}
			value = rest[valueStart:i]
		}
		t.attrs[name] = value
	}
	return tag{}, 0, false
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

func indexFold(s, sub string) int {
	return strings.Index(strings.ToLower(s), strings.ToLower(sub))
}

func lineAt(s string, at int) int {
	if at > len(s) {
		at = len(s)
	}
	return strings.Count(s[:at], "\n") + 1
}
