#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq claude
name=${1:?verdict.sh needs the name of a prompt}
prompt_file="${WF_WORKFLOW_DIR:?}/prompts/$name.md"
[ -f "$prompt_file" ] || broken "the prompt file $prompt_file does not exist"
find_change
load_default_branch
prompt=$(cat "$prompt_file")
prompt=${prompt//\{\{change\}\}/$CHANGE_PATH}
prompt=${prompt//\{\{default_branch\}\}/$DEFAULT_BRANCH}
short=$(git rev-parse --short HEAD)
scratch=$(mktemp -d "${TMPDIR:-/tmp}/wf-headless.XXXXXX") || broken "could not create a temporary directory"
trap 'rm -rf "$scratch"' EXIT
tools='Read,Grep,Glob,Bash(git diff:*),Bash(git log:*),Bash(git show:*),Bash(git status:*),Bash(git ls-files:*),Bash(openspec show:*),Bash(openspec list:*),Bash(openspec validate:*)'
if ! output=$(WF_HOME="$scratch/home" claude -p "$prompt" --allowedTools "$tools" --setting-sources user --no-session-persistence </dev/null 2>"$scratch/stderr"); then
  broken "the headless agent could not finish the $name check: $(cat "$scratch/stderr") $output"
fi
trimmed=$(printf '%s\n' "$output" | sed -e :a -e '/^[[:space:]]*$/{$d;N;ba' -e '}')
verdict=$(printf '%s\n' "$trimmed" | tail -n 1 | tr -d '\r')
case "$verdict" in
  "VERDICT: PASS" | "VERDICT: FAIL") ;;
  *) verdict="" ;;
esac
if [ -z "$verdict" ]; then
  broken "the headless agent gave no verdict for the $name check: its last line must be VERDICT: PASS or VERDICT: FAIL. It said: $output"
fi
case "$verdict" in
  *PASS*) pass "the $name check passed at $short" ;;
esac
findings=$(printf '%s\n' "$trimmed" | sed '$d' | sed -e :a -e '/^[[:space:]]*$/{$d;N;ba' -e '}')
if [ -z "$findings" ]; then
  findings="the headless agent reported a failure and gave no findings"
fi
fail "the $name check failed at $short:
$findings"
