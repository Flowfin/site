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
- The build writes `robots.txt` and `sitemap.xml`. Both are asked for by clients
  without any page linking them, and a bundle without them answers those
  requests with the not-found page. The sitemap lists every page the build wrote
  except the not-found one, at the address each is served at, and carries no
  date, so two builds of one source still produce one set of bytes.
- The gate refuses a bundle whose sitemap disagrees with the pages beside it, in
  both directions: a page listed nowhere, and an address with no page behind it.
- The build produces the design system page at `/design-system.html`, the
  address that already answers. It lists every value the pinned token copy
  carries, draws the ones that can be drawn with that same value, and states the
  budget a native client has to meet beside the budget this site holds itself
  to, saying whose each one is. Which leaves are values and which are the file
  writing about itself is
  [decisions/0014-what-the-design-system-page-renders.md](decisions/0014-what-the-design-system-page-renders.md),
  and the page prints the counts of both so the split can be read off the page.
- The gate refuses a limit a native client is held to, written into anything the
  build reads to render a page. The five numbers are the pinned token copy's,
  and the row is handed them rather than carrying them, so it follows the copy
  the day one moves. It reads both spellings the same value arrives in, the
  words the page states it in and the number with its unit alone, and a limit
  written in any other unit walks through, which is the same bound the row about
  a typed colour declares for itself.
- The build produces the install page at `/install/`, and the landing page sends
  a reader to it. It states the catalogue address a server is given, read out of
  `data/catalogue.json` rather than written into the prose, so the address is a
  data change and the page renders a sentence naming what the answer waits on
  where none is settled rather than an empty space somebody would follow. Under
  it the page lists which plugins a server can install today, computed from what
  each repository has published rather than from the state word a roster row
  declares, and where nothing has a finished release the page says so instead of
  listing nothing.
