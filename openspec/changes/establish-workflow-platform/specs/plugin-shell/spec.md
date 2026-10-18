# Spec Delta

## Purpose

Provides the simplest useful plugin: run a command and turn its result into an outcome. It lets workflows call any CLI, serves as the reference plugin for the protocol, and lets the engine be tested without any agent or service.

## ADDED Requirements

### Requirement: Run a command
The plugin SHALL provide a step that runs a command in a given working directory with given environment variables and reports captured output as step outputs.

#### Scenario: Command succeeds
- **WHEN** the step runs a command that exits zero
- **THEN** it reports the outcome success with the standard output and exit code as outputs

#### Scenario: Command fails
- **WHEN** the command exits non-zero
- **THEN** it reports the outcome failure with the exit code and the tail of standard error as outputs

### Requirement: Outcome mapping
The step SHALL allow the workflow to map exit codes, and patterns matched in standard output, to named outcomes. The set of possible outcomes SHALL be declared by the workflow so validation can check routing.

#### Scenario: Exit code mapped
- **WHEN** the mapping says exit code 2 is the outcome tasks_remaining and the command exits 2
- **THEN** the step reports tasks_remaining

#### Scenario: Pattern mapped
- **WHEN** the mapping says output matching a pattern is the outcome approved and the output matches
- **THEN** the step reports approved

#### Scenario: Nothing matches
- **WHEN** neither an exit code mapping nor a pattern matches
- **THEN** the step reports the outcome unmatched with the output attached

### Requirement: Timeout and cancellation
The step SHALL stop the command when it is cancelled or its timeout elapses, ending its child processes.

#### Scenario: Cancelled
- **WHEN** the engine cancels the step
- **THEN** the command and its children are terminated

### Requirement: No shell by default
The step SHALL run the command as an argument list without a shell unless the workflow explicitly selects a shell.

#### Scenario: Arguments with spaces
- **WHEN** an argument contains spaces or shell metacharacters
- **THEN** it is passed to the command unchanged

### Requirement: Polling wait
The plugin SHALL provide a wait step that re-runs a command at an interval until its result maps to an outcome other than a declared keep-waiting outcome.

#### Scenario: Wait until ready
- **WHEN** the command reports keep-waiting twice and then ready
- **THEN** the step stays active and then resolves with ready
