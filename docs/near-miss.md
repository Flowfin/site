# What each check has refused, and what it did not

A guard that has never refused anything is a guard nobody has tested. Worse, a
guard that refuses for the wrong reason passes on the day it matters and nobody
finds out. So each check here carries a pair: the smallest change that should
trip it, with the run where it tripped and the message it produced, and a change
that should not trip it, with the run where it did not.

Only the pair distinguishes a guard that bites from one that refuses everything.

A check with no pair is named below as owed rather than left out. An entry that
said nothing would read as a check that had been proven.

## The set this covers, and where it comes from

The names below drift against the checks that actually report. What reports is
read from a commit rather than remembered:

    gh api repos/Flowfin/site/commits/726e521c289fb1d92f8368e8831eb77ae3cfa962/check-runs \
      --jq '[.check_runs[].name] | sort | .[]'
    Analyze (go)
    Audit workflows (zizmor)
    CodeQL
    DCO sign-off
    Deterministic PR-hygiene checks
    Enforce greppable invariants
    Package (site) / Build bundle
    Reject Trojan Source Unicode
    Reject Trojan Source Unicode
    Reproducible build
    build
    dependency-review
    prettier
    zizmor

Run 2026-08-09. Three more run where no pull request can see them, on a timer or
on request, and they are covered here too because a run nobody watches is exactly
where an unproven guard survives longest:

    gh api repos/Flowfin/site/actions/workflows \
      --jq '[.workflows[] | select(.state=="active") | .name] | sort | .[]'

That command is the authority for what exists. Where it names something this
document does not, the entry is missing rather than the check being exempt.

Every neighbour below is the same one unless it says otherwise: the head at
`62be147041c7ba161a3be5f4d280412f6445abec`, which is a pull request that changed
a document, a table of rules and a suite, and tripped nothing.

One thing about the quoted output. Where a verb writes to both streams, the
runner's log interleaves them, so a line can appear above the line it follows.
The blocks below are in the order the verb writes them, not the order the log
shows, and nothing else about them is edited.

## build

It refuses a tree that `go run . ci` refuses, which is five legs.

Refused, on `fcb9fb5`:

    gh run view 31305625532 --repo Flowfin/site --log-failed
    gate: 4 legs, in order: format, vet, test, build
      format: ok, 3 file(s)
      vet: ok
      test: no test file in the tree, so this leg examined nothing
      build: FAILED
        the build refused:
            rendering the page: template: page.html.tmpl:6:14: executing "page.html.tmpl" at <.Titl>: can't evaluate field Titl in type site.page
    4 of 4 legs ran. The failure is in the last one.

One character removed from a field name in the template. The message names the
file, the line and the field.

Did not refuse: run 31318125893.

Four of the five legs have no recorded refusal of their own under this check, and
this is where they are named rather than covered by the one above.

The **format** leg and the **vet** leg are owed. What would record them is a
pull request carrying one badly formatted Go file, and one carrying a construct
the toolchain's own analysis refuses.

The **test** leg has a refusal recorded off the server rather than on it, in the
body of #113: a deleted space in the join that puts a wrapped paragraph back
together, quoted with the gate output it produced. That is a run on a working
tree, so it is weaker than the entries above, and what would retire it is a pull
request carrying that edit.

The **invariants** leg has its own check name and is the entry below.

## Enforce greppable invariants

It refuses a tree that breaks one of the rows in `internal/invariant`. Every one
of them has a pair. How many rows there are is printed by the run rather than
written here, because a number in this document drifts against the table it
describes:

    go run . invariants | tail -1
    12 rule(s) decided, 2 owed and not decided.

Run 2026-08-10 at `553810a`.

Refused, on `ccb2d5e`, with the language attribute deleted from the page
template:

    gh run view 31316531489 --repo Flowfin/site --log-failed
      page-declares-its-language: REFUSED, 1 violation(s)
        it refuses a produced page with no html element, or one whose lang attribute is missing or empty
        because a page with no language is read aloud in whichever one the reader's software guessed, and the guess is wrong for exactly the readers who depend on it
        dist/index.html: the html element carries no lang attribute
    invariants: 1 rule(s) refused this tree

The other four page and tree rows refused on `f5dc82f`, in run 31316714180: an
emptied title element, one script element with a source, an unfinished marker in
the prose that reached the output, and a tracked file carrying a produced-by
marker. Each names the row, what it refuses, why, and the file and line.

The loopback row refused on `40439b6`, in run 31317948347:

    internal/site/nonloopback_near_miss_test.go: line 16 binds "0.0.0.0:8080", which is not loopback

The origin row refused on `a162b9f`, with one stylesheet pulled from somebody
else's domain in the page template, which is the shape a page picks up the first
time anybody reaches for a font:

    gh run view 31343903523 --repo Flowfin/site --log-failed
      output-references-no-domain-outside-the-allowlist: REFUSED, 1 violation(s)
        it refuses a produced file fetching a stylesheet, a font, an image, a script or anything else from a host that is not on the allowlist, while leaving a link a reader clicks alone
        because a subresource served from somebody else's domain is a round trip the reader pays for and a record of who read what, handed to a party the reader did not choose, and a promise that this does not happen cannot rest on nobody having added a font or a badge yet
        dist/index.html: line 7 fetches https://fonts.example.invalid/inter.css, whose host fonts.example.invalid is not on the allowlist

The message names the file, the line, the address and the host. Did not refuse:
run 31343827710, on `d17f5cd`, the same branch one commit earlier, which carries
the row and no reference for it to find.

Did not refuse: run 31318125852, and in each of the runs above every row other
than the one under test stayed green, which is a neighbour per row rather than
one for the check.

The five remaining headless rows have their pair in the suite rather than in a
run on the server:

    go test ./internal/invariant -run TestEveryRowRefusesItsOwnViolationAndPassesTheNeighbour -count=1

Every row gets a violation of its own and a neighbour of the population it reads,
and the test fails if the table grows a row that no violation is written for. A
suite is a weaker record than a red check for the purpose this document serves,
and it is a stronger one for keeping the pair true as the table changes, so both
are named.

## Reproducible build

It refuses a source whose two builds do not produce the same bytes.

Refused, on `5fb0701`, with one clock read in the generator appended to every
page it renders:

    gh run view 31320295732 --repo Flowfin/site --log-failed
    reproduce: two builds of ., compared byte for byte
      1 of 1 file(s) differ between the two builds
        dist/index.html: 811 bytes against 811, and they are not the same bytes, first at line 16
    The usual causes are a time read from the clock, a map walked in whatever order it came out in, and an absolute path from the build machine.
    reproduce: 1 of 1 produced file(s) differ between two builds of one source

The message names the file, the line where the two copies part, and the three
causes this defect class is almost always one of.

Did not refuse: run 31320444172, on `726e521`, the same branch with the clock
read taken out.

`build` went red on the same commit, because the suite's own neighbour case
builds the real generator twice. That is the two names overlapping on one defect
rather than the separation failing, and what the separate name buys is the case
they do not overlap on: a difference that appears only in the real output.

## Package (site) / Build bundle

It refuses a change whose output cannot be packed into the bundle an operator
downloads.

Owed. Did not refuse: run 31318926094, on `fcff78a`, whose second attempt
produced a byte-identical archive to its first. What would record a refusal is a
change that makes the packing step fail, and the plausible version is an output
directory the build did not write to.

## prettier

It refuses a prose file whose shape disagrees with the formatter.

Refused, on `094c58d`:

    gh run view 31307134885 --repo Flowfin/site --log-failed
    Checking formatting...
    [warn] docs/near-miss-formatting.md
    [warn] Code style issues found in the above file. Run Prettier with --write to fix.

The message names the file.

Did not refuse: run 31318125874, which changed three Markdown files.

## Deterministic PR-hygiene checks

It refuses a commit message with no bracketed issue reference, and one carrying a
character outside the allowlist.

Refused, on `5de0690`:

    gh run view 31306570396 --repo Flowfin/site --log-failed
    hygiene: 3 non-merge commit(s) in 041a4240fe2e..5de0690bc59f, origin internal
      97d533ec5c73: subject carries its reference
      670629a48198: FAILED: the subject carries no bracketed issue reference: "A subject with no bracketed reference"
      5de0690bc59f: subject carries its reference
      5de0690bc59f: FAILED: the message carries 1 code point(s) outside printable ASCII: U+200B on line 3
    hygiene: 2 refusal(s) across 3 commit(s)

Both legs in one run, and the message names the commit, the rule and, for the
character leg, the code point and the line. The commit between them passed, which
is the neighbour inside the same run.

Did not refuse: run 31318125887.

## Audit workflows (zizmor)

It refuses a workflow file carrying an actionable security finding.

Refused, on `0bc55bc`:

    gh run view 31306769050 --repo Flowfin/site --log-failed
    warning[dependabot-cooldown]: insufficient cooldown in Dependabot updates
      --> ./.github/dependabot.yml:39:5
    6 findings (4 suppressed, 2 safe fixes): 0 informational, 0 low, 2 medium, 0 high

The message names the file, the line and the audit.

Did not refuse: run 31318125853.

## Required check names

It compares the names a rule on the default branch requires against the names
that report, and it refuses a run that compared nothing.

Refused, on `main`:

    gh run view 31301601892 --repo Flowfin/site --log-failed
    The rules on Flowfin/site@main require no status check, so there is no name for a reported one to satisfy and nothing was compared.
    What ends this state is a required status check on that branch. Until then this run has nothing to say about whether the two lists agree, and it says that rather than passing.

Every run of it so far has concluded the same way:

    gh api 'repos/Flowfin/site/actions/workflows/required-check-names.yml/runs?per_page=10' \
      --jq '[.workflow_runs[].conclusion] | group_by(.) | map({(.[0]): length}) | add'
    {"failure":2}

Both run 2026-08-09. So the refusal is recorded and the neighbour is owed: this
check has never passed, and it cannot until a rule on the branch requires a
status check, which is #47. Until then nothing here shows that it can tell
agreement from disagreement rather than only saying it compared nothing.

## Watchdog near miss, and the watchdog it exists for

The near-miss workflow fails on request and does nothing else. It is the only
way to put a failed run on the default branch for the watchdog to find without
breaking the branch or waiting for the incident.

Refused: run 31293272796, concluded `failure`, and run 31293188801 before it.

Did not refuse: run 31293350600, the same workflow asked for the passing outcome,
concluded `success`.

The watchdog reads that pair rather than refusing anything itself. It opened an
entry naming the failed run, and closed it once the newest completed run of that
workflow succeeded:

    gh issue view 101 --repo Flowfin/site --json state,title --jq '{state,title}'
    {"state":"CLOSED","title":"Default branch: Watchdog near miss did not succeed"}

Run 2026-08-09. That open and that close are the pair for the watchdog: the
smallest thing that should be reported, and the same thing once it should not be.

## Analyze (go), and CodeQL

Two names from one workflow, answering different questions.

`Analyze (go)` says whether the analysis ran. `CodeQL` says whether the change
introduced alerts, and it is the only one of the pair that can refuse a finding.

`CodeQL` refused on `9f88de8`, which carried a shell command built from a query
parameter in a package nothing calls:

    gh api repos/Flowfin/site/commits/9f88de84603f5d2b651687b30b7f213330c1221a/check-runs \
      --jq '.check_runs[] | select(.name=="CodeQL" or .name=="Analyze (go)") | {name, conclusion, title:.output.title}'
    {"conclusion":"failure","name":"CodeQL","title":"1 new alert including 1 critical severity security vulnerability"}
    {"conclusion":"success","name":"Analyze (go)","title":null}

Run 2026-08-09. Neighbours: runs 31318125871 and check run 93256699390.

`Analyze (go)` has no recorded refusal, and the line above is why: a critical
finding sat in the tree and it passed. It refuses a tree the extractor cannot
build rather than a tree with a finding in it, and that refusal is owed. #12
holds the consequence for which name a rule should require.

## DCO sign-off

It refuses a branch carrying a commit with no sign-off trailer, or one whose
trailer does not match its author.

Owed. Did not refuse: run 31318125875. What would record the refusal is a pull
request carrying one commit made without `-s`, which is the mistake the check
exists for and is one flag away from every commit that lands here.

## Reject Trojan Source Unicode

It refuses a tracked file carrying a bidirectional or invisible code point.

Owed. Did not refuse: run 31318125895. What would record the refusal is a pull
request carrying one such code point in a tracked file. The neighbouring check
over commit messages has its refusal recorded above, and the two are different
subjects: that one reads git metadata, this one reads the tree.

## dependency-review

It refuses a change introducing a dependency with a known advisory.

Owed, and it is the one entry where the check could not judge rather than judging
and passing. `docs/parity.md` records a run of it reporting that the dependency
graph is not enabled on this repository, which is a repository setting rather
than a property of a change. What would record a refusal is a change adding a
dependency with an advisory against it, and there is nothing in the module graph
to add one to today.

## Scorecard supply-chain security

It reports and does not refuse. There is no pair to record, because there is no
refusal it can make: it uploads a score to the code scanning tab and its
conclusion does not depend on the change. Saying so is the entry.

## What this document is not

It says that each check named above refused the change beside it and passed the
other one. It does not say the checks cover what they are meant to cover, that
the messages are the best messages, or that a check with a pair recorded once
still bites after the change that comes next. The suite over the invariant table
is the only thing here that re-derives its pair on every run; everything else is
a run id, and a run id is a fact about a day.
