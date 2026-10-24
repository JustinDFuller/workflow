#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq gh openspec
find_change
if output=$(openspec validate "$CHANGE_NAME" --strict 2>&1); then
  pass "the change $CHANGE_NAME passes openspec validate --strict"
fi
fail "openspec validate --strict fails for the change $CHANGE_NAME:
$output"
