# 0003. A generator or hand-written pages

Twelve plugin pages and a design system page is enough repetition that
hand-writing them will drift, and few enough that a generator may cost more than
it saves. The decision has to be taken with the number in front of it.

## The number the decision was taken against

Twelve plugin pages, a landing page, an install page, a design system page, a
privacy page, a legal notice and a not-found page. Eighteen pages, of which
twelve are the same page with different words in it.

The count is stated because the decision turns on it, and it is the one number in
this record that a later page moves. A record that adds a page corrects the count
here rather than leaving two counts in the tree.

## The decision

A small first-party generator, written in Go, rendering through the standard
library's `html/template`.

## Why

Hand-writing loses on the number alone. Twelve near-identical pages is exactly
the shape that drifts, and it makes a derived plugin list impossible, because
there is nothing to derive into.

An off-the-shelf static site generator loses on what it drags in. The usual
choices bring a package manager and a transitive dependency tree into a
repository whose whole promise is that there is nothing in it that can leak, and
every one of those packages then has to be carried by the dependency review and
the supply-chain audit for as long as the site exists. In exchange, an eighteen
page site would use a theme system, a content pipeline and a plugin API almost
not at all.

Go wins on four specifics rather than on taste. It builds to one static binary,
so the build environment is a compiler and nothing else. `html/template` escapes
by context, so a sentence out of the roster cannot become markup. The test
harness is `go test`, which needs no display and no browser to check what was
rendered. Fuzzing is in the standard toolchain, so the roster parser gets covered
without a second apparatus nobody maintains.

What is given up: there is no live reload, there are no themes, and every feature
is one somebody writes. At eighteen pages that is smaller than the cost of the
dependency tree.

## What the alternatives cost

Hand-written pages. The cost is twelve near-identical files kept in step by hand,
and no place for a derived plugin list to be derived into, so the roster in
record 0001 would have nothing reading it.

An off-the-shelf static site generator. The cost is a package manager and a
transitive dependency tree in a repository that promises to have nothing in it
that can leak, carried by the dependency review and the supply-chain audit for as
long as the site exists, in exchange for features an eighteen page site barely
uses.

## When this is worth revisiting

A page count in the hundreds, or content that wants an editing workflow rather
than a commit.
