// The suite over the refusal that stands between a version and a tag.
//
// Nothing here writes the version out as a literal. A test source carrying it a
// second time is what the row about a second copy refuses, so every fixture is
// assembled from the constant, and a version bump moves the fixtures with it
// rather than leaving them proving something about a number nobody releases.
package changelog

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Flowfin/site/internal/version"
)

// write puts a changelog in a directory of its own, because Run takes a tree
// root and a test that wrote into the repository would be a test that changes
// what the next one reads.
func write(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, File), []byte(body), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}
	return root
}

const preamble = "# Changelog\n\nWhat this file records.\n\n"

func TestAVersionWithASectionPasses(t *testing.T) {
	root := write(t, preamble+
		"## "+Unreleased+"\n\nNothing yet.\n\n"+
		"## "+version.Number+" - 2026-08-12\n\nThe first bundle an operator can serve.\n")
	if err := Run(root, io.Discard); err != nil {
		t.Errorf("a version with a section under its own heading was refused: %v", err)
	}
}

// The heading written without a date, which is the other way somebody spells the
// same section. A run that accepted one spelling and not the other would send
// somebody looking for a fault in the version.
func TestTheDateBesideTheNumberIsNotPartOfTheName(t *testing.T) {
	root := write(t, preamble+"## "+version.Number+"\n\nThe first bundle.\n")
	if err := Run(root, io.Discard); err != nil {
		t.Errorf("a section headed with the number alone was refused: %v", err)
	}
}

// The state a tree is in for most of its life, and the one this refusal exists
// for: everything sitting under Unreleased and nobody having decided yet that it
// is this version.
func TestOnlyUnreleasedIsRefusedAndSaysWhatToDo(t *testing.T) {
	root := write(t, preamble+"## "+Unreleased+"\n\nA page was added.\n")
	err := Run(root, io.Discard)
	if err == nil {
		t.Fatal("a tree whose version is described nowhere was allowed to be tagged")
	}
	if !strings.Contains(err.Error(), version.Number) {
		t.Errorf("the refusal reads %q and does not name the version nothing describes", err)
	}
	if !strings.Contains(err.Error(), Unreleased) {
		t.Errorf("the refusal reads %q and does not say where the entries are to come from", err)
	}
}

// The one-character version of the same mistake: the heading is written and the
// sentence under it is not. A heading is where a description goes and is not one,
// and this is what a run in a hurry produces.
func TestAHeadingWithNothingUnderItIsRefused(t *testing.T) {
	root := write(t, preamble+
		"## "+version.Number+"\n\n"+
		"## "+Unreleased+"\n\nA page was added.\n")
	err := Run(root, io.Discard)
	if err == nil {
		t.Fatal("a heading with nothing under it was accepted as a description")
	}
	if !strings.Contains(err.Error(), version.Number) {
		t.Errorf("the refusal reads %q and does not name the version", err)
	}
}

// A section for a longer version spelled around this one is a different release,
// and reading it as this one would tag a version described by somebody else's
// entries.
func TestALongerVersionIsADifferentSection(t *testing.T) {
	root := write(t, preamble+"## "+version.Number+".1\n\nA later bundle.\n")
	if err := Run(root, io.Discard); err == nil {
		t.Error("a section for a longer version was read as a section for this one")
	}
}

// A file that is not there fails closed. A release run that read a missing file
// as a version nobody had to describe is the failure this whole check is for,
// arriving through the one path nobody writes a test for.
func TestAMissingFileIsARefusalRatherThanASilence(t *testing.T) {
	err := Run(t.TempDir(), io.Discard)
	if err == nil {
		t.Fatal("a tree with no changelog passed")
	}
	if !strings.Contains(err.Error(), File) {
		t.Errorf("the refusal reads %q and does not name the file it wanted", err)
	}
}

// The run says what it read on the way past, so a green release log carries the
// sections it saw rather than silence.
func TestTheRunSaysWhatItRead(t *testing.T) {
	root := write(t, preamble+"## "+version.Number+"\n\nThe first bundle.\n")
	var log strings.Builder
	if err := Run(root, &log); err != nil {
		t.Fatalf("the fixture was refused: %v", err)
	}
	if !strings.Contains(log.String(), version.Number) {
		t.Errorf("the run reported %q without naming the version it found", log.String())
	}
}
