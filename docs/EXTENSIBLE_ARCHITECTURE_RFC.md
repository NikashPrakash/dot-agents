# Extensible Architecture RFC

Status: Draft

Owner: dot-agents

Last updated: 2026-03-30

## Summary

`dot-agents` should evolve from a collection of platform-specific link writers into a small compiler:

- canonical agent resources in
- project selection rules applied
- platform-native artifacts planned
- links or rendered files emitted

This RFC proposes one shared internal architecture for current resource families:

- `rules`
- `skills`
- `agents`
- `hooks`
- `mcp`
- `settings`
- `plugins`

It is also designed to support future families without requiring invasive changes across `refresh`, `import`, `status`, `doctor`, and every platform implementation.

The core shift is:

- author resources canonically where possible
- adapt legacy file layouts into the same internal model
- centralize resolution and planning
- make platforms responsible only for emitting artifacts for resource kinds they support

## Motivation

The repo currently mixes two architectural styles.

The older style is direct platform wiring:

- commands trigger platform implementations
- platforms inspect `~/.agents`
- platforms create links or rendered files directly
- import and refresh know many resource-family and path-specific rules

The newer style is canonical modeling:

- hooks have an explicit shared contract and renderer model
- plugins are already being designed as canonical bundles with platform emitters

That split is manageable today, but it will become expensive as the product grows.

The current cost centers are:

1. Adding a new resource family requires touching multiple commands and platforms.
2. Project manifests are tied to a fixed list of known families.
3. Import and status reason about emitted files more than logical resources.
4. Platform implementations own too much orchestration logic.
5. There is no single artifact plan that all commands can share.

## Existing State

### What is already working well

- `~/.agents` is the right canonical home for user-managed resources.
- Scope precedence is already simple and understandable: project over global.
- Platform-specific transport is already acknowledged as a separate concern:
  - symlink
  - hard link
  - rendered file
- Hooks already introduced the right vocabulary for representability and emission modes.
- Plugin planning already recognizes that there is no universal vendor schema and that `dot-agents` needs canonical bundles plus per-platform emitters.

### What currently limits extensibility

The current `Platform` interface is too coarse:

```go
type Platform interface {
    ID() string
    DisplayName() string
    IsInstalled() bool
    Version() string
    CreateLinks(project, repoPath string) error
    RemoveLinks(project, repoPath string) error
    HasDeprecatedFormat(repoPath string) bool
    DeprecatedDetails(repoPath string) string
}
```

This pushes resource discovery, precedence, selection, rendering, and artifact writing into each platform implementation.

The project manifest is also fixed to known families:

```json
{
  "skills": [],
  "rules": [],
  "agents": [],
  "hooks": true,
  "mcp": true,
  "settings": true
}
```

That shape is workable for the current set, but it does not age well as `plugins`, commands, templates, histories, or other families are added.

## Goals

- Make new resource families cheap to add.
- Keep one logical source of truth per resource.
- Support canonical bundles where they add value, without forcing every family into the same authoring shape immediately.
- Separate resource resolution from platform emission.
- Let `refresh`, `status`, `doctor`, `import`, and `explain` share one internal plan model.
- Preserve backward compatibility with the current store and emitted layouts during migration.

## Non-Goals

- This RFC does not require replacing every current on-disk format immediately.
- This RFC does not require shell implementation parity before the Go implementation lands.
- This RFC does not require every platform to support every resource family.
- This RFC does not define a networked registry or remote marketplace service.

## Design Principles

1. Canonical where it helps, adapters where it does not.
2. Storage is distinct from emission.
3. Representability is a renderer concern, not a storage concern.
4. Artifact planning must be explicit and inspectable.
5. Import should be conservative and preserve vendor-specific details rather than normalizing them incorrectly.
6. Platform transports such as hard links are implementation details, not user-facing resource semantics.

## Proposed Architecture

### Mental Model

Treat `dot-agents` as a compiler with four stages:

1. Load
2. Select
3. Plan
4. Apply

That becomes the shared pipeline for all user-facing commands.

### Stage 1: Load

Load resources from one or more sources:

- local `~/.agents`
- cached git sources
- future source types if added later

Each family gets a loader that can read its canonical form and any supported legacy form.

Examples:

- `rules`: plain files adapted into resources
- `skills`: directories containing `SKILL.md`
- `agents`: directories containing `AGENT.md`
- `hooks`: `HOOK.yaml` bundles and legacy platform files
- `plugins`: `PLUGIN.yaml` bundles

### Stage 2: Select

Resolve what a project wants after combining:

- scope precedence
- source precedence
- project manifest selectors

The selector stage should produce a project-level catalog of logical resources. This catalog is platform-neutral.

### Stage 3: Plan

Ask each platform emitter to translate selected resources into planned artifacts.

Examples of artifacts:

- symlink `AGENTS.md`
- hard link `.cursor/rules/global--rules.mdc`
- render `.codex/hooks.json`
- render `.claude-plugin/plugin.json`
- symlink `.agents/skills/review-pr`

Every artifact in the plan should record:

- which resource produced it
- which emitter produced it
- where it will be written
- how it will be written
- whether it is managed, replaceable, or stale

### Stage 4: Apply

Apply the artifact plan:

- create missing directories
- write rendered files
- create or refresh symlinks
- create or refresh hard links
- remove stale managed artifacts

This stage should be dumb. It should not contain resource-family knowledge.

## Core Domain Model

The exact types can change, but the system needs stable equivalents of these concepts.

### Resource

`Resource` is the shared logical unit that platform emitters consume.

Suggested shape:

```go
type Resource struct {
    Kind         string
    Name         string
    Scope        string
    Source       SourceRef
    Canonical    bool
    Metadata     map[string]any
    Files        []FileRef
    Overrides    map[string]map[string]any
    Capabilities []string
}
```

Examples:

- a skill named `review-pr`
- an agent named `reviewer`
- a hook named `format-write`
- a plugin named `review-toolkit`

### Resource Kind

`Kind` is the family identifier.

Initial kinds:

- `rule`
- `skill`
- `agent`
- `hook`
- `mcp_server`
- `setting`
- `plugin`

This should be an open set. New kinds should not require structural changes to the whole system.

### SourceRef

Tracks where a resource came from:

- local store
- git cache
- plugin-owned nested resource
- imported legacy file

This is important for explainability and import/export tooling.

### Artifact

`Artifact` is the platform-native output of planning.

Suggested shape:

```go
type Artifact struct {
    PlatformID   string
    ResourceRef  ResourceRef
    TargetPath   string
    Mode         ArtifactMode
    RenderedBody []byte
    SourcePath   string
    ManagedBy    string
}
```

`ArtifactMode` should generalize what hooks already introduced:

- `symlink`
- `hardlink`
- `write`
- optionally `copy` if a future family needs physical file duplication

### Support Result

Renderers should report whether a resource is representable on a platform:

```go
type SupportResult struct {
    Supported bool
    Required  bool
    Reason    string
}
```

This gives `doctor`, `status`, and `refresh` one consistent way to explain skips, warnings, and hard failures.

## Family Adapters

Not every family needs the same on-disk authoring shape right away. The internal architecture should support both canonical bundles and adapted legacy formats.

### Rules

Current state:

- plain files under `rules/<scope>/`

Near-term approach:

- adapt each rule file into a `rule` resource
- preserve the current authoring format

Future option:

- optional `RULE.yaml` bundle only if there is a real need for metadata, representability flags, or rule-local assets

### Skills

Current state:

- directories with `SKILL.md`

Approach:

- keep the current authoring model
- adapt each skill directory into a `skill` resource
- derive metadata from frontmatter where present

### Agents

Current state:

- directories with `AGENT.md`

Approach:

- keep the current authoring model
- adapt each agent directory into an `agent` resource
- derive metadata from frontmatter where present

### Hooks

Current state:

- partly canonical already

Approach:

- keep `HOOK.yaml` as the target canonical model
- support legacy platform-specific hook files during migration
- make hooks the first full user of the generic planner and artifact model

### MCP

Current state:

- platform-specific config files are normalized loosely

Approach:

- introduce a logical `mcp_server` resource model
- allow canonical server definitions plus platform override fields
- let emitters render aggregate config files for platforms that need them

### Settings

Current state:

- largely platform-shaped files

Approach:

- keep settings conservative
- split into two categories:
  - canonical settings that map to shared concepts
  - passthrough platform settings that remain vendor-shaped

Not all settings should be normalized. Many should remain platform-owned passthrough files referenced by the catalog.

### Plugins

Current state:

- design work exists, implementation is still in progress

Approach:

- implement plugins as canonical bundles
- allow nested plugin-owned resources
- use plugin emitters to produce vendor package layouts
- preserve platform-specific details under `platforms/<platform>/`

## Package Boundaries

The current `internal/platform` package contains both orchestration and emission logic. That should be split so responsibilities are easier to extend and test.

Suggested package structure:

```text
internal/
  catalog/
    catalog.go
    resolver.go
    selectors.go
  resource/
    types.go
    family.go
    registry.go
  source/
    source.go
    local.go
    git.go
  families/
    rules/
    skills/
    agents/
    hooks/
    mcp/
    settings/
    plugins/
  emit/
    artifact.go
    planner.go
    applier.go
  platform/
    registry.go
    cursor/
    claude/
    codex/
    opencode/
    copilot/
```

### Responsibilities

`resource/`

- shared resource and selector types
- family registration

`source/`

- loading from local and git-backed stores
- source precedence

`families/`

- family-specific loading
- importers for canonical and legacy forms
- family-specific validation

`catalog/`

- merge loaded resources from sources and scopes
- apply project manifest selectors
- produce the project resource catalog

`emit/`

- artifact planning
- artifact diffing
- artifact application

`platform/`

- platform capability registration
- family emitters grouped per platform

## Emitter Model

Platforms should stop owning the full refresh flow. Instead, each platform contributes emitters for supported resource kinds.

Suggested interfaces:

```go
type Family interface {
    Kind() string
    Load(store Store, scopes []string) ([]resource.Resource, error)
    Import(projectPath string) ([]resource.Resource, error)
}

type Emitter interface {
    PlatformID() string
    Kind() string
    Supports(resource.Resource, ProjectContext) SupportResult
    Plan(resource.Resource, ProjectContext) ([]emit.Artifact, error)
}
```

The planner then does:

1. iterate selected resources
2. find emitters for `(platform, kind)`
3. ask if the resource is representable
4. collect artifacts
5. report unsupported required resources cleanly

This is materially simpler than every platform re-discovering resources from disk.

## Project Manifest Evolution

The current `.agentsrc.json` shape hardcodes known families. That should be preserved for backward compatibility, but a generic selector model should be added as v2.

### Proposed Manifest Direction

```json
{
  "$schema": "https://dot-agents.dev/schemas/agentsrc-v2.json",
  "version": 2,
  "project": "myproject",
  "sources": [
    { "type": "local" }
  ],
  "resources": [
    { "family": "rules", "select": { "scopes": ["global", "project"] } },
    { "family": "skills", "select": { "names": ["deploy", "review-pr"] } },
    { "family": "agents", "select": { "names": ["reviewer"] } },
    { "family": "hooks", "select": { "all": true } },
    { "family": "mcp", "select": { "names": ["github", "filesystem"] } },
    { "family": "plugins", "select": { "names": ["review-toolkit"] } }
  ]
}
```

### Why this matters

- new families can be added without changing the manifest top level
- selectors can evolve independently per family
- family-specific selection rules can be supported without schema contortions

### Compatibility

For a migration period:

- `version: 1` manifests continue to load
- v1 fields are translated into v2 selectors internally
- generated manifests can stay on v1 until plugins or other new families need v2

## Import Model

Import should operate on resources, not just destination paths.

### Current Pain

Import and refresh currently carry many path-specific translation rules. That is brittle and hard to extend.

### Proposed Direction

Each family owns import normalization:

- rules importer
- skills importer
- agents importer
- hooks importer
- mcp importer
- settings importer
- plugins importer

Importers should emit:

- normalized resources where safe
- vendor-specific passthrough files where normalization would be lossy
- warnings when a resource cannot be represented canonically

This keeps import logic close to the family that understands it.

## Status, Doctor, And Explain

These commands should stop inferring meaning only from filesystem state and instead reuse the same planner outputs.

### Status

`status` should be able to answer:

- which logical resources are selected for this project
- which platforms support each resource
- which artifacts were planned
- which artifacts exist and are healthy
- which artifacts are stale or drifted

### Doctor

`doctor` should be able to answer:

- which required resources are unsupported on a chosen platform
- which rendered artifacts cannot be refreshed safely
- which local store entries are malformed
- which emitted files drift from the current plan

### Explain

`explain` should be able to answer:

- why a resource was selected
- where it came from
- why it mapped to a given artifact
- why a platform skipped it

This is only practical if commands share a single catalog and artifact plan model.

## Plugin-Specific Notes

Plugins should be treated as a first-class resource family, but they also create an architectural test for the rest of the system.

If plugins land as a one-off special case, the repo will get harder to evolve. If plugins land through the generic resource and emitter architecture, they validate the long-term design.

Recommended plugin rules:

- `PLUGIN.yaml` remains canonical
- plugin-owned resources are loaded as nested resources with ownership metadata
- plugin emitters render vendor package shapes
- plugin marketplace files are generated from the catalog, not treated as a separate top-level family
- import preserves non-representable vendor details under platform override paths

## Migration Plan

This should land in stages so user-visible behavior remains stable.

### Stage 1: Introduce Shared Resource And Artifact Types

Add:

- `resource.Resource`
- `emit.Artifact`
- support reporting types

No user-visible behavior change is required in this stage.

### Stage 2: Add Family Registry And Catalog Resolver

Add:

- family registration
- source loading
- scope precedence
- project selector resolution

Initially, the catalog can coexist with current platform implementations.

### Stage 3: Move Hooks Onto The Generic Planner

Hooks already have the strongest canonical model. They should be the first family fully planned through the new pipeline.

Success criteria:

- current hook behavior preserved
- hook representability reported through shared support results
- artifact planning reusable by status and doctor

### Stage 4: Land Plugins On The Same Architecture

Implement:

- plugin family loader
- plugin resource model
- plugin emitters

This stage validates that the architecture supports a genuinely new family rather than just refactoring old ones.

### Stage 5: Migrate Skills, Agents, And Rules Behind Family Adapters

Keep current disk layout, but move discovery and planning out of platform implementations and into family loaders plus emitters.

Success criteria:

- platform code no longer walks `~/.agents` directly for these families
- status and refresh use catalog data instead of bespoke path logic

### Stage 6: Normalize MCP And Selected Settings

Move only the representable subset into canonical resources.

Preserve:

- platform-owned passthrough settings
- vendor-specific escape hatches

### Stage 7: Add `.agentsrc.json` v2

Introduce generic family selectors and translate v1 manifests internally.

Success criteria:

- no breakage for existing projects
- new families can be selected without top-level schema changes

### Stage 8: Retire Duplicated Command Logic

Once the planner is authoritative:

- simplify `refresh`
- simplify `status`
- simplify `doctor`
- shrink path-mapping logic in import

## Risks

### Over-normalization

If the canonical model tries to erase too much platform specificity, the system will become lossy and brittle.

Mitigation:

- normalize only true shared concepts
- keep platform override and passthrough escape hatches

### Architecture Drift During Migration

If new families are added before the shared planner is in place, the repo may accumulate more one-off code.

Mitigation:

- use hooks and plugins as forcing functions
- require new family work to go through family loaders and emitters where possible

### Excessive Abstraction

If the architecture becomes too generic, simple families may become harder to understand.

Mitigation:

- keep the shared contracts small
- let family packages own family-specific logic
- do not invent bundle manifests where plain files still work well

## Testing Strategy

Add tests at three layers.

### Family Tests

For each family:

- canonical loading
- legacy loading
- validation
- import normalization

### Planner Tests

Cross-family tests for:

- scope precedence
- selector behavior
- required versus optional support
- stale artifact detection

### Platform Emitter Tests

For each platform:

- expected artifact planning
- rendered file content
- transport mode behavior

Stage-level integration tests should verify that a mixed project with rules, skills, agents, hooks, and plugins produces the expected artifact set across multiple platforms.

## Open Questions

1. Should `rules` remain plain files indefinitely, or should they eventually gain optional bundle metadata?
2. Should `mcp` normalize to one server-per-resource, or should some platforms keep aggregate-file ownership?
3. Should project manifests support tag-based or capability-based selectors early, or only named selectors first?
4. Should plugin-owned nested resources appear in `status` as first-class resources or be grouped under their parent plugin by default?
5. Should emitted artifact metadata be recorded on disk in a machine-readable state file, or derived purely from current planning plus filesystem inspection?

## Immediate Next Steps

1. Agree on the core internal contracts: `Resource`, `Artifact`, `Family`, `Emitter`.
2. Create the initial package skeleton for `resource`, `catalog`, and `emit`.
3. Refactor hooks to plan through the shared artifact model.
4. Implement plugins against the same architecture rather than as a standalone subsystem.
5. Add a short follow-on RFC for `.agentsrc.json` v2 once the selector model is finalized.

## Appendix: Concrete Direction For Current Repo Files

The following areas should converge on the architecture in this RFC.

- `internal/platform/platform.go`
  - reduce responsibility from end-to-end orchestration toward platform registration and emitters
- `internal/platform/hooks.go`
  - keep the hook resource model, but move generic planning concepts into shared packages
- `internal/platform/resources.go`
  - move generic resource-discovery helpers into family loaders or shared catalog utilities
- `commands/refresh.go`
  - replace direct platform orchestration with catalog plus planner plus applier flow
- `commands/import.go`
  - move path normalization into family importers
- `internal/config/agentsrc.go`
  - preserve v1, add internal translation to a generic selector model

This RFC builds directly on:

- `docs/CANONICAL_HOOKS_DESIGN.md`
- `docs/PLUGIN_SUPPORT_STRATEGY.md`

Those documents should remain the family-specific design references until their relevant stages are implemented under the shared architecture.
