# Spec Delta

## Purpose

Defines how the engine creates and advances runs of a workflow, and how it tracks the current step, wait points, attention requests, and artifacts in a durable event log. The engine alone owns this state, and a run that is waiting costs no running process.

## ADDED Requirements

### Requirement: Engine owns step tracking
The engine SHALL be the only component that records which step a run is on. Plugins SHALL NOT be required to track, report, or persist the current step.

#### Scenario: Plugin is unaware of position
- **WHEN** a plugin step is started
- **THEN** the plugin receives only that step's inputs and context, and the engine records the step as active

### Requirement: Run creation from triggers
The engine SHALL create a run when a start trigger fires or when a client starts one. The run SHALL record the workflow, the trigger identity, and the trigger payload given by the plugin.

#### Scenario: Trigger starts a run
- **WHEN** a trigger plugin reports that its start trigger fired with a payload
- **THEN** the engine creates a run of the workflow bound to that trigger and stores the payload

#### Scenario: Client starts a run
- **WHEN** a client starts a workflow with inputs
- **THEN** the engine creates a run that has the inputs and no trigger payload

#### Scenario: Duplicate trigger event
- **WHEN** a trigger reports an event whose dedupe key matches a run already created
- **THEN** the engine does not create another run

#### Scenario: Duplicate across instances
- **WHEN** two engine instances sharing a store receive the same trigger event at once
- **THEN** the store's claim of the dedupe key lets exactly one of them create the run

### Requirement: Runs pin their definition
A run SHALL record the digest of the workflow definition it started with, and the engine SHALL keep that definition available for as long as the run is not final. Replacing a workflow in the catalog SHALL affect only runs started afterward.

#### Scenario: Workflow replaced mid-run
- **WHEN** a workflow is replaced in the catalog while a run of the old definition is waiting
- **THEN** the waiting run resumes and finishes on the old definition

#### Scenario: Definition kept in the store
- **WHEN** the engine restarts or another instance takes over a run
- **THEN** it reads the run's definition from the store by digest and does not need the original file

### Requirement: Workflow catalog
The engine SHALL keep a catalog of the workflows it can start, per namespace, each identified by name and definition digest. An operator or client SHALL be able to add, replace, list, and remove workflows at run time, and the engine SHALL subscribe the start triggers of each cataloged workflow. Removing a workflow SHALL unsubscribe its start triggers and SHALL NOT affect runs that already started.

#### Scenario: Add at run time
- **WHEN** a client adds a valid workflow to a running engine
- **THEN** the workflow can be started and its start triggers are subscribed without a restart

#### Scenario: Invalid workflow refused
- **WHEN** a client adds a workflow that fails validation
- **THEN** the engine refuses it, reports every problem, and leaves the catalog unchanged

#### Scenario: Remove keeps running runs
- **WHEN** a workflow is removed from the catalog while one of its runs is waiting
- **THEN** its start triggers are unsubscribed and the waiting run still resumes

#### Scenario: Catalog survives restart
- **WHEN** the engine restarts
- **THEN** the same workflows are cataloged and their triggers are subscribed again

### Requirement: Run states
A run SHALL always be in exactly one state: running, waiting, needs-attention, succeeded, failed, or cancelled. Succeeded, failed, and cancelled SHALL be final. A run is waiting when it is at a wait step that has not resolved, and nothing runs on its behalf in that state.

#### Scenario: Waiting on an external event
- **WHEN** the active step is a wait step that is pending
- **THEN** the run state is waiting

#### Scenario: Waiting on the operator
- **WHEN** an attention request is open for the run
- **THEN** the run state is needs-attention

### Requirement: Step dispatch and resolution
The engine SHALL start a step by calling its plugin and SHALL treat the step as resolved only when the plugin or the agent session reports an outcome. The engine SHALL then follow the transition for that outcome. A step SHALL exit with an outcome and SHALL NOT remain active to wait for a person or an outside event.

#### Scenario: Normal resolution
- **WHEN** a started step reports the outcome done and the workflow routes done to a next step
- **THEN** the engine marks the step resolved and starts the next step

#### Scenario: Outcome not declared by the step
- **WHEN** a step reports an outcome its plugin did not declare
- **THEN** the engine treats it as a protocol error and moves the run to needs-human

#### Scenario: Questions are an exit
- **WHEN** an agent step exits with the outcome questions
- **THEN** the step is resolved, its outputs are recorded, and the run follows the route for questions

### Requirement: Wait points
When a run enters a wait step the engine SHALL resolve the step's trigger configurations from the run context, ask each trigger's plugin for its correlation key, and record in the store a subscription for each key, all before the run is considered waiting. The engine SHALL remove the subscriptions when the step resolves.

#### Scenario: Run rests
- **WHEN** a run enters a wait step on a review of pull request 42
- **THEN** the store holds a subscription for the review key of pull request 42 and no process is running for the run

#### Scenario: Event already happened
- **WHEN** a reviewer approved before the run reached the wait step and the trigger offers check-now
- **THEN** the engine resolves the wait step immediately with the approved outcome

#### Scenario: Subscriptions removed
- **WHEN** a wait step resolves from one of its triggers
- **THEN** the subscriptions for its other triggers are removed

#### Scenario: Subscriptions survive restart
- **WHEN** the engine restarts while a run is waiting
- **THEN** the subscription records are still in the store and the engine asks the plugins to subscribe again

### Requirement: Check-now on arrival at a wait
If any trigger listed by a wait step offers a check-now call, the engine SHALL call it after recording the subscriptions and, when the condition already holds, SHALL resolve the step from the answer.

#### Scenario: Condition already holds
- **WHEN** a run enters a wait step whose condition was met before the run arrived
- **THEN** the engine resolves the step from the check-now answer without waiting for an event

### Requirement: Resuming runs from events
When a trigger plugin reports an event, the engine SHALL look up the runs waiting on the event's correlation keys, and SHALL resolve each matching wait step with the outcome and outputs the plugin classified the event into, using the event's dedupe key as the signal key. The same event MAY also start new runs through start triggers. The first event recorded for a wait step SHALL win, and the engine SHALL ignore later events for a step that has resolved.

#### Scenario: Review resumes the run
- **WHEN** a plugin reports a review event for pull request 42 and a run waits on that key
- **THEN** the run's wait step resolves with the classified outcome and the run continues

#### Scenario: No waiting run
- **WHEN** an event matches no waiting run and no start trigger
- **THEN** the engine records nothing for it and confirms it

#### Scenario: Two events race
- **WHEN** an approval and a timeout reach a wait step at the same time
- **THEN** exactly one is recorded and the run follows its route only

#### Scenario: Event for a resolved wait
- **WHEN** a later event arrives for a wait step that already resolved
- **THEN** the engine confirms it and the run is unaffected

#### Scenario: One event starts and resumes
- **WHEN** one event matches a start trigger of a workflow and a wait step of a different run
- **THEN** a new run starts and the waiting run resumes

### Requirement: Step timeouts
A step MAY declare a timeout. The engine SHALL record the deadline as an event when the step starts. When the deadline passes before the step resolves, the engine SHALL cancel the step and resolve it with the outcome timed_out, exactly once. A wait step's timeout SHALL be recorded as a deadline on the wait in the same way and SHALL be delivered even when no process was running for the run.

#### Scenario: Deadline survives a restart
- **WHEN** the engine restarts while a step with a timeout is active and the deadline passes
- **THEN** the step is resolved with timed_out once, and not restarted with a fresh timeout

#### Scenario: Timeout routes
- **WHEN** a step with a timeout does not resolve in time and the workflow routes timed_out
- **THEN** the engine cancels the step and follows that route

#### Scenario: Timeout not routed
- **WHEN** timed_out has no route
- **THEN** the run moves to needs-human

#### Scenario: Wait deadline while the engine was down
- **WHEN** a wait step's deadline passes while no engine is running and an engine later starts
- **THEN** the engine resolves the wait with timed_out once

### Requirement: Loop accounting
The engine SHALL count how many times each step has started in a run and SHALL show the count to clients. When a transition lists steps to reset, the engine SHALL set their counts to zero when the transition is taken and SHALL record the reset in the log.

#### Scenario: Count visible
- **WHEN** a step is entered for the third time
- **THEN** the run state shows an iteration count of three for that step

#### Scenario: Reset on restart
- **WHEN** a transition that lists a step to reset is taken
- **THEN** the step's count is zero and its next start counts as the first

#### Scenario: Bound exceeded
- **WHEN** a step with a maximum of three would start a fourth time
- **THEN** the run moves to needs-human instead and the request shows the count

### Requirement: Questions exit and the ask route
When a step exits with the outcome questions and the workflow does not route it, the engine SHALL follow the core ask route. The ask route SHALL open an attention request holding the questions and the outputs the step recorded, deliver it to the workflow's attention channels and to connected clients, and leave the run in needs-attention with nothing running.

#### Scenario: Questions with draft work
- **WHEN** an exploration step exits with two questions and a draft plan
- **THEN** the request shows the questions with the draft plan, and the run is needs-attention with no process running

#### Scenario: Answers re-enter the step
- **WHEN** an operator answers both questions
- **THEN** the exploration step starts again in a fresh session and receives the answers and the previous draft plan

#### Scenario: Partial answers
- **WHEN** an operator answers only one of two questions
- **THEN** the request stays open until the others are answered or the operator marks them skipped

#### Scenario: Re-entry limit
- **WHEN** a step has already been re-entered as many times as its maximum
- **THEN** the run moves to needs-human with the count in the request

#### Scenario: Questions routed elsewhere
- **WHEN** a workflow routes questions on a step to a different step
- **THEN** the engine follows that route and does not open the ask request

### Requirement: Re-entry after the answers
When an ask request is answered, the engine SHALL start the asking step again as a fresh execution, passing the answers and the previous attempt's outputs, notes, and summary as route values. The engine SHALL count the re-entry against the step's maximum.

#### Scenario: Re-entry counted
- **WHEN** a step that allows three iterations exits with questions for the third time and the request is answered
- **THEN** the run moves to needs-human instead of starting the step again

### Requirement: Attention requests
The engine SHALL maintain attention requests for each run. A request SHALL have an id, the run, the step, a kind, a prompt, optional choices, optional context such as draft outputs, a source, and a state of open, answered, or expired. Kinds SHALL include questions, approval, plan, invariant, error, and unrouted-outcome.

#### Scenario: Questions opened
- **WHEN** a step exits with questions
- **THEN** the engine opens an attention request of kind questions with each question and the draft outputs as context, and the run becomes needs-attention

#### Scenario: Answering a request
- **WHEN** a client answers an open request
- **THEN** the engine marks it answered, resumes the run by the rule for the request's kind, and the run leaves needs-attention if no other request is open

#### Scenario: Answer to a closed request
- **WHEN** a client answers a request that is already answered or expired
- **THEN** the engine rejects the answer with the request's current state

#### Scenario: Answer through a channel
- **WHEN** an attention channel reports an answer to a request
- **THEN** the engine treats it the same as an answer from a client

### Requirement: Operator answer as a trigger
The engine SHALL provide a built-in trigger that an operator answers, for use in wait steps. A wait step that lists it SHALL open an attention request with the step's prompt and choices and resolve only from its answer. The request SHALL be open whether or not any client is connected, and the answer SHALL be recorded as a signal.

#### Scenario: Gate resolves from an answer
- **WHEN** an operator answers a gate request with an outcome the step declares
- **THEN** the step resolves with that outcome

#### Scenario: Answer text is an output
- **WHEN** an operator answers a gate request with an outcome and free text
- **THEN** the step's outputs contain the text under the name text

#### Scenario: Gate answered later
- **WHEN** a gate request is open and no client or engine process is running for days
- **THEN** an answer recorded afterward resolves the step and the run continues once an engine runs it

### Requirement: Artifact registry
The engine SHALL keep a registry of the artifacts of each run. Each entry SHALL have a kind, a name, the producing step, a state of expected or available, and when available a link or reference supplied by a plugin or session. The engine SHALL publish changes to clients.

#### Scenario: Artifact becomes available
- **WHEN** a plugin or agent reports an artifact that matches an expected entry
- **THEN** the entry becomes available with its link, and clients are notified

#### Scenario: Unexpected artifact
- **WHEN** an artifact is reported that no step declared
- **THEN** the engine adds it as available and records which step was active

#### Scenario: Deterministic detection
- **WHEN** a plugin declares an artifact kind that can be probed
- **THEN** the engine asks the plugin to check for it when the producing step resolves and updates the entry from the answer

### Requirement: Durable event log
The engine SHALL record every state change of a run in the store as an append-only event before acting on it, using the store's conditional append, including run start, step entry, step resolution with its outputs, wait entry and resolution, attention requested and answered, artifact changes, and run end. Current state SHALL be derivable from the log alone, and the outputs of every attempt of every step SHALL remain in the log.

#### Scenario: Rebuild from the log
- **WHEN** the engine restarts with a durable store
- **THEN** it rebuilds the state of every run from its events and the state matches the state before the restart

#### Scenario: Events are ordered
- **WHEN** a client reads a run's events
- **THEN** they are returned in the order recorded with a sequence number that never repeats

#### Scenario: Append conflict
- **WHEN** the store reports a conflict because another instance advanced the run first
- **THEN** the engine reloads the run from the log and does not act on its stale view

#### Scenario: Earlier attempts remain
- **WHEN** a step has run twice because the run looped back
- **THEN** the log holds the outputs of both and the run context shows the second

### Requirement: Resume on taking ownership
When an engine instance starts or takes over a run, it SHALL resume every non-final run it owns. Steps that were active SHALL be restarted or re-attached according to what their plugin declares it supports. For a waiting run the engine SHALL re-subscribe the triggers of its wait step and deliver any deadline that has passed. A restart and a takeover by another instance SHALL follow the same rules.

#### Scenario: Waiting run after restart
- **WHEN** the engine restarts while a run waits on a review
- **THEN** the engine subscribes the review trigger again, checks the condition if check-now is offered, and the run stays waiting without starting a step

#### Scenario: Non-resumable step
- **WHEN** the engine restarts during a step whose plugin does not support re-attachment
- **THEN** the engine restarts that step and records that it was restarted

#### Scenario: Takeover by another instance
- **WHEN** an instance that owned a run is gone and another instance sharing the store takes the run
- **THEN** the new instance resumes it by the same rules as a restart

### Requirement: Signals are confirmed after recording
The engine SHALL confirm a signal to its sender only after the signal is recorded. Signals include a step outcome, an attention answer, a trigger event, an artifact report, and a client control. Each signal SHALL carry a unique key, and a repeated key SHALL have no further effect.

#### Scenario: Store unreachable
- **WHEN** a plugin reports a step outcome and the store cannot be reached
- **THEN** the engine tells the sender the signal was not recorded and does not start the next step

#### Scenario: Retry after recovery
- **WHEN** the sender retries the same signal after the store returns
- **THEN** the engine records it once and confirms it

#### Scenario: Duplicate key
- **WHEN** a sender repeats a signal whose key is already recorded
- **THEN** the engine confirms it again and records nothing new

### Requirement: Signals can be recorded without a running engine
A signal that resumes a run, such as an operator answer, a delivered trigger event, or the completion of an externally hosted agent, SHALL be recordable directly in the store by a short-lived process. The next engine instance to own the run SHALL act on it.

#### Scenario: Answer with no engine
- **WHEN** an operator records an answer while no engine is running
- **THEN** the answer is in the store and a later engine continues the run from it

#### Scenario: Event delivered by a one-shot process
- **WHEN** a short-lived process delivers a trigger event and the matching run is waiting
- **THEN** the event is recorded, the process advances the run until it waits or ends, and then exits

### Requirement: Steps run at least once
The engine SHALL guarantee that a step whose resolution was never recorded is eventually resolved, and SHALL NOT guarantee that it runs only once. A step restarted for this reason SHALL be recorded as restarted.

#### Scenario: Sender dies holding a signal
- **WHEN** a plugin dies after completing work and before its outcome was confirmed
- **THEN** the step is still active in the log and is restarted or re-attached by the resume rules

### Requirement: Namespaces
Every workflow, run, event, attention request, subscription, and artifact SHALL belong to a namespace. A namespace called default SHALL be used when none is given, and runs in different namespaces SHALL NOT see each other's state.

#### Scenario: Default namespace
- **WHEN** a run is started with no namespace
- **THEN** it belongs to the namespace default

#### Scenario: Isolation
- **WHEN** a client lists runs in one namespace
- **THEN** no run from another namespace is returned

### Requirement: Run directory
The engine SHALL give each run a directory that exists for the life of the run, name it in the run metadata, and offer it to steps as a place for working files. The engine SHALL NOT remove the directory until the run is final and its cleanup has finished.

#### Scenario: Directory available
- **WHEN** a run starts
- **THEN** the run metadata names its directory and it exists

#### Scenario: Removed after cleanup
- **WHEN** a run reaches a final state and its cleanup sub-workflow finishes
- **THEN** the engine removes the run directory unless the operator asked to keep it

### Requirement: Steps are dispatched to a runner
The engine SHALL hand each step and agent session to a runner using the plugin protocol messages, and a step that uses a session SHALL return to the runner that hosts it. In this change the only runner SHALL be in the same process as the engine, and steps that share files on one disk SHALL run on the same runner.

#### Scenario: In-process runner
- **WHEN** the engine starts a plugin step
- **THEN** the runner in the same process launches the plugin and exchanges the protocol messages

#### Scenario: Runner is gone
- **WHEN** the runner hosting a step is no longer available
- **THEN** the engine applies the re-attach or restart rules for that step

### Requirement: Engine health
The engine SHALL track its own health, including whether the store is reachable and since when, and SHALL expose it to clients.

#### Scenario: Store becomes unreachable
- **WHEN** the engine cannot reach the store
- **THEN** its health reports the store unreachable and the time it was first noticed

### Requirement: Cancel and retry
A client SHALL be able to cancel a run and to retry a failed or needs-attention run. Cancelling SHALL cancel the active step, remove the run's subscriptions, and end any agent sessions. Retrying SHALL re-run the step that routed to failure or needs-human.

#### Scenario: Cancel
- **WHEN** a client cancels a running run
- **THEN** the active step is cancelled, sessions are closed, and the run ends in cancelled

#### Scenario: Cancel while waiting
- **WHEN** a client cancels a waiting run
- **THEN** the subscriptions are removed and the run ends in cancelled

#### Scenario: Retry
- **WHEN** a client retries a run that is in needs-attention
- **THEN** the routing step runs again and the iteration count increases

### Requirement: Concurrent runs
The engine SHALL run many runs at the same time and SHALL keep their state isolated from each other. A single run SHALL be at one step at a time, and running steps of one run in parallel SHALL be a non-goal of this change.

#### Scenario: Independent runs
- **WHEN** two runs of the same workflow are active
- **THEN** a step resolving in one run has no effect on the other

### Requirement: Run context
The engine SHALL keep a run context containing the trigger payload, workflow inputs, step outputs, route values, and artifacts, and SHALL resolve references from it when each step starts. A reference to a step output SHALL resolve to the most recent resolution of that step.

#### Scenario: Output available downstream
- **WHEN** a step resolves with outputs
- **THEN** a later step can reference those outputs

#### Scenario: Most recent output wins
- **WHEN** a step has resolved twice and a later step references its output
- **THEN** the value is from the second resolution
