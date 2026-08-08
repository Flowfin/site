# 0009. What counts as a plugin that ships

Record 0001 makes the shipping state computed rather than declared. Computing it
from whether a repository has any published, non-draft release is coarser than
the word it produces, and the data behind it already disagrees with itself.

## What was measured

The repository that ships carries thirty non-draft releases, nineteen of them
marked as prereleases:

    gh api repos/Flowfin/jellyfin-plugin-sso/releases \
      --jq '{nondraft:[.[]|select(.draft==false)]|length,prerelease:[.[]|select(.draft==false and .prerelease==true)]|length}'
    {"nondraft":30,"prerelease":19}

The newest non-draft release is one of them:

    gh api repos/Flowfin/jellyfin-plugin-sso/releases \
      --jq '[.[]|select(.draft==false)][0]|{tag_name,prerelease}'
    {"prerelease":true,"tag_name":"5.0.0-JF12-beta.42"}

The most recent release the flag calls finished carries `beta` in its own name:

    gh api repos/Flowfin/jellyfin-plugin-sso/releases \
      --jq '[.[]|select(.draft==false and .prerelease==false)][0]|{tag_name,prerelease}'
    {"prerelease":false,"tag_name":"4.3.0-beta.28"}

All three run 2026-08-08. Under a non-draft test the newest thing a reader would
be sent to is a prerelease, and the word on the table would still be the
strongest one the project has. A rule reading the flag and a rule reading the tag
return different answers about the same release, so the record says which signal
decides rather than leaving the build to pick one silently.

## The decision

A plugin ships when its repository has at least one published release that is
neither a draft nor a prerelease. The prerelease flag is the signal and the tag
string is not read. A repository whose only releases are prereleases is shown on
its own page as having something to test, in words that cannot be mistaken for
the table's word, and that news stays out of the state the table computes.

## Why

The word has to mean something a reader can act on. The three state words are the
site's central claim and the reason the table is derived at all. A word that is
satisfied by anybody tagging anything is a word that costs nothing to earn, and
the first reader who installs a prerelease because a table called it shipping is
the person the honest table exists for.

The flag decides because it is the field the publishing side sets deliberately and
the only one a machine can read without a convention. A tag is a string, no
repository here is held to a versioning scheme, and a rule that greps for `beta`
misreads the first plugin that spells its prereleases differently. The cost of
taking the flag is visible in the third measurement and is worth naming: a release
whose own name says beta counts as shipping, because the side that published it
said it was not a prerelease. That disagreement is a thing to fix where it is
published rather than a thing for this build to guess at.

A prerelease is not nothing, and dropping it on the floor would lose the one piece
of news a build-up plugin can offer, which is why it appears on the page rather
than in the state.

## What the alternatives cost

Count any non-draft release. The cost is the first measurement above, where the
newest release of the one shipping plugin is a prerelease, and the eleven others
inherit that rule the day they tag their first beta. It makes the table easy to
satisfy in the one direction the table exists to refuse.

Read the tag rather than the flag, refusing anything carrying a prerelease suffix.
The cost is a build depending on a naming convention nothing enforces in any of
the twelve repositories, and it contradicts the flag today on the plugin that
ships.

Require a version floor, such as a first component of at least one. The cost is
refusing the shipping plugin's whole history and calling the project's own
numbering a promise it has not made.

## When this is worth revisiting

A repository publishing releases through something that does not set the flag,
which turns the signal question back into an open one.
