# Spec Delta

## Purpose

Defines the single command-line program that serves the engine, runs and validates workflows, delivers events and answers, manages the workflow catalog and plugins, and exports workflows, so the platform is usable from a terminal, a script, or a CI job with nothing else installed.

## ADDED Requirements

### Requirement: Single executable
The platform SHALL ship as one executable named workflow that includes the engine, the built-in plugins, and the built-in clients, and needs no separately installed runtime.

#### Scenario: Fresh machine
- **WHEN** the executable is copied to a machine with no other platform files
- **THEN** it can validate and run a workflow that uses only built-in plugins

### Requirement: Serve
The serve command SHALL start the engine as a long-running process that accepts client connections, subscribes the start triggers of the cataloged workflows and the wait triggers of every waiting run, and resumes unfinished runs. It SHALL print the address clients can use and, when enabled, the address of the webhook endpoint.

#### Scenario: Serving
- **WHEN** an operator runs the serve command
- **THEN** the engine starts, resumes unfinished runs, subscribes triggers, and prints its address

#### Scenario: Port in use
- **WHEN** the configured port is already in use
- **THEN** the command exits with an error that names the port

#### Scenario: Separate webhook address
- **WHEN** the webhook endpoint is enabled
- **THEN** it listens on its own configured address and not on the client address

### Requirement: Run to completion
The run command SHALL start a workflow and advance it until the run reaches a final state, rests at a wait step, or needs the operator, and SHALL exit with a status that reflects which. When the run needs the operator, the command SHALL print the request and wait for an answer unless told not to. When the run rests at a wait step, the command SHALL exit with a distinct waiting status unless told to stay and host the polling triggers.

#### Scenario: Succeeded
- **WHEN** a run ends succeeded
- **THEN** the command exits with status zero

#### Scenario: Failed
- **WHEN** a run ends failed or cancelled
- **THEN** the command exits with a non-zero status that distinguishes the two

#### Scenario: Non-interactive
- **WHEN** the command is run with the non-interactive option and a run needs the operator
- **THEN** the command prints the request and exits with a distinct status for needs-attention

#### Scenario: Run rests at a wait
- **WHEN** a run reaches a wait step
- **THEN** the command prints what the run is waiting for and exits with the waiting status, and the run stays in the store

#### Scenario: Stay to poll
- **WHEN** the command is told to stay and the wait uses a polling trigger
- **THEN** the command keeps polling and advances the run when the trigger resolves

### Requirement: Deliver an event
The deliver command SHALL read one raw event from a file or standard input, hand it to the engine for the named plugin, start or resume every matching run, advance them until they wait, need the operator, or end, and exit. It SHALL work against a store with no server running.

#### Scenario: Resume from a delivered event
- **WHEN** a run waits for a review and an operator delivers the review event
- **THEN** the run resumes, advances to its next wait or end, and the command exits

#### Scenario: Event matches nothing
- **WHEN** a delivered event matches no start trigger and no waiting run
- **THEN** the command reports that nothing matched and exits with a status that says so

#### Scenario: Duplicate delivery
- **WHEN** the same event is delivered twice
- **THEN** the second delivery changes nothing

### Requirement: Answer an attention request
The answer command SHALL record an answer to an open attention request, identified by run and request, in the store, with no server running required, and SHALL print the request's resulting state. When asked, it SHALL then advance the run until it waits, needs the operator, or ends.

#### Scenario: Answer with no server
- **WHEN** an operator answers a request while no server runs
- **THEN** the answer is recorded and a later engine continues the run

#### Scenario: Closed request
- **WHEN** the request is already answered
- **THEN** the command prints its state and exits with an error

### Requirement: Validate
The validate command SHALL check one or more workflows and the plugins they use without running them, print every problem with its file and line, and exit non-zero if any exist.

#### Scenario: Valid
- **WHEN** a workflow is valid
- **THEN** the command prints nothing but a success line and exits zero

#### Scenario: Invalid
- **WHEN** a workflow has an unrouted outcome
- **THEN** the command prints the step, the outcome, and the file and line, and exits non-zero

### Requirement: Status
The status command SHALL print for every run its workflow, state, current step, what it is waiting for, whether it needs the operator, and its available artifacts, in a human-readable form by default and as JSON on request.

#### Scenario: Machine-readable
- **WHEN** the command is run with the JSON option
- **THEN** it prints a document containing the same information

#### Scenario: Waiting run
- **WHEN** a run is waiting
- **THEN** the output names the wait step and each trigger it waits on

### Requirement: Workflow catalog commands
The workflow commands SHALL add, replace, list, and remove cataloged workflows in a namespace, validating each before adding it, and SHALL work against the store with or without a server.

#### Scenario: Add a workflow
- **WHEN** an operator adds a valid workflow file
- **THEN** it appears in the catalog with its digest

#### Scenario: Remove a workflow
- **WHEN** an operator removes a workflow
- **THEN** it no longer appears in the catalog and runs already started are unaffected

### Requirement: Plugin management
The plugin commands SHALL add, list, and remove plugins. Adding SHALL show the command the plugin will launch and ask for confirmation unless told to proceed.

#### Scenario: List
- **WHEN** an operator lists plugins
- **THEN** each is shown with its name, version, and the steps, triggers, channels, and harnesses it provides

### Requirement: Export to SDK code
The export command SHALL convert a workflow written in YAML into equivalent source code for the Go SDK.

#### Scenario: Round trip
- **WHEN** a workflow is exported to Go and the program is run to build its IR
- **THEN** the IR equals the IR of the original YAML

### Requirement: Client launchers
The CLI SHALL provide commands that open the terminal client and that serve the web client, each connecting to a running engine or starting an embedded one.

#### Scenario: Embedded terminal client
- **WHEN** an operator opens the terminal client with no engine running
- **THEN** an embedded engine starts for the life of the client

### Requirement: Configuration and state location
The CLI SHALL read configuration from a documented file and SHALL keep its state, including the event log, in a documented directory that can be changed by an option or environment variable.

#### Scenario: Custom state directory
- **WHEN** an operator sets the state directory
- **THEN** the event log and installed plugins are kept there

### Requirement: Store configuration
The configuration file SHALL select the store and MAY name a third-party store by path or git reference. The serve command and the client launchers that start an embedded engine SHALL use the local durable store by default, and the run command with no server SHALL use the memory store by default.

#### Scenario: Default store for serve
- **WHEN** an operator runs the serve command with no store configured
- **THEN** the engine uses the local durable store

#### Scenario: Third-party store
- **WHEN** the configuration names a third-party store
- **THEN** the engine resolves it like a plugin and shows its launch command on first use

### Requirement: Store conformance command
The CLI SHALL provide a command that runs the conformance suite against the configured store, or against a store named on the command line, and exits with a status that reflects the result.

#### Scenario: Conformant store
- **WHEN** an operator runs the command against a store that passes every check for its capabilities
- **THEN** the command prints the results and exits successfully

#### Scenario: Failing store
- **WHEN** a check fails
- **THEN** the command names the failing check and exits with a failure status

### Requirement: Namespace option
The run, deliver, answer, workflow, status, and serve commands SHALL accept a namespace and SHALL use default when none is given.

#### Scenario: Status in a namespace
- **WHEN** an operator runs the status command with a namespace
- **THEN** only the runs of that namespace are printed
