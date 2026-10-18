# Spec Delta

## Purpose

Defines the contract every agent harness plugin implements, so a workflow can use Claude Code, Codex, Cursor, OpenCode, or any other agent, and mix them, while operators see the same step, attention, and artifact information regardless of harness.

## ADDED Requirements

### Requirement: Harness is a plugin capability
A harness SHALL be provided by a plugin that declares the harness capability in its manifest. The engine SHALL contain no knowledge of any specific harness.

#### Scenario: New harness added without engine change
- **WHEN** a plugin that declares the harness capability is installed
- **THEN** workflows can name it in a session without any change to the engine

### Requirement: Session lifecycle
A harness plugin SHALL support opening a session, sending it a prompt, receiving its events, and closing it. A session SHALL keep its conversation between prompts until it is closed.

#### Scenario: Open and prompt
- **WHEN** the engine opens a session and sends a prompt
- **THEN** the harness runs the agent with that prompt and reports events until the turn ends

#### Scenario: Close
- **WHEN** the engine closes a session
- **THEN** the harness ends the agent and releases its resources

### Requirement: Headless mode is required
Every harness plugin SHALL support headless mode, in which the agent runs without a terminal for a person and all questions and approvals are delivered to the engine as events.

#### Scenario: Headless run
- **WHEN** a session is opened in headless mode
- **THEN** the agent runs to the end of its turn without needing a terminal

### Requirement: Interactive mode is optional
A harness plugin MAY support interactive mode, in which the agent's own terminal interface runs in a pseudo-terminal that clients can attach to. A plugin that does not support it SHALL declare that in its manifest.

#### Scenario: Unsupported mode requested
- **WHEN** a workflow declares interactive mode for a harness that does not support it
- **THEN** validation fails and names the harness and the session

#### Scenario: Attach
- **WHEN** a client attaches to an interactive session
- **THEN** it receives the terminal output and can send input

### Requirement: Native attention becomes engine attention
A harness plugin SHALL translate the harness's native questions, plan approvals, and permission requests into attention events carrying a kind, a prompt, and any choices, and SHALL apply the engine's answer as the harness's native response.

#### Scenario: Permission request
- **WHEN** the agent asks permission to run a command
- **THEN** the plugin raises an attention event of kind approval with the command, and the answer is applied as allow or deny

#### Scenario: Agent question
- **WHEN** the agent asks the operator a question with choices
- **THEN** the plugin raises an attention event of kind question with the same choices, and the chosen answer is returned to the agent

### Requirement: Injected engine tools
The engine SHALL make three tools available to every session: one to complete the current step with an outcome and summary, one to ask the operator one or more questions, and one to report an artifact. Harness plugins SHALL expose them to the agent using the harness's own tool mechanism.

#### Scenario: Agent completes the step
- **WHEN** the agent calls the complete tool with a declared outcome
- **THEN** the engine resolves the step with that outcome and summary

#### Scenario: Summary is an output
- **WHEN** the agent calls the complete tool with a summary
- **THEN** the step's outputs contain that summary under the name summary

#### Scenario: Undeclared outcome
- **WHEN** the agent calls the complete tool with an outcome the step does not declare
- **THEN** the tool returns an error listing the allowed outcomes and the step stays active

#### Scenario: Agent asks the operator
- **WHEN** the agent calls the ask tool
- **THEN** the engine opens an attention request and the tool returns the operator's answer

#### Scenario: Agent reports an artifact
- **WHEN** the agent calls the artifact tool with a kind, name, and link
- **THEN** the engine adds it to the run's registry

### Requirement: Step prompt construction
An agent step SHALL declare a prompt and MAY declare a command prefix, such as a skill or slash command, to be placed before the prompt. The engine SHALL append instructions that name the outcomes the step allows and tell the agent to report the result with the complete tool.

#### Scenario: Prefix is applied
- **WHEN** a step declares a prefix and a prompt
- **THEN** the harness receives the prefix followed by the prompt

### Requirement: Turn end without completion
When an agent's turn ends without the agent having completed the step, the engine SHALL apply the step's declared policy, which SHALL be one of: re-prompt the agent, wait for the operator, or resolve with a named outcome.

#### Scenario: Re-prompt
- **WHEN** the turn ends without completion and the policy is re-prompt
- **THEN** the engine sends the agent a reminder to complete the step with one of the allowed outcomes

#### Scenario: Re-prompt limit
- **WHEN** the agent has been re-prompted the maximum number of times without completing
- **THEN** the run moves to needs-human

### Requirement: Session continuity across steps
Steps that name the same session SHALL continue the same conversation. A session declared fresh SHALL start a new conversation for every step.

#### Scenario: Context kept
- **WHEN** a later step uses the same non-fresh session
- **THEN** the agent still has the earlier conversation

### Requirement: Mixed harnesses in one run
A single run SHALL be able to use sessions on different harness plugins at once. Outcomes, attention, and artifacts SHALL be handled identically regardless of harness.

#### Scenario: Two harnesses
- **WHEN** one step runs on one harness and a later step runs on another
- **THEN** both report outcomes and attention to the engine in the same form

### Requirement: Harness activity summary
A harness plugin SHALL report a short human-readable activity status, such as working, waiting for the operator, or idle. Clients SHALL show this status and SHALL NOT be required to show the agent's output.

#### Scenario: Status shown
- **WHEN** the agent is running tools
- **THEN** the session status reads working

### Requirement: Session failure
When an agent process exits unexpectedly or the harness reports an error, the plugin SHALL report session failure with the reason. The engine SHALL treat the active step as failed and apply the step's routing for failure.

#### Scenario: Agent crashes
- **WHEN** the agent process exits during a step
- **THEN** the step resolves with the outcome failed and the reason is recorded
