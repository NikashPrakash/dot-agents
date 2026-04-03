# Codex Agent Field Regression Plan

## Goal

Fix the Codex subagent TOML transform so generated files use the field name Codex expects, verify the generated output, and capture a prevention plan for future platform schema drift.

## Tasks

- [complete] Confirm the root cause in the Codex platform renderer and identify affected tests, docs, and generated files.
- [complete] Update the Codex TOML renderer and regression tests to emit the correct field name.
- [complete] Run targeted verification for the Codex platform code and refresh the generated `~/.codex/agents/*.toml` files.
- [complete] Write implementation results and prevention notes under `.agents/history/codex-agent-field-regression/`.
