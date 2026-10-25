#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq gh
dir=$(dirname "$0")
output=$("$dir/ci-green.sh")
status=$?
case "$status" in
  0) pass "$output" ;;
  1) fail "Invariant failed, CI must be green: $output" ;;
  *) printf '%s\n' "$output"; exit "$status" ;;
esac
