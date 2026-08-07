# site

The public site is served today out of the hub tree, which mixes a page written for people with a manifest written for machines. Every change to a paragraph touches the tree that serves machine-readable data, and the site cannot be moved without moving the manifest with it. Static only, with no account, no form and nothing that can leak. The plugin list is derived rather than written by hand, because a hand-written list is wrong the day a plugin ships. A project whose promise is that nothing waits cannot open with a slow page, so the site states its own speed budget as numbers.

Planning happens on the issue tracker first. Every decision that shapes
the architecture is written down there with its reasons before the code
that depends on it exists.

See [NOTICE.md](NOTICE.md) for the intended-use notice.
