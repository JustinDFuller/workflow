# workflow

`wf` keeps a coding agent on a workflow. A workflow is a fixed graph of steps. Each step has an optional job written for the agent and a list of checks, which are commands that say whether the step is over. The agent never reports that it is done and never picks the next step. The checks decide, and `wf` moves the run.

This is an MVP that proves the idea with one harness, Claude Code, and one workflow, the OpenSpec change process. It runs on your machine and has no server.

## Prerequisites

- Go, to build `wf`.
- `git`.
- To run the OpenSpec workflow you also need `gh` signed in with the `gh stack` extension, `jq`, `npx` for the OpenSpec CLI, and `claude`, which the review checks run headless. The tour below needs only Go and `git`.

## Install

From a clone of this repository run `go install ./cmd/wf`. Make sure your Go bin directory, usually `$HOME/go/bin`, is on your `PATH`, because the hooks and the `/wf` command call `wf` by name.

Run `wf help` to see the commands.

## Five-minute tour

Make an empty repository and save this as `hello.yaml`.

```yaml
name: hello
description: Write a greeting, then wait for a person to approve it.
start: write
checks:
  has-greeting:
    run: test -s greeting.txt || { echo "greeting.txt is missing or empty"; exit 1; }
  approved:
    run: test -f approved.txt || { echo "waiting for approved.txt"; exit 2; }
steps:
  write:
    job: Create greeting.txt containing one friendly line.
    checks: [has-greeting]
    then: approve
  approve:
    checks: [approved]
    timeout: 1h
    needs_you: true
    then: end
```

A check passes with exit code 0, fails with 1, and is pending with 2. Anything else is broken. What the check prints becomes the message you see. A step with a job is work for the agent. A step with no job only waits, so its check should exit 2 until what it waits for happens, because a failing check on a step with no job stops the run.

```
git init
wf validate hello.yaml
wf start hello.yaml
echo "hi there" > greeting.txt
wf next
wf runs
wf show
touch approved.txt
wf next
```

`wf start` begins a run in this worktree and prints the first job. `wf next` runs the current checks and says what to do. After the greeting exists it moves to `approve` and reports pending. `wf runs` and `wf show` show the board. Once `approved.txt` exists, `wf next` says the run ended. In real use the agent runs `wf wait --timeout 1h` in the background instead of polling, and it comes back when the result changes.

## Use it with Claude Code

In the project you want to drive, run `wf install claude`. It writes `.claude/settings.local.json` and `.claude/commands/wf.md`. It adds a stop hook that blocks stopping while the run has work, a tool hook that denies tools a step forbids and any edit or command that touches the `wf` state directory, a session hook that restores the step after compaction, a status line, and an allowance for `Bash(wf:*)` so the agent can run `wf` without a prompt.

Then start Claude Code in a worktree that is on a branch, not the default branch, and run the OpenSpec workflow.

```
/wf /path/to/workflow/workflows/openspec-change/workflow.yaml
```

The agent explores your idea with you, writes the proposal, opens the pull request, and waits for your review. To continue an existing change add `--input change=<name>`. You decide by reviewing and approving pull requests on GitHub. There is no `wf` command that approves, retries, cancels, or moves a run.

## Watch a run

- The Claude Code status line shows the workflow, step, and whether it needs you, such as `openspec-change > proposal_review (1/10) . pending . NEEDS YOU`.
- `wf runs` lists every run that has not ended, with the ones that need you first. Add `--watch` to keep it live.
- `wf show` shows one run: each step as passed, current, or not reached, the current checks and their last results, the artifacts such as the pull request links, and a plain history. Add `--watch` to keep it live.

Runs live under `$HOME/.wf`, or `$WF_HOME` if set, as an append-only log per run. A run is never cancelled by a command. To abandon one, remove its directory under `runs` yourself.

## What the hooks do and do not prevent

They make it hard for an agent to forge a run, not impossible, because a shell command can hide a path. The log at `$WF_HOME/runs/<id>/events.jsonl` records every check result, so you can audit what was checked and what it said. See `CONTRIBUTING.md` for more.

## Where things are

- `openspec/changes/add-wf-cli-mvp/` holds the proposal, design, specs, tasks, and `qa.md`, the record of real runs.
- `workflows/openspec-change/` holds the OpenSpec workflow and its check scripts.
- `openspec/changes/establish-workflow-platform/` is the long-term plan this MVP tests: plugins, other harnesses, and a browser client.
- `CONTRIBUTING.md` explains how to build and test.
