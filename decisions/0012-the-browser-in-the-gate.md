# 0012. The browser in the gate, and where this repository departs from the published rule

The project already publishes a rule about what a merge gate may run, and the plan
on this board disagrees with one clause of it. Two rules and no argument between
them is worse than either one alone, because a reader who finds both has no way to
tell which was meant.

## What was measured

The clause this repository departs from:

    gh api repos/Flowfin/hub/contents/decisions/headless-and-unelevated.md \
      --jq '.content' | base64 -d | grep -n -A1 'needs a compositor'
    15:A test that opens a window, needs a compositor, or renders a page in a real
    16-browser.

The names the published rule gives the harnesses that sit outside the gate:

    gh api repos/Flowfin/hub/contents/decisions/headless-and-unelevated.md \
      --jq '.content' | base64 -d | grep -nE '^`needs-'
    50:`needs-network` for anything that makes a request off the runner. Fetching the
    55:`needs-browser` for anything that renders the site. Measuring the numbers in the
    59:`needs-jellyfin` for anything that talks to a server. Adding the repository

Both run 2026-08-08.

## The decision

The browser stays inside the gate in this repository. The rest of the published
rule is kept whole. Anything this repository puts outside the gate carries the
published harness names rather than a vocabulary of its own.

## Why the departure

For something that ships into somebody else's server, a rendered page is
incidental to what is being shipped, and a gate that drags a browser in is paying
a large dependency for a small property.

For a site the rendered page is the artefact. Layout shift, contrast against the
tokens that were actually applied, reflow at 320 pixels, what the browser is asked
to fetch after the page has loaded and whether the declared policy is violated are
all properties of a render, and none of them can be decided from the bytes the
build wrote. Refusing the browser here would move this repository's principal
properties into a harness that nothing requires before a merge, which is the
failure both rules exist to prevent.

## The cost this accepts

The gate depends on a program that is not part of the toolchain, which is exactly
what the published rule refuses, and the reason it refuses it is real: a fresh
machine has no browser, and a gate that silently skips what it cannot run is a
gate that reports on less than it appears to. That is paid for with a pin by
version and by checksum, with a run that prints that the browser-backed set was
not asked for and what asking would cost, and with a browser that is present and
will not start counting as a failure rather than as a skip.

## What is not departed from

No test opens a window and none needs a compositor: the browser runs headless.
None asks for elevation, registers a service, binds a privileged port or writes
outside its own temporary directory. None reaches the public network, which is why
release data is recorded rather than fetched during the gate and why the freshness
comparisons run on a schedule instead of as gate legs.

## When this is worth revisiting

A rendered property that a static reading of the built output can decide, which
removes the reason for the departure for that one check rather than for the set.
