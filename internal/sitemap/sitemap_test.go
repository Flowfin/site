// The suite over the comparison between a sitemap and the output beside it.
//
// The two directions are driven through the decision rather than through a
// build, because a build writes the file from the same rule the comparison
// reads, so a case that went through one could never produce the disagreement
// this leg exists to refuse. What the cases below hand it is the two lists.
//
// No case opens a window, binds a socket, reaches the network or needs anything
// that is not in the toolchain.
package sitemap

import (
	"io"
	"strings"
	"testing"

	"github.com/Flowfin/site/internal/site"
)

const (
	root    = site.Origin + "/"
	privacy = site.Origin + "/privacy/"
	legal   = site.Origin + "/legal/"
)

// The near miss. A page lands in the output and the list does not carry it,
// which is what a writer added after the sitemap is assembled produces. Every
// other check stays green: the page is valid, it is served, and nothing tells a
// crawler it is there.
func TestAProducedPageThatIsListedNowhereIsRefused(t *testing.T) {
	details := Decide([]string{root, privacy}, []string{root, privacy, legal})

	if len(details) != 1 {
		t.Fatalf("Decide gave %d detail(s), want the one missing page: %v", len(details), details)
	}
	if !strings.Contains(details[0], legal) {
		t.Errorf("the failure does not name the page that is listed nowhere: %s", details[0])
	}
}

// The same two lists with the entry restored. Without this the case above
// proves that something reds rather than that this reds for its own reason.
func TestTheSameListsAgreeingArePassed(t *testing.T) {
	details := Decide([]string{root, privacy, legal}, []string{root, privacy, legal})

	if len(details) != 0 {
		t.Fatalf("Decide refused a list that matches the output: %v", details)
	}
}

// The other direction. An address with no page behind it sends every client
// that reads the file to the not-found page, and the site is the party claiming
// the address exists.
func TestAnEntryWithNoPageBehindItIsRefused(t *testing.T) {
	details := Decide([]string{root, privacy, legal}, []string{root, privacy})

	if len(details) != 1 {
		t.Fatalf("Decide gave %d detail(s), want the one entry with nothing behind it: %v", len(details), details)
	}
	if !strings.Contains(details[0], legal) {
		t.Errorf("the failure does not name the entry: %s", details[0])
	}
}

// One page listed twice is one page fetched twice by everything that reads the
// file, and both entries have a page behind them, so neither direction above
// sees it.
func TestAPageListedTwiceIsRefused(t *testing.T) {
	details := Decide([]string{root, privacy, privacy}, []string{root, privacy})

	if len(details) != 1 {
		t.Fatalf("Decide gave %d detail(s), want the one duplicate: %v", len(details), details)
	}
	if !strings.Contains(details[0], privacy) {
		t.Errorf("the failure does not name the duplicated entry: %s", details[0])
	}
}

// Both directions at once are both reported. A run that stopped at the first
// disagreement would cost a run per repair.
func TestBothDirectionsAreReportedTogether(t *testing.T) {
	details := Decide([]string{root, legal}, []string{root, privacy})

	if len(details) != 2 {
		t.Fatalf("Decide gave %d detail(s), want one per direction: %v", len(details), details)
	}
}

// What the leg reads out of the file is the address element and nothing else.
func TestListedReadsTheAddresses(t *testing.T) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>` + root + `</loc></url>
  <url><loc>` + privacy + `</loc></url>
</urlset>
`)

	got := Listed(body)

	if len(got) != 2 || got[0] != root || got[1] != privacy {
		t.Fatalf("Listed gave %v, want the two addresses in the order the file states them", got)
	}
}

// The tree this file sits in. It is the one case here that judges the real
// output, and what it answers is whether this repository still agrees with
// itself, which is a different question from whether the rule bites.
func TestTheTreeAgreesWithItsOwnSitemap(t *testing.T) {
	var log strings.Builder
	if err := Run("../..", &log); err != nil {
		t.Fatalf("%v\n%s", err, log.String())
	}
	if !strings.Contains(log.String(), "address(es) listed") {
		t.Errorf("the run does not say what it compared:\n%s", log.String())
	}
}

// A run says what it covered whether or not it found anything, so a leg that
// compared an empty pair cannot be read as one that compared the site.
func TestRunSaysWhatItCompared(t *testing.T) {
	if err := Run("../..", io.Discard); err != nil {
		t.Fatalf("Run over this tree: %v", err)
	}
}
