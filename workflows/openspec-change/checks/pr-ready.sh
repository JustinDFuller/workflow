#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq gh
load_pr
if [ "$PR_DRAFT" = "true" ]; then
  fail "pull request #$PR_NUMBER is still a draft: mark it ready for review"
fi
pass "pull request #$PR_NUMBER is ready for review"
