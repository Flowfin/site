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

    gh api repos/Flowfin/site/commits/d357381ab0837b347a1dfc1402eb0244f10274ed/check-runs \
      --jq '[.check_runs[].name] | sort | .[]'
    Analyze (go)
    Audit workflows (zizmor)
    CodeQL
    DCO sign-off
    Deterministic PR-hygiene checks
    Enforce greppable invariants
    Package (site) / Build bundle
    Package (site) / Generate SBOM
    Reject Trojan Source Unicode
    Reject Trojan Source Unicode
    Reproducible build
    build
    dependency-review
    prettier
    zizmor

Run 2026-08-10. Three more run where no pull request can see them, on a timer or
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

It refuses a tree that `go run . ci` refuses. How many legs that is, and in which
order, is printed by the verb rather than written here:

    go run . ci | head -1
    gate: 6 legs, in order: format, vet, test, build, links, invariants

Run 2026-08-10.

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

The other legs have no recorded refusal of their own under this check, and this
is where each of them is named rather than covered by the one above.

The **format** leg and the **vet** leg are owed. What would record them is a
pull request carrying one badly formatted Go file, and one carrying a construct
the toolchain's own analysis refuses.

The **test** leg refused on the server on `0862b92`, carrying the fuzz target
over the roster parser and a parser that accepted three of its seeds:

    gh run view 31411818662 --repo Flowfin/site --log-failed
    gate: 6 legs, in order: format, vet, test, build, links, invariants
      format: ok, 19 file(s)
      vet: ok
      test: FAILED
        go test refused:
            --- FAIL: FuzzParse (0.00s)
                --- FAIL: FuzzParse/seed#2 (0.00s)
                    fuzz_test.go:122: row 1 was accepted with no identifier, and the identifier is what a page address and the per-plugin prose are keyed by
                --- FAIL: FuzzParse/seed#3 (0.00s)
                    fuzz_test.go:133: row 1 was accepted declaring the repository "", and the schema writes one as owner/name
                --- FAIL: FuzzParse/seed#6 (0.00s)
                    fuzz_test.go:135: row 1 was accepted declaring the repository "o/sub/jellyfin-plugin-a", which carries a path rather than the name after an owner
            FAIL	github.com/Flowfin/site/internal/roster	0.005s
    3 of 6 legs ran. Not reached: build, links, invariants.

Each message names the row, what the parser accepted and what the schema says
instead. Did not refuse: run 31412126264, on `4227218`, the same branch one
commit later, which carries the target and a parser that refuses those rows.

An earlier refusal of this leg is recorded off the server in the body of #113, a
deleted space in the join that puts a wrapped paragraph back together. That one
is a run on a working tree and stays weaker than the run above.

### The fuzz target over the rendering path

The other target in the same leg asks whether a page built from arbitrary prose
carries any element or attribute the template did not write. Its pair is recorded
off the server, on a working tree, which is weaker than every entry above and is
said here rather than left to be noticed.

The edit is one import, `html/template` exchanged for `text/template` in
`internal/site/site.go` and nothing else, which is the smallest change that lets
a value in the data reach a reader as markup and is the edit somebody makes on
purpose when a template has to emit a fragment:

    go test ./internal/site -run FuzzBuildEscapesPageProse
    --- FAIL: FuzzBuildEscapesPageProse (0.29s)
        --- FAIL: FuzzBuildEscapesPageProse/seed#1 (0.03s)
            fuzz_test.go:125: the page carries 18 element(s) and the same shape carries 16, so a value in the prose became markup
                from: "Title\n\n<script>alert(1)</script>\n"
                page: <p><script>alert(1)</script></p>
        --- FAIL: FuzzBuildEscapesPageProse/seed#2 (0.03s)
            fuzz_test.go:125: the page carries 17 element(s) and the same shape carries 16, so a value in the prose became markup
                from: "Title\n\n<img src=x onerror=alert(1)>\n"

Run 2026-08-10 against the tree at `4227218` with that import changed. The page
line is the one element of the produced page that matters here and the rest of it
is elided; the command reproduces the whole. What would retire this entry is a
pull request carrying the same edit, the way every entry above was recorded.

Did not refuse: run 31412126264, on `4227218`, which carries the target and the
escaping call it exists to watch.

Neither target has found anything the seeds did not already carry. Both were run
past their corpora, and a run that found nothing is recorded because the count is
what says how far past:

    go test ./internal/roster -run='^$' -fuzz='^FuzzParse$' -fuzztime=60s
    fuzz: elapsed: 1m0s, execs: 28031831 (413817/sec), new interesting: 32 (total: 436)
    PASS
    go test ./internal/site -run='^$' -fuzz='^FuzzBuildEscapesPageProse$' -fuzztime=180s
    fuzz: elapsed: 3m0s, execs: 4484 (6/sec), new interesting: 62 (total: 76)
    PASS

Run 2026-08-10 at `4227218`. Neither wrote an input under `testdata/`, which is
where a crashing input lands and is the thing that says a run found one. The two
rates are three orders of magnitude apart because the rendering target writes a
tree and builds it once per execution, so its number is a number of builds.

The **links** leg is the entry that follows this paragraph, because it has no
check name of its own and its refusal is recorded under this one.

The **invariants** leg has its own check name and is the entry below.

### The links leg

It refuses a reference in the produced output that resolves to nothing the build
wrote, and one that points at a place inside a page that does not exist.

Refused, on `a17f184`, with one letter dropped from a page address in the
template and a fragment that was never on that page, which is what a rename
leaves behind everywhere it was referenced:

    gh run view 31346218703 --repo Flowfin/site --log-failed
    gate: 6 legs, in order: format, vet, test, build, links, invariants
      links: FAILED
        links: 1 reference(s) resolve to nothing the build wrote:
            links: 1 reference(s) over 1 page(s), against 1 produced file(s)
              dist/index.html: line 11 href="/design-sytem.html#tokens" resolves to dist/design-sytem.html, which the build did not write
    5 of 6 legs ran. Not reached: invariants.
    gate: the links leg refused this tree

The message names the page, the line, the reference as written and the path it
resolved to. Only the first half of that near miss was reached: a reference whose
page does not exist is reported and its fragment is not read, because there is
nothing to read it in. The fragment half has its pair in the suite instead:

    go test ./internal/link -run TestADeadFragmentIsRefusedAndNamed -count=1

Did not refuse: run 31346155709, on `736863d`, the same branch one commit
earlier, which carries the leg and a page with no reference on it. What that
neighbour does not cover is a page whose references resolve, and the suite is
where that case is, over a tree it builds itself.

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

## Package (site) / Generate SBOM

It refuses a tree whose bill of materials cannot be produced from the files that
decide it, which is the half that keeps a document with a hole in it from
reading like a document about a tree with nothing to list.

Refused, on `cdff80e`, with the digest taken off the base image and the tag left
in place, which is the shape a Dockerfile takes the moment somebody updates the
image by editing the version they can read:

    gh run view 31345081068 --repo Flowfin/site --log-failed
    Run go run . sbom > sbom.cdx.json
    Dockerfile carries no base image pinned by digest, and a tag on its own does not say which bytes were pulled
    ##[error]Process completed with exit code 1.

The message names the file and says what a tag on its own does not answer.
`build` went red on the same commit, because the suite reads this repository's
own sources rather than only a fixture, and that overlap is the suite doing what
it is for rather than the two names failing to separate.

Did not refuse: run 31345020916, on `d357381`, where the job ran rather than
skipped and said what it had covered:

    sbom: CycloneDX 1.6, 3 component(s) for github.com/Flowfin/site
      read go.mod, Dockerfile and .github/workflows/prettier.yml
      the module graph is empty, so the document lists the toolchain, the base image and the formatter and no library

The other five sources it fails closed on have their pair in the suite rather
than in a run on the server:

    go test ./internal/bom -run TestASourceThatCannotBeReadStopsTheDocument -count=1

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

## Static analysis (semgrep)

It refuses source carrying one of the shapes the rule set in
`tools/semgrep/rules.yml` names. The pair below covers every rule in that set
rather than one of them, because a set is only as proven as its least exercised
rule, and a rule that matches nothing looks exactly like a tree with nothing in
it to match.

Refused, on `726a020`, which carried one instance of each shape in a file
nothing called. Which rules refused is quoted rather than the paragraph beside
each finding, since that paragraph is the rule's own text and is in the rule
file:

    gh run view 31475636799 --repo Flowfin/site --log-failed \
      | grep -o 'tools\.semgrep\.[a-z-]*' | sort | uniq -c
          1 tools.semgrep.file-read-with-a-path-the-call-site-cannot-vouch-for
          1 tools.semgrep.page-rendered-by-an-engine-that-does-not-escape
          1 tools.semgrep.roster-data-reaches-a-filesystem-path
          1 tools.semgrep.template-escaping-bypassed
          2 tools.semgrep.the-build-starts-a-process

    gh run view 31475636799 --repo Flowfin/site --log-failed \
      | grep -o 'Ran 5 rules on [0-9]* files: [0-9]* findings.'
    Ran 5 rules on 11 files: 6 findings.

Five rules and six findings, because the process rule matched the import and the
call as two. Every finding named the file and the line it matched.

Did not refuse: run 31476053436 on `fb9cc5d`, which is the same rules over the
same tree with that file taken out again, and run 31475222126 on `ce96a00`,
which is the rules arriving with nothing to refuse.

The findings reach the code scanning tab as well as the run, and the count there
follows the tree rather than staying at whatever the first upload said:

    gh api 'repos/Flowfin/site/code-scanning/analyses?per_page=30' \
      --jq '.[] | select(.tool.name == "Semgrep OSS") | "\(.results_count)\t\(.created_at)"'
    0	2026-08-11T09:04:42Z
    6	2026-08-11T08:59:25Z
    6	2026-08-11T08:55:48Z
    0	2026-08-11T08:53:44Z

All run 2026-08-11.

What the pair does not show. Each rule was tripped by its construct written
plainly, which is the case a reader spots too, so the pair says the rules bite
and not that they are hard to walk past. The rule about a roster value reaching
a path is where that gap is widest: it was tripped by a value concatenated
inside one function, and a value that arrives through a second function is not
something the engine follows on the analysis this repository runs.

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

## Pins

It refuses a tree whose pinned versions have fallen behind the registries that
publish them, and a run that could not read a registry at all, which it reports
as unresolved rather than as current.

Nothing in a pull request reports it. It reads three registries over the
network, so its verdict moves when somebody else publishes rather than when the
tree changes, and it runs weekly or on request instead of as a leg of the gate.
That is the reason it is in this document rather than despite it: a run nobody
watches is where an unproven guard survives longest.

Refused: run 31358584874, on `985b8a3`, asked for by hand.

    gh run view 31358584874 --repo Flowfin/site --log-failed
    pins: 3 declared in pins.json
      prettier: current at 3.9.6, npm says the same
      zizmor: BEHIND, pinned 1.26.1, pypi says 1.29.0
    pins: 1 pin(s) are behind their upstream
      golang-image: current at sha256:6c5605ab3a9a9fb3c4eafe5b3d63cdbf3881caf113262b67862547b54a9db599, docker-hub says the same
    3 pin(s) declared, 3 compared, 1 behind, 0 unresolved.
    exit status 1

Two of the three pins were current in that same run, so what it refused for is
the one that had moved rather than a run that refuses whatever it is given.

Did not refuse: owed. It is a run of this workflow over a tree where every pin
is current, and that tree does not exist while one of them is behind. Taking the
release the run named is #132, and the pair closes on the first run after it.

The states this workflow keeps apart are not all reachable from a run id. A
registry that cannot be read, and a run where none of them can, are proved by
the suite instead, which drives the comparison with an answer of each kind
rather than waiting for a registry to be down:

    go test ./internal/pins -run 'Unread|ComparedNothing' -v
    === RUN   TestRunReportsAnUnreadableUpstreamAsUnresolvedRatherThanCurrent
    --- PASS: TestRunReportsAnUnreadableUpstreamAsUnresolvedRatherThanCurrent (0.02s)
    === RUN   TestRunRefusesToPassARunThatComparedNothing
    --- PASS: TestRunRefusesToPassARunThatComparedNothing (0.01s)
    PASS
    ok  	github.com/Flowfin/site/internal/pins	1.346s

Run 2026-08-10. That is a different kind of evidence from a run id and it is
named as such: it proves the code decides those states, not that the workflow
has ever met one.

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
