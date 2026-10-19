# Spec Delta

## Purpose

Provides time as a plugin: a trigger that starts runs on a schedule or at a given moment, and a trigger that resumes a waiting run after a delay or at a given moment. The engine contains no clock logic of its own beyond the deadlines it records for timeouts.

## ADDED Requirements

### Requirement: Recurring schedule start trigger
The plugin SHALL provide a start trigger that fires on a recurring schedule written in cron syntax with an optional time zone. Each firing SHALL carry a payload with the scheduled time and a dedupe key derived from the schedule and that time.

#### Scenario: Fires on schedule
- **WHEN** the schedule is every hour and the hour arrives
- **THEN** the trigger fires once with the scheduled time

#### Scenario: Time zone
- **WHEN** the schedule names a time zone
- **THEN** it fires at the local times of that zone, including across daylight saving changes

#### Scenario: Invalid schedule
- **WHEN** a schedule cannot be parsed
- **THEN** validation fails and names the schedule

### Requirement: One-shot start trigger
The plugin SHALL provide a start trigger that fires once at a given moment.

#### Scenario: One-shot fires
- **WHEN** the moment arrives
- **THEN** the trigger fires once

### Requirement: Missed firings
The plugin SHALL record the last scheduled time it fired for each subscription in a way that survives restarts. When the engine was not running at a scheduled time, the plugin SHALL fire once for the most recent missed time by default and MAY be configured to skip missed times or to fire for each of them.

#### Scenario: Engine down at the scheduled time
- **WHEN** the engine starts after two missed hourly times and the default applies
- **THEN** the trigger fires once for the most recent missed time

#### Scenario: Skip configured
- **WHEN** the subscription is configured to skip missed times
- **THEN** the trigger waits for the next scheduled time

#### Scenario: No duplicate on restart
- **WHEN** the engine restarts shortly after a firing
- **THEN** the same time does not fire again

### Requirement: Delay and deadline resume trigger
The plugin SHALL provide a trigger for wait steps that resumes a run after a duration or at a moment computed from the run, classifying the event into the outcome elapsed. Its correlation key SHALL identify the run and the step, and it SHALL offer check-now so a moment already past resolves immediately. Delivery SHALL not depend on any process having run for the wait.

#### Scenario: Delay elapses
- **WHEN** a wait step lists a delay of two days and two days pass
- **THEN** the wait resolves with elapsed

#### Scenario: Moment already past
- **WHEN** a run reaches a wait whose moment is in the past
- **THEN** check-now resolves it with elapsed

#### Scenario: Engine down when the time arrives
- **WHEN** the moment passes while no engine runs and an engine later starts
- **THEN** the wait resolves with elapsed once
