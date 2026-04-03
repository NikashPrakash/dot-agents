# Plugin Support Strategy

This document expands plugin support planning beyond OpenCode.

Official docs checked on 2026-03-30:

- Cursor plugins: https://cursor.com/docs/plugins.md and https://cursor.com/marketplace/
- Claude Code plugins: https://code.claude.com/docs/en/plugins.md and https://code.claude.com/docs/en/plugin-marketplaces.md
- Codex plugins: https://developers.openai.com/codex/plugins/ and https://developers.openai.com/codex/plugins/build
- GitHub Copilot CLI plugins: https://docs.github.com/en/copilot/concepts/agents/copilot-cli/about-cli-plugins, https://docs.github.com/en/enterprise-cloud@latest/copilot/how-tos/copilot-cli/customize-copilot/plugins-creating, and https://docs.github.com/en/enterprise-cloud@latest/copilot/how-tos/copilot-cli/customize-copilot/plugins-marketplace
- OpenCode plugins: https://opencode.ai/docs/plugins/

## Summary

There is no single cross-platform vendor plugin manifest.

- Cursor has first-party plugin packages and marketplace support.
- Claude Code has first-party plugin packages and plugin marketplaces.
- Codex has first-party plugin packages and repo or personal marketplaces.
- OpenCode has native local plugin files plus config-declared npm plugins.
- GitHub Copilot CLI has installable plugin packages with repo-aware manifests, plus marketplace, repo, and local-path installation.

But there is a useful shared abstraction:

- a plugin has identity and metadata
- a plugin bundles resources such as agents, skills, commands, hooks, MCP servers, assets, and scripts
- a plugin may need platform-specific metadata or files
- a plugin emits into platform-specific runtime or package layouts

Because of that, `dot-agents` should not force all plugin support into one vendor-shaped config file. It should use one canonical plugin bundle model with per-platform emitters and importers.

## Recommended Product Framing

Treat `plugins` as one first-class resource family with two plugin kinds:

1. `native`
2. `package`

Use them like this:

- OpenCode: `native`
- Claude Code: `package`
- Cursor: `package`
- Codex: `package`
- GitHub Copilot CLI: `package`

This gives `dot-agents` one coherent plugin bucket without pretending the vendor manifests are interchangeable.

## Canonical Storage Model

Use one canonical top-level bucket:

```text
~/.agents/plugins/<scope>/<plugin-name>/
  PLUGIN.yaml
  resources/
    agents/
    skills/
    commands/
    hooks/
    mcp/
  files/
  platforms/
    claude/
    cursor/
    codex/
    copilot/
    opencode/
```

### Directory Roles

- `PLUGIN.yaml`: canonical plugin manifest owned by `dot-agents`
- `resources/`: canonical reusable resources that can be emitted into plugin bundles
- `files/`: generic plugin-owned source files such as scripts, assets, templates, or code
- `platforms/<platform>/`: platform-specific overrides, extra files, or passthrough content that cannot be cleanly represented in the shared resource model

This is intentionally similar in spirit to `HOOK.yaml`:

- `dot-agents` owns the canonical manifest
- platforms get emitted native formats
- import only normalizes what is representable
- non-representable vendor details stay in platform override space instead of corrupting the shared model

## Canonical Manifest

`PLUGIN.yaml` should describe the plugin bundle in canonical terms, not mirror any one vendor schema.

Example:

```yaml
kind: package
name: review-toolkit
version: 0.1.0
display_name: Review Toolkit
description: Review helpers for code and PR workflows.
authors:
  - Nikash Prakash
platforms:
  - claude
  - cursor
  - codex
resources:
  agents:
    - reviewer
  skills:
    - review-pr
  commands:
    - pr-summary
  hooks:
    - lint-on-edit
  mcp:
    - github
marketplace:
  repo: https://github.com/example/review-toolkit
  tags:
    - review
    - git
platform_overrides:
  codex:
    app_id: review-toolkit
  claude:
    permissions: []
```

### Suggested Schema

Required:

- `kind`: `native` or `package`
- `name`
- `platforms`

Strongly recommended:

- `version`
- `display_name`
- `description`

Optional:

- `authors`
- `homepage`
- `license`
- `resources`
- `marketplace`
- `dependencies`
- `platform_overrides`

### Shared Concepts Worth Normalizing

These are the commonalities across vendors that justify a canonical manifest:

- plugin identity and metadata
- supported platforms
- bundled resources
- plugin-owned source files
- marketplace-facing metadata
- dependency declarations
- platform overrides

The canonical model should stop there. It should not try to invent one universal replacement for vendor `plugin.json` files.

## Emission Model

Plugin support should work like other `dot-agents` buckets:

- users author canonical plugin bundles under `~/.agents/plugins/...`
- platform emitters generate the correct native plugin shape for each platform
- `refresh`, `import`, `status`, `doctor`, and `explain` reason over the canonical plugin bundle, not only the emitted files

### Package Platforms

For Claude, Cursor, Codex, and Copilot:

- emit the vendor plugin manifest from `PLUGIN.yaml`
- emit bundled resources from `resources/`
- copy or link plugin-owned files from `files/`
- merge in `platforms/<platform>/` overrides as needed

Expected outputs:

- Claude: `.claude-plugin/plugin.json` and optional `.claude-plugin/marketplace.json`
- Cursor: `.cursor-plugin/plugin.json` and optional `.cursor-plugin/marketplace.json`
- Codex: `.codex-plugin/plugin.json` and optional `.agents/plugins/marketplace.json`
- Copilot: `plugin.json` in a supported plugin root, plus optional `marketplace.json`

### Native Platform

For OpenCode:

- emit runtime plugin source files to `.opencode/plugins/`
- optionally emit `.opencode/package.json`
- preserve npm plugin activation in `settings/{scope}/opencode.json`

OpenCode is still a plugin resource, but its emitter is runtime-oriented rather than package-oriented.

## Marketplace Model

Marketplace metadata should not be a separate top-level canonical bucket.

Instead:

- store marketplace metadata inside each plugin's `PLUGIN.yaml`
- generate vendor marketplace manifests by scanning plugin bundles for a given scope

That avoids unnecessary sprawl and keeps marketplace registration attached to the plugin it describes.

Examples:

- Claude: generate `.claude-plugin/marketplace.json`
- Cursor: generate `.cursor-plugin/marketplace.json`
- Codex: generate `.agents/plugins/marketplace.json`
- Copilot: generate `marketplace.json` in the chosen plugin or repo authoring root

## Import Model

`import` should be conservative.

Safe import targets:

- representable plugin manifests
- representable bundled resources
- representable marketplace metadata
- platform-specific files that can be preserved under `platforms/<platform>/`

Do not force lossy reverse-mapping into `PLUGIN.yaml`.

Import policy:

- normalize shared concepts into `PLUGIN.yaml`
- preserve unknown vendor-specific details under `platforms/<platform>/`
- fall back to a mostly opaque package tree when a plugin repo cannot be cleanly decomposed

The hard part of plugin support is reverse-import, not forward emission.

## Direct Platform Import

It should be possible to adopt an existing platform-native plugin layout directly into the canonical plugin bucket.

This should mean:

- read a local platform-native plugin source tree
- normalize representable shared concepts into `plugins/<scope>/<plugin-name>/PLUGIN.yaml`
- preserve vendor-only details under `platforms/<platform>/`
- preserve plugin-owned files under `files/`
- refuse or fall back when the source is clearly an installed cache, generated output, or otherwise lossy to import

Recommended command shapes:

- `dot-agents import plugin --platform claude --from /path/to/plugin --scope project`
- `dot-agents import plugin --platform cursor --from /path/to/plugin --scope global`
- `dot-agents import plugin --platform opencode --name my-plugin --scope project`

Preferred rollout:

1. path-first import from an explicit local plugin root
2. platform-located import from known local development paths
3. no automatic broad discovery across installed plugin caches in V1

Command boundary:

- `refresh` is for re-emitting canonical plugin bundles already owned by `dot-agents`
- `import` is for adopting external native plugin layouts into canonical storage

## Cross-Cutting Rollout Surface

Plugin support will touch more than platform emitter files.

### Canonical Storage and Schema

- `init` must create and document the `plugins/` bucket
- shared plugin descriptor loading and validation must exist alongside the current shared resource helpers
- `.agentsrc.json` needs an explicit plugin story once plugins are meant to be portable through manifests

### Adoption and Reverse Mapping

- `add` must detect plugin-native repo roots and preview how they will be adopted
- `import` must detect and normalize supported plugin layouts
- `refresh` restore maps must understand emitted plugin paths and normalize them back into canonical plugin bundles

### Platform Emission

- OpenCode needs native runtime plugin emission
- Claude, Cursor, Codex, and Copilot need package plugin emission
- package platforms also need marketplace generation

### Diagnostics and Lifecycle

- `status` must show plugin bundle health and emitted plugin outputs
- `doctor` must validate plugin bundle health and repair emitted plugin outputs where possible
- `remove` and `--clean` must include plugin project directories when project-scoped plugins exist
- `explain` must document the plugin bucket and the canonical `PLUGIN.yaml` model

### Manifest Portability

- `.agentsrc.json` must carry plugin declarations as part of first-class plugin support
- `install` must understand plugin declarations
- `install --generate` must detect plugin bundles in canonical storage and write them into `.agentsrc.json`

This means plugin support should be tracked as a command/config/platform rollout, not only as emitter work.

## Platform Plan

### OpenCode

Goal:

- Manage native plugin files and dependency manifests as canonical plugin bundles.

Canonical representation:

- `plugins/<scope>/<plugin-name>/PLUGIN.yaml` with `kind: native`
- `plugins/<scope>/<plugin-name>/files/...` for runtime plugin source
- optional dependency metadata rendered into `.opencode/package.json`

Planned scope:

- emit `.opencode/plugins/<plugin-name>/...`
- optionally emit `.opencode/package.json`
- preserve npm plugin activation in `settings/{scope}/opencode.json`

### Claude Code

Goal:

- Support repo-aware plugin package authoring and marketplace metadata from canonical plugin bundles.

Canonical representation:

- `plugins/<scope>/<plugin-name>/PLUGIN.yaml` with `kind: package`
- `plugins/<scope>/<plugin-name>/resources/...`
- `plugins/<scope>/<plugin-name>/platforms/claude/...`

Outputs:

- `.claude-plugin/plugin.json`
- optional `.claude-plugin/marketplace.json`

### Cursor

Goal:

- Support repo-aware plugin package authoring and marketplace metadata from canonical plugin bundles.

Canonical representation:

- `plugins/<scope>/<plugin-name>/PLUGIN.yaml` with `kind: package`
- `plugins/<scope>/<plugin-name>/resources/...`
- `plugins/<scope>/<plugin-name>/platforms/cursor/...`

Outputs:

- `.cursor-plugin/plugin.json`
- optional `.cursor-plugin/marketplace.json`

### Codex

Goal:

- Support repo-aware plugin package authoring and marketplace metadata from canonical plugin bundles.

Canonical representation:

- `plugins/<scope>/<plugin-name>/PLUGIN.yaml` with `kind: package`
- `plugins/<scope>/<plugin-name>/resources/...`
- `plugins/<scope>/<plugin-name>/platforms/codex/...`

Outputs:

- `.codex-plugin/plugin.json`
- optional `.agents/plugins/marketplace.json`

### GitHub Copilot CLI

Goal:

- Support repo-aware plugin package authoring and marketplace metadata without managing the installed cache under `~/.copilot/state/installed-plugins/`.

Canonical representation:

- `plugins/<scope>/<plugin-name>/PLUGIN.yaml` with `kind: package`
- `plugins/<scope>/<plugin-name>/resources/...`
- `plugins/<scope>/<plugin-name>/platforms/copilot/...`

Outputs:

- `plugin.json` in a supported authoring root
- optional `marketplace.json`

Keep installed-cache management out of scope.

## Rollout Phases

### Phase A: Shared Plugin Contract

- add canonical `plugins/<scope>/<plugin-name>/` support
- define `PLUGIN.yaml` schema
- add shared plugin descriptor loading and validation
- extend `init`, `status`, and `explain` to surface the new bucket
- define `.agentsrc.json` plugin representation as part of the shared contract

### Phase B: OpenCode Native Plugin Support

- emit `.opencode/plugins/<plugin-name>/...`
- render optional `.opencode/package.json`
- extend `refresh`, `import`, `doctor`, and `remove`
- validate plugin-file and dependency-manifest round-tripping

OpenCode goes first because it maps most directly to current link and render patterns.

### Phase C: Package Plugin Emitter Support

- add package emitters for Claude, Cursor, Codex, and Copilot
- render vendor manifests from `PLUGIN.yaml`
- emit bundled `resources/`
- merge platform override files

Initial simplification:

- assume one package plugin per emitted repo target in V1

### Phase D: Manifest Portability

- extend `.agentsrc.json` with plugin declarations
- teach `install` to resolve canonical plugin bundles from sources
- teach `install --generate` to detect plugin bundles and write them into the manifest
- keep plugin manifest support in the first-class rollout, not as a future add-on
- land this immediately after the first canonical and emitter pass so commit size stays reviewable

### Phase E: Marketplace Generation

- generate vendor marketplace manifests by scanning plugin bundles in scope
- add marketplace visibility to `status` and `explain`
- validate multi-plugin ordering and filtering rules

### Phase F: Import and Refresh Hardening

- import representable plugin repos back into canonical bundles
- preserve unknown vendor details in `platforms/<platform>/`
- add fallback behavior for non-representable plugin repos
- add regression coverage for plugin import heuristics

### Phase G: Authoring UX

- add scaffolding such as `dot-agents plugins new --platform <platform>`
- add example templates for `native` and `package` plugins
- add docs for common workflows

## Suggested Delivery Order

1. Shared plugin contract
2. OpenCode native plugin support
3. Claude and Cursor package plugin support
4. Codex package plugin support
5. Copilot package plugin support
6. Manifest portability through `.agentsrc.json` and `install`, immediately after the first canonical/emitter pass
7. Marketplace generation across package platforms
8. Scaffolding and authoring UX

That order keeps the rollout aligned with the current Go-first architecture while reducing early import complexity.

## Acceptance Criteria

Broad plugin support is credible when:

- `plugins` is a first-class canonical bucket under `~/.agents`
- `PLUGIN.yaml` expresses shared plugin concepts without imitating one vendor schema
- `.agentsrc.json` can declare plugin bundles and `install` can resolve them from sources
- OpenCode native plugins are manageable through `dot-agents`
- Claude, Cursor, Codex, and Copilot plugin packages are authorable from canonical plugin bundles
- marketplace manifests are generated from plugin metadata rather than managed as unrelated loose files
- `status`, `refresh`, `import`, `doctor`, and `explain` understand plugin bundles
- non-representable vendor details can be preserved without corrupting the canonical model
