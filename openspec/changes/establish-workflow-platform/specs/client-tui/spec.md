# Spec Delta

## Purpose

A terminal client for people who run many agents. It answers three questions for every run at a glance: which step is it on, does it need me, and what has it produced.

## ADDED Requirements

### Requirement: Overview of all runs
The client SHALL show one row per run with the workflow, the current step with its position in the workflow, a visible needs-you indicator when an attention request is open, and the count of available artifacts.

#### Scenario: Many runs
- **WHEN** five runs are active
- **THEN** all five appear on one screen with their step and needs-you state

#### Scenario: Needs the operator
- **WHEN** a run has an open attention request
- **THEN** its row is visibly distinguished and sorted above runs that do not need the operator

### Requirement: No agent output by default
The client SHALL NOT show agent output or logs in the overview or the run view. Raw output SHALL be available only by explicit action.

#### Scenario: Default view
- **WHEN** an agent produces many lines of output
- **THEN** the overview shows only the changes of step, state, and status text

### Requirement: Run view with graph
The client SHALL show a run's workflow as a graph in the terminal with the current step marked, completed steps marked, and loop iteration counts shown on repeated steps.

#### Scenario: Current step marked
- **WHEN** a run is on its third verification pass
- **THEN** the graph marks the verify step as current with the count three

### Requirement: Answer attention inline
The client SHALL let the operator read an attention request and answer it, including choosing among options and typing free text, without leaving the client.

#### Scenario: Choose an option
- **WHEN** an agent asks a question with three options
- **THEN** the operator can pick one and the run continues

### Requirement: Artifacts list
The client SHALL list a run's artifacts with their state and link, and let the operator open an available artifact's link in the system browser.

#### Scenario: Open a pull request
- **WHEN** the operator opens an available pull request artifact
- **THEN** its link opens in the browser

#### Scenario: Expected artifact
- **WHEN** an artifact is expected but not yet available
- **THEN** it is listed as expected

### Requirement: Run controls
The client SHALL let the operator start a workflow, cancel a run, and retry a run, after confirming destructive actions.

#### Scenario: Cancel confirmation
- **WHEN** the operator asks to cancel a run
- **THEN** the client asks for confirmation before cancelling

### Requirement: Attach to an interactive session
For interactive sessions, the client SHALL let the operator attach to the agent's terminal and detach again.

#### Scenario: Attach and detach
- **WHEN** the operator attaches to an interactive session and then detaches
- **THEN** the session keeps running and the overview is shown again

### Requirement: Reconnect
The client SHALL reconnect to the engine after a disconnect and resume from the last event it saw, showing a connection indicator while disconnected.

#### Scenario: Engine restarts
- **WHEN** the engine restarts while the client is open
- **THEN** the client shows it is disconnected, then reconnects and shows the current state
