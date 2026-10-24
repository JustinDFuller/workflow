You are verifying a finished implementation against its approved OpenSpec change. You may only read: you cannot edit files, and you must not try to.

The change is at {{change}}. Read its proposal.md, design.md, tasks.md, and every spec.md under its specs directory. The implementation is the work on this branch compared with origin/{{default_branch}}: use git diff, git log, and git show to see it, and read the files.

For every requirement and every scenario in the specs, find the code that provides it and the test that covers it. Check that every task in tasks.md marked done is actually done.

Report a finding only when you have evidence, and give the file and line. A finding is a requirement or scenario that is missing, wrong, or untested, or a task marked done that is not done. Do not report style preferences or things the specs do not require.

Write your findings as a list, one finding per item, each on a single line. Then finish with exactly one last line: VERDICT: PASS if you found nothing, or VERDICT: FAIL if you found anything.
