# QA record for add-wf-cli-mvp

This file records the manual proof for the change: real commands, run from a terminal against the built `wf` binary. In the output below, `<scratch>` stands for a temporary directory. Tests and linters are not counted as proof here.

## Spike (tasks 1.1 to 1.4)

A throwaway bash stub of `wf` and a headless Claude Code session (`claude -p`) in a scratch project showed these behaviors. The design records the conclusions under "Spike results".

- Stop hook: a `Stop` hook returned a block decision with a reason, the agent kept working, and it stopped only after the stub said the run had ended.
- Tool hook: a `PreToolUse` hook on `Bash` denied a command that named the state directory. Claude Code printed "PreToolUse:Bash hook error: The workflow state cannot be touched by the agent." and the command did not run.
- Background wait: a `wf wait --timeout 1h` started without a Bash tool timeout was killed after exactly ten minutes. The same wait started with an explicit timeout of 3700000 milliseconds ran, and its exit woke the headless session with a task notification. A 720 second sleep started with an explicit timeout of 1800000 milliseconds finished after 740 seconds, which shows an explicit timeout is honored past ten minutes.

## Workflow validation (tasks 3.7)

```
$ wf validate internal/workflow/testdata/valid.yaml
internal/workflow/testdata/valid.yaml is valid
exit 0

$ wf validate internal/cli/testdata/several_problems.yaml
internal/cli/testdata/several_problems.yaml:10: no path leads from the start step to end or stop
internal/cli/testdata/several_problems.yaml:12: step "write" lists check "missing" which is not in the check library
internal/cli/testdata/several_problems.yaml:13: step "write" routes to "nowhere" which is not a step, end, or stop
internal/cli/testdata/several_problems.yaml:14: step "orphan" cannot be reached from the start step
internal/cli/testdata/several_problems.yaml:18: step "wait" has no job and no timeout; a step that waits needs a timeout
internal/cli/testdata/several_problems.yaml:18: step "wait" cannot be reached from the start step
6 problem(s) in internal/cli/testdata/several_problems.yaml
exit 1
```

## A run from start to end, with the status line (tasks 5.16 and 7.5)

A two-step workflow where the second step waits on a file that stands in for an approval. The `statusline:` lines are the output of `wf statusline`, which Claude Code runs and shows in its footer, given the JSON that Claude Code sends it (`{"cwd": ...}`).

```
$ wf start flow.yaml
Run 4463804f (notes) in <scratch>/qa/repo
Step: draft (visit 1 of 10)
State: work
Job:
Create notes.txt containing a line about the project.
Not done yet: notes.txt is missing
Next: wf next
exit 0

statusline: notes > draft (1/10) . work

$ wf next
Run 4463804f (notes) in <scratch>/qa/repo
Passed: draft
Step: review (visit 1 of 10)
State: pending
Job: none. This step waits for its checks.
Waiting on: waiting for approval
Next: run this in the background with the Bash tool timeout set to 3659997 milliseconds: wf wait --timeout 1h
exit 10

statusline: notes > review (1/10) . pending . NEEDS YOU

$ wf wait --timeout 1m
Run 4463804f (notes) in <scratch>/qa/repo
Passed: review
State: ended
The workflow is finished. There is nothing more to do.
exit 13

statusline: notes . ended
```

## Runs board and live show (tasks 6.8)

Three runs in three worktrees at different steps, then `wf show` for one of them. The runs are listed with the one that needs the person first.

```
f9947a31  openspec-change  PENDING  NEEDS YOU
  worktree   <scratch>/board/gamma
  step       proposal_review (7s in this step)
  waiting on no review submitted yet
  artifacts  change=change  proposal_pr=https://github.com/example/repo/pull/12

6b854341  openspec-change  PENDING  does not need you
  worktree   <scratch>/board/beta
  step       propose (7s in this step)
  waiting on 2 checks running
  artifacts  change=change  proposal_pr=https://github.com/example/repo/pull/12

cef29121  openspec-change  WORK  does not need you
  worktree   <scratch>/board/alpha
  step       propose (7s in this step)
  working on no change directory yet

openspec-change  run f9947a31  <scratch>/board/gamma  PENDING  NEEDS YOU
started 7s ago  in this step 7s

Steps
  [x] propose          1 of 10   passed 7s ago
  [>] proposal_review  1 of 10   current

Now: proposal_review
Job: none. This step waits for its checks.
Deadline: 13d23h59m52s left
Checks
  pending      reviewed: no review submitted yet

Artifacts
  change         change
  proposal_pr    https://github.com/example/repo/pull/12

History
  18:32  reviewed is pending: no review submitted yet
  18:32  propose passed, moved to proposal_review
  18:32  ci-green passed
  18:32  artifact proposal_pr reported: https://github.com/example/repo/pull/12
  18:32  pr-open passed
  18:32  artifact change reported: change
  18:32  change-present passed
  18:32  run started
```

`wf show --watch --interval 1s` redrew as the run `alpha` advanced. These are the frames, taken from the terminal output:

```
--- redraw 0 ---
openspec-change  run cef29121  <scratch>/board/alpha  WORK  does not need you
started 7s ago  in this step 7s

Steps
  [>] propose          1 of 10   current
  [ ] proposal_review            not reached

Now: propose
Job:
  Write the change, open the proposal pull request, and make CI pass.
Checks
  fail         change-present: no change directory yet
  not run yet  pr-open
  not run yet  ci-green
--- redraw 5 ---
openspec-change  run cef29121  <scratch>/board/alpha  WORK  does not need you
started 12s ago  in this step 12s

Steps
  [>] propose          1 of 10   current
  [ ] proposal_review            not reached

Now: propose
Job:
  Write the change, open the proposal pull request, and make CI pass.
Checks
  pass         change-present
  fail         pr-open: no proposal pull request yet
  not run yet  ci-green
--- redraw 6 ---
openspec-change  run cef29121  <scratch>/board/alpha  PENDING  does not need you
started 13s ago  in this step 13s

Steps
  [>] propose          1 of 10   current
  [ ] proposal_review            not reached

Now: propose
Job:
  Write the change, open the proposal pull request, and make CI pass.
Checks
  pass         change-present
  pass         pr-open
  pending      ci-green: 2 checks running
--- redraw 8 ---
openspec-change  run cef29121  <scratch>/board/alpha  PENDING  NEEDS YOU
started 15s ago  in this step 0s

Steps
  [x] propose          1 of 10   passed 0s ago
  [>] proposal_review  1 of 10   current

Now: proposal_review
Job: none. This step waits for its checks.
Deadline: 13d23h59m59s left
Checks
  pending      reviewed: no review submitted yet

Artifacts
--- redraw 9 ---
openspec-change  run cef29121  <scratch>/board/alpha  PENDING  NEEDS YOU
started 16s ago  in this step 1s

Steps
  [x] propose          1 of 10   passed 1s ago
  [>] proposal_review  1 of 10   current

Now: proposal_review
Job: none. This step waits for its checks.
Deadline: 13d23h59m58s left
Checks
  pending      reviewed: no review submitted yet

Artifacts
--- redraw 10 ---
openspec-change  run cef29121  <scratch>/board/alpha  PENDING  NEEDS YOU
started 17s ago  in this step 2s

Steps
  [x] propose          1 of 10   passed 2s ago
  [>] proposal_review  1 of 10   current

Now: proposal_review
Job: none. This step waits for its checks.
Deadline: 13d23h59m57s left
Checks
  pending      reviewed: no review submitted yet

Artifacts
```

## Hooks in a real Claude Code session (task 7.8)

`wf install claude` ran in a scratch git repository, then a headless Claude Code session (`claude -p`, Sonnet 5.5) was told to start the run, try to read the state directory, say it was done, and stop. The workflow had two steps: create `notes.txt`, then add a line containing the word summary.

The install printed each change it made and the limits of enforcement:

```
$ wf install claude
Installed for Claude Code in <scratch>/qa/inst:
  added the Stop hook (wf hook-stop)
  added the PreToolUse hook (wf hook-tool)
  added the SessionStart hook (wf hook-session)
  set the status line (wf statusline)
  allowed Bash(wf:*) without a prompt
  added the /wf slash command (.claude/commands/wf.md)
What the hooks do: they block a stop while the run has work left, deny the tools a step lists as denied, and deny edits and shell commands that touch the wf state directory.
What they do not do: they make it hard for an agent to forge a run, not impossible, because a shell command can hide a path. The log records every check result, so a person can audit what was checked and what it said.
exit 0
```

The tool hook denied the command that named the state directory:

```
TOOL:   ls $WF_HOME/runs
RESULT: PreToolUse:Bash hook error: The workflow state cannot be touched by the agent.
AGENT:  A hook blocked that command, so I couldn't read the referee's state. I won't try another way to read it.
        All done.
```

The agent said it was done while the first step was still failing. The stop hook blocked the stop and fed this back to the agent, which then did the work:

```
Stop hook feedback:
Run 6f82d5c6 (notes) in <scratch>/hooks7/repo
Step: draft (visit 1 of 10)
State: work
Job:
Create notes.txt containing a line about the project.
Not done yet: notes.txt is missing
Next: wf next
```

The agent then wrote `notes.txt`, ran `wf next`, appended the summary line, ran `wf next` again, and the run ended. The log shows only checks moving the run (times are UTC):

```
22:39:26 started
22:39:26 step_entered draft
22:39:26 check_result draft has-notes fail notes.txt is missing
22:39:32 check_result draft has-notes pass
22:39:32 step_passed draft
22:39:32 step_entered summarize
22:39:32 check_result summarize has-summary fail notes.txt has no summary line
22:39:34 check_result summarize has-summary pass
22:39:34 step_passed summarize
22:39:34 ended summarize
```

Limit of this proof: Claude Code draws the status line only in an interactive session, and this session was headless. The status line lines above are the output of the same command with the same input shape. A person running an interactive session sees them in the footer.

## Findings

- The security review of the install command noted that the permission `Bash(wf:*)` lets the agent run `wf start` and `wf next` without a prompt, and those run the check commands in the workflow file. The specification requires this permission, so the install writes it. The workflow file is a trusted input that the person chooses, and an agent that can write a new workflow file and start it could run commands through it. Narrowing the permission is a decision for the person and is listed as a follow-up.

## Stage one of the OpenSpec workflow

The workflow file `workflows/openspec-change/workflow.yaml` validates, and its eleven check scripts live in `workflows/openspec-change/checks/`. Each script was run three ways: through Go tests against a scratch git repository with a stand-in `gh`, against the real pull requests of this repository read-only, and through `wf` in a scratch repository with the stand-in `gh`.

What ran against real pull requests of this repository: the change-present, change-valid, pushed, proposal pull request, CI, ready, and review checks against the approved proposal pull request, and the title, draft, and CI checks against the draft implementation pull request. The proposal review reported `approved` for the real approval when the baseline was older than it, and was pending for a baseline in the future. The CI check named the failing prose check on a real red commit.

What could not run against a real pull request: a request for changes, a closed pull request, and a denied GitHub call. The agent account cannot review its own pull request, and closing the person's pull requests to test would be a side effect. Those three paths ran against the stand-in `gh`, which serves the same JSON shapes that the real commands returned.

A run in a scratch repository, driven only by `wf next` with the stand-in `gh`, showed these moves in order. The change is written but not pushed, then pushed with no pull request, then a draft with the wrong title, then CI running (pending, naming the running check), then green but still a draft, then ready (the run enters `proposal_review` and the board lists `change` and `proposal_pr`). After a request for changes the run enters `revise` and shows the comments, stays there until a newer commit is pushed, goes back to `proposal_review` once CI is green, and ends through `invariants` after an approval.

```
  [x] propose          1 of 3    passed 18s ago
  [x] proposal_review  2 of 10   passed 3s ago
  [x] revise           1 of 5    passed 3s ago
  [x] invariants       1 of 10   passed 2s ago

Artifacts
  change         openspec/changes/add-thing
  proposal_pr    https://example.test/pull/12
```

The failure paths, also through `wf next`:

```
Reason: Invariant failed, CI must be green: CI is failing at 76b86e3: go
Reason: step "proposal_review" routed the run to stop
A check could not run: check "proposal-review" exited with code 3
GitHub did not allow or could not finish 'gh pr view ...': HTTP 403: Resource not accessible by integration
```

The first is a CI that turned red after the approval and stopped the run at `invariants`. The second is a pull request closed without merging. The third and fourth are a denied GitHub call, which leaves the run in the broken state with the status line `openspec-change > proposal_review (1/10) . broken . NEEDS YOU`.

Findings from this part. The closed outcome stops the run, but the stop reason names the step and not the closed pull request, because the engine writes that reason. A reviewer who wants the cause in the stop message needs a spec change to routing. Also, `wf statusline` waits for its input to close, so it must be run with its input closed when tried by hand.
