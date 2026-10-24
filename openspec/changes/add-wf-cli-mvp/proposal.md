# Proposal

## Why

The OpenSpec process lives in prose skills that the model is trusted to follow, so nothing outside the model decides whether a step is really finished and nobody can see which step a session is on. The platform specified by the change `establish-workflow-platform` fixes that with an engine that drives agents, but it is large, and nobody has yet shown that the central idea works. This change builds the smallest thing that can prove the idea inside a harness people already use, without that platform.

## What Changes

- Add `wf`, a single Go executable that a coding agent calls from inside its own session, so the agent drives and the workflow referees.
- A workflow is a static graph of steps. A step is an optional job written for the agent plus an ordered list of checks, which are commands that report whether the step is over.
- A check decides when a step is over and where the run goes next. The agent never reports an outcome, never chooses a route, and never moves the run. The workflow only observes, and the agent makes every change.
- The agent has two commands. `wf next` runs the current step's checks, never blocks, and prints the state, the job, why the step is not done, and the exact command to run next. `wf wait` is the only command that blocks, it always has a timeout, and it polls the checks outside the model until the result changes.
- A step with a job is work. A step with no job is a wait or a gate. A failing check on a step with no job stops the run, and a pending check rests it.
- A person starts a run with `wf start`, and sees every run with `wf runs` and `wf show`. There is no command that approves, retries, cancels, or moves a run. People decide through systems that checks read, such as a pull request review.
- Runs are bound to a worktree and recorded as an append-only event log in a local directory that every display reads.
- Add hooks for Claude Code, installed by `wf install claude`: a stop hook that keeps the agent working while the workflow says there is work and lets it go idle only while a live waiter exists, a tool hook that denies tools a step does not allow, a session hook that restores the current step after compaction, and a status line.
- Ship the OpenSpec process as the first workflow, with its checks as small scripts over `git`, `gh`, and `openspec`. Verification, adversarial review, and the scope check are checks that run a headless agent and read its verdict.
- The agent asks the person questions with the harness's own blocking question tool. The workflow takes no part in conversation.

Nothing here replaces or edits `establish-workflow-platform`. This change is deliberately a proof of the model. What it shows feeds back into that platform.

## Capabilities

### New Capabilities

- `wf-workflow-format`: the YAML workflow definition, with steps, jobs, a library of named checks, routes by outcome, limits, and static validation.
- `wf-check-protocol`: how a check command is run and how its exit code and output become pass, fail, pending, broken, or a named outcome with values, including ordering and result caching by commit.
- `wf-run`: starting a run bound to a worktree, the event log, how `wf next` evaluates and advances, how `wf wait` blocks with a timeout and a heartbeat, loop limits and deadlines, and the stopped and ended states.
- `wf-run-board`: the commands that show runs, `wf runs` for every run and `wf show` for one.
- `wf-harness-hooks`: the Claude Code stop, tool, session, and status line hooks and the install command.
- `wf-openspec-change-workflow`: the OpenSpec process written as a workflow with its check scripts, and the bar this MVP must clear.

### Modified Capabilities

None. The project has no archived specs.

## Impact

- New Go module with one executable, `wf`, and no runtime dependencies beyond the commands that checks call.
- New directory of the OpenSpec workflow definition and its check scripts, which depend on `git`, `gh`, `jq`, `openspec`, and for headless checks `claude` and `codex`.
- Run state under a local directory that defaults to the home directory and can be moved with an environment variable.
- Writes hooks and a slash command into the Claude Code configuration only when the person runs the install command.
- Out of scope: external triggers, webhooks, a plugin protocol, a store protocol, a client protocol, harnesses other than Claude Code, a fresh session per step, a web view, an Agent Session Manager integration, and any command that steers a run. Each is a candidate for later work once the model is proven.
