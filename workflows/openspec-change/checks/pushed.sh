#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq
load_branch
upstream=$(git rev-parse --abbrev-ref --symbolic-full-name '@{upstream}' 2>/dev/null) || upstream="origin/$BRANCH"
remote_sha=$(git rev-parse --verify --quiet "refs/remotes/$upstream") || fail "the branch $BRANCH has not been pushed: there is no $upstream"
head_sha=$(git rev-parse HEAD)
dirty=$(git status --porcelain -- openspec/changes)
if [ -n "$dirty" ]; then
  fail "there are uncommitted changes under openspec/changes:
$dirty"
fi
if [ "$head_sha" != "$remote_sha" ]; then
  ahead=$(git rev-list --count "$remote_sha..$head_sha" 2>/dev/null || echo "some")
  fail "the branch $BRANCH is not pushed: $ahead commit(s) are not on $upstream"
fi
if [ "${1:-}" = "--new-commit" ]; then
  entered=$(to_epoch "${WF_STEP_ENTERED:?}")
  committed=$(git log -1 --format=%ct HEAD)
  if [ "$committed" -le "$entered" ]; then
    fail "no new commit has been pushed since this step began at $WF_STEP_ENTERED: the latest commit is older than the review feedback"
  fi
fi
pass "the branch $BRANCH is pushed at $(git rev-parse --short HEAD)"
