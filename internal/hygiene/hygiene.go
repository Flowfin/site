// SPDX-License-Identifier: AGPL-3.0-or-later

// Package hygiene judges the commit messages in a pull request.
//
// Two things about a commit cannot be repaired after it lands, only rewritten
// out of history: the message and the characters in it. So they are decided
// before the merge rather than after, and they are decided here rather than in
// a workflow, because a rule written in YAML has no suite to prove it bites and
// no way to run it on the machine where the mistake was made.
package hygiene

import (
	"flag"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
)

// The additions to printable ASCII, and there is exactly one. A commit message
// is separated into lines and nothing else here needs a byte outside the
// printable range. An allowlist is the only shape that also refuses a code
// point nobody has thought of yet, which is the Trojan Source problem applied
// to git metadata: the scanner that reads the tree never reads a message.
//
// A tab is deliberately not in the set. Nothing indents a commit message, and
// a tab is the one invisible byte a person plausibly pastes in by accident.
const allowedAdditions = "\n"

// reference is the shape a subject has to carry. Brackets rather than a bare
// hash so the link survives a subject read on its own, which is all `git blame`
// and a one-line log ever show.
var reference = regexp.MustCompile(`\[#[0-9]+\]`)

// botAuthors are the identities that cannot sign or shape their own messages.
// An explicit list of the automation this repository actually runs, not a glob
// over anything shaped like a bot, so a person cannot exempt themselves by
// choosing an address.
var botAuthors = []string{
	"dependabot[bot]@users.noreply.github.com",
	"github-actions[bot]@users.noreply.github.com",
}

// Origin says whether the pull request comes from this repository. A
// contribution from outside cannot know an issue number that does not exist
// yet, so the reference leg reports on it without refusing it. The character
// leg refuses either way, because a message nobody can read is a problem
// whoever wrote it.
const (
	Internal = "internal"
	External = "external"
)

type commit struct {
	sha     string
	author  string
	subject string
	body    string
}

// Run judges every non-merge commit in base..head and writes what it found to
// log. It returns an error when the range could not be walked or when a leg
// refused a commit.
func Run(args []string, log io.Writer) error {
	fs := flag.NewFlagSet("hygiene", flag.ContinueOnError)
	fs.SetOutput(log)
	origin := fs.String("origin", Internal,
		"internal or external; an external pull request is reported on rather than refused by the reference leg")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *origin != Internal && *origin != External {
		return fmt.Errorf("unknown origin %q, which is neither %s nor %s", *origin, Internal, External)
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("hygiene takes a base and a head commit, and was given %d argument(s)", fs.NArg())
	}
	base, head := fs.Arg(0), fs.Arg(1)

	commits, err := walk(base, head)
	if err != nil {
		return err
	}

	fmt.Fprintf(log, "hygiene: %d non-merge commit(s) in %s..%s, origin %s\n",
		len(commits), short(base), short(head), *origin)
	if len(commits) == 0 {
		// Not a pass. A range with nothing in it means the gate judged
		// nothing, and a green mark over that reads as a clean verdict.
		return fmt.Errorf("the range holds no non-merge commit, so nothing was judged")
	}

	failed := 0
	for _, c := range commits {
		if c.isBot() {
			fmt.Fprintf(log, "  %s: skipped, authored by %s\n", short(c.sha), c.author)
			continue
		}

		refErr := c.checkReference()
		chrErr := c.checkCharacters()

		switch {
		case refErr != nil && *origin == External:
			fmt.Fprintf(log, "  %s: reported, not refused: %v\n", short(c.sha), refErr)
		case refErr != nil:
			fmt.Fprintf(log, "  %s: FAILED: %v\n", short(c.sha), refErr)
			failed++
		default:
			fmt.Fprintf(log, "  %s: subject carries its reference\n", short(c.sha))
		}

		if chrErr != nil {
			fmt.Fprintf(log, "  %s: FAILED: %v\n", short(c.sha), chrErr)
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("hygiene: %d refusal(s) across %d commit(s)", failed, len(commits))
	}
	fmt.Fprintf(log, "%d commit(s) judged, none refused.\n", len(commits))
	return nil
}

func (c commit) isBot() bool {
	for _, b := range botAuthors {
		if c.author == b || strings.HasSuffix(c.author, "+"+b) {
			return true
		}
	}
	return false
}

// checkReference refuses a subject that carries no bracketed issue reference.
func (c commit) checkReference() error {
	if reference.MatchString(c.subject) {
		return nil
	}
	return fmt.Errorf("the subject carries no bracketed issue reference: %q", c.subject)
}

// checkCharacters refuses any code point outside printable ASCII and the named
// additions, naming the commit, the code point and the line it is on. Line
// numbers count from the subject, which is line 1, so they match what a reader
// sees running `git show -s --format=%B` on the commit.
func (c commit) checkCharacters() error {
	// Rebuilt the way `git show -s --format=%B` prints it, blank separator line
	// included, so a line number here is the line number a reader counts.
	message := c.subject
	if c.body != "" {
		message += "\n\n" + c.body
	}
	var bad []string
	line := 1
	for _, r := range message {
		if r == '\n' {
			line++
		}
		if allowed(r) {
			continue
		}
		bad = append(bad, fmt.Sprintf("U+%04X on line %d", r, line))
	}
	if len(bad) == 0 {
		return nil
	}
	return fmt.Errorf("the message carries %d code point(s) outside printable ASCII: %s",
		len(bad), strings.Join(bad, ", "))
}

func allowed(r rune) bool {
	if r >= 0x20 && r <= 0x7E {
		return true
	}
	return strings.ContainsRune(allowedAdditions, r)
}

// walk lists the non-merge commits in base..head with the fields the legs read.
// The record separator is a byte that cannot appear in a message, because the
// character leg refuses every byte outside printable ASCII and the additions,
// so a message cannot forge one and split itself into two commits.
func walk(base, head string) ([]commit, error) {
	const sep = "\x1e"
	const field = "\x1f"

	out, err := git("log", "--no-merges", "--reverse",
		"--format=%H"+field+"%ae"+field+"%s"+field+"%b"+sep, base+".."+head)
	if err != nil {
		return nil, fmt.Errorf("the range %s..%s could not be walked: %v", short(base), short(head), err)
	}

	var commits []commit
	for _, rec := range strings.Split(out, sep) {
		rec = strings.TrimLeft(rec, "\n")
		if strings.TrimSpace(rec) == "" {
			continue
		}
		parts := strings.SplitN(rec, field, 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("a log record came back in %d field(s) rather than 4", len(parts))
		}
		commits = append(commits, commit{
			sha:     parts[0],
			author:  parts[1],
			subject: parts[2],
			body:    strings.TrimRight(parts[3], "\n"),
		})
	}
	return commits, nil
}

func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
