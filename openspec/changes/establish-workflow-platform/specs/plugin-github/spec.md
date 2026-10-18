# Spec Delta

## Purpose

Provides everything workflows need from GitHub as a plugin: triggers from issues and pull requests, steps that open and update pull requests and wait for reviews and CI, and pull request artifacts. The engine knows nothing of GitHub.

## ADDED Requirements

### Requirement: Authenticated identity
The plugin SHALL act as whichever GitHub identity its environment provides, SHALL report that identity in its status, and SHALL NOT try to obtain broader permissions than it was given. When an action is denied it SHALL request operator attention instead of retrying differently.

#### Scenario: Permission denied
- **WHEN** a step is denied by GitHub for lack of permission
- **THEN** the step requests attention naming the action and the identity, and does not try another route

### Requirement: Issue and pull request triggers
The plugin SHALL provide triggers for an issue being opened, an issue being labeled, a pull request review being submitted, and a comment being added to a pull request. Each event SHALL carry a payload with the repository, number, author, and relevant text, and a dedupe key.

#### Scenario: Issue opened
- **WHEN** an issue is opened in a watched repository
- **THEN** the plugin reports a trigger event with the issue's repository, number, title, and body

#### Scenario: No duplicate after restart
- **WHEN** the engine restarts and the plugin polls again
- **THEN** events already reported are not reported again

### Requirement: Polling and webhooks
The plugin SHALL work by polling with no inbound network access, and SHALL also accept webhook deliveries through the engine when one is available, using the same trigger events.

#### Scenario: Polling only
- **WHEN** the engine has no reachable HTTP endpoint
- **THEN** triggers and waits still work by polling

#### Scenario: Webhook available
- **WHEN** a webhook delivery arrives for a watched event
- **THEN** the event is reported without waiting for the next poll

### Requirement: Open a pull request
The plugin SHALL provide a step that opens a pull request for a branch with a title and description, either as a draft or ready for review, and reports the pull request as an artifact. The step SHALL report the outcome opened when it opens a pull request, and reused when an open pull request for the branch already exists and is reused.

#### Scenario: Draft opened
- **WHEN** the step runs for a pushed branch with the draft option
- **THEN** a draft pull request exists, the outcome is opened, and the artifact is available with its link

#### Scenario: Already open
- **WHEN** an open pull request for the branch exists
- **THEN** the step reports the outcome reused with it as the artifact and does not open another

### Requirement: Mark ready
The plugin SHALL provide a step that marks a draft pull request ready for review.

#### Scenario: Marked ready
- **WHEN** the step runs for a draft pull request
- **THEN** the pull request is ready for review and the step reports done

### Requirement: Wait for review
The plugin SHALL provide a step that stays pending until a pull request has an approval, a request for changes, or is closed, and reports the outcome approved, changes_requested, or closed. The step SHALL include the review text and comments in its outputs under the name comments.

#### Scenario: Approved
- **WHEN** a required reviewer approves
- **THEN** the step reports approved

#### Scenario: Changes requested
- **WHEN** a reviewer requests changes
- **THEN** the step reports changes_requested with the review text and comments as outputs

#### Scenario: New push resets
- **WHEN** a review step is entered again after new commits
- **THEN** only reviews made after the previous resolution count

### Requirement: Check status
The plugin SHALL provide a step that reports the state of a pull request's required checks as green, red, or pending, and MAY wait until they are no longer pending. For red checks it SHALL include the failing check names and links in its outputs under the name failures.

#### Scenario: Red
- **WHEN** a required check has failed
- **THEN** the step reports red with the failing check names

### Requirement: Pull request state
The plugin SHALL provide a step that reports open_approved_green when a pull request is open, approved, and has green required checks, and otherwise reports not_ready with each unmet condition in its outputs.

#### Scenario: Not approved
- **WHEN** a pull request is open and green but not approved
- **THEN** the step reports not_ready and lists approval as unmet

### Requirement: Review comments resolved
The plugin SHALL provide a step that reports whether a pull request has unresolved review conversations.

#### Scenario: Unresolved
- **WHEN** a pull request has an unresolved conversation
- **THEN** the step reports unresolved with the conversations as outputs

### Requirement: Stacked pull requests
The plugin SHALL provide steps to create a stack of pull requests, add a pull request below or above another in a stack, and report whether a stack is linked in the order given. The create and add steps SHALL report linked on success. It SHALL use the GitHub CLI stack extension and SHALL report a clear outcome when it is not available.

#### Scenario: Stack linked
- **WHEN** three pull requests are linked bottom to top
- **THEN** the verify step reports linked

#### Scenario: Added to a stack
- **WHEN** the add step runs for a pull request and the pull request it goes above
- **THEN** the step reports linked

#### Scenario: Extension missing
- **WHEN** the stack extension is not installed
- **THEN** the step reports the outcome unavailable with an installation hint

### Requirement: Pull request artifact kind
The plugin SHALL declare a pull request artifact kind whose entries carry the link, a reference usable by other steps, the number, the branch, the title, draft state, review state, and check state, and SHALL support probing it.

#### Scenario: Probe
- **WHEN** the engine probes a pull request artifact
- **THEN** the plugin returns its current draft, review, and check state

### Requirement: Merge
The plugin SHALL provide a step that merges a pull request or a stack only when it is approved and its checks are green, and otherwise reports the blocking reason without merging.

#### Scenario: Not approved
- **WHEN** the step runs for a pull request that is not approved
- **THEN** nothing is merged and the step reports blocked with the reason
