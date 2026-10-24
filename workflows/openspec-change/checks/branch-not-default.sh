#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq gh
load_branch
load_default_branch
if [ "$BRANCH" = "$DEFAULT_BRANCH" ]; then
  fail "Invariant failed, the work must be on a branch other than the default branch: this worktree is on $BRANCH"
fi
pass "the work is on $BRANCH, not on the default branch $DEFAULT_BRANCH"
