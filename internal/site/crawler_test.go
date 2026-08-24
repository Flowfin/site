// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the two files a crawler asks for.
//
// The cases about which addresses belong in a sitemap drive the rule directly
// rather than through a build, because the rule is what the leg over the output
// reads and a case that went through a build would be testing the build.
package site

import (
	"io"
	"path/filepath"
	"strings"
	"testing"
)

// The not-found page is produced, is a page, and is the one page a sitemap must
// not carry. Listing it asks a crawler to index the document a host serves for
// addresses this site does not have, under an address the site says it has.
func TestSitemapLeavesOutTheNotFoundPage(t *testing.T) {
	produced := []string{"index.html", "privacy/index.html", NotFoundPath}

	got := SitemapAddresses(produced)

	for _, a := range got {
		if strings.HasSuffix(a, "/"+NotFoundPath) {
			t.Fatalf("the sitemap lists the not-found page: %v", got)
		}
	}
	if len(got) != 2 {
		t.Fatalf("SitemapAddresses(%v) = %v, want the two pages that are not the not-found one", produced, got)
	}
}

// A directory address is served by the index document inside it, and the
// sitemap states the address rather than the file. A crawler told to fetch the
// file would index a second address for a page that already has one.
func TestSitemapStatesTheAddressAndNotTheFile(t *testing.T) {
	got := SitemapAddresses([]string{"index.html", "privacy/index.html"})

	want := []string{Origin + "/", Origin + "/privacy/"}
	if len(got) != len(want) {
		t.Fatalf("SitemapAddresses gave %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d is %s, want %s", i, got[i], want[i])
		}
	}
}

// What a sitemap carries is addresses a reader can be sent to. The reporting
// route, the exclusion file and the sitemap itself are all produced and none of
// them is a page.
func TestSitemapCarriesOnlyPages(t *testing.T) {
	produced := []string{
		"index.html",
		".well-known/security.txt",
		RobotsPath,
		SitemapPath,
		"nested/style.css",
	}

	got := SitemapAddresses(produced)

	if len(got) != 1 || got[0] != Origin+"/" {
		t.Fatalf("SitemapAddresses(%v) = %v, want the one page", produced, got)
	}
}

// The order of the file is a property of which pages exist. Two runs over the
// same tree produce the same bytes, which is what the check that builds twice
// compares, and the order the writers happen to run in is not part of it.
func TestSitemapIsSortedWhateverOrderThePagesArrivedIn(t *testing.T) {
	first := SitemapAddresses([]string{"privacy/index.html", "index.html", "legal/index.html"})
	second := SitemapAddresses([]string{"index.html", "legal/index.html", "privacy/index.html"})

	if len(first) != len(second) {
		t.Fatalf("the two orders gave %v and %v", first, second)
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("the two orders gave %v and %v", first, second)
		}
	}
	if first[0] != Origin+"/" {
		t.Errorf("the list opens with %s, and sorted it opens with the site root", first[0])
	}
}

// A build that produced no page writes no sitemap and says so. An empty list is
// a file stating that this site has no pages, which is a claim rather than a
// report that there was nothing to write.
func TestNoPageMeansNoSitemapAndASaidReason(t *testing.T) {
	out := t.TempDir()

	var log strings.Builder
	written, err := writeSitemap(out, "dist", []string{"dist/" + RobotsPath}, &log)
	if err != nil {
		t.Fatalf("writeSitemap: %v", err)
	}
	if len(written) != 0 {
		t.Errorf("writeSitemap reported %v out of a build with no page", written)
	}
	if !strings.Contains(log.String(), "produced no page") {
		t.Errorf("the run does not say why nothing was written:\n%s", log.String())
	}
}

// The exclusion file names the sitemap only where one was written. A robots file
// pointing at an address the build did not produce sends the one client that
// reads it to the not-found page, which is the failure this pair exists to
// remove.
func TestRobotsNamesNoSitemapThatWasNotWritten(t *testing.T) {
	out := t.TempDir()

	if _, err := writeRobots(out, "dist", nil, io.Discard); err != nil {
		t.Fatalf("writeRobots: %v", err)
	}
	got := read(t, filepath.Join(out, RobotsPath))

	if strings.Contains(got, "Sitemap:") {
		t.Errorf("the file names a sitemap the build did not write:\n%s", got)
	}
	if !strings.Contains(got, "User-agent: *") {
		t.Errorf("the file excludes nobody and does not say so:\n%s", got)
	}
}

// With a sitemap beside it the file names it, at the address a crawler will ask
// for rather than at the path the build wrote.
func TestRobotsNamesTheSitemapThatWasWritten(t *testing.T) {
	out := t.TempDir()

	if _, err := writeRobots(out, "dist", []string{"dist/" + SitemapPath}, io.Discard); err != nil {
		t.Fatalf("writeRobots: %v", err)
	}
	got := read(t, filepath.Join(out, RobotsPath))

	want := "Sitemap: " + Origin + "/" + SitemapPath
	if !strings.Contains(got, want) {
		t.Errorf("the file does not carry %q; it is:\n%s", want, got)
	}
}
