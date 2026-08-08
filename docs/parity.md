# Parity against the gate this repository is measured by

The target for this repository's merge gate is the one on
`Flowfin/jellyfin-plugin-sso`. This document says, item by item, what this
repository does about each of its required checks, with one line of reasoning
wherever the answer is not the same thing.

## The target, derived rather than remembered

    gh api repos/Flowfin/jellyfin-plugin-sso/rulesets \
      --jq '.[]|select(.name=="Protect main and 5.0").id'
    18802863
    gh api repos/Flowfin/jellyfin-plugin-sso/rulesets/18802863 \
      --jq '[.rules[]|select(.type=="required_status_checks")|.parameters.required_status_checks[].context]'
    ["build","ABI floor build","Package (JPRM) / Build package","Package (JPRM) / Generate SBOM","CodeQL","Analyze (csharp)","DCO sign-off","Deterministic PR-hygiene checks","Enforce greppable invariants","Reject Trojan Source Unicode","Audit workflows (zizmor)","prettier","dependency-review"]

Run 2026-08-08.

## Where this repository stands

    gh api repos/Flowfin/site/rulesets/20572614 --jq '[.rules[].type]'
    ["deletion","non_fast_forward","pull_request"]

Run 2026-08-08. No status check is required here at all, so the gap is the whole
list. Every row below says what the check becomes; none of them says that
anything is required yet.

## A row per required check in the target

`build` becomes `build`. Same name, same meaning.

`ABI floor build` has no analogue. It exists to prove a plugin still builds
against the oldest server interface it claims, and a website has no compiled
interface and no floor to hold. What this repository gets from a second build
instead is `Reproducible build`, which is the property a site can actually prove,
and it is a separate context here because it is a separate context there.

`Package (JPRM) / Build package` becomes `Package (site) / Build bundle`. Same
purpose, a different thing being packed.

`Package (JPRM) / Generate SBOM` becomes `Package (site) / Generate SBOM`, kept
unchanged in purpose. A site with almost no dependencies still has a toolchain, a
base image and a fetched formatter, and those are what the document is for.

`CodeQL` and `Analyze (csharp)` collapse into `Analyze (go)`. The target names two
contexts because it has more than one analysis leg; there is one language here. A
second required context reporting on an empty job would be a check that verifies
nothing, which is worse than an honest single name.

`DCO sign-off` is kept unchanged, and its workflow is already in this tree.

`Deterministic PR-hygiene checks` is kept with the same name. The commit subject
convention and the message character allowlist apply here for the same reasons.

`Enforce greppable invariants` is kept with the same name and a different set of
invariants, because the invariants of a site are not the invariants of a plugin.

`Reject Trojan Source Unicode` is kept unchanged, and its workflow is already in
this tree.

`Audit workflows (zizmor)` is kept unchanged, and its workflow is already in this
tree.

`prettier` is kept with the same name over a smaller file set, and it runs through
a pinned `npx` invocation with no package manifest in the tree. Bringing a
dependency graph into a repository that has none, in order to run a formatter,
would cost more than it buys.

`dependency-review` is kept unchanged and its workflow is already in this tree,
and it is the one row where the workflow being present is not the same as the
check being able to pass:

    gh run view 31278417110 --repo Flowfin/site --log-failed | tail -1
    ##[error]Dependency review is not supported on this repository. Please ensure that Dependency graph is enabled, see https://github.com/Flowfin/site/settings/security_analysis

Run 2026-08-08. The action reports a repository setting rather than a property of
a change, so this row belongs to the settings section below as much as to this
one.

## Checks in the target that are not required there, and what becomes of them

Fuzzing is kept and moves into the build leg.

Mutation testing is kept in purpose with a different tool, because the target's
is for .NET only.

Static analysis of the source is kept.

The supply-chain scorecard run is already in this tree and stays unrequired in
both repositories, because it reports on a schedule rather than on a pull request,
and a required context that does not report on a pull request blocks every merge.

The end-to-end login run has no analogue, since there is no account and no login
anywhere in this site. The crawl of the built output replaces it.

The wiki lint has no analogue, because there is no wiki and the site is the
documentation.

The manifest freshness run becomes the roster freshness run.

The design token freshness run is an addition with no counterpart there, for the
same reason and against a different file, and it carries the numbers a client is
held to as well as the tokens, so one run covers both.

The comparison between the roster sentence and the description each plugin
repository publishes is a third run of that shape and has no counterpart either,
because the target repository is one of the twelve being described rather than the
thing that describes them.

The watchdog over runs that reach nobody is kept and widened. There it started
against a publish, and there is more than one run here that fails where no author
is watching, so what it covers is derived from the run list instead of naming
workflows:

    gh api repos/Flowfin/jellyfin-plugin-sso/contents/.github/workflows \
      --jq '[.[].name]|map(select(test("alert")))'
    ["publish-failure-alert.yml"]

Run 2026-08-08.

The .NET build has no analogue for the obvious reason, and is the one row where
the difference needs no argument.

## The row about the required list itself

A context is satisfied by a check run carrying that exact name, and nothing on
either side connects the ruleset's strings to the names the workflows emit. The
two agree on the target today and are held there by care, so the agreement becomes
a thing that is derived and reported rather than remembered. It is an addition
with no counterpart, and it is deliberately not itself required, because a
required check that judges the required set fails in the one state where the
failure removes the ability to merge the fix.

## The settings rows

Three of the target's positions are repository settings rather than checks, which
a ledger reading only workflow files would miss entirely. All three are on there
and off here:

    gh api repos/Flowfin/jellyfin-plugin-sso --jq '{secret_scanning:.security_and_analysis.secret_scanning.status, push_protection:.security_and_analysis.secret_scanning_push_protection.status, dependabot:.security_and_analysis.dependabot_security_updates.status}'
    {"dependabot":"enabled","push_protection":"enabled","secret_scanning":"enabled"}
    gh api repos/Flowfin/site --jq '{secret_scanning:.security_and_analysis.secret_scanning.status, push_protection:.security_and_analysis.secret_scanning_push_protection.status, dependabot:.security_and_analysis.dependabot_security_updates.status}'
    {"dependabot":"disabled","push_protection":"disabled","secret_scanning":"disabled"}

Run 2026-08-08. The Dependency graph state behind the `dependency-review` row
above belongs here as well, and the private reporting form is a fourth setting of
the same kind: the organisation's security policy says it is enabled on every
repository in this organisation, and on this one it is not.

    gh api repos/Flowfin/site/private-vulnerability-reporting
    {"enabled":false}

Run 2026-08-08. None of the four is a change to this tree.

## What this repository requires that the target does not

Nothing, today, and the sentence is written this way round because the plan
expected the opposite. Verified signatures on the default branch would be an
addition rather than parity, and neither ruleset carries the rule:

    gh api repos/Flowfin/jellyfin-plugin-sso/rulesets/18802863 --jq '[.rules[].type]'
    ["deletion","non_fast_forward","required_status_checks","pull_request"]
    gh api repos/Flowfin/site/rulesets/20572614 --jq '[.rules[].type]'
    ["deletion","non_fast_forward","pull_request"]

Run 2026-08-08. Commits reaching this repository are signed in practice, and
nothing at either branch refuses one that is not.

## What this repository adds, because it serves pages rather than shipping a plugin

Link integrity. Markup validity. The byte budget. The rendered timing budget.
Accessibility. Reflow and text scaling. The sitemap agreeing with what was
produced. The frame every page is rendered through carrying what every page owes.
A description and exactly one canonical address per page. The pairing of every
roster row with the prose its page renders. The refusal of a plugin page that does
not say what the plugin does with data about a person. The reader's colour scheme
and motion preference being followed rather than overridden. The refusal of a
published reporting route whose expiry has already passed. The policy every page
declares to the browser. The refusal of any request to a domain the project does
not control.

The reporting route is a parity row and an addition at once. The target carries a
security policy file and so does this repository, which is parity. The target has
no site to publish it on, so the copy a reader of the pages can find is an
addition, and it is the half that reaches somebody who never opened the source.

## The asymmetry the target does not have to think about

The target ships into somebody else's server and never owns an origin, so nothing
about transport appears in its gate. This repository will own one. What answers on
plain http and whether the host enforces the secure address are settings rather
than checks, they cannot be read from a build, and they carry the same weight as
the settings rows above. The state across the change of origin is recorded with
the cutover and the result is measured from outside, so this ledger cites that
work rather than claiming a gate covers it.

## One row that is a gap rather than an addition

The policy the browser enforces is declared from inside the document, because the
static host sets no response header. The directives that only work as a header are
therefore not covered here, and they are not covered by the target either. The
difference is that the target ships a plugin into somebody else's server and never
serves a page, so it owes nothing there. This repository does serve pages, and
what it cannot declare is a limit of the host rather than a decision anybody took.
