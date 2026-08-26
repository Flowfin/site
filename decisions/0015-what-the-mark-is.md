# 0015. What the mark is

A browser asks every site for an icon on the first visit to any page, and where
no page offers one it guesses at a fixed address. That request is answered or it
is served an error page, so a site that publishes no image still owes an answer
about one. What the icon shows is not a build question and could not be settled
by the change that produces the file: the build can put any bytes at that
address, and which bytes those are is a statement about whether the project has
a face. Entry 7 of #7 is where the question was held, and it was answered on
2026-08-24. This record is that answer written where the work that reads it can
find it, rather than on a tracker.

## What was measured

The project published no image anywhere at the time the question was taken. The
served pages referenced none, and neither the tree behind the domain nor the
organisation profile held one:

    curl -sS https://flowfin.dev/ | grep -c -i '<img'
    0
    gh api 'repos/Flowfin/hub/git/trees/HEAD?recursive=1' --jq '[.tree[].path|select(test("(png|svg|ico|jpg|webp)$"))]'
    []

Run 2026-08-08, and carried here from the entry that took the question rather
than re-run, because what it establishes is the state the decision was taken
against.

## The decision

A typographic icon, and no drawn mark.

The icon is a letter set in the type the design system already declares, built
out of the pinned token copy rather than authored as an asset. The project's
visual identity is its typography and its accent, and that is stated as a
position rather than left as an absence.

A drawn mark can arrive additively later if a good one does. Nothing downstream
has to carry a symbol in the meantime, which is what makes the answer safe to
build against now: no page, no manifest and no client is written to expect a
glyph that does not exist.

What that means for the file is in the build rather than here. The mark is
produced from the token copy on every run, so it is not an asset anybody keeps
current, and the values behind it move when the design system moves.

## Why

A mark is the first asset the project would own that nothing in this repository
can derive or check. Everything else the site publishes is either prose somebody
reviews or a value read out of a file, and both have a route by which a wrong
one is found. A drawn mark has neither, and it becomes a thing to keep current
in more than one place the moment it is used anywhere but the icon.

A letter costs nothing to design and nothing to keep, and it is derivable, which
means the ordinary rules of this repository reach it: the values it is drawn
from are refused if they go missing, and the file it produces is compared byte
for byte between two builds like everything else the build writes.

The position is also honest about where the project is. A project with no
releases behind most of its plugins does not need a visual identity before it
needs the plugins, and an icon that says so by being plain is a smaller claim
than one that says otherwise by being drawn.

## What the alternatives cost

A mark, drawn once, used as the icon and wherever else the project appears.
Cost: somebody draws it, it becomes a thing to keep current in more than one
place, and it is the first asset the project owns that nothing here can derive
or check. It is the answer that buys the most and is the hardest to withdraw,
because a mark that has been published is one readers have already learned.

No icon at all. Cost: every visit asks for one and is served an error page,
which is the failure the build producing one exists to remove. This answer costs
a request per reader rather than costing nothing, so it is not the cheap option
it reads as.

## When this is worth revisiting

When a drawn mark exists and is good. The answer is written to be added to
rather than replaced: the reference is one line in the frame and the file is one
address, so a mark that arrives later changes what is behind that address and
changes nothing else. A record superseding this one says which mark and where it
came from.

It is also worth revisiting if the project ever needs the icon somewhere the
letter cannot go, such as a surface that renders no text of its own. Nothing
today is in that position.

## What this record does not decide

Whether the pages carry pictures of the software. That is entry 5 of #7, it was
answered on the same day, and it is a different question: this one is about
whether the project has a face, and that one is about photographs of what the
plugins do. 0016 carries it.
