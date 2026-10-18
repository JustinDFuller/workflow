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
- [ ] 2.3 Implement the four step kinds, default routes, and route values on transitions
- [ ] 2.4 Implement reference parsing, the no-expressions rule, and path availability analysis
- [ ] 2.5 Implement terminal states, the needs-human target, and loop bound analysis
- [ ] 2.6 Implement named sessions, sub-workflows with session bindings, and expected artifacts at workflow and step level
- [ ] 2.7 Implement static validation against plugin manifests: declared outcomes routed, reachability, valid step inputs
- [ ] 2.8 Implement the graph export used by clients
- [ ] 2.9 Test every scenario in the specification with fixture workflows, including every rejection
- [ ] 2.10 Manual proof: validate a good and a bad workflow from the command line and show the messages

## 3. Run engine and store (run-engine, run-store)

- [ ] 3.1 Implement the event types and the engine side of the store protocol: conditional append with conflict handling, read, dedupe claim, and list
- [ ] 3.2 Implement the built-in memory store and local durable store using the same store messages, with declared capabilities
- [ ] 3.3 Implement state derivation from the log and resume on taking ownership, reloading after an append conflict
- [ ] 3.4 Implement run creation from triggers and clients, with dedupe through the store's claim
- [ ] 3.5 Implement step dispatch, outcome routing, default routes, and route values
- [ ] 3.6 Implement loop accounting and the needs-human stop on a bound
- [ ] 3.7 Implement timeouts as recorded deadlines that resolve with the timed_out outcome exactly once, including across a restart
- [ ] 3.8 Implement attention requests, human gates, and answers
- [ ] 3.9 Implement the artifact registry with expected, available, and probed states
- [ ] 3.10 Implement cancel, retry, and concurrent runs
- [ ] 3.11 Implement sub-workflow execution and the nested step detail
- [ ] 3.12 Implement the run summary that answers the three questions
- [ ] 3.13 Implement signal confirmation after recording, with unique keys and no effect for a repeated key
- [ ] 3.14 Implement namespaces on every record, with default as the namespace when none is given
- [ ] 3.15 Implement the runner boundary with the in-process runner, and the re-attach or restart rules when a runner is gone
- [ ] 3.16 Implement engine health, store restart limits, and the rule that a store failure never moves a run to needs-human
- [ ] 3.17 Implement store capability checks that refuse unsafe setups, and the resolution of third-party stores like plugins
- [ ] 3.18 Implement the conformance suite and run it against both built-in stores
- [ ] 3.19 Write a minimal third-party store in a language other than Go that passes the conformance suite
- [ ] 3.20 Test the engine with an in-process fake plugin: every scenario in the specification, plus a crash and resume in the middle of a wait, two engine instances racing on a shared store, and a store that fails and recovers
- [ ] 3.21 Manual proof: run a small workflow built from fake steps, stop the engine during a wait, restart it, and show the run continuing
- [ ] 3.22 Manual proof: kill the store adapter during a run and show it recover, then make the store unreachable while a step completes and show the run continue on its own once the store returns, with the unhealthy state shown in a client

## 4. Plugin host (plugin-protocol)

- [ ] 4.1 Implement the manifest format and its schema
- [ ] 4.2 Implement the process host: launch, handshake, capability check, shutdown
- [ ] 4.3 Implement step execution, progress, long-running steps, and cancellation
- [ ] 4.4 Implement trigger subscription and events, and webhook delivery through the engine, with a retryable error when an event cannot be recorded
- [ ] 4.5 Implement artifact reporting, probing, attention from plugins, and logging
- [ ] 4.6 Implement keyed signals and the retry loop in the SDK helpers, so a plugin holds a signal until it is confirmed
- [ ] 4.7 Implement crash handling and restart behavior
- [ ] 4.8 Implement plugin resolution from a local path and a git reference, the launch command confirmation, and recording the version used by each run
- [ ] 4.9 Write a minimal example plugin in a language other than Go that passes the protocol tests
- [ ] 4.10 Manual proof: install the example plugin from a git reference and run a workflow that uses it

## 5. Client protocol server and command line (client-protocol, workflow-cli)

- [ ] 5.1 Implement the WebSocket and standard input and output transports, loopback binding, and token requirement for other binds
- [ ] 5.2 Implement discovery of a running engine
- [ ] 5.3 Implement workflow, run, attention, and artifact methods, the event subscription with resume from a sequence number, and client identity
- [ ] 5.4 Implement protocol versioning
- [ ] 5.5 Implement the serve, run, validate, status, plugin, and export commands with the documented exit codes
- [ ] 5.6 Implement the client launcher commands
- [ ] 5.7 Implement config and state location handling, store selection, the namespace option, and the store conformance command
- [ ] 5.8 Implement namespaces and engine health in the client protocol, and the not-recorded answer for answers and controls
- [ ] 5.9 Test every scenario with a scripted client
- [ ] 5.10 Manual proof: drive a run with a scripted client over both transports and show the event stream and an answered attention request

## 6. Shell plugin (plugin-shell)

- [ ] 6.1 Implement the run step with command and argument arrays, working directory, environment, and no shell by default
- [ ] 6.2 Implement outcome mapping by exit code and output pattern, with unmatched fallback
- [ ] 6.3 Implement timeout and cancellation, including child process cleanup
- [ ] 6.4 Implement the polling wait step
- [ ] 6.5 Test every scenario in the specification
- [ ] 6.6 Manual proof: run a workflow made only of shell steps with a loop, a timeout, and a failure route

## 7. Harness contract and generic agent plugin (harness-protocol, plugin-harness-acp)

- [ ] 7.1 Implement the session lifecycle, the headless requirement, the interactive capability declaration, and validation of declared modes
- [ ] 7.2 Implement the injected tool server with the complete, ask, and artifact tools, including the summary output, undeclared outcome error, and holding an unconfirmed outcome for retry
- [ ] 7.3 Implement prompt construction with prefixes and outcome instructions
- [ ] 7.4 Implement the turn-end policies and the re-prompt limit
- [ ] 7.5 Implement session continuity, fresh sessions, and mixed harnesses in one run
- [ ] 7.6 Implement the activity status and session failure handling
- [ ] 7.7 Implement the generic plugin that drives an Agent Client Protocol agent, mapping permission requests and questions to attention
- [ ] 7.8 Check each target harness against the generic plugin, record the adapter command or native mode that works for each, and list the gaps
- [ ] 7.9 Add dedicated plugins only for gaps that block the reference workflow
- [ ] 7.10 Test every scenario with a scripted fake agent, and test the plugin against at least two real harnesses
- [ ] 7.11 Manual proof: a workflow with two sessions on two different harnesses, answering a permission request and a question from a client, with recordings

## 8. GitHub plugin (plugin-github)

- [ ] 8.1 Implement identity reporting and the permission denied behavior
- [ ] 8.2 Implement the issue and pull request triggers with dedupe, by polling
- [ ] 8.3 Implement webhook delivery through the engine
- [ ] 8.4 Implement open pull request, mark ready, and merge
- [ ] 8.5 Implement wait for review, check status, pull request state, and review comments resolved
- [ ] 8.6 Implement the stacked pull request steps using the stack extension, with the unavailable outcome
- [ ] 8.7 Implement the pull request artifact kind and its probe
- [ ] 8.8 Test every scenario against a recorded fake API and against a real scratch repository
- [ ] 8.9 Manual proof: in a scratch repository, a workflow that opens a draft pull request, waits for a real review, handles changes requested and approval, and links a stack

## 9. OpenSpec plugin (plugin-openspec)

- [ ] 9.1 Implement validate, task progress, and proposal presence checks
- [ ] 9.2 Implement the invariant checks
- [ ] 9.3 Implement archive
- [ ] 9.4 Implement the change artifact kind and the skill prefix values
- [ ] 9.5 Test every scenario against fixture repositories
- [ ] 9.6 Manual proof: run the plugin steps against a scratch repository with a valid change, an invalid change, and unchecked tasks

## 10. Terminal client (client-tui)

- [ ] 10.1 Implement the overview of runs with the current step, the needs-you indicator, and artifact links
- [ ] 10.2 Implement the run view with the live graph
- [ ] 10.3 Implement answering attention requests inline
- [ ] 10.4 Implement the run controls, attach to interactive sessions when offered, and reconnect
- [ ] 10.5 Test with a scripted engine and screenshot comparisons of the main views
- [ ] 10.6 Manual proof: use the client against several concurrent runs and record the three questions being answered at a glance

## 11. Web client (client-web)

- [ ] 11.1 Serve the client from the engine and implement run cards
- [ ] 11.2 Implement the live graph with the current step highlighted
- [ ] 11.3 Implement the attention inbox and the title count
- [ ] 11.4 Implement artifact links, the in-page terminal for interactive sessions, and reconnect
- [ ] 11.5 Meet the accessibility requirements and test them with an automated audit and keyboard use
- [ ] 11.6 Test with a scripted engine and browser tests
- [ ] 11.7 Manual proof: use the client in a browser against several concurrent runs and record the three questions being answered at a glance

## 12. Go SDK (workflow-sdk-go)

- [ ] 12.1 Implement the builder that emits the IR
- [ ] 12.2 Implement fragments and reuse
- [ ] 12.3 Implement the export command that turns YAML into SDK code
- [ ] 12.4 Test round-trip parity between YAML and the SDK for every fixture workflow
- [ ] 12.5 Manual proof: export the reference workflow to code, build it, and show the identical IR

## 13. Reference workflow end to end (openspec-workflow)

- [ ] 13.1 Add the reference workflow and its sub-workflow to the repository, based on the design appendix, and make them pass validation
- [ ] 13.2 Resolve the open questions in the design that affect the workflow, and update the design with the answers
- [ ] 13.3 Test each phase against fakes: exploration feedback, proposal checks red, review changes requested, invariant failure, loops, scope check, and permission denied
- [ ] 13.4 Run the workflow end to end on a real repository with real harnesses, from a prompt and from an issue
- [ ] 13.5 Capture recordings or screenshots from the terminal client and the web client at several points
- [ ] 13.6 Attach the recordings to the pull request that completes this group
- [ ] 13.7 Manual proof: the recordings from 13.5 are the proof
