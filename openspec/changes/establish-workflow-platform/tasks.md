## 1. Repo skeleton and guardrails

- [ ] 1.1 Create the Go module, the directory layout for the engine, the executable, built-in plugins, and clients, and a build that produces one executable
- [ ] 1.2 Add lint, formatting, and a check that bans code comments
- [ ] 1.3 Add the check that bans fixed-width prose, and the OpenSpec validation check, to CI as required checks
- [ ] 1.4 Publish the JSON Schemas for the IR, the plugin protocol, the harness protocol, the client protocol, and the store protocol, and generate Go and TypeScript types from them
- [ ] 1.5 Add a test that fails when generated types differ from the schemas
- [ ] 1.6 Write a short contributor guide that points at the specs and the design
- [ ] 1.7 Manual proof: a pull request showing every check running and failing on a deliberate violation, then passing

## 2. Workflow definition (workflow-definition)

- [ ] 2.1 Implement the IR types and schema validation, including the version check
- [ ] 2.2 Implement the YAML loader that maps one to one onto the IR, with file and line in errors
- [ ] 2.3 Implement the four step kinds (plugin, agent, wait, workflow), default routes at step and workflow level, and route values on transitions
- [ ] 2.4 Implement reference parsing, the no-expressions rule, and path availability analysis, including run metadata and the run directory
- [ ] 2.5 Implement terminal states, the needs-human and ask targets, the built-in outcomes, and loop bound analysis with reset lists
- [ ] 2.6 Implement agent profiles, agent step outputs per outcome, context lists, and working directories that reference earlier steps
- [ ] 2.7 Implement wait steps with their triggers and timeouts, and start triggers with input mapping
- [ ] 2.8 Implement plugin declarations, sub-workflows with profile bindings, the cleanup sub-workflow, and expected artifacts at workflow and step level
- [ ] 2.9 Implement static validation against plugin manifests: declared outcomes routed, reachability, valid step and trigger inputs, triggers that support waiting, and required inputs supplied by start triggers
- [ ] 2.10 Implement the graph export used by clients, showing waits and marking the ask and needs-human routes without drawing an edge for each
- [ ] 2.11 Test every scenario in the specification with fixture workflows, including every rejection
- [ ] 2.12 Manual proof: validate a good and a bad workflow from the command line and show the messages

## 3. Run engine and store (run-engine, run-store)

- [ ] 3.1 Implement the event types and the engine side of the store protocol: conditional append with conflict handling, read, dedupe claim, list, wait subscriptions, and definitions
- [ ] 3.2 Implement the built-in memory store and local durable store using the same store messages, with declared capabilities
- [ ] 3.3 Implement state derivation from the log and resume on taking ownership, reloading after an append conflict
- [ ] 3.4 Implement run creation from triggers and clients, with dedupe through the store's claim
- [ ] 3.5 Implement the pinned definition: record the digest at run start, keep definitions by digest while a run that pinned them is not final, and pin sub-workflows
- [ ] 3.6 Implement the workflow catalog: add, replace, and remove at run time, with start triggers subscribed for what is cataloged
- [ ] 3.7 Implement step dispatch, outcome routing, default routes, built-in outcomes, and route values
- [ ] 3.8 Implement loop accounting, reset lists, and the needs-human stop on a bound
- [ ] 3.9 Implement timeouts and deadlines as recorded deadlines that resolve exactly once, including across a restart and when no engine was running at the time
- [ ] 3.10 Implement wait steps: register subscriptions with correlation keys on entry, call check-now, rest the run with nothing running, and remove subscriptions when the wait resolves
- [ ] 3.11 Implement resuming from events: derive the key, find the subscriptions, resolve the first event by conditional append, ignore later events, and let one event start runs and resume runs together
- [ ] 3.12 Implement the questions exit and the core ask route, including the attention request, rest, and re-entry of the asking step with the answers and the previous attempt
- [ ] 3.13 Implement attention requests, the operator answer as a trigger, needs-human requests, and answers that record without a running engine
- [ ] 3.14 Implement the artifact registry with expected, available, and probed states
- [ ] 3.15 Implement the run directory and the cleanup sub-workflow that runs once when a run reaches a final state, including after failure and cancellation
- [ ] 3.16 Implement cancel, retry, and concurrent runs
- [ ] 3.17 Implement sub-workflow execution and the nested step detail
- [ ] 3.18 Implement the run summary that answers the three questions and shows what a waiting run waits on
- [ ] 3.19 Implement signal confirmation after recording, with unique keys and no effect for a repeated key
- [ ] 3.20 Implement namespaces on every record, with default as the namespace when none is given
- [ ] 3.21 Implement the runner boundary with the in-process runner, and the re-attach or restart rules when a runner is gone
- [ ] 3.22 Implement engine health, store restart limits, and the rule that a store failure never moves a run to needs-human
- [ ] 3.23 Implement store capability checks that refuse unsafe setups, including refusing to rest a run on the memory store, and the resolution of third-party stores like plugins
- [ ] 3.24 Implement the conformance suite, including subscriptions and definitions, and run it against both built-in stores
- [ ] 3.25 Write a minimal third-party store in a language other than Go that passes the conformance suite
- [ ] 3.26 Test the engine with an in-process fake plugin: every scenario in the specification, plus a crash and resume in the middle of a wait, an event that arrives before the wait is entered, an event that arrives with no engine running, two engine instances racing on a shared store, and a store that fails and recovers
- [ ] 3.27 Manual proof: run a small workflow built from fake steps, stop the engine during a wait, restart it, and show the run continuing
- [ ] 3.28 Manual proof: rest a run at a wait with no engine process running, then deliver an event and show the run resume and end
- [ ] 3.29 Manual proof: kill the store adapter during a run and show it recover, then make the store unreachable while a step completes and show the run continue on its own once the store returns, with the unhealthy state shown in a client

## 4. Plugin host (plugin-protocol)

- [ ] 4.1 Implement the manifest format and its schema, including per trigger the configuration and payload schemas, outcomes, wait support, check-now, and key derivation
- [ ] 4.2 Implement the process host: launch, handshake, capability check, shutdown
- [ ] 4.3 Implement step execution that does work and exits, progress, long-running steps, and cancellation
- [ ] 4.4 Implement start triggers and their events, with dedupe keys
- [ ] 4.5 Implement resume triggers: subscription for a wait, key derivation from configuration and from events, classification into outcomes and outputs, and check-now
- [ ] 4.6 Implement delivery of events from outside the engine, so the deliver command reaches the same trigger code as a webhook
- [ ] 4.7 Implement the webhook listener as its own address that serves only paths plugins registered, with a retryable error when an event cannot be recorded
- [ ] 4.8 Implement attention channels, artifact reporting, probing, and logging
- [ ] 4.9 Implement keyed signals and the retry loop in the SDK helpers, so a plugin holds a signal until it is confirmed
- [ ] 4.10 Implement crash handling and restart behavior, including subscribing triggers again
- [ ] 4.11 Implement plugin resolution from a local path and a git reference, the launch command confirmation, and recording the version used by each run
- [ ] 4.12 Write a minimal example plugin in a language other than Go that passes the protocol tests, including a step, a start trigger, and a resume trigger
- [ ] 4.13 Manual proof: install the example plugin from a git reference and run a workflow that uses it, including a wait resumed by its trigger

## 5. Client protocol server and command line (client-protocol, workflow-cli)

- [ ] 5.1 Implement the WebSocket and standard input and output transports, loopback binding, and token requirement for other binds
- [ ] 5.2 Implement discovery of a running engine
- [ ] 5.3 Implement workflow, run, attention, and artifact methods, the event subscription with resume from a sequence number, and client identity
- [ ] 5.4 Implement waiting detail, questions with draft work, the retrieval of the prompt each agent step received, and the launch details of an externally hosted interactive agent
- [ ] 5.5 Implement protocol versioning
- [ ] 5.6 Implement the serve and run commands, including the waiting exit status and the option to stay and host polling triggers
- [ ] 5.7 Implement the deliver command, which starts or resumes matching runs from one raw event, advances them until they rest, and exits
- [ ] 5.8 Implement the answer command, which records an answer in the store and exits
- [ ] 5.9 Implement the validate, status, workflow catalog, plugin, and export commands with the documented exit codes
- [ ] 5.10 Implement the client launcher commands
- [ ] 5.11 Implement config and state location handling, store selection, the namespace option, and the store conformance command
- [ ] 5.12 Implement namespaces and engine health in the client protocol, and the not-recorded answer for answers and controls
- [ ] 5.13 Test every scenario with a scripted client
- [ ] 5.14 Manual proof: drive a run with a scripted client over both transports and show the event stream and an answered attention request
- [ ] 5.15 Manual proof: from a shell with no engine running, deliver an event to a resting run and answer an open request, and show the run move on at the next start

## 6. Shell plugin (plugin-shell)

- [ ] 6.1 Implement the run step with command and argument arrays, working directory, environment, and no shell by default
- [ ] 6.2 Implement outcome mapping by exit code and output pattern, with unmatched fallback
- [ ] 6.3 Implement timeout and cancellation, including child process cleanup
- [ ] 6.4 Implement the polling trigger that runs a command on an interval and resumes a wait or starts a run from its result
- [ ] 6.5 Test every scenario in the specification
- [ ] 6.6 Manual proof: run a workflow made only of shell steps with a loop, a timeout, a failure route, and a wait resumed by a polling trigger

## 7. Harness contract and generic agent plugin (harness-protocol, plugin-harness-acp)

- [ ] 7.1 Implement the session lifecycle in which a session lives for one step, the headless requirement, the interactive capability declaration, and validation of declared modes
- [ ] 7.2 Implement profiles: harness, mode, working directory resolved when the step starts, and the permission policy
- [ ] 7.3 Implement the injected tool server with the complete, record-questions, and artifact tools, including outputs validated against the step's declaration, the undeclared outcome error, and holding an unconfirmed outcome for retry
- [ ] 7.4 Implement outputs per outcome: required for normal outcomes, optional draft work for questions, and the standard summary and notes
- [ ] 7.5 Implement prompt construction from the prefix, prompt, context sections, re-entry sections with answers and the previous attempt, and outcome and output instructions, and expose the result to clients
- [ ] 7.6 Implement the turn-end policies, including saving work and exiting with the recorded questions, and the re-prompt limit
- [ ] 7.7 Implement the permission policy interpretation: allow and deny rules, a denial that reaches the agent as an ordinary tool error, and no request that waits for a person
- [ ] 7.8 Implement interactive mode as a hand-off: hand the prompt and a run-scoped token to an external host, exit with launched, and accept the completion signal recorded in the store
- [ ] 7.9 Implement mixed harnesses in one run, the activity status, and session failure handling
- [ ] 7.10 Implement the generic plugin that drives an Agent Client Protocol agent, mapping permission requests to the policy and questions to the recorded questions
- [ ] 7.11 Check each target harness against the generic plugin, record the adapter command or native mode that works for each, and list the gaps
- [ ] 7.12 Add dedicated plugins only for gaps that block the reference workflow
- [ ] 7.13 Test every scenario with a scripted fake agent, and test the plugin against at least two real harnesses
- [ ] 7.14 Manual proof: a workflow with two profiles on two different harnesses in which one step exits with questions and a draft, the answers are given, and the fresh session continues from the draft, with recordings
- [ ] 7.15 Manual proof: a workflow whose interactive step is hosted outside the engine, completes through the store while no engine runs, and resumes the run

## 8. GitHub plugin (plugin-github)

- [ ] 8.1 Implement identity reporting and the permission denied behavior
- [ ] 8.2 Implement the start triggers for an issue opened, an issue labeled, and a comment, with label and author filters and dedupe, by polling
- [ ] 8.3 Implement the review trigger with the outcomes approved, changes_requested, commented, and closed, reviewer filters, correlation keys, and check-now
- [ ] 8.4 Implement the checks trigger with the outcomes green and red, the failures output, correlation keys, and check-now
- [ ] 8.5 Implement the pull request comment trigger
- [ ] 8.6 Implement webhook delivery through the engine for every trigger, and polling as the fallback
- [ ] 8.7 Implement the branch workspace steps: create a checkout on a branch, reuse an existing branch, and remove the checkout
- [ ] 8.8 Implement open pull request, mark ready, check status, pull request state, review comments resolved, and merge
- [ ] 8.9 Implement the issue steps and the issue artifact kind
- [ ] 8.10 Implement the attention channel that posts a request as a comment and reports replies as answers
- [ ] 8.11 Implement the stacked pull request steps using the stack extension, with the unavailable outcome
- [ ] 8.12 Implement the pull request artifact kind and its probe
- [ ] 8.13 Test every scenario against a recorded fake API and against a real scratch repository
- [ ] 8.14 Manual proof: in a scratch repository, a workflow that opens a draft pull request, rests with no process running until a real review arrives, handles changes requested and approval, and links a stack
- [ ] 8.15 Manual proof: a question posted to a pull request through the attention channel and answered by a reply

## 9. OpenSpec plugin (plugin-openspec)

- [ ] 9.1 Implement validate, task progress, and proposal presence checks
- [ ] 9.2 Implement the invariant checks
- [ ] 9.3 Implement archive
- [ ] 9.4 Implement the change artifact kind and the skill prefix values
- [ ] 9.5 Test every scenario against fixture repositories
- [ ] 9.6 Manual proof: run the plugin steps against a scratch repository with a valid change, an invalid change, and unchecked tasks

## 10. Schedule plugin (plugin-schedule)

- [ ] 10.1 Implement the recurring start trigger with time zones and the documented missed-firing behavior
- [ ] 10.2 Implement the one-shot start trigger
- [ ] 10.3 Implement the delay and deadline resume trigger with check-now, recorded as a deadline so delivery does not depend on a process having run
- [ ] 10.4 Test every scenario, including a moment that passes while no engine runs
- [ ] 10.5 Manual proof: a recurring run that starts on schedule, and a run that rests on a delay and resumes when it elapses

## 11. Terminal client (client-tui)

- [ ] 11.1 Implement the overview of runs with the current step, the needs-you indicator, what a waiting run waits on, and artifact links
- [ ] 11.2 Implement the run view with the live graph
- [ ] 11.3 Implement answering attention requests inline, with the draft work shown beside questions
- [ ] 11.4 Implement the view of the prompt and context each agent step received
- [ ] 11.5 Implement the run controls, the launch details for externally hosted interactive agents, and reconnect
- [ ] 11.6 Test with a scripted engine and screenshot comparisons of the main views
- [ ] 11.7 Manual proof: use the client against several concurrent runs, including a waiting run, and record the three questions being answered at a glance

## 12. Web client (client-web)

- [ ] 12.1 Serve the client from the engine and implement run cards, including waiting runs
- [ ] 12.2 Implement the live graph with the current step highlighted and the ask route drawn as a badge
- [ ] 12.3 Implement the attention inbox and the title count, with the draft work shown beside questions
- [ ] 12.4 Implement the view of the prompt and context each agent step received
- [ ] 12.5 Implement artifact links, the launch details for externally hosted interactive agents, and reconnect
- [ ] 12.6 Meet the accessibility requirements and test them with an automated audit and keyboard use
- [ ] 12.7 Test with a scripted engine and browser tests
- [ ] 12.8 Manual proof: use the client in a browser against several concurrent runs, including a waiting run, and record the three questions being answered at a glance

## 13. Go SDK (workflow-sdk-go)

- [ ] 13.1 Implement the builder that emits the IR, including profiles, wait steps, triggers, outputs, and context
- [ ] 13.2 Implement fragments and reuse
- [ ] 13.3 Implement the export command that turns YAML into SDK code
- [ ] 13.4 Test round-trip parity between YAML and the SDK for every fixture workflow
- [ ] 13.5 Manual proof: export the reference workflow to code, build it, and show the identical IR

## 14. Reference workflow end to end (openspec-workflow)

- [ ] 14.1 Add the reference workflow and its sub-workflows to the repository, based on the design appendix, and make them pass validation
- [ ] 14.2 Resolve the open questions in the design that affect the workflow, and update the design with the answers
- [ ] 14.3 Test each phase against fakes: exploration that asks questions with a draft, plan feedback, proposal checks red, review changes requested with the loop counts reset, a review that arrives after a long rest, invariant failure, loops, scope check, cleanup after cancellation, and permission denied
- [ ] 14.4 Run the workflow end to end on a real repository with real harnesses, from a prompt and from an issue
- [ ] 14.5 Capture recordings or screenshots from the terminal client and the web client at several points, including a run that rested at a review with no engine process running and resumed from a delivered event
- [ ] 14.6 Attach the recordings to the pull request that completes this group
- [ ] 14.7 Manual proof: the recordings from 14.5 are the proof
