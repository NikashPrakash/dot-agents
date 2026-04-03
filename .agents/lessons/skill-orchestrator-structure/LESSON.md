# Skill Orchestrator Structure

## Pattern

When creating a new skill, it is easy to put the full operating policy directly into `SKILL.md`. That works, but it does not scale well and it misses the orchestrator pattern expected by `skill-architect`.

## Guardrail

- Keep `SKILL.md` workflow-only.
- Move operational rules into `instructions/` files and load them progressively.
- Always add `instructions/gotchas.md`.
- Add `templates/` and `eval/` when the skill produces output that can look correct while still being wrong.

## Applied Here

- The first version of `platform-docs-refresh` mixed workflow and policy in `SKILL.md`.
- The refined version now follows the orchestrator structure and is easier to maintain and extend.
