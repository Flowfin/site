// What every page says about itself before a reader opens it, and the one place
// the site's own name is written.
//
// Most readers meet a page as a line in a search result or a card in a chat
// window, and what they see there is whatever the head of the document offered.
// A page with no description is shown its address instead, and a project whose
// site exists to explain what it is spends that first impression saying one
// word.
//
// Three things are decided here and none of them is written into the template. A
// description belongs to the page, so it comes out of the same file the prose
// does; one written into the frame would be one description for every page,
// which is worse than none because it makes every result look like every other.
// A canonical address is decisions/0008-the-url-shape.md stated inside the
// document, so a page reachable at a second address does not compete with
// itself. And the card a shared link renders carries the title, the description
// and the site name and references nothing, because the project publishes no
// image and a card image would be a file per page against a budget that counts
// every byte.
//
// Nothing here reaches another origin. A card is metadata this site serves about
// itself, and the row that refuses a reference to a domain this project does not
// control reads the same pages afterwards.
package site

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Host is the name this site is published under and Origin is that name with the
// scheme a reader arrives over. They are here rather than in the package that
// refuses a reference to anywhere else, because this is the package that puts an
// address on a page, and a second copy of a host name is the shape that goes
// quietly stale.
//
// The name is the one decisions/0008-the-url-shape.md gives the site root rather
// than whichever spelling happens to answer on the day, so a change of name
// arrives as an argument somebody makes rather than as an address nobody
// noticed.
const (
	Host   = "flowfin.dev"
	Origin = "https://" + Host
)

// SiteName is what a card calls this site, and it is the project's name rather
// than the name of a page on it. A reader who sees three cards from here should
// be able to tell they came from one place.
const SiteName = "Flowfin"

// descriptionKeyword is what the block carrying a page's description opens with.
// It is a keyword rather than the first paragraph, because a paragraph is
// written for somebody who has already arrived and a description is written for
// somebody deciding whether to.
const descriptionKeyword = "description:"

// indexDocument is the file name a static host serves for a directory address
// without being told to, and it is why a page at a directory address is written
// into a file rather than at the address itself.
const indexDocument = "index.html"

// keywordLine is what a block's first line looks like before the keyword is
// read. It exists so that a misspelled keyword is refused rather than read as
// prose: a block that opens like a statement and is not one is the mistake that
// silently empties a register, and the rendered page looks finished either way.
var keywordLine = regexp.MustCompile(`^[a-z]+:\s`)

// blocks reads a content file into its blocks, one per run of non-empty lines,
// each joined into one string. The files are prose and prose is wrapped, so a
// reader that took a line at a time would read the last line of a wrapped block
// as a block of its own.
func blocks(name string) ([][]string, error) {
	f, err := os.Open(filepath.Clean(name))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out [][]string
	var block []string
	flush := func() {
		if len(block) > 0 {
			out = append(out, block)
			block = nil
		}
	}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			flush()
			continue
		}
		block = append(block, line)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	flush()
	return out, nil
}

// describe takes the keyword off a description block and returns the sentence,
// or the reason the block was refused.
func describe(joined string) (text, reason string) {
	text = strings.TrimSpace(strings.TrimPrefix(joined, descriptionKeyword))
	if text == "" {
		return "", fmt.Sprintf(
			"a %s block carries no sentence, and a page whose description is empty is shown its address in a search result instead",
			strings.TrimSuffix(descriptionKeyword, ":"))
	}
	return text, ""
}

// missingDescription is the reason a file with no description block is refused.
// It is one sentence in one place because all three readers give it.
func missingDescription() string {
	return fmt.Sprintf(
		"the file carries no %s block, and a page without one is met as its address rather than as what it says, which is invisible from the page itself",
		descriptionKeyword)
}

// locate puts the page's own address and the site's name on it, from the path
// the build is about to write it to. It is called by whatever writes a page, so
// a page added later cannot be written without them: the field is on the struct
// the template reads, and the row over the produced pages refuses a page whose
// head is missing.
func (p *page) locate(produced string) {
	p.Canonical = Origin + addressOf(produced)
	p.SiteName = SiteName
}

// addressOf is the address decisions/0008-the-url-shape.md gives a page, read off
// the path the build writes it to. A directory address is served by the index
// document inside it, so a page written to that name is served at the directory
// and states the directory as its address rather than the file.
//
// The not-found document is the one page this returns an address for that a
// reader mostly does not arrive at. That is the reason it states one at all: it
// is served in answer to addresses that are not its own, so the address it
// carries is the only one that identifies it.
func addressOf(produced string) string {
	produced = path.Clean(filepath.ToSlash(produced))
	if path.Base(produced) != indexDocument {
		return "/" + produced
	}
	dir := path.Dir(produced)
	if dir == "." {
		return "/"
	}
	return "/" + dir + "/"
}

// descriptions is what the build has already put on a page, keyed by the
// sentence rather than by the page, because what it exists to refuse is one
// sentence on two pages.
//
// It is held by the build rather than by a row over the produced pages, and the
// reason is that no row sees two pages at once: a rule there is given one page's
// bytes and asked about them. The build is the only place that holds every page,
// so it is where the question can be asked at all. The invariant verb builds
// before it reads, so a duplicate reds that verb as well as this one.
type descriptions map[string]string

// add records a page's description and refuses a sentence a page already
// carries. Two pages with one description is worse than two with none: it makes
// a search result for either of them look like a result for the other, and
// nothing about either page looks wrong.
func (d descriptions) add(produced, text string) error {
	if first, ok := d[text]; ok {
		return fmt.Errorf(
			"%s and %s carry the same description, and one sentence over two pages makes a result for either read as a result for the other: %q",
			addressOf(first), addressOf(produced), text)
	}
	d[text] = produced
	return nil
}
