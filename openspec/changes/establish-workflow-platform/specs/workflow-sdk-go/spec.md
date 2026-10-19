# Spec Delta

## Purpose

Lets people write workflow definitions as Go programs for generation and reuse, while the engine still runs only the shared intermediate representation, so a workflow written in code can be drawn and validated like any other.

## ADDED Requirements

### Requirement: Builder emits IR
The Go SDK SHALL provide a builder for profiles, steps, wait steps and their triggers, outputs, context lists, outcomes, defaults, and transitions whose output is the same IR the YAML format produces. It SHALL NOT run workflow logic at execution time.

#### Scenario: Builder output
- **WHEN** a program builds a workflow and asks for its IR
- **THEN** it receives a valid IR document

### Requirement: Same validation
The SDK SHALL apply the same static validation as the CLI, and report problems when the IR is built.

#### Scenario: Unrouted outcome in code
- **WHEN** a built workflow leaves a declared outcome unrouted
- **THEN** building fails with the same error as validating the equivalent YAML

### Requirement: Reuse of fragments
The SDK SHALL allow a group of steps to be defined once as a fragment and included in several workflows, with step ids made unique by a prefix.

#### Scenario: Fragment included twice
- **WHEN** a fragment is included twice with different prefixes
- **THEN** the IR contains both copies with distinct step ids

### Requirement: Parity with YAML
Every construct available in the YAML format SHALL be available in the SDK, and the reverse.

#### Scenario: Round trip
- **WHEN** an IR produced by the SDK is written as YAML and loaded again
- **THEN** it equals the original IR

### Requirement: Runnable through the CLI
A program built with the SDK SHALL be usable by the CLI by passing the program's IR output, with no change to the engine.

#### Scenario: Validate SDK output
- **WHEN** an operator validates the IR printed by an SDK program
- **THEN** the validate command accepts it
