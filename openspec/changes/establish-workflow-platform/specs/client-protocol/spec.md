# Spec Delta

## Purpose

Defines how any client, built in or written by a third party, observes and controls runs. A client needs only this protocol to show which step each run is on, what needs the operator, and what has been produced.

## ADDED Requirements

### Requirement: Transports
The engine SHALL accept client connections as JSON-RPC 2.0 over WebSocket and as JSON-RPC 2.0 over standard input and output for clients that embed the engine.

#### Scenario: WebSocket client
- **WHEN** a client connects over WebSocket
- **THEN** it can call every client method and receive events

#### Scenario: Embedded client
- **WHEN** a client launches the engine as a child process in embedded mode
- **THEN** it uses the same methods over standard input and output

### Requirement: Local-only by default
The engine SHALL listen only on the loopback interface by default. When configured to listen elsewhere it SHALL require an authentication token on every connection.

#### Scenario: Default bind
- **WHEN** the engine starts with no network configuration
- **THEN** it accepts connections only from the local machine

#### Scenario: Remote bind without token
- **WHEN** the engine is configured to listen on a non-loopback address with no token configured
- **THEN** it refuses to start and explains why

#### Scenario: Wrong token
- **WHEN** a client presents an invalid token on a non-loopback bind
- **THEN** the connection is rejected

### Requirement: Namespaces
Every client method and subscription SHALL take a namespace and SHALL use the namespace default when none is given.

#### Scenario: No namespace given
- **WHEN** a client lists runs without naming a namespace
- **THEN** it receives the runs of the namespace default

### Requirement: Workflow discovery
A client SHALL be able to list workflows and fetch a workflow's graph, including steps, transitions, sessions, and expected artifacts.

#### Scenario: Graph fetch
- **WHEN** a client requests a workflow
- **THEN** it receives the full graph

### Requirement: Run listing and detail
A client SHALL be able to list runs, filtered by workflow and state, and to fetch a run. A run summary SHALL include the current step, the run state, the open attention requests, and the artifacts, so a client can answer the three at-a-glance questions without reading anything else.

#### Scenario: Summary answers the three questions
- **WHEN** a client lists runs
- **THEN** each entry contains the current step, whether the run needs the operator, and its available artifacts

### Requirement: Event subscription
A client SHALL be able to subscribe to the events of one run or of all runs and to resume from a sequence number. The engine SHALL send events in order and SHALL send a snapshot first when the client asks for one.

#### Scenario: Live updates
- **WHEN** a run's step changes
- **THEN** every subscribed client receives the event

#### Scenario: Resume after disconnect
- **WHEN** a client reconnects and asks to resume from a sequence number
- **THEN** it receives every event after that number

#### Scenario: Run advanced elsewhere
- **WHEN** a run is advanced by a different engine instance than the one the client is connected to
- **THEN** the client still receives the events, because they are served from the recorded log

### Requirement: Starting and controlling runs
A client SHALL be able to start a run of a workflow with inputs, cancel a run, and retry a run.

#### Scenario: Start
- **WHEN** a client starts a workflow with valid inputs
- **THEN** the engine returns the new run id

#### Scenario: Invalid inputs
- **WHEN** a client starts a workflow with inputs that fail the workflow's input schema
- **THEN** the engine rejects the request with the validation errors

### Requirement: Recorded before confirmed
The engine SHALL confirm an answer or a control request from a client only after it is recorded. When it cannot be recorded, the engine SHALL answer that it was not recorded so the client can keep it and retry.

#### Scenario: Answer not recorded
- **WHEN** a client answers a request and the store is unreachable
- **THEN** the engine tells the client the answer was not recorded and the request stays open

### Requirement: Engine health
A client SHALL be able to read and subscribe to the engine's health. When the store is unreachable, reads the engine serves SHALL be marked as possibly out of date.

#### Scenario: Store unreachable
- **WHEN** the engine cannot reach the store
- **THEN** subscribed clients receive an unhealthy event with the time it began, and later reads are marked possibly out of date

### Requirement: Answering attention
A client SHALL be able to answer an attention request with a choice or free text, and an answer from any client SHALL close the request for all clients.

#### Scenario: Two clients
- **WHEN** two clients show the same open request and one answers it
- **THEN** the other client receives an event that the request was answered

#### Scenario: Concurrent answers
- **WHEN** two clients answer the same request at once
- **THEN** exactly one answer is accepted and the other receives the request's answered state

### Requirement: Interactive session attach
For interactive sessions, a client SHALL be able to attach to the session's terminal, receive output, send input, and resize it. Multiple clients MAY attach at once.

#### Scenario: Attach to terminal
- **WHEN** a client attaches to an interactive session
- **THEN** it receives the current screen and subsequent output

#### Scenario: Headless session
- **WHEN** a client attaches to a headless session
- **THEN** the engine returns an error that the session has no terminal

### Requirement: Client identity
Each connected client SHALL identify itself with a name and kind. The engine SHALL record the client in the events for answers and controls it performs.

#### Scenario: Who answered
- **WHEN** an operator answers a request from a client
- **THEN** the answered event records the client's name

### Requirement: Versioning
The protocol SHALL be versioned. The engine SHALL tell the client the version it speaks on connection, and SHALL reject a client that requires a newer version.

#### Scenario: Version mismatch
- **WHEN** a client requires a protocol version the engine does not support
- **THEN** the engine closes the connection with the supported versions
