# Design

## Context

This repository is greenfield. The only existing material is the author's hand-written skills for the OpenSpec process (plan-draft, execution-loop, shepherd-merge-and-release), which describe the process as prose that the model follows. See proposal.md for motivation.

The author already runs many agents in Agent Session Manager, a macOS app. It detects agent state through harness hooks that write files, exposes a local MCP server, and launches agents in git worktrees. It has no way for an outside process to start a pane with a prompt or push state into it, because its MCP server only accepts tokens issued to panes it launched. This shapes the integration story: Agent Session Manager is a future client and does not constrain the first design.

The platform has three kinds of parts and a strict dependency direction:

```
  TRIGGERS                ENGINE (base program)                 CLIENTS
  (plugins)         +--------------------------------+     (TUI, web, others)
 github.issue  -->  | workflow definitions (IR)      |        ^         |
 prompt        -->  | run state machine              | events |         | commands
 (more)        -->  | durable event log              | -------+         |
                    | attention + artifact registry  | <----------------+
                    +---------------+----------------+   answer, start,
                                    |                    cancel, attach
                    plugin protocol | JSON-RPC over stdio
          +-----------+-------------+-----------+-------------+
          v           v             v           v             v
       github      openspec      harness-    shell      (third-party
                                 acp                     plugins)
```

Clients depend on the engine. The engine depends on plugins only through the protocol. Plugins never depend on each other or on clients.

Glossary used consistently across the specs:

- Workflow: a static graph of steps, in the IR.
- IR: the versioned JSON form of a workflow, the only thing the engine executes.
- Run: one execution of a workflow, with its own state and event log.
- Step: a node in the graph. Its kind is plugin, agent, gate, or workflow.
- Outcome: the named result a step resolves with. Transitions route on outcomes.
- Trigger: something a plugin reports that starts a run.
- Session: a named, long-lived agent conversation on a harness plugin.
- Attention request: a thing the run needs from an operator.
- Artifact: something a run produces that a person may want to open, such as a pull request.
- Run context: the trigger payload, inputs, step outputs, and artifacts of a run.
- Namespace: the scope that owns workflows, runs, and events. Every record belongs to one, and the default is `default`.
- Store: the place that keeps run records, reached through the store protocol.
- Signal: a message that changes a run, such as a step outcome, an answer, a trigger event, or an artifact report.
- Runner: the part that hosts plugin processes and agent sessions on behalf of the engine. In this change it is always in the same process as the engine.

## Goals / Non-Goals

**Goals:**
- A core small enough to read in a sitting, with no knowledge of any specific service, harness, or process.
- Any workflow can be drawn before it runs, and any run can answer the three at-a-glance questions from engine state alone.
- Anyone can write a plugin or a client in any language, and share it as a path or a git reference.
- One executable that runs on a laptop, in a container, in CI, or behind a bot.
- Mixed harnesses in one workflow, with identical attention and artifact handling.

**Non-Goals:**
- A plugin marketplace or central registry.
- Running on edge isolates or inside a browser. The engine needs a real operating system, as the agents it hosts do.
- Imperative control flow in definitions. Logic belongs in plugin steps.
- Showing or storing the full transcript of agent output. Clients may reach raw output only by explicit action.
- Sandboxing plugins. Plugins are trusted executables. A future managed service keeps this true by having each organization run its own runners, so plugins are trusted by the organization that runs them.
- Running many engine instances, remote runners, or a managed service. This change only specifies the seams that keep those possible, listed in D10.

## Decisions

### D1. Core language is Go, shipped as one executable

Go produces a single static executable per operating system and architecture with no runtime to install, which is the most direct route to "run anywhere a process can run": a laptop, a container, a CI runner, a backend, or bundled beside another application. A GitHub Action can download the release executable or use a container image. It is also the author's strongest language, which matters because the author reviews behavior and not code.

Alternatives considered:
- TypeScript on Node or Bun: natural for browsers and GitHub Actions, but needs a runtime or produces a very large compiled executable.
- Rust: smallest executables and good WebAssembly options, but the author does not use it and would depend entirely on the agent for idioms.

Portability comes more from the protocols than from the language. Plugins and clients speak JSON-RPC, so they can use any language. The JSON Schemas are the source of truth and types for Go and TypeScript are generated from them. Dagger is prior art for one engine that runs identically on a laptop and in CI.

### D2. Plugins are separate processes speaking JSON-RPC 2.0 over stdio

This is the model used by editor language servers and MCP: an executable plus a manifest, a capability handshake on start, and messages in both directions. It makes plugins language-independent, isolates crashes from the engine, and makes sharing a plugin as simple as sharing a repository. Built-in plugins use the same protocol, which keeps the engine honest about knowing nothing specific.

Alternatives considered:
- In-process Go plugins or shared libraries: fragile across platforms and versions, Go only.
- WebAssembly plugins: strong sandboxing, but plugins need to spawn processes and open network connections, which defeats the point.
- gRPC: good tooling, but heavier than needed and awkward to hand-write in small plugins.

### D3. Every step resolves to an outcome, and routing is by outcome

A step is a call that eventually resolves to one outcome string. Everything else is a use of that one mechanism:

- A condition is a step with no side effects.
- A wait is a step that stays active until its plugin or agent resolves it.
- A loop is a transition to an earlier step, bounded by `max_iterations`.
- A human gate is a step resolved by an operator's answer.
- An invariant failure is an outcome routed to the core needs-human target.

The definition language has references and routing and nothing else: no expressions. This keeps the graph static so that every client can draw it, and it forces logic into plugins where it can be tested. Transitions can pass named values to the next step (for example, review comments to a revision step) so that a revision step can be reached from several places without a conditional.

Alternative considered: a general expression language for conditions, as in GitHub Actions. Rejected because it makes definitions into programs and reintroduces the unpredictability the platform exists to remove.

### D4. One IR, two ways to author

Both YAML and code produce the same versioned IR, and only the IR is executed. This is the pattern used by Kubeflow Pipelines, whose Python SDK compiles to an IR that backends run, and by Microsoft's Agent Framework, which loads declarative and code-first workflows into the same workflow abstraction.

This is different from "YAML that computes code". YAML is the primary authoring format and maps one to one onto the IR. The Go SDK is a builder that emits the IR, useful for generating and reusing parts of a graph. Behavior that needs code is written as a plugin step. An export command converts YAML to SDK code for anyone who prefers to maintain a workflow in code.

Alternative considered: workflows as programs executed by a durable runtime, in the style of Temporal. Rejected because the graph cannot be drawn until the program runs, which defeats the visual goal.

### D5. Agent sessions are headless first, with an optional interactive mode

Each harness offers some mix of a terminal interface, a structured protocol, and hooks. The Claude Agent SDK lets a program intercept the agent's questions and permission requests. Codex's app-server exposes threads, turns, and approval requests as JSON-RPC. The Agent Client Protocol, created by Zed, is a JSON-RPC protocol over standard input and output that agents and editors use for sessions and permission requests, and Claude, Codex, Cursor, Gemini, and OpenCode are reported to support it directly or through adapters.

The platform therefore requires every harness plugin to support headless mode and to translate native questions and approvals into attention requests. Any client can then render and answer them, which gives the platform its own equivalent of the agent's question tool with identical behavior in a terminal, a browser, a pipeline, or a bot. Interactive mode is an optional capability in which the agent's own terminal runs in a pseudo-terminal that clients attach to.

To make "the agent says it is done" and "the agent needs me" work the same in every harness, the engine injects a small tool server into each session through the harness's own tool mechanism, offering three tools: complete the step with an outcome and summary, ask the operator, and report an artifact.

The first harness plugin is a generic one that drives any agent speaking the Agent Client Protocol, so that one plugin covers many harnesses. Dedicated plugins are added where the protocol falls short.

Alternatives considered:
- The client hosts the agent, as Agent Session Manager does today: fits that app but every client must reimplement launching and state detection, and a pipeline or bot has no client to host it.
- Headless only, with no interactive mode: simplest, but gives up each agent's own interface for exploratory steps. This remains an acceptable first release, because the contract leaves room for interactive mode later.

### D6. Run state is an append-only event log

Every state change is recorded as an event before it is acted on, and current state is computed from the log. This makes runs durable across restarts and multi-day review waits, auditable, and simple to stream to clients that reconnect from a sequence number. Storage sits behind the store protocol described in D10: a memory store and a local durable store first, with other stores possible for pipelines and hosted use.

The engine alone tracks the current step. This is the reason plugins stay small: a plugin never needs to know where in a workflow it was called.

### D7. Clients speak JSON-RPC over WebSocket or standard input and output

Clients list workflows and their graphs, subscribe to events, answer attention requests, control runs, and attach to interactive sessions. Because the protocol is the only interface, built-in clients get no private access and anyone can write another. The engine listens on loopback by default and requires a token for anything else.

The summary of a run in the protocol carries the three things people running many agents want: the current step, whether the run needs them, and its artifacts.

### D8. Plugin trust and distribution

Plugins are trusted programs. The engine shows the launch command before a plugin is first added, passes secrets only through environment variables that the workflow names, and records the exact version used by each run. Plugins resolve from a local path or a git reference, in the same spirit as referencing an action by repository and tag. No registry is built.

### D9. Reference workflow is in-repo and is the conformance bar

The OpenSpec process is written as a real workflow (see the appendix) and later shipped in the repository. It exercises every core mechanism: agent steps across two harnesses, fresh-context sessions, human gates, waits, bounded loops, an invariant stop, sub-workflows, artifacts, and stacked pull requests. The platform is conformant when that workflow runs end to end with recorded proof.

### D10. Built for one process, safe for many

The default is one executable on one machine. The goal of this decision is that nothing in the first build blocks running the engine as many instances behind a load balancer, or offering it to many organizations later. Several things are free in one process and break across many: the event log, who advances a run, trigger deduplication, timers, "exactly one answer wins", client event streams, and the plugin and agent processes and their working directories on one disk.

```
  ONE PROCESS (default)                      MANY INSTANCES (future)
  +--------------------------------+
  | engine | store  | runner       |         clients --> load balancer
  |        | (local)| plugins +    |                         |
  |        |        | agents       |             [engine] [engine] [engine]
  +--------------------------------+                  |       |       |
                                                +-----+-------+-------+----+
                                                |  shared store + feed     |
                                                +-----+--------------------+
                                                      |  runners connect
                                                [runner] [runner] [runner]
```

The seams that make this possible:

- The store is a protocol, and its central operation is a conditional append: add events to a run only if the run's last sequence number is what the caller expects. If two instances race to advance a run, the second gets a conflict, reloads, and sees the step has moved. The same operation makes "exactly one answer wins" a guarantee and not a lucky outcome.
- Trigger deduplication is a claim in the store and not a set in memory, so every instance can poll the same service safely.
- Timeouts are recorded as deadlines in the log and not as timers in memory, so any instance can fire one, and the conditional append means it fires once.
- Clients watch runs through the store's records, not through the memory of the process that happened to advance them.
- Resume means taking ownership of a run. A restart is the case where the instance takes back its own runs, and failover is the case where another instance takes them. The same scenarios cover both.
- Every record carries a namespace from the start, with a single namespace called default. Per-organization authorization is future work, but adding the field to every identifier later would be a data migration.
- The engine hands steps and sessions to a runner using the plugin protocol messages. The only runner in this change is in the same process, so nothing is serialized over a network, and a later change can add runners on other machines. The split follows the pattern of Temporal workers and CI runners, where the part that decides is separate from the part that does the work.

The store is a protocol and not a compiled-in interface. The built-in memory and local stores implement the same messages inside the binary, so the default stays one executable with nothing to install. Third-party stores are separate processes in any language, shared as a path or a git reference like any plugin. A compiled-in interface would force every community store to be written in Go and merged upstream or built into a custom executable, which is the opposite of how the rest of the platform is extended. Atomicity is not a problem over a protocol, because the whole contract is a few operations that are each atomic alone: append, read, claim, and list, plus optional watch and lease. The engine never needs a transaction across several calls. Latency is not a problem either, because the store sits behind steps that take seconds to days. A store declares whether it is durable, whether it is safe for many instances, and whether it can watch. The engine refuses setups that contradict the declaration.

Failure is handled by one rule instead of a stop mode: a signal is confirmed only after it is recorded, and every sender keeps each signal and retries it with a unique key until it is confirmed.

```
 sender (runner, plugin, client, webhook)      engine           store
   |  resolve(step, key=k, outcome=done)         |                |
   |-------------------------------------------->|  append        |
   |                                             |--------------->|
   |                                             |  unreachable   |
   |  not recorded, retry                        |                |
   |<--------------------------------------------|                |
   |  later, same key k                          |                |
   |-------------------------------------------->|--------------->|  ok
   |  confirmed                                  |<---------------|
   |<--------------------------------------------|  start next step
```

A crashed store adapter is restarted like any plugin, and the senders' retries cover the gap. An unreachable database makes confirmations fail, and runs continue on their own when it returns, with no operator action. An engine instance that dies mid-request looks the same to the sender. The cost is that a step may run more than once, because a sender that dies while holding an unconfirmed signal leaves the step active and the existing re-attach or restart rules apply. The platform therefore promises that a step runs at least once. A local spill journal that replays when the store returns was rejected, because it is a second source of truth that breaks with many instances: by the time the store returns, another instance may own the run and the replayed events would conflict.

Alternative considered: a mode in which the engine stops everything when the store fails. Rejected because it leaves open what happens to the work in flight, such as an agent that just finished or an operator who just answered, and because it makes the store a special case when the retry rule already covers it.

## Risks / Trade-offs

- **Harness protocol coverage differs by agent** → Start with the generic Agent Client Protocol plugin, check each target harness against it early (task group 7), and add dedicated plugins only for real gaps. Exact adapter commands for each harness are resolved at implementation time.
- **Headless mode loses each agent's native interface** → Keep interactive mode in the contract and the client protocol so it can be added without breaking anything, and use attention requests as the shared question surface.
- **A static, expression-free language may feel limiting** → Put logic in plugin steps, make writing one cheap through the shell plugin and a small manifest, and pass values along transitions so branching on data is rarely needed.
- **Agents may not call the complete tool** → A per-step policy re-prompts, waits for the operator, or resolves with a named outcome, with a bound on re-prompts, after which the run asks for help.
- **Event log growth** → Events carry state changes and summaries, not agent transcripts, so volume is low. Compaction can be added later since state is derivable.
- **Seams cost effort now for futures that may never ship** → Keep to the cheap ones (the store protocol, namespaces, recorded deadlines, the runner boundary in one process) and leave fleets, remote runners, and the managed service for later changes.
- **Plugins are trusted code** → Show the launch command, scope secrets by name, record versions, and defer sandboxing. Third-party plugins should be read before use.
- **Senders must retry signals with unique keys, which burdens plugin authors** → The response to a JSON-RPC request is already the confirmation, and the SDK handles the retry loop so a small plugin needs no extra code.
- **Steps run at least once, not exactly once** → Say so in the plugin and harness contracts, key every signal so duplicates are no-ops, and keep steps that must not repeat idempotent on their own side.
- **A third-party store could corrupt runs** → Ship the conformance suite as a command that tests any store in any language, and make the engine refuse setups that contradict a store's declared capabilities.
- **While the store is unreachable the engine cannot advance runs** → Signals are held by their senders and recover on their own, clients see engine health and reads marked as possibly out of date, and no run is moved to needs-human because of a store outage.
- **IR versioning** → Version the IR from the start, reject unknown versions with a clear message, and keep the schema additive within a version.
- **Polling cost and latency for services like GitHub** → Plugins poll by default so they work anywhere, and accept webhooks through the engine when a public endpoint exists.
- **Large first release** → The task list orders work so each group is demonstrable alone: the engine runs workflows made of shell steps before any agent or service plugin exists.

## Migration Plan

There is nothing to migrate. Rollout is by task group; each group ends with a recorded demonstration, and the final group runs the reference workflow end to end. The author's existing skills remain usable until the reference workflow reaches parity.

## Open Questions

- Which adapter command, or native mode, each target harness uses with the generic Agent Client Protocol plugin. Resolved when task group 7 starts, without changing specs.
- Whether the reference workflow should open the implementation pull request as a draft before the apply loops, so CI runs after every push as the author's execution-loop skill requires. The author's step list opens it after the scope check, so the appendix follows that order and checks CI after the pull request exists. Either order is expressible without changing any spec.
- How feedback on the implementation pull request, which the author's list sends back to exploration, interacts with the branches already created. The appendix routes it to the same revision step as the proposal review, which reruns the proposal steps from the implementation branch. A later change may add an implementation-specific revision path.
- Where the engine keeps its event log by default on each operating system. A task-group decision that does not affect the specs.

## Appendix: reference workflow

The OpenSpec process as the platform's first workflow. The notation is illustrative of the YAML format and must be validated by the validate command once it exists. Names after `uses` refer to steps provided by plugins. The engine's four step kinds are `agent`, `gate`, `workflow`, and plugin steps via `uses`. Gate answers expose their free text as `text`, and agent steps expose their summary as `summary`.

openspec-change.yaml:

```yaml
ir_version: 1
name: openspec-change
description: Take an idea through exploration, proposal, implementation, and archive, with human review at the plan and each pull request.

inputs:
  prompt: {type: string, required: true}
  repo: {type: string, required: true}
  workdir: {type: string, required: true}
  change: {type: string, required: true}
  default_branch: {type: string, default: main}

plugins:
  github: {uses: builtin/github}
  openspec: {uses: builtin/openspec}
  shell: {uses: builtin/shell}
  claude: {uses: builtin/harness-acp, with: {command: ["claude-acp-adapter"]}}
  codex: {uses: builtin/harness-acp, with: {command: ["codex-acp-adapter"]}}

triggers:
  - uses: github/issue_labeled
    with: {repo: "${{ inputs.repo }}", label: workflow}
    inputs: {prompt: "${{ trigger.title }}\n\n${{ trigger.body }}"}

sessions:
  author: {harness: claude, mode: headless}
  reviewer: {harness: codex, mode: headless, fresh: true}
  checker: {harness: claude, mode: headless, fresh: true}

artifacts:
  change: {kind: openspec/change}
  proposal_pr: {kind: github/pull_request}
  implementation_pr: {kind: github/pull_request}
  archive_pr: {kind: github/pull_request}

start: explore

steps:
  explore:
    type: agent
    session: author
    prefix: ${{ plugins.openspec.prefix.explore }}
    prompt: ${{ inputs.prompt }}
    outcomes: [plan_ready]
    on: {plan_ready: plan_gate}
    default: needs_human

  revise:
    type: agent
    session: author
    prefix: ${{ plugins.openspec.prefix.explore }}
    prompt: "Continue exploring and revise the plan using this feedback:\n${{ route.feedback }}"
    outcomes: [plan_ready]
    max_iterations: 10
    on: {plan_ready: plan_gate}
    default: needs_human

  plan_gate:
    type: gate
    prompt: Is the plan ready to be written up as a proposal?
    outcomes: [approved, feedback]
    on:
      approved: propose
      feedback: {goto: revise, pass: {feedback: "${{ steps.plan_gate.outputs.text }}"}}

  propose:
    type: agent
    session: author
    prefix: ${{ plugins.openspec.prefix.propose }}
    prompt: Write the OpenSpec change named ${{ inputs.change }}. Do not implement anything.
    outcomes: [proposed]
    max_iterations: 5
    on: {proposed: proposal_push}
    default: needs_human

  proposal_push:
    type: workflow
    workflow: ./commit-and-push.yaml
    inputs: {workdir: "${{ inputs.workdir }}", message: "Proposal for ${{ inputs.change }}"}
    max_iterations: 20
    on: {succeeded: open_proposal_pr, failed: needs_human, cancelled: cancelled}

  open_proposal_pr:
    uses: github/open_pull_request
    with:
      repo: ${{ inputs.repo }}
      workdir: ${{ inputs.workdir }}
      base: ${{ inputs.default_branch }}
      title_prefix: Proposal
      draft: true
    expects: [{name: proposal_pr, kind: github/pull_request}]
    on: {opened: proposal_checks, reused: proposal_checks}
    default: needs_human

  proposal_checks:
    uses: github/check_status
    with: {pull_request: "${{ artifacts.proposal_pr.ref }}", wait: true}
    on: {green: validate_proposal, red: proposal_ci_fix}
    default: needs_human

  proposal_ci_fix:
    type: agent
    session: author
    prompt: "Fix the failing checks and nothing else:\n${{ steps.proposal_checks.outputs.failures }}"
    outcomes: [fixed]
    max_iterations: 5
    on: {fixed: proposal_push}
    default: needs_human

  validate_proposal:
    uses: openspec/validate
    with: {workdir: "${{ inputs.workdir }}"}
    on: {valid: ready_proposal, invalid: fix_proposal}
    default: needs_human

  fix_proposal:
    type: agent
    session: author
    prompt: "Fix these OpenSpec validation problems:\n${{ steps.validate_proposal.outputs.messages }}"
    outcomes: [fixed]
    max_iterations: 5
    on: {fixed: proposal_push}
    default: needs_human

  ready_proposal:
    uses: github/mark_ready
    with: {pull_request: "${{ artifacts.proposal_pr.ref }}"}
    on: {done: proposal_review}
    default: needs_human

  proposal_review:
    uses: github/wait_for_review
    with: {pull_request: "${{ artifacts.proposal_pr.ref }}"}
    on:
      approved: invariants
      changes_requested: {goto: revise, pass: {feedback: "${{ steps.proposal_review.outputs.comments }}"}}
      closed: needs_human
    default: needs_human

  invariants:
    uses: openspec/check_invariants
    with: {workdir: "${{ inputs.workdir }}", change: "${{ inputs.change }}", default_branch: "${{ inputs.default_branch }}"}
    on: {ok: pr_state, violated: needs_human}
    default: needs_human

  pr_state:
    uses: github/pull_request_state
    with: {pull_request: "${{ artifacts.proposal_pr.ref }}"}
    on: {open_approved_green: impl_branch, not_ready: needs_human}
    default: needs_human

  impl_branch:
    uses: shell/run
    with: {cwd: "${{ inputs.workdir }}", command: [git, switch, -c, "implement/${{ inputs.change }}"]}
    on: {success: apply, failure: needs_human}
    default: needs_human

  apply:
    type: agent
    session: author
    prefix: ${{ plugins.openspec.prefix.apply }}
    prompt: Apply the change ${{ inputs.change }}.
    outcomes: [applied]
    max_iterations: 50
    on: {applied: apply_push}
    default: needs_human

  apply_push:
    type: workflow
    workflow: ./commit-and-push.yaml
    inputs: {workdir: "${{ inputs.workdir }}", message: "Apply ${{ inputs.change }}"}
    max_iterations: 50
    on: {succeeded: tasks_left, failed: needs_human, cancelled: cancelled}

  tasks_left:
    uses: openspec/task_progress
    with: {workdir: "${{ inputs.workdir }}", change: "${{ inputs.change }}"}
    on: {tasks_remaining: apply, complete: verify}
    default: needs_human

  verify:
    type: agent
    session: author
    prefix: ${{ plugins.openspec.prefix.verify }}
    prompt: Verify the implementation of ${{ inputs.change }} and report issues.
    outcomes: [issues_found, clean]
    max_iterations: 20
    on:
      issues_found: {goto: fix_verify, pass: {findings: "${{ steps.verify.outputs.summary }}"}}
      clean: review
    default: needs_human

  fix_verify:
    type: agent
    session: author
    prompt: "Fix these verification issues:\n${{ route.findings }}"
    outcomes: [fixed]
    max_iterations: 20
    on: {fixed: verify_push}
    default: needs_human

  verify_push:
    type: workflow
    workflow: ./commit-and-push.yaml
    inputs: {workdir: "${{ inputs.workdir }}", message: "Fix verification issues"}
    max_iterations: 20
    on: {succeeded: verify, failed: needs_human, cancelled: cancelled}

  review:
    type: agent
    session: reviewer
    prompt: Review the work for ${{ inputs.change }} adversarially. Assume something is wrong and find what. Report only actionable findings.
    outcomes: [findings, no_findings]
    max_iterations: 10
    on:
      findings: {goto: fix_review, pass: {findings: "${{ steps.review.outputs.summary }}"}}
      no_findings: scope_check
    default: needs_human

  fix_review:
    type: agent
    session: author
    prompt: "Fix these review findings:\n${{ route.findings }}"
    outcomes: [fixed]
    max_iterations: 10
    on: {fixed: review_push}
    default: needs_human

  review_push:
    type: workflow
    workflow: ./commit-and-push.yaml
    inputs: {workdir: "${{ inputs.workdir }}", message: "Address review findings"}
    max_iterations: 10
    on: {succeeded: review, failed: needs_human, cancelled: cancelled}

  scope_check:
    type: agent
    session: checker
    prompt: Compare the work with the proposal, design, and specs for ${{ inputs.change }}. Report scope creep or missing requirements. Edge cases and invariants are not scope creep.
    outcomes: [clean, scope_creep, missing_requirements]
    on: {clean: impl_push}
    default: needs_human

  impl_push:
    type: workflow
    workflow: ./commit-and-push.yaml
    inputs: {workdir: "${{ inputs.workdir }}", message: "Implementation of ${{ inputs.change }}"}
    max_iterations: 20
    on: {succeeded: open_impl_pr, failed: needs_human, cancelled: cancelled}

  open_impl_pr:
    uses: github/open_pull_request
    with:
      repo: ${{ inputs.repo }}
      workdir: ${{ inputs.workdir }}
      base: ${{ artifacts.proposal_pr.branch }}
      title_prefix: Implementation
      draft: true
    expects: [{name: implementation_pr, kind: github/pull_request}]
    on: {opened: stack_impl, reused: stack_impl}
    default: needs_human

  stack_impl:
    uses: github/stack_add
    with: {pull_request: "${{ artifacts.implementation_pr.ref }}", below: "${{ artifacts.proposal_pr.ref }}"}
    on: {linked: impl_checks, unavailable: needs_human}
    default: needs_human

  impl_checks:
    uses: github/check_status
    with: {pull_request: "${{ artifacts.implementation_pr.ref }}", wait: true}
    on: {green: impl_review, red: impl_ci_fix}
    default: needs_human

  impl_ci_fix:
    type: agent
    session: author
    prompt: "Fix the failing checks and nothing else:\n${{ steps.impl_checks.outputs.failures }}"
    outcomes: [fixed]
    max_iterations: 5
    on: {fixed: impl_ci_push}
    default: needs_human

  impl_ci_push:
    type: workflow
    workflow: ./commit-and-push.yaml
    inputs: {workdir: "${{ inputs.workdir }}", message: "Fix failing checks"}
    max_iterations: 5
    on: {succeeded: impl_checks, failed: needs_human, cancelled: cancelled}

  impl_review:
    uses: github/wait_for_review
    with: {pull_request: "${{ artifacts.implementation_pr.ref }}"}
    on:
      approved: mark_impl_ready
      changes_requested: {goto: revise, pass: {feedback: "${{ steps.impl_review.outputs.comments }}"}}
      closed: needs_human
    default: needs_human

  mark_impl_ready:
    uses: github/mark_ready
    with: {pull_request: "${{ artifacts.implementation_pr.ref }}"}
    on: {done: archive_branch}
    default: needs_human

  archive_branch:
    uses: shell/run
    with: {cwd: "${{ inputs.workdir }}", command: [git, switch, -c, "archive/${{ inputs.change }}"]}
    on: {success: archive, failure: needs_human}
    default: needs_human

  archive:
    uses: openspec/archive
    with: {workdir: "${{ inputs.workdir }}", change: "${{ inputs.change }}"}
    on: {done: archive_push, failed: needs_human}
    default: needs_human

  archive_push:
    type: workflow
    workflow: ./commit-and-push.yaml
    inputs: {workdir: "${{ inputs.workdir }}", message: "Archive ${{ inputs.change }}"}
    on: {succeeded: open_archive_pr, failed: needs_human, cancelled: cancelled}

  open_archive_pr:
    uses: github/open_pull_request
    with:
      repo: ${{ inputs.repo }}
      workdir: ${{ inputs.workdir }}
      base: ${{ artifacts.implementation_pr.branch }}
      title_prefix: Archive
      draft: true
    expects: [{name: archive_pr, kind: github/pull_request}]
    on: {opened: stack_archive, reused: stack_archive}
    default: needs_human

  stack_archive:
    uses: github/stack_add
    with: {pull_request: "${{ artifacts.archive_pr.ref }}", below: "${{ artifacts.implementation_pr.ref }}"}
    on: {linked: succeeded, unavailable: needs_human}
    default: needs_human
```

commit-and-push.yaml, the small sub-workflow used above:

```yaml
ir_version: 1
name: commit-and-push

inputs:
  workdir: {type: string, required: true}
  message: {type: string, required: true}

plugins:
  shell: {uses: builtin/shell}

start: commit

steps:
  commit:
    uses: shell/run
    with:
      cwd: ${{ inputs.workdir }}
      command: [sh, -c, "git add -A && git commit -m \"$MESSAGE\""]
      shell: true
      env: {MESSAGE: "${{ inputs.message }}"}
      match:
        - {pattern: "nothing to commit", outcome: nothing_to_commit}
    outcomes: [success, nothing_to_commit, failure]
    on: {success: push, nothing_to_commit: push, failure: failed}

  push:
    uses: shell/run
    with:
      cwd: ${{ inputs.workdir }}
      command: [git, push, --set-upstream, origin, HEAD]
    on: {success: succeeded, failure: failed}
```

How the appendix maps to the author's original list:

- Explore, feedback keeps exploring, approval goes forward: `explore`, `plan_gate`, `revise`.
- Propose, open draft pull request, wait for human review, feedback returns to explore: `propose` through `proposal_review`.
- Invariants hold or stop and ask: `invariants`, `pr_state`, with `needs_human`.
- Loop apply until no tasks remain: `apply`, `apply_push`, `tasks_left`.
- Loop verify until no issues: `verify`, `fix_verify`, `verify_push`.
- Loop adversarial review until no actionable findings: `review`, `fix_review`, `review_push`.
- Scope check: `scope_check`.
- Open implementation pull request, wait for review, feedback returns to explore, approval goes forward, mark ready, open archive pull request: `open_impl_pr` through `stack_archive`.
