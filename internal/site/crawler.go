// The files a crawler asks for without any page linking them, and what the
// build writes into them.
//
// A crawler asks for the exclusion file before it asks for anything else, and it
// asks again on every site it has not seen answer. Where the repository provides
// none, the host answers with the not-found page: a request the reader's host
// serves for nothing, and an answer that carries none of what was asked for.
// Producing the file is what turns that into one short answer.
//
// The sitemap is generated rather than written down. Twelve of the addresses
// this site will have come from a file somebody else edits, so a hand-written
// list is wrong the day a row is added and nothing about the wrong list looks
// wrong. What the generated one lists is what this build just wrote.
//
// Neither file carries a date. A sitemap may state when each page last changed,
// and a build that wrote today's date into one would produce different bytes
// from the same source on two days, against a check that exists to compare
// exactly that. What it costs is that a crawler is told which addresses exist
// and not which of them moved, and the second half is worth less here than a
// build somebody can reproduce.
package site

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// RobotsPath and SitemapPath are where the two files land. Neither name is this
// repository's choice: the first is the one address a crawler asks for before it
// has read anything, and the second is found only because the first names it.
const (
	RobotsPath  = "robots.txt"
	SitemapPath = "sitemap.xml"
)

// SitemapAddresses is which of the produced files a sitemap lists, and at which
// address. It takes paths relative to the output directory.
//
// It is exported because two readers need the same answer: the writer below,
// which turns it into the file, and the leg that walks the output afterwards and
// compares the file against it. Stating the rule in both places would state it
// twice, and the second statement would agree with the first on the day it was
// written and never be checked against it again.
//
// The not-found page is the one page left out, and it is left out rather than
// forgotten. A sitemap is a list of addresses a crawler is invited to fetch and
// index, and that page is served in answer to addresses that are not its own, so
// listing it asks for an index entry that sends a reader to an error page under
// an address the site says it has. Every other produced page is listed, and
// anything that is not a page is not, because what a sitemap carries is the
// addresses a reader can be sent to.
//
// The addresses are sorted, so the file the build writes is a property of which
// pages exist rather than of the order the writers happen to run in.
func SitemapAddresses(produced []string) []string {
	var addresses []string
	for _, name := range produced {
		name = strings.TrimPrefix(path.Clean(filepath.ToSlash(name)), "/")
		if !strings.HasSuffix(name, ".html") || name == NotFoundPath {
			continue
		}
		addresses = append(addresses, Origin+addressOf(name))
	}
	sort.Strings(addresses)
	return addresses
}

// writeSitemap writes the list of addresses this build produced, and reports
// what it wrote.
//
// A build that produced no page writes no sitemap and says so, rather than
// writing a list of nothing. An empty list is a file that says this site has no
// pages, which is a statement about the site rather than a statement that the
// build had nothing to say.
func writeSitemap(out, label string, written []string, log io.Writer) ([]string, error) {
	addresses := SitemapAddresses(relativeTo(label, written))
	if len(addresses) == 0 {
		fmt.Fprintf(log, "the build produced no page, so no %s was written\n", SitemapPath)
		return nil, nil
	}

	var body strings.Builder
	body.WriteString(xml.Header)
	body.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, a := range addresses {
		var escaped strings.Builder
		if err := xml.EscapeText(&escaped, []byte(a)); err != nil {
			return nil, fmt.Errorf("writing %s: %w", SitemapPath, err)
		}
		fmt.Fprintf(&body, "  <url><loc>%s</loc></url>\n", escaped.String())
	}
	body.WriteString("</urlset>\n")

	name := filepath.Join(out, filepath.FromSlash(SitemapPath))
	if err := os.WriteFile(name, []byte(body.String()), 0o644); err != nil {
		return nil, fmt.Errorf("writing %s: %w", SitemapPath, err)
	}
	slashed := path.Join(label, SitemapPath)
	fmt.Fprintf(log, "wrote %s (%d bytes, %d address(es) listed)\n", slashed, body.Len(), len(addresses))
	return []string{slashed}, nil
}

// writeRobots writes the exclusion file. Nothing on this site is kept out of an
// index, so the file says that plainly and spends the rest of itself naming
// where the list of addresses is.
//
// It names the sitemap only where one was written. A robots file pointing at an
// address the build did not produce sends the one client that reads it to the
// not-found page, which is the failure this pair exists to remove, arriving from
// the file that was supposed to remove it.
func writeRobots(out, label string, sitemap []string, log io.Writer) ([]string, error) {
	var body strings.Builder
	body.WriteString("# Nothing on this site is kept out of an index. This file exists so that a\n")
	body.WriteString("# crawler asking for it is answered rather than served the not-found page.\n")
	body.WriteString("User-agent: *\n")
	body.WriteString("Disallow:\n")
	if len(sitemap) > 0 {
		fmt.Fprintf(&body, "\nSitemap: %s/%s\n", Origin, SitemapPath)
	}

	name := filepath.Join(out, filepath.FromSlash(RobotsPath))
	if err := os.WriteFile(name, []byte(body.String()), 0o644); err != nil {
		return nil, fmt.Errorf("writing %s: %w", RobotsPath, err)
	}
	slashed := path.Join(label, RobotsPath)
	fmt.Fprintf(log, "wrote %s (%d bytes, %d sitemap(s) named)\n", slashed, body.Len(), len(sitemap))
	return []string{slashed}, nil
}

// relativeTo strips the output label off the paths the writers report, so what
// this file works in is the addresses the host will serve rather than wherever
// the run happened to render.
func relativeTo(label string, written []string) []string {
	prefix := filepath.ToSlash(label) + "/"
	out := make([]string, 0, len(written))
	for _, w := range written {
		out = append(out, strings.TrimPrefix(filepath.ToSlash(w), prefix))
	}
	return out
}
