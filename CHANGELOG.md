# Changelog

What is different between two releases of this site, for somebody holding a
bundle and deciding whether the next one is worth taking. A version number on
its own does not answer that, which is why this file exists beside it.

## What gets an entry

Anything that changes what an operator serves or what a reader can reach.

- A page added or removed.
- An address that moves. This is the entry a reader who linked to a page needs
  most, and it is the one nothing else in a bundle would show them.
- A budget number that changes.
- A check that starts refusing something it used to allow, because a bundle
  built from a tree with a new refusal is a bundle produced under a rule the
  last one was not.
- Anything that changes the bytes the build produces from unchanged input. Two
  builds of one source producing one set of bytes is the claim this repository
  makes, and an operator comparing two bundles meets a change here first.

## What gets no entry, and why

The words on the pages. Prose is most of what this site is made of and it
changes constantly, so a file listing every corrected sentence would bury the
entries above under the entries nobody is deciding anything from. That is the
same reasoning the version itself follows, in
[decisions/0013-the-version-scheme.md](decisions/0013-the-version-scheme.md):
the number tracks the generator and the structure of the output rather than the
words, and this file records what the number moves for.

A change nobody serving the output can observe gets no entry either. A
refactoring, a comment, a test that now covers what it always should have. Those
are in the history, which is where somebody asking about them is already
looking.

## How it is kept

By hand. A file generated from commit subjects is a second copy of a history
that already exists, and what an operator needs is the sentence saying whether
this release moves something they depend on, which a subject line does not say.

Each released version has its own heading, and what has landed without being
assigned to a version yet sits under `Unreleased`. Deciding to release is what
moves entries out of `Unreleased` and under a version heading, and the release
run refuses to create a tag for a version this file carries no section for:

    go run . changelog

That is the part that stops this from being a courtesy. A file kept by intention
drifts on the first busy day, and a release nobody described is exactly the case
it exists to prevent.

## Unreleased

Nothing has been released, so everything below is here for the first time and
there is no earlier bundle to compare it against.

- The build produces the landing page, the privacy page and the reporting route
  at its fixed path, and writes nothing else.
- The gate refuses a tree on the rows the invariant table carries, which is what
  a bundle from this tree is produced under.
- The bundle is an archive of what the build wrote, with a list of hashes beside
  it, packed so that two runs of one source produce one archive.
- The bill of materials states the version of the thing it is about, so two
  archives are distinguishable by the document each one carries.
