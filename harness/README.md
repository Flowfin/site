# The sets that are run deliberately, and never by the gate

Some things about a published site cannot be decided on a build machine. Whether
the name serves the bytes this build produced, whether the certificate behind it
is the right one and still valid, whether a request over plain http is answered
with a redirect rather than served: none of those is a property of this tree, and
a check that reported on them from inside the gate would be reporting on
something it had not looked at.

They go here instead, in a directory per set, and the gate verb says on every run
that the set was not asked for and what asking would cost. A partial run that
said nothing about what it left out would read as a full one.

## The names

The names are not coined here. The project publishes a vocabulary for exactly
this, one name per thing a run needs rather than one per thing it tests, so that
somebody reading a list of jobs can tell which would go red because a service
somebody else operates was down:

    gh api repos/Flowfin/hub/contents/decisions/headless-and-unelevated.md \
      --jq '.content' | base64 -d | grep -nE '^`needs-'
    50:`needs-network` for anything that makes a request off the runner. Fetching the
    55:`needs-browser` for anything that renders the site. Measuring the numbers in the
    59:`needs-jellyfin` for anything that talks to a server. Adding the repository

Run 2026-08-11.

`needs-network` is the one directory here today. Where this repository departs
from the rule those names come from, and what that costs, is
[decisions/0012-the-browser-in-the-gate.md](../decisions/0012-the-browser-in-the-gate.md)
rather than an argument restated here.

## What a run produces

A record, written under the set's own directory, carrying the date, the
conditions it was taken under, every reading with the request that produced it,
and every reading it did not take with the reason. A number from a real network
is a measurement of one machine at one moment, so reporting it as a property of
the site would be a claim dressed as evidence, and the record is shaped so that
it cannot be read as one.

Nothing in a record is called a verification of something the run did not verify.
A reading that could not be taken says so and never appears as a result.

## Running it

    go run ./harness/needs-network

It writes into `harness/needs-network/record/` and prints the path it wrote. Read
`go run ./harness/needs-network -h` for the name it reads and the directory it
writes to.

It is not a leg of `go run . ci` and it is not part of `go test ./...`. Its code
carries unit tests that reach loopback and nothing else, so the suite the gate
runs stays inside the headless rule while the program those tests cover
deliberately leaves it.
