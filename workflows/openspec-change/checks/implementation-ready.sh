#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq gh
load_pr
require_stack archive Implementation implementation
if [ "$BASE_PR_STATE" = "MERGED" ]; then
  pass "the implementation pull request #$BASE_PR_NUMBER is merged"
fi
if [ "$BASE_PR_STATE" != "OPEN" ]; then
  fail "the implementation pull request #$BASE_PR_NUMBER is $BASE_PR_STATE, not open"
fi
if [ "$BASE_PR_DRAFT" = "true" ]; then
  fail "the implementation pull request #$BASE_PR_NUMBER is still a draft: mark it ready for review"
fi
pass "the implementation pull request #$BASE_PR_NUMBER is ready for review"
