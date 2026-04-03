# Eval Checklist

Pass the skill only if all of these are true:

- `SKILL.md` is workflow-only and uses `Load ->` directives.
- `instructions/gotchas.md` exists and covers at least three concrete failure modes.
- The workflow tells the agent to separate official vendor behavior from repo implementation behavior.
- The workflow tells the agent how to handle a partial refresh of a larger matrix.
- The workflow tells the agent to record stale-link replacements.
- The workflow tells the agent to save markdown-friendly sources when available.
- The workflow avoids requiring raw HTML dumps.
- The skill still points back to `docs/PLATFORM_DIRS_DOCS.md` as the default inventory.
- A reusable summary template exists for reporting refresh results.
