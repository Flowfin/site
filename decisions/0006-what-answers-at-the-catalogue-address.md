# 0006. What answers at the catalogue address once this repository is the origin

The plan on this board takes over the origin that serves the project's pages. The
same origin serves the address an operator pastes into a Jellyfin server, and
that address was published as a commitment that cannot be withdrawn.

## What was measured

What the domain answers:

    for p in / /manifest.json /design-system.html; do
      printf "%s " "https://flowfin.dev$p"
      curl -sS -o /dev/null -w "%{http_code}\n" "https://flowfin.dev$p"
    done
    https://flowfin.dev/ 200
    https://flowfin.dev/manifest.json 404
    https://flowfin.dev/design-system.html 200

Run 2026-08-08. The catalogue address does not answer yet, which is why this is
cheap to settle now and expensive to settle after it does.

Where the origin sits:

    gh api repos/Flowfin/hub/pages --jq '{cname,source}'
    {"cname":"flowfin.dev","source":{"branch":"main","path":"/docs"}}
    gh api repos/Flowfin/site/pages
    gh: Not Found (HTTP 404)

Run 2026-08-08.

## A claim this record rests on, stated as a claim

The host attaches one custom domain to one published site, so the day this
repository becomes the origin is the day every path under that name is served out
of what this build produced, including paths this repository does not author.
That sentence is a claim about how the host behaves rather than something either
measurement above shows, and it is written as one.

## The decision

The catalogue address keeps answering across a change of origin. What answers
there is the file the other repository generates and this one never authors. A
publish that cannot obtain that file fails instead of publishing a site without
it.

## Why

A catalogue address that stops answering raises no error on any server polling
it. The server keeps asking, the plugins stop updating, and the operator finds
out only if somebody notices a version standing still. That is the failure the
project already wrote down as the reason the address may never move, and a change
of origin produces it without moving the address at all.

Carrying a file is not authoring it. The generator, the schema and the release
pairing stay where the machine-readable data lives. This build obtains the
published artefact, checks it is the artefact it expected, and puts it at the
path the commitment already names. A build that renders it, reformats it or
regenerates it has taken ownership of the thing this repository was split off
from.

Failing closed is the only honest behaviour left. A publish that quietly omits
the file produces a site that looks right to every reader and is broken for every
server, and from the outside those are the same site until somebody fetches that
one path deliberately.

## What the alternatives cost

Serve the catalogue from a second address, on a subdomain or another host. The
cost is that the published commitment is the address operators already hold, so a
second one does not replace it and the first still has to answer.

Generate the file here as well. The cost is two generators for one artefact,
which is the drift the split exists to remove, in the one file where drift is
invisible.

## What this record does not decide

Whether the domain moves at all is the maintainer's call, as entry 3 of issue #7.
Nothing here decides it, and what this record states has to hold on whichever
answer comes back.

## When this is worth revisiting

The catalogue moving to an address that is not under this name, which ends the
coupling rather than managing it.
