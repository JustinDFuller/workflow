# Contributing

Specifications live under `openspec/changes` until archived and under `openspec/specs` after. Read the proposal, design, and specs of the change you touch before writing code. Do not build anything the approved specs do not describe.

## Build and test

Build the executable with `go build -o wf ./cmd/wf`.

Run every test with `go test ./...`.

Check formatting with `gofmt -l .` and vet with `go vet ./...`.

## Guardrails

CI bans code comments in Go and shell files. Run `go run ./tools/nocomments .` to check.

Write each paragraph and list item of prose on one line, with no line breaks inside sentences.

Total test coverage must stay at or above 90 percent.

## Hooks and what they prevent

Run `wf install claude` in a worktree to add the stop, tool, and session hooks, the status line, and the `/wf` command to `.claude/settings.local.json`.

The hooks make it hard for an agent to forge a run, not impossible, because a shell command can hide a path. The log at `$WF_HOME/runs/<id>/events.jsonl` records every check result, so a person can audit what was checked and what it said.

Hooks fail open and never run a check. Keep it that way: a broken referee must not lock a person out of the session.
