# Spec Delta

## Purpose

Defines the contract every agent harness plugin implements, so a workflow can use Claude Code, Codex, Cursor, OpenCode, or any other agent, and mix them, while operators see the same step, question, and artifact information regardless of harness. Every agent step is a fresh session that does its work and exits.

## ADDED Requirements

### Requirement: Harness is a plugin capability
A harness SHALL be provided by a plugin that declares the harness capability in its manifest. The engine SHALL contain no knowledge of any specific harness.

#### Scenario: New harness added without engine change
- **WHEN** a plugin that declares the harness capability is installed
- **THEN** workflows can name it in a profile without any change to the engine

### Requirement: Session lifecycle
A harness plugin SHALL support opening a session for one agent step, sending it a prompt, receiving its events, and closing it. A session SHALL keep its conversation between the prompts of that step and SHALL be closed when the step exits. A session SHALL NOT be reused by a later step.

#### Scenario: Open and prompt
- **WHEN** the engine opens a session and sends a prompt
- **THEN** the harness runs the agent with that prompt and reports events until the turn ends

#### Scenario: Close at exit
- **WHEN** an agent step exits
- **THEN** the engine closes its session and the harness ends the agent and releases its resources

#### Scenario: Later step starts fresh
- **WHEN** a later step uses the same profile
- **THEN** the harness opens a new session with no conversation from the earlier step

### Requirement: Headless mode is required
Every harness plugin SHALL support headless mode, in which the agent runs without a terminal for a person and nothing in the session waits for a person. Questions SHALL accumulate during the turn and be reported when the step exits, and permission requests SHALL be answered by the profile's permission policy.

#### Scenario: Headless run
- **WHEN** a session is opened in headless mode
- **THEN** the agent runs to the end of its turn without needing a terminal or a person

### Requirement: Permission policy
A profile SHALL declare a permission policy of allow and deny rules that the harness plugin interprets for its harness. In headless mode the plugin SHALL answer every permission request from the policy, and a request that no allow rule covers SHALL be denied. A denial SHALL reach the agent as an ordinary tool error that names the denied action.

#### Scenario: Allowed action
- **WHEN** the agent requests an action that an allow rule covers
- **THEN** the plugin approves it without involving a person

#### Scenario: Denied action
- **WHEN** the agent requests an action that no allow rule covers
- **THEN** the plugin denies it and the agent receives a tool error

#### Scenario: Deny wins
- **WHEN** an action matches both an allow rule and a deny rule
- **THEN** the plugin denies it

### Requirement: Interactive mode is optional
A harness plugin MAY support interactive mode, in which the agent runs in a host outside the engine, such as the operator's terminal or a session manager, and the person in front of it answers its questions directly. A plugin that does not support interactive mode SHALL declare that in its manifest. The engine SHALL NOT host or attach to an interactive agent.

#### Scenario: Unsupported mode requested
- **WHEN** a workflow declares interactive mode for a harness that does not support it
- **THEN** validation fails and names the harness and the profile

#### Scenario: Hand off to a host
- **WHEN** an agent step runs on an interactive profile
- **THEN** the plugin hands the prompt and a run-scoped token to a host, the step exits with the outcome launched, and clients and channels are shown how to reach the host

### Requirement: Interactive completion
The engine SHALL provide a built-in trigger that resolves a wait step when an interactive agent reports completion, keyed by the run and the step. The injected tools of an interactive agent SHALL record that signal in the store with the run-scoped token, without a running engine, and SHALL carry the outcome and outputs the agent reported.

#### Scenario: Agent finishes days later
- **WHEN** an interactive agent calls the complete tool days after launch and no engine is running
- **THEN** the signal is recorded in the store and the next engine to run the run resumes the wait with that outcome and its outputs

#### Scenario: Token scope
- **WHEN** a token for one run and step is used for another
- **THEN** the store rejects the signal

#### Scenario: Questions in an interactive agent
- **WHEN** an interactive agent has a question
- **THEN** it asks the person at its terminal and the engine opens no attention request

### Requirement: Injected engine tools
The engine SHALL make three tools available to every session: one to complete the current step with an outcome, outputs, notes, and a summary, one to record questions for the operator, and one to report an artifact. Harness plugins SHALL expose them to the agent using the harness's own tool mechanism. In headless mode the ask tool SHALL return immediately and SHALL NOT wait for an answer.

#### Scenario: Agent completes the step
- **WHEN** the agent calls the complete tool with a declared outcome and its required outputs
- **THEN** the engine validates the outputs, resolves the step with that outcome, and the session ends

#### Scenario: Summary is an output
- **WHEN** the agent calls the complete tool with a summary
- **THEN** the step's outputs contain that summary under the name summary

#### Scenario: Undeclared outcome
- **WHEN** the agent calls the complete tool with an outcome the step does not declare
- **THEN** the tool returns an error listing the allowed outcomes and the step stays active

#### Scenario: Outputs fail validation
- **WHEN** the agent calls the complete tool with an outcome whose required outputs are missing or have the wrong type
- **THEN** the tool returns an error naming each problem and the step stays active

#### Scenario: Agent records questions
- **WHEN** a headless agent calls the ask tool
- **THEN** the tool records the questions and returns at once, and the agent continues its work

#### Scenario: Agent exits with questions
- **WHEN** the agent calls the complete tool with the outcome questions, its draft outputs, and notes
- **THEN** the step resolves with the outcome questions, carrying the questions it recorded, the draft outputs, the notes, and the summary

#### Scenario: Agent reports an artifact
- **WHEN** the agent calls the artifact tool with a kind, name, and link
- **THEN** the engine adds it to the run's registry

#### Scenario: Completion not yet recorded
- **WHEN** the agent calls the complete tool and the engine cannot yet record the outcome
- **THEN** the harness plugin holds the outcome and retries it with the same key, and the step stays active until it is confirmed

### Requirement: Outputs per outcome
The engine SHALL tell the agent, in the instructions it appends to the prompt, the outcomes the step allows and the outputs each requires. Declared outputs SHALL be required for every outcome except questions, and for questions they SHALL be optional draft work. The standard outputs summary and notes SHALL be accepted for every outcome.

#### Scenario: Required for normal outcomes
- **WHEN** a step declares an output named plan and the agent completes with plan_ready without it
- **THEN** the report is rejected

#### Scenario: Optional for questions
- **WHEN** the agent exits with questions and a partial plan
- **THEN** the report is accepted and the partial plan is recorded as an output of the attempt

### Requirement: Step prompt construction
An agent step SHALL declare a prompt and MAY declare a command prefix, such as a skill or slash command, to be placed before the prompt, and a list of context references. The engine SHALL build the text the agent receives as the prefix, the prompt, a labelled section for each context reference, and instructions for reporting the result, and SHALL make that text visible to clients.

#### Scenario: Prefix is applied
- **WHEN** a step declares a prefix and a prompt
- **THEN** the harness receives the prefix followed by the prompt

#### Scenario: Context sections
- **WHEN** a step lists the plan as context
- **THEN** the text the harness receives contains a section labelled for the plan with its value

#### Scenario: Re-entry after questions
- **WHEN** a step is started again after its questions were answered
- **THEN** the text contains the original prompt, a section with the questions and answers, and a section with the previous attempt's draft outputs, notes, and summary

#### Scenario: Prompt visible
- **WHEN** a client asks what a step received
- **THEN** it can show the assembled text

### Requirement: Reporting instructions
The instructions in a step's prompt SHALL name the outcomes the step allows and the outputs each requires, and SHALL tell the agent to report the result with the complete tool.

#### Scenario: Instructions name outcomes
- **WHEN** a step allows the outcomes fixed and blocked
- **THEN** its prompt names both and tells the agent to call the complete tool with one of them

### Requirement: Prompt sections after the ask route
When a step is re-entered through the ask route the engine SHALL add labelled sections to its prompt for the questions and answers and for the previous attempt's outputs, notes, and summary, so the workflow author writes nothing for them.

#### Scenario: Re-entry sections
- **WHEN** a step that exited with questions and a draft is started again with the answers
- **THEN** its prompt contains a section with the questions and answers and a section with the previous attempt's draft outputs, notes, and summary

### Requirement: Turn end without completion
When an agent's turn ends without the agent having completed the step, the engine SHALL apply the step's declared policy, which SHALL be one of: re-prompt the agent, or resolve with a named outcome. When questions are recorded and the agent has not completed, the engine SHALL re-prompt once to save its work and exit with its questions before applying the policy.

#### Scenario: Re-prompt
- **WHEN** the turn ends without completion and the policy is re-prompt
- **THEN** the engine sends the agent a reminder to complete the step with one of the allowed outcomes

#### Scenario: Re-prompt limit
- **WHEN** the agent has been re-prompted the maximum number of times without completing
- **THEN** the run moves to needs-human

#### Scenario: Questions without completion
- **WHEN** the turn ends with questions recorded and no completion
- **THEN** the engine re-prompts once to save the work so far and exit with the questions

#### Scenario: Resolve with a named outcome
- **WHEN** the turn ends without completion and the policy names an outcome
- **THEN** the step resolves with that outcome

### Requirement: Mixed harnesses in one run
A single run SHALL be able to use profiles on different harness plugins. Outcomes, questions, and artifacts SHALL be handled identically regardless of harness.

#### Scenario: Two harnesses
- **WHEN** one step runs on one harness and a later step runs on another
- **THEN** both report outcomes and questions to the engine in the same form

### Requirement: Harness activity summary
A harness plugin SHALL report a short human-readable activity status, such as working or idle. Clients SHALL show this status and SHALL NOT be required to show the agent's output.

#### Scenario: Status shown
- **WHEN** the agent is running tools
- **THEN** the session status reads working

### Requirement: Session failure
When an agent process exits unexpectedly or the harness reports an error, the plugin SHALL report session failure with the reason. The engine SHALL resolve the active step with the outcome failed and apply the step's routing for failure.

#### Scenario: Agent crashes
- **WHEN** the agent process exits during a step
- **THEN** the step resolves with the outcome failed and the reason is recorded
