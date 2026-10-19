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
                                    |                    cancel
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
- Step: a node in the graph. Its kind is plugin, agent, wait, or workflow. Every step does its work and exits with an outcome.
- Outcome: the named result a step resolves with. Transitions route on outcomes. Asking questions and failing are outcomes.
- Trigger: something a plugin reports. A start trigger starts a run. A resume trigger moves a waiting run on from a wait step.
- Wait step: the only place a run waits. It lists the triggers that can resume it, and while it is pending the run rests in the store with nothing running.
- Correlation key: a string a trigger derives from its configuration and from each event, so the engine can match events to waiting runs without understanding them.
- Profile: a named bundle of harness, mode, working directory, and permission policy that agent steps run on.
- Session: one agent conversation on a harness plugin. It lives for exactly one agent step.
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
- A run can rest for any length of time, such as a review over a two-week vacation, at no cost and with no process running, and resume on an event.

**Non-Goals:**
- A plugin marketplace or central registry.
- Running on edge isolates or inside a browser. The engine needs a real operating system, as the agents it hosts do.
- Imperative control flow in definitions. Logic belongs in plugin steps.
- Showing or storing the full transcript of agent output. Clients may reach raw output only by explicit action.
- Sandboxing plugins. Plugins are trusted executables. A future managed service keeps this true by having each organization run its own runners, so plugins are trusted by the organization that runs them.
- Running many engine instances, remote runners, or a managed service. This change only specifies the seams that keep those possible, listed in D10.
- Running the steps of one run in parallel. A run is at one step at a time. Several runs run at once.
- Hosting an interactive agent's terminal inside the engine. Interactive agents run in a host outside the engine.

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

### D3. Every step exits with an outcome, and routing is by outcome

A step is a call that does its work and exits with one outcome string. Everything else is a use of that one mechanism:

- A condition is a step with no side effects.
- A loop is a transition to an earlier step, bounded by `max_iterations`.
- Asking a question is an outcome. Failing is an outcome. An invariant failure is an outcome routed to the core needs-human target.
- Waiting is not a step. It is a wait step, described in D12, where the run rests until a trigger moves it on. A human gate is a wait step whose trigger is an operator's answer.

The definition language has references and routing and nothing else: no expressions. This keeps the graph static so that every client can draw it, and it forces logic into plugins where it can be tested. Transitions can pass named values to the next step (for example, review comments to a revision step) so that a revision step can be reached from several places without a conditional. Loop counts reset only where a transition says so, so that restarting from the beginning does not spend the budget of the inner loops (see D16).

Two outcomes are built in because every workflow would otherwise have to route them on every step. An agent step can always exit with `questions`, which follows the core ask route unless the workflow routes it elsewhere (see D13). A plugin step can always exit with `needs_help`, which follows the needs-human target. A step or a workflow can also declare a default route for the outcomes it does not list, other than `questions`, so a workflow that cares only about its happy path stays short.

Alternative considered: a general expression language for conditions, as in GitHub Actions. Rejected because it makes definitions into programs and reintroduces the unpredictability the platform exists to remove.

### D4. One IR, two ways to author

Both YAML and code produce the same versioned IR, and only the IR is executed. This is the pattern used by Kubeflow Pipelines, whose Python SDK compiles to an IR that backends run, and by Microsoft's Agent Framework, which loads declarative and code-first workflows into the same workflow abstraction.

This is different from "YAML that computes code". YAML is the primary authoring format and maps one to one onto the IR. The Go SDK is a builder that emits the IR, useful for generating and reusing parts of a graph. Behavior that needs code is written as a plugin step. An export command converts YAML to SDK code for anyone who prefers to maintain a workflow in code.

Alternative considered: workflows as programs executed by a durable runtime, in the style of Temporal. Rejected because the graph cannot be drawn until the program runs, which defeats the visual goal.

### D5. Every agent step is a fresh, headless-first session

Each harness offers some mix of a terminal interface, a structured protocol, and hooks. The Claude Agent SDK lets a program intercept the agent's questions and permission requests. Codex's app-server exposes threads, turns, and approval requests as JSON-RPC. The Agent Client Protocol, created by Zed, is a JSON-RPC protocol over standard input and output that agents and editors use for sessions and permission requests, and Claude, Codex, Cursor, Gemini, and OpenCode are reported to support it directly or through adapters.

Several of these can in principle keep a conversation open or resume one later. The platform does not depend on that. A session lives for exactly one agent step, because a run may rest for days, move to another machine, or restart between two steps, and none of that can be allowed to depend on whether a harness can resume a conversation. A session may receive more than one prompt within its step (the turn-end re-prompt needs this), and it ends when the step exits. Named, long-lived sessions are replaced by profiles, which name a harness, a mode, a working directory, and a permission policy, and which any number of steps can use.

What a fresh step knows comes from the channels in D11. The engine builds the text the agent receives from the step's prefix, its prompt, its context sections, and instructions that name the allowed outcomes and the outputs each requires, and it shows that text to clients.

The platform requires every harness plugin to support headless mode. A headless session has nothing in it that waits for a person. The ask tool records questions and returns, and the agent keeps working. At the end of the turn the step exits with the outcome the agent passed to the complete tool, or with `questions` carrying everything it recorded. Permission requests cannot wait for a person either, so the profile's permission policy, a list of allow and deny rules interpreted by the harness plugin, answers them, and a denial reaches the agent as an ordinary tool error so it can try another way or exit with a question.

Interactive mode is an optional capability, and the agent is a separate process the engine does not host. The step hands the prompt and a run-scoped token to a host (the operator's terminal, Agent Session Manager, or another host) and exits with `launched`. The run then rests at a wait step on the built-in trigger for the agent's completion. The person at the terminal answers the agent's questions directly, and when the agent calls the complete tool it records its signal in the store with the token, so the engine need not be running. The engine stays stateless and event-driven, and does not care whether the agent takes ten minutes or ten days. The engine-hosted pseudo-terminal and the client attach protocol that earlier drafts of this design carried are removed.

To make "the agent says it is done" and "the agent has questions" work the same in every harness, the engine injects a small tool server into each session through the harness's own tool mechanism, offering three tools: complete the step with an outcome, outputs, notes, and a summary, record questions for the operator, and report an artifact.

The first harness plugin is a generic one that drives any agent speaking the Agent Client Protocol, so that one plugin covers many harnesses. Dedicated plugins are added where the protocol falls short.

Alternatives considered:
- Named long-lived sessions that several steps continue: possible in some harnesses, but it ties a run to a conversation that cannot survive days of waiting, a restart, or a move to another machine, and it makes the prompt for each step depend on hidden history.
- Letting a headless agent block on a question: it holds a process for as long as a person takes to answer, which is the cost the wait step exists to remove.
- The client hosts the agent, as Agent Session Manager does today: fits that app but every client must reimplement launching and state detection, and a pipeline or bot has no client to host it. It remains the model for interactive mode, where the host is outside the engine.
- An engine-hosted interactive terminal with client attach: it keeps a process and a pseudo-terminal alive for the life of the step and ties the engine to a host that can render terminals.

### D6. Run state is an append-only event log

Every state change is recorded as an event before it is acted on, and current state is computed from the log. This makes runs durable across restarts and multi-day review waits, auditable, and simple to stream to clients that reconnect from a sequence number. Storage sits behind the store protocol described in D10: a memory store and a local durable store first, with other stores possible for pipelines and hosted use.

The engine alone tracks the current step. This is the reason plugins stay small: a plugin never needs to know where in a workflow it was called.

### D7. Clients speak JSON-RPC over WebSocket or standard input and output

Clients list workflows and their graphs, subscribe to events, answer attention requests, and control runs. Because the protocol is the only interface, built-in clients get no private access and anyone can write another. The engine listens on loopback by default and requires a token for anything else.

The summary of a run in the protocol carries the three things people running many agents want: the current step, whether the run needs them or is waiting on something outside, and its artifacts. Clients also show the prompt each agent step received, so an operator can see exactly what a fresh session was told.

### D8. Plugin trust and distribution

Plugins are trusted programs. The engine shows the launch command before a plugin is first added, passes secrets only through environment variables that the workflow names, and records the exact version used by each run. Plugins resolve from a local path or a git reference, in the same spirit as referencing an action by repository and tag. No registry is built.

### D9. Reference workflow is in-repo and is the conformance bar

The OpenSpec process is written as a real workflow (see the appendix) and later shipped in the repository. It exercises every core mechanism: agent steps across two harnesses on named profiles, fresh sessions that hand work on through the branch and structured outputs, operator and review waits that rest for days, questions without explicit routing, bounded loops with resets, an invariant stop, sub-workflows, cleanup, artifacts, and stacked pull requests. The platform is conformant when that workflow runs end to end with recorded proof, including a run that rested at a review with no engine process running.

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
- Waiting runs are records in the store, not processes. A wait subscription is a record keyed by correlation key, so any instance that receives an event can find the runs it moves, and the first instance to append the resolution wins. The store also keeps definitions by digest, so an instance that takes over a run needs nothing from the machine that started it.
- Every record carries a namespace from the start, with a single namespace called default. Per-organization authorization is future work, but adding the field to every identifier later would be a data migration.
- The engine hands steps and sessions to a runner using the plugin protocol messages. The only runner in this change is in the same process, so nothing is serialized over a network, and a later change can add runners on other machines. The split follows the pattern of Temporal workers and CI runners, where the part that decides is separate from the part that does the work.

The store is a protocol and not a compiled-in interface. The built-in memory and local stores implement the same messages inside the binary, so the default stays one executable with nothing to install. Third-party stores are separate processes in any language, shared as a path or a git reference like any plugin. A compiled-in interface would force every community store to be written in Go and merged upstream or built into a custom executable, which is the opposite of how the rest of the platform is extended. Atomicity is not a problem over a protocol, because the whole contract is a few operations that are each atomic alone: append, read, claim, and list, plus optional watch and lease. The engine never needs a transaction across several calls. Latency is not a problem either, because the store sits behind steps that take seconds to days. The store protocol's operations include wait subscriptions and definitions beside append, read, claim, and list. A store declares whether it is durable, whether it is safe for many instances, and whether it can watch. The engine refuses setups that contradict the declaration.

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

### D11. Fresh steps get what they need from declared, durable channels

Because every agent step starts with no memory, the design says exactly how information reaches it. There are five channels, and each one survives a restart, a move to another machine, and days of rest:

1. Repository state. The branch and its files are the primary handoff. For the OpenSpec process, the change's proposal, design, specs, and tasks are the memory of the work, and prompts point the agent at them.
2. Structured step outputs. An agent step declares output fields, and the complete tool takes outputs validated against that declaration. The free-text summary is one standard output, and notes is another for anything that does not fit.
3. Route values. A transition can pass named values to its target, and the core ask route passes the answers and the previous attempt.
4. Artifacts and the trigger payload, which already exist in the run context.
5. Context lists. A step can list named references as context, and the engine renders each as a labelled section after the prompt.

```
  prompt text a step receives (built by the engine, shown to clients)

  +------------------------------------------------------------+
  | prefix            e.g. the OpenSpec explore command        |
  | prompt            written by the workflow author           |
  | context sections  one labelled section per context entry   |
  | re-entry sections questions and answers, previous attempt   |
  |                   (only after the ask route)                |
  | instructions      allowed outcomes, outputs each requires,  |
  |                   how to call the complete tool             |
  +------------------------------------------------------------+
```

Outputs are declared once per step and are required for every outcome except `questions`, where they are optional draft work. The reference workflow needs this: exploration outputs the plan and the change name, the plan gate shows the plan to the operator, and the revision path receives it again. An author never writes the plumbing for questions and answers because the engine renders it.

Alternative considered: resuming the harness's own conversation on re-entry, so the agent remembers more. Rejected for this change because behavior would then depend on whether a given harness, on a given machine, can resume. Outputs set a floor that is the same everywhere. A harness plugin could still use resume as an optimization later, without changing any spec.

### D12. Wait steps rest the run, and resume triggers move it on

Triggers do not only start runs. A wait step lists the triggers that can move the run on, and while it is pending the run rests in the store with no process running for it. The run might rest for a weekend or for two weeks of vacation.

```
  [step] --outcome--> [step] --outcome--> (wait step) --event--> [step]
   does work, exits     does work, exits    nothing runs:        does work, exits
                                            run rests in store
```

How an event finds the run that is resting:

```
  entering the wait                              later, an event arrives
  -----------------                              -----------------------
  resolve trigger config from run context        webhook, poll, or `workflow deliver`
  plugin derives correlation key repo#42                  |
  store: register subscription (key -> run, step)         v
  call check-now if the trigger offers it        plugin derives the same key from the event
  run state = waiting, nothing running           plugin classifies the event: outcome + outputs
                                                 store: find subscriptions for the key
                                                 engine: resolve the wait step, conditional append
                                                         -> run moves on along that outcome's route
```

- The plugin manifest declares per trigger the configuration schema, the payload schema, the outcomes the trigger can classify an event into, whether it can resume a wait, whether it offers check-now, and how it derives a key. The engine compares keys and knows nothing about what they mean, so the same mechanism serves a GitHub review, a Slack thread, a JIRA transition, and a clock.
- An event can happen before the run reaches the wait (a reviewer approves while the checks are still running). A trigger can offer a check-now call that the engine makes right after recording the subscription, and the wait resolves at once if the condition already holds.
- The first event recorded for a wait step wins, because resolving a wait is a conditional append like any other. Later events for a resolved wait are confirmed and ignored. One event can also start new runs through start triggers and resume waiting runs at the same time.
- A deadline is one more trigger on the wait. It is recorded as a deadline on the subscription, so whichever engine starts next delivers it even if none was running when the time passed.
- A gate is a wait whose trigger is the operator's answer. An interactive agent's completion is another built-in trigger.
- Polling triggers (a command run on an interval, a GitHub poll) are how a plugin implements a resume trigger when nothing can push events. A polling trigger needs an engine that is running, so a one-off `run` exits with a waiting status unless told to stay, and a `serve` instance polls for every waiting run.

How the engine is driven follows from this. It is stateless between events:

- `serve` subscribes plugins to start triggers and to the wait triggers of every waiting run, and resumes unfinished runs.
- `run` advances a run until it ends, needs the operator, or rests, then exits with a distinct status. With the memory store it refuses to rest, because nothing would remain to resume.
- `deliver` hands the engine one raw event, such as the payload of the event that started a GitHub Actions job, starts or resumes matching runs, advances them until they rest, and exits.
- `answer` records an operator's answer in the store and exits. The injected agent tools record their completion signals the same way. No engine needs to be running when someone answers or an interactive agent finishes, and the next engine to own the run acts on the signal.

Alternative considered: keeping the step active while it waits, with the plugin polling inside it. Rejected because it holds a process for days, makes restarts and takeover harder, and cannot be done at all in a short-lived environment such as a CI job.

### D13. Questions are an exit, and the default route asks and re-enters

Without help, every agent step would need a route for what happens when the agent has questions, and every workflow would be as complex as its most uncertain step. Instead `questions` is a core outcome with a core default route, in the same way `needs_human` is a core target.

```
   [explore] --plan_ready--> [plan_gate]
      |  ^
      |  |  re-enter explore, fresh, with the answers and the previous attempt
      |  |
      +--+  questions (core default route, drawn as a badge on the step, not an edge)
            = deliver the questions, rest, answers arrive, back to the asking step
```

- The core ask route opens an attention request holding the questions, delivers it to the workflow's attention channels (D15) and to connected clients, and rests. When the answers arrive it starts the asking step again in a fresh session with the answers and the previous attempt as route values. The bound comes from the step's `max_iterations`.
- A workflow can set defaults by outcome name (for example, a failed outcome always goes to needs-human), and a step can override one. A step can route `questions` somewhere else, such as to a different reviewer, or to failure in CI where no one can answer.
- Clients draw the default as a badge on the step, so the graph stays as simple as a workflow with no questions.
- `needs_human` uses the same mechanism with a question the engine writes ("this step failed, what now?") and answers that choose to retry or to go to a step.

An exit with questions is a full exit and carries outputs like any other, so the work done before asking is not thrown away.

```
 explore, attempt 1                                 explore, attempt 2 (fresh session)
 +---------------------------------------+          +----------------------------------------+
 | exit: questions                       |  answers | prompt (unchanged)                     |
 |   questions: [q1, q2]                 | -------> | context, added by the engine:          |
 |   outputs:   plan (draft), notes      |          |   "Your previous attempt": draft plan, |
 |   summary:   "started on X, unsure Y" |          |     notes, summary                     |
 |   files written to the branch remain  |          |   "Your questions and the answers"     |
 +---------------------------------------+          |   files on the branch are still there  |
                                                    +----------------------------------------+
```

- Declared outputs are required for the normal outcomes and optional for `questions`, where they are draft work. `notes` is a standard output for anything that does not fit the schema.
- The agent exits with questions by calling the complete tool with the outcome `questions`. If a turn ends with questions recorded and no completion, the turn-end policy re-prompts once to save the work so far and exit with the questions.
- Files written to the workspace stay for free because the re-run uses the same workspace. Every attempt's outputs stay in the event log, so a client can show the draft plan while the questions are open.

### D14. Catalog, pinned definitions, workspaces, and the webhook listener

A server that outlives any one process needs to know which workflows exist and what each run was started on. These pieces make that explicit.

- The engine keeps a catalog of workflows per namespace, each by name and definition digest. Operators and clients add, replace, and remove workflows at run time, and the engine subscribes start triggers for what is cataloged. Anything that can change a catalog (a form, a pull request merging a workflow file) goes through the same operations.
- A run records the digest of its definition and the store keeps that definition by digest while any run that pinned it is not final. Replacing a workflow never changes a run in flight, and a takeover by another instance needs no access to the original file. A sub-workflow is recorded by digest when the parent definition is resolved, so the whole tree is pinned.
- The engine gives each run a directory. A plugin step creates a checkout on a branch inside it (the GitHub plugin provides this), and a profile's working directory can reference the checkout, resolved each time an agent step starts. A workflow can name a cleanup sub-workflow that runs once at the end of the run, including after failure or cancellation, to remove the checkout. Steps that share files on one disk run on one runner, which is the only runner in this change.
- The webhook endpoint is a separate listener with its own address. It serves only paths that plugins registered, so it can face the internet while the client protocol stays on loopback or behind a token.

### D15. Attention channels deliver questions where people are

The engine delivers an attention request to clients that are connected, but a person may be in Slack, a ticket, or a pull request. An attention channel is a plugin capability that posts a request somewhere it can read replies, and reports a reply as an answer, which has the same effect as an answer from a client. A workflow names the channels it wants. The GitHub plugin provides one that comments on a pull request or issue. Who a request is for (an audience or an assignee) is not part of this change.

### D16. Loop budgets reset where a transition says so

Iteration counts last for the whole run, so a run that restarts from the beginning after a review would otherwise spend the inner loops' budgets on its second pass. A transition can list the steps whose counts are reset to zero when it is taken, and validation checks the names. A reference to a step's output resolves to its most recent resolution, and earlier attempts stay in the event log for clients.

### D17. What is deliberately left for later

These were found while checking the design against ten scenarios and are recorded so that they are decisions and not omissions.

- A concurrency policy for runs started by triggers, such as skipping a second run for the same ticket or queueing it. Deduplication of identical events exists, but not a policy across different events about the same thing.
- Secrets scoped per plugin and plugin processes scoped per workflow. Today secrets are the environment variables a workflow names, and a plugin process serves every workflow that declares it.
- Retention and deletion of finished runs.
- An audience or assignee on attention requests.
- Running the steps of one run in parallel.
- An HTTP health endpoint.
- Dynamic selection of a sub-workflow at run time, which validation rejects.
- How an interactive host is launched, whether by a host plugin capability or by the operator opening it by hand with a run-scoped token.

## Risks / Trade-offs

- **Harness protocol coverage differs by agent** → Start with the generic Agent Client Protocol plugin, check each target harness against it early (task group 7), and add dedicated plugins only for real gaps. Exact adapter commands for each harness are resolved at implementation time.
- **Headless mode loses each agent's native interface** → Questions are collected and presented when the step exits, and interactive mode remains available through a host outside the engine. Operators who want to talk to an agent mid-step use that mode.
- **Fresh sessions forget everything** → Say exactly how information moves (D11), show clients the prompt each step receives, preserve draft outputs on questions, and let the repository carry the state that matters. A harness plugin may use native resume as an optimization later without a spec change.
- **A step that has questions on every attempt could loop** → The ask route is bounded by the step's maximum iterations, and the run then asks for help with the count in the request.
- **A run can rest forever if no event ever comes** → Wait steps may declare a deadline trigger or timeout, and clients show how long each run has been waiting and on what.
- **An event arrives before the run reaches its wait** → Triggers offer a check-now call that the engine makes after recording the subscription, and the first resolution recorded wins.
- **A static, expression-free language may feel limiting** → Put logic in plugin steps, make writing one cheap through the shell plugin and a small manifest, and pass values along transitions so branching on data is rarely needed.
- **Agents may not call the complete tool** → A per-step policy re-prompts, exits with the questions recorded so far, or resolves with a named outcome, with a bound on re-prompts, after which the run asks for help.
- **Event log growth** → Events carry state changes, outputs, and summaries, not agent transcripts, so volume is low. Compaction can be added later since state is derivable. Retention for finished runs is deferred (D17).
- **Seams cost effort now for futures that may never ship** → Keep to the cheap ones (the store protocol, namespaces, recorded deadlines, subscriptions as records, the runner boundary in one process) and leave fleets, remote runners, and the managed service for later changes.
- **Plugins are trusted code** → Show the launch command, scope secrets by name, record versions, and defer sandboxing and per-plugin secret scoping. Third-party plugins should be read before use.
- **A webhook endpoint faces the internet** → It is a separate listener that serves only paths plugins registered, and plugins verify signatures. The client protocol stays on loopback or behind a token.
- **Senders must retry signals with unique keys, which burdens plugin authors** → The response to a JSON-RPC request is already the confirmation, and the SDK handles the retry loop so a small plugin needs no extra code.
- **Steps run at least once, not exactly once** → Say so in the plugin and harness contracts, key every signal so duplicates are no-ops, and keep steps that must not repeat idempotent on their own side.
- **A third-party store could corrupt runs** → Ship the conformance suite as a command that tests any store in any language, and make the engine refuse setups that contradict a store's declared capabilities.
- **While the store is unreachable the engine cannot advance runs** → Signals are held by their senders and recover on their own, clients see engine health and reads marked as possibly out of date, and no run is moved to needs-human because of a store outage.
- **IR versioning** → Version the IR from the start, reject unknown versions with a clear message, and keep the schema additive within a version.
- **Polling cost and latency for services like GitHub** → Plugins poll by default so they work anywhere, and accept webhooks through the engine when a public endpoint exists. A polling trigger needs a running engine, so a one-off run rests and a `serve` instance or a delivered event moves it on.
- **Large first release** → The task list orders work so each group is demonstrable alone: the engine runs workflows made of shell steps before any agent or service plugin exists.

## Migration Plan

There is nothing to migrate. Rollout is by task group; each group ends with a recorded demonstration, and the final group runs the reference workflow end to end. The author's existing skills remain usable until the reference workflow reaches parity.

## Open Questions

- Which adapter command, or native mode, each target harness uses with the generic Agent Client Protocol plugin. Resolved when task group 7 starts, without changing specs.
- How an interactive host is launched: by a host plugin capability, or by the operator opening it by hand with the prompt and a run-scoped token. The contract (hand off, rest, complete through the store) is the same either way.
- Whether the reference workflow should open the implementation pull request as a draft before the apply loops, so CI runs after every push as the author's execution-loop skill requires. The appendix follows the author's step list, which opens it after the scope check, and waits on the checks after the pull request exists. Either order is expressible without changing any spec.
- How feedback on the implementation pull request, which the author's list sends back to exploration, interacts with the branches already created. The appendix routes it to the same revision step as the proposal review, resetting the inner loop counts, which reruns the proposal steps in the checkout that is now on the implementation branch. A later change may add an implementation-specific revision path.
- Where the engine keeps its event log by default on each operating system. A task-group decision that does not affect the specs.

## Appendix: reference workflow

The OpenSpec process as the platform's first workflow. The notation is illustrative of the YAML format and must be validated by the validate command once it exists. Names after `uses` refer to steps and triggers provided by plugins. The engine's four step kinds are `agent`, `wait`, `workflow`, and plugin steps via `uses`. A wait step lists triggers, and the operator's answer is the built-in trigger `operator`. Operator answers expose their free text as `text`, agent steps expose their summary as `summary`, and every agent step on this page is a fresh session.

The exploration steps run on a read-only profile in a local clone that the run was given. The change's branch and checkout are created only after the plan is approved, and later agent steps run in that checkout. No step routes `questions`: the core ask route sends the questions to the operator and starts the asking step again with the answers and its previous attempt. The plan comes out of exploration as a structured output, so the plan gate can show it and the revision step can receive it. Where the workflow returns to exploration after a review, the transition resets the counts of the loops that follow it.

openspec-change.yaml:

```yaml
ir_version: 1
name: openspec-change
description: Take an idea through exploration, proposal, implementation, and archive, with human review at the plan and each pull request.

inputs:
  prompt: {type: string, required: true}
  repo: {type: string, required: true}
  workdir: {type: string, required: true}
  default_branch: {type: string, default: main}

plugins:
  github: {uses: builtin/github, env: [GITHUB_TOKEN]}
  openspec: {uses: builtin/openspec}
  shell: {uses: builtin/shell}
  claude: {uses: builtin/harness-acp, with: {command: ["claude-acp-adapter"]}}
  codex: {uses: builtin/harness-acp, with: {command: ["codex-acp-adapter"]}}

triggers:
  - uses: github/issue_labeled
    with: {repo: example/repository, label: workflow}
    inputs:
      prompt: "${{ trigger.title }}\n\n${{ trigger.body }}"
      repo: ${{ trigger.repo }}
      workdir: /srv/checkouts/example-repository

profiles:
  explorer:
    harness: claude
    mode: headless
    working_directory: ${{ inputs.workdir }}
    permissions: {allow: [read, search], deny: [edit, shell]}
  author:
    harness: claude
    mode: headless
    working_directory: ${{ run.directory }}/checkout
    permissions: {allow: [read, search, edit, "shell:git", "shell:openspec"], deny: ["shell:rm"]}
  reviewer:
    harness: codex
    mode: headless
    working_directory: ${{ run.directory }}/checkout
    permissions: {allow: [read, search, "shell:git"], deny: [edit]}
  checker:
    harness: claude
    mode: headless
    working_directory: ${{ run.directory }}/checkout
    permissions: {allow: [read, search], deny: [edit, shell]}

artifacts:
  change: {kind: openspec/change}
  proposal_pr: {kind: github/pull_request}
  implementation_pr: {kind: github/pull_request}
  archive_pr: {kind: github/pull_request}

defaults:
  failed: needs_human

cleanup:
  workflow: ./cleanup.yaml
  inputs: {checkout: "${{ run.directory }}/checkout"}

start: explore

steps:
  explore:
    type: agent
    profile: explorer
    prefix: ${{ plugins.openspec.prefix.explore }}
    prompt: ${{ inputs.prompt }}
    outputs: {plan: {type: string}, change: {type: string}}
    outcomes: [plan_ready]
    max_iterations: 5
    on:
      plan_ready: {goto: plan_gate, pass: {plan: "${{ steps.explore.outputs.plan }}"}}
    default: needs_human

  revise:
    type: agent
    profile: explorer
    prefix: ${{ plugins.openspec.prefix.explore }}
    prompt: "Continue exploring the change ${{ steps.explore.outputs.change }} and revise the plan using the feedback. The files of the change in the repository supersede the plan if they differ."
    context:
      - {name: Feedback, value: "${{ route.feedback }}"}
      - {name: Plan so far, value: "${{ route.plan }}"}
    outputs: {plan: {type: string}}
    outcomes: [plan_ready]
    max_iterations: 10
    on:
      plan_ready: {goto: plan_gate, pass: {plan: "${{ steps.revise.outputs.plan }}"}}
    default: needs_human

  plan_gate:
    type: wait
    triggers:
      - uses: operator
        with:
          prompt: Is the plan ready to be written up as a proposal?
          show: ${{ route.plan }}
          outcomes: [approved, feedback]
    on:
      approved: branch
      feedback: {goto: revise, pass: {feedback: "${{ steps.plan_gate.outputs.text }}", plan: "${{ route.plan }}"}}

  branch:
    uses: github/create_checkout
    with:
      repo: ${{ inputs.repo }}
      branch: ${{ steps.explore.outputs.change }}
      base: ${{ inputs.default_branch }}
      directory: ${{ run.directory }}/checkout
    on: {created: propose, reused: propose}
    default: needs_human

  propose:
    type: agent
    profile: author
    prefix: ${{ plugins.openspec.prefix.propose }}
    prompt: Write the OpenSpec change named ${{ steps.explore.outputs.change }}. Do not implement anything.
    context:
      - {name: Approved plan, value: "${{ steps.explore.outputs.plan }}"}
    outcomes: [proposed]
    max_iterations: 5
    on: {proposed: proposal_push}
    default: needs_human

  proposal_push:
    type: workflow
    workflow: ./commit-and-push.yaml
    inputs: {workdir: "${{ run.directory }}/checkout", message: "Proposal for ${{ steps.explore.outputs.change }}"}
    max_iterations: 20
    on: {succeeded: open_proposal_pr, failed: needs_human, cancelled: cancelled}

  open_proposal_pr:
    uses: github/open_pull_request
    with:
      repo: ${{ inputs.repo }}
      workdir: ${{ run.directory }}/checkout
      base: ${{ inputs.default_branch }}
      title_prefix: Proposal
      draft: true
    expects: [{name: proposal_pr, kind: github/pull_request}]
    on: {opened: proposal_checks, reused: proposal_checks}
    default: needs_human

  proposal_checks:
    type: wait
    triggers:
      - uses: github/checks
        with: {pull_request: "${{ artifacts.proposal_pr.ref }}"}
    on: {green: validate_proposal, red: proposal_ci_fix}
    default: needs_human

  proposal_ci_fix:
    type: agent
    profile: author
    prompt: Fix the failing checks and nothing else.
    context:
      - {name: Failing checks, value: "${{ steps.proposal_checks.outputs.failures }}"}
    outcomes: [fixed]
    max_iterations: 5
    on: {fixed: proposal_push}
    default: needs_human

  validate_proposal:
    uses: openspec/validate
    with: {workdir: "${{ run.directory }}/checkout"}
    on: {valid: ready_proposal, invalid: fix_proposal}
    default: needs_human

  fix_proposal:
    type: agent
    profile: author
    prompt: Fix the OpenSpec validation problems and nothing else.
    context:
      - {name: Validation problems, value: "${{ steps.validate_proposal.outputs.messages }}"}
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
    type: wait
    triggers:
      - uses: github/review
        with: {pull_request: "${{ artifacts.proposal_pr.ref }}"}
    on:
      approved: invariants
      changes_requested: &revise_proposal
        goto: revise
        pass: {feedback: "${{ steps.proposal_review.outputs.comments }}", plan: "${{ steps.explore.outputs.plan }}"}
        reset: [propose, proposal_ci_fix, fix_proposal]
      commented: *revise_proposal
      closed: cancelled

  invariants:
    uses: openspec/check_invariants
    with:
      workdir: ${{ run.directory }}/checkout
      change: ${{ steps.explore.outputs.change }}
      default_branch: ${{ inputs.default_branch }}
    on: {ok: pr_state, violated: needs_human}
    default: needs_human

  pr_state:
    uses: github/pull_request_state
    with: {pull_request: "${{ artifacts.proposal_pr.ref }}"}
    on: {open_approved_green: impl_branch, not_ready: needs_human}
    default: needs_human

  impl_branch:
    uses: shell/run
    with:
      cwd: ${{ run.directory }}/checkout
      command: [git, switch, -c, "implement/${{ steps.explore.outputs.change }}"]
    on: {success: apply, failure: needs_human}
    default: needs_human

  apply:
    type: agent
    profile: author
    prefix: ${{ plugins.openspec.prefix.apply }}
    prompt: Apply the change ${{ steps.explore.outputs.change }}.
    outcomes: [applied]
    max_iterations: 50
    on: {applied: apply_push}
    default: needs_human

  apply_push:
    type: workflow
    workflow: ./commit-and-push.yaml
    inputs: {workdir: "${{ run.directory }}/checkout", message: "Apply ${{ steps.explore.outputs.change }}"}
    max_iterations: 50
    on: {succeeded: tasks_left, failed: needs_human, cancelled: cancelled}

  tasks_left:
    uses: openspec/task_progress
    with: {workdir: "${{ run.directory }}/checkout", change: "${{ steps.explore.outputs.change }}"}
    on: {tasks_remaining: apply, complete: verify}
    default: needs_human

  verify:
    type: agent
    profile: author
    prefix: ${{ plugins.openspec.prefix.verify }}
    prompt: Verify the implementation of ${{ steps.explore.outputs.change }} and report issues.
    outcomes: [issues_found, clean]
    max_iterations: 20
    on:
      issues_found: {goto: fix_verify, pass: {findings: "${{ steps.verify.outputs.summary }}"}}
      clean: review
    default: needs_human

  fix_verify:
    type: agent
    profile: author
    prompt: Fix the verification issues.
    context:
      - {name: Verification issues, value: "${{ route.findings }}"}
    outcomes: [fixed]
    max_iterations: 20
    on: {fixed: verify_push}
    default: needs_human

  verify_push:
    type: workflow
    workflow: ./commit-and-push.yaml
    inputs: {workdir: "${{ run.directory }}/checkout", message: "Fix verification issues"}
    max_iterations: 20
    on: {succeeded: verify, failed: needs_human, cancelled: cancelled}

  review:
    type: agent
    profile: reviewer
    prompt: Review the work for ${{ steps.explore.outputs.change }} adversarially. Assume something is wrong and find what. Report only actionable findings.
    outcomes: [findings, no_findings]
    max_iterations: 10
    on:
      findings: {goto: fix_review, pass: {findings: "${{ steps.review.outputs.summary }}"}}
      no_findings: scope_check
    default: needs_human

  fix_review:
    type: agent
    profile: author
    prompt: Fix the review findings.
    context:
      - {name: Review findings, value: "${{ route.findings }}"}
    outcomes: [fixed]
    max_iterations: 10
    on: {fixed: review_push}
    default: needs_human

  review_push:
    type: workflow
    workflow: ./commit-and-push.yaml
    inputs: {workdir: "${{ run.directory }}/checkout", message: "Address review findings"}
    max_iterations: 10
    on: {succeeded: review, failed: needs_human, cancelled: cancelled}

  scope_check:
    type: agent
    profile: checker
    prompt: Compare the work with the proposal, design, and specs for ${{ steps.explore.outputs.change }}. Report scope creep or missing requirements. Edge cases and invariants are not scope creep.
    outputs: {findings: {type: string}}
    outcomes: [clean, scope_creep, missing_requirements]
    on: {clean: impl_push, scope_creep: needs_human, missing_requirements: needs_human}

  impl_push:
    type: workflow
    workflow: ./commit-and-push.yaml
    inputs: {workdir: "${{ run.directory }}/checkout", message: "Implementation of ${{ steps.explore.outputs.change }}"}
    max_iterations: 20
    on: {succeeded: open_impl_pr, failed: needs_human, cancelled: cancelled}

  open_impl_pr:
    uses: github/open_pull_request
    with:
      repo: ${{ inputs.repo }}
      workdir: ${{ run.directory }}/checkout
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
    type: wait
    triggers:
      - uses: github/checks
        with: {pull_request: "${{ artifacts.implementation_pr.ref }}"}
    on: {green: impl_review, red: impl_ci_fix}
    default: needs_human

  impl_ci_fix:
    type: agent
    profile: author
    prompt: Fix the failing checks and nothing else.
    context:
      - {name: Failing checks, value: "${{ steps.impl_checks.outputs.failures }}"}
    outcomes: [fixed]
    max_iterations: 5
    on: {fixed: impl_ci_push}
    default: needs_human

  impl_ci_push:
    type: workflow
    workflow: ./commit-and-push.yaml
    inputs: {workdir: "${{ run.directory }}/checkout", message: "Fix failing checks"}
    max_iterations: 5
    on: {succeeded: impl_checks, failed: needs_human, cancelled: cancelled}

  impl_review:
    type: wait
    triggers:
      - uses: github/review
        with: {pull_request: "${{ artifacts.implementation_pr.ref }}"}
    on:
      approved: mark_impl_ready
      changes_requested: &revise_implementation
        goto: revise
        pass: {feedback: "${{ steps.impl_review.outputs.comments }}", plan: "${{ steps.explore.outputs.plan }}"}
        reset: [propose, proposal_ci_fix, fix_proposal, apply, fix_verify, verify, review, fix_review, impl_ci_fix]
      commented: *revise_implementation
      closed: cancelled

  mark_impl_ready:
    uses: github/mark_ready
    with: {pull_request: "${{ artifacts.implementation_pr.ref }}"}
    on: {done: archive_branch}
    default: needs_human

  archive_branch:
    uses: shell/run
    with:
      cwd: ${{ run.directory }}/checkout
      command: [git, switch, -c, "archive/${{ steps.explore.outputs.change }}"]
    on: {success: archive, failure: needs_human}
    default: needs_human

  archive:
    uses: openspec/archive
    with: {workdir: "${{ run.directory }}/checkout", change: "${{ steps.explore.outputs.change }}"}
    on: {done: archive_push, failed: needs_human}
    default: needs_human

  archive_push:
    type: workflow
    workflow: ./commit-and-push.yaml
    inputs: {workdir: "${{ run.directory }}/checkout", message: "Archive ${{ steps.explore.outputs.change }}"}
    on: {succeeded: open_archive_pr, failed: needs_human, cancelled: cancelled}

  open_archive_pr:
    uses: github/open_pull_request
    with:
      repo: ${{ inputs.repo }}
      workdir: ${{ run.directory }}/checkout
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

cleanup.yaml, the sub-workflow that removes the checkout after the run ends:

```yaml
ir_version: 1
name: cleanup

inputs:
  checkout: {type: string, required: true}

plugins:
  github: {uses: builtin/github}

start: remove

steps:
  remove:
    uses: github/remove_checkout
    with: {path: "${{ inputs.checkout }}"}
    on: {removed: succeeded, absent: succeeded, failed: failed}
```

How the appendix maps to the author's original list:

- Explore, feedback keeps exploring, approval goes forward: `explore`, `plan_gate`, `revise`. A question from exploration needs no step, because the core ask route handles it.
- Create the branch only after the plan is approved: `branch`.
- Propose, open draft pull request, wait for checks, wait for human review, feedback returns to explore: `propose` through `proposal_review`. The wait for review rests the run with nothing running, so a Friday review may arrive on Monday.
- Invariants hold or stop and ask: `invariants`, `pr_state`, with `needs_human`.
- Loop apply until no tasks remain: `apply`, `apply_push`, `tasks_left`.
- Loop verify until no issues: `verify`, `fix_verify`, `verify_push`.
- Loop adversarial review until no actionable findings: `review`, `fix_review`, `review_push`.
- Scope check: `scope_check`.
- Open implementation pull request, wait for checks and review, feedback returns to explore, approval goes forward, mark ready, open archive pull request: `open_impl_pr` through `stack_archive`.
- Remove the checkout whatever happens: `cleanup`.
