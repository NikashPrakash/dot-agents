# Handoff: Plugin Support Next

**Created:** 2026-03-30
**Author:** Claude Code session
**For:** AI Agent
**Status:** In progress

---

## Summary

The current task was research and planning for plugin support as a major next resource type in `dot-agents`. I reviewed official plugin docs across Cursor, Claude Code, Codex, OpenCode, and GitHub Copilot CLI, then updated the local docs to reflect the actual current platform capabilities and wrote a multi-platform plugin strategy. The next agent should treat plugin support as one of the main next buckets after the canonical stage work, with platform-specific implementations rather than a fake universal plugin abstraction.

## Project Context

`dot-agents` is a unified config layer for AI coding agents. The active project plan is the canonical `~/.agents` rollout, with Stage 1 Go work largely completed and Phase 5 reserved for new buckets including `plugins`. The current repo is on branch `claude/scalable-skill-syncing-sfxOd` and has unrelated in-progress code changes outside this docs/planning work, so do not revert or disturb them.

## The Plan

Primary source plan: `.agents/active/platform-dir-unification.plan.md`

Relevant current plan state:

- Stage 1 shared Go spine and contract: completed
- Stage 2 Go platform emitter wave: completed
- Stage 3 Go integration and validation: in progress, but core acceptance is effectively met and remaining work is optional cleanup/coverage
- Stage 4 bash parity wave: not started
- Stage 5 new bucket expansion: not started

The Stage 5 section is directly relevant:

- Extend the shared contract for `commands`, `output-styles`, `ignore`, `modes`, `plugins`, `themes`, and `prompts`
- Extend `init`, `import`, `refresh`, `status`, and `explain`
- Worker split:
  - Worker A: Cursor and Claude resource additions
  - Worker B: OpenCode resource additions
  - Worker C: Copilot resource additions
  - Codex likely has no new standalone bucket unless docs or product behavior change

Ownership rules that still matter:

- Shared schema, normalization, command UX, and shared tests stay coordinator-owned
- Platform workers should stay within assigned platform files
- Bash parity is later; Go-first remains the intended rollout shape

## Key Files

| File | Why It Matters |
|------|----------------|
| `.agents/active/platform-dir-unification.plan.md` | Active rollout plan; plugins are explicitly listed in Phase 5 |
| `docs/PLATFORM_DIRS_DOCS.md` | Main researched matrix of official platform resource/plugin locations and current repo gaps |
| `docs/PLUGIN_SUPPORT_STRATEGY.md` | New multi-platform plugin strategy and rollout order |
| `docs/OPENCODE_PLUGIN_SUPPORT_PLAN.md` | New OpenCode-specific plugin implementation plan |
| `internal/platform/opencode.go` | Current OpenCode implementation; still only supports `opencode.json`, `.opencode/agent/`, and `.agents/skills/` |
| `commands/import.go` | Will need plugin path normalization/import work when implementation starts |
| `commands/refresh.go` | Will need plugin reverse-mapping once plugin resources are supported |
| `commands/status.go` | Will need plugin visibility in audits/status output |

## Current State

**Done:**
- Reviewed official plugin docs for:
  - Cursor
  - Claude Code
  - Codex
  - OpenCode
  - GitHub Copilot CLI
- Updated `docs/PLATFORM_DIRS_DOCS.md` so plugin support is represented correctly across platforms.
- Added `docs/PLUGIN_SUPPORT_STRATEGY.md` with a broad plugin rollout strategy.
- Added `docs/OPENCODE_PLUGIN_SUPPORT_PLAN.md` with a concrete OpenCode plan.
- Corrected earlier assumptions as better source links were provided, especially for Claude, Cursor, and Copilot.

**In Progress:**
- Plugin support is only documented and planned. No implementation has started for the new plugin buckets.
- The repo has unrelated in-progress code changes in command/config files that predate or sit outside this docs pass.

**Not Started:**
- Shared canonical plugin descriptor layer
- Go implementation of plugin resource support
- Import/refresh/status/doctor/remove support for plugin buckets
- Bash parity for plugin buckets

## Decisions Made

- **Plugins should be treated as platform-specific systems, not one universal abstraction** — The docs show materially different shapes: Cursor/Claude/Codex/Copilot use package manifests and marketplaces; OpenCode uses native plugin source files plus config-based npm plugin activation.
- **Plugins should be one of the main next supported resources** — This came directly from the user and aligns with Phase 5 in the active plan.
- **OpenCode remains a native-resource-management case** — Its plugin model is executable JS/TS under `.opencode/plugins/` plus `.opencode/package.json`, so it fits the existing link model best.
- **Claude, Codex, Cursor, and Copilot all now qualify as repo-aware plugin targets** — Earlier weaker interpretations were corrected after reviewing the more direct vendor docs.
- **Copilot should not be modeled only as an installed-cache system** — The better docs show repo-aware manifests and marketplace support.
- **Go-first, bash-later still applies** — Plugin support should follow the established rollout pattern rather than forcing bash parity up front.

## Important Context

- The user explicitly wants plugin support to become one of the main supported resource types next.
- The best current planning artifact for that is `docs/PLUGIN_SUPPORT_STRATEGY.md`.
- Current dirty repo state is not only from this task. At handoff time:
  - modified: `commands/agents.go`
  - modified: `commands/agentsrc_mutations_test.go`
  - modified: `commands/skills.go`
  - modified: `commands/sync.go`
  - modified: `docs/PLATFORM_DIRS_DOCS.md`
  - modified: `internal/config/agentsrc.go`
  - modified: `internal/config/agentsrc_test.go`
  - modified: `src/lib/commands/agents.sh`
  - untracked: `docs/PLUGIN_SUPPORT_STRATEGY.md`
  - untracked: `docs/OPENCODE_PLUGIN_SUPPORT_PLAN.md`
  - untracked: `dot-agents`
- There is also a deleted active handoff and a corresponding history copy:
  - deleted: `.agents/active/handoffs/2026-03-30-hook-bundle-rendering.md`
  - untracked history copy: `.agents/history/platform-dir-unification/handoffs/2026-03-30-hook-bundle-rendering.md`
- I did not change production code for plugins in this session.
- Earlier in the session, `go test ./internal/platform ./commands` passed while this was still docs-only work.

## Next Steps

1. **Review and ratify `docs/PLUGIN_SUPPORT_STRATEGY.md` as the execution plan** — Confirm the platform ordering and whether `plugins` should be elevated ahead of some other Phase 5 buckets.
2. **Design the shared Go plugin contract** — Define canonical storage, emission modes, and command-layer normalization for plugin resources without pretending the platforms share one manifest.
3. **Start with OpenCode implementation** — Add `plugins/{scope}` and optional `package.json` support into `internal/platform/opencode.go`, then extend `status`, `refresh`, and `import`.
4. **Implement package-based plugin support for Claude, Codex, Cursor, and Copilot** — Focus on authoring and marketplace metadata first, not installation wrappers.
5. **Add CLI scaffolding once the canonical models settle** — `dot-agents plugins new --platform <platform>` is a strong follow-up after the shared contract exists.

## Constraints

- Do not revert or overwrite the unrelated dirty work already present in the repo.
- Keep the rollout aligned with the active canonical-storage plan.
- Prefer primary vendor docs only when refining plugin behavior.
- Preserve the distinction between:
  - native plugin source management
  - plugin package authoring
  - marketplace metadata
  - installation flows
- Avoid inventing a fake cross-platform plugin manifest.

---

## Update — 2026-03-30

Plugin support moved from planning into a substantial first implementation wave. The current state is no longer "docs only".

### What Changed

- Added a shared canonical plugin manifest layer in `internal/platform/plugins.go`.
- Added plugin bucket support to:
  - `commands/init.go`
  - `commands/explain.go`
  - `commands/status.go`
- Added command-layer plugin lifecycle edges:
  - `commands/add.go` now creates `~/.agents/plugins/<project>/` and surfaces discovered plugin-native roots
  - `commands/remove.go` now includes project plugin dirs in `--clean`
- Added OpenCode native plugin emission in `internal/platform/opencode.go`:
  - project plugins emit to `.opencode/plugins/<name>/`
  - global plugins emit to `~/.config/opencode/plugins/<name>/`
  - only canonical `kind: native` plugins targeting `opencode` are emitted
- Added OpenCode plugin reverse normalization:
  - `commands/import.go` can normalize `.opencode/plugins/<name>/...` into canonical `plugins/<scope>/<name>/...`
  - `commands/refresh.go` now maps emitted OpenCode plugin files back into canonical plugin paths
- Added `.agentsrc.json` plugin portability:
  - `internal/config/agentsrc.go` now supports a `plugins` field
  - `GenerateAgentsRC` detects canonical plugin bundles via `PLUGIN.yaml`
  - `commands/install.go` now resolves plugin bundles from sources and includes them in `install --generate`
- Added OpenCode plugin health visibility:
  - `commands/status.go` now counts/audits OpenCode plugin trees
  - `commands/doctor.go` now detects broken OpenCode plugin symlinks in both repo and global locations

### Files Added

- `internal/platform/plugins.go`
- `internal/platform/plugins_test.go`
- `internal/platform/opencode_plugins_test.go`
- `commands/init_test.go`
- `commands/status_test.go`
- `commands/remove_test.go`

### Files Changed

- `commands/add.go`
- `commands/add_test.go`
- `commands/agentsrc_mutations_test.go`
- `commands/doctor.go`
- `commands/explain.go`
- `commands/import.go`
- `commands/import_test.go`
- `commands/init.go`
- `commands/install.go`
- `commands/refresh.go`
- `commands/refresh_test.go`
- `commands/remove.go`
- `commands/status.go`
- `internal/config/agentsrc.go`
- `internal/config/agentsrc_test.go`
- `internal/platform/opencode.go`
- `internal/platform/platform.go`

### Verification

The combined focused package suite passed after integration:

- `go test ./commands ./internal/config ./internal/platform -count=1 -timeout 60s`

Earlier worker-local focused runs also passed for:

- OpenCode native plugin emission/cleanup
- add/remove plugin lifecycle edges
- `.agentsrc.json` plugin round-tripping and install support

### Decisions Locked In

- `.agentsrc.json` plugin support is now implemented and should remain part of the first-class plugin model.
- OpenCode plugin support is the first live platform emitter and import target.
- OpenCode `.opencode/package.json` emission was intentionally not implemented yet because the current canonical model does not define enough data to emit it cleanly.
- Package-platform plugin emitters are still deferred until their manifest/output mapping is nailed down more precisely.

### Remaining Work

Still not implemented:

- Claude package plugin emission
- Cursor package plugin emission
- Codex package plugin emission
- Copilot package plugin emission
- Marketplace manifest generation
- Broader package-platform plugin import logic
- Bash parity for plugins

The immediate next best step is Phase 5D/5E:

1. define/render package-platform plugin outputs for Claude, Cursor, Codex, and Copilot
2. add marketplace generation from canonical plugin metadata
3. extend health/import tooling for package-plugin trees once those emitters exist

### Status Change

This handoff should now be treated as:

- **Status:** In progress
- Plugin rollout has begun and already spans canonical storage, OpenCode emission, command lifecycle, reverse import for OpenCode plugin files, and manifest portability.

---

## Update — 2026-03-30

Package-platform plugin support is now implemented for the remaining Phase 5D/5E targets.

### What Changed

- Added shared package-plugin helpers in `internal/platform/package_plugins.go`:
  - preferred-scope package plugin selection
  - shared overlay/symlink tree sync and prune
  - helper accessors for canonical plugin directories and metadata
- Added shared marketplace rendering helpers in `internal/platform/plugin_marketplaces.go`.
- Implemented Claude package plugin emission in `internal/platform/claude.go`:
  - emits `.claude-plugin/plugin.json`
  - emits `.claude-plugin/marketplace.json`
  - emits managed `commands/`, `agents/`, `skills/`, `hooks/`, and `.mcp.json` when present
  - overlays `files/` and `platforms/claude/`
- Implemented Cursor package plugin emission in `internal/platform/cursor.go`:
  - emits `.cursor-plugin/plugin.json`
  - emits `.cursor-plugin/marketplace.json`
  - emits managed `rules/`, `commands/`, `agents/`, `skills/`, `hooks/`, and `mcp.json` when present
  - overlays `files/` and `platforms/cursor/`
- Implemented Codex package plugin emission in `internal/platform/codex.go`:
  - emits `.codex-plugin/plugin.json`
  - emits `.agents/plugins/marketplace.json`
  - emits managed `skills/` plus repo-root plugin-owned files from `files/` and `platforms/codex/`
  - supports documented Codex package fields like `hooks`, `mcpServers`, and `apps` when the emitted files exist
- Implemented Copilot package plugin emission in `internal/platform/copilot.go`:
  - emits root `plugin.json`
  - emits `.github/plugin/marketplace.json`
  - emits managed `agents/`, `skills/`, and `commands/`
  - overlays `files/` and `platforms/copilot/`
- Added focused test coverage for package-platform emitters and shared helpers:
  - `internal/platform/package_plugins_test.go`
  - `internal/platform/package_plugins_helpers_test.go`
  - `internal/platform/codex_test.go`
  - `internal/platform/copilot_plugin_test.go`

### Files Added

- `internal/platform/package_plugins.go`
- `internal/platform/plugin_marketplaces.go`
- `internal/platform/package_plugins_test.go`
- `internal/platform/package_plugins_helpers_test.go`
- `internal/platform/copilot_plugin_test.go`

### Files Changed

- `internal/platform/claude.go`
- `internal/platform/cursor.go`
- `internal/platform/codex.go`
- `internal/platform/codex_test.go`
- `internal/platform/copilot.go`

### Verification

Focused verification passed after integration:

- `go test ./internal/platform -count=1`
- `go test ./commands ./internal/config ./internal/platform -count=1 -timeout 60s`

Worker-local focused runs also passed during implementation for:

- Claude and Cursor package plugin emitters
- Codex package plugin emitter
- Copilot package plugin emitter

### Decisions Locked In

- Package-platform plugins use the same canonical `PLUGIN.yaml` bundle model as OpenCode, but emit vendor-specific package trees and marketplaces.
- Package-plugin emission is conservative:
  - use the preferred scope
  - emit only when exactly one package plugin bundle targets that platform
  - clean up managed output when there are zero or ambiguous bundles
- Marketplace generation is now part of the package-platform emitter wave, not a separate speculative future design.
- Copilot marketplace output is rooted at `.github/plugin/marketplace.json`.
- Codex marketplace output is rooted at `.agents/plugins/marketplace.json`.

### Remaining Work

Still not implemented:

- Package-platform plugin import/adoption hardening beyond the current OpenCode path
- Reverse `refresh` mapping for package-platform plugin outputs back into canonical bundles
- Rich package-platform plugin visibility in `status` and `doctor`
- Bash parity for plugin support
- Optional authoring UX such as `dot-agents plugins new --platform ...`

The next best implementation slice is now Phase 5F and later:

1. add package-platform import and refresh normalization
2. extend `status` and `doctor` for package plugin outputs
3. add bash parity once Go behavior settles
4. add scaffolding/authoring UX after that

### Status Change

This handoff should still be treated as:

- **Status:** In progress
- Plugin rollout now covers canonical storage, OpenCode native plugins, `.agentsrc.json` portability, and package-plugin emission plus marketplace generation for Claude, Cursor, Codex, and Copilot.

---

## Update — 2026-03-31

Sub-agent partition plan for the next plugin wave (Phase 5F and Phase 5G prep) is now defined.

### Baseline Check

- Current focused suite passes before additional changes:
  - `go test ./commands ./internal/platform ./internal/config -count=1`
- Current major gaps confirmed in code:
  - package-platform plugin reverse mapping is not implemented in `commands/import.go` and `commands/refresh.go`
  - package-plugin diagnostics are not yet surfaced in `commands/status.go` and `commands/doctor.go`
  - bash command parity for plugin bucket is still missing

### Team Partition

#### Coordinator (integration owner)

Owned files:

- `commands/import.go`
- `commands/refresh.go`
- `commands/status.go`
- `commands/doctor.go`
- `commands/explain.go`
- `commands/*_test.go` for integration-only gaps
- `.agents/active/platform-dir-unification.plan.md`

Scope:

- define final plugin reverse-map rules and acceptance behavior
- enforce no-collision ownership across worker lanes
- integrate worker branches in order and resolve conflicts in coordinator-owned files only
- update plan status for Phase 5E/5F/5G once merged

#### Worker A: Package import + refresh normalization

Owned files:

- `commands/import.go`
- `commands/import_test.go`
- `commands/refresh.go`
- `commands/refresh_test.go`

Scope:

- add conservative package-platform canonicalization for representable paths:
  - `.claude-plugin/...`
  - `.cursor-plugin/...`
  - `.codex-plugin/...`
  - repo-root `plugin.json` plus `.github/plugin/marketplace.json` for Copilot
  - `.agents/plugins/marketplace.json` for Codex
- keep existing non-lossy fallback behavior when source shapes are ambiguous
- add reverse `refresh` normalization for emitted package-plugin trees back to canonical:
  - `plugins/<scope>/<name>/resources/...`
  - `plugins/<scope>/<name>/files/...`
  - `plugins/<scope>/<name>/platforms/<platform>/...`

Out of scope:

- `status`/`doctor` rendering changes
- bash parity

Acceptance:

- new table-driven tests for each package platform mapping direction
- no regressions in existing OpenCode plugin mapping tests

#### Worker B: Plugin status + doctor observability

Owned files:

- `commands/status.go`
- `commands/status_test.go`
- `commands/doctor.go`
- new `commands/doctor_test.go` (if introduced)

Scope:

- include package-platform plugin outputs in health counts and audit views:
  - `.claude-plugin/`
  - `.cursor-plugin/`
  - `.codex-plugin/`
  - `.agents/plugins/marketplace.json`
  - repo-root `plugin.json`
  - `.github/plugin/marketplace.json`
- expand broken-link detection to include plugin-managed trees for package platforms
- keep OpenCode plugin diagnostics unchanged and additive

Out of scope:

- changing platform emitter semantics
- import/refresh mapping logic

Acceptance:

- status tests for package-plugin tree counting and marketplace visibility
- doctor tests for broken package-plugin symlink detection and healthy counts

#### Worker C: Bash parity for plugin lifecycle

Owned files:

- `src/lib/commands/import.sh`
- `src/lib/commands/refresh.sh`
- `src/lib/commands/status.sh`
- `src/lib/commands/doctor.sh`
- `src/lib/commands/remove.sh`
- `src/lib/utils/resource-restore-map.sh`

Scope:

- bring bash command behavior to parity with Go plugin mappings and lifecycle
- add plugin reverse-map rules in shell restore mapping
- include plugin trees in bash status and doctor flows
- ensure `remove --clean` keeps plugin cleanup behavior consistent with Go command surface

Out of scope:

- Go command changes
- platform Go emitter changes

Acceptance:

- dry-run verification on plugin-rich fixture repo
- no regressions for existing non-plugin bash flows

#### Worker D: Platform regression and fixture hardening

Owned files:

- `internal/platform/*_test.go` plugin-focused tests only
- `commands/*_test.go` fixture helpers shared by plugin tests only

Scope:

- add targeted regression fixtures for multi-plugin ambiguity and fallback behavior
- add transition tests for cleanup when package plugin source disappears or changes scope
- validate marketplace generation remains deterministic when plugin order changes

Out of scope:

- production Go logic unless a failing test proves a defect and coordinator approves patch

Acceptance:

- deterministic, table-driven tests that reproduce all edge cases listed in Phase 5F notes

### Merge and Execution Order

1. Worker A lands import/refresh mapping.
2. Worker B rebases on A and lands status/doctor visibility.
3. Worker D lands final regression hardening after A+B are in.
4. Worker C (bash parity) lands after Go behavior stabilizes from A+B+D.
5. Coordinator updates active plan statuses and final handoff summary.

### Risk Controls

- Do not relax conservative import policy: ambiguous package layouts must fallback, not guess.
- Preserve OpenCode plugin behavior as the baseline reference path.
- Keep package plugin support "single selected bundle per platform target" unless a deliberate contract change is approved.
- Avoid broad edits outside owned file sets to reduce merge churn in the existing dirty tree.

---

## Update — 2026-03-31 (Execution Progress)

Worker A and Worker B partitions have now been executed and integrated locally.

### Completed This Round

- Worker A (package import/refresh normalization lane):
  - implemented package-platform plugin import canonicalization in `commands/import.go`
  - added package-plugin import coverage in `commands/import_test.go`
  - extended plugin mapping coverage in `commands/refresh_test.go`
  - touched `commands/refresh.go` for plugin pass-through/mapping alignment
- Worker B (status/doctor observability lane):
  - extended package-plugin health and audit visibility in `commands/status.go`
  - extended broken-link and healthy-link accounting for package-plugin trees in `commands/doctor.go`
  - added regression tests in `commands/status_test.go` and `commands/doctor_test.go`

### Verification This Round

- `go test ./commands -count=1`
- `go test ./commands ./internal/platform ./internal/config -count=1`
- `go test ./... -count=1`

### Remaining Partition Work

- Worker C: bash parity for plugin lifecycle and reverse mapping
- Worker D: additional fixture/edge-case hardening after bash parity or as a follow-up tightening pass

---

## Update — 2026-03-31 (Worker D then Worker C)

Execution order was adjusted to stabilize Go behavior first: Worker D completed before Worker C.

### Worker D Completion

- Added additional regression hardening in test lanes:
  - conservative package-plugin import fallback coverage
  - deterministic package-plugin selection/ambiguity coverage
  - transition cleanup coverage when package plugin source disappears
- Validation passed:
  - `go test ./commands ./internal/platform -count=1`

### Worker C Completion

- Landed conservative bash parity for plugins:
  - restore map now supports OpenCode plugin file canonicalization and canonical `plugins/...` pass-through
  - import scanning now includes `.opencode/plugins`
  - status/doctor now surface plugin roots additively
  - remove `--clean` now includes `~/.agents/plugins/<project>/`
- Validation passed:
  - `bash -n` on updated shell files
  - `go test ./commands ./internal/platform ./internal/config -count=1`
  - `go test ./... -count=1`

### Remaining Known Gap

- Bash still intentionally avoids lossy package-plugin manifest synthesis; package import remains conservative in shell.
