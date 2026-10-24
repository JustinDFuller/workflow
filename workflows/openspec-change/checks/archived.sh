#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq
find_change
case "$CHANGE_PATH" in
  openspec/changes/archive/*) pass "the change $CHANGE_NAME is archived at $CHANGE_PATH" ;;
esac
fail "the change $CHANGE_NAME is not archived: it is still at $CHANGE_PATH"
