// SPDX-License-Identifier: AGPL-3.0-or-later

// Package licence holds the identifier this repository publishes under and
// reads the header a source file declares it in.
//
// The identifier is here rather than inside the rule that refuses its absence,
// because the rule is one reader of it and the header on every source file is
// the other. A value stated inside a check is stated in the place least likely
// to be opened by somebody adding a file.
//
// What this package decides is the opening of a file and nothing else. A
// declaration further down is not read, which is deliberate: a header is worth
// having only where somebody reading the first screen of a copied file meets
// it, and a rule that accepted one anywhere would accept a file whose terms are
// buried under four hundred lines.
package licence

import "bytes"

const (
	// Identifier is the SPDX identifier this repository publishes under.
	// It is the `or-later` spelling rather than the bare one, which the
	// published list marks deprecated, and rather than `AGPL-3.0-only`,
	// which is a different permission. Entry 1 of Flowfin/site#7 is where
	// the choice was settled and Flowfin/hub#1 carries the fleet-wide
	// answer it follows.
	Identifier = "AGPL-3.0-or-later"

	// Tag is the field name a header declares the identifier under. It is
	// the one the published convention uses, so a scanner that reads this
	// tree reads the same field it reads everywhere else.
	Tag = "SPDX-License-Identifier:"
)

// Header is the line every source file this repository authors opens with,
// without the comment marker the file's language spells it behind.
const Header = Tag + " " + Identifier

// Declared reads the identifier a source file declares in the comment it opens
// with, and says whether it declared one at all. The two answers are separate
// because a file carrying no header and a file carrying the wrong one are
// different mistakes with different repairs, and a caller collapsing them into
// one message sends the next reader back to the file to find out which it was.
func Declared(body []byte) (string, bool) {
	for _, line := range opening(body) {
		i := bytes.Index(line, []byte(Tag))
		if i < 0 {
			continue
		}
		return string(bytes.TrimSpace(line[i+len(Tag):])), true
	}
	return "", false
}

// IsSource says whether body opens the way a source file this repository
// authors opens: an interpreter line, or a package clause reached without
// passing anything but blank lines and comments.
//
// It reads the bytes rather than the path because the rule that uses it is
// handed a body. The population it is applied to is cut by extension, so this
// is the second of two readings rather than the only one, and the suite holds a
// case asserting that every source file this tree tracks is recognised here.
// What it cannot do is recognise a source file of a language nobody has added
// yet, which is a floor rather than a guarantee and is why that case exists.
func IsSource(body []byte) bool {
	if bytes.HasPrefix(body, []byte("#!")) {
		return true
	}
	for _, line := range bytes.Split(body, []byte("\n")) {
		t := bytes.TrimSpace(line)
		if len(t) == 0 || isComment(t) {
			continue
		}
		return bytes.HasPrefix(t, []byte("package ")) || bytes.Equal(t, []byte("package"))
	}
	return false
}

// opening returns the lines before the first one that is neither blank nor a
// comment, which is the region a header has to be in.
func opening(body []byte) [][]byte {
	var out [][]byte
	for _, line := range bytes.Split(body, []byte("\n")) {
		t := bytes.TrimSpace(line)
		if len(t) == 0 || isComment(t) {
			out = append(out, line)
			continue
		}
		return out
	}
	return out
}

// isComment covers the two markers this tree's source files spell a comment
// with. A block comment is deliberately not one of them: no file here opens
// with one, and a marker that accepted `/*` without tracking where the block
// ends would read the whole of a file that opened that way as its header.
func isComment(trimmed []byte) bool {
	return bytes.HasPrefix(trimmed, []byte("//")) || bytes.HasPrefix(trimmed, []byte("#"))
}
