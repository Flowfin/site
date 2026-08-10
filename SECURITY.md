# Reporting a security problem

Report privately, through the form under Security on this repository:

    https://github.com/Flowfin/site/security/advisories/new

Please do not open an issue about a security problem. An issue here is public
from the moment it is submitted, and a report of this kind is the one thing that
should not be.

The route is a repository setting rather than a file, and it does not answer
today:

    gh api repos/Flowfin/site/private-vulnerability-reporting
    {"enabled":false}

Run 2026-08-10. So this file names the destination and states plainly that the
destination is currently shut, rather than sending a reporter to a door that
does not open. Opening it is issue #57, and it is not a change to this tree. The
address above is the one the form uses once the setting is on, so nothing here
moves when it changes. Until then the honest alternative is a public issue with
as little detail as the report can carry, which is a poor arrangement and is
written down as one.

## What this repository is, and what a problem in it can be

This is a generator and the static files it produces. What follows is the list
of things somebody could actually report, because a policy that repeats a
sentence about responsible disclosure and stops leaves a reader guessing at
which half of the project they are looking at.

Markup produced from roster data that escaped its context, so a sentence in the
data became script in a page.

A workflow that grants more permission than its job needs, or that can be made
to run against a branch on a fork.

A build that pulls something unpinned, so the bytes that get published are not
the bytes the source describes.

A published file that reaches another origin. The site promises that a reader's
address goes nowhere but the host they asked for, and anything in the output
that fetches from a domain this project does not control breaks that promise.

The domain, or the certificate on it, serving something this repository did not
build.

## What is out of scope, and why it looks like a gap

There is no server here, no account, no session, no database, and no personal
data. The categories a reader arrives with mostly do not exist: there is nothing
to log in to, nothing to enumerate, nothing to escalate into, and no record of
anybody.

So a report that the site has no login is not a finding. Neither is a missing
response header that a static host cannot set, and neither is the absence of a
rate limit on something that serves files.

A problem in Jellyfin itself belongs to
[the Jellyfin project](https://github.com/jellyfin/jellyfin/security/policy).
A report that lands here instead is pointed the right way rather than closed.

## What a reporter gets

Every report is answered, whether or not it turns out to be a problem. A report
that is not a problem gets the reason it is not, which is the answer that costs
the reporter nothing to receive and takes the guessing out of the silence.

There is no response deadline here. Nothing holds anybody to a window, so
promising one would be a sentence that goes quietly wrong on the first busy
week, and a policy that misses its own stated window is worse than one that
promises a reply and means it.

Credit goes to the reporter unless they ask otherwise. A working exploit is
useful in the report and not in public before there is a fix.

## What is covered

What is published now. There is nothing to backport a fix to: the site is
whatever the current build produced, an operator serving an older bundle
replaces it with a newer one rather than patching it, and no version of it is
maintained in parallel.
