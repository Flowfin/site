# site

The public site is served today out of the hub tree, which mixes a page written for people with a manifest written for machines. Every change to a paragraph touches the tree that serves machine-readable data, and the site cannot be moved without moving the manifest with it. Static only, with no account, no form and nothing that can leak. The plugin list is derived rather than written by hand, because a hand-written list is wrong the day a plugin ships. A project whose promise is that nothing waits cannot open with a slow page, so the site states its own speed budget as numbers.

Planning happens on the issue tracker first. Every decision that shapes
the architecture is written down there with its reasons before the code
that depends on it exists.

## What this repository produces

A static site, and the generator that renders it. One command produces the
site:

    go run . build

It renders into `dist/` and prints every file it wrote. That directory is not
tracked, so what is served is always something a run just produced rather than
a copy somebody committed.

One command checks the tree:

    go run . ci

Its legs run in order and the run stops at the first failure, so it prints
which legs ran and names the ones it never reached. What the legs are is what
that run prints. This file does not list them, because a list here drifts
against the thing that decides them and a reader who trusts the list is
trusting the day it was written.

Other verbs exist for rules that the server decides and somebody may want to
decide on their own machine first. Running the generator with no argument
prints what they are.

The container is the route for producing the site without installing a
toolchain, and [CONTRIBUTING.md](CONTRIBUTING.md) is where both routes are
written out.

## Where the rest is written down

[CONTRIBUTING.md](CONTRIBUTING.md) is the contributor guide: what to run
before pushing, what shape a change takes, and what refuses one.

[GOVERNANCE.md](GOVERNANCE.md) is who decides, who holds access and what
happens to a change that arrives from outside.

[SECURITY.md](SECURITY.md) is how to report a security problem and what
follows a report.

[NOTICE.md](NOTICE.md) is the intended-use notice.

[decisions/](decisions/) holds one file per decision that shaped the
architecture, with the reasons that were current when it was taken.

This repository has no code of conduct. Issue #62 is where one is added, and
until it lands there is nothing here to link.

There is no licence file either, so this repository states no SPDX identifier
and the licence is not chosen. Nothing follows from that absence except the
default, which is that nobody may copy the pages, the generator or the prose.
Entry 1 of issue #7 is the choice, and issue #18 is what lands the file and
the identifier once that answer comes back.
