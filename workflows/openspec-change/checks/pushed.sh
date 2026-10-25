#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq
scope=openspec/changes
new_commit=no
for arg in "$@"; do
  case "$arg" in
    --all) scope=. ;;
    --new-commit) new_commit=yes ;;
    *) broken "pushed.sh does not understand $arg" ;;
  esac
done
load_branch
upstream="origin/$BRANCH"
remote_sha=$(git rev-parse --verify --quiet "refs/remotes/$upstream") || fail "the branch $BRANCH has not been pushed: there is no $upstream"
head_sha=$(git rev-parse HEAD)
dirty=$(git status --porcelain -- "$scope" ':!.claude/commands/wf.md' ':!.claude/settings.local.json') || broken "git could not read the status of $scope"
if [ -n "$dirty" ]; then
  fail "there are uncommitted changes under $scope:
$dirty"
fi
if [ "$head_sha" != "$remote_sha" ]; then
  ahead=$(git rev-list --count "$remote_sha..$head_sha" 2>/dev/null || echo "some")
  fail "the branch $BRANCH is not pushed: $ahead commit(s) are not on $upstream"
fi
if [ "$new_commit" = yes ]; then
  entered=$(to_epoch "${WF_STEP_ENTERED:?}") || broken "could not read the step start time $WF_STEP_ENTERED"
  committed=$(git log -1 --format=%ct HEAD) || broken "git could not read the latest commit"
  if [ "$committed" -le "$entered" ]; then
    fail "no new commit has been pushed since this step began at $WF_STEP_ENTERED: the latest commit is older than the review feedback"
  fi
fi
pass "the branch $BRANCH is pushed at $(git rev-parse --short HEAD)"
