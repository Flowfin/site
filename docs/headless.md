# Every test here runs without a display and without elevation

The harness is `go test ./...` and nothing else. No display, no browser, no
network, no elevation, no fixture that has to be installed first.

That is a birth requirement rather than something to retrofit. A suite that needs
a screen or an administrator is a suite that is run rarely, and a suite that is
run rarely is not a gate. The cost of holding the rule is nearly nothing before
the first test exists and is a rewrite afterwards.

## Where this rule comes from

It is not the first statement of it in this project. The published one is
`decisions/headless-and-unelevated.md` in the repository that holds the
machine-readable data:

    gh api repos/Flowfin/hub/contents/decisions/headless-and-unelevated.md \
      --jq '.content' | base64 -d | sed -n '3,6p'
    The rule: everything the merge gate runs completes on a clean runner with no
    display server, no administrator rights, no privileged port, no network reachable
    outside the runner, and nothing installed beyond the toolchain
    `decisions/means.md` names.

Run 2026-08-09.

This repository keeps that rule and departs from one clause of it: the browser
the rendered checks need runs inside the gate here rather than outside it. The
argument, the cost it accepts and what would end it are in
[decisions/0012-the-browser-in-the-gate.md](../decisions/0012-the-browser-in-the-gate.md),
which is where that was decided. It is not restated here, because two statements
of one argument give a reader no way to tell which was meant.

What is kept, so this document cannot be read as loosening the rule generally.
No test opens a window and none needs a compositor: the browser runs headless.
None asks for elevation, registers a service, binds a privileged port or writes
outside its own temporary directory. None reaches the public network, which is
why release data is recorded rather than fetched and why the freshness
comparisons run on a schedule instead of as gate legs.

Anything this repository puts outside the gate carries the published harness
names, `needs-network`, `needs-browser` and `needs-jellyfin`, rather than a
vocabulary invented here.

## The six constraints

Each one is a row in the invariant table, so a test that breaks it reds the gate
rather than being caught by whoever happens to read the diff. The row ids are
what a failure prints.

**No test opens a window.** Refused by `test-opens-no-window`, which reads the
test sources for a driver that opens a window or drives a real browser. A window
needs a session with a desktop in it, which a runner does not have and a person
running the suite in the background does not want.

**No test needs a display server.** Refused by `test-needs-no-display-server`,
which reads for a display server or the variable that points at one. A test that
reaches for a display passes on the machine it was written on and fails on every
runner, and the failure reads as a broken test rather than as a broken
assumption.

**No test binds a listening socket on anything but loopback.** Refused by
`test-binds-only-loopback`, which reads the address a listen call was given and
permits `127.0.0.1`, `localhost` and `::1`. This is the one that bites in
practice. Binding a machine's own interface address rather than loopback is what
makes a desktop firewall ask an administrator for permission, and the dialog is
answered per executable path, so answering it settles nothing for the next build
directory. An address with no host at all is refused as loudly as a written-out
one, because it means every interface on the machine and it is the spelling
somebody writes without meaning to.

**No test writes into a certificate store.** Refused by
`test-writes-no-certificate-store`, which reads for a tool that installs or
trusts a certificate. A certificate store is machine-wide state, so a test that
writes one changes the machine for everything else on it and asks for elevation
on the way.

**No test shells out to a tool that asks for elevation.** Refused by
`test-asks-for-no-elevation`. Elevation does not fail as a red test. On a machine
with a person sitting at it, it is a consent prompt that takes the screen away
from whoever was working, and a test that does that once is a test people learn
to skip.

**No test needs a package that is not either in the toolchain or fetched by the
build itself.** Refused by `test-needs-nothing-outside-the-toolchain`, which
refuses an import in a test source that is neither standard library nor inside
this module. A package somebody has to install first is a setup step, and a suite
with a setup step is a suite that is run rarely.

## What the rows read, and what that does not reach

The rows read the bytes of the test sources. That is what makes them cheap, and
it is the whole of what they prove: a test source that spells one of these shapes
out is refused, and a test that arrives at the same behaviour through a value the
row cannot follow is not. An address built at run time from a variable, a command
name assembled from parts, a package pulled in by a dependency rather than by an
import line here: none of those is refused by anything in this repository today.

So the rows are a floor rather than a guarantee. They hold the shapes that
actually arrive, they will not catch one nobody has written yet, and the review is
where the rest is caught.

## What to do with a test that needs one of these

It is skipped, and the skip is disclosed where the work is argued rather than
worked around. A test that needs a browser goes to the set the gate runs
deliberately under the published name for what it needs. A test that needs
elevation does not get written: there is nothing in this repository that a person
has to be an administrator to check.

## Running it

    go run . invariants

The rows are decided by that verb, which is also a leg of `go run . ci`, so the
gate covers them. The workflow that reports the result on a pull request runs the
same verb.
