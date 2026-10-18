# Spec Delta

## Purpose

Provides one harness plugin that drives any coding agent that speaks the Agent Client Protocol, so a single plugin covers many agents until a harness needs a dedicated one.

## ADDED Requirements

### Requirement: Drive any ACP agent
The plugin SHALL launch a configured agent command that speaks the Agent Client Protocol, open sessions on it, and send prompts, implementing the harness protocol.

#### Scenario: Configure an agent
- **WHEN** a workflow names the plugin with an agent command
- **THEN** the plugin launches that command and uses it for the session

#### Scenario: Agent does not speak the protocol
- **WHEN** the launched command fails the protocol handshake
- **THEN** the session fails with the reason

### Requirement: Permission requests become attention
The plugin SHALL translate the agent's permission requests into engine attention events of kind approval and answer the agent with the option the operator chose.

#### Scenario: Approve
- **WHEN** the agent requests permission with options and the operator picks allow
- **THEN** the plugin answers the agent with the allow option

#### Scenario: Cancelled while waiting
- **WHEN** the step is cancelled while a permission request is open
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

### Requirement: Session reuse
The plugin SHALL keep a session open between steps that share it and SHALL create a new agent session for a fresh session on each step.

#### Scenario: Reuse
- **WHEN** two steps share a non-fresh session
- **THEN** both prompts go to the same agent session

### Requirement: Declared limits
The plugin SHALL declare in its manifest that it supports headless mode only, so validation rejects interactive sessions that name it.

#### Scenario: Interactive rejected
- **WHEN** a workflow asks for interactive mode on this plugin
- **THEN** validation fails

### Requirement: Failure reporting
The plugin SHALL report session failure when the agent process exits or the protocol connection closes unexpectedly.

#### Scenario: Process exits
- **WHEN** the agent process exits during a turn
- **THEN** the plugin reports session failure with the exit status
