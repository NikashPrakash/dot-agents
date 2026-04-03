# Implementation Results

## Task

Fix malformed Codex subagent files that were emitted with `instructions` instead of the required `developer_instructions` field.

## Root Cause

- The Codex TOML renderer added in commit `c8ea014` serialized the agent Markdown body under `instructions`.
- The regression test asserted the same incorrect key, so the bug was locked in instead of caught.
- The implementation did not include a live Codex smoke check or a primary-source schema verification step when native `.codex/agents/*.toml` support was added.

## Changes

- Updated `internal/platform/codex.go` to emit `developer_instructions` for agent body content.
- Updated `internal/platform/codex_test.go` to assert the correct key and fail if the legacy top-level `instructions` key reappears.
- Refreshed the `dot-agents` project so the generated user-level files under `~/.codex/agents/` were rewritten with the corrected field.

## Verification

- `go test ./internal/platform`
- `go test ./commands`
- `go run ./cmd/dot-agents refresh dot-agents`
- Confirmed `~/.codex/agents/test-runner.toml` and `~/.codex/agents/verifier.toml` now contain `developer_instructions`.
- Confirmed `codex --help` no longer prints malformed agent role warnings. It still prints an unrelated PATH warning: `WARNING: proceeding, even though we could not update PATH: Operation not permitted (os error 1)`.

## Prevention Plan

1. Keep exact-key regression tests for every native platform transform, not just “body rendered” coverage.
2. When adding or changing platform-native formats, verify against the current primary docs and record the checked contract in the task notes.
3. Add a lightweight post-refresh smoke check for Codex-native outputs when Codex is installed locally, so malformed agent files surface immediately.
4. Close the remaining Codex drift outside this fix:
   - reverse-import and refresh mapping still assume directory-shaped `.codex/agents/<name>/...` paths
   - bash parity still describes and emits legacy Codex agent layouts in several scripts and docs
