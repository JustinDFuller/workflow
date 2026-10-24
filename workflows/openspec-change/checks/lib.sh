#!/usr/bin/env bash

pass() {
  if [ $# -gt 0 ]; then
    printf '%s\n' "$*"
  fi
  exit 0
}

fail() {
  printf '%s\n' "$*"
  exit 1
}

pending() {
  printf '%s\n' "$*"
  exit 2
}

broken() {
  printf '%s\n' "$*" >&2
  exit 3
}

need() {
  local tool
  for tool in "$@"; do
    command -v "$tool" >/dev/null 2>&1 || broken "$tool is not installed or is not on the PATH"
  done
}

gh_json() {
  local err
  err=$(mktemp "${TMPDIR:-/tmp}/wf-gh.XXXXXX") || broken "could not create a temporary file"
  if GH_RESULT=$(GH_PROMPT_DISABLED=1 NO_COLOR=1 gh "$@" 2>"$err"); then
    rm -f "$err"
    return 0
  fi
  GH_ERROR=$(cat "$err")
  rm -f "$err"
  case "$GH_ERROR" in
    *"no pull requests found"*) return 4 ;;
  esac
  broken "GitHub did not allow or could not finish 'gh $*': $GH_ERROR"
}

load_branch() {
  BRANCH=$(git symbolic-ref --quiet --short HEAD) || fail "HEAD is detached, so there is no branch"
}

load_default_branch() {
  local ref
  if ref=$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null); then
    DEFAULT_BRANCH=${ref#origin/}
    return 0
  fi
  gh_json repo view --json defaultBranchRef
  DEFAULT_BRANCH=$(jq -r '.defaultBranchRef.name' <<<"$GH_RESULT")
}

load_pr() {
  load_branch
  gh_json pr view --json number,url,title,state,isDraft,baseRefName,headRefName,reviewDecision
  local status=$?
  if [ "$status" -eq 4 ]; then
    fail "no pull request exists for the branch $BRANCH"
  fi
  PR_NUMBER=$(jq -r '.number' <<<"$GH_RESULT")
  PR_URL=$(jq -r '.url' <<<"$GH_RESULT")
  PR_TITLE=$(jq -r '.title' <<<"$GH_RESULT")
  PR_STATE=$(jq -r '.state' <<<"$GH_RESULT")
  PR_DRAFT=$(jq -r '.isDraft' <<<"$GH_RESULT")
  PR_BASE=$(jq -r '.baseRefName' <<<"$GH_RESULT")
  PR_HEAD=$(jq -r '.headRefName' <<<"$GH_RESULT")
  PR_DECISION=$(jq -r '.reviewDecision // ""' <<<"$GH_RESULT")
}

to_epoch() {
  jq -nr --arg t "$1" '$t | fromdateiso8601'
}

change_names() {
  local base
  load_default_branch
  base=$(git merge-base HEAD "origin/$DEFAULT_BRANCH" 2>/dev/null || git merge-base HEAD "$DEFAULT_BRANCH" 2>/dev/null || git hash-object -t tree /dev/null)
  {
    git diff --name-only "$base" 2>/dev/null
    git ls-files --others --exclude-standard 2>/dev/null
  } | sed -n -E \
    -e 's#^openspec/changes/archive/[0-9]{4}-[0-9]{2}-[0-9]{2}-([^/]+)/.*#\1#p' \
    -e 's#^openspec/changes/([^/]+)/.*#\1#p' | grep -v '^archive$' | sort -u
}

find_change() {
  local names count
  CHANGE_NAME=""
  CHANGE_PATH=""
  if [ -n "${WF_INPUT_CHANGE:-}" ]; then
    CHANGE_NAME=$WF_INPUT_CHANGE
  else
    names=$(change_names)
    if [ -z "$names" ]; then
      fail "no OpenSpec change is present on this branch: nothing under openspec/changes differs from the default branch"
    fi
    count=$(printf '%s\n' "$names" | wc -l | tr -d ' ')
    if [ "$count" -gt 1 ]; then
      fail "$count OpenSpec changes differ from the default branch ($(printf '%s' "$names" | tr '\n' ' ')): start the run with --input change=<name> to say which one this run is about"
    fi
    CHANGE_NAME=$names
  fi
  if [ -d "openspec/changes/$CHANGE_NAME" ]; then
    CHANGE_PATH="openspec/changes/$CHANGE_NAME"
    return 0
  fi
  CHANGE_PATH=$(ls -d openspec/changes/archive/*-"$CHANGE_NAME" 2>/dev/null | sort | tail -n 1)
  if [ -z "$CHANGE_PATH" ]; then
    fail "the OpenSpec change $CHANGE_NAME has no directory under openspec/changes"
  fi
}

ci_state() {
  local sha=$1
  gh_json api "repos/{owner}/{repo}/commits/$sha/check-runs?per_page=100"
  CI_RUNS=$GH_RESULT
  gh_json api "repos/{owner}/{repo}/commits/$sha/status"
  CI_STATUS=$GH_RESULT
  CI_SUMMARY=$(jq -n --argjson runs "$CI_RUNS" --argjson status "$CI_STATUS" '
    [ $runs.check_runs[]
      | { name: .name,
          state: (if .status != "completed" then "running"
                  elif (.conclusion | IN("success", "neutral", "skipped")) then "ok"
                  else "bad" end) } ]
    + [ $status.statuses[]
      | { name: .context,
          state: (if .state == "success" then "ok"
                  elif .state == "pending" then "running"
                  else "bad" end) } ]')
}

reviews_json() {
  gh_json api --paginate "repos/{owner}/{repo}/pulls/$1/reviews?per_page=100"
  REVIEWS=$(jq -s 'add // []' <<<"$GH_RESULT")
}

review_comments_text() {
  local pr=$1 review=$2
  gh_json api --paginate "repos/{owner}/{repo}/pulls/$pr/reviews/$review/comments?per_page=100"
  jq -rs 'add // [] | map("- " + .path + (if .line then ":" + (.line | tostring) else "" end) + ": " + .body) | join("\n")' <<<"$GH_RESULT"
}

outcome() {
  jq -n --arg outcome "$1" --arg comments "${2:-}" '
    if $comments == "" then { outcome: $outcome } else { outcome: $outcome, values: { comments: $comments } } end'
  exit 0
}

review_outcome() {
  local entered_at=$1 decisive latest state id login body inline text
  load_pr
  case "$PR_STATE" in
    CLOSED) outcome closed ;;
    MERGED) outcome approved ;;
  esac
  reviews_json "$PR_NUMBER"
  decisive=$(jq -c --arg since "$entered_at" '
    [ .[] | select((.state == "APPROVED" or .state == "CHANGES_REQUESTED") and .submitted_at > $since) ]
    | sort_by(.submitted_at) | last // empty' <<<"$REVIEWS")
  if [ -z "$decisive" ]; then
    pending "no approving or change-requesting review has been submitted on $PR_URL since $entered_at"
  fi
  state=$(jq -r '.state' <<<"$decisive")
  if [ "$state" = "APPROVED" ]; then
    outcome approved
  fi
  id=$(jq -r '.id' <<<"$decisive")
  login=$(jq -r '.user.login' <<<"$decisive")
  body=$(jq -r '.body' <<<"$decisive")
  inline=$(review_comments_text "$PR_NUMBER" "$id")
  text="Review from $login on $PR_URL:"
  if [ -n "$body" ]; then
    text="$text
$body"
  fi
  if [ -n "$inline" ]; then
    text="$text

Comments on the code:
$inline"
  fi
  outcome changes_requested "$text"
}
