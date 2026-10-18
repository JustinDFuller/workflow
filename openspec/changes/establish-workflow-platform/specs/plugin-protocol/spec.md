# Spec Delta

## Purpose

Defines how the engine finds, launches, and talks to plugins, so that anyone can write and share a plugin in any language and the engine needs no knowledge of what a plugin does.

## ADDED Requirements

### Requirement: Plugins are separate processes
A plugin SHALL be an executable that the engine launches and talks to using JSON-RPC 2.0 messages over standard input and output. Built-in plugins SHALL use the same protocol as third-party plugins.

#### Scenario: Built-in plugin uses the protocol
- **WHEN** the engine uses a built-in plugin
- **THEN** it launches it and communicates through the same messages as for any other plugin

#### Scenario: Any language
- **WHEN** an executable written in any language implements the protocol
- **THEN** the engine can use it as a plugin

### Requirement: Manifest
Every plugin SHALL ship a manifest that declares its name, version, protocol version, command to launch, and every trigger, step, artifact kind, and harness capability it provides. Each step SHALL declare its input schema, its output schema, and the complete list of outcomes it can report.

#### Scenario: Manifest read without launching
- **WHEN** the engine validates a workflow
- **THEN** it reads plugin manifests without launching plugin processes

#### Scenario: Manifest invalid
- **WHEN** a manifest is missing required fields or fails the schema
- **THEN** the engine refuses to load the plugin and reports the problem

### Requirement: Initialization handshake
On launch the engine SHALL send an initialize request carrying its protocol version and capabilities, and the plugin SHALL answer with the protocol version it will speak and its capabilities. The engine SHALL send shutdown before ending a plugin.

#### Scenario: Incompatible protocol version
- **WHEN** a plugin answers with a protocol version the engine does not support
- **THEN** the engine stops the plugin and reports the mismatch

### Requirement: Step execution
The engine SHALL start a step with a start request carrying the step id, the namespace, inputs, run context needed by the step, and a unique execution id. A step MAY be started more than once for the same run when its earlier resolution was never recorded. The plugin SHALL report completion with the execution id, one declared outcome, and outputs that match the declared output schema.

#### Scenario: Step completes
- **WHEN** a plugin reports completion with a declared outcome and valid outputs
- **THEN** the engine resolves the step with that outcome

#### Scenario: Invalid outputs
- **WHEN** a plugin reports outputs that fail the declared schema
- **THEN** the engine treats it as a protocol error and moves the run to needs-human

### Requirement: Signals are keyed and retried
Every message from a plugin that changes a run, such as a step completion, a trigger event, an artifact report, or an attention request, SHALL carry a unique key. The engine SHALL answer it only after it is recorded. A plugin SHALL keep each such message and retry it with the same key until it is confirmed or the plugin exits.

#### Scenario: Completion not recorded
- **WHEN** a plugin reports completion and the engine answers that it was not recorded
- **THEN** the plugin keeps the completion and retries it with the same key

#### Scenario: Retry is harmless
- **WHEN** a plugin repeats a completion whose key was already recorded
- **THEN** the engine confirms it and the step resolves only once

### Requirement: Long-running steps
A step MAY stay pending for any length of time. A plugin MAY send progress messages with a short human-readable status. The engine SHALL surface the latest status to clients and SHALL NOT treat silence as failure unless a timeout is declared.

#### Scenario: Waiting for a review
- **WHEN** a plugin step waits for a human review for several days
- **THEN** the step stays active and clients see the latest status

### Requirement: Step cancellation
The engine SHALL be able to cancel a running step with a cancel request carrying the execution id. A plugin SHALL stop work and answer when cancellation is complete.

#### Scenario: Cancel on run cancel
- **WHEN** a run is cancelled while a step is active
- **THEN** the engine sends a cancel request for that execution

### Requirement: Triggers
A plugin MAY provide triggers. The engine SHALL ask a plugin to subscribe to a trigger with the configuration from a workflow. The plugin SHALL report each event with a payload and a dedupe key, and SHALL do its own polling or webhook handling.

#### Scenario: Subscribe
- **WHEN** the engine loads a workflow with a trigger from a plugin
- **THEN** it sends a subscribe request with the trigger configuration

#### Scenario: Event fired
- **WHEN** the plugin reports a trigger event
- **THEN** the engine starts a run as described in the run engine

#### Scenario: Unsubscribe
- **WHEN** a workflow is removed or the engine stops
- **THEN** the engine sends an unsubscribe request

### Requirement: Webhook delivery
The engine MAY expose an HTTP endpoint for webhooks. When it receives a request for a path registered by a plugin, it SHALL pass the request to that plugin and return the plugin's response. The engine SHALL NOT interpret the request.

#### Scenario: Webhook reaches the plugin
- **WHEN** an HTTP request arrives at a path a plugin registered
- **THEN** the plugin receives the method, headers, and body and its response is returned to the sender

#### Scenario: Unregistered path
- **WHEN** a request arrives at a path no plugin registered
- **THEN** the engine returns not found

#### Scenario: Webhook cannot be recorded
- **WHEN** the plugin's resulting event cannot be recorded because the store is unreachable
- **THEN** the engine returns a retryable error so the sender can deliver the request again

### Requirement: Artifact reporting
A plugin SHALL be able to report an artifact with a declared kind, a name, and a link or reference at any time while a step is active, and SHALL be able to answer a probe request for an artifact kind it declared probe support for.

#### Scenario: Report during a step
- **WHEN** a plugin reports a pull request artifact with a link
- **THEN** the engine adds it to the run's registry

#### Scenario: Undeclared kind
- **WHEN** a plugin reports an artifact kind it did not declare
- **THEN** the engine rejects the report

### Requirement: Attention from plugins
A plugin step SHALL be able to request operator attention with a kind and prompt, and SHALL receive the answer.

#### Scenario: Plugin asks for help
- **WHEN** a plugin step cannot continue and requests attention
- **THEN** the engine opens a request and returns the answer to the plugin when given

### Requirement: Logging
A plugin SHALL be able to send log messages with a level. The engine SHALL keep plugin logs separate from run events and SHALL NOT show them in the default client view.

#### Scenario: Logs retrievable
- **WHEN** an operator asks for a plugin's logs
- **THEN** the engine returns them without placing them in the run's events

### Requirement: Crash handling
When a plugin process exits unexpectedly, the engine SHALL record it, restart the plugin up to a configured limit, and re-attach or restart its active steps as the plugin declared it supports. After the limit the engine SHALL move affected runs to needs-human.

#### Scenario: Plugin restarts
- **WHEN** a plugin process crashes once
- **THEN** the engine restarts it and affected steps continue as declared

#### Scenario: Repeated crashes
- **WHEN** a plugin crashes more times than the limit
- **THEN** affected runs move to needs-human with the plugin's last logs attached

### Requirement: Plugin resolution
A workflow SHALL refer to a plugin by a name and a version constraint. The engine SHALL resolve it from a local path or a git reference, and SHALL record the resolved version with each run.

#### Scenario: Local path
- **WHEN** a plugin is referenced by a local path
- **THEN** the engine loads the manifest and executable from that path

#### Scenario: Version recorded
- **WHEN** a run starts
- **THEN** it records the exact version of every plugin it uses

### Requirement: Trust model
Plugins SHALL be trusted executables that run with the permissions of the engine. The engine SHALL show the command a plugin will run before it is first added, and SHALL pass secrets to a plugin only as environment variables the workflow names.

#### Scenario: Adding a plugin
- **WHEN** an operator adds a plugin for the first time
- **THEN** the engine shows the command it will launch and requires confirmation

#### Scenario: Secrets are explicit
- **WHEN** a plugin is launched
- **THEN** it receives only the environment variables its workflow configuration names, plus the engine's required protocol variables
