# Spec Delta

## Purpose

Provides one harness plugin that drives any coding agent that speaks the Agent Client Protocol, so a single plugin covers many agents until a harness needs a dedicated one.

## ADDED Requirements

### Requirement: Drive any ACP agent
The plugin SHALL launch a configured agent command that speaks the Agent Client Protocol, open a session on it for each agent step, and send prompts, implementing the harness protocol.

#### Scenario: Configure an agent
- **WHEN** a profile names the plugin with an agent command
- **THEN** the plugin launches that command and uses it for the session

#### Scenario: Agent does not speak the protocol
- **WHEN** the launched command fails the protocol handshake
- **THEN** the session fails with the reason

### Requirement: Permission requests follow the policy
The plugin SHALL answer the agent's permission requests from the profile's permission policy, choosing the allow or reject option the agent offered, and SHALL NOT wait for a person.

#### Scenario: Allowed
- **WHEN** the agent requests permission and an allow rule covers the action
- **THEN** the plugin answers the agent with its allow option

#### Scenario: Denied
- **WHEN** the agent requests permission and no allow rule covers the action
- **THEN** the plugin answers the agent with its reject option

#### Scenario: Cancelled while a request is pending
- **WHEN** the step is cancelled while a permission request is being answered
- **THEN** the plugin answers the agent with cancelled

### Requirement: Engine tools are provided to the agent
The plugin SHALL give the agent the engine's complete, ask, and artifact tools through a tool server that the agent connects to when the session opens.

#### Scenario: Tools visible
- **WHEN** a session opens
- **THEN** the agent can see and call the three engine tools

### Requirement: Activity reporting
The plugin SHALL map the agent's streamed updates to a short activity status and SHALL NOT forward the full stream to the engine by default.

#### Scenario: Status only
- **WHEN** the agent streams many updates during a turn
- **THEN** the engine receives status changes, not the full text

### Requirement: One session per step
The plugin SHALL create a new agent session for each agent step, send every prompt of that step to it, and end it when the step exits. The plugin SHALL NOT load or resume an earlier session.

#### Scenario: Fresh for each step
- **WHEN** two steps use the same profile
- **THEN** each step's prompts go to its own agent session

#### Scenario: Several prompts in one step
- **WHEN** the engine re-prompts the agent within a step
- **THEN** the prompt goes to the same agent session

### Requirement: Declared limits
The plugin SHALL declare in its manifest that it supports headless mode only, so validation rejects interactive profiles that name it.

#### Scenario: Interactive rejected
- **WHEN** a workflow declares an interactive profile on this plugin
- **THEN** validation fails

### Requirement: Failure reporting
The plugin SHALL report session failure when the agent process exits or the protocol connection closes unexpectedly.

#### Scenario: Process exits
- **WHEN** the agent process exits during a turn
- **THEN** the plugin reports session failure with the exit status
