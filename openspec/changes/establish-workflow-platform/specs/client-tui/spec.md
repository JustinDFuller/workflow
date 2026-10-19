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
The client SHALL show a run's workflow as a graph in the terminal with the current step marked, completed steps marked, wait steps distinguished from other steps, loop iteration counts shown on repeated steps, and a badge on each agent step that relies on the core ask route.

#### Scenario: Current step marked
- **WHEN** a run is on its third verification pass
- **THEN** the graph marks the verify step as current with the count three

#### Scenario: Wait step shown
- **WHEN** a run is waiting at a review
- **THEN** the graph marks the wait step as current and shows it as a wait

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

### Requirement: Waiting runs
The client SHALL distinguish a waiting run from a run that needs the operator, and for a waiting run SHALL show what it waits for, such as a review of a named pull request or a time, and for how long it has waited.

#### Scenario: Waiting for a review
- **WHEN** a run is waiting on a review of a pull request
- **THEN** its row shows waiting, the pull request, and the time since the wait began, and is not marked as needing the operator

#### Scenario: Several triggers
- **WHEN** a wait lists a review and a deadline
- **THEN** the run view lists both

### Requirement: Questions with draft work
The client SHALL show the questions of an attention request together with the draft outputs, notes, and summary the step recorded before it asked, so the operator can answer with the draft in view.

#### Scenario: Questions and a draft plan
- **WHEN** a step exited with questions and a draft plan
- **THEN** the request view shows each question and the draft plan

### Requirement: Step prompt and context
The client SHALL let the operator see the text an agent step received, including its context sections, by explicit action.

#### Scenario: See what the agent got
- **WHEN** the operator asks to see the prompt of a step
- **THEN** the client shows the assembled text

### Requirement: Interactive agent launch details
For an interactive profile, the client SHALL show how to reach the externally hosted agent, and SHALL NOT attach to its terminal.

#### Scenario: Launched agent
- **WHEN** an agent step has launched an interactive agent
- **THEN** the run view shows how to reach the host and that the run is waiting for the agent to finish

### Requirement: Reconnect
The client SHALL reconnect to the engine after a disconnect and resume from the last event it saw, showing a connection indicator while disconnected.

#### Scenario: Engine restarts
- **WHEN** the engine restarts while the client is open
- **THEN** the client shows it is disconnected, then reconnects and shows the current state
