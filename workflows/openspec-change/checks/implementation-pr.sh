#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq gh
load_pr
if [ "$PR_STATE" != "OPEN" ]; then
  fail "pull request #$PR_NUMBER for $BRANCH is $PR_STATE, not open"
fi
if ! title_starts_with "$PR_TITLE" Implementation; then
  fail "the title of pull request #$PR_NUMBER is \"$PR_TITLE\": the implementation pull request title must start with Implementation, after an optional prefix such as feat:"
fi
printf 'artifact: implementation_pr %s\n' "$PR_URL"
gh_json pr view --json body
recording=$(jq -r '.body // ""' <<<"$GH_RESULT" | grep -oE 'https?://[^][:space:])>"]+' | grep -iE '\.(mp4|mov|webm|gif|cast)([?#]|$)|user-attachments/assets|private-user-images|asciinema\.org/a/' | head -n 1 || true)
if [ -n "$recording" ]; then
  printf 'artifact: recording %s\n' "$recording"
fi
require_stack implementation Proposal proposal
pass "pull request #$PR_NUMBER is open and stacked on the proposal pull request #$BASE_PR_NUMBER"
