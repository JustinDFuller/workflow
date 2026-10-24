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
