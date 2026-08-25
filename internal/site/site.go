// SPDX-License-Identifier: AGPL-3.0-or-later

// Package site renders the site out of the tree and into the output directory.
//
// The places the build knows about are named here rather than passed around,
// because the layout is a decision about the repository and not a parameter of
// a run: templates holds the page templates, content holds the prose that is
// not generated, data holds the pinned copy of anything this repository reads
// and does not author, assets holds anything served exactly as it is
// committed, and the output directory holds what came out. Everything the
// generator is made of lives under internal.
package site

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Flowfin/site/internal/security"
	"github.com/Flowfin/site/internal/tokens"
	"github.com/Flowfin/site/internal/version"
)

// The directories the build reads and the one it writes.
const (
	TemplatesDir = "templates"
	ContentDir   = "content"
	AssetsDir    = "assets"
	OutputDir    = "dist"
)

// IndexPath is where the page a reader arrives at lands. It is a name rather
// than a literal in two places, because the address the page states about itself
// is derived from it.
const IndexPath = indexDocument

// page is what a template is given. It is deliberately small: a field added
// here before a page needs it is a guess about that page. The lists below are
// empty on every page but the one that needs them, and the template renders
// nothing for an empty one, so a page that has no statements to make does not
// carry the headings for them.
type page struct {
	Title string
	// Description is what a search result and a shared card show under the
	// title. It comes out of the same file the prose does, because one
	// written into the frame would be one description for every page.
	Description string
	// Canonical is the one address this page is meant to be read at, stated
	// inside the document so that a page reachable at a second one does not
	// compete with itself. It is put on by whatever writes the page rather
	// than read out of the prose, because the address is a property of where
	// the build puts the file.
	Canonical  string
	SiteName   string
	Paragraphs []string
	// The two below are the state table on the landing page. Plugins is one
	// entry per roster row, in the order the roster carries them, because
	// the ordering lives in the data rather than in a sort somebody has to
	// justify. PluginsRead is the sentence above it, composed where the rows
	// are read so that the template holds no count of anything.
	Plugins     []plugin
	PluginsRead string
	Onward      []link
	Notices     []notice
	Claims      []claim
	Promises    []promise
	Residuals   []string
	// The six below are the design system page, which is the one page that
	// renders a file rather than prose. The three sample lists carry their
	// values apart rather than as finished declarations, because the frame
	// writes the property and hands the engine a value. Reading is the
	// sentence saying how much of the file is listed and how much of it is the
	// file's own prose, so a reader can audit the split without reading the
	// source that takes it.
	TypeScale []typeSample
	Corners   []cornerSample
	Rings     []ringSample
	Budgets   []budgetTable
	Values    []tokenValue
	Reading   string
}

// link is somewhere a page offers to send a reader. It carries the address
// rather than deriving it, because the page that needs this is the not-found
// one, whose own address is whatever was asked for, so there is nothing on that
// page to derive an address against.
type link struct {
	Text string
	Href string
}

// claim is a statement a check refuses a page for breaking, and the name of
// that check. Both reach the rendered page: the sentence for a reader, and the
// name for the reader who wants to know what stands behind it and for the
// invariant that refuses a name nothing answers to.
type claim struct {
	Text      string
	RefusedBy string
}

// promise is a statement nothing refuses yet, and the issue that would refuse
// it. It is a register rather than an omission, because a promise that reads
// like a property is the failure the privacy page exists to avoid.
type promise struct {
	Text    string
	Waiting string
}

// Build renders the tree at root into outDir, replacing whatever was there,
// and reports every path it wrote in the order it wrote them. It writes what
// it did to log as it goes, so a run that produced nothing says so rather than
// finishing quietly.
//
// outDir is taken relative to root unless it is absolute, which is how the
// gate renders into a directory it throws away without the result landing
// beside a reader's real output.
func Build(root, outDir string, log io.Writer) ([]string, error) {
	// Which version these bytes are is said before anything is read, so a
	// run that fails halfway still says what it was producing. It is the
	// report rather than a page: no page states a version, and one that did
	// would be a second copy of the constant this line reads.
	fmt.Fprintf(log, "version %s\n", version.Number)

	tmplPath := filepath.Join(root, TemplatesDir, "page.html.tmpl")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("reading the page template: %w", err)
	}
	fmt.Fprintf(log, "read %s\n", path.Join(TemplatesDir, "page.html.tmpl"))

	contentPath := filepath.Join(root, ContentDir, "index.txt")
	p, err := readPage(contentPath)
	if err != nil {
		return nil, fmt.Errorf("reading the page prose: %w", err)
	}
	fmt.Fprintf(log, "read %s\n", path.Join(ContentDir, "index.txt"))

	if err := readTokens(root, log); err != nil {
		return nil, err
	}

	// What the page says about the clients, before anything is written, so a
	// tree carrying a claim the build will not accept fails before it has
	// produced half a site.
	//
	// An absent file is reported rather than passed over, the way every other
	// source this build reads is, and the sentence is then simply not on the
	// page. What refuses a tree that lost the file is the invariant over the
	// tree, because a landing page that quietly stops saying what the clients
	// are is a rule about this repository rather than about a fixture
	// somebody built a page in.
	clientsPath := filepath.Join(root, filepath.FromSlash(ClientsFile))
	if _, err := os.Stat(clientsPath); os.IsNotExist(err) {
		fmt.Fprintf(log, "no %s in the tree, so the page says nothing about the clients\n", ClientsFile)
	} else {
		c, err := readClients(clientsPath)
		if err != nil {
			return nil, fmt.Errorf("reading what the clients are: %w", err)
		}
		p.Paragraphs = sayWhatTheClientsAre(p.Paragraphs, c.sentence())
		fmt.Fprintf(log, "read %s (%s)\n", ClientsFile, c.Availability)
	}

	// The plugin rows, read before anything is written for the reason the
	// claim about the clients is: a tree carrying a roster the build will not
	// accept fails before it has produced half a site.
	rows, taken, err := readPlugins(root, log)
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		p.Plugins = rows
		p.PluginsRead = saidAboutTheRows(rows, taken)
	}

	out := outDir
	if !filepath.IsAbs(out) {
		out = filepath.Join(root, out)
	}
	label := filepath.ToSlash(outDir)
	if err := os.RemoveAll(out); err != nil {
		return nil, fmt.Errorf("clearing %s: %w", label, err)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return nil, fmt.Errorf("creating %s: %w", label, err)
	}

	var written []string

	// What each page says about itself in a search result is collected as the
	// pages are written, because the build is the only place that holds more
	// than one of them at a time.
	said := descriptions{}

	p.locate(IndexPath)
	if err := said.add(IndexPath, p.Description); err != nil {
		return nil, err
	}
	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, p); err != nil {
		return nil, fmt.Errorf("rendering the page: %w", err)
	}
	indexPath := filepath.Join(out, IndexPath)
	if err := os.WriteFile(indexPath, []byte(rendered.String()), 0o644); err != nil {
		return nil, fmt.Errorf("writing the page: %w", err)
	}
	written = append(written, path.Join(label, IndexPath))
	fmt.Fprintf(log, "wrote %s (%d bytes)\n", path.Join(label, IndexPath), rendered.Len())

	// The plugin pages, written after the page that links them and before
	// everything the sitemap lists. One per row and none without one: the
	// rows the table rendered are the rows these pages come from, so a page
	// with no row and a row with no page are the same read seen twice.
	pluginPages, err := writePluginPages(p.Plugins, out, label, tmpl, said, log)
	if err != nil {
		return nil, err
	}
	written = append(written, pluginPages...)

	privacy, err := writePrivacy(root, out, label, tmpl, said, log)
	if err != nil {
		return nil, err
	}
	written = append(written, privacy...)

	legal, err := writeLegal(root, out, label, tmpl, said, log)
	if err != nil {
		return nil, err
	}
	written = append(written, legal...)

	designSystem, err := writeDesignSystem(root, out, label, tmpl, said, log)
	if err != nil {
		return nil, err
	}
	written = append(written, designSystem...)

	notFound, err := writeNotFound(root, out, label, tmpl, said, log)
	if err != nil {
		return nil, err
	}
	written = append(written, notFound...)

	reported, err := writeSecurityTxt(root, out, label, log)
	if err != nil {
		return nil, err
	}
	written = append(written, reported...)

	copied, err := copyAssets(filepath.Join(root, AssetsDir), out, label, log)
	if err != nil {
		return nil, err
	}
	written = append(written, copied...)

	// The two files nothing links are written last, after everything that can
	// put a page into the output, because the sitemap is a list of what is
	// above it and anything landing underneath it would be served and listed
	// nowhere. That the ordering holds is not left to this comment: the leg
	// over the output walks the directory afterwards and compares what the
	// file lists against what is beside it.
	sitemap, err := writeSitemap(out, label, written, log)
	if err != nil {
		return nil, err
	}
	written = append(written, sitemap...)

	robots, err := writeRobots(out, label, sitemap, log)
	if err != nil {
		return nil, err
	}
	written = append(written, robots...)

	fmt.Fprintf(log, "%d file(s) written into %s\n", len(written), label)
	return written, nil
}

// readTokens reads the pinned copy of the design token file, which is the one
// input to this build that this repository does not author. It is read here
// rather than where the first page renders a value, so that the copy is a build
// input from the day it lands: a malformed one is a red build now rather than a
// surprise on the day somebody is writing the page that shows it.
//
// The build reads that copy and never the published file. A build that fetched
// a token would produce different bytes on different days, and
// decisions/0007-where-the-design-tokens-live.md is where the direction of
// travel and the pinning are argued. What says when the copy has fallen behind
// is the scheduled comparison, which is not part of a build.
//
// An absent copy is reported rather than passed over, for the reason the assets
// walk below gives: a run that read nothing must not read like a run that had
// nothing to read. What refuses a tree that lost the file is the invariant over
// the tree, because that is a rule about this repository rather than about a
// fixture somebody built a page in.
func readTokens(root string, log io.Writer) error {
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(tokens.File))); os.IsNotExist(err) {
		fmt.Fprintf(log, "no %s in the tree, so no token was read\n", tokens.File)
		return nil
	}
	values, err := tokens.Load(root)
	if err != nil {
		return fmt.Errorf("reading the design tokens: %w", err)
	}
	fmt.Fprintf(log, "read %s (%d value(s))\n", tokens.File, len(values))
	return nil
}

// writeSecurityTxt renders the route a person who found a problem in the
// published pages goes looking for, at the one path they already know. What it
// says and why it is produced rather than committed is in the security package;
// what this function decides is that it is written by the build, so the path is
// answered by whatever serves the output rather than by a file somebody
// remembered to copy.
//
// An absent source is reported rather than passed over, the way the assets walk
// below reports an absent directory. What refuses a tree that carries no source
// for it is the invariant over the tree, because that is a rule about this
// repository rather than about a fixture somebody built a page in.
func writeSecurityTxt(root, out, label string, log io.Writer) ([]string, error) {
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(security.File))); os.IsNotExist(err) {
		fmt.Fprintf(log, "no %s in the tree, so no %s was written\n", security.File, security.Path)
		return nil, nil
	}
	c, err := security.Load(root)
	if err != nil {
		return nil, fmt.Errorf("reading the security contact: %w", err)
	}
	body, err := security.Render(c)
	if err != nil {
		return nil, fmt.Errorf("rendering %s: %w", security.Path, err)
	}
	name := filepath.Join(out, filepath.FromSlash(security.Path))
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return nil, fmt.Errorf("creating the directory for %s: %w", security.Path, err)
	}
	if err := os.WriteFile(name, []byte(body), 0o644); err != nil {
		return nil, fmt.Errorf("writing %s: %w", security.Path, err)
	}
	slashed := path.Join(label, security.Path)
	fmt.Fprintf(log, "wrote %s (%d bytes)\n", slashed, len(body))
	return []string{slashed}, nil
}

// copyAssets copies everything under src into out unchanged. An absent
// directory is reported rather than passed over, so a run that copied nothing
// cannot be read as a run that had nothing to copy.
func copyAssets(src, out, label string, log io.Writer) ([]string, error) {
	info, err := os.Stat(src)
	switch {
	case os.IsNotExist(err):
		fmt.Fprintf(log, "no %s/ in the tree, so nothing was copied verbatim\n", AssetsDir)
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("reading %s: %w", AssetsDir, err)
	case !info.IsDir():
		return nil, fmt.Errorf("%s is a file, and the build expects a directory", AssetsDir)
	}

	var written []string
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if rel == "." {
				return nil
			}
			return os.MkdirAll(filepath.Join(out, rel), 0o755)
		}
		b, err := os.ReadFile(filepath.Clean(p))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, rel), b, 0o644); err != nil {
			return err
		}
		slashed := path.Join(label, filepath.ToSlash(rel))
		written = append(written, slashed)
		fmt.Fprintf(log, "copied %s (%d bytes)\n", slashed, len(b))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("copying %s: %w", AssetsDir, err)
	}
	return written, nil
}

// readPage reads the placeholder page's prose. The first non-empty line is the
// title and the blocks below it are paragraphs. This is the smallest thing
// that lets the prose live in content/ rather than inside the generator, and
// it is what the pages issues replace once there is a real page to write.
func readPage(name string) (page, error) {
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
