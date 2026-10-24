#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq
find_change
tasks="$CHANGE_PATH/tasks.md"
[ -f "$tasks" ] || fail "the change $CHANGE_NAME has no tasks.md at $tasks"
remaining=$(grep -E '^[[:space:]]*[-*][[:space:]]+\[ \]' "$tasks" || true)
done_count=$(grep -cE '^[[:space:]]*[-*][[:space:]]+\[[xX]\]' "$tasks" || true)
if [ -z "$remaining" ]; then
  if [ "$done_count" -eq 0 ]; then
    fail "$tasks lists no tasks"
  fi
  pass "all $done_count tasks in $tasks are complete"
fi
count=$(printf '%s\n' "$remaining" | wc -l | tr -d ' ')
noun=tasks
verb=remain
if [ "$count" -eq 1 ]; then
  noun=task
  verb=remains
fi
first=$(printf '%s\n' "$remaining" | head -n 1 | sed -E 's/^[[:space:]]*[-*][[:space:]]+\[ \][[:space:]]*//')
fail "$count $noun $verb in $tasks; the next is: $first"
