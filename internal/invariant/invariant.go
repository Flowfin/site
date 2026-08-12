// Package invariant holds the rules about this repository that a machine can
// decide by reading bytes, and refuses a tree that breaks one.
//
// A rule is one row. A row carries the population it reads, the reason it
// exists, what it refuses, and the code that decides it. Adding a rule is adding
// a row, and nothing else changes.
//
// The rows live here rather than in the workflow that reports them. A rule
// written as a shell block in a workflow file has no suite that can prove it
// bites and no way to run it on the machine where the mistake was made, which is
// the position this repository already took for the commit message rules. What
// the workflow supplies is the name the result reports under.
//
// A run says what it examined, and it also says what it did not. Some of the
// rules this gate is meant to carry have nothing in the tree to read yet, and
// they are printed as owed with the issue that lands what they would read. A run
// that listed only the rows it could decide would read as a tree that satisfies
// all of them. How many are owed is what the run prints, and a count written
// here would drift against the table the run reads.
package invariant

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Flowfin/site/internal/markup"
	"github.com/Flowfin/site/internal/pins"
	"github.com/Flowfin/site/internal/security"
	"github.com/Flowfin/site/internal/site"
	"github.com/Flowfin/site/internal/tokens"
	"github.com/Flowfin/site/internal/version"
)

// The populations a row can read. A row names one of them, and the run reads
// each population once however many rows ask for it.
const (
	// ProducedPages is every HTML file the build just wrote.
	ProducedPages = "every page the build produced"
	// ProducedFiles is everything the build just wrote, markup or not.
	ProducedFiles = "every file the build produced"
	// TrackedText is every tracked file that is not binary, except the file
	// that declares the marker vocabulary. That exclusion is real and is
	// stated rather than hidden: a file listing the strings a rule refuses
	// would be refused by its own rule, and the alternative is a vocabulary
	// spelled in fragments, which nobody can read or maintain.
	TrackedText = "every tracked text file, except the one declaring the vocabulary"
	// TestSources is every tracked test source. The headless rule is a rule
	// about what the suite does, so it reads the suite.
	TestSources = "every tracked test source"
	// Workflows is every tracked workflow file. A rule about what a step
	// pins reads the steps.
	Workflows = "every tracked workflow file"
	// TrackedTextOutsideTheVersionRegister is every tracked text file
	// except the registers a version is allowed to appear in. The
	// exclusion is what makes the rule decidable at all: the file holding
	// the constant carries it by definition, so a population that included
	// it would refuse the original along with every copy.
	TrackedTextOutsideTheVersionRegister = "every tracked text file, outside the register that holds the version"
	// BuildInputs is every tracked file the build reads to render a page.
	// It is the population where a value typed by hand is a second
	// definition of something: a value in a document is prose, a value in a
	// test is what a case needs in order to fail, and a value in a golden
	// file is a copy of output rather than a source of it. Only what the
	// build reads can put a wrong value on a page.
	BuildInputs = "every tracked file the build reads to render a page"
)

// vocabularyFile is the path excluded from TrackedText, and it is derived from
// nothing: it is this file. A rule that guessed at its own path would stop
// excluding it the day the package moved.
const vocabularyFile = "internal/invariant/invariant.go"

// Rule is one invariant.
type Rule struct {
	ID      string
	Subject string
	Reason  string
	Refuses string
	// decide returns one detail per violation in body, or nothing.
	decide func(body []byte) []string
}

// Owed is a rule this gate is meant to carry and cannot yet, because what it
// would read is not in the tree. It is printed by every run.
type Owed struct {
	ID      string
	Waiting string
}

var (
	htmlElement  = regexp.MustCompile(`(?is)<html\b[^>]*>`)
	langAttr     = regexp.MustCompile(`(?is)\blang\s*=\s*"([^"]*)"`)
	titleElement = regexp.MustCompile(`(?is)<title\b[^>]*>(.*?)</title>`)
	scriptSrc    = regexp.MustCompile(`(?is)<script\b[^>]*\bsrc\s*=`)
	unfinished   = regexp.MustCompile(`TODO|FIXME`)
	imgElement   = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	srcAttr      = regexp.MustCompile(`(?is)\bsrc\s*=\s*"([^"]*)"`)
	widthAttr    = regexp.MustCompile(`(?is)\bwidth\s*=\s*"([^"]*)"`)
	heightAttr   = regexp.MustCompile(`(?is)\bheight\s*=\s*"([^"]*)"`)
	// The hex forms CSS reads, longest first so that the six digits of a
	// full colour are not matched as a four-digit one with a stray digit
	// after it. The shorthand forms are in the set because they spell
	// published values exactly: #FFFFFF is one of them and #fff is the same
	// colour.
	hexColour         = regexp.MustCompile(`(?i)#(?:[0-9a-f]{8}|[0-9a-f]{6}|[0-9a-f]{4}|[0-9a-f]{3})\b`)
	fragmentReference = regexp.MustCompile(`(?is)\b(?:href|src)\s*=\s*"#[^"]*"`)
	// A run of digits separated by dots, which is every version-shaped thing
	// in a line. The row compares each whole run against the one version
	// this repository holds rather than searching for that version inside a
	// line, so a longer version spelled around it stays a different version
	// and a version at the end of a sentence is still found with the full
	// stop after it.
	versionShaped = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)+`)
	// What a page cites a check by. The attribute is what a machine reads
	// and the name is also written where a reader sees it, both from one
	// value, so a page cannot show a reader one name and this row another.
	citedCheck = regexp.MustCompile(`(?is)\bdata-refused-by\s*=\s*"([^"]*)"`)
)

// markers is the vocabulary the tool-marker rule refuses, lower-cased. It is a
// floor rather than a guarantee: it holds the shapes that actually leak, and it
// will not catch one nobody has written yet.
var markers = []string{
	"generated by chatgpt",
	"generated by copilot",
	"generated by an ai",
	"generated with claude",
	"co-authored-by: claude",
	"this file was generated by an assistant",
}

// allowedOrigins is the whole allowlist, and it is the project's own domain and
// nothing else. The name is the one decisions/0008-the-url-shape.md gives the
// site root rather than whichever spelling happens to answer on the day, so a
// second host arrives here as a change somebody argues rather than as a
// reference nobody noticed.
//
// It is a list holding one entry rather than a constant, because the shape of
// the rule is a comparison against a set, and record 0011 takes the position
// that the set has one member rather than that the comparison is a special case.
var allowedOrigins = []string{
	"flowfin.dev",
}

// affiliationNotice is the sentence every produced page has to carry. It is
// written into the template, which is where a reader meets it, and it is
// repeated here because a rule refusing a page that lost it has to know what it
// is looking for. The two copies are held to each other by the row: a template
// whose wording drifts reds every page until this line moves with it, so the
// second copy cannot go quietly stale the way a second copy usually does.
//
// It is compared with runs of whitespace collapsed, so the sentence may be
// wrapped in the template without the row deciding on the line breaks.
const affiliationNotice = "Flowfin is not affiliated with the Jellyfin project."

// Rules is the table. The order is the order a run reports them in.
func Rules() []Rule {
	return []Rule{
		{
			ID:      "page-declares-its-language",
			Subject: ProducedPages,
			Reason:  "a page with no language is read aloud in whichever one the reader's software guessed, and the guess is wrong for exactly the readers who depend on it",
			Refuses: "a produced page with no html element, or one whose lang attribute is missing or empty",
			decide:  decideLang,
		},
		{
			ID:      "page-carries-a-title",
			Subject: ProducedPages,
			Reason:  "the title is what a search result, a tab and a shared link show, so a page without one is a page nobody can tell apart from another",
			Refuses: "a produced page with no title element, or one holding only whitespace",
			decide:  decideTitle,
		},
		{
			ID:      "page-carries-the-affiliation-notice",
			Subject: ProducedPages,
			Reason:  "this project uses another project's name on every page it produces and is not affiliated with it, and a notice a page can ship without is a notice that reaches the pages somebody remembered",
			Refuses: "a produced page that does not carry the affiliation notice",
			decide:  decideAffiliation,
		},
		{
			ID:      "page-fetches-no-script",
			Subject: ProducedPages,
			Reason:  "the budget puts required scripting at zero bytes, and a script element with a source is a request this site did not have to make and a party that learns who is reading",
			Refuses: "a produced page carrying a script element with a src attribute, wherever it points",
			decide:  decideScriptSrc,
		},
		{
			ID:      "image-carries-its-own-dimensions",
			Subject: ProducedPages,
			Reason:  "the budget puts layout shift at exactly zero, and the usual cause of a page missing it is an image whose size the browser only learns once the bytes have arrived, so everything under it moves when they do",
			Refuses: "a produced page carrying an image element without a usable width and height on it",
			decide:  decideImageDimensions,
		},
		{
			ID:      "page-cites-only-checks-that-exist",
			Subject: ProducedPages,
			Reason:  "a page that says a check refuses a violation of what it claims is worth the name it gives, and a row renamed or taken out leaves the sentence standing on the page reading as a property that nothing decides, which is the one defect on a privacy page that a reader cannot see",
			Refuses: "a produced page citing a check this gate does not decide, or citing one with no name at all",
			decide:  decideCitedChecks,
		},
		{
			ID:      "output-carries-no-unfinished-marker",
			Subject: ProducedFiles,
			Reason:  "a note to the author is a note to every reader once it is served, and it is the kind of thing nobody greps for after the fact",
			Refuses: "a produced file containing TODO or FIXME",
			decide:  decideUnfinished,
		},
		{
			ID:      "output-references-no-domain-outside-the-allowlist",
			Subject: ProducedFiles,
			Reason:  "a subresource served from somebody else's domain is a round trip the reader pays for and a record of who read what, handed to a party the reader did not choose, and a promise that this does not happen cannot rest on nobody having added a font or a badge yet",
			Refuses: "a produced file fetching a stylesheet, a font, an image, a script or anything else from a host that is not on the allowlist, while leaving a link a reader clicks alone",
			decide:  decideForeignOrigin,
		},
		{
			ID:      "output-carries-no-expired-reporting-route",
			Subject: ProducedFiles,
			Reason:  "an expiry that has passed tells somebody who found a problem that the route was abandoned, which is worse than the file not being there, and it is the one thing in the output that goes wrong by sitting still",
			Refuses: "a produced file whose expiry is not in the future, or whose expiry is not a moment",
			decide:  decideExpiry,
		},
		{
			ID:      "page-parses",
			Subject: ProducedPages,
			Reason:  "a browser recovers from broken markup, so a template can produce an unclosed element for months while every page still renders and nothing says a word about it",
			Refuses: "a produced page with an element that is never closed, an end tag that closes something else, or a tag that never ends",
			decide:  decideMarkup(markup.Structure),
		},
		{
			ID:      "page-uses-no-identifier-twice",
			Subject: ProducedPages,
			Reason:  "an identifier is what a link, a label and an assistive technology reach an element by, and two elements answering to one name means a reference reaches whichever the reader's software picked",
			Refuses: "a produced page carrying one identifier on two elements, or an empty one",
			decide:  decideMarkup(markup.Identity),
		},
		{
			ID:      "page-skips-no-heading-level",
			Subject: ProducedPages,
			Reason:  "a reader moving through a page by heading is reading the outline, and a level that jumps leaves them unable to tell what is under what while the page looks right to everybody else",
			Refuses: "a produced page whose heading level rises by more than one",
			decide:  decideMarkup(markup.Heading),
		},
		{
			ID:      "page-image-carries-alternative-text",
			Subject: ProducedPages,
			Reason:  "an image with no alternative text is a hole in the page for anybody who cannot see it, and an image that genuinely says nothing carries an empty one deliberately rather than none at all",
			Refuses: "a produced page carrying an image with no alt attribute",
			decide:  decideMarkup(markup.Alt),
		},
		{
			ID:      "page-names-every-control",
			Subject: ProducedPages,
			Reason:  "a control with nothing naming it asks somebody to fill in a field without telling them what goes in it, and the page looks complete to whoever wrote it",
			Refuses: "a produced page carrying a form control with no accessible name on it",
			decide:  decideMarkup(markup.Label),
		},
		{
			ID:      "tracked-text-names-no-tool",
			Subject: TrackedText,
			Reason:  "a marker naming what produced a file says nothing about the file and outlives every reason it was written, and removing one after it has landed rewrites history rather than editing a line",
			Refuses: "a tracked text file carrying one of the produced-by markers in the vocabulary",
			decide:  decideMarkers,
		},
		{
			ID:      "workflow-step-carries-no-version-literal",
			Subject: Workflows,
			Reason:  "a version fetched at run time is in neither ecosystem the updater covers, so one written into a step is a pin nobody watches, and a pin nobody watches is not a stable version but an old one",
			Refuses: "a version literal in a workflow step, outside a comment and outside the action reference that carries its own updater",
			decide:  decideWorkflowVersions,
		},
		{
			ID:      "design-tokens-live-in-exactly-one-file",
			Subject: BuildInputs,
			Reason:  "a colour typed into a template is a second definition of a value published somewhere else, and the day the published one moves the page goes on rendering the old one perfectly, so nobody sees it",
			Refuses: "a colour written into what the build reads, in any of the hex forms the published file uses, outside a fragment reference",
			decide:  decideTypedColour,
		},
		{
			ID:      "version-lives-in-exactly-one-file",
			Subject: TrackedTextOutsideTheVersionRegister,
			Reason:  "a version written a second time is right on the day it is typed, and the copy that goes stale is the one a reader takes the version from rather than the one the release run reads, so the tree announces a release nobody tagged and nothing about the tag says otherwise",
			Refuses: "the version this repository releases under, written into a tracked file outside the register that holds it",
			decide:  decideSecondVersion,
		},
		{
			ID:      "test-opens-no-window",
			Subject: TestSources,
			Reason:  "a suite that needs a screen is a suite that is run rarely, and a suite that is run rarely is not a gate",
			Refuses: "a test source naming a driver that opens a window or drives a real browser",
			decide:  literal(windowDrivers, "names a driver that opens a window"),
		},
		{
			ID:      "test-needs-no-display-server",
			Subject: TestSources,
			Reason:  "a test that reaches for a display passes on the machine it was written on and fails on every runner, and the failure reads as a broken test rather than as a broken assumption",
			Refuses: "a test source naming a display server or the variable that points at one",
			decide:  literal(displayServers, "names a display server"),
		},
		{
			ID:      "test-binds-only-loopback",
			Subject: TestSources,
			Reason:  "binding a machine's own interface address rather than loopback is what makes a desktop firewall ask an administrator for permission, and that dialog is answered per executable path, so answering it settles nothing for the next build directory",
			Refuses: "a listen address in a test source whose host is neither loopback nor absent by being loopback",
			decide:  decideBind,
		},
		{
			ID:      "test-writes-no-certificate-store",
			Subject: TestSources,
			Reason:  "a certificate store is machine-wide state, so a test that writes one changes the machine for everything else on it and asks for elevation on the way",
			Refuses: "a test source naming a tool that installs or trusts a certificate",
			decide:  literal(certificateStores, "names a tool that writes a certificate store"),
		},
		{
			ID:      "test-asks-for-no-elevation",
			Subject: TestSources,
			Reason:  "elevation does not fail as a red test but as a consent prompt that takes the screen away from whoever was working, and a test that does that once is a test people learn to skip",
			Refuses: "a test source naming a way to ask for administrator or root",
			decide:  pattern(elevation, "asks for elevation"),
		},
		{
			ID:      "test-needs-nothing-outside-the-toolchain",
			Subject: TestSources,
			Reason:  "a package that is neither in the toolchain nor part of this module is a thing somebody has to install before the suite runs, and a suite with a setup step is a suite that is run rarely",
			Refuses: "an import in a test source that is neither standard library nor inside this module",
			decide:  decideImports,
		},
	}
}

// The vocabularies the headless rows read. Each is a floor rather than a
// guarantee: it holds the shapes that actually arrive, and it will not catch one
// nobody has written yet. They are declared here, in the file already excluded
// from the tracked-text population, so no row refuses the list it reads from.
var (
	windowDrivers = []string{
		"github.com/chromedp/chromedp",
		"github.com/go-rod/rod",
		"github.com/playwright-community/playwright-go",
		"github.com/tebeka/selenium",
		"github.com/webview/webview",
		"fyne.io/fyne",
	}
	displayServers = []string{
		"xvfb",
		"wayland_display",
		`"display"`,
		"display=:",
	}
	certificateStores = []string{
		"dev-certs",
		"certutil",
		"add-trusted-cert",
		"update-ca-certificates",
		"cert:\\localmachine",
	}
	// modulePath is this module, and an import under it is this repository
	// rather than something to install.
	modulePath = "github.com/Flowfin/site"
)

var (
	// Three dot-separated numbers, with whatever release suffix follows,
	// bounded so that a longer number is not read as a version hiding
	// inside it.
	versionLiteral = regexp.MustCompile(`(^|[^0-9.])([0-9]+\.[0-9]+\.[0-9]+(?:[-.][0-9A-Za-z-]+)*)`)
	elevation      = regexp.MustCompile(`(?i)\b(sudo|gsudo|pkexec|runas|netsh|schtasks|sc\.exe)\b|-verb\s+runas`)
	listenCall     = regexp.MustCompile(`(?s)Listen[A-Za-z]*\(\s*(?:"[a-z0-9]+"\s*,\s*)?"([^"]*)"`)
	importLine     = regexp.MustCompile(`(?m)^\s*(?:_\s+|\.\s+|[a-zA-Z0-9_]+\s+)?"([^"]+)"\s*$`)
)

// What the origin row reads. Markup is read as elements and attributes rather
// than as one pattern per shape, because the question is what the browser
// fetches and that is a property of which attribute sits on which element.
var (
	htmlTag      = regexp.MustCompile(`(?is)<([a-z][a-z0-9-]*)((?:[^>"']|"[^"]*"|'[^']*')*)>`)
	tagAttribute = regexp.MustCompile(`(?is)([a-z][a-z0-9-]*)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	cssURL       = regexp.MustCompile(`(?is)url\(\s*["']?([^"')\s]+)`)
	cssImport    = regexp.MustCompile(`(?is)@import\s+["']([^"']+)["']`)
	// A reference with an authority, whatever the scheme. mailto: and data:
	// have none, so neither reaches a domain and neither is matched here.
	authority = regexp.MustCompile(`(?i)^(?:[a-z][a-z0-9+.-]*:)?//([^/?#]+)`)
)

// subresourceAttributes are the attributes whose value the browser fetches
// without the reader having done anything. href is not among them because its
// meaning depends on the element it sits on, which decideForeignOrigin handles
// separately.
var subresourceAttributes = map[string]bool{
	"src":         true,
	"srcset":      true,
	"imagesrcset": true,
	"poster":      true,
	"data":        true,
	"action":      true,
	"formaction":  true,
}

// readerFollows are the elements on which href is a link a reader clicks rather
// than something the page fetches. That is the one exception in this rule: a
// link is a choice and a subresource is not.
var readerFollows = map[string]bool{
	"a":    true,
	"area": true,
}

// literal returns a decide that refuses any of the vocabulary, compared
// lower-cased so a capitalisation does not walk through.
func literal(vocabulary []string, detail string) func([]byte) []string {
	return func(body []byte) []string {
		lower := bytes.ToLower(body)
		var details []string
		for _, v := range vocabulary {
			at := 0
			for {
				i := bytes.Index(lower[at:], []byte(v))
				if i < 0 {
					break
				}
				at += i
				details = append(details, fmt.Sprintf("line %d %s", lineOf(body, at), detail))
				at += len(v)
			}
		}
		return details
	}
}

// pattern returns a decide that refuses every match of re.
func pattern(re *regexp.Regexp, detail string) func([]byte) []string {
	return func(body []byte) []string {
		var details []string
		for _, loc := range re.FindAllIndex(body, -1) {
			details = append(details, fmt.Sprintf("line %d %s", lineOf(body, loc[0]), detail))
		}
		return details
	}
}

// decideBind reads the address a listen call was given. An empty host means
// every interface on the machine, which is the shape that raises the prompt, so
// it is refused as loudly as a written-out address.
func decideBind(body []byte) []string {
	var details []string
	for _, m := range listenCall.FindAllSubmatchIndex(body, -1) {
		addr := string(body[m[2]:m[3]])
		host := addr
		if i := strings.LastIndex(addr, ":"); i >= 0 {
			host = addr[:i]
		}
		host = strings.Trim(host, "[]")
		switch strings.ToLower(host) {
		case "127.0.0.1", "localhost", "::1":
			continue
		case "":
			details = append(details, fmt.Sprintf("line %d binds %q, which is every interface on the machine rather than loopback", lineOf(body, m[0]), addr))
		default:
			details = append(details, fmt.Sprintf("line %d binds %q, which is not loopback", lineOf(body, m[0]), addr))
		}
	}
	return details
}

// decideWorkflowVersions refuses a version written into a workflow step.
//
// Two things it deliberately does not read. A comment, because the version
// beside an action reference is where the updater writes what the commit it
// pinned was, and refusing that would refuse the pinning convention itself. And
// a `uses:` line, for the same reason one step further: an action is one of the
// two ecosystems the updater already covers, so it is watched and this rule is
// about the pins that are not.
//
// The bound is the shape of the literal rather than its meaning. Three
// dot-separated numbers is what a fetched version looks like and it is what
// somebody writes; a version spelled any other way walks through, which is why
// the file is the authority for the set rather than this row.
func decideWorkflowVersions(body []byte) []string {
	var details []string
	for i, line := range strings.Split(string(body), "\n") {
		text := withoutComment(line)
		if strings.HasPrefix(strings.TrimSpace(text), "uses:") {
			continue
		}
		for _, m := range versionLiteral.FindAllStringSubmatch(text, -1) {
			details = append(details, fmt.Sprintf(
				"line %d writes the version %s into a step, and %s is where a version no updater watches is declared and read from",
				i+1, m[2], pins.File))
		}
	}
	return details
}

// withoutComment drops what YAML reads as a comment, which is a hash preceded
// by whitespace or standing at the start of the line.
func withoutComment(line string) string {
	for i, r := range line {
		if r != '#' {
			continue
		}
		if i == 0 {
			return ""
		}
		if prev := line[i-1]; prev == ' ' || prev == '\t' {
			return line[:i]
		}
	}
	return line
}

// decideMarkup makes the row for one kind of problem. The page is walked once
// per row, which is a walk of a few kilobytes and is worth what it buys: each
// property is a row of its own, with its own reason and its own refusal, rather
// than one row that says a page is wrong somehow.
func decideMarkup(kind string) func(body []byte) []string {
	return func(body []byte) []string {
		var details []string
		for _, p := range markup.Of(kind, markup.Read(body)) {
			details = append(details, p.String())
		}
		return details
	}
}

// decideExpiry refuses a produced file whose expiry has passed.
//
// This is the one row that reads the clock, and the distinction is worth
// stating because the rest of this repository refuses one. A build may not read
// the clock: what it writes has to be the same bytes twice from one commit.
// Whether a date has passed is a question about the world rather than about the
// tree, so it is asked here, over what the build wrote, and the answer is
// allowed to change on a day when nothing was committed. That is the whole
// point of an expiry.
//
// A file carrying no expiry is not this row's subject and is passed over. Today
// exactly one produced file carries the field.
func decideExpiry(body []byte) []string {
	at, err := security.ExpiryOf(body)
	switch {
	case errors.Is(err, security.ErrNoExpiry):
		return nil
	case err != nil:
		return []string{"this file " + err.Error()}
	}
	if !at.After(time.Now()) {
		return []string{fmt.Sprintf(
			"this file expired on %s, and %s is where the day it was last confirmed is moved forward",
			at.Format(time.RFC3339), security.File)}
	}
	return nil
}

// decideTypedColour refuses a colour written into what the build reads.
//
// The design system is published as data and this repository vendors a copy of
// it, which is decisions/0007-where-the-design-tokens-live.md: the page and the
// stylesheet are generated from that copy and neither is typed. A hex value
// typed into a template is the second definition that record refuses, and it is
// the one mistake somebody actually makes, because the value is on the screen
// in front of them while they are writing the markup that needs it.
//
// It reads the shape rather than the published values. A rule comparing against
// the copy would refuse the values that are in it today and pass the ones
// somebody typed that are not, which is the wrong half: what is wrong with a
// typed colour is that it was typed, whether or not it is currently right.
//
// A fragment reference is dropped first, because `href="#abc"` is an address
// and not a colour, and the two are the same bytes.
// versionRegisters is every tracked file the version may appear in. It holds
// the file that defines it, and it is a list rather than a comparison against
// one constant because the shape of the rule is a comparison against a set: a
// second register arrives as an entry here rather than as a branch beside the
// row, and a file listing released versions is exactly that case.
var versionRegisters = []string{
	version.SourceFile,
}

func versionRegister(name string) bool {
	for _, r := range versionRegisters {
		if name == r {
			return true
		}
	}
	return false
}

// decideSecondVersion refuses the version this repository releases under,
// written anywhere the release run does not read it from.
//
// It compares whole version-shaped runs rather than searching for a substring,
// because a pinned tool and this repository are unrelated facts that happen to
// be spelled with digits, and a row that could not tell them apart is a row
// somebody switches off.
//
// What it reaches is the copy at the moment it is written, which is the moment
// it can still be repaired cheaply. A copy made before a version moved carries
// the old number and reads to this row as a different version, so the row does
// not find yesterday's stale sentence and is not what would.
func decideSecondVersion(body []byte) []string {
	var details []string
	for i, line := range strings.Split(string(body), "\n") {
		for _, m := range versionShaped.FindAllString(line, -1) {
			if m != version.Number {
				continue
			}
			details = append(details, fmt.Sprintf(
				"line %d writes the version %s, and %s is the one file it is read from",
				i+1, version.Number, version.SourceFile))
		}
	}
	return details
}

func decideTypedColour(body []byte) []string {
	var details []string
	for i, line := range strings.Split(string(body), "\n") {
		text := fragmentReference.ReplaceAllString(line, "")
		for _, m := range hexColour.FindAllString(text, -1) {
			details = append(details, fmt.Sprintf(
				"line %d writes the colour %s, and %s is the one file a colour is read from",
				i+1, m, tokens.File))
		}
	}
	return details
}

// decideImports refuses an import that is neither standard library nor inside
// this module. A standard library path has no dot in its first element, which is
// the rule the toolchain itself uses to tell the two apart.
func decideImports(body []byte) []string {
	var details []string
	for _, m := range importLine.FindAllSubmatchIndex(body, -1) {
		p := string(body[m[2]:m[3]])
		first := p
		if i := strings.Index(p, "/"); i >= 0 {
			first = p[:i]
		}
		if !strings.Contains(first, ".") {
			continue
		}
		if p == modulePath || strings.HasPrefix(p, modulePath+"/") {
			continue
		}
		details = append(details, fmt.Sprintf("line %d imports %s, which is neither standard library nor inside this module", lineOf(body, m[0]), p))
	}
	return details
}

// Owing is what this gate is meant to refuse and cannot decide yet. Each entry
// names what has to exist first. Printed by every run, passing or not.
func Owing() []Owed {
	return []Owed{
		{
			ID:      "image-dimensions-match-the-file",
			Waiting: "the first image the build writes, in #69. The row above refuses a produced image with no usable dimensions on it; this one is the other half, that the numbers written are the ones the image file carries rather than numbers somebody typed, and the build writes no image today, so there is nothing on either side of that comparison",
		},
	}
}

// Run decides every rule against the tree at root and writes what it examined
// to log. It builds the site into a directory it throws away, so what the page
// rules read is what a build produces rather than whatever is sitting in the
// output directory from an earlier one.
func Run(root string, log io.Writer) error {
	rules := Rules()
	ids := make([]string, len(rules))
	for i, r := range rules {
		ids[i] = r.ID
	}
	fmt.Fprintf(log, "invariants: %d rule(s), in order: %s\n", len(rules), strings.Join(ids, ", "))

	populations, err := gather(root)
	if err != nil {
		return err
	}

	refused := 0
	for _, r := range rules {
		files, ok := populations[r.Subject]
		if !ok {
			return fmt.Errorf("rule %s reads %q, which is not a population this run gathered", r.ID, r.Subject)
		}
		var details []string
		for _, f := range files {
			for _, d := range r.decide(f.body) {
				details = append(details, fmt.Sprintf("%s: %s", f.name, d))
			}
		}
		if len(details) > 0 {
			refused++
			fmt.Fprintf(log, "  %s: REFUSED, %d violation(s)\n", r.ID, len(details))
			fmt.Fprintf(log, "    it refuses %s\n", r.Refuses)
			fmt.Fprintf(log, "    because %s\n", r.Reason)
			for _, d := range details {
				fmt.Fprintf(log, "    %s\n", d)
			}
			continue
		}
		if len(files) == 0 {
			fmt.Fprintf(log, "  %s: %s held no file, so this rule examined nothing\n", r.ID, r.Subject)
			continue
		}
		fmt.Fprintf(log, "  %s: ok, %d file(s) of %s\n", r.ID, len(files), r.Subject)
	}

	for _, o := range Owing() {
		fmt.Fprintf(log, "  %s: not decided, waiting on %s\n", o.ID, o.Waiting)
	}

	fmt.Fprintf(log, "%d rule(s) decided, %d owed and not decided.\n", len(rules), len(Owing()))
	if refused > 0 {
		return fmt.Errorf("invariants: %d rule(s) refused this tree", refused)
	}
	return nil
}

type file struct {
	name string
	body []byte
}

// gather reads each population once. The build goes into a temporary directory
// so that a run leaves nothing behind that a reader might mistake for the result
// of `go run . build`.
func gather(root string) (map[string][]file, error) {
	tmp, err := os.MkdirTemp("", "site-invariants-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	out := filepath.Join(tmp, site.OutputDir)
	written, err := site.Build(root, out, io.Discard)
	if err != nil {
		return nil, fmt.Errorf("the build refused, so there is no output to read: %w", err)
	}
	if len(written) == 0 {
		return nil, fmt.Errorf("the build wrote no file, and there is nothing to decide a page rule against")
	}

	// The build labels what it wrote with the output directory it was given,
	// and it was given an absolute one so that nothing lands beside a
	// reader's real output. Names are reported back as if it had been the
	// tree's own dist/, because that is the path a person opens.
	label := filepath.ToSlash(out) + "/"

	var produced, pages []file
	for _, w := range written {
		rel := strings.TrimPrefix(filepath.ToSlash(w), label)
		b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		f := file{name: path.Join(site.OutputDir, rel), body: b}
		produced = append(produced, f)
		if strings.HasSuffix(rel, ".html") {
			pages = append(pages, f)
		}
	}

	tracked, err := trackedText(root)
	if err != nil {
		return nil, err
	}

	// The rule about where a colour is read from is a rule about there being
	// exactly one such file, and a run that decided it against a tree
	// carrying none would report that no second definition was found in a
	// tree with no first one. So the copy being absent is refused here, by
	// name, rather than passing as a row with nothing to compare.
	if len(tracked.securitySources) == 0 {
		return nil, fmt.Errorf("%s is not tracked in this tree, so the build wrote no %s and the row about an expired reporting route has nothing to read", security.File, security.Path)
	}
	if len(tracked.tokenCopies) == 0 {
		return nil, fmt.Errorf("%s is not tracked in this tree, and the row about where a colour is read from is a row about there being exactly one such file", tokens.File)
	}

	return map[string][]file{
		ProducedPages:                        pages,
		ProducedFiles:                        produced,
		TrackedText:                          tracked.text,
		TestSources:                          tracked.tests,
		Workflows:                            tracked.workflows,
		BuildInputs:                          tracked.buildInputs,
		TrackedTextOutsideTheVersionRegister: tracked.outsideVersion,
	}, nil
}

// tracked is what one walk of the index produces. The populations are cut from
// one listing rather than from one walk each, because two listings of the same
// index taken a moment apart is a difference nobody would look for.
type tracked struct {
	text            []file
	outsideVersion  []file
	tests           []file
	workflows       []file
	buildInputs     []file
	tokenCopies     []file
	securitySources []file
}

// workflowDir is where a workflow has to live for the server to run it, so it
// is what a rule about workflow files reads.
const workflowDir = ".github/workflows/"

// trackedText reads every tracked file that is not binary, and separates out the
// test sources. It asks git rather than walking, because the rule is about what
// this repository carries and a walk also reads whatever happens to be sitting in
// the working directory.
func trackedText(root string) (tracked, error) {
	cmd := exec.Command("git", "ls-files", "-z")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return tracked{}, fmt.Errorf("listing the tracked files: %w", err)
	}

	var found tracked
	for _, name := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
		if name == "" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			if os.IsNotExist(err) {
				// A tracked path that is not in the working tree is a
				// checkout somebody is in the middle of, not a violation.
				continue
			}
			return tracked{}, err
		}
		if bytes.IndexByte(b, 0) >= 0 {
			continue
		}
		f := file{name: name, body: b}
		if strings.HasSuffix(name, "_test.go") {
			found.tests = append(found.tests, f)
		}
		if strings.HasPrefix(name, workflowDir) {
			found.workflows = append(found.workflows, f)
		}
		if name != vocabularyFile {
			found.text = append(found.text, f)
		}
		if !versionRegister(name) {
			found.outsideVersion = append(found.outsideVersion, f)
		}
		if name == tokens.File {
			found.tokenCopies = append(found.tokenCopies, f)
			continue
		}
		if name == security.File {
			found.securitySources = append(found.securitySources, f)
			continue
		}
		if strings.HasPrefix(name, site.TemplatesDir+"/") || strings.HasPrefix(name, site.ContentDir+"/") {
			found.buildInputs = append(found.buildInputs, f)
		}
	}
	return found, nil
}

func decideLang(body []byte) []string {
	el := htmlElement.Find(body)
	if el == nil {
		return []string{"there is no html element on this page"}
	}
	m := langAttr.FindSubmatch(el)
	if m == nil {
		return []string{"the html element carries no lang attribute"}
	}
	if strings.TrimSpace(string(m[1])) == "" {
		return []string{"the lang attribute on the html element is empty"}
	}
	return nil
}

func decideTitle(body []byte) []string {
	m := titleElement.FindSubmatch(body)
	if m == nil {
		return []string{"there is no title element on this page"}
	}
	if strings.TrimSpace(string(m[1])) == "" {
		return []string{"the title element holds only whitespace"}
	}
	return nil
}

// decideAffiliation refuses a produced page that does not carry the notice.
//
// It reads the output rather than the template, which is the case that matters:
// the value of putting the sentence in one file is that every page rendered
// through it carries the sentence, and what proves that is the pages rather than
// the file they came from. A second template, or a page written by hand, is
// exactly what this catches and is invisible to a rule that read the template.
func decideAffiliation(body []byte) []string {
	if strings.Contains(collapseSpace(string(body)), affiliationNotice) {
		return nil
	}
	return []string{"this page carries no affiliation notice, and the sentence it has to carry is " + affiliationNotice}
}

// collapseSpace turns every run of whitespace into one space, so a sentence
// wrapped across lines in a template reads the same to the row as one written
// on a single line.
func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func decideScriptSrc(body []byte) []string {
	var details []string
	for _, loc := range scriptSrc.FindAllIndex(body, -1) {
		details = append(details, fmt.Sprintf("line %d carries a script element with a src attribute", lineOf(body, loc[0])))
	}
	return details
}

// decideImageDimensions refuses an image on a produced page that does not carry
// the space it is going to take.
//
// The property the budget states is layout shift of exactly zero, and #35
// measures that in a browser against the built output served. Measuring is the
// honest last line and it is the expensive one, and the usual cause of a
// failure is one thing that a reading of the markup decides for nothing: an
// image the browser has to fetch before it knows how tall the box is. So the
// cause is refused here and the browser run stays as the thing that catches
// what a reading cannot see.
//
// An attribute that is present and empty is refused exactly as an absent one
// is. That is the mistake somebody actually makes: a template writing
// width="{{ .Width }}" against a value that is not there produces an attribute
// a grep for the word finds and a browser ignores, which is the shape a rule
// reading only for the name of the attribute would pass.
//
// The bound is that this reads the produced markup and nothing else. Whether
// the number on the element is the number the image file carries is a
// comparison against a file this build does not write yet, and it is owed
// rather than decided here.
func decideImageDimensions(body []byte) []string {
	var details []string
	for _, loc := range imgElement.FindAllIndex(body, -1) {
		tag := body[loc[0]:loc[1]]
		where := fmt.Sprintf("line %d, the image %s", lineOf(body, loc[0]), sourceOf(tag))
		for _, d := range []struct {
			name string
			re   *regexp.Regexp
		}{{"width", widthAttr}, {"height", heightAttr}} {
			switch value, present := attribute(tag, d.re); {
			case !present:
				details = append(details, fmt.Sprintf("%s carries no %s attribute", where, d.name))
			case !isDimension(value):
				details = append(details, fmt.Sprintf("%s carries the %s %q, which is not a number of pixels a browser can reserve space from", where, d.name, value))
			}
		}
	}
	return details
}

// sourceOf names the image a detail is about, so a page with several says which
// one. An element with no source at all is named as that rather than as an
// empty string, because the two read identically in a message and are different
// mistakes.
func sourceOf(tag []byte) string {
	if src, ok := attribute(tag, srcAttr); ok && src != "" {
		return src
	}
	return "with no source on it"
}

// attribute reads one attribute off an element, and reports whether it was
// there at all separately from what it held.
func attribute(tag []byte, re *regexp.Regexp) (string, bool) {
	m := re.FindSubmatch(tag)
	if m == nil {
		return "", false
	}
	return string(m[1]), true
}

// isDimension answers whether a value is a whole number of pixels. A browser
// reserves space from a bare number, and anything else on these two attributes
// is either ignored or is a unit the attribute does not take.
func isDimension(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// decideForeignOrigin refuses every reference in a produced file that would
// make the reader's browser reach a host the project does not control.
//
// It reads the output rather than the templates, which is the case that matters
// most: a reference that only appears once some data value takes a particular
// shape is invisible in a template and present in the page.
//
// Two bounds, stated rather than left to be found. A cross-origin script trips
// this row and the script row both, because they refuse different things about
// the same element: that row refuses a fetched script wherever it points, this
// one refuses a foreign host whatever it serves. And the stylesheet vocabulary
// is read over the whole file rather than only inside a style element, so an
// absolute url() spelled out in running text would be refused. That is not a
// link a reader can click, and the alternative is a rule that misses a stylesheet
// the build wrote as a file of its own.
func decideForeignOrigin(body []byte) []string {
	var details []string
	report := func(at int, verb, ref, host string) {
		details = append(details, fmt.Sprintf("line %d %s %s, whose host %s is not on the allowlist", lineOf(body, at), verb, ref, host))
	}

	for _, tag := range htmlTag.FindAllSubmatchIndex(body, -1) {
		element := strings.ToLower(string(body[tag[2]:tag[3]]))
		attrs := body[tag[4]:tag[5]]
		for _, a := range tagAttribute.FindAllSubmatchIndex(attrs, -1) {
			name := strings.ToLower(string(attrs[a[2]:a[3]]))
			fetched := subresourceAttributes[name] || (name == "href" && !readerFollows[element])
			if !fetched {
				continue
			}
			for _, ref := range references(name, attributeValue(attrs, a)) {
				if host, ok := foreignHost(ref); ok {
					report(tag[4]+a[0], "fetches", ref, host)
				}
			}
		}
	}

	for _, m := range cssURL.FindAllSubmatchIndex(body, -1) {
		ref := string(body[m[2]:m[3]])
		if host, ok := foreignHost(ref); ok {
			report(m[0], "loads", ref, host)
		}
	}
	for _, m := range cssImport.FindAllSubmatchIndex(body, -1) {
		ref := string(body[m[2]:m[3]])
		if host, ok := foreignHost(ref); ok {
			report(m[0], "imports", ref, host)
		}
	}

	return details
}

// attributeValue returns whichever of the three quotings the attribute was
// written in. An unquoted value is the third, and it is read because a page is
// judged as it was produced rather than as somebody meant to produce it.
func attributeValue(attrs []byte, a []int) string {
	for _, g := range [][2]int{{4, 5}, {6, 7}, {8, 9}} {
		if a[g[0]] >= 0 {
			return string(attrs[a[g[0]]:a[g[1]]])
		}
	}
	return ""
}

// references splits an attribute into the addresses it actually names. A
// candidate list holds several, each followed by a descriptor, and a list whose
// second entry points elsewhere is exactly the reference a reader of the first
// entry would miss.
func references(name, value string) []string {
	if name != "srcset" && name != "imagesrcset" {
		return []string{strings.TrimSpace(value)}
	}
	var out []string
	for _, candidate := range strings.Split(value, ",") {
		if f := strings.Fields(candidate); len(f) > 0 {
			out = append(out, f[0])
		}
	}
	return out
}

// foreignHost reports the host a reference reaches, and whether that host is off
// the allowlist. A reference with no authority reaches whatever served the page,
// so it is not this rule's business.
func foreignHost(ref string) (string, bool) {
	m := authority.FindStringSubmatch(strings.TrimSpace(ref))
	if m == nil {
		return "", false
	}
	host := m[1]
	if i := strings.LastIndex(host, "@"); i >= 0 {
		host = host[i+1:]
	}
	if !strings.HasPrefix(host, "[") {
		if i := strings.Index(host, ":"); i >= 0 {
			host = host[:i]
		}
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if host == "" {
		return "", false
	}
	for _, allowed := range allowedOrigins {
		if host == allowed {
			return "", false
		}
	}
	return host, true
}

// decideCitedChecks refuses a page naming a check this gate does not decide.
//
// The names on a page are a second copy of the table below, and a second copy
// is exactly what goes stale: renaming a row is a change somebody makes in this
// file, and nothing outside it moves, so the sentence on the page goes on
// reading as a property while the name under it answers to nothing. That is
// invisible to a reader, who has no way to ask which names exist.
//
// A row that is owed rather than decided is refused here as well. A page citing
// one would be naming a check the run itself prints as not decided, which is a
// promise wearing a property's clothes, and the register for that on the page is
// the one that names an issue instead.
func decideCitedChecks(body []byte) []string {
	var details []string
	for _, m := range citedCheck.FindAllSubmatch(body, -1) {
		name := strings.TrimSpace(string(m[1]))
		if name == "" {
			details = append(details, "cites a check with no name, and the sentence beside it reads as refused by something")
			continue
		}
		if decided(name) {
			continue
		}
		details = append(details, fmt.Sprintf(
			"cites the check %q, which this gate does not decide, so the sentence beside it reads as a property and is a promise", name))
	}
	return details
}

// decided answers whether a name is a row this gate decides. It reads the table
// rather than a list written beside it, so a row added, renamed or removed
// changes this answer without anybody remembering to.
func decided(name string) bool {
	for _, r := range Rules() {
		if r.ID == name {
			return true
		}
	}
	return false
}

func decideUnfinished(body []byte) []string {
	var details []string
	for _, loc := range unfinished.FindAllIndex(body, -1) {
		details = append(details, fmt.Sprintf("line %d carries %s", lineOf(body, loc[0]), body[loc[0]:loc[1]]))
	}
	return details
}

func decideMarkers(body []byte) []string {
	lower := bytes.ToLower(body)
	var details []string
	for _, m := range markers {
		at := 0
		for {
			i := bytes.Index(lower[at:], []byte(m))
			if i < 0 {
				break
			}
			at += i
			details = append(details, fmt.Sprintf("line %d names a tool that produced it", lineOf(body, at)))
			at += len(m)
		}
	}
	return details
}

// lineOf is one-based, so a detail reads like something a person can open.
func lineOf(body []byte, at int) int {
	return bytes.Count(body[:at], []byte("\n")) + 1
}
