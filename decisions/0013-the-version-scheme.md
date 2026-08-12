# 0013. What a version of this site means, and what moves it

This repository is going to publish a bundle somebody else serves, and record
0002 already says the container builds rather than serves, so the thing an
operator holds is an archive rather than a running site. An archive needs a name
that says whether the next one is worth taking. A version that moved every time
a sentence was corrected would say nothing, and a version that never moved would
say less, so what each part means has to be settled before the first tag exists
rather than argued from the first disagreement about one.

## The decision

Three parts, `major.minor.patch`, and the version tracks the generator and the
structure of the output rather than the words on the pages.

The major part moves when something an operator or a reader depends on stops
working the way it did. An address that a page used to answer at and no longer
does, a file that leaves the output, a change to the shape of the bundle or of
the list of hashes beside it. This is the part that says a link somebody else
made may now be dead.

The minor part moves when the output gains something without taking anything
away. A page appears, a file appears beside the pages, a check starts refusing
something it used to allow, a budget number moves. Everything an operator can
take without editing anything of their own is here.

The patch part moves for a correction that changes no address, no set of
produced files and no number. A wrong word, a repaired sentence, a generator
change nobody serving the output can observe.

The words on the pages do not move the version on their own. Prose is what this
site is mostly made of and it changes constantly, so a version that followed it
would be a counter of edits, and the question it exists to answer is whether the
next bundle is worth taking.

The version is held in one place, as a constant in the source, and everything
that states it reads it from there. The build names it in its report, the bill
of materials states it on the component the document is about, and the release
run builds the tag out of it by running the verb that prints it. The tag is the
version with a `v` in front of it, and that prefix is written beside the constant
rather than in the workflow, so the tag and the version cannot be spelled
differently by two files. A row in the invariant table refuses the version
written into any other tracked file.

## Why

The alternative that keeps suggesting itself is to version the content, because
a website is what a reader sees and what a reader sees is the words. It fails on
the first correction: the person the version is for is the operator deciding
whether to pull a new archive onto their host, and the only thing that decision
turns on is whether anything they depend on moved. A typo repair does not move
it, and a page that changed address does.

Three parts rather than a date because the parts carry the answer. A date says
when and never says whether, so an operator reading two dates has to diff the
archives to learn what a version number could have told them.

One place for the number rather than a rule that copies stay in agreement,
because a copy is right the day it is typed and the copy that goes stale is the
one somebody reads. That failure is invisible from the tag: the tag is correct
and the document announcing a different release is what is wrong. The row is
what makes the single place a property rather than an intention.

## What the alternatives cost

A date-based version, such as the year and the day. Cost: nothing to decide per
release, no argument about which part moves, and the number answers none of the
questions an operator has. It also forces a second mechanism to say whether a
release breaks anything, which is the thing the parts were doing.

Versioning the content along with the generator. Cost: a version that moves on
every corrected sentence, which trains the operator reading it to ignore it, so
the release that does move an address arrives looking like all the others.

The version in a plain file at the root rather than in source. Cost: it reads
more directly, and both the build and the workflow parse it, which is two
parsers over one line and a trailing newline that becomes a tag nobody can
resolve. The verb that prints the constant gives the workflow the same value
without the second parser.

Two numbers, one for the generator and one for the output. Cost: it is honest
about there being two things here, and an operator holding an archive has to
know which of the two the archive is named after, which is a question a single
number does not raise.

## What this record does not decide

What is written down about a release, and where. This record says which part of
the number moves for which kind of change; the file that describes a release in
sentences, and what the release run does when a version arrives with no
description behind it, is #83.

Whether the built output is published anywhere is #58, and this record is
deliberately silent about it. Tagging a release and publishing to a host are
separate acts, and joining them means a bad publish cannot be undone without
deleting a tag.

## When this is worth revisiting

The bundle gaining a consumer that is not a person deciding whether to take it,
such as a package index or a client that resolves a version range. Those read a
version rather than a release note, and the parts would then have to satisfy
whatever they compare with rather than a reader.
