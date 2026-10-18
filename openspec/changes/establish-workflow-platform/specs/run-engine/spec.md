# Spec Delta

## Purpose

Defines how the engine creates and advances runs of a workflow, and how it tracks the current step, attention requests, and artifacts in a durable event log. The engine alone owns this state.

## ADDED Requirements

### Requirement: Engine owns step tracking
The engine SHALL be the only component that records which step a run is on. Plugins SHALL NOT be required to track, report, or persist the current step.

#### Scenario: Plugin is unaware of position
- **WHEN** a plugin step is started
- **THEN** the plugin receives only that step's inputs and context, and the engine records the step as active

### Requirement: Run creation from triggers
The engine SHALL create a run when a trigger fires or when a client starts one. The run SHALL record the workflow, the trigger identity, and the trigger payload given by the plugin.

#### Scenario: Trigger starts a run
- **WHEN** a trigger plugin reports that its trigger fired with a payload
- **THEN** the engine creates a run of the workflow bound to that trigger and stores the payload

#### Scenario: Client starts a run
- **WHEN** a client starts a workflow with inputs
- **THEN** the engine creates a run that has the inputs and no trigger payload

#### Scenario: Duplicate trigger event
- **WHEN** a trigger reports an event whose dedupe key matches a run already created
- **THEN** the engine does not create another run

### Requirement: Run states
A run SHALL always be in exactly one state: running, waiting, needs-attention, succeeded, failed, or cancelled. Succeeded, failed, and cancelled SHALL be final.

#### Scenario: Waiting on an external event
- **WHEN** the active step is a wait that is pending
- **THEN** the run state is waiting

#### Scenario: Waiting on the operator
- **WHEN** an attention request is open for the run
- **THEN** the run state is needs-attention

### Requirement: Step dispatch and resolution
The engine SHALL start a step by calling its plugin and SHALL treat the step as resolved only when the plugin or the agent session reports an outcome. The engine SHALL then follow the transition for that outcome.

#### Scenario: Normal resolution
- **WHEN** a started step reports the outcome done and the workflow routes done to a next step
- **THEN** the engine marks the step resolved and starts the next step

#### Scenario: Outcome not declared by the step
- **WHEN** a step reports an outcome its plugin did not declare
- **THEN** the engine treats it as a protocol error and moves the run to needs-human

### Requirement: Step timeouts
A step MAY declare a timeout. When it elapses before the step resolves, the engine SHALL cancel the step and resolve it with the outcome timed_out.

#### Scenario: Timeout routes
- **WHEN** a step with a timeout does not resolve in time and the workflow routes timed_out
- **THEN** the engine cancels the step and follows that route

#### Scenario: Timeout not routed
- **WHEN** timed_out has no route
- **THEN** the run moves to needs-human

### Requirement: Loop accounting
The engine SHALL count how many times each step has started in a run and SHALL show the count to clients.

#### Scenario: Count visible
- **WHEN** a step is entered for the third time
- **THEN** the run state shows an iteration count of three for that step

### Requirement: Attention requests
The engine SHALL maintain attention requests for each run. A request SHALL have an id, the run, the step, a kind, a prompt, optional choices, a source, and a state of open, answered, or expired. Kinds SHALL include question, approval, plan, invariant, error, and unrouted-outcome.

#### Scenario: Agent asks a question
- **WHEN** an agent session raises a question
- **THEN** the engine opens an attention request of kind question with the prompt and choices, and the run becomes needs-attention

#### Scenario: Answering a request
- **WHEN** a client answers an open request
- **THEN** the engine marks it answered, delivers the answer to its source, and the run leaves needs-attention if no other request is open

#### Scenario: Answer to a closed request
- **WHEN** a client answers a request that is already answered or expired
- **THEN** the engine rejects the answer with the request's current state

### Requirement: Human gates
A step MAY be a human gate whose outcomes are chosen by an operator. A gate SHALL open an attention request and resolve only from its answer.

#### Scenario: Gate resolves from an answer
- **WHEN** an operator answers a gate request with an outcome the step declares
- **THEN** the step resolves with that outcome

#### Scenario: Answer text is an output
- **WHEN** an operator answers a gate request with an outcome and free text
- **THEN** the step's outputs contain the text under the name text

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
The engine SHALL record every state change of a run as an append-only event before acting on it, including run start, step entry, step resolution, attention requested and answered, artifact changes, and run end. Current state SHALL be derivable from the log alone.

#### Scenario: Rebuild from the log
- **WHEN** the engine restarts
- **THEN** it rebuilds the state of every run from its events and the state matches the state before the restart

#### Scenario: Events are ordered
- **WHEN** a client reads a run's events
- **THEN** they are returned in the order recorded with a sequence number that never repeats

### Requirement: Resume after restart
After a restart the engine SHALL resume every non-final run. Steps that were active SHALL be restarted or re-attached according to what their plugin declares it supports.

#### Scenario: Resumable wait
- **WHEN** the engine restarts while a run waits for a review and the plugin supports re-attachment
- **THEN** the wait continues without starting a new step

#### Scenario: Non-resumable step
- **WHEN** the engine restarts during a step whose plugin does not support re-attachment
- **THEN** the engine restarts that step and records that it was restarted

### Requirement: Cancel and retry
A client SHALL be able to cancel a run and to retry a failed or needs-attention run. Cancelling SHALL cancel the active step and end any agent sessions. Retrying SHALL re-run the step that routed to failure or needs-human.

#### Scenario: Cancel
- **WHEN** a client cancels a running run
- **THEN** the active step is cancelled, sessions are closed, and the run ends in cancelled

#### Scenario: Retry
- **WHEN** a client retries a run that is in needs-attention
- **THEN** the routing step runs again and the iteration count increases

### Requirement: Concurrent runs
The engine SHALL run many runs at the same time and SHALL keep their state isolated from each other.

#### Scenario: Independent runs
- **WHEN** two runs of the same workflow are active
- **THEN** a step resolving in one run has no effect on the other

### Requirement: Run context
The engine SHALL keep a run context containing the trigger payload, workflow inputs, step outputs, and artifacts, and SHALL resolve references from it when each step starts.

#### Scenario: Output available downstream
- **WHEN** a step resolves with outputs
- **THEN** a later step can reference those outputs
