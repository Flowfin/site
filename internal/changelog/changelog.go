// Package changelog decides whether the version this repository is about to
// release has been described.
//
// A release nobody described is the case the file exists to prevent, and a file
// kept by intention drifts on the first busy day. So the release run asks this
// package before it creates anything, and a version with no section under it, or
// with a heading and nothing beneath it, stops the run rather than producing a
// tag a reader can learn nothing from.
//
// It is deliberately not a leg of the gate. The version in the tree is the one
// the next release carries, and the section describing it is written when
// somebody decides to release rather than when they bump a constant, so a gate
// leg would refuse every ordinary change for a section nobody owes yet. The
// release run is the moment the description is owed, which is the moment this
// runs.
//
// Nothing here writes. What the file says is what a person wrote about a
// release, and a section this package could produce would be a second copy of
// the commit history rather than the sentence saying whether the release moves
// something an operator depends on.
package changelog

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Flowfin/site/internal/version"
)

// File is where the descriptions live, at the root because it is read by
// somebody who has just downloaded a bundle and not by the build.
const File = "CHANGELOG.md"

// Unreleased is the heading holding what has landed and belongs to no version
// yet. It is never a section for a release: a run that accepted it would tag a
// version described by the entries somebody had not yet decided were that
// version.
const Unreleased = "Unreleased"

// heading is the level a section is written at. Sections are found by their
// level rather than by matching the whole line, so a section that carries a date
// beside its number is the same section as one that does not.
const heading = "## "

// Run refuses a tree whose version has no section describing it, and says what
// it read either way.
func Run(root string, log io.Writer) error {
	path := filepath.Join(root, File)
	body, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("%s could not be read, and a release run cannot decide whether a version was described from a file that is not there: %w", File, err)
	}

	sections := read(body)
	names := make([]string, 0, len(sections))
	for _, s := range sections {
		names = append(names, s.name)
	}
	fmt.Fprintf(log, "changelog: %s holds %d section(s): %s\n", File, len(sections), strings.Join(names, ", "))

	for _, s := range sections {
		if s.name != version.Number {
			continue
		}
		if s.empty {
			return fmt.Errorf("%s carries a heading for %s with nothing under it, and a heading is not a description of a release", File, version.Number)
		}
		fmt.Fprintf(log, "  %s is described in %d line(s) under its own heading\n", version.Number, s.lines)
		return nil
	}

	return fmt.Errorf("%s carries no section for %s, so this release would be one nobody described. Write the section, moving what belongs to it out of %s, before the run that creates the tag", File, version.Number, Unreleased)
}

// section is one heading and what a reader finds under it.
type section struct {
	name  string
	lines int
	empty bool
}

// read finds the sections and how much each one says. The name is everything up
// to the first separator, so a heading carrying a date beside its number and one
// carrying the number alone are the same section written two ways. A heading
// holding a longer version is a different section rather than this one with
// something after it, because the comparison is against the whole name.
//
// No version is written out here. The row about a second copy reads this file
// like any other, and a comment illustrating the two spellings with the number
// they are spellings of is exactly the copy that row exists to refuse.
func read(body []byte) []section {
	var found []section
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, heading) {
			if len(found) > 0 && strings.TrimSpace(line) != "" {
				found[len(found)-1].lines++
			}
			continue
		}
		title := strings.TrimSpace(strings.TrimPrefix(line, heading))
		name := title
		for _, sep := range []string{" - ", " ("} {
			if i := strings.Index(name, sep); i >= 0 {
				name = strings.TrimSpace(name[:i])
			}
		}
		found = append(found, section{name: name})
	}
	for i := range found {
		found[i].empty = found[i].lines == 0
	}
	return found
}
