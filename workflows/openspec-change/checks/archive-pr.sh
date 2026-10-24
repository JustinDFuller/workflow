#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq gh
load_pr
if [ "$PR_STATE" != "OPEN" ]; then
  fail "pull request #$PR_NUMBER for $BRANCH is $PR_STATE, not open"
fi
if ! title_starts_with "$PR_TITLE" Archive; then
  fail "the title of pull request #$PR_NUMBER is \"$PR_TITLE\": the archive pull request title must start with Archive, after an optional prefix such as chore:"
fi
printf 'artifact: archive_pr %s\n' "$PR_URL"
require_stack archive Implementation implementation
pass "pull request #$PR_NUMBER is open and stacked on the implementation pull request #$BASE_PR_NUMBER"
