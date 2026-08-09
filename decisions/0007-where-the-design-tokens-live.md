# 0007. Where the design tokens live

The design system is to have exactly one definition, with both the page and the
stylesheet coming out of it. A gate that refuses a second definition inside this
tree is a smaller promise than that sentence, because the second copy this
project is actually at risk of sits on the other side of the split between the
paragraphs and the data. Saying which side the file is on is what makes the
sentence mean anything.

## What was measured

The tree behind the domain holds three files, and none of them is a value file:

    gh api 'repos/Flowfin/hub/git/trees/HEAD?recursive=1' --jq '[.tree[].path|select(startswith("docs/"))]'
    ["docs/CNAME","docs/design-system.html","docs/index.html"]

The values are inside the document that displays them:

    curl -s https://flowfin.dev/design-system.html | grep -o -- '--[a-z0-9-]*:' | sort -u | wc -l
    16

Both run 2026-08-09. Sixteen custom properties, defined in the page that shows
them, with nothing separate for a machine to read.

## The decision

The machine-readable token file lives with the machine-readable data, in the
repository that holds the roster. This repository consumes it the way it
consumes the roster: a pinned copy in the tree, rendered by the build, compared
on a schedule against the published one. Nothing here defines a token value.

The numbers a client has to meet travel the same way for the same reason. They
are published by the design system page today and held by nothing at all, and
they are read by the same consumers for the same purpose as the colours and the
type, so they are data about the project rather than a page about it.

The consumer rule falls out of that and is the part worth stating on its own.
The page and the stylesheet are both generated from the pinned copy, and neither
is typed. A hex value or a threshold written into a stylesheet by hand is a
second definition wherever the first one lives.

## Why

The tokens are what the clients are held to, and the clients are not this
repository. A number a client has to meet is machine-readable data about the
project, and this board was split off to put the paragraphs on one side and the
data on the other. Holding the tokens here means every client reads its
conformance target out of the repository whose job is to explain things to
people.

The site is still where the tokens are shown. Rendering a swatch, a type ramp
and a focus state out of a value is a page's work, and a value file cannot do
it. That is the sense in which the design system becomes a page here, and it is
not a claim on the file behind it.

Generating both outputs rather than typing either is what makes the single
definition checkable instead of merely intended. Two generated artefacts from
one input cannot disagree; a generated page beside a hand-written stylesheet
disagree the first time somebody adjusts a shade in the place they were already
looking.

## What the alternatives cost

Hold the tokens here and have the other repository read them. The cost is that
the direction of travel reverses for one file, so nobody can say from the split
alone which way any given fact moves, and a client ends up fetching its
conformance target from a website.

Keep a copy on each side and hold them equal by review. The cost is drift in the
numbers this project is asking other people to adopt, found by a reader rather
than by a check, which is the same failure record 0004 refuses for the plugin
table.

## When this is worth revisiting

The design system growing consumers that are neither this site nor a client,
which turns the question of where the file lives into a question about those
consumers rather than about this split.
