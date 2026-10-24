# Spec Delta

## Purpose

Defines how a check command is run and how its exit code and output become a result the run can route on, so that anyone can write a check in any language and the agent cannot influence what a check reports.

## ADDED Requirements

### Requirement: Check execution
A check SHALL run as a shell command in the root of the run's worktree. It SHALL receive the run id, the step name, the worktree path, the workflow directory, the time the step was entered, the run's inputs, and the route values passed into the step, all as environment variables.

#### Scenario: Environment
- **WHEN** a check runs
- **THEN** it can read the run id, step name, step entry time, inputs, and route values from its environment

#### Scenario: Working directory
- **WHEN** a check runs
- **THEN** its working directory is the root of the worktree the run is bound to

### Requirement: Result from exit code
A check SHALL report pass with exit code 0, fail with exit code 1, and pending with exit code 2. Any other exit code SHALL report broken. The check's standard output SHALL be the message for the result, and for broken the standard error SHALL be appended.

#### Scenario: Pass
- **WHEN** a check exits with 0
- **THEN** the result is pass

#### Scenario: Fail with a reason
- **WHEN** a check exits with 1 and prints "3 tasks remain"
- **THEN** the result is fail and the message is "3 tasks remain"

#### Scenario: Pending
- **WHEN** a check exits with 2 and prints "checks are running"
- **THEN** the result is pending and the message is "checks are running"

#### Scenario: Broken
- **WHEN** a check exits with any other code
- **THEN** the result is broken and the message carries what the check printed, including its standard error

### Requirement: Named outcomes
A check that declares outcomes SHALL report an outcome by exiting 0 and printing a JSON object with an outcome and an optional map of string values. The outcome SHALL be one the check declares. Exit codes 1 and 2 SHALL keep their meanings.

#### Scenario: Outcome with values
- **WHEN** a check that declares changes_requested exits 0 and prints an outcome of changes_requested with a comments value
- **THEN** the result is that outcome and the comments value is kept as a route value

#### Scenario: Undeclared outcome
- **WHEN** a check prints an outcome it did not declare
- **THEN** the result is broken and the message names the outcome

#### Scenario: Not JSON
- **WHEN** a check that declares outcomes exits 0 and prints something that is not a JSON object
- **THEN** the result is broken

### Requirement: Ordered evaluation
A step's checks SHALL run in the listed order and SHALL stop at the first check that does not pass. That check's result SHALL be the step's result. A step whose checks all pass SHALL be over.

#### Scenario: First failure decides
- **WHEN** the second of three checks fails
- **THEN** the third does not run and the step's result is the second check's failure

#### Scenario: All pass
- **WHEN** every check on a step passes
- **THEN** the step is over

### Requirement: Check timeout
A check that runs longer than its timeout SHALL be stopped along with every process it started, and SHALL report broken with a message that it timed out. The default timeout SHALL be two minutes.

#### Scenario: Slow check
- **WHEN** a check with a one minute timeout is still running after one minute
- **THEN** the check and its child processes are stopped and the result is broken

### Requirement: Output limit
The message kept from a check SHALL be limited to 64 KiB, and a longer message SHALL be cut with a marker saying it was cut.

#### Scenario: Long output
- **WHEN** a check prints more than 64 KiB
- **THEN** the kept message is cut and ends with the marker

### Requirement: Result cache by commit
A check that sets the commit cache SHALL have its pass, fail, and outcome results stored and reused while the worktree's commit and uncommitted changes are unchanged. Pending and broken results SHALL NOT be cached.

#### Scenario: Reused
- **WHEN** a cached check is evaluated twice with no change to the worktree between
- **THEN** the command runs once

#### Scenario: New commit
- **WHEN** a commit is made between two evaluations
- **THEN** the command runs again

#### Scenario: Uncommitted change
- **WHEN** a file is edited and not committed between two evaluations
- **THEN** the command runs again

#### Scenario: Not cached
- **WHEN** a cached check reports pending
- **THEN** the next evaluation runs the command

### Requirement: Artifacts reported by checks
A check MAY print lines of the form "artifact: name url" in its standard output with any result. The run SHALL record each artifact by name and show it on the board, and a later report of the same name SHALL replace the earlier one.

#### Scenario: Pull request found
- **WHEN** a check passes and prints "artifact: proposal_pr https://example.test/pull/1"
- **THEN** the run records an artifact named proposal_pr with that address

#### Scenario: Replaced
- **WHEN** a later check run prints a different address for the same name
- **THEN** the artifact has the new address
