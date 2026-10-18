# Spec Delta

## Purpose

Defines the protocol between the engine and the place that keeps run records, so that the default store is simple, a durable or shared store can be swapped in by anyone in any language, and the engine behaves the same with one instance or many.

## ADDED Requirements

### Requirement: Store protocol
The engine SHALL reach the store only through a versioned protocol defined in JSON Schema. The built-in stores SHALL implement the same messages as third-party stores.

#### Scenario: Built-in store uses the protocol
- **WHEN** the engine uses a built-in store
- **THEN** it exchanges the same messages as it would with a third-party store

#### Scenario: Incompatible store version
- **WHEN** a store answers with a protocol version the engine does not support
- **THEN** the engine refuses to use the store and reports the mismatch

### Requirement: Conditional append
The store SHALL append events to a run only when the run's last sequence number equals the one the caller expects, and SHALL otherwise answer with a conflict and the current last sequence number. Each append SHALL be atomic.

#### Scenario: Append succeeds
- **WHEN** the engine appends events expecting the run's current last sequence number
- **THEN** the store records all of the events in order and answers with the new last sequence number

#### Scenario: Competing appends
- **WHEN** two callers append to the same run expecting the same last sequence number
- **THEN** exactly one append is recorded and the other receives a conflict

### Requirement: Reading a run
The store SHALL return the events of a run in recorded order from a given sequence number, and SHALL never reuse a sequence number within a run.

#### Scenario: Read from a sequence number
- **WHEN** the engine reads a run from a sequence number
- **THEN** it receives every event after that number in order

### Requirement: Deduplication claims
The store SHALL provide an atomic claim of a dedupe key within a namespace that answers whether the key is new or already claimed.

#### Scenario: First claim
- **WHEN** the engine claims a dedupe key that no one has claimed
- **THEN** the store answers that the key is new

#### Scenario: Competing claims
- **WHEN** two callers claim the same dedupe key at once
- **THEN** exactly one is told the key is new

### Requirement: Listing runs
The store SHALL list the run summaries of a namespace, filtered by workflow and state.

#### Scenario: Filtered list
- **WHEN** the engine lists the runs of a namespace in the state waiting
- **THEN** the store returns only the matching runs of that namespace

### Requirement: Optional watching
A store MAY support watching a namespace from a cursor and delivering new records as they are written. When a store does not, the engine SHALL poll it.

#### Scenario: Store can watch
- **WHEN** a store declares that it can watch
- **THEN** the engine receives new records as they are written without polling

#### Scenario: Store cannot watch
- **WHEN** a store does not declare that it can watch
- **THEN** the engine polls it and clients still receive events in order

### Requirement: Optional leasing
A store MAY support granting a run to one holder for a limited time. A store that declares it is safe for many instances SHALL support leasing.

#### Scenario: Lease held
- **WHEN** an instance asks for a lease on a run that another instance holds
- **THEN** the store refuses and says who holds it and until when

#### Scenario: Lease expires
- **WHEN** a holder stops renewing its lease and the time passes
- **THEN** another instance can be granted the lease

### Requirement: Declared capabilities
Every store SHALL declare whether it is durable, whether it is safe for many instances, and whether it can watch. The engine SHALL refuse a setup that contradicts a declaration.

#### Scenario: Second instance on an unsafe store
- **WHEN** a second engine instance is configured to use a store that is not safe for many instances
- **THEN** the instance refuses to start and explains why

#### Scenario: Serve on a store that is not durable
- **WHEN** an operator starts the server with a store that is not durable and has not allowed it explicitly
- **THEN** the engine refuses to start and explains that runs would not survive a restart

### Requirement: Built-in stores
The engine SHALL include a memory store that keeps nothing after the process ends, and a local durable store that keeps runs across restarts on one machine. The server SHALL use the local durable store unless configured otherwise, and a one-off run command without a server SHALL use the memory store.

#### Scenario: Server default
- **WHEN** an operator starts the server with no store configured
- **THEN** it uses the local durable store

#### Scenario: One-off run
- **WHEN** an operator runs a workflow with no server and no store configured
- **THEN** the run uses the memory store

### Requirement: Third-party stores
A store written by anyone SHALL run as a separate process that the engine launches and talks to using JSON-RPC 2.0 over standard input and output, and SHALL be resolved from a local path or a git reference in the same way as a plugin.

#### Scenario: Any language
- **WHEN** an executable written in any language implements the store protocol
- **THEN** the engine can use it as its store

#### Scenario: Adding a store
- **WHEN** an operator configures a third-party store for the first time
- **THEN** the engine shows the command it will launch and requires confirmation

### Requirement: Store selection
The store SHALL be chosen in the engine's configuration and SHALL NOT be chosen by a workflow. The engine SHALL resolve and connect to the store before it loads workflows or plugins.

#### Scenario: Workflow names a store
- **WHEN** a workflow tries to select a store
- **THEN** validation rejects it

### Requirement: Store failure
When a store adapter process exits unexpectedly, the engine SHALL restart it up to a configured limit. While the store is unreachable the engine SHALL NOT confirm any signal and SHALL report itself unhealthy. The engine SHALL NOT move any run to needs-human because of a store failure.

#### Scenario: Adapter crashes once
- **WHEN** a store adapter process crashes once
- **THEN** the engine restarts it and unconfirmed signals are retried by their senders

#### Scenario: Store unreachable
- **WHEN** the store cannot be reached
- **THEN** the engine confirms no signals, reports itself unhealthy, and continues the runs on its own when the store returns

### Requirement: Conformance suite
The platform SHALL provide a conformance suite that runs against any store, built in or third party, through the store protocol alone. A store SHALL be considered conformant only when it passes the checks for every capability it declares.

#### Scenario: Passing store
- **WHEN** the suite runs against a store that follows the protocol
- **THEN** every check for its declared capabilities passes

#### Scenario: Broken append
- **WHEN** the suite runs against a store whose conditional append accepts two appends at the same expected sequence number
- **THEN** the suite fails and names the check
