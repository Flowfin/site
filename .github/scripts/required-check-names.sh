#!/usr/bin/env bash
#
# A required status check is a string. What satisfies it is a check run whose
# name is that same string, and nothing connects the two ends. Rename a job, add
# a `name:` where a job had none, narrow a trigger so it no longer covers pull
# requests, and the required context is never reported at all. A ruleset reads
# never reported as never satisfied, so the result is not a red check. It is a
# pull request that waits with nothing to click and no message saying why.
#
# This reads both ends and compares them. It holds neither: the required names
# come from the rules that apply to the default branch, and the reported names
# come from the check runs on the head commit of a pull request that really was
# merged. A list written here would be a third copy that drifts against both.
#
# It lives in a file rather than inside the workflow that calls it because every
# state it can end in has to be reproducible by somebody who is not inside
# Actions. Point it at a repository and a pull request and it does the same
# thing on a laptop that it does on a schedule:
#
#     REPO=owner/name PR=7 .github/scripts/required-check-names.sh
#
# REPO is required. PR is optional and defaults to the most recently merged pull
# request against the default branch.
#
# Four states are not a pass, and each says which one it is. A list that could
# not be read. A repository where no pull request has ever been merged. A branch
# whose rules require no context at all. And a required context that is not among
# the reported names, which is the failure the whole file is for. The first three
# have nothing to compare, and a green mark over a comparison that did not happen
# is the thing this exists to prevent.
#
# The other direction is information rather than a failure. A check run that
# reports and is required by nothing is a decision somebody took, so the set is
# printed and the run passes.

set -euo pipefail

# One collation for the whole run. `comm` refuses input its own locale would
# have ordered differently, and a check name is arbitrary text: a space, a
# bracket and a mixture of cases all land in different places under a locale
# that sorts by word rather than by byte. Sorting in one locale and comparing in
# another is how a set difference comes out wrong while both lists look right.
export LC_ALL=C

REPO=${REPO:-}
PR=${PR:-}

if [ -z "$REPO" ]; then
  echo "::error::REPO is empty. Set it to the owner/name of the repository to read."
  exit 1
fi

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# gh writes an empty line for an empty result, and comm reads that as a name.
# Every list goes through here so an empty one is an empty file.
lines_to() {
  sed '/^$/d' | sort -u >"$1"
}

if ! branch=$(gh api "repos/${REPO}" --jq '.default_branch'); then
  echo "::error::Could not read the default branch of ${REPO}. Failing rather than comparing against a branch nobody named."
  exit 1
fi
echo "repository:     ${REPO}"
echo "default branch: ${branch}"
echo

# The rules that apply to the branch rather than the ruleset list. This endpoint
# needs no administrative scope, and it answers with everything that reaches the
# branch, so a context required through a second ruleset or through a classic
# protection rule is in the answer instead of being missed by a reader who only
# knew about one of them.
if ! required_raw=$(gh api "repos/${REPO}/rules/branches/${branch}" \
  --jq '.[] | select(.type == "required_status_checks") | .parameters.required_status_checks[].context'); then
  echo "::error::Could not read the rules on ${REPO}@${branch}. Failing closed: a run that cannot see what is required cannot report that everything required is reported."
  exit 1
fi
printf '%s\n' "$required_raw" | lines_to "${work}/required"

echo "required contexts, from the rules on ${branch}:"
if [ -s "${work}/required" ]; then
  sed 's/^/    /' "${work}/required"
else
  echo "    (none)"
fi
echo

# Which pull request. An explicit number is taken as given and is refused if it
# was closed without merging, because an unmerged head is not evidence of what
# reports on the way in. Otherwise the newest merged one, ordered by when it
# merged rather than by when it was last touched, so an old pull request that
# somebody commented on today does not become the subject.
if [ -n "$PR" ]; then
  if ! merged=$(gh api "repos/${REPO}/pulls/${PR}" \
    --jq 'select(.merged_at != null) | [.number, .merged_at, .head.sha] | @tsv'); then
    echo "::error::Could not read pull request ${PR} on ${REPO}."
    exit 1
  fi
  if [ -z "$merged" ]; then
    echo "::error::Pull request ${PR} on ${REPO} was not merged. Nothing was compared."
    exit 1
  fi
else
  if ! merged=$(gh api "repos/${REPO}/pulls?state=closed&base=${branch}&sort=updated&direction=desc&per_page=100" \
    --jq '[.[] | select(.merged_at != null)] | sort_by(.merged_at) | reverse | .[0] // empty | [.number, .merged_at, .head.sha] | @tsv'); then
    echo "::error::Could not read the closed pull requests on ${REPO}@${branch}."
    exit 1
  fi
fi

if [ -z "$merged" ]; then
  echo "No pull request has been merged into ${branch} on ${REPO}, so there is no head commit carrying check runs and nothing was compared."
  echo "What ends this state is the first merge into ${branch}. Until then this run has nothing to say about whether the required names and the reported names agree, and it says that rather than passing."
  exit 1
fi

IFS=$'\t' read -r number merged_at head_sha <<<"$merged"
echo "subject:        pull request ${number}, merged ${merged_at}"
echo "head commit:    ${head_sha}"
echo

if ! reported_raw=$(gh api "repos/${REPO}/commits/${head_sha}/check-runs?per_page=100" --paginate \
  --jq '.check_runs[].name'); then
  echo "::error::Could not read the check runs on ${head_sha}. Failing closed: a run that cannot see what reported cannot report that everything required reported."
  exit 1
fi
printf '%s\n' "$reported_raw" | lines_to "${work}/reported"

echo "reported names, from the check runs on that commit:"
if [ -s "${work}/reported" ]; then
  sed 's/^/    /' "${work}/reported"
else
  echo "    (none)"
fi
echo

if [ ! -s "${work}/required" ]; then
  echo "The rules on ${REPO}@${branch} require no status check, so there is no name for a reported one to satisfy and nothing was compared."
  echo "What ends this state is a required status check on that branch. Until then this run has nothing to say about whether the two lists agree, and it says that rather than passing."
  exit 1
fi

comm -23 "${work}/required" "${work}/reported" >"${work}/missing"
comm -13 "${work}/required" "${work}/reported" >"${work}/extra"

if [ -s "${work}/extra" ]; then
  echo "reported and required by nothing, which is information rather than a failure:"
  sed 's/^/    /' "${work}/extra"
  echo
fi

if [ -s "${work}/missing" ]; then
  echo "required and not among the reported names:"
  sed 's/^/    /' "${work}/missing"
  echo
  echo "::error::$(wc -l <"${work}/missing" | tr -d ' ') required context(s) on ${REPO}@${branch} were not reported on the head of pull request ${number}. A pull request needing one of these waits with nothing to click. Either the rule names something that no longer reports under that name, or the job that reports it stopped running on pull requests."
  exit 1
fi

echo "Every required context is among the reported names."
