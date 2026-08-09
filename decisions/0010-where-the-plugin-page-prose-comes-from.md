# 0010. Where the prose on a plugin page comes from

There is to be a page per plugin, and the roster is to carry one sentence per
row. That sentence is already spoken for twice over: it is what the landing
page's table prints and what the organisation profile publishes. If nothing else
is decided, a plugin page is that same sentence, a state word and a link to a
repository, which is the table row with more space around it.

## What was measured

    gh api repos/Flowfin/.github/contents/profile/README.md --jq '.content' \
      | base64 -d | grep -c '^| \['
    12

Run 2026-08-09. Twelve rows, one sentence each. Twelve pages that reprint them
are twelve addresses a reader can be sent to for no reason, and they would still
have to be kept, checked, budgeted and rendered.

## The decision

Per-plugin prose lives in this repository, one file per roster identifier under
`content/`. The roster keeps exactly its one sentence and gains no prose field.

The build refuses a roster row with no prose file, and it refuses a prose file
matching no roster row. Both directions are failures, and each failure names the
identifier rather than the file, because the identifier is the thing the two
halves share.

## Why

The split this board was made for puts machine-readable data on one side and
paragraphs on the other. A paragraph in the roster is prose sitting in the file
whose other consumer is a table that will never print it, and it would mean the
site's voice is edited in the repository that exists to hold data, reviewed by
people reviewing data.

Keying by the roster identifier is what keeps the two halves from drifting apart
without anybody noticing. A row with no page prose and a page with no row are
both decidable by reading the tree, with no network and no judgement, and it is
the same walk the sitemap check already has to do. Refusing in both directions
matters because the two failures have opposite causes: a row without prose is a
plugin the site is silently thin about, and prose without a row is a page nobody
can reach and nobody will notice has gone stale.

The one sentence stays in the roster because two consumers outside this
repository read it. Moving it here would make the profile table depend on a
website for the words in it, which is the direction record 0004 rejected.

## What the alternatives cost

A second, longer prose field in the roster. The cost is a data file grown a
field one of its two consumers ignores, and the reviewing of the site's prose
moved to a repository whose review is about data.

Fetch each plugin repository's README at build time. The cost is a build that
reaches the network and stops working offline, which the freshness rules refuse
for the roster and the tokens for the same reason; a document written for
somebody reading source landing on a page written for somebody deciding whether
to install; and the site publishing text nobody here reviewed.

Leave the pages at one sentence. The cost is twelve addresses that repeat the
table, counted against a budget that counts every page, and the install question
a reader actually arrived with answered nowhere.

## When this is worth revisiting

Prose per plugin that a client also has to show. That would make it data with
two consumers rather than a page, and it would be the same argument record 0007
makes about the numbers a client is held to.
