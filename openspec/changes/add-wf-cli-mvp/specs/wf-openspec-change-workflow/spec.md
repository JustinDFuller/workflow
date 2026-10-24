# Spec Delta

## Purpose

Defines the OpenSpec process as the first workflow, with the check scripts that observe it and the recorded demonstration this MVP must produce, so that the idea is judged on a real process and not on a toy.

## ADDED Requirements

### Requirement: Workflow location and shape
The OpenSpec change process SHALL be one workflow file with its check scripts beside it, where every check is a small script that only reads the repository, GitHub, and OpenSpec. No check SHALL change the repository, a pull request, or a change.

#### Scenario: Files
- **WHEN** the workflow directory is listed
- **THEN** it holds the workflow file and one script for each check

#### Scenario: Valid
- **WHEN** the workflow file is validated
- **THEN** it is valid

#### Scenario: Read-only checks
- **WHEN** every check script is read
- **THEN** none of them commits, pushes, edits a pull request, or edits a change

### Requirement: Artifacts the workflow reports
The checks SHALL report these artifacts as soon as each exists: the change directory as change, the proposal pull request as proposal_pr, the implementation pull request as implementation_pr, the archive pull request as archive_pr, and the demonstration recording as recording.

#### Scenario: Change created
- **WHEN** the change-present check passes
- **THEN** it reports the change directory as the artifact change

#### Scenario: Pull request opened
- **WHEN** the proposal pull request check passes
- **THEN** it reports the pull request address as the artifact proposal_pr

#### Scenario: Visible while the run is still going
- **WHEN** the run is waiting on the proposal review
- **THEN** the board lists change and proposal_pr

### Requirement: Propose step
The workflow SHALL begin with a propose step whose job tells the agent to explore the idea with the person using the harness's question tool, write the OpenSpec change, commit and push it, open a draft pull request whose title starts with Proposal, make CI pass, and mark the pull request ready. Its checks SHALL run in this order: the change is present, the change is valid, the branch is pushed, the proposal pull request is open, CI is green, the pull request is ready.

#### Scenario: Nothing done yet
- **WHEN** the run starts on a branch with no change
- **THEN** the first check fails with a message saying no change is present

#### Scenario: Cheap checks first
- **WHEN** the change is not pushed
- **THEN** the CI check does not run

#### Scenario: CI still running
- **WHEN** the pull request is open and CI is running
- **THEN** the result is pending and names the running checks

### Requirement: Proposal review step
After the propose step the workflow SHALL wait for the proposal pull request review with no job. The wait SHALL need the person, SHALL last at most fourteen days, and SHALL report approved, changes_requested, or closed from reviews submitted after the step was entered. Approved SHALL go to the invariants step, changes_requested SHALL go to the revise step with the review comments, and closed SHALL stop the run.

#### Scenario: Approved
- **WHEN** a review approving the proposal pull request is submitted after the step was entered
- **THEN** the run moves to the invariants step

#### Scenario: Old review ignored
- **WHEN** an approval was submitted before the step was entered
- **THEN** it does not count and the result is pending

#### Scenario: Changes requested
- **WHEN** a review requests changes
- **THEN** the run moves to the revise step and shows the comments

#### Scenario: Closed
- **WHEN** the pull request is closed without merging
- **THEN** the run stops

### Requirement: Revise step
The revise step SHALL give the agent the review comments and the same checks as the propose step, and then SHALL return to the proposal review.

#### Scenario: Comments reach the agent
- **WHEN** the run enters the revise step after a request for changes
- **THEN** the comments are shown with the job

### Requirement: Invariants step
The invariants step SHALL have no job and SHALL check that the work is in a worktree on a branch other than the default branch, that the proposal pull request is approved, and that its checks are green. A failing invariant SHALL stop the run with a message that names the invariant.

#### Scenario: Invariant holds
- **WHEN** the worktree is not on the default branch and the pull request is approved and green
- **THEN** the run moves to the apply step

#### Scenario: Invariant fails
- **WHEN** CI has turned red since the approval
- **THEN** the run stops and the message names the failing invariant

### Requirement: Apply step
The apply step SHALL give the agent the tasks as its job and SHALL check that every task is marked complete, that the branch is pushed, and that CI is green.

#### Scenario: Tasks remain
- **WHEN** three tasks are not marked complete
- **THEN** the result is a failure that says three tasks remain

### Requirement: Headless judgment checks
The verify, review, and scope steps SHALL each have a check that runs a headless agent and maps its verdict to pass or fail. Each step SHALL be marked slow, and each check SHALL keep its result by commit.

#### Scenario: Same commit
- **WHEN** a headless check is evaluated twice on the same commit
- **THEN** the headless agent runs once

#### Scenario: Slow
- **WHEN** the run is on a verify, review, or scope step
- **THEN** the output tells the agent to run the next command in the background

### Requirement: Verify and review steps
The verify step SHALL check that the implementation matches the specs, and the review step SHALL run an adversarial review. A failure in either SHALL return the agent to its job with the findings.

#### Scenario: Findings reach the agent
- **WHEN** the review check fails with findings
- **THEN** the agent's job shows the findings

### Requirement: Scope step
The scope step SHALL check that nothing was built outside the approved specs, and a failure SHALL stop the run.

#### Scenario: Scope violation
- **WHEN** the scope check finds work outside the approved specs
- **THEN** the run stops and names what was found

### Requirement: Implementation pull request step
The implementation pull request step SHALL tell the agent to open a draft pull request titled Implementation, stacked above the proposal pull request, with proof in its description, and SHALL check that it exists, is stacked on the proposal, is pushed, and is green.

#### Scenario: Not stacked
- **WHEN** the implementation pull request has the default branch as its base
- **THEN** the check fails and says it must be stacked on the proposal

### Requirement: Implementation review step
After the implementation pull request step the workflow SHALL wait for its review in the same way as the proposal review. Approved SHALL go to the archive step, changes_requested SHALL go to a revise step that returns through verification, and closed SHALL stop the run.

#### Scenario: Approved
- **WHEN** the implementation pull request is approved
- **THEN** the run moves to the archive step

#### Scenario: Changes requested
- **WHEN** the implementation review requests changes
- **THEN** the agent revises, and the run passes through verify before returning to the review

### Requirement: Archive step
The archive step SHALL tell the agent to mark the implementation pull request ready, archive the change, and open a draft pull request titled Archive stacked above the implementation. It SHALL check that the change is archived, the pull request exists, and CI is green. The run SHALL then end.

#### Scenario: Archived
- **WHEN** the change is archived and its pull request is green
- **THEN** the run ends

### Requirement: Permission failures are broken
A check that cannot reach GitHub or a headless agent because of missing authentication or permission SHALL report broken, and the job text SHALL tell the agent to report that to the person and not to work around it.

#### Scenario: No access
- **WHEN** the GitHub command is denied
- **THEN** the result is broken and the state is broken

### Requirement: Built in two stages
The workflow SHALL be built in two stages. The first stage SHALL contain the propose, proposal review, revise, and invariants steps and SHALL be demonstrated before the second stage is built. The second stage SHALL add the remaining steps.

#### Scenario: Stage one proves the idea
- **WHEN** the first stage is demonstrated
- **THEN** a real session has been kept on its workflow from an idea to an approved proposal

### Requirement: Recorded demonstration
The MVP SHALL be accepted only with a recording of a real Claude Code session on a real repository showing: the agent being kept going by a failing check, a background wait resolving on a real review, the stop hook blocking an early stop, the tool hook denying a call, the status line changing with the step, and the runs and show commands describing the run. The recording SHALL be linked from the implementation pull request.

#### Scenario: Recording exists
- **WHEN** the implementation pull request is opened
- **THEN** its description links a recording that shows each listed behavior

#### Scenario: Missing behavior
- **WHEN** a listed behavior does not appear in the recording
- **THEN** the MVP is not accepted
