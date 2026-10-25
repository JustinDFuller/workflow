# Spec Delta

## Purpose

Defines what a workflow is: a static, versioned graph of steps whose transitions route on outcomes. Steps always exit, and a run waits only at wait steps that triggers resume. Every client can draw the graph before it runs, and the engine executes only this form.

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
- **THEN** the resulting IR contains the same steps, profiles, outcomes, and transitions as the YAML, and nothing else

#### Scenario: Malformed YAML
- **WHEN** a YAML file cannot be parsed or does not match the schema
- **THEN** loading fails with the file, line, and a description of the problem

### Requirement: Definition identity
A workflow definition SHALL have a digest computed from its canonical IR after every sub-workflow reference has been resolved and recorded by digest. Two definitions with the same digest SHALL behave identically, and a definition SHALL NOT change once it has a digest.

#### Scenario: Same content, same digest
- **WHEN** the same workflow is loaded twice from different files with identical content
- **THEN** both loads produce the same digest

#### Scenario: Sub-workflow pinned
- **WHEN** a workflow references a sub-workflow by relative path
- **THEN** the definition records the digest of the sub-workflow and does not depend on the file afterward

### Requirement: Workflow inputs
A workflow SHALL declare its inputs, each with a name, a type, whether it is required, and an optional default. A default SHALL be a literal or a reference to run metadata. The engine SHALL validate the inputs of every run against these declarations before the run starts.

#### Scenario: Missing required input
- **WHEN** a run is started without a required input that has no default
- **THEN** the start is rejected and names the input

#### Scenario: Default from run metadata
- **WHEN** an input has a default that references the run directory and the run is started without that input
- **THEN** the input takes the run's directory value

### Requirement: Steps and outcomes
A workflow SHALL be a set of steps, each with a unique id, a reference to the plugin step it runs, its inputs, and a map from each outcome to a transition target. A transition target SHALL be another step id, a terminal state, or a core target.

#### Scenario: Outcome routing
- **WHEN** a step resolves with an outcome that has a transition target
- **THEN** the run moves to that target

#### Scenario: Unrouted outcome
- **WHEN** a step resolves with an outcome that has no transition target and no default applies
- **THEN** the run moves to needs-human and an attention request describes the unrouted outcome

### Requirement: Steps exit and wait steps wait
Every step other than a wait step SHALL do its work and exit with an outcome, and an exit that asks questions or reports failure SHALL be an outcome like any other. A run SHALL wait only at a wait step, and no process, plugin step, or agent session SHALL be held for a run while it waits.

#### Scenario: Waiting holds nothing
- **WHEN** a run is at a wait step
- **THEN** the run is recorded as waiting and nothing is running on its behalf except any trigger subscriptions the wait recorded

### Requirement: Step kinds
A step SHALL be exactly one of four kinds: a plugin step that runs a step provided by a plugin, an agent step that prompts a fresh session on a named profile, a wait step that waits for one of several triggers, or a workflow step that runs a sub-workflow. The set of kinds SHALL NOT be extended by plugins.

#### Scenario: Agent step
- **WHEN** a step is an agent step
- **THEN** it names a profile, a prompt, the outcomes the agent may report, and the outputs the agent must return

#### Scenario: Wait step
- **WHEN** a step is a wait step
- **THEN** it lists the triggers that can resume it, and its outcomes are the outcomes those triggers classify events into

#### Scenario: Two kinds in one step
- **WHEN** a step declares both a plugin step and a profile
- **THEN** validation fails and states that a step has exactly one kind

### Requirement: Wait steps and triggers
A wait step SHALL list one or more triggers, each from a plugin or one of the built-in triggers, the operator answer and the completion of an interactive agent, with configuration whose references are resolved when the step is entered. The first event any listed trigger delivers SHALL resolve the step, with the outcome the trigger classified the event into and the outputs it extracted. A wait step MAY declare a timeout, which resolves it with the outcome timed_out.

#### Scenario: Operator gate
- **WHEN** a wait step lists the built-in operator answer with a prompt and outcomes
- **THEN** the step resolves with the outcome the operator chooses and exposes any free text as the output text

#### Scenario: First trigger wins
- **WHEN** a wait step lists a review trigger and a schedule trigger and the review event arrives first
- **THEN** the step resolves from the review and the schedule subscription is removed

#### Scenario: Trigger cannot be used in a wait
- **WHEN** a wait step lists a trigger whose manifest does not declare wait support
- **THEN** validation fails and names the trigger

#### Scenario: Unrouted wait outcome
- **WHEN** a trigger in a wait step can classify events into an outcome the step does not route
- **THEN** validation fails and names the outcome

### Requirement: Start triggers
A workflow MAY declare start triggers, each from a plugin with literal configuration and a mapping from the trigger payload to the workflow's inputs. References in the mapping SHALL be limited to the trigger payload. Validation SHALL fail when a required input without a default is not supplied by the mapping.

#### Scenario: Mapping supplies inputs
- **WHEN** a trigger maps the issue title and body to the prompt input
- **THEN** a run started by the trigger has the prompt input set from the payload

#### Scenario: Required input not supplied
- **WHEN** a workflow has a required input that no trigger mapping supplies and that has no default
- **THEN** validation fails and names the trigger and the input

### Requirement: Plugin declarations
A workflow SHALL declare the plugins it uses by name, each with a source, a version constraint, configuration, and the names of the environment variables the plugin may receive. References to plugin steps, triggers, and constants SHALL use these names.

#### Scenario: Undeclared plugin
- **WHEN** a step refers to a plugin the workflow did not declare
- **THEN** validation fails and names the plugin

#### Scenario: Secrets named
- **WHEN** a plugin declaration names one environment variable
- **THEN** the plugin receives that variable and no other from the workflow's configuration

### Requirement: Agent steps start fresh sessions
Every agent step SHALL start a new agent session that has no conversation from any earlier step, and the session SHALL end when the step exits. Information SHALL reach an agent step only through its prompt, its declared context, the workspace, and route values.

#### Scenario: Same profile, fresh conversation
- **WHEN** two agent steps name the same profile
- **THEN** the second step's agent has none of the first step's conversation

#### Scenario: Prompt carries the input
- **WHEN** an agent step's prompt references an earlier step's output
- **THEN** the resolved value is part of the prompt the agent receives

### Requirement: Agent profiles
A workflow SHALL declare agent profiles by name, each with a harness plugin, a mode of headless or interactive, a working directory, and a permission policy that the harness plugin interprets. Agent steps SHALL name the profile they run on.

#### Scenario: Mixed harnesses
- **WHEN** one workflow declares profiles on two different harness plugins
- **THEN** each step runs on the harness of the profile it names

#### Scenario: Working directory
- **WHEN** a profile declares a working directory
- **THEN** every session started on that profile runs in that directory

#### Scenario: Working directory from a step output
- **WHEN** a profile's working directory references the output of an earlier step that created a checkout
- **THEN** the reference is resolved when each agent step starts and the session runs there

### Requirement: Agent step outputs
An agent step MAY declare output fields, each with a name and a type. Every declared field SHALL be required for every outcome except questions, and optional for questions. Every agent step SHALL also accept the standard outputs summary and notes. The engine SHALL validate the outputs an agent reports before it resolves the step.

#### Scenario: Outputs flow downstream
- **WHEN** an exploration step declares an output named plan and the agent reports it
- **THEN** a later step can reference it as the step's output

#### Scenario: Missing required output
- **WHEN** an agent reports plan_ready without a declared required output
- **THEN** the report is rejected, the step stays active, and the agent is told which output is missing

### Requirement: Agent step context
An agent step MAY declare a list of named references as context. The engine SHALL render each as a labelled section after the prompt and SHALL make the same sections visible to clients.

#### Scenario: Context rendered
- **WHEN** a step lists the plan and the review comments as context
- **THEN** the agent receives a labelled section for each after the prompt

### Requirement: Built-in outcomes
Every agent step SHALL be able to report the built-in outcomes questions and failed in addition to its declared outcomes and every plugin step SHALL be able to report the built-in outcome needs_help. The built-in outcome questions SHALL be routed by the core ask target unless the workflow routes it, and needs_help SHALL be routed by the needs-human target unless the workflow routes it.

#### Scenario: Questions need no routing
- **WHEN** an agent step declares one outcome and routes only that outcome
- **THEN** validation succeeds and the step's questions outcome follows the core ask target

#### Scenario: Questions overridden
- **WHEN** a step routes questions to the failed terminal state
- **THEN** an exit with questions ends the run in failed instead of asking

### Requirement: Launched outcome
An agent step on an interactive profile SHALL report the built-in outcome launched when it hands off to its host, and the workflow SHALL route it to a wait step.

#### Scenario: Launched routed to a wait
- **WHEN** an interactive agent step reports launched and the workflow routes it to a wait step on the agent's completion
- **THEN** the run rests at the wait step

#### Scenario: Launched not routed to a wait
- **WHEN** a workflow routes launched to a step that is not a wait step
- **THEN** validation fails and names the step

### Requirement: Default route
A step MAY declare a default transition target that applies to every outcome the step can report that has no transition of its own, except questions. A workflow MAY declare defaults by outcome name that apply to every step that can report that outcome and neither routes it nor declares a default. Validation SHALL treat outcomes covered by a default as routed.

#### Scenario: Default covers the rest
- **WHEN** a step routes approved explicitly and declares a default of needs-human
- **THEN** every other outcome the step can report, except questions, moves the run to needs-human

#### Scenario: Default does not swallow questions
- **WHEN** an agent step declares a default of needs-human and reports questions
- **THEN** the run follows the core ask target and not the default

#### Scenario: Workflow default by outcome name
- **WHEN** a workflow declares a default for the outcome failed and a step reports failed without routing it
- **THEN** the run follows the workflow default

### Requirement: Values passed along a transition
A transition MAY pass named values to its target step, each given as a reference or literal. The target step SHALL be able to read them under the route namespace, and validation SHALL fail if a step reads a route value that some incoming transition does not pass. The core ask target SHALL pass the answers and the previous attempt to the step it re-enters.

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

### Requirement: Ask routing target
The ask target SHALL open an attention request holding the questions the step reported, deliver it to the workflow's attention channels and to connected clients, and wait for the answers. It SHALL then run the asking step again as a fresh execution with the answers and the outputs of the previous attempt available to it.

#### Scenario: Questions with draft work
- **WHEN** an exploration step exits with two questions and a draft plan
- **THEN** an attention request holds the questions with the draft plan visible, and the answers start a fresh execution of the step that receives the answers and the draft plan

#### Scenario: Ask loop bound
- **WHEN** a step has been re-entered through the ask target as many times as its maximum
- **THEN** the run moves to needs-human with the iteration count in the request

### Requirement: Attention channels
A workflow MAY name attention channels, each from a plugin with configuration. The engine SHALL deliver every attention request of the run to each channel, and an answer reported by a channel SHALL have the same effect as an answer from a client.

#### Scenario: Channel configured
- **WHEN** a workflow names a channel and a step exits with questions
- **THEN** the request is delivered to the channel and to connected clients

### Requirement: Loops with bounds
A transition MAY target an earlier step or the same step, forming a loop. Every step that can be re-entered SHALL declare a maximum number of iterations, and the engine SHALL enforce it. A transition MAY list steps whose iteration counts are reset to zero when the transition is taken.

#### Scenario: Loop within bound
- **WHEN** a step routes back to itself and has run fewer times than its maximum
- **THEN** it runs again and the iteration count increases

#### Scenario: Loop exceeds bound
- **WHEN** a step would run more times than its maximum
- **THEN** the run moves to needs-human with the iteration count in the request instead of running the step

#### Scenario: Unbounded loop rejected
- **WHEN** a workflow contains a cycle in which no step declares a maximum
- **THEN** validation fails and names the cycle

#### Scenario: Restart resets inner loops
- **WHEN** a transition that returns the run to an early step lists the steps of the inner loops
- **THEN** their iteration counts are zero when they next start

#### Scenario: Unknown reset target
- **WHEN** a transition lists a step id that does not exist
- **THEN** validation fails and names the step id

### Requirement: No expressions
The definition language SHALL support only references to values and routing on outcomes. It SHALL NOT support arithmetic, comparison, conditionals, or user-defined functions.

#### Scenario: Expression rejected
- **WHEN** a value contains an expression such as a comparison or a function call
- **THEN** validation fails and states that logic belongs in a plugin step

### Requirement: References
A value MAY contain references in the form of a dollar sign followed by double braces around a path. The paths available SHALL be the trigger payload, workflow inputs, the outputs of earlier steps, the artifacts of earlier steps, route values, constants published by plugins, and run metadata, which includes the run id, the namespace, and the run directory. A reference to a step output SHALL resolve to the output of that step's most recent resolution.

#### Scenario: Reference to an earlier step output
- **WHEN** a step input references the output of a step that can precede it on every path
- **THEN** the value is resolved from that output when the step starts

#### Scenario: Output after looping back
- **WHEN** a step has resolved twice because the run looped back and a later step references its output
- **THEN** the reference resolves to the output of the second resolution, and the first remains in the event log

#### Scenario: Reference to an unavailable value
- **WHEN** a reference names a step that cannot have run before the referencing step on some path
- **THEN** validation fails and names the reference and the path

### Requirement: Sub-workflows
A step MAY run another workflow as a sub-workflow, passing inputs and receiving an outcome and outputs. A sub-workflow MAY declare profiles it requires, and the parent SHALL bind each to one of its own profiles. The parent run SHALL show the sub-workflow's current step as detail beneath the parent step. A sub-workflow SHALL be named statically and SHALL NOT be selected by a value at run time.

#### Scenario: Sub-workflow outcome
- **WHEN** a sub-workflow ends in a terminal state
- **THEN** the parent step resolves with the outcome succeeded, failed, or cancelled matching that state

#### Scenario: Profile binding
- **WHEN** a sub-workflow requires a profile named author and the parent binds it to its own profile
- **THEN** agent steps in the sub-workflow run on the parent's profile

#### Scenario: Dynamic selection rejected
- **WHEN** a step names its sub-workflow with a reference
- **THEN** validation fails and states that sub-workflows are named statically

### Requirement: Cleanup
A workflow MAY name a cleanup sub-workflow with inputs. The engine SHALL run it once when the run reaches a final state, including when the run is cancelled, and its outcome SHALL NOT change the run's final state.

#### Scenario: Cleanup after failure
- **WHEN** a run ends in failed and the workflow names a cleanup sub-workflow
- **THEN** the cleanup runs once and the run stays failed

### Requirement: Expected artifacts
A workflow or a step MAY declare the artifacts it is expected to produce, each with a kind and a name. The engine SHALL display expected artifacts as not yet available until they are reported or detected.

#### Scenario: Step-level declaration
- **WHEN** a step declares an expected artifact of kind pull request
- **THEN** the run shows that artifact as expected from the moment the step starts

#### Scenario: Workflow-level declaration
- **WHEN** a workflow declares an expected artifact
- **THEN** the run shows it as expected from the moment the run starts

### Requirement: Static validation
The platform SHALL validate a workflow without running it. Validation SHALL check that the schema is satisfied, step ids are unique, the start step exists, every transition target exists, every outcome a step's plugin declares is routed, every step is reachable, every referenced plugin, step, and trigger exists with valid inputs, and that no workflow selects a store.

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
The platform SHALL be able to return the full graph of a workflow, including every step, transition, wait, and expected artifact, without starting a run. The graph SHALL show which steps use the built-in ask and needs-human routes without drawing an edge for each.

#### Scenario: Drawing before running
- **WHEN** a client asks for a workflow's graph
- **THEN** it receives all steps and transitions

#### Scenario: Built-in routes
- **WHEN** an agent step relies on the core ask target
- **THEN** the graph marks the step as asking and does not add an edge
