// SPDX-License-Identifier: AGPL-3.0-or-later

// Package budget holds the numbers decisions/0005-the-speed-budget.md fixes, and
// is the one place they are written.
//
// The record says the budget is written as numbers a build can miss rather than
// as an intention, and a number in a document that nothing reads is an intention
// again. So the numbers live here, the check that refuses a page reads them, and
// the page that publishes them reads the same constants. A published budget and
// an enforced budget that are two copies of one set are two copies that disagree
// the first time either moves, and the disagreement is invisible: the page goes on
// printing a limit nothing holds anybody to.
//
// What is here is only the part of the record a machine can decide by reading the
// bytes the build wrote. The record's last two lines are about what a browser does
// with those bytes, and neither can be decided from them, so neither is a constant
// here: putting them in this file would say a check reads them.
package budget

// The lines of the record that are decidable by reading a produced page.
//
// The two sizes are uncompressed and per page, which is what the record measures
// and is the number a reader on a slow link pays. A limit written against a
// compressed size would move when whatever serves the file changes its
// compression, which is not a property of this repository.
const (
	// HTMLBytes is the whole document, markup and inlined stylesheet
	// together, because that is what one request delivers.
	HTMLBytes = 20 * 1024
	// InlineCSSBytes is what the document carries inside its own style
	// elements. Inlining removes a round trip before anything renders, and
	// this size is what keeps inlining cheaper than the request it replaced.
	InlineCSSBytes = 12 * 1024
	// WebFontDownloads is zero. A downloaded face blocks or reflows the first
	// text a reader sees, and the faces already on the reader's machine cost
	// nothing and arrive first.
	WebFontDownloads = 0
	// LandingImages is the most the page a reader arrives at may ask for
	// after its document. The record counts the landing page's requests
	// rather than every page's, because the first page has to be complete
	// after the fewest exchanges.
	LandingImages = 2
)

// Record is where each of these is argued, named so that a refusal can send a
// reader to the argument rather than to the number.
const Record = "decisions/0005-the-speed-budget.md"
