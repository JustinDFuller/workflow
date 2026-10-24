## Context

See proposal.md for why. The active change `establish-workflow-platform` specifies an engine that drives agent harnesses through plugins. This change does not edit it. It tests the opposite arrangement on the smallest possible surface: the harness drives, and a command-line tool referees. Whatever this proves or disproves is evidence for that platform.

This design is for an MVP whose job is to prove an idea, not to ship a product. Where a choice trades robustness for speed to a first recording, it takes speed and says so.

```
  person: wf start / wf runs / wf show
              |
              v
  +-----------------------------+        +-------------------+
  | Claude Code session (agent) |<------>|  person via       |
  |  - does the job             |        |  AskUserQuestion  |
  |  - runs wf next / wf wait   |        +-------------------+
  +-----------------------------+
        |   ^            ^
 hooks  |   | text       | stop / tool / session / statusline
        v   |            |
  +-----------------------------+      reads      +-----------+
  |            wf               |---------------->| git, gh,  |
  |  next, wait, hooks, board   |  (check scripts)| openspec  |
  +-----------------------------+                 +-----------+
        |
        v
  ~/.wf/runs/<id>/events.jsonl   (the only state)
```

## Goals / Non-Goals

**Goals:**

- Show that a session stays on a workflow and finishes it when only checks can move the run.
- Show that waits of any length cost the model nothing and survive a closed laptop or a long gap.
- Show a person where every run is, and whether it needs them, from one log.
- Run the real OpenSpec process end to end, at least through an approved proposal.

**Non-Goals:**

- Stopping a determined agent from cheating. The hooks make forging a run hard, not impossible.
- Any process that stays alive between agent calls, a network protocol, or a plugin interface.
- Support for a harness other than Claude Code.
- A graphical view. The board is a terminal command.

## Decisions

### A single Go binary with no daemon

`wf` is a Go program with the standard library plus a YAML parser. Every command is a short process that reads the log, does one thing, and exits. The only long-lived process is the wait command, which the agent starts in its own background shell.

Alternatives: a local daemon would give push updates and a shared lock, but it needs lifecycle, ports, and recovery, which are the costs this MVP exists to avoid. A script in shell and `jq` would be quicker to start, but the state machine, the lock, and the validation want real types and tests.

### CLI first, no MCP server

Hooks can only call commands, so a CLI is required anyway. The agent can call it through its shell tool with no extra setup. An MCP wrapper over the same commands can come later if a harness benefits from typed tools.

### Checks are shell commands with a four-way exit code

A check is any command. Exit 0 passes, 1 fails with a message on standard output, 2 is pending, and anything else is broken. This lets a check be a five-line script and makes broken, which means the check could not run, different from fail, which means the work is not done. The spec for outcomes adds JSON on standard output for the few checks that route, such as reviews.

Alternatives: a plugin protocol was rejected for this MVP because it adds a contract to design before the idea is proven. Checks as in-process Go functions were rejected because the OpenSpec workflow should be editable without rebuilding.

### Append-only event log as the only state

Each run is a directory with a JSON Lines file and a lock file. Current state is a fold over the log, so there is no second copy to drift. The board, the status line, and the hooks all read the same file. A result is appended only when it differs from the last one for that visit, so a wait that polls for fourteen days appends a handful of lines.

Alternatives: SQLite gives queries the MVP does not need and a dependency it does not want. A single mutable state file is simpler to read but loses the history that the show command and an audit need.

### The agent never steers

No command takes an outcome or a step. The run moves only because a check passes or reports an outcome, or because a visit limit or deadline stops it. A person steers through the systems the checks read, such as a pull request review, or by editing the workflow and starting a new run. This is what makes the log trustworthy: every transition has a check result behind it.

### Stopped is terminal and a new start replaces it

There is no retry. When a limit or a gate stops a run, the person changes what caused the stop and starts a new run in the same worktree, and the old run stays in history. This keeps the number of states small and the log honest.

### Evaluate checks before enforcing deadlines

When an evaluation finds a step's deadline passed, it still runs the checks first, so a review that arrived a minute late advances the run instead of stopping it.

### Waiting is explicit, short by default, and background when long

Asking never blocks, and the only blocking command has a timeout. Its default of ninety seconds sits under the harness's usual foreground command limit. When the state is pending, the printed command is the wait command with the time left on the step, and it tells the agent to run it in the background. Claude Code can wait on a background shell without spending tokens, and the next evaluation re-checks whatever the waiter could not.

A live waiter is a heartbeat file touched at each poll and removed on exit, with a lifetime of twice the interval, so a killed waiter stops counting without any cleanup.

### The stop hook is the enforcement point

The stop hook reads the last recorded state and blocks the stop while work remains or while the run is pending with no live waiter. It never evaluates checks, so it is fast and cannot fail because GitHub is down. If the recorded state is stale because the agent edited files since the last evaluation, the worst case is one extra stop, because the next evaluation corrects it.

A progress guard allows the stop after five blocks in a row with no new event. Without it, a check that can never pass would trap the session forever. With it, the person sees an idle agent and the log shows why.

### All hooks fail open

A hook that errors, or a run that cannot be read, allows the action. A broken referee must never lock a person out of their own session.

### Headless checks for judgment

Verification, adversarial review, and the scope check need judgment. They run as checks that start a headless agent (`claude -p` or `codex exec`), read a verdict, and are cached by commit and marked slow. This keeps the model inside the session honest by asking a second one, and it brings back review on a different harness. They run after the cheap checks, so they never run on unpushed work.

### Review wait uses an entered-at baseline

A review check receives the time the step was entered and only counts reviews submitted after it. This stops a stale approval from satisfying a later wait. GitHub's review state is the source, and the check does not trust the agent's report.

### No fresh session per step

The platform change gives each step a fresh agent session. This MVP keeps one session for the whole run and relies on the session hook to restore the step after compaction. Fresh context would be cleaner but needs a runner, and that is what this MVP omits. Judgment-heavy steps get fresh context through headless checks and subagents.

### The plan gate is not enforced

Exploring the idea with the person and agreeing on a plan happen inside the propose job through the harness's question tool, so nothing outside the session can prove they happened. The enforced human gate is the proposal pull request review. This is a known gap that this MVP accepts.

## Risks / Trade-offs

- [A long single session degrades or compacts] → The workflow is the memory, and the session hook restores the step, job, and next command.
- [A background wait of days may not survive in Claude Code] → The spike group proves it first. If it does not, the pending text falls back to repeated short waits, which keep the same checks and cost more turns.
- [A slow check can pass the foreground command limit] → Slow steps tell the agent to run the next command in the background, and every check has a timeout that kills its process group and reports broken.
- [The agent forges the log or the state directory] → The tool hook denies edits and shell commands that name the state directory, and the log keeps every check result. This is best effort. A shell command can obscure a path, so a determined agent can get around it.
- [The agent runs checks by hand and acts on the answer] → That is allowed and harmless, because only evaluation by the workflow records results.
- [Two evaluations race] → A per-run lock serializes them, and an evaluation re-reads the log inside the lock.
- [A stale cache hides a change] → The cache key is the commit plus a hash of the uncommitted diff, and pending and broken are never cached.
- [Headless checks cost money and time] → They run last, are cached by commit, and have a timeout.
- [The workflow works only for the author's process] → Accepted. The MVP is judged on one real process. The format is the part that must generalize, and the second workflow will show whether it does.
- [GitHub rate limits during long waits] → The default interval is thirty seconds, and a single `gh` call per poll is enough for the review checks.

## Migration Plan

There is nothing to migrate. The change adds a new executable and a new directory of workflow files, and it writes into the Claude Code settings only when a person runs the install command. To remove it, delete the hooks the install command lists and the state directory.

Delivery order: a spike that proves the three Claude Code behaviors this design depends on, then the core and the board, then the hooks, then stage one of the workflow with its recording, then stage two.

## Open Questions

- Whether the shepherd steps, merging and releasing, belong at the end of this workflow or in a separate one. The MVP stops at the archive pull request.
- Whether Codex and Cursor support stop hooks and long background waits. The MVP proves Claude Code only.
- Whether an Agent Session Manager view should read the log directly or through a small API. Nothing here depends on the answer.

## Spike results

The spike ran against a throwaway bash stub of `wf` and a headless Claude Code (`claude -p`) in a scratch project. Each behavior below was observed, not assumed.

- A stop hook can block a stop. The hook returned `{"decision":"block","reason":...}`, the agent kept working and ran the stub's next command, and it stopped only after the stub reported the run ended.
- A tool hook can deny a shell command. A `PreToolUse` hook on `Bash` that read `.tool_input.command` from stdin and returned a `deny` decision made Claude Code show "PreToolUse:Bash hook error: The workflow state cannot be touched by the agent." and the command did not run.
- A background shell outlives the foreground limit, with one catch. Claude Code kills a background Bash command after ten minutes by default. A `wf wait --timeout 1h` was killed at exactly ten minutes. The same command started with an explicit Bash tool `timeout` of 3700000 milliseconds kept running, and a 720-second sleep started with an explicit 1800000 millisecond timeout finished after 740 seconds. So an explicit timeout is honored past ten minutes.
- Exit of a background shell wakes the agent. The task-completion notification started a new model turn in the headless session with no other input, and the exit code (13 for an ended run) was visible to the agent. A headless session stays alive while a background task runs.

What changed in the design because of the spike. The pending text already carries the wait command with the time left on the step. It now also tells the agent to set the Bash tool timeout to the wait time plus one minute, because without it a long wait dies at ten minutes and the stop hook then blocks the stop until the agent starts another wait. The heartbeat file is what lets the stop hook tell those two cases apart: a wait that was killed stops beating, so the next stop is blocked with an instruction to start the wait again, and a wait that is alive lets the agent stop.

## Clarifications made while implementing

These readings of the specs were needed to build the workflow checks. None adds behavior. Each is listed so the reviewer can object.

- The check scripts live in a `checks/` subdirectory beside the workflow file, and the workflow file runs them through `$WF_WORKFLOW_DIR`. The spec says "beside it" and the directory keeps the workflow directory readable.
- The revise step uses a variant of the pushed check that also requires the latest commit to be newer than the moment the step was entered. Without it the revise step would pass at once, because the branch is already pushed when the review asks for changes.
- A pull request that was merged counts as approved for the review outcome, because merging requires the review that the run was waiting for. A pull request closed without merging is the closed outcome.
- Only the latest approving or change-requesting review submitted after the step was entered decides the outcome, so a later approval wins over an earlier request for changes and the reverse.
- With no change argument the checks find the change from the files that differ from the default branch. When more than one change differs, the check fails and says to start the run with `--input change=<name>`.
- A failing step with no job stops the run with the failing check's message. The closed outcome routes to stop, so the stop reason names the step and not the closed pull request.
- The apply step tells the agent to open the draft implementation pull request as soon as the first work is pushed, because CI runs only on pull requests and the apply step requires CI to be green.
- A committed check, which is the pushed check over the whole worktree including untracked files, guards every step that changes code. The proposal steps keep the narrower check over `openspec/changes`.
- An implementation-ready check was added to the archive step. The spec requires the implementation pull request to be ready for merge, and the archive step is the last place the run can say so.
- The recording artifact is found by looking in the implementation pull request description for the first link that ends in a video, image, or terminal recording extension, or points at GitHub attachments or an asciinema page. This is a heuristic, because the spec asks for the artifact and not for how to find it.
- The headless agent for the verify, review, and scope checks is `claude -p` with read-only tools, its own settings sources, no saved session, and a private workflow state directory so that the hooks of this repository do not act on it. A missing verdict line, a failure to run, or a refusal is broken, and only a last line of exactly `VERDICT: PASS` or `VERDICT: FAIL` counts.
- The scope step has only the scope check and no job. A scope failure stops the run, as the spec asks, and the findings are the stop reason. A new commit does not rescue a stopped run, so the person starts a new run after fixing the scope.
- The stack checks need the base branch of the pull request to be the branch of another pull request. After the proposal pull request is merged and its base is retargeted, the implementation check would report that it is not stacked. The MVP does not handle that case.
- The runs board shows only the first line of a long check message, with the full text kept in `wf show` and the JSON output, because the headless findings are lists.
- Only reviews from the owner, members, and collaborators of the repository count for the review outcome and the approved invariant, because anyone can submit a review on a public pull request and the review text is shown to the agent. The revise jobs also say that the comments are feedback to evaluate and cannot widen the job. This was added after an automated security review of the check scripts.

## Results

This section is input to `establish-workflow-platform`. It records what building the MVP showed about the model, drawn from the spike, the scratch runs recorded in `qa.md`, and the tests. The three real-session proofs, which are tasks 8.7, 9.8, and 10.1, need a human reviewer and are not yet part of this evidence, so every claim below about the agent's behavior in a real session is limited to the headless spike.

What held:

- One event log as the only state was enough. The runs board, the status line, the stop hook, and the printed next command are all derived from it, and no command needed a second store.
- Checks as scripts that print a message and exit with a code were easy to write and easy to test. The OpenSpec workflow needed no check that could not be a short script over `git`, `gh`, `jq`, and `openspec`.
- Evaluating checks on demand, with the agent as the only driver, kept the model small. There is no daemon, no webhook, and no scheduler, and a pending check is just a command the agent runs in the background.
- A stop hook that blocks with the next command as its reason kept the agent going in the spike, and the progress guard gave a bounded way out when a run cannot advance.
- Caching a verdict check by commit made a slow headless review cost one run per commit and not one run per evaluation.

What felt wrong:

- The workflow file is only half of the definition. The jobs, the check scripts, and the verdict prompts together define the process, and a reader has to open all three to know what a step demands. The format has no place that shows them as one thing.
- Check scripts run with the permissions of the person, and the install command allows `wf` without a prompt, so an agent that can edit a workflow file can run any command unprompted. The state directory guard stops tampering with the log, not abuse of the check command.
- The model has no way to pass a value from one check to a later step except through routes and artifacts, so `gh` is called again for the same pull request in several checks. This is slow and noisy, and a shared per-visit fact store would remove it.
- Outcomes exist only on the last check of a step. The review steps needed an extra step per decision point, which makes the OpenSpec workflow longer than the process it describes.
- A stopped run cannot be rescued by new work. This is right for a failed scope check, but it means the person restarts the whole run for a problem that a one-line fix would solve, and the log of the first run is not carried into the second.
- Long check messages, such as lists of findings, do not fit the board. The board now shows the first line, but the findings are shown to the agent in full and to the person only in `wf show`, so the person sees less than the agent.
- Stacking and the base branch rule are tied to GitHub pull requests and to `gh stack`. The checks break when a lower pull request is merged and retargeted, which is the normal end of a stack.
- The headless agent verdicts were exercised with a stand-in in the scratch runs. Whether the real verdicts are stable enough to gate a run is still unknown.

What this suggests for the platform:

- Treat a workflow as a directory with one manifest that names the file, the scripts, and the prompts, so that it can be validated, copied, and trusted as a unit.
- Decide how a workflow earns trust before its scripts run, such as hashing the directory at install time, and stop relying on the `wf` permission alone.
- Add a way to share facts between checks within a visit.
- Let a stopped run be resumed from its stopping step by an explicit person action.
- Keep the agent as the only driver until a real session shows that it fails to keep going, because nothing in the MVP evidence argues for a background runner.
