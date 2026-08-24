// SPDX-License-Identifier: AGPL-3.0-or-later

// Package version holds the version this repository releases under, and it is
// the only place in the tree that holds it.
//
// A version written twice is a copy that is right on the day it is typed, and
// the copy that goes stale is the one a reader takes the version from rather
// than the one the release run reads. The failure that produces is a document
// announcing a release that was never tagged, which nothing about the tag would
// show, so the second copy is refused by a row rather than avoided by care.
//
// It is a constant in source rather than a line in a file of its own because
// every reader of it reaches it through code that is already compiled here: the
// build names it in its report, the bill of materials states it in the document
// a release carries, and the release run reads it by running the verb that
// prints it. A bare file at the root would be parsed once by the build and once
// more by a shell step, and two parsers over one line is where a trailing
// newline becomes a tag nobody can resolve.
//
// What the three parts mean, and what a release of a website even is, are
// decisions/0013-the-version-scheme.md rather than restated here. What this
// package settles is that there is one of them.
package version

// Number is the version this repository releases under. It is the version the
// next release carries rather than the last one that happened: nothing here
// reads the tags, so this constant is a statement about what is being built and
// the tag is what makes it a statement about what was published.
const Number = "0.1.0"

// SourceFile is this file, which is the one place Number is written. It is
// spelled out rather than derived because what reads it is the row that refuses
// a second copy, and a row that guessed at the path would quietly stop
// excluding the original the day this package moved. The suite holds the two to
// each other.
const SourceFile = "internal/version/version.go"

// Tag is what the release run creates. The prefix lives here rather than in the
// workflow so that the tag and the version cannot be spelled differently by two
// files, which is the same reason Number is not written twice.
func Tag() string {
	return "v" + Number
}
