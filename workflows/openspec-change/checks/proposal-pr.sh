#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq gh
load_pr
if [ "$PR_STATE" != "OPEN" ]; then
  fail "pull request #$PR_NUMBER for $BRANCH is $PR_STATE, not open"
fi
if ! [[ $PR_TITLE =~ ^([A-Za-z]+(\([^\)]*\))?!?:[[:space:]]+)?Proposal ]]; then
  fail "the title of pull request #$PR_NUMBER is \"$PR_TITLE\": the proposal pull request title must start with Proposal, after an optional prefix such as feat:"
fi
printf 'artifact: proposal_pr %s\n' "$PR_URL"
pass "pull request #$PR_NUMBER is open"
