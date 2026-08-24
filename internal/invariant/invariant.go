// SPDX-License-Identifier: AGPL-3.0-or-later

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
//
// A row in the table that found nothing of the kind it judges says the same
// thing one step in. It prints what it read rather than ok, so a set of pages
// carrying none of the thing yet cannot be read as one carrying it correctly.
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
	"strconv"
	"strings"
	"time"

	"github.com/Flowfin/site/internal/budget"
	"github.com/Flowfin/site/internal/changelog"
	"github.com/Flowfin/site/internal/licence"
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
	// SourceFiles is every tracked source this repository authors, cut by
	// extension. The rule about a licence header is a rule about the files
	// somebody copies out of here, and what gets copied is source.
	SourceFiles = "every tracked source file"
	// Workflows is every tracked workflow file. A rule about what a step
	// pins reads the steps.
	Workflows = "every tracked workflow file"
	// TrackedTextOutsideTheVersionRegister is every tracked text file
	// except the registers a version is allowed to appear in. The
	// exclusion is what makes the rule decidable at all: the file holding
	// the constant carries it by definition, and the file describing each
	// release carries a heading per version because that is what it is for,
	// so a population that included either would refuse the register along
	// with every copy.
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
	// Counted names what a row counts, where a row judges a thing a page may
	// carry rather than the page itself. It is what the run prints, so it
	// reads as a plural noun.
	Counted string
	// Only narrows the population to one produced path, where the rule is
	// about one page rather than about all of them. It exists because the
	// speed budget counts the landing page's requests and says nothing about
	// the other pages, so a row applying that line everywhere would refuse
	// more than the record does, which is a rule nobody argued. A row that
	// names a path the build did not write reports that it decided nothing
	// rather than passing, so a page renamed out from under this does not
	// leave a silently green row behind.
	Only string
	// decide returns one detail per violation in body, or nothing.
	decide func(body []byte) []string
	// counts says how many things of the kind this row judges the body
	// holds. A row that carries one and read none of them reports that
	// rather than ok, because a green mark over a population that turned out
	// to be empty reads as a tree the rule was exercised against, and what
	// it says is that nobody has written the thing yet. A row judging the
	// page itself carries none of this: every page is a subject, so its
	// population is the file count the run already prints.
	counts func(body []byte) int
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
	// The two places a browser reads a colour scheme from, in the order it
	// reads them. The element is what answers before a stylesheet has
	// arrived, which is the whole reason the row exists, and the property
	// is the same answer written for a page that has one. Both are matched
	// because refusing a page that declares it in CSS would be a row
	// somebody switches off the day a stylesheet lands.
	colourSchemeMeta = regexp.MustCompile(`(?is)<meta\b[^>]*\bname\s*=\s*"color-scheme"[^>]*>`)
	// What precedes the property is part of the pattern: a media query
	// asking what the reader prefers spells the same word with a prefix on
	// it, and reading that as a declaration would let a page ask the
	// question without ever answering it.
	colourSchemeProperty = regexp.MustCompile(`(?is)(^|[^-a-z])color-scheme\s*:\s*([^;{}"']*)`)
	contentAttr          = regexp.MustCompile(`(?is)\bcontent\s*=\s*"([^"]*)"`)
	imgElement           = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	srcAttr              = regexp.MustCompile(`(?is)\bsrc\s*=\s*"([^"]*)"`)
	widthAttr            = regexp.MustCompile(`(?is)\bwidth\s*=\s*"([^"]*)"`)
	heightAttr           = regexp.MustCompile(`(?is)\bheight\s*=\s*"([^"]*)"`)
	// The hex forms CSS reads, longest first so that the six digits of a
	// full colour are not matched as a four-digit one with a stray digit
	// after it. The shorthand forms are in the set because they spell
	// published values exactly: #FFFFFF is one of them and #fff is the same
	// colour.
	hexColour         = regexp.MustCompile(`(?i)#(?:[0-9a-f]{8}|[0-9a-f]{6}|[0-9a-f]{4}|[0-9a-f]{3})\b`)
	fragmentReference = regexp.MustCompile(`(?is)\b(?:href|src)\s*=\s*"#[^"]*"`)
	// A number carrying one of the units a length is written in on the web.
	// The token file states its lengths as bare numbers of
	// density-independent pixels and says so in its own how-to-read-this
	// block, so what a page writes is that number with a unit stuck to it,
	// and the unit is what tells a length in a stylesheet apart from a
	// number in a sentence. The digits have to touch the unit: a prose
	// sentence putting a space between a figure and a word is not a length,
	// and a word ending in one of these letters carries no digits in front
	// of it.
	cssLength = regexp.MustCompile(`(?i)\b[0-9]+(?:\.[0-9]+)?(?:px|rem|em|ch|vh|vw|pt)\b`)
	// The two declarations the file's font stacks and type weights are
	// spelled as. The property rather than the value, because a stack is a
	// list somebody shortens and a weight is three digits that look like
	// nothing in particular, and neither is recognisable on its own the way
	// a hex colour is. What precedes the name is part of the pattern for the
	// reason the colour scheme property carries one: a hyphen in front of it
	// is a different property.
	fontFamilyProperty = regexp.MustCompile(`(?is)(^|[^-a-z])font-family\s*:`)
	fontWeightProperty = regexp.MustCompile(`(?is)(^|[^-a-z])font-weight\s*:`)
	// A run of digits separated by dots, which is every version-shaped thing
	// in a line. The row compares each whole run against the one version
	// this repository holds rather than searching for that version inside a
	// line, so a longer version spelled around it stays a different version
	// and a version at the end of a sentence is still found with the full
	// stop after it.
	versionShaped = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)+`)
	// The two spellings a server generation arrives in, each recognised by
	// what stands beside the number rather than by the number itself. A run
	// of digits and dots on its own is a line height, a release of somebody
	// else's tool or an address, and a row that read the shape alone would
	// refuse all three and be switched off within the week.
	//
	// The first is how a page states it, which is the server's name and then
	// the generation, with room for the few words a sentence puts between
	// them. The second is how a build manifest declares it, which is the key
	// the published metadata carries, and it is the spelling a roster
	// vendored into this tree would arrive in.
	//
	// Both are bounded to one line, and the vocabulary is a floor rather
	// than a guarantee: a generation written with neither the server's name
	// nor the manifest's key beside it is refused by nothing here.
	statedGeneration   = regexp.MustCompile(`(?i)jellyfin[^0-9\n]{0,24}([0-9]+(?:\.[0-9]+)+)`)
	declaredGeneration = regexp.MustCompile(`(?i)\btarget[_-]?abi\b[^0-9\n]{0,8}([0-9]+(?:\.[0-9]+)+)`)
	// What a page cites a check by. The attribute is what a machine reads
	// and the name is also written where a reader sees it, both from one
	// value, so a page cannot show a reader one name and this row another.
	citedCheck = regexp.MustCompile(`(?is)\bdata-refused-by\s*=\s*"([^"]*)"`)
	// The properties a page moves something with, the compound names before
	// the shorthand so that one is not matched as the other with a suffix
	// left over. What precedes the name is part of the pattern for the
	// reason the colour scheme property carries one: a hyphen in front of it
	// is a different property.
	//
	// The set is a floor rather than a guarantee. Motion written through a
	// vendor-prefixed property, through an animation element inside an image
	// or through anything none of these names is refused by nothing here.
	motionProperty = regexp.MustCompile(`(?is)(^|[^-a-z])(transition-duration|transition-property|transition|animation-duration|animation-name|animation|scroll-behavior)\s*:\s*([^;{}"']*)`)
	// An at-rule and its condition, up to the brace that opens it. Where the
	// block ends is walked rather than matched, because nesting is counted
	// and a regexp cannot count.
	atRule = regexp.MustCompile(`(?is)@media\b([^{]*)\{`)
	// The question a condition asks about motion, and the answer it asks
	// for. The answer is optional because a condition may name the feature
	// with no value, which asks whether the reader wants less of it.
	motionPreference = regexp.MustCompile(`(?is)prefers-reduced-motion\s*(?::\s*([a-z-]+))?`)
	// The attribute that takes a control out of the focus order. It is
	// ordinarily written with no value at all, which the attribute pattern
	// does not match, and what precedes it is part of the pattern so that
	// the same word inside another attribute's value is not read as the
	// attribute itself.
	disabledAttribute = regexp.MustCompile(`(?is)(^|\s)disabled(\s|=|$)`)
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
// nothing else. The name is read from the package that puts an address on a page
// rather than written again here: that package needs it in order to state a
// page's own address, so a copy in this file would be a second definition of the
// one name this whole rule turns on, and the two would part company silently the
// day either moved.
//
// It is a list holding one entry rather than a constant, because the shape of
// the rule is a comparison against a set, and record 0011 takes the position
// that the set has one member rather than that the comparison is a special case.
var allowedOrigins = []string{
	site.Host,
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

// contentElement is where a page keeps the thing a reader came for, and it is
// what the link at the top of the frame has to reach. It is the landmark the
// standard already defines rather than a name this repository invented, so a
// reader's own software knows what it is without being told, and a link that
// reaches anything else is a link that skips less than it looks like it does.
const contentElement = "main"

// publishedLanguages is what this site publishes in, held as the primary subtag
// of a language tag. A page announcing anything else is a page in the wrong
// language, and the row below is where that is decided.
//
// It is a second copy of an answer taken elsewhere. The record is
// decisions/site-language.md in Flowfin/hub, which decided that the published
// pages are English and stated the same three-part obligation this row carries.
// Nothing here compares the two: no leg of the gate reaches the network, and the
// two verbs that do are outside it for that reason. So the day that record moves
// is a day this line is wrong and no run says so, and it is moved by hand with
// the record.
//
// A set rather than a single value, so an answer that later publishes a second
// language is a change to what this holds rather than to the row that reads it.
//
// The comparison is on the primary subtag, which is the record's own reading: a
// tag that is more specific about the same language is a page in that language,
// so en-GB passes. Refusing it would push whoever writes the next page towards
// the least specific tag available, which is the wrong direction for a rule that
// exists so a reader's software picks the right voice.
var publishedLanguages = []string{"en"}

// Rules is the table. The order is the order a run reports them in.
//
// It is handed the numbers a client is held to, because one row compares against
// values rather than against a shape, and there is no other way for a value to
// reach a row: a row is given the bytes of one file and nothing else. Carrying
// the numbers in this file instead would make it the second declaration of the
// file the row exists to keep as the only one, which is the failure with the
// name of the rule on it.
//
// What the table holds does not depend on what it is handed. Every row is here
// on every call, and only what one of them compares against moves, so a caller
// that wants the count rather than the decisions may hand it nothing.
func Rules(numbers []tokens.Number) []Rule {
	return []Rule{
		{
			ID:      "page-declares-its-language",
			Subject: ProducedPages,
			Reason:  "a page with no language is read aloud in whichever one the reader's software guessed, and the guess is wrong for exactly the readers who depend on it; a page declaring a language this site does not publish in stops the guessing and sends the same reader to the wrong answer with nothing on the page to signal it",
			Refuses: "a produced page with no html element, one whose lang attribute is missing or empty, or one whose lang names a language this site does not publish in",
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
			ID:      "page-declares-the-schemes-it-supports",
			Subject: ProducedPages,
			Reason:  "a browser picks the background it paints before any stylesheet arrives and the rendering of every form control from the same answer, so a page that says nothing is drawn light for a reader whose machine is set dark, and the served pages this generator replaces carry the declaration",
			Refuses: "a produced page that declares no colour scheme, or one whose declaration names nothing a browser reads",
			decide:  decideColourScheme,
		},
		{
			ID:      "page-motion-answers-the-reader",
			Subject: ProducedPages,
			Reason:  "a reader who has told their system to reduce motion has told every page, and the answer belongs to them rather than to whoever wrote the page; the pages this generator replaces carry the query already, so the generator is the thing that would drop it",
			Refuses: "a produced page carrying a declaration that moves something, where no query asking the reader about motion switches it off",
			Counted: "declarations that move something",
			decide:  decideMotion,
			counts:  countMotion,
		},
		{
			ID:      "page-carries-the-affiliation-notice",
			Subject: ProducedPages,
			Reason:  "this project uses another project's name on every page it produces and is not affiliated with it, and a notice a page can ship without is a notice that reaches the pages somebody remembered",
			Refuses: "a produced page that does not carry the affiliation notice",
			decide:  decideAffiliation,
		},
		{
			ID:      "page-reaches-the-content-first",
			Subject: ProducedPages,
			Reason:  "a keyboard reader starts above the text and reaches it through everything in front of it, which is one key press per link once a page carries a navigation, and the link that skips them is one element in the frame rather than something every page has to remember",
			Refuses: "a produced page whose first focusable element is not a link to that page's own content, including one pointing at an identifier no element on the page carries",
			decide:  decideContentFirst,
		},
		{
			ID:      "page-fetches-no-script",
			Subject: ProducedPages,
			Reason:  "the budget puts required scripting at zero bytes, and a script element with a source is a request this site did not have to make and a party that learns who is reading",
			Refuses: "a produced page carrying a script element with a src attribute, wherever it points",
			decide:  decideScriptSrc,
		},
		{
			ID:      "page-touches-no-browser-storage",
			Subject: ProducedPages,
			Reason:  "the privacy page states that no cookie is set and that nothing is written into either browser storage area, and a reader can check none of that from the outside, so the statement is worth the check that refuses a page breaking it rather than the paragraph making it",
			Refuses: "a produced page carrying a meta element that sets a cookie, or naming an interface that reads or writes a cookie, a browser storage area or a reporting beacon",
			decide:  decideBrowserStorage,
		},
		{
			ID:      "page-carries-no-inline-handler",
			Subject: ProducedPages,
			Reason:  "the row above about a script element reads the src attribute, so a page carrying no source and a handler written onto an element runs in a reader's browser and passes it, which is the shape a site with a zero-byte scripting budget stops noticing",
			Refuses: "a produced page carrying an event handler attribute on an element, or an address whose scheme is a script",
			decide:  decideInlineHandler,
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
			ID:      "page-carries-a-description",
			Subject: ProducedPages,
			Reason:  "most readers meet a page before they open it, as a line in a search result or a card in a chat window, and a page offering nothing there is shown its address instead, so a project whose site exists to explain what it is spends that first impression saying one word",
			Refuses: "a produced page with no description element, or one whose content is missing or holds only whitespace",
			decide:  decideDescription,
		},
		{
			ID:      "page-fits-the-markup-budget",
			Subject: ProducedPages,
			Reason:  "the budget is written as numbers a build can miss rather than as an intention, and a page of prose that cannot be written inside that much markup is carrying structure the reader is not being shown",
			Refuses: fmt.Sprintf("a produced page whose whole document is %d bytes or more, uncompressed", budget.HTMLBytes),
			decide:  decideMarkupBudget,
		},
		{
			ID:      "page-fits-the-stylesheet-budget",
			Subject: ProducedPages,
			Reason:  "inlining the stylesheet removes a round trip before anything renders, and the size is what keeps inlining cheaper than the request it replaced, so a page that outgrows it has given back what inlining bought",
			Refuses: fmt.Sprintf("a produced page carrying %d bytes or more of inlined stylesheet", budget.InlineCSSBytes),
			Counted: "inlined stylesheets",
			decide:  decideStylesheetBudget,
			counts:  countStylesheets,
		},
		{
			ID:      "page-downloads-no-web-font",
			Subject: ProducedPages,
			Reason:  "a downloaded face blocks or reflows the first text a reader sees, and the faces already on the reader's machine cost nothing and arrive first; the row about a foreign domain catches a face served from somebody else's host and says nothing about one this site would serve itself",
			Refuses: "a produced page declaring a font face that fetches a file, wherever it is served from",
			decide:  decideWebFont,
		},
		{
			ID:      "landing-page-asks-for-at-most-two-images",
			Subject: ProducedPages,
			Only:    site.IndexPath,
			Reason:  "the first page has to be complete after the fewest exchanges, and a limit counted in requests is the one a reader on a slow link actually feels; the record counts this page's requests and says nothing about the others, so this row reads this page and no other",
			Refuses: fmt.Sprintf("the produced landing page carrying more than %d image elements", budget.LandingImages),
			decide:  decideLandingImages,
		},
		{
			ID:      "page-links-the-legal-notice",
			Subject: ProducedPages,
			Reason:  "the page saying who publishes this site is worth what a reader can reach, and a link a page can ship without is a link that reaches the pages somebody remembered; it lives in the frame for the same reason the affiliation notice does",
			Refuses: "a produced page carrying no link to the address the legal notice is published at",
			decide:  decideLegalLink,
		},
		{
			ID:      "page-references-everything-from-the-site-root",
			Subject: ProducedPages,
			Reason:  "the not-found document is served in answer to a request of any depth, so a reference a browser resolves against the current document points somewhere different on every request and is broken on most of them; the same reference on a page inside a directory address is broken for every reader one level further in, and both look correct from the page they were written on",
			Refuses: "a produced page carrying a reference that a browser resolves against the document it sits on rather than against the site root, an empty one included",
			Counted: "references",
			decide:  decideRelativeReference,
			counts:  countReferences,
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
			Reason:  "a value typed into a template is a second definition of a value published somewhere else, and the day the published one moves the page goes on rendering the old one perfectly, so nobody sees it; the file carries lengths, weights and font stacks beside the colours, and a wrong length is the harder one to see because a wrong colour at least looks wrong",
			Refuses: "a colour, a length, a font family or a font weight written into what the build reads, outside a fragment reference",
			decide:  decideTypedTokenValue,
		},
		{
			ID:      "client-budget-numbers-live-in-exactly-one-file",
			Subject: BuildInputs,
			Reason:  "a number a client is held to is the same class of fact as a spacing step, and it is the harder one to see when it goes stale: a wrong colour looks wrong on the page and a wrong millisecond looks like every other millisecond, so a second copy of one is a conformance target somebody meets while the published one says something else",
			Refuses: "a limit a client is held to, written into what the build reads, in either the words the page states it in or the number and its unit alone",
			decide:  decideTypedBudgetNumber(numbers),
		},
		{
			ID:      "version-lives-in-exactly-one-file",
			Subject: TrackedTextOutsideTheVersionRegister,
			Reason:  "a version written a second time is right on the day it is typed, and the copy that goes stale is the one a reader takes the version from rather than the one the release run reads, so the tree announces a release nobody tagged and nothing about the tag says otherwise",
			Refuses: "the version this repository releases under, written into a tracked file outside the register that holds it",
			decide:  decideSecondVersion,
		},
		{
			ID:      "build-input-carries-no-server-generation",
			Subject: BuildInputs,
			Reason:  "which server generation a build is for is a fact about what was published rather than an opinion anybody records, and a generation typed here is right on the day it is typed; the release that moves to the next one leaves the page stating the old one, looking exactly as correct as it did before, and the reader it is wrong for is the one deciding what to install",
			Refuses: "a server generation written into what the build reads, in either the spelling a page states it in or the key a build manifest declares it under",
			decide:  decideTypedServerGeneration,
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
		{
			ID:      "source-carries-its-licence-header",
			Subject: SourceFiles,
			Reason:  "the licence at the root says what this repository is under and travels with nothing, so a file lifted out of here into another tree arrives carrying no terms and looks like something anybody may do anything with; the header is what makes the terms a property of the file rather than of the place it was found, and a file whose header names a different identifier is worse than one carrying none, because it is read rather than looked up",
			Refuses: "a tracked source file whose opening comment declares no licence identifier, and one declaring an identifier this repository does not publish under",
			decide:  decideLicenceHeader,
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
	// The file describing each release, where a heading per version is what
	// the file is for. It is a register rather than a copy: the release run
	// refuses a version this file does not carry, so the two are held to
	// each other by a refusal rather than drifting the way a second copy
	// usually does.
	changelog.File,
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

// decideTypedServerGeneration refuses a server generation written into what the
// build reads.
//
// The value belongs to the release that was published, and nothing else can say
// it: a source file says what somebody intends to publish and only a release
// says what somebody can install. So there is no register here for this row to
// point a repair at, and the message names what says it instead of naming a
// file that would have to be invented to receive the copy.
//
// It reads the words beside the number rather than the number, which is the
// opposite of the version row above and is the reason the two are separate rows.
// That one knows the exact string it is looking for and can compare whole runs
// against it. This one is looking for a value nobody in this tree holds, so the
// only thing that distinguishes a generation from a line height is what somebody
// wrote next to it.
//
// The population is what the build reads, which is where a wrong value reaches a
// page. It is also where a vendored roster lands, so the roster is a subject of
// this row on the day it arrives rather than on the day somebody remembers to
// widen the population.
func decideTypedServerGeneration(body []byte) []string {
	var details []string
	for i, line := range strings.Split(string(body), "\n") {
		for _, m := range statedGeneration.FindAllStringSubmatch(line, -1) {
			details = append(details, fmt.Sprintf(
				"line %d states the server generation %s, and only a published release says which generation a build is for",
				i+1, m[1]))
		}
		for _, m := range declaredGeneration.FindAllStringSubmatch(line, -1) {
			details = append(details, fmt.Sprintf(
				"line %d declares the server generation %s under the key a build manifest carries, and what a build manifest intends is not what a release published",
				i+1, m[1]))
		}
	}
	return details
}

// decideTypedTokenValue refuses a value the token file is the authority for,
// written into something the build reads.
//
// It reads a shape per group of values the file carries rather than the colours
// alone. The row's name has always been about tokens and what it decided was
// about colour, and the gap is easy to miss for the reason it exists: a colour
// typed next to the thing it describes is visible to anybody who opens the page
// afterwards with the published file beside it, and a type size, a corner radius
// or a font stack typed the same way is not. The page that demonstrates the
// design system is where all four get written next to what they describe, which
// is why the shapes are here before that page rather than after it.
//
// The bound is the shape of the literal rather than its meaning, which is the
// bound the version row declares for itself and for the same reason. A hex run
// is a colour, digits touching a unit are a length, and the two font
// declarations are named by their property because a stack and a three-digit
// weight are not recognisable on their own. A value spelled any other way walks
// through, so the file stays the authority for the set rather than this row.
//
// What it deliberately does not read is a fragment reference. An address ending
// in a name of hex length is a link to a place on the page and not a colour, and
// a row that could not tell the two apart would refuse the link the frame is
// built around.
func decideTypedTokenValue(body []byte) []string {
	var details []string
	for i, line := range strings.Split(string(body), "\n") {
		text := fragmentReference.ReplaceAllString(line, "")
		for _, m := range hexColour.FindAllString(text, -1) {
			details = append(details, fmt.Sprintf(
				"line %d writes the colour %s, and %s is the one file a colour is read from",
				i+1, m, tokens.File))
		}
		for _, m := range cssLength.FindAllString(text, -1) {
			details = append(details, fmt.Sprintf(
				"line %d writes the length %s, and %s is the one file a length is read from",
				i+1, m, tokens.File))
		}
		if fontFamilyProperty.MatchString(text) {
			details = append(details, fmt.Sprintf(
				"line %d declares a font family, and %s is the one file a font stack is read from",
				i+1, tokens.File))
		}
		if fontWeightProperty.MatchString(text) {
			details = append(details, fmt.Sprintf(
				"line %d declares a font weight, and %s is the one file a weight is read from",
				i+1, tokens.File))
		}
	}
	return details
}

// decideTypedBudgetNumber refuses one of the numbers a client is held to,
// written into something the build reads.
//
// It is the same rule as the one above with a different unit, and the unit is
// what makes it a separate row rather than a fifth shape in that one. A colour,
// a length, a font stack and a weight are recognisable from their own spelling,
// so that row can be told what to look for once and never revisited. A latency
// ceiling is digits and a word, indistinguishable from a transition duration or
// from a figure in a sentence, so the only thing that separates a copy of the
// budget from an unrelated number is the budget itself. That is why this one is
// handed the values and the others are not.
//
// Two spellings per number, and the longer one first so that a line stating the
// limit in the words the page uses is reported as that rather than as the bare
// number inside it. What it compares is exact: a limit written in another unit
// walks through, which is the same bound the row above declares for a colour
// spelled in a form CSS does not read. The file stays the authority for the set,
// and this row is what stops a second copy of one member of it.
//
// A run handed no numbers refuses nothing, which is why Run reads the copy
// before it builds the table and refuses a tree whose copy carries none. A green
// mark from this row over an empty set would say the tree holds no second copy
// of a number, when what it means is that nobody told it what the numbers are.
func decideTypedBudgetNumber(numbers []tokens.Number) func([]byte) []string {
	return func(body []byte) []string {
		var details []string
		for i, line := range strings.Split(string(body), "\n") {
			for _, n := range numbers {
				for _, spelling := range []string{n.Stated(), n.Bare()} {
					if !strings.Contains(line, spelling) {
						continue
					}
					details = append(details, fmt.Sprintf(
						"line %d writes %q, which is what %s says %s is, and %s is the one file it is read from",
						i+1, spelling, tokens.File, n.Name, tokens.File))
					break
				}
			}
		}
		return details
	}
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

// decideLicenceHeader refuses a source file that does not open under the
// identifier this repository publishes under.
//
// It passes a body that is not a source file rather than refusing it, and that
// is the difference between this row and every other row in the table. The
// others refuse the presence of something, so a body of the wrong kind carries
// none of it and passes without being asked. This one refuses an absence, and
// an absence is what a produced page, a copy of the design tokens and a
// document all have. The population the run hands it is already cut to source,
// so the reading here is the second of two rather than the only one; what it
// buys is that the row can be exercised beside the others against one body, and
// what it costs is a file of a language nobody has taught it to recognise.
// `internal/licence` carries that bound and the case that reports it.
func decideLicenceHeader(body []byte) []string {
	if !licence.IsSource(body) {
		return nil
	}
	declared, carried := licence.Declared(body)
	if !carried {
		return []string{fmt.Sprintf("the comment this file opens with declares no %s, and this repository publishes under %s",
			licence.Tag, licence.Identifier)}
	}
	if declared != licence.Identifier {
		return []string{fmt.Sprintf("the comment this file opens with declares %s %s, and this repository publishes under %s",
			licence.Tag, declared, licence.Identifier)}
	}
	return nil
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

// budgetNumbers reads the numbers a client is held to out of the pinned copy, so
// that the row about a second copy of one has something to compare against.
//
// It fails closed in both directions the row cannot survive, and each is refused
// in its own words rather than collapsed into one. A copy that could not be read
// is a tree the row was never decided against. A copy that carries no such
// number is a row that would report ok having compared nothing, which reads as a
// tree holding no second copy and means that nobody said what the numbers are.
// That is the same position gather takes about the copy existing at all, one
// step further in.
func budgetNumbers(root string) ([]tokens.Number, error) {
	values, err := tokens.Load(root)
	if err != nil {
		return nil, fmt.Errorf("the row about where a client budget number is read from cannot be decided: %w", err)
	}
	numbers, reasons := tokens.Numbers(values)
	if len(reasons) > 0 {
		return nil, fmt.Errorf("%s, %d reason(s):\n  %s", tokens.File, len(reasons), strings.Join(reasons, "\n  "))
	}
	if len(numbers) == 0 {
		return nil, fmt.Errorf("%s carries no client budget number, and the row about where one is read from is a row about there being exactly one file that carries it", tokens.File)
	}
	return numbers, nil
}

// Run decides every rule against the tree at root and writes what it examined
// to log. It builds the site into a directory it throws away, so what the page
// rules read is what a build produces rather than whatever is sitting in the
// output directory from an earlier one.
func Run(root string, log io.Writer) error {
	numbers, err := budgetNumbers(root)
	if err != nil {
		return err
	}

	rules := Rules(numbers)
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
		if r.Only != "" {
			files = onlyNamed(files, path.Join(site.OutputDir, r.Only))
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
		if len(files) == 0 && r.Only != "" {
			fmt.Fprintf(log, "  %s: the build wrote no %s, so this rule examined nothing\n",
				r.ID, path.Join(site.OutputDir, r.Only))
			continue
		}
		if len(files) == 0 {
			fmt.Fprintf(log, "  %s: %s held no file, so this rule examined nothing\n", r.ID, r.Subject)
			continue
		}
		if r.counts != nil {
			read := 0
			for _, f := range files {
				read += r.counts(f.body)
			}
			if read == 0 {
				fmt.Fprintf(log, "  %s: %d file(s) of %s carried no %s, so this rule decided nothing\n",
					r.ID, len(files), r.Subject, r.Counted)
				continue
			}
			fmt.Fprintf(log, "  %s: ok, %d %s in %d file(s) of %s\n",
				r.ID, read, r.Counted, len(files), r.Subject)
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

	// The rule about where a token value is read from is a rule about there
	// being exactly one such file, and a run that decided it against a tree
	// carrying none would report that no second definition was found in a
	// tree with no first one. So the copy being absent is refused here, by
	// name, rather than passing as a row with nothing to compare.
	if len(tracked.securitySources) == 0 {
		return nil, fmt.Errorf("%s is not tracked in this tree, so the build wrote no %s and the row about an expired reporting route has nothing to read", security.File, security.Path)
	}
	if len(tracked.tokenCopies) == 0 {
		return nil, fmt.Errorf("%s is not tracked in this tree, and the row about where a token value is read from is a row about there being exactly one such file", tokens.File)
	}
	// The claim about the clients is refused here for a different reason
	// than the two above, and it is the reason the file exists. A tree that
	// lost it still builds, and the landing page it produces is a page that
	// leads with what this project is and says nothing about the clients at
	// all. That is not a row deciding nothing; it is the page silently going
	// back to the shape the value was introduced to end, and no reading of
	// the produced bytes tells a sentence that was never composed from one
	// that was never asked for.
	if len(tracked.clientClaims) == 0 {
		return nil, fmt.Errorf("%s is not tracked in this tree, so the landing page says nothing about the clients and reads as a page about a project that has none", site.ClientsFile)
	}

	return map[string][]file{
		ProducedPages:                        pages,
		ProducedFiles:                        produced,
		TrackedText:                          tracked.text,
		TestSources:                          tracked.tests,
		SourceFiles:                          tracked.sources,
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
	sources         []file
	workflows       []file
	buildInputs     []file
	tokenCopies     []file
	securitySources []file
	clientClaims    []file
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
		if strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".sh") {
			found.sources = append(found.sources, f)
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
		if name == site.ClientsFile {
			found.clientClaims = append(found.clientClaims, f)
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
	tag := strings.TrimSpace(string(m[1]))
	if tag == "" {
		return []string{"the lang attribute on the html element is empty"}
	}
	if !published(tag) {
		return []string{fmt.Sprintf("the html element declares lang=%q, and this site publishes in %s",
			tag, strings.Join(publishedLanguages, ", "))}
	}
	return nil
}

// published says whether a language tag names one of the languages this site
// publishes in. It reads the primary subtag and folds case, because a tag is
// case-insensitive and a page written EN is the same page as one written en.
func published(tag string) bool {
	primary, _, _ := strings.Cut(tag, "-")
	for _, l := range publishedLanguages {
		if strings.EqualFold(primary, l) {
			return true
		}
	}
	return false
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
// The words a browser acts on. Anything else in the value is a forward
// compatible identifier the specification allows and no browser reads, so a
// value holding none of these three declares nothing whatever it says.
var readableSchemes = map[string]bool{"normal": true, "light": true, "dark": true}

func decideColourScheme(body []byte) []string {
	var details []string
	declared := false

	for _, loc := range colourSchemeMeta.FindAllIndex(body, -1) {
		declared = true
		value := ""
		if m := contentAttr.FindSubmatch(body[loc[0]:loc[1]]); m != nil {
			value = string(m[1])
		}
		details = append(details, schemeValue(value, lineOf(body, loc[0]))...)
	}
	for _, loc := range colourSchemeProperty.FindAllSubmatchIndex(body, -1) {
		declared = true
		details = append(details, schemeValue(string(body[loc[4]:loc[5]]), lineOf(body, loc[0]))...)
	}

	if !declared {
		return []string{"this page declares no colour scheme, and a browser picks the background it paints and the way it draws every control from that answer before a stylesheet has arrived"}
	}
	return details
}

// schemeValue judges one declaration. It reads the value rather than the name
// of the attribute, because the mistake a template actually makes is writing
// the declaration against something that was not there, and a rule looking for
// the word finds it while a browser reads nothing.
func schemeValue(value string, line int) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return []string{fmt.Sprintf(
			"line %d declares a colour scheme with no value on it, which is what a template writes from a value that was not there",
			line)}
	}
	for _, word := range strings.Fields(strings.ToLower(trimmed)) {
		if readableSchemes[word] {
			return nil
		}
	}
	return []string{fmt.Sprintf(
		"line %d declares the colour scheme %q, and a browser reads none of light, dark or normal out of it",
		line, trimmed)}
}

// decideMotion refuses a produced page that moves something without letting the
// reader stop it.
//
// Two shapes answer the reader and both are in what this project publishes
// today. A declaration inside a query asking for no preference exists only for
// a reader who has not asked for less, so the answer decides whether it applies
// at all. A declaration in the ordinary cascade is answered by a query asking
// for reduce that switches the same property off, which is the blanket override
// most stylesheets are written with; a row refusing that shape is a row somebody
// takes out the first time they write one.
//
// What it cannot do is resolve a selector. A reduce query switching a property
// off covers every declaration of that property as far as this row can see, so
// a page that asks the question and misses one rule passes here and the headless
// legs are where that is caught. What it refuses is the page that never asks,
// which is the ordinary failure and the one a generator produces by rendering a
// stylesheet somebody wrote without the query.
//
// The third case is the one worth the row on its own: motion written inside the
// reduce query, which moves something for the only reader who said they did not
// want it, and which reads at a glance like the repair.
func decideMotion(body []byte) []string {
	queries := motionQueries(body)
	declarations := motionDeclarations(body)

	// What a blanket override switched off, read before anything is judged,
	// because the override is written after the declarations it answers for
	// as often as before them.
	stopped := map[string]bool{}
	for _, d := range declarations {
		if inside, reduce := enclosing(queries, d.at); inside && reduce && d.stopped {
			stopped[d.family] = true
		}
	}

	var details []string
	for _, d := range declarations {
		inside, reduce := enclosing(queries, d.at)
		switch {
		case d.stopped:
			// A value that moves nothing is the repair wherever it
			// is written, and a row reading the property name alone
			// would refuse every repair.
		case inside && !reduce:
			// Motion that exists only for a reader who has not
			// asked for less of it.
		case inside && reduce:
			details = append(details, fmt.Sprintf(
				"line %d declares %s: %s inside the query asking for reduced motion, so what a reader gets for asking is the motion",
				d.line, d.property, d.value))
		case !stopped[d.family]:
			details = append(details, fmt.Sprintf(
				"line %d declares %s: %s, and no query asking the reader about motion switches %s off",
				d.line, d.property, d.value, d.family))
		}
	}
	return details
}

// countMotion is how many declarations that move something the page carried, so
// a run over pages that move nothing says that rather than ok. A value that
// stops motion is not one of them: a page whose only such declaration is the
// repair moves nothing, and counting it would have the row report that it
// decided a page it never had to judge.
func countMotion(body []byte) int {
	moving := 0
	for _, d := range motionDeclarations(body) {
		if !d.stopped {
			moving++
		}
	}
	return moving
}

// declaration is one motion-bearing property as it was written on a page.
type declaration struct {
	property string
	family   string
	value    string
	at       int
	line     int
	stopped  bool
}

func motionDeclarations(body []byte) []declaration {
	var out []declaration
	for _, m := range motionProperty.FindAllSubmatchIndex(body, -1) {
		property := strings.ToLower(string(body[m[4]:m[5]]))
		value := strings.TrimSpace(string(body[m[6]:m[7]]))
		out = append(out, declaration{
			property: property,
			family:   motionFamily(property),
			value:    value,
			at:       m[4],
			line:     lineOf(body, m[4]),
			stopped:  motionStopped(value),
		})
	}
	return out
}

// motionFamily is the property a blanket override is written against. A reduce
// query saying transition: none and one saying transition-duration: 0s stop the
// same thing, so both answer for every transition on the page.
func motionFamily(property string) string {
	switch {
	case strings.HasPrefix(property, "transition"):
		return "transition"
	case strings.HasPrefix(property, "animation"):
		return "animation"
	default:
		return property
	}
}

// motionStopped answers whether a value moves nothing. Every word of it has to
// be one of the stopping words: a value naming a property to transition and a
// duration to do it over moves something however short the duration is, and
// reading only the first word would pass it.
func motionStopped(value string) bool {
	v := strings.ToLower(value)
	// The priority marker is not part of the value, and the override that
	// has to win over an author's own rule is the place it is written.
	if i := strings.Index(v, "!"); i >= 0 {
		v = v[:i]
	}
	fields := strings.Fields(v)
	if len(fields) == 0 {
		// A declaration written against a value that was not there.
		// Nothing moves, and the row that refuses a template writing an
		// empty value is the one about the scheme rather than this one.
		return true
	}
	for _, f := range fields {
		switch strings.TrimSuffix(f, ",") {
		case "none", "auto", "0", "0s", "0ms", "initial", "unset", "revert":
		default:
			return false
		}
	}
	return true
}

// query is an at-rule that asks the reader about motion, and what it asks for.
type query struct {
	from   int
	to     int
	reduce bool
}

// motionQueries is every such at-rule on the page, with the extent of its block.
//
// The extent is walked brace by brace. That reads braces and nothing else, so a
// brace inside a comment or inside a string in the block would move where the
// block is thought to end; both are things a stylesheet can hold and neither is
// something this build writes today.
func motionQueries(body []byte) []query {
	var out []query
	for _, m := range atRule.FindAllSubmatchIndex(body, -1) {
		condition := body[m[2]:m[3]]
		asked := motionPreference.FindSubmatch(condition)
		if asked == nil {
			continue
		}
		// A condition naming the feature with no value asks whether the
		// reader wants less motion, so it is the reduce side of the
		// question rather than a third state.
		answer := strings.ToLower(string(asked[1]))
		open := m[1] - 1
		out = append(out, query{
			from:   open,
			to:     blockEnd(body, open),
			reduce: answer != "no-preference",
		})
	}
	return out
}

// blockEnd walks from an opening brace to the brace that closes it, or to the
// end of the page where nothing does.
func blockEnd(body []byte, open int) int {
	depth := 0
	for i := open; i < len(body); i++ {
		switch body[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return len(body)
}

func enclosing(queries []query, at int) (inside, reduce bool) {
	for _, q := range queries {
		if at > q.from && at < q.to {
			return true, q.reduce
		}
	}
	return false, false
}

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

// decideContentFirst refuses a produced page that does not put a link to its own
// content in front of everything else a keyboard reader can reach.
//
// The failure is invisible to everybody who reads with a pointer, and it gets
// worse as the site grows: every link the frame puts above the text is one more
// key press between a keyboard reader and the thing they came for. So the row is
// about position rather than about the link existing, and it reads the page in
// document order and judges the first element a tab key stops on.
//
// The address is judged as well as the position, because a link that skips
// nothing is the shape this actually rots into. A fragment naming an element the
// page does not carry moves nobody, looks correct in the markup, and is what a
// frame produces the day the identifier on the content is renamed.
func decideContentFirst(body []byte) []string {
	elements := walk(body)

	// Every identifier on the page and what carries it, read before the
	// order is judged, because the link is written above the element it
	// reaches. The first of a repeated identifier is the one kept, which is
	// the same one a browser reaches, and the row about an identifier used
	// twice is where that page is refused.
	carries := map[string]string{}
	for _, e := range elements {
		if id, ok := e.attrs["id"]; ok {
			if _, seen := carries[id]; !seen {
				carries[id] = e.name
			}
		}
	}

	for _, e := range elements {
		if !focusable(e) {
			continue
		}
		if e.name != "a" {
			return []string{fmt.Sprintf(
				"line %d: the first thing a keyboard reader reaches is a %s element rather than a link to the content",
				e.line, e.name)}
		}
		href := strings.TrimSpace(e.attrs["href"])
		switch {
		case href == "":
			return []string{fmt.Sprintf(
				"line %d: the first thing a keyboard reader reaches is a link with no address on it, which is what a frame writes from a value that was not there",
				e.line)}
		case !strings.HasPrefix(href, "#"):
			return []string{fmt.Sprintf(
				"line %d: the first thing a keyboard reader reaches is a link to %s, which leaves the content behind everything else this page carries",
				e.line, href)}
		}
		named := strings.TrimPrefix(href, "#")
		switch carrier, answered := carries[named]; {
		case !answered:
			return []string{fmt.Sprintf(
				"line %d: the link jumps to %s and no element on this page answers to that identifier, so the key press moves nobody",
				e.line, href)}
		case carrier != contentElement:
			return []string{fmt.Sprintf(
				"line %d: the link jumps to %s, which is a %s element rather than the %s element this page keeps its content in",
				e.line, href, carrier, contentElement)}
		}
		return nil
	}

	return []string{"this page offers a keyboard reader nothing to focus at all, so it carries no link to its content, and that link is written in the frame every page is rendered through rather than on the page"}
}

// element is one start tag as the walk met it. What a browser stops on is
// decided from the name and the attributes together, so both are carried, and
// the raw attributes come with them because an attribute written with no value
// at all is a shape the attribute pattern does not match.
type element struct {
	name  string
	attrs map[string]string
	raw   string
	line  int
}

// walk reads every start tag on the page in the order a reader meets them.
//
// It reads bytes rather than a parsed document, which is the floor every row
// here reads at: a tag written inside a stylesheet or a comment is counted as an
// element. What stands behind the assumption that the markup is well formed is
// the row about the page parsing, which refuses the page this one would misread.
func walk(body []byte) []element {
	var out []element
	for _, tag := range htmlTag.FindAllSubmatchIndex(body, -1) {
		attrs := body[tag[4]:tag[5]]
		e := element{
			name:  strings.ToLower(string(body[tag[2]:tag[3]])),
			attrs: map[string]string{},
			raw:   string(attrs),
			line:  lineOf(body, tag[0]),
		}
		for _, a := range tagAttribute.FindAllSubmatchIndex(attrs, -1) {
			name := strings.ToLower(string(attrs[a[2]:a[3]]))
			if _, seen := e.attrs[name]; !seen {
				e.attrs[name] = attributeValue(attrs, a)
			}
		}
		out = append(out, e)
	}
	return out
}

// focusable answers whether a tab key stops on this element.
//
// It is a floor and it is stated as one. It holds the elements a browser puts in
// the order by default and the attribute that overrides that answer in either
// direction, which is the whole of what this build can produce today. An element
// made focusable through something none of this reads is focusable to a reader
// and invisible here, and the headless leg is where that is seen.
func focusable(e element) bool {
	if v, ok := e.attrs["tabindex"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n >= 0
		}
		// A value a browser cannot read is not an answer, so the element
		// keeps whatever it would have been without the attribute.
	}
	switch e.name {
	case "a", "area":
		_, ok := e.attrs["href"]
		return ok
	case "button", "select", "textarea":
		return !disabledAttribute.MatchString(e.raw)
	case "input":
		return !disabledAttribute.MatchString(e.raw) &&
			!strings.EqualFold(strings.TrimSpace(e.attrs["type"]), "hidden")
	}
	return false
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

// What the budget rows read. A style element is the only thing on these pages
// that carries a stylesheet, and a font face is the declaration that fetches one.
var (
	styleElement = regexp.MustCompile(`(?is)<style\b[^>]*>(.*?)</style>`)
	fontFace     = regexp.MustCompile(`(?is)@font-face\b[^{]*\{`)
)

// decideMarkupBudget refuses a produced page larger than the record allows.
//
// The measured number and the limit are both in the refusal, because a budget
// failure that says only that the build failed makes the next person measure by
// hand, and the number they would reach for is the one this already has.
func decideMarkupBudget(body []byte) []string {
	if len(body) < budget.HTMLBytes {
		return nil
	}
	return []string{fmt.Sprintf(
		"this page is %d bytes and the budget in %s puts a document under %d",
		len(body), budget.Record, budget.HTMLBytes)}
}

// decideStylesheetBudget refuses a produced page carrying more inlined stylesheet
// than the record allows. Every style element on the page is counted together,
// because what the reader waits for is the document and a second element is not a
// second budget.
func decideStylesheetBudget(body []byte) []string {
	total := 0
	for _, m := range styleElement.FindAllSubmatchIndex(body, -1) {
		total += m[3] - m[2]
	}
	if total < budget.InlineCSSBytes {
		return nil
	}
	return []string{fmt.Sprintf(
		"this page inlines %d bytes of stylesheet and the budget in %s puts it under %d",
		total, budget.Record, budget.InlineCSSBytes)}
}

// countStylesheets says how many style elements the page carries, so a page with
// none reports that rather than reporting ok against a limit it never approached.
func countStylesheets(body []byte) int {
	return len(styleElement.FindAllIndex(body, -1))
}

// decideWebFont refuses a produced page declaring a face that fetches a file.
//
// It reads the declaration rather than the address, which is what makes it
// different from the row about a foreign domain: a face this site served itself
// would satisfy that row and cost the reader exactly what the budget puts at zero.
func decideWebFont(body []byte) []string {
	var details []string
	for _, m := range fontFace.FindAllIndex(body, -1) {
		details = append(details, fmt.Sprintf(
			"line %d declares a font face, and the budget in %s puts web font downloads at %d",
			lineOf(body, m[0]), budget.Record, budget.WebFontDownloads))
	}
	return details
}

// decideLandingImages refuses the landing page for asking for more images than the
// record allows. The document itself is the one request the record counts beside
// them, and a page is one document by construction.
func decideLandingImages(body []byte) []string {
	found := imgElement.FindAllIndex(body, -1)
	if len(found) <= budget.LandingImages {
		return nil
	}
	return []string{fmt.Sprintf(
		"this page asks for %d image(s) after its document and the budget in %s allows %d",
		len(found), budget.Record, budget.LandingImages)}
}

// onlyNamed keeps the one file a narrowed row is about. A row naming a path the
// build did not write is left with nothing, which the run reports rather than
// passing over.
func onlyNamed(files []file, name string) []file {
	for _, f := range files {
		if f.name == name {
			return []file{f}
		}
	}
	return nil
}

// decideLegalLink refuses a produced page that offers no way to the page saying
// who publishes this site.
//
// The address is read from the package that writes that page rather than typed
// here, so a page that moves does not leave this row refusing every page for
// missing an address nothing answers at. What it looks for is a link a reader
// follows: an element that fetches the address would satisfy the letter of this
// and reach nobody.
func decideLegalLink(body []byte) []string {
	want := site.LegalAddress
	for _, tag := range htmlTag.FindAllSubmatchIndex(body, -1) {
		if !readerFollows[strings.ToLower(string(body[tag[2]:tag[3]]))] {
			continue
		}
		attrs := body[tag[4]:tag[5]]
		for _, a := range tagAttribute.FindAllSubmatchIndex(attrs, -1) {
			if strings.ToLower(string(attrs[a[2]:a[3]])) != "href" {
				continue
			}
			if strings.TrimSpace(attributeValue(attrs, a)) == want {
				return nil
			}
		}
	}
	return []string{fmt.Sprintf(
		"this page carries no link a reader can follow to %s, so whoever publishes this site is reachable from the pages that happened to keep the link", want)}
}

// descriptionMeta is the element a search result and a shared card read the
// sentence under the title out of.
var descriptionMeta = regexp.MustCompile(`(?is)<meta\b[^>]*\bname\s*=\s*"description"[^>]*>`)

// decideDescription refuses a produced page that says nothing about itself.
//
// The absence is invisible from the page, which is why it is a row rather than
// something a reader of the page would catch: the document renders identically,
// and what changes is a line in a search result and a card in a chat window,
// neither of which anybody looks at while writing the page.
//
// An element that is present and empty is refused with an absent one. It is the
// shape a description rendered from a value that did not arrive leaves behind,
// and it is the one a reader of the markup is most likely to read as done.
func decideDescription(body []byte) []string {
	m := descriptionMeta.Find(body)
	if m == nil {
		return []string{"there is no description element on this page, so a search result for it carries its address instead of a sentence"}
	}
	c := contentAttr.FindSubmatch(m)
	if c == nil {
		return []string{"the description element carries no content attribute, so there is nothing for a result to show"}
	}
	if strings.TrimSpace(string(c[1])) == "" {
		return []string{"the description element holds only whitespace, which reads in the markup as a description and is not one"}
	}
	return nil
}

// anchored is a reference a browser resolves against something other than the
// document it was written in: one that names a scheme or an authority, one that
// is absolute from the site root, and one that is a fragment of the current page
// and so carries no path to resolve at all.
var anchored = regexp.MustCompile(`^(?:[a-z][a-z0-9+.-]*:|//|/|#)`)

// decideRelativeReference refuses a produced page carrying a reference that a
// browser resolves against the document it sits on.
//
// It reads href wherever it appears rather than only where the page fetches it,
// which is the one place this row is wider than the origin row above and is the
// whole point of it: the reference this rule exists for is the link out of the
// not-found page, which a reader follows rather than something the page fetches.
//
// An empty value is refused with the rest. It resolves to whatever URL the
// request asked for, which on the not-found page is the address that was not
// there, so it is the one relative reference that reads as deliberate.
func decideRelativeReference(body []byte) []string {
	var details []string
	report := func(at int, attribute, ref string) {
		details = append(details, fmt.Sprintf(
			"line %d %s=%q is resolved against the document it sits on, and every reference in a produced page is absolute from the site root",
			lineOf(body, at), attribute, ref))
	}

	for _, tag := range htmlTag.FindAllSubmatchIndex(body, -1) {
		attrs := body[tag[4]:tag[5]]
		for _, a := range tagAttribute.FindAllSubmatchIndex(attrs, -1) {
			name := strings.ToLower(string(attrs[a[2]:a[3]]))
			if name != "href" && !subresourceAttributes[name] {
				continue
			}
			for _, ref := range references(name, attributeValue(attrs, a)) {
				if !anchored.MatchString(strings.ToLower(strings.TrimSpace(ref))) {
					report(tag[4]+a[0], name, ref)
				}
			}
		}
	}

	for _, expression := range []*regexp.Regexp{cssURL, cssImport} {
		for _, m := range expression.FindAllSubmatchIndex(body, -1) {
			ref := string(body[m[2]:m[3]])
			if !anchored.MatchString(strings.ToLower(strings.TrimSpace(ref))) {
				report(m[0], "url", ref)
			}
		}
	}

	return details
}

// countReferences says how many references a page carries, so that a page
// holding none reports that rather than reporting ok. A green mark over a page
// with nothing to resolve reads as a page whose references were checked.
func countReferences(body []byte) int {
	n := 0
	for _, tag := range htmlTag.FindAllSubmatchIndex(body, -1) {
		attrs := body[tag[4]:tag[5]]
		for _, a := range tagAttribute.FindAllSubmatchIndex(attrs, -1) {
			name := strings.ToLower(string(attrs[a[2]:a[3]]))
			if name != "href" && !subresourceAttributes[name] {
				continue
			}
			n += len(references(name, attributeValue(attrs, a)))
		}
	}
	n += len(cssURL.FindAllIndex(body, -1))
	n += len(cssImport.FindAllIndex(body, -1))
	return n
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
//
// The table is asked for its names rather than for its decisions, so it is
// handed nothing. What a row compares against does not change whether it is in
// the table, which is the sentence Rules carries and the suite checks.
func decided(name string) bool {
	for _, r := range Rules(nil) {
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

// What the two rows about a reader's browser read.
//
// The interface names are the vocabulary a page has to spell in order to reach
// a cookie, either storage area or a reporting call. They are matched over the
// produced bytes rather than inside a script element, because there is no
// script element to look inside: what a page here could carry is a handler on
// an element or a style declaration, and a name appearing anywhere in the
// document is the thing being refused either way.
//
// The bound is that this reads names and not behaviour. A page reaching the
// same interface through a value none of these spellings finds is refused by
// nothing here, and the headless leg is where that is seen. It is also why a
// produced page that merely mentions one of these words in a sentence is
// refused: separating a name from a mention needs a reading of the document
// that these rows do not make, and refusing the mention is the direction that
// fails closed.
var (
	setCookieMeta = regexp.MustCompile(`(?is)<meta\b[^>]*\bhttp-equiv\s*=\s*["']?\s*set-cookie`)
	storageName   = regexp.MustCompile(`(?i)\b(document\.cookie|localStorage|sessionStorage|indexedDB|openDatabase|navigator\.sendBeacon|navigator\.cookieEnabled)\b`)
	// An address whose scheme runs code rather than fetching anything. The
	// space is allowed because a browser reads one and a pattern that did
	// not would miss the spelling somebody actually pastes.
	scriptScheme = regexp.MustCompile(`(?is)\b(?:href|src|action|formaction)\s*=\s*["']?\s*javascript\s*:`)
	// An event handler attribute. The name is the whole of what a browser
	// needs to run it, so the pattern is the name rather than anything about
	// the value.
	handlerAttribute = regexp.MustCompile(`^on[a-z]+$`)
)

// decideBrowserStorage refuses a produced page that reaches for a cookie, a
// browser storage area or a reporting call.
//
// A meta element setting a cookie is separated from the interface names because
// the two fail differently and the repair is not the same. The element is a
// header this site chose to write into the document, which a host serves
// verbatim and a reader's browser acts on with no script involved at all; the
// names are code that would have to run. A refusal naming only one of them
// would send the next person looking in the wrong half of the page.
func decideBrowserStorage(body []byte) []string {
	var details []string
	for _, loc := range setCookieMeta.FindAllIndex(body, -1) {
		details = append(details, fmt.Sprintf(
			"line %d carries a meta element that sets a cookie, which a browser acts on with nothing running on the page",
			lineOf(body, loc[0])))
	}
	for _, loc := range storageName.FindAllIndex(body, -1) {
		details = append(details, fmt.Sprintf(
			"line %d names %s, which reads or writes something this site says it leaves alone",
			lineOf(body, loc[0]), string(body[loc[0]:loc[1]])))
	}
	return details
}

// decideInlineHandler refuses a produced page carrying code written onto an
// element or into an address.
//
// This is the near miss the row above it does not catch. The scripting budget
// is zero bytes and the row that reads a script element reads its src
// attribute, so a page with no source anywhere and one handler on one element
// is a page that runs code in a reader's browser and passes every row this gate
// had. It is also the mistake somebody actually makes: a template gains a
// button, the button gains an onclick, and nothing about either looks like
// fetching a script.
func decideInlineHandler(body []byte) []string {
	var details []string
	for _, e := range walk(body) {
		for name := range e.attrs {
			if handlerAttribute.MatchString(name) {
				details = append(details, fmt.Sprintf(
					"line %d: the %s element carries the handler attribute %s, which runs in a reader's browser",
					e.line, e.name, name))
			}
		}
	}
	for _, loc := range scriptScheme.FindAllIndex(body, -1) {
		details = append(details, fmt.Sprintf(
			"line %d carries an address whose scheme is a script rather than something to fetch",
			lineOf(body, loc[0])))
	}
	return details
}
