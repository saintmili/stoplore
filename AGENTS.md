# AGENTS.md

Guidance for coding agents working in this repository.

## Project Overview

StopLore is a Go library for recording and analyzing trading journal entries.


## Coding Guidelines

- Keep changes small, explicit, and rooted in the domain model.
- Return meaningful errors instead of silently returning nil values.
- Avoid global mutable state.
- Do not introduce a framework or code generation unless explicitly requested.
- Do not add comments that merely restate code; add comments only where they explain non-obvious domain or storage behavior.


## Commands

Useful commands:

- `go test ./...`
- `gofmt -w <files>`

Before running commands that may write files or caches, consider whether the user requested read-only inspection.

## Repository Hygiene

- Do not commit changes unless explicitly requested.

## Agent skills

### Issue tracker

Specs and tickets use GitHub Issues. Before reading or publishing them, read
`docs/agents/issue-tracker.md`.

### Triage labels

Use the default triage vocabulary. Before changing issue status, read
`docs/agents/triage-labels.md`.

### Domain docs

Use the single-context layout. Before exploring domain code, read
`docs/agents/domain.md`.
