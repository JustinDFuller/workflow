# Spec Delta

## Purpose

Provides everything workflows need from GitHub as a plugin: triggers that start runs from issues and resume runs from reviews and checks, steps that prepare branches, open and update pull requests, and file issues, an attention channel for questions, and pull request and issue artifacts. The engine knows nothing of GitHub.

## ADDED Requirements

### Requirement: Authenticated identity
The plugin SHALL act as whichever GitHub identity its environment provides, SHALL report that identity in its status, and SHALL NOT try to obtain broader permissions than it was given. When an action is denied it SHALL report the outcome needs_help naming the action and the identity instead of retrying differently.

#### Scenario: Permission denied
- **WHEN** a step is denied by GitHub for lack of permission
- **THEN** the step reports needs_help naming the action and the identity, and does not try another route

### Requirement: Start triggers
The plugin SHALL provide start triggers for an issue being opened, an issue being labeled, and a comment being added to an issue or pull request. Each event SHALL carry a payload with the repository, number, author, and relevant text, and a dedupe key. A start trigger SHALL accept an optional filter on labels and on the author, so that events from some authors can be ignored.

#### Scenario: Issue opened
- **WHEN** an issue is opened in a watched repository
- **THEN** the plugin reports a trigger event with the issue's repository, number, title, and body

#### Scenario: Label filter
- **WHEN** the trigger is configured for a label and an issue is opened without it
- **THEN** no event is reported

#### Scenario: No duplicate after restart
- **WHEN** the engine restarts and the plugin polls again
- **THEN** events already reported are not reported again

### Requirement: Review trigger
The plugin SHALL provide a trigger for wait steps that resumes a run when a pull request receives a review, or is closed. It SHALL classify the event into the outcome approved, changes_requested, commented, or closed, and SHALL include the review text and comments in its outputs under the name comments. Its correlation key SHALL identify the repository and the pull request number, and it SHALL offer check-now.

#### Scenario: Approved
- **WHEN** a counted reviewer approves
- **THEN** the trigger resolves the wait with approved

#### Scenario: Changes requested
- **WHEN** a counted reviewer requests changes
- **THEN** the trigger resolves the wait with changes_requested and the review text and comments as outputs

#### Scenario: Review bot comments
- **WHEN** a reviewer that is a bot submits a review with comments and no verdict
- **THEN** the trigger resolves the wait with commented and the comments as outputs

#### Scenario: Ignored reviewer
- **WHEN** a reviewer on the ignore list reviews
- **THEN** the event is ignored and the wait continues

#### Scenario: Only reviews after the wait began
- **WHEN** a wait is entered again after new commits
- **THEN** only reviews made after the wait began count, and check-now ignores earlier ones

#### Scenario: Already approved
- **WHEN** the run reaches the wait after the pull request was approved
- **THEN** check-now resolves the wait with approved

#### Scenario: Run waits for days
- **WHEN** no review arrives for two weeks
- **THEN** no process is held for the run and the next review event resumes it

### Requirement: Review trigger reviewer filters
The review trigger SHALL accept optional lists of reviewers to count and to ignore, and a reviewer MAY be a person or a bot.

#### Scenario: Counted list
- **WHEN** the trigger lists the reviewers to count and another reviewer reviews
- **THEN** the event is ignored and the wait continues

### Requirement: Checks trigger
The plugin SHALL provide a trigger for wait steps that resumes a run when the required checks of a pull request are no longer pending, classifying the event into green or red, and including the failing check names and links in its outputs under the name failures. Its correlation key SHALL identify the pull request, and it SHALL offer check-now.

#### Scenario: Green
- **WHEN** the required checks finish green
- **THEN** the trigger resolves the wait with green

#### Scenario: Red
- **WHEN** a required check has failed
- **THEN** the trigger resolves the wait with red and the failing check names

#### Scenario: Already finished
- **WHEN** the checks finished before the run reached the wait
- **THEN** check-now resolves the wait from their result

### Requirement: Pull request comment trigger
The plugin SHALL provide a trigger for wait steps that resumes a run when a comment is added to a pull request or issue, with an optional filter on the author, classifying the event into the outcome commented with the text as output.

#### Scenario: Reply arrives
- **WHEN** a person comments on the watched pull request
- **THEN** the wait resolves with commented and the text

### Requirement: Polling and webhooks
The plugin SHALL work by polling with no inbound network access, and SHALL also accept webhook deliveries through the engine, and raw events delivered from outside, using the same trigger events.

#### Scenario: Polling only
- **WHEN** the engine has no reachable HTTP endpoint
- **THEN** triggers and waits still work by polling

#### Scenario: Webhook available
- **WHEN** a webhook delivery arrives for a watched event
- **THEN** the event is reported without waiting for the next poll

#### Scenario: Delivered from a job
- **WHEN** a continuous integration job delivers the payload of the event that started it
- **THEN** the plugin classifies it like a webhook and the matching run starts or resumes

### Requirement: Branch workspace
The plugin SHALL provide a step that creates a checkout of a repository on a new branch from a named base, inside the run directory or a given directory, and reports the path and branch as outputs. It SHALL provide a step that removes such a checkout. Creating a checkout whose branch already exists SHALL reuse it and report the outcome reused.

#### Scenario: Branch created
- **WHEN** the step runs for a repository and a new branch name
- **THEN** a checkout exists on that branch based on the base and the step reports created

#### Scenario: Branch exists
- **WHEN** the branch already exists
- **THEN** the step checks it out and reports reused

#### Scenario: Removed
- **WHEN** the remove step runs for a checkout
- **THEN** the directory is gone and the branch remains

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

### Requirement: Check status
The plugin SHALL provide a step that reports the state of a pull request's required checks as green, red, or pending right now, and does not wait. For red checks it SHALL include the failing check names and links in its outputs under the name failures.

#### Scenario: Red
- **WHEN** a required check has failed
- **THEN** the step reports red with the failing check names

#### Scenario: Pending
- **WHEN** checks are still running
- **THEN** the step reports pending and exits

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

### Requirement: Issue steps
The plugin SHALL provide a step that creates an issue with a title, body, and labels, and a step that adds a comment to an issue or pull request. The create step SHALL report the outcome created, or reused when an open issue with the same deduplication key already exists, and SHALL report the issue as an artifact.

#### Scenario: Issue created
- **WHEN** the step runs with a title and body
- **THEN** an issue exists and is an available artifact with its link

#### Scenario: Issue already exists
- **WHEN** an open issue was created earlier for the same deduplication key
- **THEN** the step reports reused with that issue and creates no other

#### Scenario: Comment added
- **WHEN** the comment step runs for a pull request
- **THEN** the comment appears on the pull request and the step reports done

### Requirement: Attention channel
The plugin SHALL provide an attention channel that posts an attention request as a comment on a named issue or pull request, and reports a later comment by a permitted author as the answer. The comment SHALL carry the run and request identity so replies are matched to the right request.

#### Scenario: Questions posted
- **WHEN** a run exits with questions and the workflow names this channel for a pull request
- **THEN** a comment on the pull request lists the questions

#### Scenario: Reply answers
- **WHEN** a permitted person replies to that comment
- **THEN** the channel reports the reply as the answer

#### Scenario: Reply from someone else
- **WHEN** an author who is not permitted replies
- **THEN** the reply is ignored

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

### Requirement: Artifact kinds
The plugin SHALL declare a pull request artifact kind whose entries carry the link, a reference usable by other steps, the number, the branch, the title, draft state, review state, and check state, and SHALL support probing it. It SHALL declare an issue artifact kind whose entries carry the link, reference, number, title, and state.

#### Scenario: Probe
- **WHEN** the engine probes a pull request artifact
- **THEN** the plugin returns its current draft, review, and check state

### Requirement: Merge
The plugin SHALL provide a step that merges a pull request or a stack only when it is approved and its checks are green, and otherwise reports the blocking reason without merging.

#### Scenario: Not approved
- **WHEN** the step runs for a pull request that is not approved
- **THEN** nothing is merged and the step reports blocked with the reason
