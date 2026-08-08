# 0001. Where the plugin list comes from

The site shows twelve plugins and the state each one is really in. Writing that
list into the pages by hand guarantees it drifts, and there is already a second
hand-written copy of it on the organisation profile.

## The decision

The list comes from one machine-readable roster file that is published in
public. The build reads it. The shipping state is computed rather than declared,
and a check reds when a declared state and the computed one disagree.

Each row carries an identifier, the repository the plugin lives in, one sentence
saying what the plugin does, and a declared state that may only be `build-up` or
`shell`. The third state, `ships`, is not declarable, because whether something
ships is a fact about published releases and not an opinion. The build asks the
release list and computes it. A row that declares `shell` while its repository
has published releases reds the build, and that is what makes the table honest
rather than merely generated. Which releases count is finer than this record
settles and is decided in record 0009.

The roster lives beside the manifest, in the repository that already exists to
hold machine-readable data, so this repository keeps its side of the split and
holds paragraphs. The site vendors a pinned copy of the roster so that a build is
reproducible and works offline, and a scheduled check reds when the pinned copy
has fallen behind the published one.

## What was measured

How many of the plugin repositories have published anything at all:

    for r in $(gh repo list Flowfin --limit 60 --json name \
      --jq '.[].name|select(startswith("jellyfin-plugin-"))'); do
      printf "%-36s %s\n" "$r" "$(gh api repos/Flowfin/$r/releases --jq 'length')"
    done | awk '$2>0'
    jellyfin-plugin-requests             1
    jellyfin-plugin-sso                  30

Run 2026-08-08. Two of the twelve carry a release between them and ten carry
none.

What the organisation profile says about those same two:

    gh api repos/Flowfin/.github/contents/profile/README.md --jq '.content' \
      | base64 -d | grep -E '^\| \[(sso|requests)\]' \
      | sed -E 's/^\| \[([a-z-]+)\][^|]*\|[^|]*\|/\1/'
    sso Ships |
    requests Shell only |

Run 2026-08-08. The plugin with one published release is listed as a shell, and
that disagreement between a declared state and a published fact is the one a
check refuses rather than reports.

## Why

The plugin catalogue manifest cannot carry the list on its own. A Jellyfin
manifest lists what has a release, and almost nothing here does. A site derived
from the manifest alone would be silent about most of what the organisation
profile lists, which is the opposite of the honest table this site is supposed to
be.

The other candidate source is not public. A site whose build depends on an input
nobody outside can read is a site nobody outside can rebuild, and being
rebuildable from public inputs is most of the reason for moving the site out of
the tree that serves the manifest.

## What the alternatives cost

The manifest as the source. The cost is a table that omits ten of the twelve
plugins, and a site that says less than the organisation profile already says.

A private source. The cost is that no reader can reproduce the site from what the
project publishes, and the separation this repository exists for stops being
visible from outside.

## When this is worth revisiting

A roster that grows fields no page uses, or a day when every plugin ships and the
computed state stops distinguishing anything.
