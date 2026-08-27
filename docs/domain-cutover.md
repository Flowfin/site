# Moving the domain to the site this repository produces

The sequence for moving `flowfin.dev` from the origin that serves it today to
the one this repository builds, written before it is performed rather than
improvised on the day.

It does not decide whether the domain moves, or when. That was entry 3 of #7,
and it has an answer:

    gh issue view 7 --repo Flowfin/site --json comments \
      --jq '.comments[]|select(.createdAt=="2026-08-24T18:18:14Z")|.body' \
      | sed -n '1p'
    Answer to entry 3, decided 2026-08-24: the new site goes live on a subdomain first, is proven there, and the cutover happens against a standing target - the catalogue address answers throughout, with no resolution gap. Two live sites for a bounded while is the price, paid knowingly.

Run 2026-08-27. What this covers is the move itself, and it is written now
because the expensive part is the ordering rather than the typing.

The sequence below was written before that answer and does not carry it. This
paragraph used to say the sequence is the same on any answer that ends with this
repository as the origin, and the answer that arrived is one it is not the same
on. Where the two disagree is set out under
[Where this runbook departs from the answer](#where-this-runbook-departs-from-the-answer)
and is not repaired there, because what would close the gap is a decision
nobody has taken.

Nothing in this repository performs any of it. Every step is a settings action
or a reading of what a settings action did, and the settings belong to whoever
holds the account.

## The state before the change

Recorded here so that rolling back means restoring something rather than
guessing at it. Every command below was run 2026-08-11 and every output is what
it printed.

Where the origin sits, and how it is serving:

    gh api repos/Flowfin/hub/pages \
      --jq '{cname,source,status,https_certificate:.https_certificate.state,https_enforced}'
    {"cname":"flowfin.dev","https_certificate":"approved","https_enforced":true,"source":{"branch":"main","path":"/docs"},"status":"built"}

This repository publishes nothing:

    gh api repos/Flowfin/site/pages
    gh: Not Found (HTTP 404)

What the origin serves, which is four files:

    gh api 'repos/Flowfin/hub/git/trees/HEAD?recursive=1' \
      --jq '[.tree[].path|select(startswith("docs/"))]'
    ["docs/CNAME","docs/design-system.html","docs/design-tokens.json","docs/index.html"]

The name in DNS. The apex carries the host's four addresses and `www` is a name
of its own pointing at the account's site:

    curl -sS -H 'accept: application/dns-json' \
      'https://dns.google/resolve?name=flowfin.dev&type=A' \
      | grep -o '"data":"[^"]*"' | sort
    "data":"185.199.108.153"
    "data":"185.199.109.153"
    "data":"185.199.110.153"
    "data":"185.199.111.153"
    curl -sS -H 'accept: application/dns-json' \
      'https://dns.google/resolve?name=www.flowfin.dev&type=CNAME' \
      | grep -o '"data":"[^"]*"'
    "data":"flowfin.github.io."

The answers arrive in a different order on each query, so the sort is what makes
two runs comparable rather than a preference about reading them.

The ownership check on the name resolves. Its answer is a token and is
deliberately not pasted, because a document holding it gains nothing a reader
cannot fetch and it is the kind of string that outlives the reason it was
written down:

    curl -sS -H 'accept: application/dns-json' \
      'https://dns.google/resolve?name=_github-pages-challenge-Flowfin.flowfin.dev&type=TXT' \
      | grep -o '"Status":[0-9]*'
    "Status":0

What answers, and with what:

    for p in / /index.html /design-system.html /design-tokens.json /manifest.json; do
      printf "%-40s " "https://flowfin.dev$p"
      curl -sS -o /dev/null -w "%{http_code}\n" "https://flowfin.dev$p"
    done
    https://flowfin.dev/                     200
    https://flowfin.dev/index.html           200
    https://flowfin.dev/design-system.html   200
    https://flowfin.dev/design-tokens.json   200
    https://flowfin.dev/manifest.json        404

One line of that reading has been overtaken, and it stays as it was printed
because this section exists so that rolling back restores a state rather than
guessing at one. The catalogue address answers now:

    curl -sS -o /dev/null -w '%{http_code}\n' https://flowfin.dev/manifest.json
    200

Run 2026-08-23. So a rollback restores an origin that serves that path, which is
what step 9 reads for, and the reading above is the state on 2026-08-11 rather
than the state to expect on the day.

A request over plain http is answered with a redirect rather than served:

    curl -sS -o /dev/null -w 'http->%{http_code} %{redirect_url}\n' http://flowfin.dev/
    http->301 https://flowfin.dev/

And nothing pins a reader to the secure address beyond that redirect:

    curl -sS -D - -o /dev/null https://flowfin.dev/ | grep -i -E 'HTTP/|strict-transport'
    HTTP/1.1 200 OK

One line came back and it is the status line. There is no transport security
header, so the redirect in the reading above is the whole of what carries a
reader from the plain address to the secure one. That matters for step 8 rather
than being a complaint about the current origin.

## What moves, and what does not

DNS does not move. The four addresses above are the host's rather than either
repository's, and they are the same addresses whichever repository publishes, so
no record is edited during the move and no propagation delay is part of the
window. The `www` name is not part of the move either.

The ownership check does not move. It resolves at the account rather than at a
repository, which is what its name says, and a name verified for the account
covers the repositories under it. That sentence is a claim about how the host
behaves, not something the reading above shows.

What moves is which published site claims the name. One custom domain is
attached to one published site at a time, so the old site releases the name
before the new one can take it, and that is where the window comes from. This is
also a claim about the host rather than a measurement, and measuring it costs
performing the move once.

## The sequence

### 1. Record the state as it is

Run the readings in the section above and keep the output. Rollback below is
written as restoring these values, and a rollback to a state nobody wrote down
is a repair somebody is inventing under time pressure.

Nothing to back out. This step only reads.

### 2. Publish this site at its own address, with no custom domain

Turn Pages on for `Flowfin/site` against whatever the publish work lands in #58,
and leave the custom domain field empty. The site then answers at the address
the host gives a project site, which is `https://flowfin.github.io/site/`.

What says it worked is that address serving the bytes this build produced:

    curl -sS -o /dev/null -w '%{http_code}\n' https://flowfin.github.io/site/

The live name is untouched for the whole of this step, so it can be repeated as
often as it takes.

Backing out is turning Pages off again for this repository. Nothing a reader can
see changes either way.

### 3. Compare what the two serve, path by path

Before the name moves, every address that answers today has to answer from the
new site at its own address, and the addresses this site introduces have to
answer there too. The list of both is in the section below.

What says it worked is each path in that list returning what it should from
`https://flowfin.github.io/site/`, read the same way as in step 1 rather than in
a browser, because a browser hides a redirect and caches what it was served.

Backing out is the same as step 2.

### 4. Carry the files this repository does not author

Two paths under the name are produced elsewhere. The catalogue file is held by
`decisions/0006-what-answers-at-the-catalogue-address.md` and refused as a
missing publish by #67. The token file is the second one and is covered by
neither: it answers today, this build does not produce it, and the section below
says what that leaves open.

What says it worked is the check #67 lands, which distinguishes a file that was
carried from one that could not be obtained and from an upstream with nothing
published yet.

Backing out is the same as step 2, and no reader has seen anything yet.

### 5. Release the name from the old origin

This is the step that opens the window.

Clear the custom domain on `Flowfin/hub` and delete `docs/CNAME` in the same
change. Doing one without the other puts the domain back: the file is what the
next build of that site declares the name from.

From here until step 6 completes, `flowfin.dev` resolves to the host's addresses
and no published site claims it, so what a reader gets is whatever the host
answers for an unclaimed name rather than anything either repository wrote.

How long that lasts is two settings actions apart, provided step 6 is performed
immediately and the new site is already built and checked by steps 2 to 4. What
is not controlled is how long the host takes to serve the name from the new site
once it is set, and there is no measurement of that here, because taking one
means performing the move. So the honest form of the number is minutes rather
than hours, held as a claim, and the runbook is ordered so that the window
contains no work: everything that can fail has already been done by step 4.

Backing out is setting the custom domain back on `Flowfin/hub` and restoring
`docs/CNAME`. The certificate there is already approved and enforcement is
already on, so the old origin comes back complete rather than in stages.

### 6. Claim the name at the new origin

Set `flowfin.dev` as the custom domain on `Flowfin/site`. This closes the window
for plain http and does not close it for https, which is step 7.

What says it worked:

    gh api repos/Flowfin/site/pages --jq '{cname,status,https_enforced}'
    curl -sS -o /dev/null -w '%{http_code}\n' http://flowfin.dev/

The second command deliberately asks over http. At this point https either fails
to negotiate or presents a certificate for another name, so a reading over https
here reports a failure that is expected rather than the state of the move.

Backing out is step 5's rollback, plus clearing the custom domain here so the
name is not claimed in two places.

### 7. Wait for the certificate

Poll until the state reads what the old origin reads today:

    gh api repos/Flowfin/site/pages --jq '.https_certificate.state'

Until it says `approved`, a reader who reaches the name over https meets a
security warning rather than a page. This is the step whose duration nobody
controls.

Telling a certificate that is still being issued from one that is not going to
be: the first changes state between two readings, the second does not, and a
state that has not moved in an hour is the second case however encouraging it
reads. The causes worth checking in that order are the DNS answers in step 1
having changed, the ownership check in step 1 no longer resolving, and the
custom domain field holding a name that differs from the one in DNS by a
trailing dot or a `www`.

Backing out is step 5's rollback. Nothing about a certificate at the new origin
prevents the old one from serving again, because that one already holds its own.

### 8. Turn enforcement on, and check it separately

A new site starts with enforcement off. A cutover that stops at step 7 produces
a name that serves the site correctly to anybody who typed https or followed a
link to it, and serves it over plain http to anybody who did not, and the
difference is invisible in a browser.

The reading in step 1 shows why that is not caught by anything else. There is no
transport security header on this name, so nothing tells a browser to upgrade on
its own, and the redirect is the whole mechanism. It exists only when the flag is
on.

Turn it on, then read both the flag and the behaviour, because the flag is the
setting and the redirect is what a reader meets:

    gh api repos/Flowfin/site/pages --jq '{https_certificate:.https_certificate.state,https_enforced}'
    curl -sS -o /dev/null -w 'http->%{http_code} %{redirect_url}\n' http://flowfin.dev/

The second has to answer with a redirect to the https address, which is what the
old origin answers today.

Backing out is turning the flag off, which is worth doing only if the site is
being rolled back entirely, in which case it is step 5's rollback.

### 9. Read the paths that are not pages

The catalogue address is polled by servers rather than opened by readers, so
nothing reports it broken. Read it deliberately, with the token file beside it:

    for p in /manifest.json /design-tokens.json; do
      printf "%-40s " "https://flowfin.dev$p"
      curl -sS -o /dev/null -w "%{http_code}\n" "https://flowfin.dev$p"
    done

What each of those should answer at that moment depends on what step 4 carried.
Both answer `200` today, so on the day of the move a `404` at either of them is
the move having dropped a file rather than a state to read past. A `404` on the
catalogue address is also the state before anything is published upstream, and
that is what it meant until 2026-08-23; it has stopped meaning that, and neither
reading is evidence that the move went well.

### 10. Record what the move produced, from outside

The readings above are taken from one machine on one network at one moment.
Recording them as the state of the site would be a claim dressed as a
measurement, which is what the harness in #39 exists for: it runs from outside,
it carries the date and the conditions, and the plain http response and the
certificate state are two of the things it records.

Backing out is not a step here. This one only reads, and it reads after the
change rather than instead of it.

## The paths that change

The addresses this site produces are settled in
[decisions/0008-the-url-shape.md](../decisions/0008-the-url-shape.md) rather
than here. What matters on the day is which of the addresses answering today
answer afterwards.

`/` keeps answering, and so does `/index.html`, because the build writes that
file and the host serves it for the directory above it.

`/design-system.html` keeps the address it already has. Record 0008 chose that
deliberately over moving it, so the one published address besides the root does
not become a dead link.

`/install/`, `/plugins/<id>/`, `/privacy/`, `/legal/` and `/404.html` are new.
Nothing links to them today, so nothing breaks by their arriving.

`/manifest.json` answers today with a `200`, carrying a catalogue this
repository does not author, and it is the commitment record 0006 holds.
It is the reason #67 refuses a publish rather than completing one.

`/design-tokens.json` answers today with a `200`, is authored in the repository
that holds the machine-readable data, and is served under this name only because
that repository is the origin. Record 0006 puts the catalogue file in exactly
that position and names only that file; this one is in the same position and is
named in neither that record nor #67. So on the day the origin moves, this
address stops answering unless something carries the file the way #67 carries
the other one, and whether it should be carried, redirected or allowed to stop
is a decision nobody has taken. It is written here so that the cutover does not
discover it, and it is not settled by this document.

## Where this runbook departs from the answer

Two of the steps above were written against a different answer to entry 3 than
the one that was taken, and neither is repaired here.

The first is the address the new site is proven at. Step 2 publishes it at the
address the host gives a project site, `https://flowfin.github.io/site/`, and
the answer says a subdomain. Whether the host's project address satisfies that
word or whether it asks for a name under `flowfin.dev` is not decided here, and
the difference is not cosmetic: a name under `flowfin.dev` is a second custom
domain, which is a settings action this document does not describe and a
certificate this document does not wait for.

The second is the window, and it is a contradiction rather than a reading. Step
5 releases the name from the old origin before step 6 claims it at the new one,
and says in its own words that from there until step 6 completes no published
site claims the name. The answer says the catalogue address answers throughout,
with no resolution gap. Both cannot hold as this sequence is written.

What makes it structural rather than an ordering mistake is stated above under
`## What moves, and what does not`: one custom domain is attached to one
published site at a time, so the old site releases the name before the new one
can take it, and that is where the window comes from. That sentence is a claim
about how the host behaves rather than a measurement, and measuring it costs
performing the move once. So closing the window means either a route this
document has not found, or the clause that forbids the gap giving way, and
neither is chosen here.

## What this runbook does not cover

Whether the domain moves at all, and on what schedule, was entry 3 of #7 and is
answered above. What the answer costs this sequence is the section before this
one; reconciling the two is not covered here.

Publishing the built output is #58, and the refusal of a publish that would
leave the catalogue address unanswered is #67. This document assumes both exist
by step 4 and checks their result rather than describing them.

Measuring the outcome from a real network is #39. Steps 6 to 9 read enough to
decide whether to continue or to roll back, and that is a different question
from whether the published site is good.
