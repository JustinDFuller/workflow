# Spec Delta

## Purpose

Provides OpenSpec knowledge as a plugin: checking that a change is valid and in the expected state, counting remaining tasks, and exposing a change as an artifact, so workflows can drive the OpenSpec process without the engine knowing OpenSpec.

## ADDED Requirements

### Requirement: Validate a change
The plugin SHALL provide a step that validates OpenSpec files in a working directory in strict mode and reports valid or invalid, with the validator's messages as outputs.

#### Scenario: Invalid
- **WHEN** the files fail strict validation
- **THEN** the step reports invalid with the messages

### Requirement: Task progress
The plugin SHALL provide a step that reports tasks_remaining or complete for a named change, with the counts and the next unchecked task as outputs.

#### Scenario: Tasks remain
- **WHEN** a change has unchecked tasks
- **THEN** the step reports tasks_remaining with the remaining count

#### Scenario: All done
- **WHEN** every task is checked
- **THEN** the step reports complete

### Requirement: Proposal presence check
The plugin SHALL provide a step that reports whether a change contains the artifacts required by the project's schema, naming the missing ones.

#### Scenario: Missing design
- **WHEN** a change lacks a design file the schema requires
- **THEN** the step reports incomplete and names it

### Requirement: Invariant checks
The plugin SHALL provide a step that checks, in one call, that the working directory is a git repository, is not the default branch, contains a valid change, and has any additional invariants the workflow lists, and reports ok or the first failing invariant with a message.

#### Scenario: On the default branch
- **WHEN** the working directory is on the default branch
- **THEN** the step reports violated with the message that work must be on a branch in a worktree

### Requirement: Archive
The plugin SHALL provide a step that archives a named change using the OpenSpec command line and reports done or failed.

#### Scenario: Archived
- **WHEN** the step runs for a completed change
- **THEN** the change is moved to the archive and the step reports done

### Requirement: Change artifact kind
The plugin SHALL declare a change artifact kind with the change name, path, and task progress, and SHALL support probing it.

#### Scenario: Probe shows progress
- **WHEN** the engine probes the artifact
- **THEN** it receives the current checked and total task counts

### Requirement: Skill-prefixed agent prompts
The plugin SHALL provide helper values that name the OpenSpec commands an agent step can use as its command prefix, so a workflow can write explore, propose, apply, and verify steps without hard-coding command text.

#### Scenario: Prefix for apply
- **WHEN** a workflow references the apply prefix
- **THEN** it resolves to the command prefix for the apply workflow in the project
