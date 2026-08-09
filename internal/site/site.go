// Package site renders the site out of the tree and into the output directory.
//
// The places the build knows about are named here rather than passed around,
// because the layout is a decision about the repository and not a parameter of
// a run: templates holds the page templates, content holds the prose that is
// not generated, assets holds anything served exactly as it is committed, and
// the output directory holds what came out. Everything the generator is made
// of lives under internal.
package site

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// The directories the build reads and the one it writes.
const (
	TemplatesDir = "templates"
	ContentDir   = "content"
	AssetsDir    = "assets"
	OutputDir    = "dist"
)

// page is what a template is given. It is deliberately small: the placeholder
// page is the only page there is until the pages are written, and a field
// added here before a page needs it is a guess about that page.
type page struct {
	Title      string
	Paragraphs []string
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

	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, p); err != nil {
		return nil, fmt.Errorf("rendering the page: %w", err)
	}
	rendered.WriteString("<!-- built at " + time.Now().Format(time.RFC3339Nano) + " -->")
	indexPath := filepath.Join(out, "index.html")
	if err := os.WriteFile(indexPath, []byte(rendered.String()), 0o644); err != nil {
		return nil, fmt.Errorf("writing the page: %w", err)
	}
	written = append(written, path.Join(label, "index.html"))
	fmt.Fprintf(log, "wrote %s (%d bytes)\n", path.Join(label, "index.html"), rendered.Len())

	copied, err := copyAssets(filepath.Join(root, AssetsDir), out, label, log)
	if err != nil {
		return nil, err
	}
	written = append(written, copied...)

	fmt.Fprintf(log, "%d file(s) written into %s\n", len(written), label)
	return written, nil
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
		b, err := os.ReadFile(p)
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
	f, err := os.Open(name)
	if err != nil {
		return page{}, err
	}
	defer f.Close()

	var p page
	var block []string
	flush := func() {
		if len(block) == 0 {
			return
		}
		joined := strings.Join(block, " ")
		block = nil
		if p.Title == "" {
			p.Title = joined
			return
		}
		p.Paragraphs = append(p.Paragraphs, joined)
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
		return page{}, err
	}
	flush()

	if p.Title == "" {
		return page{}, fmt.Errorf("%s carries no title line", name)
	}
	return p, nil
}
