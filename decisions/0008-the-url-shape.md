# 0008. The URL shape, and the addresses that already answer

Every page this site produces is an address, and an address somebody has
bookmarked or linked is as permanent as this project decides to make it. The
scheme has to be settled before the generator writes the first path, because
every later change that walks links, checks assets or writes a not-found page
reads it.

## What was measured

What answers, and at which shape:

    for p in / /index.html /design-system.html /manifest.json; do
      printf "%-40s " "https://flowfin.dev$p"
      curl -sS -o /dev/null -w "%{http_code}\n" "https://flowfin.dev$p"
    done
    https://flowfin.dev/                     200
    https://flowfin.dev/index.html           200
    https://flowfin.dev/design-system.html   200
    https://flowfin.dev/manifest.json        404

Run 2026-08-08. Two page addresses answer, both flat files at the root, and the
whole served tree is three files:

    gh api 'repos/Flowfin/hub/git/trees/HEAD?recursive=1' \
      --jq '[.tree[].path|select(startswith("docs/"))]'
    ["docs/CNAME","docs/design-system.html","docs/index.html"]

Run 2026-08-08.

## The decision

Pages this site introduces live at directory addresses with a trailing slash.
`/design-system.html` keeps the address it already answers at rather than moving
to one. Every asset reference in a produced page is absolute from the site root.

The address of every page the plan produces:

    /                       the landing page
    /install/               the install page
    /plugins/<id>/          one per plugin, <id> being the roster identifier
    /design-system.html     the design system page, at the address it already has
    /privacy/               the privacy page
    /legal/                 the legal notice
    /404.html               the not-found document

Eighteen pages, which is the count record 0003 was taken against.

The not-found document is the one entry not chosen here. The host serves a file
of that name at the root in response to a request that matches nothing, and it is
never linked, so the name is fixed by the host rather than by this scheme. That
is a claim about the host rather than a measurement. The same holds for the files
a browser and a crawler ask for without being linked, which keep the names those
agents ask for.

## Why

The twelve plugin pages need a container or they collide with everything else at
the root, so `/plugins/<id>/` is where they go, and depth exists in this site
whether or not anything else moves. Directory addresses carry no file extension,
which is the part that survives a change of generator, and the host serves the
index document inside one without being told to.

`/design-system.html` is the one address the project has published that is not the
root. Moving it buys consistency a reader never sees and costs a dead link a
reader does see. The static host sets no redirect header, so the only way to carry
somebody onward from a moved address is a file at the old path holding a refresh
element, which is slower than the page it replaces and is a second thing to keep.
Not moving it costs one inconsistent address in the tree.

Absolute asset references follow from depth rather than from taste. A relative
path renders correctly from the page it was written for and breaks from a page one
level down, and the not-found document is served in response to a URL of any depth
at all, so a relative reference in it is broken by definition on some requests.
Absolute from the root is correct at every depth and is checkable by reading the
output.

## What the alternatives cost

Flat `.html` files for everything, matching what is served today. The cost is
twelve plugin pages competing with the rest of the site for names at the root, and
an extension in every published address, which is a detail of how the site was
built appearing in the address of what it says.

Directory addresses for everything, moving the design system with them. The cost
is the refresh file above, or a dead link, on the only address besides the root
that this project has published. It is not linked from the organisation profile,
which points at the root:

    gh api repos/Flowfin/.github/contents/profile/README.md --jq '.content' \
      | base64 -d | grep -o 'https://flowfin.dev[^)"]*' | sort -u
    https://flowfin.dev

Run 2026-08-08. So what a move would break is a link somebody made themselves,
which is the kind this project cannot see and cannot repair.

## What this record does not decide

Where the addresses are served from is entry 3 of issue #7 and is my call. The
catalogue address is held by record 0006 and is fixed by a commitment rather
than by this scheme.

The scheme above is written for a site published in one language. Entry 2 of issue
#7 has an option that publishes two, and on that answer every address here gains a
language segment, the root has to decide which language it answers in without
reading anything about the reader, and each page gains a link naming its
counterpart so the two do not compete as duplicates of each other. That is a
different scheme rather than an addition to this one, so it arrives as a new
record superseding this one by number. It is named here because the cost of the
two-language answer is partly paid in addresses that were already published.

## When this is worth revisiting

A host that sets redirect rules, which turns moving an address into something
cheaper than carrying it.
