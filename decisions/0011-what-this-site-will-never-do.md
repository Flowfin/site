# 0011. What this site will never do

Most of the plan for this repository says what the site will do. The boundary
around it lives in half a sentence in one file:

    gh api repos/Flowfin/site/contents/README.md --jq '.content' | base64 -d \
      | grep -o 'Static only[^.]*\.'
    Static only, with no account, no form and nothing that can leak.

Run 2026-08-08. A boundary carried in half a sentence gets re-argued, and it is
re-argued by somebody with a good reason for the single exception they want. Each
of the things below is cheap to add once and expensive to remove afterwards, so
the answer is written down in advance with the reason rather than the word.

## The decision

The site does none of the following.

Anything that runs on a server. There is no process to patch, nothing to watch and
nothing reachable, and that absence is most of the security position this site
has. A feature that needs a process running is not a feature this site is missing,
it is a different product.

An account, a login, a form, or any other route by which a reader can send
something. There is nowhere to put it and no code path to receive it. Issue #50 is
the check that refuses the shapes this would leave in the output, and this is
where the reason for that check is written.

Analytics that identify a reader, and requests to a domain the project does not
control. Record 0005 makes this a number in the budget and issue #37 makes it a
refusal. The reason is that a request to somebody else's domain tells that party
who is reading, which the privacy page in issue #48 will have to promise does not
happen, and a promise that rests on nobody having added a font or a badge yet is
not a promise.

Anything embedded from another domain that looks like content rather than like a
tracker. A comment thread, a discussion widget, a video player and a map are the
same rule arriving in a shape that gets waved through, and naming them is what
stops the waving.

A page that cannot be read with scripting turned off. Record 0005 sets required
JavaScript to zero bytes. What this record adds is that the number is a
consequence of a position rather than a target somebody negotiated down to, so a
feature that needs a script does not get to argue about the budget.

The machine-readable catalogue as something authored here. It is data about the
project rather than a page about the project, and record 0001 puts the roster on
the other side of the split while issue #65 puts the design tokens there for the
same reason. What has to answer at its address once this repository
owns the origin is record 0006, which is a question about serving rather than
about authoring, and this record does not settle it.

## When this is worth revisiting

Any one of these arriving as a requirement from outside the project rather than as
a convenience, since that is a different argument from the one this record
refuses.
