# workflow

Orchestrate agentic workflows.

A workflow is a fixed graph of steps that agents move through. The engine decides what happens next, so the agent no longer carries the process in its head. People running many agents can see at a glance which step each run is on, whether it needs them, and which artifacts it has produced.

```
  TRIGGERS                ENGINE                       CLIENTS
  (plugins)        +----------------------+      (terminal, browser,
 github issue  --> | workflow definitions |       your own)
 prompt        --> | run state machine    | ----> what step, what needs
 (your own)    --> | event log            | <---- you, which artifacts
                   +----------+-----------+       answer, start, cancel
                              |
              +---------------+----------------+
              v               v                v
         github plugin   harness plugins    shell plugin ...
         openspec plugin (Claude Code, Codex, Cursor, ...)
```

- Plugins do everything specific. The engine knows nothing about GitHub, OpenSpec, or any agent.
- Any agent harness is a plugin, and one workflow can mix several.
- Workflows are written in YAML or built with code. Both produce the same intermediate form, so every client can draw the graph before it runs.
- Questions and approvals from any agent reach you the same way, in a terminal, a browser, or a pipeline.
- It ships as one executable that runs anywhere a process can run.

## Status

This repository is at the specification stage. The whole platform is specified in one OpenSpec change, `establish-workflow-platform`, and will be built over many sessions.

- `openspec/changes/establish-workflow-platform/proposal.md` explains what and why.
- `openspec/changes/establish-workflow-platform/design.md` records the architecture, the decisions, and a complete example workflow.
- `openspec/changes/establish-workflow-platform/specs/` holds one specification per capability.
- `openspec/changes/establish-workflow-platform/tasks.md` is the build order.
