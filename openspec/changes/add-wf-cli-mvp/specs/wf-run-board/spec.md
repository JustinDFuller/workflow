# Spec Delta

## Purpose

Defines the read-only commands a person uses to see every run at a glance and one run in detail, so that anyone running several agents can tell which step each is on, whether it needs them, and what it has produced.

## ADDED Requirements

### Requirement: Runs board
The runs command SHALL list the runs in the state directory with each run's id, workflow, worktree, current step, state, whether it needs the person, what it waits on, its artifacts, and how long it has been in the step. By default it SHALL show runs that have not ended, and an option SHALL include ended runs.

#### Scenario: Several runs
- **WHEN** three runs exist in three worktrees
- **THEN** the list shows all three with their step and state

#### Scenario: Ended runs hidden
- **WHEN** a run has ended
- **THEN** it is not listed unless the option to include ended runs is given

#### Scenario: Running with no arguments
- **WHEN** the command is run with no subcommand
- **THEN** it shows the same list

### Requirement: Needs you
A run SHALL be shown as needing the person when its state is stopped or broken, or when it is pending on a step marked as needing the person. Otherwise it SHALL be shown as not needing the person, with a waiting run showing what it waits on.

#### Scenario: Review wait
- **WHEN** a run is pending on a review step marked as needing the person
- **THEN** the board shows that the run needs the person and shows the message the check gave

#### Scenario: Waiting on CI
- **WHEN** a run is pending on a step not marked as needing the person
- **THEN** the board shows that the run does not need the person and shows what it waits on

#### Scenario: Stopped
- **WHEN** a run is stopped
- **THEN** the board shows that it needs the person and shows the stop reason

### Requirement: Live view
The runs command SHALL accept an option that redraws the list at an interval until it is interrupted.

#### Scenario: Step changes
- **WHEN** a run advances while the live view is open
- **THEN** the next redraw shows the new step

### Requirement: Run detail
The show command SHALL print one run's workflow, id, worktree, state, and current step, the steps in definition order with a marker at the current step and each step's visits against its maximum, the last check message, the artifacts, and the most recent events.

#### Scenario: Position in the workflow
- **WHEN** a run is on the apply step
- **THEN** the steps are listed in order and the apply step is marked

#### Scenario: Loop counts
- **WHEN** the verify step has been entered three times with a maximum of ten
- **THEN** the detail shows three of ten for that step

### Requirement: Board is read-only
The runs and show commands SHALL NOT run any check, change any run, or take any lock that an evaluation holds.

#### Scenario: During a slow check
- **WHEN** a slow check is running for a run and the board is opened
- **THEN** the board shows at once from the log

### Requirement: Machine-readable board
The runs and show commands SHALL accept an option that prints JSON.

#### Scenario: JSON list
- **WHEN** the runs command is run with the JSON option
- **THEN** the output is a JSON array with one object per run
