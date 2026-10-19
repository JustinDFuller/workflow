# Spec Delta

## Purpose

Defines how the engine finds, launches, and talks to plugins, so that anyone can write and share a plugin in any language and the engine needs no knowledge of what a plugin does. Plugins do work and report events, and the engine decides when a run waits and what an event means for it.

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
Every plugin SHALL ship a manifest that declares its name, version, protocol version, command to launch, and every trigger, step, attention channel, artifact kind, and harness capability it provides. Each step SHALL declare its input schema, its output schema, and the complete list of outcomes it can report.

#### Scenario: Manifest read without launching
- **WHEN** the engine validates a workflow
- **THEN** it reads plugin manifests without launching plugin processes

#### Scenario: Manifest invalid
- **WHEN** a manifest is missing required fields or fails the schema
- **THEN** the engine refuses to load the plugin and reports the problem

#### Scenario: Trigger outcomes known statically
- **WHEN** a workflow lists a trigger in a wait step
- **THEN** validation uses the outcomes declared in the manifest to check that every outcome is routed

### Requirement: Trigger declarations
Each trigger in a manifest SHALL declare its configuration schema, its payload schema, whether it can start runs, whether it can resume a wait, the outcomes it classifies an event into when it resumes a wait, whether it offers check-now, and how it derives a correlation key.

#### Scenario: Incomplete trigger declaration
- **WHEN** a manifest declares a trigger that can resume a wait without declaring its outcomes
- **THEN** the engine rejects the manifest and names the trigger

### Requirement: Initialization handshake
On launch the engine SHALL send an initialize request carrying its protocol version and capabilities, and the plugin SHALL answer with the protocol version it will speak and its capabilities. The engine SHALL send shutdown before ending a plugin.

#### Scenario: Incompatible protocol version
- **WHEN** a plugin answers with a protocol version the engine does not support
- **THEN** the engine stops the plugin and reports the mismatch

### Requirement: Step execution
The engine SHALL start a step with a start request carrying the step id, the namespace, inputs, run context needed by the step, and a unique execution id. A step MAY be started more than once for the same run when its earlier resolution was never recorded. The plugin SHALL report completion with the execution id, one declared outcome, and outputs that match the declared output schema. Every plugin step SHALL also be able to report the built-in outcome needs_help with a message.

#### Scenario: Step completes
- **WHEN** a plugin reports completion with a declared outcome and valid outputs
- **THEN** the engine resolves the step with that outcome

#### Scenario: Invalid outputs
- **WHEN** a plugin reports outputs that fail the declared schema
- **THEN** the engine treats it as a protocol error and moves the run to needs-human

#### Scenario: Step cannot continue
- **WHEN** a plugin step reports needs_help with a message
- **THEN** the engine resolves the step with that outcome and the workflow's route or the needs-human target applies

### Requirement: Signals are keyed and retried
Every message from a plugin that changes a run, such as a step completion, a trigger event, an artifact report, or an answer from an attention channel, SHALL carry a unique key. The engine SHALL answer it only after it is recorded. A plugin SHALL keep each such message and retry it with the same key until it is confirmed or the plugin exits.

#### Scenario: Completion not recorded
- **WHEN** a plugin reports completion and the engine answers that it was not recorded
- **THEN** the plugin keeps the completion and retries it with the same key

#### Scenario: Retry is harmless
- **WHEN** a plugin repeats a completion whose key was already recorded
- **THEN** the engine confirms it and the step resolves only once

### Requirement: Steps do work and exit
A plugin step MAY take any length of time to do its work and MAY send progress messages with a short human-readable status, which the engine SHALL surface to clients. A plugin step SHALL NOT remain active to wait for a person or an outside event, and SHALL instead exit so that the workflow can wait at a wait step. The engine SHALL NOT treat silence from a working step as failure unless a timeout is declared.

#### Scenario: Long work
- **WHEN** a plugin step spends an hour building and sends progress messages
- **THEN** the step stays active and clients see the latest status

#### Scenario: Waiting is a wait step
- **WHEN** a workflow needs to wait for a human review
- **THEN** it uses a wait step with a review trigger and no plugin step stays active during the wait

### Requirement: Step cancellation
The engine SHALL be able to cancel a running step with a cancel request carrying the execution id. A plugin SHALL stop work and answer when cancellation is complete.

#### Scenario: Cancel on run cancel
- **WHEN** a run is cancelled while a step is active
- **THEN** the engine sends a cancel request for that execution

### Requirement: Start triggers
A plugin MAY provide triggers that start runs. The engine SHALL ask a plugin to subscribe to a start trigger with the configuration from a workflow. The plugin SHALL report each event with a payload and a dedupe key, and SHALL do its own polling or webhook handling.

#### Scenario: Subscribe
- **WHEN** the engine loads a workflow with a start trigger from a plugin
- **THEN** it sends a subscribe request with the trigger configuration

#### Scenario: Event fired
- **WHEN** the plugin reports a trigger event
- **THEN** the engine starts a run as described in the run engine

#### Scenario: Unsubscribe
- **WHEN** a workflow is removed or the engine stops
- **THEN** the engine sends an unsubscribe request

### Requirement: Resume triggers
A plugin MAY provide triggers that resume a wait. For such a trigger the plugin SHALL derive a correlation key from the trigger configuration when asked, and SHALL derive the same key from each event it reports. The engine SHALL subscribe the trigger with the configuration of a waiting run and SHALL match events to runs by comparing keys without interpreting them. For each event the plugin SHALL report the outcome the event is classified into and the outputs extracted from it.

#### Scenario: Key from configuration
- **WHEN** the engine asks a review trigger for the key of a configuration that names pull request 42 in a repository
- **THEN** the plugin answers with a key that identifies that pull request

#### Scenario: Key from an event
- **WHEN** the plugin receives a review event on that pull request
- **THEN** it reports the event with the same key, an outcome, and outputs

#### Scenario: Unrelated event
- **WHEN** the plugin receives an event whose key matches no subscription
- **THEN** the engine records nothing for it

### Requirement: Check-now
A trigger MAY offer a check-now call. The engine SHALL call it with the trigger's configuration when a run enters a wait step, and the plugin SHALL answer either that the condition does not yet hold or with the outcome and outputs it would have reported for the event that satisfied it.

#### Scenario: Condition already holds
- **WHEN** a check-now call finds the pull request is already approved
- **THEN** the plugin answers with the approved outcome and the engine resolves the wait

#### Scenario: Condition does not hold
- **WHEN** the call finds nothing that satisfies the trigger
- **THEN** the plugin answers that it does not hold and the run waits

### Requirement: Event delivery from outside
The engine SHALL accept a raw event addressed to a plugin from a short-lived process, such as one handling a continuous integration job's event payload, and SHALL pass it to the plugin the same way as an event the plugin received itself.

#### Scenario: Delivered event
- **WHEN** a raw event is delivered for a plugin whose trigger matches a waiting run
- **THEN** the plugin classifies it and the engine resumes the run

### Requirement: Webhook delivery
The engine MAY expose an HTTP endpoint for webhooks, separate from the endpoint that clients connect to and configured on its own address. When it receives a request for a path registered by a plugin, it SHALL pass the request to that plugin and return the plugin's response. The engine SHALL NOT interpret the request.

#### Scenario: Webhook reaches the plugin
- **WHEN** an HTTP request arrives at a path a plugin registered
- **THEN** the plugin receives the method, headers, and body and its response is returned to the sender

#### Scenario: Unregistered path
- **WHEN** a request arrives at a path no plugin registered
- **THEN** the engine returns not found

#### Scenario: Webhook cannot be recorded
- **WHEN** the plugin's resulting event cannot be recorded because the store is unreachable
- **THEN** the engine returns a retryable error so the sender can deliver the request again

#### Scenario: Clients not exposed
- **WHEN** the webhook endpoint is reachable from the internet
- **THEN** it serves only registered webhook paths and none of the client protocol

### Requirement: Attention channels
A plugin MAY provide attention channels. The engine SHALL send an attention request to a channel with its kind, prompt, questions, choices, and context, and the channel SHALL report each answer with the request id and a unique key. A channel MAY post the request anywhere it can read replies from, such as a chat thread, a ticket comment, or a pull request comment.

#### Scenario: Request delivered
- **WHEN** a run opens an attention request and the workflow names a channel
- **THEN** the engine sends the request to that channel

#### Scenario: Reply becomes an answer
- **WHEN** a person replies in the place the channel posted
- **THEN** the channel reports an answer and the engine records it

#### Scenario: Channel down
- **WHEN** a channel cannot be reached
- **THEN** the request stays open for clients and the engine records the failed delivery

### Requirement: Artifact reporting
A plugin SHALL be able to report an artifact with a declared kind, a name, and a link or reference at any time while a step is active, and SHALL be able to answer a probe request for an artifact kind it declared probe support for.

#### Scenario: Report during a step
- **WHEN** a plugin reports a pull request artifact with a link
- **THEN** the engine adds it to the run's registry

#### Scenario: Undeclared kind
- **WHEN** a plugin reports an artifact kind it did not declare
- **THEN** the engine rejects the report

### Requirement: Logging
A plugin SHALL be able to send log messages with a level. The engine SHALL keep plugin logs separate from run events and SHALL NOT show them in the default client view.

#### Scenario: Logs retrievable
- **WHEN** an operator asks for a plugin's logs
- **THEN** the engine returns them without placing them in the run's events

### Requirement: Crash handling
When a plugin process exits unexpectedly, the engine SHALL record it, restart the plugin up to a configured limit, subscribe its triggers again, and re-attach or restart its active steps as the plugin declared it supports. After the limit the engine SHALL move affected runs to needs-human.

#### Scenario: Plugin restarts
- **WHEN** a plugin process crashes once
- **THEN** the engine restarts it, subscribes its triggers again, and affected steps continue as declared

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
