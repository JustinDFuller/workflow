# Proposal

## Why

People who run many coding agents at once cannot tell, without reading scrollback, which step each agent is on, whether it needs them, or what it has produced. Today the control flow of a process like "explore, propose, review, apply, verify, open PRs" lives in prose skills that the model itself is trusted to follow, so the process is neither deterministic nor visible. This change establishes a platform that moves control flow out of the model and into a small deterministic engine, and shows the process visually. It is a single change on purpose: it is the specification of the whole platform, approved up front and implemented over many later sessions.

## What Changes

- Introduce a workflow platform made of three kinds of parts: a base program (the engine) that orchestrates runs, plugins that supply every concrete behavior, and clients that display runs and take operator input.
- The engine holds no knowledge of GitHub, OpenSpec, any agent harness, or any particular trigger, step, or condition. It only hosts plugins that speak a defined protocol, tracks the current step of every run, and records attention requests and artifacts.
- A workflow is a static graph of steps. Every step resolves to a named outcome, and transitions route on outcomes. Waits, conditions, loops, and human gates are all expressed with that one mechanism.
- Workflows are authored in YAML or built by code. Both produce the same versioned intermediate representation, which is the only thing the engine executes.
- Agent harnesses (Claude Code, Codex, Cursor, OpenCode, and others) are plugins. A workflow can declare several named sessions on different harnesses and mix them. Harness plugins run headless by default and turn native questions and approvals into engine attention requests, so any client can answer them. An interactive terminal mode is an optional harness capability.
- The engine gives every agent session a small set of tools to signal that a step is done, to ask the operator for something, and to report an artifact, so these signals work the same in every harness.
- Runs are recorded as an append-only event log so they survive restarts and multi-day waits.
- The event log lives behind a store protocol. The built-in memory and local stores use the same messages as third-party stores, which run as separate processes in any language. The engine is specified so that it behaves the same as one instance or as many sharing a store, but only single-instance operation ships in this change.
- Every workflow, run, and event belongs to a namespace, and a single namespace called default is used unless told otherwise.
- The engine confirms a signal (a step outcome, an answer, a trigger event, or an artifact report) only after recording it, and senders keep each signal and retry it until it is confirmed. A store outage therefore pauses progress and recovers on its own, and steps run at least once.
- Specify the first built-in plugins (shell, a generic Agent Client Protocol harness, GitHub, OpenSpec), the first clients (terminal UI and a local web visualizer), a command line, and a Go graph-builder SDK.
- Specify the user's OpenSpec workflow in full, as the reference workflow that the platform must be able to run.
- Add repository guardrails that fit a prose-only repository (fixed-width prose and OpenSpec validation) and point future sessions at these specs through the OpenSpec project context.

Nothing in this change implements the platform. The deliverable is the specification, design, and ordered task list that later sessions will build from.

## Capabilities

### New Capabilities

- `workflow-definition`: the workflow graph, its versioned intermediate representation, the YAML authoring format, references between steps, named sessions, sub-workflows, and static validation.
- `run-engine`: how runs are created, advanced, paused, resumed, cancelled, and retried, including loops, attention requests, the artifact registry, the durable event log, namespaces, and confirmation of signals after they are recorded.
- `run-store`: the protocol between the engine and the place that keeps run records, including conditional appends, deduplication, optional watching and leasing, declared store capabilities, the built-in memory and local stores, and a conformance suite for any store.
- `plugin-protocol`: how the engine discovers, launches, and talks to plugins, including the manifest, triggers, steps, artifact reporting, webhooks, and failure handling.
- `harness-protocol`: the contract every agent harness plugin implements, including headless and interactive modes, session continuity, native attention mapping, and the tools injected into agent sessions.
- `client-protocol`: how clients list workflows, observe runs, answer attention requests, and control runs, over WebSocket or standard input and output.
- `workflow-cli`: the command line that serves the engine, runs and validates workflows, manages plugins, and exports workflows.
- `workflow-sdk-go`: a Go library that builds the same intermediate representation as the YAML format.
- `plugin-shell`: a built-in plugin that runs commands and maps their results to outcomes.
- `plugin-harness-acp`: a built-in harness plugin that drives any agent that speaks the Agent Client Protocol.
- `plugin-github`: a built-in plugin for GitHub triggers, pull request steps, review waits, and pull request artifacts.
- `plugin-openspec`: a built-in plugin for OpenSpec validation, task status, and change artifacts.
- `client-tui`: a built-in terminal client that shows many runs at a glance and lets the operator answer requests.
- `client-web`: a built-in local web client that draws a live graph of each run with the current step highlighted.
- `openspec-workflow`: the reference workflow for the OpenSpec process, and the conformance bar for the platform.

### Modified Capabilities

None. The project has no existing specs.

## Impact

- New Go module with a single binary, the JSON Schemas that define the intermediate representation and every protocol, and generated types for Go and TypeScript.
- New built-in plugins and clients that talk to the engine only through the published protocols.
- New repository CI checks for fixed-width prose and OpenSpec validity. Further guardrails (code comments, lint, tests) arrive with the first task group, when code exists.
- Future work outside this change, each of which will be its own change: implementing the interactive terminal attach mode, packaging as a GitHub Action, running many engine instances against a shared store with runners on separate machines, a managed service across organizations with per-namespace authorization, a TypeScript SDK, a plugin marketplace, and integration with Agent Session Manager. That integration will need an external client credential path and panes that attach to engine sessions, neither of which exists in Agent Session Manager today.
