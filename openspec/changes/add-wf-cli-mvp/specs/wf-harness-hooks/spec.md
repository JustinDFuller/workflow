# Spec Delta

## Purpose

Defines the Claude Code hooks, status line, and install command that keep an agent on its workflow without a long-running process, so that the referee is enforced by the harness and not only requested in prose.

## ADDED Requirements

### Requirement: Hooks fail open
Every hook command SHALL allow the action when the worktree has no run, when the run cannot be read, or when the hook itself fails. A hook SHALL NOT run any check.

#### Scenario: No run
- **WHEN** a hook runs in a worktree with no run
- **THEN** it allows the action and prints nothing

#### Scenario: Unreadable state
- **WHEN** the run's log cannot be read
- **THEN** the hook allows the action

#### Scenario: Hook never evaluates
- **WHEN** a hook runs while a slow check is running for the run
- **THEN** the hook answers from the last recorded state at once

### Requirement: Stop hook
The stop hook SHALL decide from the last recorded state of the run. It SHALL allow the agent to stop when there is no run, or the run has ended, stopped, or is broken, or the run is pending and a live waiter exists. It SHALL block the stop when the state is work, or when the state is pending and no waiter is live, and the reason it gives SHALL be the text the next command would print.

#### Scenario: Work remains
- **WHEN** the agent tries to stop while the last recorded state is work
- **THEN** the stop is blocked and the reason shows the step, the job, and the next command

#### Scenario: Waiting with a live waiter
- **WHEN** the agent tries to stop while the run is pending and a background wait is live
- **THEN** the stop is allowed

#### Scenario: Waiting without a waiter
- **WHEN** the agent tries to stop while the run is pending and no waiter is live
- **THEN** the stop is blocked and the reason tells the agent to start the wait command in the background

#### Scenario: Run is over
- **WHEN** the run has ended or stopped
- **THEN** the stop is allowed

#### Scenario: Broken check
- **WHEN** the last recorded state is broken
- **THEN** the stop is allowed so that the agent can tell the person

### Requirement: Stop hook progress guard
The stop hook SHALL allow the agent to stop after it has blocked five times in a row with no new event recorded for the run, and SHALL record that it did so.

#### Scenario: Agent cannot make progress
- **WHEN** the stop hook has blocked five times and nothing was recorded between the blocks
- **THEN** the next stop is allowed and the log records that the guard released it

#### Scenario: Progress resets the guard
- **WHEN** an event is recorded between two blocks
- **THEN** the count starts again

### Requirement: Tool hook
The tool hook SHALL deny a tool call that uses a tool the current step lists as denied, with a reason that names the step. It SHALL also deny file edits whose target is inside the state directory and shell commands that mention the state directory.

#### Scenario: Denied tool
- **WHEN** a step denies the edit tool and the agent tries to edit a file
- **THEN** the call is denied and the reason names the step

#### Scenario: Editing the run files
- **WHEN** the agent tries to edit a file in the state directory
- **THEN** the call is denied

#### Scenario: Shell command on the run files
- **WHEN** the agent runs a shell command that names the state directory
- **THEN** the call is denied

#### Scenario: Allowed tool
- **WHEN** the tool is not denied by the current step
- **THEN** the call is allowed

### Requirement: Session hook
The session hook SHALL, when a session starts, resumes, is cleared, or is compacted, print the current step, its job, and the next command into the agent's context when a run exists.

#### Scenario: After compaction
- **WHEN** the session is compacted during a run
- **THEN** the agent's context receives the current step, its job, and the next command

#### Scenario: No run
- **WHEN** the session starts in a worktree with no run
- **THEN** nothing is added

### Requirement: Status line
The status line command SHALL print one line with the workflow, the step, the visit against the step's maximum, and the state, and SHALL mark a run that needs the person. It SHALL print nothing when there is no run, and it SHALL read only the log.

#### Scenario: Running
- **WHEN** a run is on the verify step on its third visit with a maximum of ten
- **THEN** the line names the workflow, the step, three of ten, and the state

#### Scenario: Needs you
- **WHEN** the run is pending on a step marked as needing the person
- **THEN** the line carries a marker that it needs the person

### Requirement: Install command
The install command for Claude Code SHALL write the hooks, the status line, a permission that lets the agent call the workflow's own commands without a prompt, and a slash command that starts a run, into the project's local Claude Code settings. It SHALL be safe to run again and SHALL report each change it made.

#### Scenario: First install
- **WHEN** the install command runs in a project with no workflow settings
- **THEN** the hooks, status line, permission, and slash command are written and listed

#### Scenario: Second install
- **WHEN** the install command runs again
- **THEN** nothing changes and the output says everything is already installed

#### Scenario: Existing settings kept
- **WHEN** the settings file has unrelated keys
- **THEN** they are kept

### Requirement: Limits of enforcement are stated
The install command and the documentation SHALL say that the hooks make it hard for an agent to forge the run, not impossible, and that the log records every check result so that a person can audit it.

#### Scenario: Install output
- **WHEN** the install command finishes
- **THEN** its output says what the hooks do and do not prevent
