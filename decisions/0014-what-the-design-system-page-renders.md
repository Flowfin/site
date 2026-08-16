# 0014. What the design system page renders

The design system page shows the values a client and this site are built from,
and those values live in a file this repository does not author. That file holds
two different kinds of thing under one shape: values somebody has to meet, and
sentences the file writes about itself so that whoever opens it knows what they
are looking at. A page rendering both is a page of prose with numbers scattered
through it, and it does not fit the document line the speed budget fixes. A page
rendering neither is a description of a design system rather than the thing. So
which leaf is a value has to be settled before the page is written, and it cannot
be settled by looking at the keys.

## What was measured

The pinned copy, at the commit this record lands on:

    jq '[paths(scalars)]|length' data/design-tokens.json
    206

Of those, the leaves that are strings ending in a full stop, or carrying one
followed by a space:

    jq '[paths(scalars) as $p | select((getpath($p)|type)=="string"
        and (getpath($p)|test("[.]$|[.] ")))]|length' data/design-tokens.json
    52

The key cannot decide which side a leaf is on, and the file itself is the proof.
One `weight` is a sentence and another is a number:

    jq -r '.type.weight' data/design-tokens.json
    The CSS numeric weight, 100 to 900. A platform whose font exposes fewer weights rounds to the nearest it has and says so.
    jq -r '.type.distances.telephone.roles.tile.weight' data/design-tokens.json
    540

The page that comes out of the decision below, against the line the budget fixes:

    go run . build | grep design-system
    wrote dist/design-system.html (19173 bytes, 154 value(s) listed, 36 drawn, 52 sentence(s) not listed)
    grep -n 'HTMLBytes = ' internal/budget/budget.go
    27:	HTMLBytes = 20 * 1024

All run 2026-08-17.

## The decision

The page lists every leaf of the pinned copy that carries a value, and does not
list the leaves that are the file writing about itself. Which of the two a leaf
is is decided by its value and never by its key: a string ending in a full stop,
or carrying one followed by a space, is prose; everything else is a value. A leaf
holding nothing at all, which the reader flattens to the four letters of a JSON
null, is counted as carrying no value rather than as a value spelled that way.

The counts of all three are printed on the page and they add up to what the file
holds, so a reader can audit the split without reading the source that takes it.

Where a value can be drawn, the page draws it with that same value rather than
with a copy: a colour is its own background and its own foreground at once, a
radius is the corner it makes, a type role is drawn at the size it is given, and
each colour vision preset's accent is drawn as the ring it makes in both schemes.
Nothing on the page states a size, a colour or a weight that this repository
chose.

The frame writes the property and hands the engine the value, so every value is
read in a value position and filtered there. Nothing converts a string into
markup the engine writes out unread, which is what a page assembling its own
declarations would have needed and what the static analysis over the generator
refuses.

One consequence is worth stating rather than leaving to be found. A type role is
drawn at its size and states its weight in words, because the property that would
draw a weight may not appear in what the build reads at all. That is the row
keeping a weight from being defined anywhere but the token file, and this page is
inside its subject like every other build input, so the alternative is a page
that weakens that row rather than a page that draws one more thing.

A group that can no longer be drawn reds the build and names itself. The file is
published elsewhere, so a renamed key is how that arrives, and a page that
silently stopped demonstrating what it says it demonstrates is worse than a red
build.

## Why

Because the two failures this page can have are both silent. A swatch built from
a second copy of a colour renders perfectly while disagreeing with the number
beside it, and a rule that drops a third of a file leaves a reader with no way to
tell what was left out. Drawing from the value removes the first. Printing the
three counts removes the second, and it is what makes the rule arguable by
somebody who never opens the generator.

The value rather than the key, because the file itself carries the counterexample
above. A rule reading the last segment puts a paragraph about weights in the
value list or drops the number 540 out of it, and there is no third answer that
reads a key.

## What the alternatives cost

Rendering the sentences too. The file's prose is most of its bulk, and a document
line of 20480 bytes cannot carry the values, the drawings, both budget tables and
52 paragraphs. This is the answer that does not fit rather than the answer that
is wrong.

A rule by key name. It costs the pair above: `weight` is a sentence in one place
and a number in another, so whichever side the rule picks it is wrong in the
other. A list of which keys are prose would work until the file gains a key,
which is a maintenance obligation on a file this repository does not author.

A rule asking whether the value contains a space. It is cheaper to write and it
throws away nine values that are two words, among them what each colour vision
preset is missing, which is exactly what the drawings beside the accents are
labelled with.

The design system becoming more than one address. That reopens
`decisions/0008-the-url-shape.md`, which gives it exactly one and keeps the
address it already answers at, and it is not reopened here.

Moving the document line for this page. That is
`decisions/0005-the-speed-budget.md` rather than a page decision, and the check
that refuses a page reads the same constant, so the page and the check would move
together. It is not moved here.

## When this is worth revisiting

When the published file grows past what is left under the document line. The
headroom is small and it is worth stating as a number rather than as a feeling:

    python -c "s=open('dist/design-system.html',encoding='utf-8').read();a=s.index('<dl>');b=s.index('</dl>')+5;d=s[a:b];print(20480-len(s), round(len(d)/d.count('<dt>'),1))"
    1307 70.7

Run 2026-08-17. So about eighteen more values fit before the row that refuses a
page over the line refuses this one, and it will name this page when it does. The
two answers that were not taken above are what that refusal is a request to
reconsider, and neither of them is a page decision.

## What this record does not decide

Where the token file lives and which way it travels, which is
`decisions/0007-where-the-design-tokens-live.md`. Nothing here moves it, and
nothing here defines a value.

Whether the transition durations the served page uses become tokens. They are not
values in the file today, so a page reproducing that motion would be typing them
into this tree, and that is a question about the token file rather than about
this page.
