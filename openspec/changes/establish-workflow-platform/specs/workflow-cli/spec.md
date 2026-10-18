# Spec Delta

## Purpose

Defines the single command-line program that serves the engine, runs and validates workflows, manages plugins, and exports workflows, so the platform is usable from a terminal, a script, or a CI job with nothing else installed.

## ADDED Requirements

### Requirement: Single executable
The platform SHALL ship as one executable named workflow that includes the engine, the built-in plugins, and the built-in clients, and needs no separately installed runtime.

#### Scenario: Fresh machine
- **WHEN** the executable is copied to a machine with no other platform files
- **THEN** it can validate and run a workflow that uses only built-in plugins

### Requirement: Serve
The serve command SHALL start the engine as a long-running process that accepts client connections, hosts triggers, and resumes unfinished runs. It SHALL print the address clients can use.

#### Scenario: Serving
- **WHEN** an operator runs the serve command
- **THEN** the engine starts, resumes unfinished runs, and prints its address

#### Scenario: Port in use
- **WHEN** the configured port is already in use
- **THEN** the command exits with an error that names the port

### Requirement: Run to completion
The run command SHALL start a workflow, wait until the run reaches a final state, and exit with a status that reflects the result. When the run needs the operator, the command SHALL print the request and wait for an answer unless told not to.

#### Scenario: Succeeded
- **WHEN** a run ends succeeded
- **THEN** the command exits with status zero

#### Scenario: Failed
- **WHEN** a run ends failed or cancelled
- **THEN** the command exits with a non-zero status that distinguishes the two

#### Scenario: Non-interactive
- **WHEN** the command is run with the non-interactive option and a run needs the operator
- **THEN** the command prints the request and exits with a distinct status for needs-attention

### Requirement: Validate
The validate command SHALL check one or more workflows and the plugins they use without running them, print every problem with its file and line, and exit non-zero if any exist.

#### Scenario: Valid
- **WHEN** a workflow is valid
- **THEN** the command prints nothing but a success line and exits zero

#### Scenario: Invalid
- **WHEN** a workflow has an unrouted outcome
- **THEN** the command prints the step, the outcome, and the file and line, and exits non-zero

### Requirement: Status
The status command SHALL print for every run its workflow, state, current step, whether it needs the operator, and its available artifacts, in a human-readable form by default and as JSON on request.

#### Scenario: Machine-readable
- **WHEN** the command is run with the JSON option
- **THEN** it prints a document containing the same information

### Requirement: Plugin management
The plugin commands SHALL add, list, and remove plugins. Adding SHALL show the command the plugin will launch and ask for confirmation unless told to proceed.

#### Scenario: List
- **WHEN** an operator lists plugins
- **THEN** each is shown with its name, version, and the steps, triggers, and harnesses it provides

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
