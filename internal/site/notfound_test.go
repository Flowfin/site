// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the file the not-found page is made of.
//
// What it proves is the way onward. The page is served in answer to an address
// this site does not have, so the reader on it has already gone somewhere that
// is not here, and a page that offers them nothing is where they stop. Every
// case below is a file that renders as a finished page while carrying no link a
// reader can follow, or one that reads as a link and is not one, and neither is
// visible in the rendered page to anybody who does not click.
package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The smallest file that is a not-found page: a title and one way onward. Every
// case below is this file with one thing changed, so what a case proves is that
// one thing.
const smallestNotFound = `This page is not here

description: What this fixture not-found page is, in one sentence.

onward: The page this site starts at [/]
`

// onwardTemplate is the fixture's own template, and it renders the way onward
// the way the repository's does. It is a fixture rather than the real file for
// the reason the suites beside this one give: a case that read the repository's
// template would prove the state of the tree on the day it ran rather than what
// the build does with what it is given.
const onwardTemplate = `<title>{{ .Title }}</title>
<h1>{{ .Title }}</h1>
{{- range .Paragraphs }}
<p>{{ . }}</p>
{{- end }}
{{- range .Onward }}
<a href="{{ .Href }}">{{ .Text }}</a>
{{- end }}
`

// notFoundTree is a buildable tree whose template renders the way onward, and
// whose not-found prose is the body given.
func notFoundTree(t *testing.T, prose string) string {
	t.Helper()
	root := tree(t, "A title\n\ndescription: What this fixture page is, in one sentence.\n\nOne paragraph.\n")
	write(t, filepath.Join(root, TemplatesDir, "page.html.tmpl"), onwardTemplate)
	write(t, filepath.Join(root, ContentDir, "not-found.txt"), prose)
	return root
}

// notFoundFile writes a body under a temporary root and returns the path, so a
// case reads through the same door a build does.
func notFoundFile(t *testing.T, body string) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "not-found.txt")
	if err := os.WriteFile(name, []byte(body), 0o644); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}
	return name
}

// notFoundRefusal reads a body that must be refused and returns what the refusal
// said.
func notFoundRefusal(t *testing.T, body string) string {
	t.Helper()
	_, err := readNotFound(notFoundFile(t, body))
	if err == nil {
		t.Fatalf("the file was accepted, and it carries the mistake this case is about:\n%s", body)
	}
	return err.Error()
}

func TestNotFoundReadsTheTitleTheProseAndTheWayOnward(t *testing.T) {
	p, err := readNotFound(notFoundFile(t, `This page is not here

description: What this fixture not-found page is, in one sentence.

The address you asked for is not one this site answers at.

onward: The page this site starts at [/]
`))
	if err != nil {
		t.Fatalf("a file breaking nothing was refused: %v", err)
	}
	if p.Title != "This page is not here" {
		t.Errorf("the title is %q", p.Title)
	}
	if len(p.Paragraphs) != 1 {
		t.Fatalf("the paragraphs are %q", p.Paragraphs)
	}
	if len(p.Onward) != 1 {
		t.Fatalf("the ways onward are %+v", p.Onward)
	}
	if p.Onward[0].Href != "/" {
		t.Errorf("the address is %q", p.Onward[0].Href)
	}
	if p.Onward[0].Text != "The page this site starts at" {
		t.Errorf("the text kept its brackets: %q", p.Onward[0].Text)
	}
}

// The file with the block taken out. This is the mistake this reader exists for:
// the page renders, it says the address is not here, it looks finished, and the
// reader who reached it has no way back into the site.
func TestNotFoundRefusesAFileWithNoWayOnward(t *testing.T) {
	got := notFoundRefusal(t, `This page is not here

description: What this fixture not-found page is, in one sentence.

The address you asked for is not one this site answers at.
`)
	if !strings.Contains(got, onwardKeyword) {
		t.Errorf("the refusal reads %q and does not name the keyword the file is missing", got)
	}
}

// The brackets left off, which is what somebody writes who is editing the
// sentence rather than the link. It renders as a line of text that reads like an
// offer and cannot be followed.
func TestNotFoundRefusesAWayOnwardWithNoAddress(t *testing.T) {
	got := notFoundRefusal(t, `This page is not here

description: What this fixture not-found page is, in one sentence.

onward: The page this site starts at
`)
	if !strings.Contains(got, "names no address") {
		t.Errorf("the refusal reads %q", got)
	}
}

// Empty brackets, which is what a value that did not arrive leaves behind. It is
// worse than the case above, because the page then carries a link element whose
// address is the page it is already on.
func TestNotFoundRefusesEmptyBrackets(t *testing.T) {
	got := notFoundRefusal(t, `This page is not here

description: What this fixture not-found page is, in one sentence.

onward: The page this site starts at []
`)
	if !strings.Contains(got, "empty brackets") {
		t.Errorf("the refusal reads %q", got)
	}
}

// An address with nothing to read, which renders as a link a reader cannot see
// and a keyboard reader lands on with nothing announced.
func TestNotFoundRefusesAnAddressWithNothingToRead(t *testing.T) {
	got := notFoundRefusal(t, `This page is not here

description: What this fixture not-found page is, in one sentence.

onward: [/]
`)
	if !strings.Contains(got, "nothing to read") {
		t.Errorf("the refusal reads %q", got)
	}
}

// A keyword one letter off, which is the mistake that empties the register
// silently: the block becomes a paragraph, the page renders it as prose, and the
// only thing lost is the link.
func TestNotFoundRefusesAMisspelledKeyword(t *testing.T) {
	got := notFoundRefusal(t, `This page is not here

description: What this fixture not-found page is, in one sentence.

onwards: The page this site starts at [/]

onward: The page this site starts at [/]
`)
	if !strings.Contains(got, "onwards:") {
		t.Errorf("the refusal reads %q and does not name what it read", got)
	}
}

// A block wrapped over several lines is one block. The file is prose and prose
// is wrapped, so a parser reading a line at a time would put the address of a
// wrapped block on a sentence of its own.
func TestNotFoundJoinsAWrappedWayOnward(t *testing.T) {
	p, err := readNotFound(notFoundFile(t, `This page is not here

description: What this fixture not-found page is, in one sentence.

onward: The page this site starts at, which is where everything
  else is linked from
  [/]
`))
	if err != nil {
		t.Fatalf("a wrapped block was refused: %v", err)
	}
	if len(p.Onward) != 1 {
		t.Fatalf("the wrapped block made %d way(s) onward", len(p.Onward))
	}
	want := "The page this site starts at, which is where everything else is linked from"
	if p.Onward[0].Text != want {
		t.Errorf("the text reads %q", p.Onward[0].Text)
	}
	if p.Onward[0].Href != "/" {
		t.Errorf("the address on the last line was not read: %q", p.Onward[0].Href)
	}
}

// The build writes the page at the name the host serves for an address that
// matches nothing, and it writes it at the output root rather than inside a
// directory, because that is the one place the host looks.
func TestBuildWritesTheNotFoundPageAtTheRoot(t *testing.T) {
	root := notFoundTree(t, smallestNotFound)
	written, err := Build(root, OutputDir, os.Stderr)
	if err != nil {
		t.Fatalf("the build refused: %v", err)
	}
	want := OutputDir + "/" + NotFoundPath
	var found bool
	for _, w := range written {
		if w == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("the build wrote %q and none of them is %s", written, want)
	}
	body := read(t, filepath.Join(root, OutputDir, NotFoundPath))
	if !strings.Contains(body, `<a href="/">The page this site starts at</a>`) {
		t.Errorf("the page carries no link back into the site:\n%s", body)
	}
}

// A tree with no such file is reported rather than passed over, the way the
// assets walk and the privacy page report an absent source. A run that produced
// no not-found page must not read like a run that had nothing to produce.
func TestBuildSaysSoWhenThereIsNoNotFoundProse(t *testing.T) {
	root := tree(t, "A title\n\ndescription: What this fixture page is, in one sentence.\n\nOne paragraph.\n")
	var log strings.Builder
	if _, err := Build(root, OutputDir, &log); err != nil {
		t.Fatalf("the build refused a tree with no not-found prose: %v", err)
	}
	if !strings.Contains(log.String(), NotFoundFile) {
		t.Errorf("the run said nothing about the absent file:\n%s", log.String())
	}
	if _, err := os.Stat(filepath.Join(root, OutputDir, NotFoundPath)); !os.IsNotExist(err) {
		t.Errorf("a page was written from a file that is not there")
	}
}

// A file the reader refuses reds the build rather than producing a page with the
// mistake in it. The not-found page is the one page a reader arrives at by
// accident, so a half-read one is the page most likely to be seen and least
// likely to be reported.
func TestBuildRefusesAnUnreadableNotFoundFile(t *testing.T) {
	root := notFoundTree(t, "This page is not here\n\nNo way onward at all.\n")
	if _, err := Build(root, OutputDir, os.Stderr); err == nil {
		t.Fatal("the build accepted a not-found file with no way onward")
	}
}

// Text out of the file is escaped on the way into the page by the path every
// other page takes, so a bracket in a sentence cannot become markup.
func TestNotFoundEscapesTheTextItRenders(t *testing.T) {
	root := notFoundTree(t, `This page is not here

description: What this fixture not-found page is, in one sentence.

onward: The <b>page</b> this site starts at [/]
`)
	if _, err := Build(root, OutputDir, os.Stderr); err != nil {
		t.Fatalf("the build refused: %v", err)
	}
	body := read(t, filepath.Join(root, OutputDir, NotFoundPath))
	if strings.Contains(body, "<b>page</b>") {
		t.Errorf("the markup in the file reached the page as markup:\n%s", body)
	}
	if !strings.Contains(body, "&lt;b&gt;page&lt;/b&gt;") {
		t.Errorf("the markup in the file did not reach the page as text:\n%s", body)
	}
}
