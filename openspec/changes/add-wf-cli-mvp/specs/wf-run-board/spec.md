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
The runs command and the show command SHALL each accept an option that redraws their output at an interval until interrupted.

#### Scenario: Step changes
- **WHEN** a run advances while the live view of the runs command is open
- **THEN** the next redraw shows the new step

#### Scenario: Detail changes
- **WHEN** a check result changes while the live view of the show command is open
- **THEN** the next redraw shows the new result

### Requirement: Run detail
The show command SHALL print one run's workflow, id, worktree, state, current step, time since the run started, and time in the current step, then the steps in definition order, the current step's detail, the artifacts, and the recent history.

#### Scenario: Position in the workflow
- **WHEN** a run is on the apply step
- **THEN** the steps are listed in order and the apply step is marked as current

#### Scenario: What is done and what is left
- **WHEN** a run is on the apply step
- **THEN** the steps before it show as passed, the apply step shows as current, and the steps after it show as not reached

#### Scenario: Loop counts
- **WHEN** the verify step has been entered three times with a maximum of ten
- **THEN** the detail shows three of ten for that step

#### Scenario: When a step passed
- **WHEN** a step has passed
- **THEN** the list shows how long ago it last passed

### Requirement: Current step detail
For the current step the show command SHALL print the job, the deadline with the time left when the step has one, and every check in order with its last result in the current visit. A result SHALL be pass, fail with its message, pending with its message, broken with its error, or not run yet.

#### Scenario: Checks in order
- **WHEN** the current step lists six checks and the fourth is pending
- **THEN** the first three show pass, the fourth shows pending with its message, and the last two show not run yet

#### Scenario: The job is visible
- **WHEN** a person opens the detail of a run on a step with a job
- **THEN** the job text is shown without reading the agent's session

#### Scenario: Step with no job
- **WHEN** the current step has no job
- **THEN** the detail says the step waits for its checks

### Requirement: Artifacts on the board
The artifacts that checks have reported SHALL be listed by name with their address, which is a URL or a path in the worktree, in the show command and in the runs command's JSON output.

#### Scenario: Several artifacts
- **WHEN** the change directory and a pull request have been reported
- **THEN** both are listed by name with their address

#### Scenario: None yet
- **WHEN** no check has reported an artifact
- **THEN** the detail says there are none

### Requirement: Readable history
The show command SHALL print the most recent events as one line each, with the time and a plain sentence that names the step or check, and not an internal event name.

#### Scenario: Step passed
- **WHEN** the propose step passed and the run moved to the review step
- **THEN** a line says the propose step passed and the run moved to the review step

#### Scenario: Newest first
- **WHEN** the history is printed
- **THEN** the newest event is first

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
