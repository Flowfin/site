# 0005. The speed budget

A project whose promise is that nothing waits cannot open with a slow page. A
budget written as an intention is not a budget, so this one is written as numbers
a build can miss.

## The decision

The budget below holds per page, measured on the built output rather than on a
development server. Each line is a refusal rather than a target. A build that
misses one line is a failed build.

    HTML, uncompressed                              under 20 KB
    CSS, uncompressed, inlined in the document      under 12 KB
    JavaScript required to read the page            0 bytes
    Requests to a domain the project does not own   0
    Web font downloads                              0
    Requests for the landing page                   1 document, at most 2 images
    Largest contentful paint, throttled mobile      under 1.0 s
    Cumulative layout shift                         exactly 0

## Why, one line at a time

HTML under 20 KB uncompressed: a page of prose that cannot be written inside that
much markup is carrying structure the reader is not being shown.

CSS under 12 KB uncompressed and inlined: inlining removes a round trip before
anything renders, and the size is what keeps inlining cheaper than the request it
replaced.

Zero bytes of required JavaScript: this is the strong form of the promise. A
script may exist as an enhancement, but the page has to be readable with
scripting off, which also removes a whole class of things that could go wrong on
a reader's machine.

Zero requests to a domain the project does not own: this is here for speed and it
happens to be the same rule the privacy page will need. A font, an icon set or a
badge served from somebody else's domain is a round trip the reader pays for and
a request that tells a third party who is reading.

Zero web font downloads: a downloaded face blocks or reflows the first text a
reader sees, and the faces already on the reader's machine cost nothing and
arrive first.

One document and at most two images for the landing page: the first page has to
be complete after the fewest exchanges, and a limit counted in requests is the
one a reader on a slow link actually feels.

Largest contentful paint under 1.0 s on throttled mobile: the number is set on
the worst connection the site expects rather than on the machine the page was
written on.

Cumulative layout shift of exactly zero rather than the usual small allowance:
the project already says that no image arriving late may shift the layout, and a
static page with known image dimensions has no excuse for the shift.

## What was measured

The landing page as served, at the time this record was written:

    curl -s -o /dev/null -w '%{size_download}\n' https://flowfin.dev/
    2050

Run 2026-08-08. The numbers above are loose enough to write real content against.
This measurement is of the page served today rather than of anything this build
produces, and it moves when that page moves, so a later reader re-runs the
command rather than trusting the number.

## When this is worth revisiting

A page that needs an image-heavy shape the request line cannot hold, or a
measurement route that reports a different largest contentful paint for the same
bytes, which would make the threshold a property of the harness rather than of
the page.
