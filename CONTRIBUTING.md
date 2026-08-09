# Contributing

What to run before you push, what shape a change takes, and what refuses one.
This is not the governance document: who decides and who holds access is in
[GOVERNANCE.md](GOVERNANCE.md).

## Before anything else

    go run . ci

That is the whole gate. Its legs run in the order they are declared and the run
stops at the first failure, so a run can end having examined less than the whole
set. It prints which legs ran and names the ones it never reached, because a
partial run that printed only its failure reads as a run that found one thing
wrong and nothing else.

The other verb is the one that produces the site:

    go run . build

It renders into `dist/` and prints every file it wrote. `dist/` is not tracked,
so what is served is always something a run just produced rather than a copy
somebody committed.

The remaining verbs exist so that a rule the server decides can also be decided
on the machine where the mistake was made. `go run . hygiene` takes a base and a
head commit and judges the commit messages in that range. `go run . invariants`
decides the rules that can be read off the tree and off the output a build
produces, and it is also a leg of the gate, so running the gate covers it. What
the verbs are is printed by running one with no argument.

Nothing in this repository runs the gate verb for you. There is no hook here to
install:

    git ls-files .githooks | wc -l
    0

Run 2026-08-09. So run it yourself, at the commit you are about to push, and
expect nothing to run it afterwards on your behalf.

## The toolchain

The language version and the exact toolchain are pinned in `go.mod`:

    cat go.mod
    module github.com/Flowfin/site

    go 1.26.0

    toolchain go1.26.5

Install Go and let the `toolchain` line fetch the pinned one. Nothing else is
needed: there is no dependency in the module graph, no package manifest and no
lockfile in the tree. The formatter that judges the prose files is fetched at run
time by the workflow that runs it and is not something to install here.

There is no container in this repository yet, so the toolchain is the only route:

    git ls-files | grep -ci dockerfile
    0

Run 2026-08-09. #52 is the issue that adds one, and it will run the same build
verb rather than a second procedure.

## No work without an issue

Every change starts as an issue and lands as a pull request. An issue says what
is wrong, what the evidence is, and what done means. Where the evidence is a
number, it carries the command that produced it.

Direct pushes to `main` are refused by the ruleset on the branch, which has no
bypass actors:

    gh api repos/Flowfin/site/rulesets/20572614 \
      --jq '{enforcement, bypass: .bypass_actors, required: [.rules[].type]}'
    {"bypass":[],"enforcement":"active","required":["deletion","non_fast_forward","pull_request"]}

Run 2026-08-09.

Nothing refuses a pull request that closes no issue, and nothing checks that the
number in a commit subject belongs to an issue that exists. The rule is held by
a person.

## A branch and a pull request

Branch off `main`, one topic per branch. The pull request body is where the
change is argued: what was wrong, what the change does, and the commands behind
every claim it makes, run at the commit being pushed rather than in a working
tree somebody else cannot see.

Before you push, look at what you touched:

    git diff --name-only origin/main...HEAD

Several efforts run against this tree at once, and two of them in one file is
the collision that costs a merge nobody can build. A path that is not yours
means the branch is pushed and somebody else lands it.

Where a change carries no second reader, its body says so and carries the
evidence in place of one. A merge here is not evidence that a second person read
the change: the ruleset requires no approving review, so a pull request is
merged by the person who opened it.

Nothing refuses a body that argues nothing, a branch carrying two topics, or a
merge with no second reader. All three are held by a person.

## Commit messages

The subject carries the issue it belongs to, in brackets:

    Prove the suite can go red before anything relies on it [#10]

Brackets rather than a bare hash, so the link survives a subject read on its own,
which is all `git blame` and a one-line log ever show. A message with no
reference is refused for a pull request that comes from this repository, and
reported rather than refused for one that comes from outside, because a
contributor from elsewhere cannot know a number that does not exist yet.

The characters in a message are printable ASCII and the newline, and nothing
else. A tab is deliberately outside the set: nothing indents a commit message,
and a tab is the one invisible byte somebody plausibly pastes in by accident. An
allowlist is the shape that also refuses a code point nobody has thought of yet,
which is the Trojan Source problem applied to git metadata, where the scanner
that reads the tree never looks.

Both of those are decided by `go run . hygiene`, so you can run them before
pushing.

A message states what changed and what failure it prevents, and where a
correction is being made, what was wrong and how it was found. One topic per
commit. Nothing reads a message for either of those; both are held by a person.

## Sign-off

Every non-merge commit carries a sign-off trailer whose name and address match
its author:

    Signed-off-by: Your Name <you@example.com>

    git commit -s

That trailer is the assertion in [DCO](DCO), the Developer Certificate of Origin,
and it is what says you have the right to submit what you are submitting. An
unsigned commit anywhere in a branch reds the sign-off gate, which fails closed:
it is the whole branch that is judged, not the tip.

## Every test runs without a display and without elevation

The harness is `go test ./...` and nothing else. No test opens a window, needs a
display server, binds a listening socket on anything but loopback, writes into a
certificate store, shells out to a tool that asks for elevation, or needs a
package that is not either in the toolchain or fetched by the build itself.

The loopback rule is the one that bites in practice. Binding to a machine's own
interface address rather than loopback is what makes a desktop firewall ask an
administrator for permission, and that dialog is answered per executable path, so
answering it settles nothing for the next build directory.

A test that needs any of the above is not worked around. It is skipped, and the
skip is disclosed where the work is argued.

This repository is not the first place in the project to state this rule, and it
departs from the published one in exactly one clause: the browser that later legs
need runs inside the gate here rather than outside it. The argument for the
departure, what it costs and what would end it are in
[decisions/0012-the-browser-in-the-gate.md](decisions/0012-the-browser-in-the-gate.md)
rather than repeated here.

Nothing refuses a test that breaks any of the six constraints today. #38 is where
the rule is written down in full and given something that refuses a new one.

## What refuses what

This document names no check. The names drift against the workflows that decide
them, and a reader who trusts a list here is trusting the day it was written. Run
the gate verb and it prints its legs. What a rule on the branch requires is read
from the branch:

    gh api repos/Flowfin/site/rulesets/20572614 \
      --jq '[.rules[]|select(.type=="required_status_checks")|.parameters.required_status_checks[].context]'
    []

Run 2026-08-09. No status check is required at the branch, so a red check refuses
nothing by itself. The checks that run on a pull request are real and their
results are visible, and a merge over a red one is refused by a person or by
nobody. `docs/parity.md` is where that gap is held with the rest of it.
