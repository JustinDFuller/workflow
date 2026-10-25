#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq gh
sha=$(git rev-parse HEAD)
short=$(git rev-parse --short HEAD)
ci_state "$sha"
bad=$(jq -r '[.[] | select(.state == "bad") | .name] | join(", ")' <<<"$CI_SUMMARY")
running=$(jq -r '[.[] | select(.state == "running") | .name] | join(", ")' <<<"$CI_SUMMARY")
total=$(jq -r 'length' <<<"$CI_SUMMARY")
case "$total" in
  '' | *[!0-9]*) broken "could not count the CI checks of $short" ;;
esac
if [ -n "$bad" ]; then
  fail "CI is failing at $short: $bad"
fi
if [ -n "$running" ]; then
  pending "CI is still running at $short: $running"
fi
if [ "$total" -eq 0 ]; then
  pending "CI has not reported any checks for $short yet"
fi
pass "CI is green at $short ($total checks)"
