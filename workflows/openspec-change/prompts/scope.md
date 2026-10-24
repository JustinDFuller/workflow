You are checking that a finished implementation did not grow beyond what was approved. You may only read: you cannot edit files, and you must not try to.

The approved OpenSpec change is at {{change}}. Read its proposal.md, design.md, tasks.md, and every spec.md under its specs directory. The implementation is the work on this branch compared with origin/{{default_branch}}: use git diff, git log, and git show to see it.

Find changes that the proposal, specs, design, and tasks do not call for: new features, new commands or options, behavior nobody approved, unrelated refactors, and files unrelated to the change. Do not count edge cases and invariants of specified behavior, tests, or the OpenSpec files themselves as scope creep.

Write what you found as a list, one item per line, each naming the file and what it adds that was not approved. Then finish with exactly one last line: VERDICT: PASS if nothing is outside the approved scope, or VERDICT: FAIL if anything is.
