# Spec Delta

## Purpose

Defines what a workflow is: a static, versioned graph of steps whose transitions route on outcomes. Every client can draw it before it runs, and the engine executes only this form.

## ADDED Requirements

### Requirement: Single intermediate representation
The platform SHALL execute workflows only from one versioned intermediate representation (IR) that is a JSON document validated by a published JSON Schema. Every authoring format, including YAML and the code SDK, SHALL produce this IR.

#### Scenario: YAML and SDK produce the same IR
- **WHEN** a workflow is authored in YAML and the same workflow is built with the Go SDK
- **THEN** both produce IR documents that are equal after canonical serialization

#### Scenario: Unknown IR version is rejected
- **WHEN** the engine is given an IR whose version it does not support
- **THEN** it rejects the workflow and reports the supported versions

### Requirement: YAML authoring format
The platform SHALL accept workflows written in YAML whose structure maps one to one onto the IR, with no YAML-only features that change meaning.

#### Scenario: YAML maps to IR
- **WHEN** a valid YAML workflow is loaded
- **THEN** the resulting IR contains the same steps, sessions, outcomes, and transitions as the YAML, and nothing else

#### Scenario: Malformed YAML
- **WHEN** a YAML file cannot be parsed or does not match the schema
- **THEN** loading fails with the file, line, and a description of the problem

### Requirement: Steps and outcomes
A workflow SHALL be a set of steps, each with a unique id, a reference to the plugin step it runs, its inputs, and a map from each outcome to a transition target. A transition target SHALL be another step id, a terminal state, or the core needs-human state.

#### Scenario: Outcome routing
- **WHEN** a step resolves with an outcome that has a transition target
- **THEN** the run moves to that target

#### Scenario: Unrouted outcome
- **WHEN** a step resolves with an outcome that has no transition target
- **THEN** the run moves to needs-human and an attention request describes the unrouted outcome

### Requirement: Step kinds
A step SHALL be exactly one of four kinds: a plugin step that runs a step provided by a plugin, an agent step that prompts a named session, a gate step that waits for an operator to choose an outcome, or a workflow step that runs a sub-workflow. The set of kinds SHALL NOT be extended by plugins.

#### Scenario: Agent step
- **WHEN** a step is an agent step
- **THEN** it names a session, a prompt, and the outcomes the agent may report

#### Scenario: Gate step
- **WHEN** a step is a gate step
- **THEN** it declares a prompt and the outcomes the operator may choose

#### Scenario: Two kinds in one step
- **WHEN** a step declares both a plugin step and a session
- **THEN** validation fails and states that a step has exactly one kind

### Requirement: Default route
A step MAY declare a default transition target that applies to every outcome the step can report that has no transition of its own. Validation SHALL treat outcomes covered by a default as routed.

#### Scenario: Default covers the rest
- **WHEN** a step routes approved explicitly and declares a default of needs-human
- **THEN** every other outcome the step can report moves the run to needs-human

### Requirement: Values passed along a transition
A transition MAY pass named values to its target step, each given as a reference or literal. The target step SHALL be able to read them under the route namespace, and validation SHALL fail if a step reads a route value that some incoming transition does not pass.

#### Scenario: Feedback passed to a revision step
- **WHEN** a review step routes changes_requested to a revision step and passes the review comments as feedback
- **THEN** the revision step reads the comments as a route value

#### Scenario: Missing route value
- **WHEN** a step reads a route value and one transition into it passes no such value
- **THEN** validation fails and names the transition

### Requirement: Terminal states
A workflow SHALL end in one of three terminal states: succeeded, failed, or cancelled. A workflow SHALL declare a start step.

#### Scenario: Reaching a terminal state
- **WHEN** a transition targets a terminal state
- **THEN** the run ends in that state and no further steps start

### Requirement: Needs-human routing target
The needs-human target SHALL pause the run and open an attention request that includes the reason and the step that routed there. The run SHALL resume at a step chosen by the operator's answer, or at the routing step when the operator retries.

#### Scenario: Invariant failure asks for help
- **WHEN** a check step resolves with an outcome routed to needs-human
- **THEN** the run is paused, an attention request is opened with the failing step and message, and the run does not advance until it is answered

### Requirement: Loops with bounds
A transition MAY target an earlier step or the same step, forming a loop. Every step that can be re-entered SHALL declare a maximum number of iterations, and the engine SHALL enforce it.

#### Scenario: Loop within bound
- **WHEN** a step routes back to itself and has run fewer times than its maximum
- **THEN** it runs again and the iteration count increases

#### Scenario: Loop exceeds bound
- **WHEN** a step would run more times than its maximum
- **THEN** the run moves to needs-human with the iteration count in the request instead of running the step

#### Scenario: Unbounded loop rejected
- **WHEN** a workflow contains a cycle in which no step declares a maximum
- **THEN** validation fails and names the cycle

### Requirement: No expressions
The definition language SHALL support only references to values and routing on outcomes. It SHALL NOT support arithmetic, comparison, conditionals, or user-defined functions.

#### Scenario: Expression rejected
- **WHEN** a value contains an expression such as a comparison or a function call
- **THEN** validation fails and states that logic belongs in a plugin step

### Requirement: References
A value MAY contain references in the form of a dollar sign followed by double braces around a path. The paths available SHALL be the trigger payload, workflow inputs, the outputs of earlier steps, the artifacts of earlier steps, route values, constants published by plugins, and run metadata.

#### Scenario: Reference to an earlier step output
- **WHEN** a step input references the output of a step that can precede it on every path
- **THEN** the value is resolved from that output when the step starts

#### Scenario: Reference to an unavailable value
- **WHEN** a reference names a step that cannot have run before the referencing step on some path
- **THEN** validation fails and names the reference and the path

### Requirement: Named sessions
A workflow SHALL declare agent sessions by name, each with a harness plugin, a mode of headless or interactive, and an optional fresh-context flag. Agent steps SHALL name the session they run in.

#### Scenario: Steps continue a session
- **WHEN** two agent steps name the same session
- **THEN** the second continues the conversation of the first

#### Scenario: Fresh session
- **WHEN** a session is declared fresh
- **THEN** each step that uses it starts a new conversation with no prior context

#### Scenario: Mixed harnesses
- **WHEN** one workflow declares sessions on two different harness plugins
- **THEN** each step runs on the harness of the session it names

### Requirement: Sub-workflows
A step MAY run another workflow as a sub-workflow, passing inputs and receiving an outcome and outputs. A sub-workflow MAY declare sessions it requires, and the parent SHALL bind each to one of its own sessions. The parent run SHALL show the sub-workflow's current step as detail beneath the parent step.

#### Scenario: Sub-workflow outcome
- **WHEN** a sub-workflow ends in a terminal state
- **THEN** the parent step resolves with the outcome succeeded, failed, or cancelled matching that state

#### Scenario: Session binding
- **WHEN** a sub-workflow requires a session named author and the parent binds it to its own session
- **THEN** agent steps in the sub-workflow run in the parent's session

### Requirement: Expected artifacts
A workflow or a step MAY declare the artifacts it is expected to produce, each with a kind and a name. The engine SHALL display expected artifacts as not yet available until they are reported or detected.

#### Scenario: Step-level declaration
- **WHEN** a step declares an expected artifact of kind pull request
- **THEN** the run shows that artifact as expected from the moment the step starts

#### Scenario: Workflow-level declaration
- **WHEN** a workflow declares an expected artifact
- **THEN** the run shows it as expected from the moment the run starts

### Requirement: Static validation
The platform SHALL validate a workflow without running it. Validation SHALL check that the schema is satisfied, step ids are unique, the start step exists, every transition target exists, every outcome a step's plugin declares is routed, every step is reachable, and every referenced plugin step exists with valid inputs.

#### Scenario: Unrouted declared outcome
- **WHEN** a plugin step declares outcomes approved and changes_requested and the workflow routes only approved
- **THEN** validation fails and names the unrouted outcome

#### Scenario: Unreachable step
- **WHEN** a step cannot be reached from the start step
- **THEN** validation fails and names the step

#### Scenario: Valid workflow
- **WHEN** a workflow satisfies all checks
- **THEN** validation succeeds and reports no errors

### Requirement: Graph is available before a run
The platform SHALL be able to return the full graph of a workflow, including every step, transition, and expected artifact, without starting a run.

#### Scenario: Drawing before running
- **WHEN** a client asks for a workflow's graph
- **THEN** it receives all steps and transitions
