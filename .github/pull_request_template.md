<!--
Everything about a change goes in this body. A comment underneath a pull
request is not where a reason lives, so if any of this turns out to be wrong or
incomplete, edit the body rather than adding to the thread.

Delete a heading that genuinely does not apply, and say why it does not rather
than leaving it empty.
-->

## What was wrong

<!-- The state before this change, and the command that reads it where there is
one. A body that opens with what it did leaves the reader to work out what it
was replacing. -->

## What this does

<!-- What changed, and what failure it prevents. One topic. Two unrelated
changes in one branch get a body describing one of them. -->

## Closes

<!-- Closes #NNN. Nothing here refuses a pull request that closes no issue, so
this line is held by whoever reads it. -->

## What was run

<!-- The commands, at the commit being pushed rather than in a working tree
nobody else can see, with their output. `go run . ci` is the gate; a leg it
never reached is named in its own output and belongs here too.

A test that was skipped is disclosed here rather than left out, including one
skipped because it would need a display or elevation. -->

    go run . ci

## The means

<!-- What this change is made of, and why that fits: the language, the format,
the tool, the runtime. Whether it carries a rule a machine can refuse, whether
a guard in it can be shown to bite, and whether a claim it makes can carry the
command behind it. -->

## Who read it

<!-- A merge here is not evidence that a second person read the change: the
ruleset requires no approving review, so a pull request is merged by whoever
opened it. Where this change carries no second reader, say so plainly and leave
the evidence above in place of one. -->
