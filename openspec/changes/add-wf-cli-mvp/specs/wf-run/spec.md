# Spec Delta

## Purpose

Defines how a run is started, bound to a worktree, recorded, and moved forward by the two agent commands, so that only the workflow's own checks decide where a run goes and a waiting agent can never be stuck without a limit.

## ADDED Requirements

### Requirement: Starting a run
The start command SHALL load and validate a workflow, take its inputs, create a run bound to the current git worktree, record the workflow definition the run started with, and enter the start step. A run SHALL be refused if the worktree already has a run that has not ended or stopped.

#### Scenario: Run starts
- **WHEN** a person starts a valid workflow with its required inputs in a worktree with no active run
- **THEN** a run exists bound to that worktree and the output shows the start step

#### Scenario: Invalid workflow
- **WHEN** the workflow fails validation
- **THEN** no run is created and the problems are shown

#### Scenario: Worktree already has a run
- **WHEN** the worktree has a run that is neither ended nor stopped
- **THEN** the start is refused and names that run

#### Scenario: Replacing a finished run
- **WHEN** the worktree's previous run has ended or stopped
- **THEN** a new run starts and the previous run stays in history

### Requirement: Run binding
Every agent and hook command SHALL find its run from the git worktree of the current directory, from any subdirectory of it. A command MAY name a run with a run option instead. A command run outside any worktree with a run SHALL say so.

#### Scenario: From a subdirectory
- **WHEN** the next command is run in a subdirectory of a bound worktree
- **THEN** it acts on that worktree's run

#### Scenario: No run
- **WHEN** the next command is run in a worktree with no run
- **THEN** it says there is no run and how to start one

### Requirement: Event log
A run SHALL be recorded as an append-only event log in the state directory, and the run's current state SHALL be derived from the log. The state directory SHALL default to a directory under the home directory and SHALL be changeable with an environment variable. Evaluations of one run SHALL not overlap.

#### Scenario: State from the log
- **WHEN** the next command runs in a new process
- **THEN** it finds the run's step, visits, and route values from the log alone

#### Scenario: Concurrent evaluations
- **WHEN** a background wait and the agent's next command evaluate the same run at once
- **THEN** the evaluations run one after the other and the run advances once

#### Scenario: Moved state directory
- **WHEN** the environment variable names another directory
- **THEN** runs are read and written there

### Requirement: Next command
The next command SHALL evaluate the current step's checks without waiting for any outside event, record the result, and print the step, the state, the job, the reason the step is not over, and the exact command the agent should run next. When every check passes it SHALL move the run to the target, enter that step, and evaluate it in the same way, until a step does not pass or the run reaches end or stop.

#### Scenario: Work remains
- **WHEN** a check fails with "3 tasks remain"
- **THEN** the output shows the job, the reason "3 tasks remain", and the next command

#### Scenario: Step passes
- **WHEN** every check on the current step passes
- **THEN** the run moves to the next step and the output shows that step

#### Scenario: Several steps pass at once
- **WHEN** the next step has no job and its checks already pass
- **THEN** the run moves through it and the output shows the first step that does not pass

#### Scenario: Never blocks
- **WHEN** a check reports pending
- **THEN** the command returns at once with the pending state

### Requirement: Run states
Every evaluation SHALL end in one of five states and the commands SHALL exit with a different code for each. Work means the step's job is not finished. Pending means a check waits on something outside. Broken means a check could not run. Stopped means the run stopped and cannot continue. Ended means the run finished.

#### Scenario: Distinct exit codes
- **WHEN** a command ends in each of the five states
- **THEN** each exit code is different, and none of them is the code for a usage error

#### Scenario: Job step with a failing check
- **WHEN** a step with a job has a failing check
- **THEN** the state is work

#### Scenario: Wait step pending
- **WHEN** a step with no job has a pending check
- **THEN** the state is pending

### Requirement: Steps with no job
On a step with no job, a failing check SHALL stop the run with the check's message, because nothing is left for the agent to do. A passing step SHALL advance and a pending step SHALL rest.

#### Scenario: Gate fails
- **WHEN** a step with no job has a check that fails with "proposal not approved"
- **THEN** the run is stopped with that reason and the output says it stopped

### Requirement: Agent cannot report outcomes
The workflow SHALL NOT accept an outcome, a route, or a step from the agent. No command SHALL move a run to a named step, approve, retry, or cancel a run. A run SHALL move only when a check passes or reports an outcome, or when a limit stops it.

#### Scenario: No steering command
- **WHEN** the list of commands is inspected
- **THEN** none of them takes an outcome or a step name as a way to move a run

### Requirement: Route values
Values that an outcome check reports SHALL be passed to the step the outcome routes to, shown there under a heading naming where they came from, and offered to that step's checks.

#### Scenario: Review comments reach the next step
- **WHEN** a review check reports changes_requested with comments
- **THEN** the revise step's output shows the comments

### Requirement: Visit limits
Entering a step SHALL count as a visit, and a run SHALL stop when it would exceed the step's maximum number of visits, with a reason naming the step and the limit. The default maximum SHALL be ten.

#### Scenario: Limit reached
- **WHEN** a loop would enter a step for the eleventh time with the default maximum
- **THEN** the run stops and the reason names the step and the limit

### Requirement: Timeouts
A step's timeout SHALL run from the moment the step was entered. When an evaluation finds the timeout passed and the checks do not pass, the run SHALL stop with a reason naming the step and the timeout. A step whose checks pass SHALL advance even if the timeout has passed.

#### Scenario: Review never arrives
- **WHEN** a wait step with a timeout of fourteen days is still pending after fourteen days
- **THEN** the next evaluation stops the run and the reason says the wait timed out

#### Scenario: Late but present
- **WHEN** the review arrives after the timeout and before any evaluation
- **THEN** the checks pass and the run advances

### Requirement: Stopped and ended
A stopped run SHALL stay stopped with its reason, and an ended run SHALL stay ended. Neither SHALL evaluate checks. A person SHALL continue by editing the workflow or the repository and starting a new run.

#### Scenario: Evaluating a stopped run
- **WHEN** the next command is run on a stopped run
- **THEN** it prints the stop reason and does not run any check

### Requirement: Wait command
The wait command SHALL be the only command that blocks. It SHALL evaluate the checks, return at once when the state is not pending, and otherwise evaluate again at an interval until the state changes, a check breaks, the run reaches a timeout, or the command's own timeout passes. It SHALL then print what the next command would. Its timeout SHALL default to ninety seconds and its interval to thirty seconds.

#### Scenario: Not pending
- **WHEN** the wait command is run and the state is work
- **THEN** it returns at once with that state

#### Scenario: Resolves
- **WHEN** a pending review arrives during a wait
- **THEN** the wait ends, the run advances, and the output shows the next step

#### Scenario: Command timeout
- **WHEN** nothing changes before the wait command's timeout
- **THEN** it returns the pending state and the command to wait again

#### Scenario: Broken check
- **WHEN** a check breaks during a wait
- **THEN** the wait ends at once and shows the error

#### Scenario: Default is short
- **WHEN** the wait command is run with no timeout
- **THEN** it returns within about ninety seconds

### Requirement: Waiter heartbeat
While the wait command runs it SHALL record a heartbeat each time it evaluates, and it SHALL remove its heartbeat when it ends for any reason. A waiter SHALL count as live while its latest heartbeat is no older than twice its interval.

#### Scenario: Live waiter
- **WHEN** a wait command has evaluated within twice its interval
- **THEN** the run has a live waiter

#### Scenario: Killed waiter
- **WHEN** the wait process is killed without cleaning up
- **THEN** the waiter stops counting as live after twice its interval

### Requirement: Long waits are explicit
When the state is pending, the printed next command SHALL be the wait command with a timeout equal to the time left on the step, and SHALL tell the agent to run it in the background. When the step is marked slow, the printed next command for any state SHALL tell the agent to run the next command in the background.

#### Scenario: Review wait
- **WHEN** a review step with fourteen days left is pending
- **THEN** the output tells the agent to run the wait command with a timeout of fourteen days in the background

#### Scenario: Slow step
- **WHEN** the current step is marked slow
- **THEN** the output tells the agent to run the next command in the background

### Requirement: Changes are recorded only when they differ
The run SHALL record a check result only when it differs from the last result recorded for the same visit of the step, so that polling for days does not grow the log.

#### Scenario: Polling a pending review
- **WHEN** a wait evaluates a pending check a thousand times with the same message
- **THEN** one result event is recorded

### Requirement: Machine-readable output
The next and wait commands SHALL accept an option that prints the same information as JSON.

#### Scenario: JSON
- **WHEN** the next command is run with the JSON option
- **THEN** the output is a JSON object with the step, state, job, reason, and next command
