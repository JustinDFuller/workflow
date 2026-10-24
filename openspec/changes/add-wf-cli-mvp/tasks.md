# Tasks

## 1. Spike: prove the Claude Code behaviors

- [x] 1.1 Build a throwaway three-step stub of `wf next` and `wf wait` in a scratch directory, and verify it prints the state, job, and next command for a step whose check fails and then passes
- [x] 1.2 Prove that Claude Code can run a command in a background shell for longer than the foreground limit and be notified when it exits, and record a session where it waits at least ten minutes without further model turns
- [x] 1.3 Prove that a stop hook can block a stop with a reason and that the agent continues, and record a session showing the block and the continued work
- [x] 1.4 Prove that a tool hook can deny a shell command that names a directory, and record the denial
- [x] 1.5 Write down what the spike found in the design under a spike results heading, and change the design if a behavior did not hold

## 2. Repo skeleton and guardrails

- [x] 2.1 Create the Go module with one `wf` executable, and verify `go build ./...` produces a binary that prints a usage message
- [x] 2.2 Add formatting, vet, and lint checks, and verify they pass on the skeleton
- [x] 2.3 Add a CI check that bans code comments in Go and shell files, and verify it fails on a deliberate comment and passes without it
- [x] 2.4 Add a CI job that builds and tests the module, and verify it is green on a pull request
- [x] 2.5 Write a short contributor guide that points at the specs and the design, and verify every command in it runs as written

## 3. Workflow format (wf-workflow-format)

- [x] 3.1 Implement the workflow types and the YAML loader that rejects unknown keys with file and line, and verify with fixture files for a valid file and an unknown key
- [x] 3.2 Implement inputs with required and default values, and verify the missing and default scenarios
- [x] 3.3 Implement steps and the check library with references, and verify the shared and unknown check scenarios
- [x] 3.4 Implement routing by outcome, the outcome-only-last rule, and the waits-need-a-deadline rule, and verify each rejection scenario
- [x] 3.5 Implement static validation for reachability, a path to a terminal, durations, and visit limits, reporting every problem, and verify the several-problems scenario
- [x] 3.6 Implement the validate command and verify it exits non-zero on a bad fixture and zero on a good one
- [x] 3.7 Manual proof: validate a good and a bad workflow from the terminal and record the messages

## 4. Check protocol (wf-check-protocol)

- [x] 4.1 Implement running a check as a shell command in the worktree root with the documented environment, and verify the environment and working directory scenarios
- [x] 4.2 Implement exit-code results and messages, including broken with standard error, and verify the pass, fail, pending, and broken scenarios
- [x] 4.3 Implement named outcomes with values and their three rejection cases, and verify each scenario
- [x] 4.4 Implement ordered evaluation with short-circuit, and verify the first-failure scenario does not run later checks
- [x] 4.5 Implement the check timeout that kills the whole process group, and verify a child process is gone after the timeout
- [x] 4.6 Implement the output limit with a cut marker, and verify with an oversized output
- [x] 4.7 Implement the commit cache keyed by commit and uncommitted changes, never caching pending or broken, and verify the four cache scenarios
- [x] 4.8 Implement artifact lines with URL or worktree-path addresses and their recording, and verify the pull request, file path, and replaced scenarios

## 5. Runs and the event log (wf-run)

- [x] 5.1 Implement the event types, the append-only log, state derivation, the state directory setting, and the per-run lock, and verify state is rebuilt from the log in a new process
- [x] 5.2 Implement the start command with worktree binding, inputs, pinning of the workflow, refusal when a run is active, and replacement of a finished run, and verify the four start scenarios
- [x] 5.3 Implement run lookup from any subdirectory and the run option, and verify the subdirectory and no-run scenarios
- [x] 5.4 Implement the next command with evaluation, recording each check's result only when it changes in the visit, cascading through passing steps, and the printed next command, and verify the work, pass, cascade, never-blocks, polling, earlier-checks-visible, and new-visit scenarios
- [x] 5.5 Implement the five states and their exit codes, and verify the exit codes are distinct and none is the usage-error code
- [x] 5.6 Implement steps with no job so a failing check stops the run, and verify the gate scenario
- [x] 5.7 Implement route values passed to the destination step, and verify review comments reach the revise step
- [x] 5.8 Implement visit limits and step timeouts that evaluate checks first, and verify the limit, review-never-arrives, and late-but-present scenarios
- [x] 5.9 Implement stopped and ended runs that run no checks, and verify the stopped scenario
- [x] 5.10 Implement the wait command with its defaults, immediate return when not pending, change detection, broken detection, and command timeout, and verify each wait scenario with a fake check
- [x] 5.11 Implement the waiter heartbeat and its removal on every exit path, and verify the live and killed-waiter scenarios
- [x] 5.12 Implement the long-wait and slow-step wording of the printed next command, and verify both scenarios
- [x] 5.13 Implement the JSON output option, and verify the output has the documented fields
- [x] 5.14 Verify two evaluations of one run at once advance it once
- [x] 5.15 Verify no command takes an outcome or a step name, by listing the commands and their flags in a test
- [x] 5.16 Manual proof: run a fixture workflow from the terminal through work, pending, a resolved wait, and end, and record it

## 6. Board (wf-run-board)

- [x] 6.1 Implement the runs command reading every run from the log, with the ended filter and the bare-command default, and verify the several-runs and ended-hidden scenarios
- [x] 6.2 Implement needs-you, waiting-on, artifacts, and time in step, and verify the review, CI, and stopped scenarios
- [x] 6.3 Implement the live view option for both the runs and show commands, and verify a step change and a check change show on the next redraw
- [x] 6.4 Implement the show command with the step list marked passed, current, or not reached, visits against maximum, and how long ago each step passed, and verify the position, done-and-left, loop-count, and when-passed scenarios
- [x] 6.5 Implement the current step detail with the job, deadline, and each check's last result including not run yet, and verify the checks-in-order, job-visible, and no-job scenarios
- [x] 6.6 Implement the artifacts list and the readable newest-first history, and verify the artifact and history scenarios
- [x] 6.7 Verify the board takes no lock by opening it while a slow fake check holds the run, and implement the JSON option for both commands
- [x] 6.8 Manual proof: record the runs board with three runs in three worktrees at different steps, and `wf show --watch` on one run as it advances

## 7. Claude Code hooks (wf-harness-hooks)

- [x] 7.1 Implement the shared fail-open behavior and the stop hook decision from the last recorded state, and verify the work, waiting, waiter, over, and broken scenarios
- [x] 7.2 Implement the stop hook progress guard and its log entry, and verify the guard and reset scenarios
- [x] 7.3 Implement the tool hook for denied tools, edits in the state directory, and shell commands naming it, and verify each scenario
- [x] 7.4 Implement the session hook for start, resume, clear, and compact, and verify the compaction and no-run scenarios
- [x] 7.5 Implement the status line command and verify the running and needs-you scenarios
- [x] 7.6 Implement the install command for Claude Code that merges hooks, status line, permission, and the start slash command into the local settings, and verify the first, second, and existing-settings scenarios
- [x] 7.7 Document what the hooks prevent and do not, in the install output and the contributor guide, and verify the install output states it
- [x] 7.8 Manual proof: record a real Claude Code session where the stop hook blocks an early stop, the tool hook denies a call, and the status line changes with the step

## 8. OpenSpec workflow, stage one (wf-openspec-change-workflow)

- [x] 8.1 Write the workflow file with the propose, proposal review, revise, and invariants steps, and verify it validates
- [x] 8.2 Write the change-present, change-valid, and pushed check scripts, and verify each against a scratch repository for both pass and fail
- [x] 8.3 Write the proposal-pull-request, CI, and ready check scripts over `gh`, make the change-present and pull-request checks report the change and proposal_pr artifacts, and verify each against a real draft pull request including the pending case and that both artifacts show on the board
- [x] 8.4 Write the proposal review check with the entered-at baseline and the three outcomes and comment values, and verify approved, old review ignored, changes requested, and closed against a real pull request
- [x] 8.5 Write the invariants check scripts and verify the hold and fail scenarios
- [x] 8.6 Verify every check script only reads, by reviewing each for commands that write
- [ ] 8.7 Manual proof: record a real Claude Code session that takes an idea through exploration with the person, a proposal pull request, a real review, and the invariants, driven only by `wf next` and `wf wait`

## 9. OpenSpec workflow, stage two (wf-openspec-change-workflow)

- [x] 9.1 Add the apply step and the tasks-complete check, and verify the tasks-remain scenario
- [x] 9.2 Add the headless verdict check with commit caching and a timeout, and verify a stub verdict maps to pass and fail and the same commit runs the agent once
- [x] 9.3 Add the verify, review, and scope steps marked slow, with findings reaching the job and scope failures stopping the run, and verify each scenario
- [x] 9.4 Add the implementation pull request step and its stacked-on-proposal check, report the implementation_pr artifact, and verify the not-stacked scenario and that the artifact shows on the board
- [x] 9.5 Add the implementation review step and the revise path through verify, and verify the approved and changes-requested scenarios
- [x] 9.6 Add the archive step and its checks, report the archive_pr artifact, and verify the archived scenario
- [x] 9.7 Make a denied GitHub or headless command report broken with job text telling the agent not to work around it, and verify the no-access scenario
- [ ] 9.8 Manual proof: record the full run on a real repository from idea to the archive pull request

## 10. Acceptance

- [ ] 10.1 Assemble the recorded demonstration with every behavior listed in the specification, link it from the implementation pull request, report it as the recording artifact, and verify each listed behavior appears
- [x] 10.2 Write down what the MVP showed about the model, including what felt wrong, in the design under a results heading, as input to `establish-workflow-platform`

## Workflow follow-up

- Archive the change after the project's review requirements are satisfied.
- Feed the results back into `establish-workflow-platform`.
