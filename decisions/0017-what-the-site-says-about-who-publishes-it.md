# 0017. What the site says about who publishes it

Publishing a site under a name of one's own carries a provider identification
duty in some jurisdictions, and the page that would carry the answer exists and
states nothing: it was built to hold no words that identify anybody, because
what goes on it is a personal cost rather than a design question. That is a
page that can stay open indefinitely, and a duty that is not decided is not
thereby absent. Entry 8 of #7 is where the question was held, and it was
answered on 2026-08-24. This record is that answer written where the work that
reads it can find it, rather than on a tracker.

## What was measured

The served site named nobody and offered no route to anybody at the time the
question was taken, and the organisation profile carried a donation link, which
is the kind of detail that decides whether a site is read as private or as
commercial:

    curl -sS https://flowfin.dev/ | grep -o -i -E 'impressum|imprint|datenschutz|privacy'
    exit=1
    gh api repos/Flowfin/.github/contents/profile/README.md --jq '.content' \
      | base64 -d | grep -c -i 'buymeacoffee'
    1

Run 2026-08-08, and carried here from the entry that took the question rather
than re-run, because what it establishes is the state the decision was taken
against.

The legal notice this repository builds reads its values out of a file, and
every value in that file is still waiting on the entry:

    jq -r 'to_entries[] | "\(.key)  \(.value.state)  \(.value.waiting)"' data/publisher.json
    publisher  undecided  entry 8 of issue 7
    contact  undecided  entry 8 of issue 7
    postal  undecided  entry 8 of issue 7
    go run . build | grep legal
    wrote dist/legal/index.html (3562 bytes, 0 of 3 answered)

Run 2026-09-05 at `2add524`. So the page points a reader at a question that has
an answer, and the answer is written nowhere the page could point at instead.
That is what this record is for.

## The decision

Full provider identification, carried by a paid imprint-address service rather
than a home address, and a rotatable contact alias on the legal page.

The service booking happens outside this repository. Once the address exists,
the legal page and the privacy statement are ordinary build work: the values
arrive in the file the page reads, and the lines saying the question is open
become answers. Professional advice stays sensible and this decision does not
replace it. What it ends is the state of publishing nothing.

## Why

The duty, read at its strictest, asks for an address at which documents can be
served, and the only two answers that meet it put an address on a page built to
be indexed. One of them is a home. A private address published once is
collected within days and kept by scrapers after any later change, so that
answer is paid for permanently, by a person, and cannot be withdrawn. The
service costs money every year and puts a third party in the chain, and in
return the address is real for service of documents without being anybody's
home. That is the trade, and it is the only one of the four options that meets
the strictest reading at a price that can be stopped paying.

The alias is the same reasoning applied to the contact route. A route printed on
a public page is collected within days, so one that can be rotated without
changing anything else is worth more here than a memorable one, and the page
renders it from data so that rotating it is a change to one value rather than an
edit to prose.

Whether the site reads as private or as commercial is what decides which duty
applies, and this answer does not wait on that reading. Full identification
satisfies the stricter of the two, so how the site reads, which the donation
link and entry 9 bear on, stops being a question the legal page has to settle
first.

## What the alternatives cost

A full provider identification with a home address. Cost: a private address
becomes permanently public on a page built to be indexed, and scrapers keep a
copy after any later change. It is what the strictest reading of the duty asks
for, and it costs the most in the one currency that cannot be recovered.

A name and a contact route with no postal address. Cost: a reader can reach
somebody, and a jurisdiction that asks for a summonable address is not
satisfied. It is what most personal projects do, and it leaves the obligation
half met.

A name and nothing else. Cost: cheapest, and it leaves both a reader and any
obligation with nothing.

Publishing nothing, which is the state the page was in. Cost: the page says who
publishes this site by saying nothing, for as long as nobody decides, and the
duty is not smaller for being undecided.

## When this is worth revisiting

When the service stops being available or stops being worth its price. The
answer rests on the address being real for service of documents without being a
home, so a replacement that carries the address is a new record naming what
carries it instead.

When professional advice says the identification should carry more or less than
this. The decision ends the state of publishing nothing and is not a legal
opinion, so advice that arrives later changes it without contradicting it.

A compromised alias is not a revisit. Rotating it is the data change the answer
was built for, and this record does not name the alias.

## What this record does not decide

The values. The name, the address and the alias are what the legal page will
show, they exist once the booking is done, and this record names none of them.
Until they arrive the page states that the question is open, and what it should
state in the meantime about an answer that is decided and a value that is not
yet available is a page question rather than this record's.

Whether the site asks for money, and in what form. That is entry 9 of #7, it
was answered on the same day, and 0018 carries it. The two touch where the
donation link bears on how the site reads, and this record already says why
that reading does not have to be settled first.

Which jurisdiction's duty applies. Nothing in this repository decides that, and
nothing here is legal advice.
