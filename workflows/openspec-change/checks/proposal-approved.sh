#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq gh
load_pr
case "$PR_DECISION" in
  APPROVED) pass "pull request #$PR_NUMBER is approved" ;;
  CHANGES_REQUESTED) fail "Invariant failed, the proposal pull request must be approved: changes are requested on $PR_URL" ;;
  REVIEW_REQUIRED) fail "Invariant failed, the proposal pull request must be approved: $PR_URL still needs a review" ;;
esac
reviews_json "$PR_NUMBER"
verdict=$(jq -r '
  group_by(.user.login) | map(sort_by(.submitted_at) | map(select(.state == "APPROVED" or .state == "CHANGES_REQUESTED")) | last)
  | map(select(. != null) | .state) | if length == 0 then "none" elif any(. == "CHANGES_REQUESTED") then "changes" else "approved" end' <<<"$REVIEWS") || broken "could not read the reviews of $PR_URL"
if [ "$verdict" = "approved" ]; then
  pass "pull request #$PR_NUMBER is approved"
fi
fail "Invariant failed, the proposal pull request must be approved: $PR_URL has no approving review"
