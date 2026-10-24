#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq gh
find_change
printf 'artifact: change %s\n' "$CHANGE_PATH"
pass "the change $CHANGE_NAME is present"
