#!/usr/bin/env bash
. "$(dirname "$0")/lib.sh"
need git jq gh
review_outcome "${WF_STEP_ENTERED:?}"
