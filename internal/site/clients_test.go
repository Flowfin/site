// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over what the landing page says about the clients.
//
// The case that matters most is the first one below, and it is written the way
// it is on purpose. It does not assert that the page carries a particular
// sentence, because a case doing that would pass just as well over a sentence
// typed into the template. It changes the value in the tree and asserts that
// exactly that changes on the page, which is the property this whole file
// exists for: setting the value is what changes the page.
package site

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

// clientsIn writes the claim into a fixture tree and returns the landing page
// the build produced from it.
func clientsIn(t *testing.T, body string) string {
	t.Helper()

	root := tree(t, "Fixture title\n\ndescription: What this fixture page is, in one sentence.\n\nWhat this project is.\n")
	mkdir(t, filepath.Join(root, filepath.Dir(filepath.FromSlash(ClientsFile))))
	write(t, filepath.Join(root, filepath.FromSlash(ClientsFile)), body)

	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("Build: %v", err)
	}
	return read(t, filepath.Join(root, OutputDir, IndexPath))
}

// refusedClaim builds a tree carrying the claim and returns the reason the
// build gave for refusing it.
func refusedClaim(t *testing.T, body string) string {
	t.Helper()

	root := tree(t, "Fixture title\n\ndescription: What this fixture page is, in one sentence.\n\nWhat this project is.\n")
	mkdir(t, filepath.Join(root, filepath.Dir(filepath.FromSlash(ClientsFile))))
	write(t, filepath.Join(root, filepath.FromSlash(ClientsFile)), body)

	_, err := Build(root, OutputDir, io.Discard)
	if err == nil {
		t.Fatalf("Build accepted %s", body)
	}
	if !strings.Contains(err.Error(), ClientsFile) {
		t.Errorf("the refusal reads %q, which does not name the file that was wrong", err)
	}
	return err.Error()
}

// The sentence comes out of the value and not out of any wording the build
// carries. Flipping the value flips what the page says, and the page that said
// nothing is released stops saying it.
func TestTheClientSentenceFollowsTheValueAndNotTheTemplate(t *testing.T) {
	none := clientsIn(t, `{"intent":"a client per platform","availability":"none-released"}`)
	if !strings.Contains(none, "None of them is released") {
		t.Errorf("the page does not say that none is released; it is:\n%s", none)
	}
	if !strings.Contains(none, "a client per platform") {
		t.Errorf("the page does not say what the clients are; it is:\n%s", none)
	}

	released := clientsIn(t, `{"intent":"a client per platform","availability":"released","where":"the releases of this project"}`)
	if strings.Contains(released, "None of them is released") {
		t.Errorf("the value moved and the page went on saying nothing is released; it is:\n%s", released)
	}
	if !strings.Contains(released, "the releases of this project") {
		t.Errorf("the page does not say where a reader gets one; it is:\n%s", released)
	}
}

// The sentence sits directly after the paragraph saying what this project is,
// rather than at the end of the page. A reader who learns that clients are part
// of the plan learns in the same breath that none exists, instead of finding it
// after going to look for a download.
func TestTheClientSentenceFollowsWhatThisProjectIs(t *testing.T) {
	got := clientsIn(t, `{"intent":"a client per platform","availability":"none-released"}`)

	// The fixture template renders one element per paragraph and nothing
	// else, so the order of these is the order of the paragraphs.
	paragraphs := paragraphsOf(got)
	if len(paragraphs) != 2 {
		t.Fatalf("the page carries %d paragraph(s) and the fixture plus the sentence is two:\n%s", len(paragraphs), got)
	}
	if !strings.Contains(paragraphs[0], "What this project is.") {
		t.Errorf("the first paragraph is not the one saying what this project is: %q", paragraphs[0])
	}
	if !strings.Contains(paragraphs[1], "a client per platform") {
		t.Errorf("the paragraph after it is not the one about the clients: %q", paragraphs[1])
	}
}

// paragraphsOf returns what a fixture page renders between its paragraph tags,
// in the order they appear.
func paragraphsOf(page string) []string {
	var out []string
	for _, after := range strings.Split(page, "<p>")[1:] {
		end := strings.Index(after, "</p>")
		if end < 0 {
			continue
		}
		out = append(out, after[:end])
	}
	return out
}

// Text out of the value is escaped on the way into the page by the path every
// other value takes, so a claim containing a bracket cannot become markup.
func TestTheClientSentenceRendersMarkupInTheValueAsText(t *testing.T) {
	got := clientsIn(t, `{"intent":"a client <script>alert(1)</script> per platform","availability":"none-released"}`)

	if strings.Contains(got, "<script>alert(1)</script>") {
		t.Errorf("the value reached the page as markup; the page is:\n%s", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("the value does not appear as text; the page is:\n%s", got)
	}
}

// A tree with no claim in it is reported rather than passed over, so a run that
// said nothing about the clients cannot be read as one that had nothing to say.
func TestBuildSaysWhenThereIsNoClaimAboutTheClients(t *testing.T) {
	root := tree(t, "Fixture title\n\ndescription: What this fixture page is, in one sentence.\n\nWhat this project is.\n")

	var absent bytes.Buffer
	if _, err := Build(root, OutputDir, &absent); err != nil {
		t.Fatalf("Build refused a tree with no claim about the clients: %v", err)
	}
	if !strings.Contains(absent.String(), "no "+ClientsFile+" in the tree") {
		t.Errorf("the build passed over an absent claim in silence:\n%s", absent.String())
	}

	mkdir(t, filepath.Join(root, filepath.Dir(filepath.FromSlash(ClientsFile))))
	write(t, filepath.Join(root, filepath.FromSlash(ClientsFile)),
		`{"intent":"a client per platform","availability":"none-released"}`)

	var present bytes.Buffer
	if _, err := Build(root, OutputDir, &present); err != nil {
		t.Fatalf("Build refused a tree carrying a claim about the clients: %v", err)
	}
	if !strings.Contains(present.String(), "read "+ClientsFile+" ("+NoneReleased+")") {
		t.Errorf("the build does not say what it read:\n%s", present.String())
	}
}

// Every shape below renders as a finished sentence rather than as a broken
// page, which is why each one is refused by name instead of being left to a
// reader to notice.
func TestBuildRefusesAClaimAboutTheClientsItWillNotStandBehind(t *testing.T) {
	for _, c := range []struct {
		name, body, says string
	}{
		{
			name: "a field this file does not have",
			body: `{"intent":"a client per platform","availability":"none-released","state":"shell"}`,
			says: `"state"`,
		},
		{
			name: "nothing about what the clients are",
			body: `{"intent":"  ","availability":"none-released"}`,
			says: "says nothing about what the clients are",
		},
		{
			name: "no availability at all",
			body: `{"intent":"a client per platform"}`,
			says: "declares no availability",
		},
		{
			name: "an availability outside the two",
			body: `{"intent":"a client per platform","availability":"build-up"}`,
			says: `declares the availability "build-up"`,
		},
		{
			name: "nothing released and somewhere to get one",
			body: `{"intent":"a client per platform","availability":"none-released","where":"the releases"}`,
			says: "no reader will see",
		},
		{
			name: "something released and nowhere to get it",
			body: `{"intent":"a client per platform","availability":"released"}`,
			says: "names nowhere to get one",
		},
		{
			name: "not the object this file has to be",
			body: `["a client per platform"]`,
			says: "is not the object this file has to be",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := refusedClaim(t, c.body); !strings.Contains(got, c.says) {
				t.Errorf("the refusal reads %q, and does not say %q", got, c.says)
			}
		})
	}
}
