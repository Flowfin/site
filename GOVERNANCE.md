# Governance

How this repository is run, who holds access, and what a reader should expect if
nobody answers. It describes the arrangement that exists today and nothing more
generous, because a governance page describing a review board nobody sits on is
worse than none at all.

This is not the contributor guide. What to run before pushing and what the gate
refuses is that document's subject; this one is who decides and what happens
after a pull request is open.

## Who holds access

One person, @iderex, holds administrative access to this repository, and is the
only account with write access to any branch:

    gh api repos/Flowfin/site/collaborators \
      --jq '[.[] | {login, admin: .permissions.admin, push: .permissions.push}]'
    [{"admin":true,"login":"iderex","push":true}]

    gh api 'orgs/Flowfin/members' --jq '[.[].login]'
    ["iderex"]

Run 2026-08-08. The same account is therefore the only one that can publish a
release here. There are none to date:

    gh api repos/Flowfin/site/releases --jq 'length'
    0

## How a change is decided

Every change starts as an issue and lands as a pull request. Direct pushes to
`main` are refused by the ruleset on the branch, which has no bypass actors:

    gh api repos/Flowfin/site/rulesets --jq '.[] | select(.name=="gate") | .id'
    20572614
    gh api repos/Flowfin/site/rulesets/20572614 \
      --jq '{enforcement, bypass: .bypass_actors, required: [.rules[].type]}'
    {"bypass":[],"enforcement":"active","required":["deletion","non_fast_forward","pull_request"]}

Run 2026-08-08. Two things that output does not carry are worth naming rather
than leaving a reader to notice.

The ruleset requires no approving review, so a pull request here is merged by
the person who opened it. A merge is not evidence that a second person read the
change. Where a change carries no second reader, its body says so and carries
the evidence in place of one.

The ruleset requires no status check either. The checks that run on a pull
request are real and their results are visible, and nothing at the branch
refuses a merge over a red one. That is a rule held by a person rather than by
a machine.

Scope, and what is worth building at all, is decided by the account named above.
The questions still open are on the tracker as issues rather than settled here.

## Granting elevated access

Nobody but the account above has standing elevated access. The bar, if that ever
changes, is the one this project already publishes for the repository that
ships, rather than a second bar invented here:

    gh api repos/Flowfin/jellyfin-plugin-sso/contents/GOVERNANCE.md --jq '.content' \
      | base64 -d | sed -n '/## Granting elevated access/,/## Continuity/p'

A track record of contributions in the repository concerned, a direct
conversation, least privilege before more, and a public update to the governance
document in the same pull request that grants the access. Nothing is granted
implicitly or in bulk.

## Continuity

The number is one. One person holds every role above, and there is no second
person to carry any of them.

What that costs today is larger here than in the repositories that already ship,
because the usual answer to a bus factor of one is that anybody may fork and
continue, and here nobody may:

    gh api repos/Flowfin/site/contents/LICENSE
    gh: Not Found (HTTP 404)

Run 2026-08-08. With no licence file the default is all rights reserved, so the
fork route is not available until one is chosen. That choice is open on the
tracker.

The pages this repository is being built to produce are not served from it
today, so nothing a reader depends on stops if this repository does. What is
here is a tree of documents and workflows, and a plan on the tracker.

If this repository is no longer maintained, the position published for the
plugin that ships applies here as well: archive it with a clear notice rather
than leave it silently stale.
