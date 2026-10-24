# Spec Delta

## Purpose

Defines the YAML file that describes a workflow as a static graph of steps, each with an optional job for the agent and checks that decide when it is over, and the validation that rejects a broken definition before a run starts.

## ADDED Requirements

### Requirement: Workflow file
A workflow SHALL be one YAML file with a name, an optional description, optional inputs, a start step, a library of named checks, and a map of steps. Unknown keys SHALL be rejected.

#### Scenario: Valid file loads
- **WHEN** a file with a name, a start step, a check library, and steps is loaded
- **THEN** the workflow loads with no errors

#### Scenario: Unknown key
- **WHEN** a step contains a key the format does not define
- **THEN** loading fails and names the key, the file, and the line

### Requirement: Inputs
A workflow SHALL declare its inputs by name, each marked required or given a default. Input values SHALL be strings.

#### Scenario: Required input missing
- **WHEN** a run is started without a required input
- **THEN** the start is refused and names the missing input

#### Scenario: Default used
- **WHEN** a run is started without an input that has a default
- **THEN** the run records the default as the input value

### Requirement: Steps
A step SHALL have an optional job, which is text written for the agent, and an ordered list of one or more named checks from the library. A step MAY declare a maximum number of visits, a timeout, a list of tools the agent may not use, a flag marking its checks as slow, and a flag marking that a pending step needs the person.

#### Scenario: Step with a job
- **WHEN** a step has a job and checks
- **THEN** the run shows the job to the agent while the checks do not pass

#### Scenario: Step with no job
- **WHEN** a step has checks and no job
- **THEN** the run has nothing for the agent to do on that step except wait for the checks

### Requirement: Check library
A workflow SHALL define each check once by name with a command, an optional timeout, an optional list of the outcomes it can report, and an optional cache setting. Steps SHALL refer to checks by name.

#### Scenario: Shared check
- **WHEN** two steps list the same check name
- **THEN** both use the same command and settings

#### Scenario: Unknown check
- **WHEN** a step lists a check that is not in the library
- **THEN** validation fails and names the step and the check

### Requirement: Routing by outcome
A step SHALL route in one of two ways. A step whose final check declares outcomes SHALL map every declared outcome to a target and SHALL NOT have a plain next target. Any other step SHALL have a plain next target. A target SHALL be a step name, the terminal end, or the terminal stop.

#### Scenario: Plain next
- **WHEN** every check on a step passes and the step has a plain next target
- **THEN** the run moves to that target

#### Scenario: Outcome routing
- **WHEN** the final check reports an outcome the step maps
- **THEN** the run moves to the target mapped to that outcome

#### Scenario: Unrouted outcome
- **WHEN** a check declares an outcome that its step does not map
- **THEN** validation fails and names the step and the outcome

#### Scenario: Both routings
- **WHEN** a step has a plain next target and a final check that declares outcomes
- **THEN** validation fails

### Requirement: Outcomes only on the final check
A check that declares outcomes SHALL be the last check of every step that lists it.

#### Scenario: Outcome check in the middle
- **WHEN** a step lists a check that declares outcomes before another check
- **THEN** validation fails and names the step and the check

### Requirement: Waits need a deadline
A step with no job SHALL declare a timeout. The timeout SHALL be the longest the run may spend on the step before it stops.

#### Scenario: Wait without a timeout
- **WHEN** a step has no job and no timeout
- **THEN** validation fails and names the step

### Requirement: Static validation
The validate command SHALL check a workflow without running it and report every problem with its file and line. It SHALL reject a missing start step, a target that is not a step or a terminal, a step that cannot be reached from the start, a workflow with no path to a terminal, durations that do not parse, and a maximum number of visits below one.

#### Scenario: Unreachable step
- **WHEN** a step is not reachable from the start step
- **THEN** validation fails and names the step

#### Scenario: No way out
- **WHEN** no path leads from the start step to end or stop
- **THEN** validation fails

#### Scenario: Several problems
- **WHEN** a workflow has more than one problem
- **THEN** validation reports all of them and exits with a failure

#### Scenario: Valid workflow
- **WHEN** a workflow has no problems
- **THEN** validation prints that it is valid and exits successfully
