# 0002. What Docker is for

A static site can be built in a container and served from anywhere, or it can be
served out of a container. Those are different things, they cost different
amounts, and a repository that has not said which one it means gets a Dockerfile
that quietly means the other. Writing it down before the file exists is cheaper
than arguing about the file afterwards.

## What was measured

The domain is already served by the host, with the certificate already attached:

    gh api repos/Flowfin/hub/pages --jq '{cname,html_url,https_enforced,source}'
    {"cname":"flowfin.dev","html_url":"https://flowfin.dev/","https_enforced":true,"source":{"branch":"main","path":"/docs"}}

Run 2026-08-09. That is the site as served out of the other tree today, and
record 0008 is where the addresses under that name are fixed. What the
measurement settles here is narrower: an origin for these bytes already exists,
is already answering, and already holds a certificate, so a container that
served them would be a second origin rather than the first one.

## The decision

Docker is the build environment. It is not the serving environment. The site
stays served from GitHub Pages.

Three things the container may not become, and they are refusals rather than
present intentions. It does not serve. It does not publish. It holds no
credential.

The same build verb runs inside the container and outside it, so there is one
procedure and not two. A container that can build something the toolchain
cannot, or that produces different bytes from the same source, is a defect in
the container and never a second supported way to build.

## Why

Serving out of a container means running a server, and nothing that runs on a
server is in scope for this site. A serving container is an origin somebody has
to patch, watch, reach and pay for, bought in exchange for handing out files
that a static host already hands out for nothing.

A build container costs one Dockerfile pinned to an image digest, and it buys
two things worth having. A contributor who installs no toolchain produces the
same bytes the build produces, and the bytes that were published can be produced
again later from the tagged source.

The three refusals are what keep that cheap. A container that serves is the
first paragraph again. A container that publishes is a second route to the
default branch, and the route that publishes is the one worth attacking. A
container that holds a credential turns an image somebody may pull into a place
a secret can leak from, and a build that needs no secret should not be given the
chance to want one.

## What the alternatives cost

Serve from a container. The cost is an origin to run: patching, monitoring,
reachability and a bill, plus a certificate this project would then own rather
than borrow, all for files that need none of it.

No container at all, and the toolchain is the only way to build. The cost is
that a contributor's output depends on what is installed on their machine, and
that reproducing a past release means reconstructing a toolchain rather than
pulling a digest.

A container that also publishes, so one image does the whole job. The cost is a
credential inside a thing that gets pulled, and a publishing route that no
longer runs where the rest of the automation can be read.

## When this is worth revisiting

A hosting need the static host cannot meet, such as a redirect rule or a
response header it will not set. That would be a question about the origin, and
it would reopen this record rather than being decided around it.
