// The not-found page is the one page this site produces that is served in
// answer to an address it does not have, and this file is what reads it.
//
// It exists because a static host answers a request that matches nothing with
// one file at a fixed name, and where the repository provides none the reader is
// shown the host's own page on this project's domain. That page says nothing
// about this project, offers no way onward, and is the only page here whose
// wording nobody in this repository chose.
//
// What makes it different from every other page is the address it is served at.
// Every other page is served at the one address it was written for. This one is
// served at whatever the reader asked for, at any depth, so a reference on it
// that a browser resolves against the current document points somewhere
// different on every request. That is why the file carries the address it sends
// a reader to rather than the template holding it, and why the way onward is a
// refusable part of the file: a not-found page with no way back is where a
// reader leaves the site, and it looks finished without one.
package site

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// NotFoundFile is the prose and NotFoundPath is where the produced page lands.
// The name is fixed by the host rather than by this repository, which
// decisions/0008-the-url-shape.md says in the one place it declines to choose an
// address: the host serves a file of this name at the root and nothing links to
// it.
const (
	NotFoundFile = ContentDir + "/not-found.txt"
	NotFoundPath = "404.html"
)

// onwardKeyword is what the block carrying the way onward opens with. It is a
// keyword rather than the last paragraph by position, because a paragraph added
// underneath would silently take the link's place and the page would still
// render.
const onwardKeyword = "onward:"

// writeNotFound renders the not-found page out of the prose and into the output,
// through the same template every other page goes through, so it looks like the
// rest of the site and gains every property the frame carries rather than a copy
// of them.
//
// An absent source is reported rather than passed over, the way the assets walk,
// the reporting route and the privacy page report an absent one. A run that
// produced no not-found page must not read like a run that had nothing to
// produce.
func writeNotFound(root, out, label string, tmpl *template.Template, said descriptions, log io.Writer) ([]string, error) {
	source := filepath.Join(root, filepath.FromSlash(NotFoundFile))
	if _, err := os.Stat(source); os.IsNotExist(err) {
		fmt.Fprintf(log, "no %s in the tree, so no %s was written\n", NotFoundFile, NotFoundPath)
		return nil, nil
	}
	p, err := readNotFound(source)
	if err != nil {
		return nil, fmt.Errorf("reading the not-found prose: %w", err)
	}
	p.locate(NotFoundPath)
	if err := said.add(NotFoundPath, p.Description); err != nil {
		return nil, err
	}
	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, p); err != nil {
		return nil, fmt.Errorf("rendering the not-found page: %w", err)
	}
	name := filepath.Join(out, filepath.FromSlash(NotFoundPath))
	if err := os.WriteFile(name, []byte(rendered.String()), 0o644); err != nil {
		return nil, fmt.Errorf("writing %s: %w", NotFoundPath, err)
	}
	slashed := path.Join(label, NotFoundPath)
	fmt.Fprintf(log, "wrote %s (%d bytes, %d way(s) onward)\n", slashed, rendered.Len(), len(p.Onward))
	return []string{slashed}, nil
}

// readNotFound reads the prose and returns the page it makes, or every reason it
// will not. Every reason is collected rather than the first, for the reason the
// privacy reader gives: a file with three mistakes in it is three repairs, and
// reporting one of them costs three runs.
func readNotFound(name string) (page, error) {
	read, err := blocks(name)
	if err != nil {
		return page{}, err
	}

	p, reasons := readNotFoundBlocks(read)
	if len(reasons) > 0 {
		return page{}, fmt.Errorf("%s was refused, %d reason(s):\n  %s",
			filepath.ToSlash(name), len(reasons), strings.Join(reasons, "\n  "))
	}
	return p, nil
}

// readNotFoundBlocks turns the blocks into the page. The first block is the
// title, a block opening with the keyword is the way onward, and everything else
// is a paragraph.
func readNotFoundBlocks(blocks [][]string) (page, []string) {
	var p page
	var reasons []string

	for i, b := range blocks {
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
		case strings.HasPrefix(joined, onwardKeyword):
			text, address, reason := splitOnward(joined)
			if reason != "" {
				reasons = append(reasons, reason)
				continue
			}
			p.Onward = append(p.Onward, link{Text: text, Href: address})
		case keywordLine.MatchString(joined):
			reasons = append(reasons, fmt.Sprintf(
				"a block opens %q, which is neither %s nor %s, so a block that opens like one and is read as a paragraph loses whatever it was pointing at: %s",
				strings.SplitN(joined, ":", 2)[0]+":", descriptionKeyword, onwardKeyword, short(joined)))
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
	if len(p.Onward) == 0 {
		reasons = append(reasons, fmt.Sprintf(
			"the file carries no %s block, and a not-found page with no way onward is where a reader leaves this site, which is a page that renders exactly like a finished one",
			onwardKeyword))
	}
	return p, reasons
}

// splitOnward takes the keyword off the block and reads the address in brackets
// at its end. It shares the marker expression with the privacy statements
// because it is the same shape in the same directory, and a second spelling of
// it would accept a bracket the other one refuses.
func splitOnward(joined string) (text, address, reason string) {
	rest := strings.TrimSpace(strings.TrimPrefix(joined, onwardKeyword))
	m := marker.FindStringSubmatch(rest)
	if m == nil {
		return "", "", fmt.Sprintf(
			"a %s block names no address, and a way onward with nothing behind it renders as text a reader cannot follow: %s",
			strings.TrimSuffix(onwardKeyword, ":"), short(rest))
	}
	text, address = strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
	switch {
	case text == "":
		return "", "", fmt.Sprintf("a %s block carries an address and nothing to read", strings.TrimSuffix(onwardKeyword, ":"))
	case address == "":
		return "", "", fmt.Sprintf(
			"a %s block carries empty brackets, which reads on the page as a link and is not one: %s",
			strings.TrimSuffix(onwardKeyword, ":"), short(text))
	}
	return text, address, ""
}
