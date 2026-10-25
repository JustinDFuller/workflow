# Spec Delta

## Purpose

A local web client that draws each run's workflow as a live graph, so people can see where every run is, which need them, and what they produced, from a browser.

## ADDED Requirements

### Requirement: Served by the engine
The engine's serve command SHALL serve the web client from the same executable on the loopback interface, and the client SHALL connect to the engine using the client protocol only.

#### Scenario: Open in browser
- **WHEN** an operator starts the engine and opens the printed address in a browser
- **THEN** the web client loads and shows current runs

#### Scenario: No private API
- **WHEN** the client is inspected
- **THEN** every call it makes is a documented client protocol method

### Requirement: Run overview
The client SHALL show a card per run with the workflow, the current step, the run state, a prominent needs-you indicator, and the artifacts, updating live without reloading.

#### Scenario: Live update
- **WHEN** a run moves to the next step
- **THEN** its card updates without a page reload

### Requirement: Live workflow graph
The client SHALL draw a run's workflow as a graph with the current step highlighted, finished steps distinguished from pending ones, wait steps drawn differently from other steps, loops shown with their iteration counts, a badge on each agent step that relies on the core ask route, and the path taken visible.

#### Scenario: Highlighted position
- **WHEN** a run is on the review wait step
- **THEN** that node is highlighted and the previous nodes show as done

#### Scenario: Ask badge
- **WHEN** an agent step has no route for questions
- **THEN** the node shows an ask badge and no extra edge

#### Scenario: Graph before running
- **WHEN** an operator opens a workflow that has no runs
- **THEN** the full graph is drawn with nothing highlighted

### Requirement: Attention inbox
The client SHALL provide an inbox of open attention requests across all runs, each answerable in place, and SHALL update the browser tab title with the number of open requests.

#### Scenario: Answer in place
- **WHEN** the operator answers a request in the inbox
- **THEN** it disappears and the run continues

#### Scenario: Count in title
- **WHEN** two requests are open
- **THEN** the tab title shows the count two

### Requirement: Artifact links
The client SHALL list each run's artifacts, with expected artifacts shown as pending and available ones as links.

#### Scenario: Available link
- **WHEN** a pull request artifact becomes available
- **THEN** it appears as a link on the run

### Requirement: Accessibility and themes
The client SHALL be operable with a keyboard, SHALL NOT rely on color alone to show state, and SHALL follow the system light or dark setting.

#### Scenario: Colorblind-safe state
- **WHEN** a run needs the operator
- **THEN** the state is shown by a label or icon as well as a color

### Requirement: Waiting runs
The client SHALL distinguish a waiting run from a run that needs the operator, and for a waiting run SHALL show what it waits for and for how long it has waited.

#### Scenario: Waiting for a review
- **WHEN** a run is waiting on a review of a pull request
- **THEN** its card shows waiting, the pull request, and the time since the wait began, and no needs-you indicator

### Requirement: Questions with draft work
The client SHALL show the questions of an attention request together with the draft outputs, notes, and summary the step recorded before it asked, so the operator can answer with the draft in view.

#### Scenario: Questions and a draft plan
- **WHEN** a step exited with questions and a draft plan
- **THEN** the inbox entry shows each question and the draft plan

### Requirement: Step prompt and context
The client SHALL let the operator see the text an agent step received, including its context sections, by explicit action.

#### Scenario: See what the agent got
- **WHEN** the operator opens a step's prompt
- **THEN** the client shows the assembled text

### Requirement: Interactive agent launch details
For an interactive profile, the client SHALL show how to reach the externally hosted agent, and SHALL NOT embed its terminal.

#### Scenario: Launched agent
- **WHEN** an agent step has launched an interactive agent
- **THEN** the run card shows how to reach the host and that the run is waiting for the agent to finish

### Requirement: Reconnect
The client SHALL reconnect after a lost connection and resume from the last event received.

#### Scenario: Laptop wakes
- **WHEN** the connection is lost and then restored
- **THEN** the client shows the current state without a manual reload
