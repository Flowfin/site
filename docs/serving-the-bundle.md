# Serving the bundle

What a release attaches, what has to be true of the host that serves it, and how
to check that what arrived is what was built. It is written so that somebody
serving this site never has to read the generator to run it.

## What the bundle is

An archive of the output directory and a list of hashes beside it. Nothing else:
no build step to run, no runtime to install, no configuration file to fill in.
The packing is what `.github/workflows/package.yml` does, and running the same
three commands against a checkout produces the same archive:

    go run . build
    (cd dist && find . -type f -print0 | sort -z | xargs -0 sha256sum > ../SHA256SUMS)
    tar --sort=name --owner=0 --group=0 --numeric-owner \
        --mtime='UTC 1970-01-01' --format=gnu \
        -cf - dist SHA256SUMS | gzip -n > site-bundle.tar.gz

What comes out is `dist/` with the hash list beside it. The names inside are not
listed here: there is one entry per page the build wrote, so a copy of that list
in this document goes stale between one page landing and the next, and it has.
Ask the archive:

    tar -tzf site-bundle.tar.gz | grep -c -v '/$'
    23
    tar -tzf site-bundle.tar.gz | grep -c '\.html$'
    18
    tar -tzf site-bundle.tar.gz | grep -v '/$'

Run 2026-08-28. Twenty-three entries that are not directories: eighteen pages,
the icon, the two files a crawler asks for without being sent, the security
contact, and `SHA256SUMS`. The contents of `dist/` are what a host serves, and
`SHA256SUMS` stays outside it so that serving the directory does not also serve
the list.

The bill of materials is attached to the release beside the archive rather than
packed inside it, as `sbom.cdx.json`. It says which toolchain and which base
image produced the bytes, and it is the document to read before deciding whether
the archive is something you want on your host:

    go run . sbom | sed -n '15,25p'
      "components": [
        {
          "type": "platform",
          "bom-ref": "toolchain/go@1.26.5",
          "name": "go",
          "version": "1.26.5",
          "purl": "pkg:golang/go@1.26.5",
          "description": "the toolchain go.mod pins, which is the one the build refuses to run without"
        },
        {
          "type": "container",

Run 2026-08-16. The lines above the excerpt name the generator and the version
these bytes were produced under, and they are elided here rather than pasted,
because that version is read from one file in the tree and a copy of it in a
document is the copy that goes stale. Read it from the document itself.

## What serves it

Anything that hands a file back for a path. A static host, a plain web server
pointed at the unpacked `dist/`, a container serving a mounted directory. There
is no application to keep running and nothing that has to be able to write.

Serve `dist/` as the document root. Every reference in every page is written
from the site root rather than relative to the page it is on, which is what lets
a page at any depth resolve the same way:

    grep -o -E '(href|src)="[^"]*"' dist/404.html
    href="https://flowfin.dev/404.html"
    href="/icon.svg"
    href="#content"
    href="/"
    href="/legal/"
    href="https://github.com/Flowfin/site/blob/main/NOTICE.md"

Run 2026-08-28. That is also the reason opening a page from a file path does not
work the way it looks like it should. A reference beginning with `/` resolves
against the root of the filesystem there, not against the directory the file is
in, so the links inside the site go nowhere. Point a local server at `dist/`
instead. Any of them will do, and the site does not care which.

## The not-found mapping

`dist/404.html` is the page for an address that matches nothing, and it is the
one thing a host has to be told about. Configure the host to return that file,
with a 404 status, for any request it cannot answer from a file. A host that
serves its own default page instead puts somebody else's page on your name, and
a host that returns the file with a 200 status tells a crawler the address
exists.

The file sits at the root of what is served rather than in a subdirectory,
because the root is the only place every host that supports this looks:

    ls dist/404.html
    dist/404.html

Run 2026-08-16. That a host serves a file of this name for an unmatched request
is a claim about hosts rather than something read off one, and
`decisions/0008-the-url-shape.md` records it as a claim. Check it against your
own host rather than assuming it: ask for an address the site does not have and
read both the body and the status.

## The addresses written into the output are fixed

Three things in the bundle name `https://flowfin.dev` and go on naming it
wherever the bundle is served:

    grep -c -o 'https://flowfin.dev' dist/index.html dist/sitemap.xml dist/robots.txt
    dist/index.html:2
    dist/sitemap.xml:17
    dist/robots.txt:1

Run 2026-08-28. The counts are here and the addresses are not, for the reason
the archive listing above is not written out either: the sitemap carries one
line per page it invites a crawler to, and a copy of those lines in this
document is a copy nothing keeps in step. The landing page is one page of the
eighteen and states its own address twice, once as the canonical link and once
as the address a shared card carries; every other page does the same. The
sitemap lists addresses rather than paths, and the robots file names where the
sitemap is. Serving the bundle under a different name leaves all of them
pointing at the original one, which is correct for a mirror and wrong for a
fork. Nothing in the bundle can be edited to change that without the hashes
ceasing to match; rebuild from a source tree instead.

## Nothing in it reaches anywhere

No page fetches anything from another origin, sets a cookie, touches browser
storage or carries a handler that could do any of it. Those are refused rather
than promised, and the refusals run over every file a build wrote:

    go run . invariants | grep -E '^  (page-fetches-no-script|page-touches-no-browser-storage|page-carries-no-inline-handler|output-references-no-domain-outside-the-allowlist):'
      page-fetches-no-script: ok, 18 file(s) of every page the build produced
      page-touches-no-browser-storage: ok, 18 file(s) of every page the build produced
      page-carries-no-inline-handler: ok, 18 file(s) of every page the build produced
      output-references-no-domain-outside-the-allowlist: ok, 22 file(s) of every file the build produced

Run 2026-08-28, with the pattern anchored so that the line the verb opens with,
which names every row it holds, is not matched by the names inside it. There is
one address off this origin in the pages, and it is a link to the notice file
rather than something a browser fetches. A reader who does not click it makes no
request anywhere but to your host.

Two bounds on that, because the rows read what a page spells rather than what it
does. A page reaching an interface through a value none of the names finds is
refused by none of them, and a script element carrying its code inside the page
is refused by none of them either. Both are what the browser leg in issue 50 is
for, and it does not exist yet.

## Checking what you downloaded

Two different questions, and only one of them is answered by the archive.

Whether the archive arrived intact is answered from inside it. Unpack it and
check the files against the list that came with them:

    tar -xzf site-bundle.tar.gz
    (cd dist && sha256sum -c ../SHA256SUMS)

Whether the archive is what this repository built is not answerable from inside
it, because the list and the files it describes came from the same download.
What answers that is building the tag yourself and comparing. The build is
reproducible and the packing is deterministic, so two runs of one source tree
produce the same archive byte for byte:

    go run . reproduce
    reproduce: two builds of ., compared byte for byte
      22 file(s), identical in both builds

    sha256sum pack-1.tar.gz pack-2.tar.gz
    c0c27bcd42ef373f774f0d907bd4a1787c192624ac5ead66dc770efba831442b *pack-1.tar.gz
    c0c27bcd42ef373f774f0d907bd4a1787c192624ac5ead66dc770efba831442b *pack-2.tar.gz

Both runs 2026-08-28, the second over two archives packed one after the other
from the same checkout by the commands at the top of this document. Check out
the tag, pack it the same way, and compare the digest against the file you
downloaded.

## What you are not getting

No search, no comments, no analytics, and nothing that updates itself. There is
no process to restart and no database to back up, which is most of the point,
and the price is that the archive is stale from the moment it is downloaded.
Nothing on your host will tell you a newer one exists. What says whether a newer
one is worth taking is the version, and what each part of it means is
`decisions/0013-the-version-scheme.md`.

Publishing and tagging are separate acts, so a bundle that turned out to be
wrong is replaced by serving the previous one rather than by deleting anything.

## What is not measured here

No release exists yet, so nothing described above has been downloaded from one:

    gh release list --repo Flowfin/site --limit 5
    (no output)

Run 2026-08-16. Everything above was measured against an archive packed from a
checkout by the commands this document quotes, which is the same definition the
release run calls, and not against a published file.

Nothing in this tree holds this document to the output it describes. Every
figure above was taken by running the command beside it, and the next page that
lands moves several of them without reddening anything, which is what happened
to the readings taken on 2026-08-16 and repaired on 2026-08-28. Where a number
would have been a list, the command is handed over instead of the answer, so
the shape that drifts is smaller than it was; the rest are numbers somebody re-
took and will have to re-take again. Re-run the commands rather than trusting
the outputs pasted beside them.

The two archives compared above were packed on one machine, so what is measured
is that the packing does not vary between runs. Whether it also produces the
same bytes under a different `tar` or a different `gzip` was not measured, and
the reproducibility check compares the built files rather than the archive
around them.
