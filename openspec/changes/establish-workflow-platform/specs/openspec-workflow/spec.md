# Spec Delta

## Purpose

Defines the reference workflow for the OpenSpec process as the platform's conformance bar. If the platform can run this workflow end to end with only plugins, the plugin, harness, and client designs are sufficient for a real process.

## ADDED Requirements

### Requirement: Reference workflow ships in the repository
The repository SHALL contain the OpenSpec reference workflow as a YAML file that passes the validate command, and that uses only plugin steps declared by the built-in plugins.

#### Scenario: Validates
- **WHEN** the validate command is run on the reference workflow
- **THEN** it succeeds with no errors

### Requirement: Starts from a prompt or an issue
The reference workflow SHALL start from an operator prompt, and SHALL also be startable from a GitHub issue being opened or labeled, with the issue's title and body used as the prompt.

#### Scenario: Prompt start
- **WHEN** an operator starts the workflow with a prompt
- **THEN** the exploration step receives that prompt

#### Scenario: Issue start
- **WHEN** a watched issue is opened
- **THEN** a run starts with the issue title and body as the prompt

### Requirement: Exploration until the plan is approved
The workflow SHALL begin with an exploration step that uses the OpenSpec explore command, followed by a human gate on the plan. Feedback SHALL return to exploration with the feedback text. Approval SHALL continue to the proposal step.

#### Scenario: Feedback keeps exploring
- **WHEN** the operator answers the plan gate with feedback
- **THEN** the run returns to exploration and the feedback is given to the agent

#### Scenario: Approval proceeds
- **WHEN** the operator approves the plan
- **THEN** the run moves to the proposal step

### Requirement: Proposal and draft pull request
After approval the workflow SHALL run the proposal step, commit and push the OpenSpec files, open a draft pull request whose title starts with Proposal, wait for green checks and valid OpenSpec files, and then mark it ready for review.

#### Scenario: Draft opened
- **WHEN** the proposal step completes
- **THEN** a draft proposal pull request exists and appears as an available artifact

#### Scenario: Checks red
- **WHEN** the checks on the proposal pull request are red
- **THEN** the workflow runs a fix step on the same session and checks again, within its bound

### Requirement: Human review of the proposal
The workflow SHALL wait for a human review of the proposal pull request. Changes requested SHALL return to exploration with the review comments. Approval SHALL continue.

#### Scenario: Changes requested
- **WHEN** a reviewer requests changes on the proposal pull request
- **THEN** the run returns to exploration with the review text

#### Scenario: Approved
- **WHEN** the proposal pull request is approved
- **THEN** the run continues to the invariant check

### Requirement: Invariants before implementation
Before implementation the workflow SHALL check that the work is in a worktree on a non-default branch, that a valid change exists, and that the proposal pull request is open, approved, and green. If any invariant fails the run SHALL stop and request operator help.

#### Scenario: Invariant holds
- **WHEN** all invariants hold
- **THEN** the workflow proceeds to apply

#### Scenario: Invariant fails
- **WHEN** the proposal pull request is not approved
- **THEN** the run moves to needs-attention naming the failed invariant and does not start apply

### Requirement: Apply until no tasks remain
The workflow SHALL repeat the OpenSpec apply step, committing and pushing after each pass and waiting for green checks, until the task progress step reports complete.

#### Scenario: Looping
- **WHEN** a pass leaves tasks unchecked
- **THEN** the workflow runs apply again

#### Scenario: Done
- **WHEN** no tasks remain
- **THEN** the workflow moves on to verification

### Requirement: Verify until clean
The workflow SHALL repeat the OpenSpec verify step, fixing and pushing after each pass with issues, until a pass reports no issues.

#### Scenario: Issues found
- **WHEN** verification reports issues
- **THEN** the workflow fixes them and verifies again

### Requirement: Adversarial review until no actionable findings
The workflow SHALL run an adversarial review in a fresh-context session, which MAY use a different harness than the implementing session, and SHALL fix findings and review again until a pass reports no actionable findings.

#### Scenario: Fresh context
- **WHEN** the adversarial review step starts
- **THEN** its session has none of the implementing session's conversation

#### Scenario: Findings fixed
- **WHEN** the review reports actionable findings
- **THEN** the implementing session fixes them and a new review pass runs

### Requirement: Scope check
After review the workflow SHALL run a scope check in a fresh-context session that reports whether the work contains scope creep or missing requirements. A report of either SHALL stop the run and request operator help.

#### Scenario: Scope creep
- **WHEN** the scope check reports a change not intended by the spec
- **THEN** the run moves to needs-attention with the findings

#### Scenario: Clean
- **WHEN** the scope check reports none
- **THEN** the workflow opens the implementation pull request

### Requirement: Implementation pull request and review
The workflow SHALL open an implementation pull request whose title starts with Implementation, linked as a stack above the proposal, and wait for human review. Changes requested SHALL return to exploration with the review text. Approval SHALL continue.

#### Scenario: Stacked
- **WHEN** the implementation pull request is opened
- **THEN** it is linked above the proposal pull request in a stack

#### Scenario: Feedback
- **WHEN** a reviewer requests changes
- **THEN** the run returns to exploration with the review text

### Requirement: Ready and archive
After approval the workflow SHALL mark the implementation pull request ready and open an archive pull request whose title starts with Archive, above the implementation in the stack, and then end succeeded.

#### Scenario: Archive PR opened
- **WHEN** the implementation pull request is ready
- **THEN** an archive pull request exists above it and the run ends succeeded

### Requirement: Artifacts shown throughout
The workflow SHALL declare the proposal, implementation, and archive pull requests and the OpenSpec change as expected artifacts so clients show them as pending from the start and as links when available.

#### Scenario: Pending from the start
- **WHEN** a run starts
- **THEN** clients show all three pull requests as expected

### Requirement: Permission limits respected
The workflow SHALL NOT include any step that tries to obtain permissions beyond the identity it runs as. Denied actions SHALL request operator attention.

#### Scenario: Cannot merge
- **WHEN** the identity lacks permission for an action
- **THEN** the run moves to needs-attention and no workaround is attempted

### Requirement: Conformance demonstration
The platform SHALL be considered conformant only when the reference workflow has been run end to end on a real repository using real harnesses, with recordings or screenshots from a terminal client and a web client showing the step, needs-you state, and artifacts at several points.

#### Scenario: Recorded proof
- **WHEN** the conformance run completes
- **THEN** the recordings are attached to the pull request that completes the final task group
