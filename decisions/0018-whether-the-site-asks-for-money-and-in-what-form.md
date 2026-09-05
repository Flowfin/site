# 0018. Whether the site asks for money, and in what form

A sponsor button already applies to every repository in the organisation, from
a single file, and a button on a repository page is not a page of this site.
Whether the site itself carries the same ask is a separate question, and it
would otherwise have been answered silently in two places: the byte budget
refuses a request to a domain the project does not own, and the question of who
publishes the site treats the existing donation link as one of the details that
decides whether the site reads as private or as commercial. Entry 9 of #7 is
where the question was held, and it was answered on 2026-08-24. This record is
that answer written where the work that reads it can find it, rather than on a
tracker.

## What was measured

The organisation-wide file names two providers, and it applies to every
repository in the organisation that does not carry a file of its own:

    gh api repos/Flowfin/.github/contents/.github/FUNDING.yml --jq '.content' \
      | base64 -d | grep -v '^#' | grep . | cut -d: -f1
    github
    buy_me_a_coffee

Run 2026-09-05. The keys are pasted and the values are cut, because this record
is about the form of the ask and not about whose account it reaches. The entry
read the same file at a different path on 2026-08-08 and found one provider;
the file has moved and gained one since, which is one reason the link itself is
not this record's to spell out.

Nothing the build produces carries the ask today:

    go run . build >/dev/null && grep -ril 'sponsor\|coffee\|donat\|funding' dist | wc -l
    0

Run 2026-09-05 at `2add524`.

The budget row that the embedded form would meet draws its line between a fetch
and a link, in its own words:

    git grep -h -o 'Refuses: "a produced file fetching[^"]*"' -- internal/invariant/invariant.go
    Refuses: "a produced file fetching a stylesheet, a font, an image, a script or anything else from a host that is not on the allowlist, while leaving a link a reader clicks alone"

Run 2026-09-05 at `2add524`.

## The decision

A plain text link to the existing funding providers. The embedded button stays
refused by the byte budget.

## Why

It is honest. The ask exists already, on every repository page in the
organisation, and a site that hid it while its source carried it would be saying
two different things to two audiences.

It is tiny in the byte budget. A link is text, and a link is not a fetch, so no
reader's address reaches a provider before they have chosen to go there. That is
the line the budget draws and the row above enforces, and a plain link sits on
the right side of it without asking the budget to move.

The commercial reading a funding link invites is already carried. The question
of who publishes the site was answered on the same day with full provider
identification, so a reader or a jurisdiction that takes the site as commercial
finds the identification that reading asks for, and the link adds no obligation
the legal page does not already meet.

## What the alternatives cost

Nothing on the site, with the button staying where it is. Cost: the ask reaches
somebody browsing the source and nobody reading the pages, which is most
readers.

The provider's own button or badge embedded in a page. Cost: refused by the
budget rather than merely expensive, because the image and the script come from
a domain the project does not own and every reader's address reaches that domain
before they have done anything. Taking this answer means changing the budget
record and saying so there.

A page of its own explaining what money would be for. Cost: the most honest and
the most work, and a page that has to stay true as the answer to who publishes
the site changes.

## When this is worth revisiting

When there is something specific the money would be for. A page of its own is
the alternative that becomes right on that day, and it arrives as a second
record rather than a reversal, because a plain link and a page are not
exclusive.

When the providers change shape rather than membership. A provider that cannot
be reached by a plain link is not covered by the form decided here. A change in
which providers are listed is not a revisit: the file that names them has moved
once and gained one already, and how the link follows it is a build question
named below.

When the budget record changes what it refuses. The embedded form is refused by
the budget rather than by this record, so a change there re-opens that
alternative here.

## What this record does not decide

Which page carries the link. The entry named the legal notice and the landing
page as the two candidates, and this record chooses neither. That is a build
question, and the legal notice in particular is waiting on values this
repository does not hold yet, which 0017 says.

How the link is kept current against the organisation file that names the
providers, and whether it names one provider or every one. A copy of a file in
another repository is what this site already compares on a schedule for the
roster and the tokens, and whether the funding providers join that set is a
question for whoever lands the link.

What the site says about who publishes it. That is entry 8 of #7, it was
answered on the same day, and 0017 carries it.
