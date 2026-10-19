# Spec Delta

## Purpose

Defines the reference workflow for the OpenSpec process as the platform's conformance bar. If the platform can run this workflow end to end with only plugins, the plugin, harness, trigger, and client designs are sufficient for a real process, including one that rests for days between a person's actions.

## ADDED Requirements

### Requirement: Reference workflow ships in the repository
The repository SHALL contain the OpenSpec reference workflow as a YAML file that passes the validate command, and that uses only plugin steps and triggers declared by the built-in plugins.

#### Scenario: Validates
- **WHEN** the validate command is run on the reference workflow
- **THEN** it succeeds with no errors

### Requirement: Starts from a prompt or an issue
The reference workflow SHALL start from an operator prompt and a repository, and SHALL also be startable from a GitHub issue being opened or labeled, with the issue's title and body used as the prompt. It SHALL NOT require the change name as an input.

#### Scenario: Prompt start
- **WHEN** an operator starts the workflow with a prompt
- **THEN** the exploration step receives that prompt

#### Scenario: Issue start
- **WHEN** a watched issue is opened
- **THEN** a run starts with the issue title and body as the prompt

### Requirement: Every agent step is fresh
Every agent step of the reference workflow SHALL start a fresh session and SHALL receive what it needs from the prompt, its context, the branch contents, and route values. The handoffs between steps SHALL be the repository files for the OpenSpec change, the exploration step's structured plan and change name, and the review and check results passed along transitions.

#### Scenario: No conversation carried
- **WHEN** the fix step runs after the proposal step
- **THEN** its agent has no conversation from the proposal step and works from the branch and the failures it was given

### Requirement: Exploration until the plan is approved
The workflow SHALL begin with an exploration step that uses the OpenSpec explore command on a profile whose permissions do not allow editing files, and that outputs the plan and the change name. The exploration step SHALL be followed by a plan gate, which is a wait step on the operator's answer. Feedback SHALL return to exploration with the feedback text and the previous plan. Approval SHALL continue to the proposal step.

#### Scenario: Plan is an output
- **WHEN** exploration exits with plan_ready
- **THEN** the plan and the change name are outputs and the plan gate shows the plan

#### Scenario: Feedback keeps exploring
- **WHEN** the operator answers the plan gate with feedback
- **THEN** the run returns to exploration and the feedback and the previous plan are given to the new session

#### Scenario: Approval proceeds
- **WHEN** the operator approves the plan
- **THEN** the run moves to the branch creation step

### Requirement: Questions need no routing
The reference workflow SHALL NOT route the questions outcome of any agent step, and SHALL rely on the core ask route. Questions SHALL be delivered to the operator's clients and to the channels the workflow names.

#### Scenario: Exploration asks
- **WHEN** exploration exits with questions and a draft plan
- **THEN** the operator is shown the questions with the draft, and after the answers exploration starts again with them and the draft

### Requirement: Branch and workspace
After the plan is approved the workflow SHALL create a branch named from the change name and a checkout on it, and every later agent step that works on the change SHALL run in that checkout.

#### Scenario: Branch created
- **WHEN** the plan is approved with the change name add-widgets
- **THEN** a branch and checkout for add-widgets exist and later agent steps run in the checkout

#### Scenario: Branch already exists
- **WHEN** the branch exists from an earlier attempt
- **THEN** the checkout reuses it

### Requirement: Proposal and draft pull request
After the branch exists the workflow SHALL run the proposal step, commit and push the OpenSpec files, open a draft pull request whose title starts with Proposal, wait on the checks trigger for the checks to finish, and then mark it ready for review when they are green and the OpenSpec files are valid.

#### Scenario: Draft opened
- **WHEN** the proposal step completes
- **THEN** a draft proposal pull request exists and appears as an available artifact

#### Scenario: Checks red
- **WHEN** the checks on the proposal pull request are red
- **THEN** the workflow runs a fix step in a fresh session that receives the failures, pushes, and waits on the checks again, within its bound

### Requirement: Human review of the proposal
The workflow SHALL wait at a wait step on the review trigger for the proposal pull request, with a closed outcome and no deadline. Changes requested SHALL return to exploration with the review comments and reset the counts of the inner loops. Approval SHALL continue.

#### Scenario: Rests for the weekend
- **WHEN** the proposal pull request opens on a Friday and no one reviews it until Monday
- **THEN** the run is waiting with no process running and resumes on the review

#### Scenario: Changes requested
- **WHEN** a reviewer requests changes on the proposal pull request
- **THEN** the run returns to exploration with the review text and the inner loop counts are zero

#### Scenario: Approved
- **WHEN** the proposal pull request is approved
- **THEN** the run continues to the invariant check

#### Scenario: Closed
- **WHEN** the proposal pull request is closed without approval
- **THEN** the run ends cancelled

### Requirement: Invariants before implementation
Before implementation the workflow SHALL check that the work is in a worktree on a non-default branch, that a valid change exists, and that the proposal pull request is open, approved, and green. If any invariant fails the run SHALL move to needs-human naming the failed invariant.

#### Scenario: Invariant holds
- **WHEN** all invariants hold
- **THEN** the workflow proceeds to apply

#### Scenario: Invariant fails
- **WHEN** the proposal pull request is not approved
- **THEN** the run moves to needs-attention naming the failed invariant and does not start apply

### Requirement: Apply until no tasks remain
The workflow SHALL repeat the OpenSpec apply step in a fresh session each pass, committing and pushing after each pass, until the task progress step reports complete.

#### Scenario: Looping
- **WHEN** a pass leaves tasks unchecked
- **THEN** the workflow runs apply again in a fresh session that finds the checked tasks on the branch

#### Scenario: Done
- **WHEN** no tasks remain
- **THEN** the workflow moves on to verification

### Requirement: Verify until clean
The workflow SHALL repeat the OpenSpec verify step, fixing and pushing after each pass with issues, until a pass reports no issues.

#### Scenario: Issues found
- **WHEN** verification reports issues
- **THEN** the workflow fixes them in a fresh session and verifies again

### Requirement: Adversarial review until no actionable findings
The workflow SHALL run an adversarial review on a profile that MAY use a different harness than the implementing profile, and SHALL fix findings in a fresh session and review again until a pass reports no actionable findings.

#### Scenario: Different harness
- **WHEN** the review profile names a different harness plugin
- **THEN** the review runs on that harness

#### Scenario: Findings fixed
- **WHEN** the review reports actionable findings
- **THEN** a fix step receives them, and a new review pass runs

### Requirement: Scope check
After review the workflow SHALL run a scope check in a fresh session that reports whether the work contains scope creep or missing requirements. A report of either SHALL move the run to needs-human with the findings.

#### Scenario: Scope creep
- **WHEN** the scope check reports a change not intended by the spec
- **THEN** the run moves to needs-attention with the findings

#### Scenario: Clean
- **WHEN** the scope check reports none
- **THEN** the workflow opens the implementation pull request

### Requirement: Implementation pull request and review
The workflow SHALL open an implementation pull request whose title starts with Implementation, linked as a stack above the proposal, and wait at a wait step on the review trigger. Changes requested SHALL return to exploration with the review text and reset the counts of the inner loops. Approval SHALL continue.

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

### Requirement: Checkout cleanup
The workflow SHALL name a cleanup sub-workflow that removes the checkout, so that the checkout is removed whether the run succeeds, fails, or is cancelled.

#### Scenario: Cancelled run
- **WHEN** the run is cancelled
- **THEN** the cleanup removes the checkout and the run stays cancelled

### Requirement: Artifacts shown throughout
The workflow SHALL declare the proposal, implementation, and archive pull requests and the OpenSpec change as expected artifacts so clients show them as pending from the start and as links when available.

#### Scenario: Pending from the start
- **WHEN** a run starts
- **THEN** clients show all three pull requests as expected

### Requirement: Permission limits respected
The workflow SHALL NOT include any step that tries to obtain permissions beyond the identity it runs as. Denied actions SHALL move the run to needs-human and no workaround SHALL be attempted.

#### Scenario: Cannot merge
- **WHEN** the identity lacks permission for an action
- **THEN** the step reports needs_help, the run moves to needs-attention, and no workaround is attempted

### Requirement: Conformance demonstration
The platform SHALL be considered conformant only when the reference workflow has been run end to end on a real repository using real harnesses, with recordings or screenshots from a terminal client and a web client showing the step, the waiting and needs-you states, and artifacts at several points, including a run that rested at a review with no engine process running and resumed from a delivered event.

#### Scenario: Recorded proof
- **WHEN** the conformance run completes
- **THEN** the recordings are attached to the pull request that completes the final task group
