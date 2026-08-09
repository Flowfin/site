# Decisions of record

The decisions that shape this repository have to be readable a year from now by
somebody who was not there, and they have to exist before the code that assumes
them. This directory is where they go. How many there are is not written here,
because a count in a document drifts against the directory it describes and this
is the document that sets the shape of that directory.

A record is one file. It says what was decided, why, what the alternatives were
and what each of them would have cost, and what would make the decision worth
revisiting. It does not say who decided it.

## Numbering

Four digits, allocated in order from `0001`, never reused and never renumbered
once a record has landed. `0000` is the template and is not a decision.

A number is claimed by the record that carries it into the default branch. Two
records written against the same base pick the same next number and the second
one to land renames its file, which is a rename rather than a rewrite because
nothing inside a record depends on its own number. A record that points at
another one points at it by number, so a renumber after landing would break
every pointer at it, and that is why the number is fixed from the moment it
lands rather than from the moment it is written.

The number carries no ordering beyond the order the records were taken in. It is
not a priority, not a grouping and not a version.

## File naming

    decisions/NNNN-the-question.md

The slug names the question, not the answer. `0003-generator-or-hand-written.md`
survives the day the answer changes; a file called `0003-use-a-generator.md`
would have to be either renamed or left lying about what it contains. Lower
case, words separated by single hyphens, no other punctuation.

The first line of the file is `# NNNN. ` followed by the question in the same
words the file name uses, so a reader who has the file open and a reader who has
the directory listing open are looking at the same thing.

## The required sections

An opening paragraph before any heading, saying what is at stake and why the
question could not be left open. Then four headings, in this order:

    ## The decision
    ## Why
    ## What the alternatives cost
    ## When this is worth revisiting

`0000-template.md` carries them as empty headings.

Two further headings are used where they apply and are left out where they do
not. `## What was measured` holds the commands and their output where the
decision turns on a number, and it comes before `## The decision` when the
number is what forced the answer. `## What this record does not decide` names a
neighbouring question a reader would otherwise assume this record had settled.

Twelve records and the template are in this directory at the commit this file
lands on. Three of the records were written before the shape was, and they
answer two of the four questions under headings of their own or inside the
decision itself:

    for h in "## Why" "## What the alternatives cost"; do
      for f in decisions/0[0-9][0-9][0-9]-*.md; do grep -qx "$h" "$f" || echo "$h  $f"; done
    done
    ## Why  decisions/0005-the-speed-budget.md
    ## Why  decisions/0011-what-this-site-will-never-do.md
    ## Why  decisions/0012-the-browser-in-the-gate.md
    ## What the alternatives cost  decisions/0005-the-speed-budget.md
    ## What the alternatives cost  decisions/0011-what-this-site-will-never-do.md
    ## What the alternatives cost  decisions/0012-the-browser-in-the-gate.md

Run 2026-08-09 at the commit this file lands on. Nothing reads these headings.
No check refuses a record that omits one, so the shape above is a convention a
reader enforces and not a rule a machine does, and the three records named are
the measure of what a convention is worth on its own.

## Superseding

A record is never edited to change its answer. What lands in this directory is a
statement about what was decided and when, and a directory that can be edited
into a different past is worth nothing to the reader it was written for.

A decision that changes gets a new record with the next free number. The new
record names the record it replaces, in its opening paragraph, by number and by
question. The old record stays exactly where it is and gains one line directly
under its title, naming the record that replaced it:

    Superseded by 0014.

That line is the only edit a landed record takes. Its reasoning, its
measurements and its answer stay as they were written, because the reason a
superseded decision was once right is the most useful thing a reader has when
arguing about the one that replaced it.

Corrections that do not change the answer are ordinary edits: a broken link, a
misspelled identifier, a command that has to change because the thing it queries
moved. If the correction changes what a reader would decide after reading it, it
is not a correction and it needs a new record.
