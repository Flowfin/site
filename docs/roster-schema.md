# The roster file

One file, machine-readable, published in public, holding one row per plugin.
Everything this site says about the twelve plugins comes out of it, and so does
the table on the organisation profile.

Where the file lives and why is `decisions/0001-where-the-plugin-list-comes-from.md`.
This document is the shape of it, and the shape is the same wherever the bytes
come from.

Where they come from today is this repository. Entry 6 of #7 asked who lands the
file this board only reads and what the build starts from, and it is answered:
the site builds against a copy committed here and vendors the published file
once that file exists in the repository holding the machine-readable data. So
`data/roster.json` is a copy this repository authors for now and a vendored one
later, and nothing about the build moves on that day. What moves is where the
bytes come from, which is #24.

`go run . roster` is what says the copy has fallen behind. It compares the
pinned rows against the published ones, names every row that differs, writes
nothing, and reports OFF rather than green where nothing is published to compare
against. Nothing is published there today, so every run reports OFF; the address
it reads and what turns its schedule on are in `internal/freshness` and in the
workflow beside it.

Beside it is `data/releases.json`, which is not part of this shape. It carries
what each repository the roster names has published, taken once by
`go run . releases` and written down with the request that took it. The build
reads it for two things: whether the plugin ships, which
`decisions/0009-what-counts-as-shipping.md` decides and no row may declare, and
whether the row's repository is there at all, which the parser asks and refuses
a read for skipping. Both are requests off the machine, which a build may not
make, so both are answered from the tree.

## The format

JSON, and a top-level array.

JSON rather than YAML, because the file is read by a program and edited rarely,
the parser is in the standard library, and YAML's implicit typing turns a version
string into a number when nobody is looking.

An array rather than an object, because the order of the rows is the order the
site presents them. The ordering lives in the data, where somebody can change it
by moving a row, rather than in a sort somebody has to justify.

## The fields

Every field is required and every one is a string. A row carrying a field not
named here is malformed, and so is a row missing one.

Required means carrying a value. A field present and empty is the same state to
anybody reading the file as a field that is not there, and it is refused the same
way, because the alternative is a page whose address is built out of nothing.

`id` is the plugin's identifier, and it matches the suffix of the repository name
after `jellyfin-plugin-`. It is what the address of the plugin's page is built
from and what the per-plugin prose in this repository is keyed by, so it is the
one value in the row that other files depend on being stable.

`repository` is the repository the plugin lives in, written as `owner/name`. It
is what the build asks for the release list when it computes whether the plugin
ships.

`summary` is one sentence saying what the plugin does, in the voice the site
uses. It is one sentence and not a paragraph: the paragraphs that make a plugin
page worth opening live in this repository keyed by `id`, so this file stays
small enough for the two consumers outside it to read.

`state` is what the row declares about the plugin, and it may only be `build-up`
or `shell`.

## `ships` is not a value this file may carry

Whether something ships is a fact about published releases rather than an
opinion. The build asks the release list and computes it, and a schema that lets
a person write `ships` is a schema that lets the table lie.

So `state` declares the floor rather than the answer. A row says what is true when
nothing is published, and the build raises it where the releases say otherwise. A
row declaring `shell` for a repository that has published releases is a
contradiction rather than a state, and it reds the build.

Which releases count as shipping is
`decisions/0009-what-counts-as-shipping.md`. This file carries no field about it
at all, so a change to that rule changes no row here.

## An example row

    [
      {
        "id": "watchlist",
        "repository": "Flowfin/jellyfin-plugin-watchlist",
        "summary": "A private per-user watchlist kept on the server, shown by clients that were never changed",
        "state": "build-up"
      }
    ]

The sentence in that row is the one the organisation profile already publishes
for the same plugin, which is where the twelve sentences come from on the day the
file is first written:

    gh api repos/Flowfin/.github/contents/profile/README.md --jq '.content' \
      | base64 -d | grep -E '^\| \[watchlist\]' | cut -d'|' -f3
     A private per-user watchlist kept on the server, shown by clients that were never changed

Run 2026-08-08.
