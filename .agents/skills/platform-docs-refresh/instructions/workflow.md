# Workflow

Refresh the repo doc from the current vendor sources.

## Per-platform loop

1. Open the current official docs for the platform topics already tracked in the target doc.
2. Compare the current vendor docs against the repo's existing wording.
3. Update the repo doc for:
   - factual path changes
   - precedence or compatibility notes
   - added or removed official surfaces
   - checked dates
4. Keep "official vendor behavior" separate from "current repo implementation behavior."

## Topic checklist

Use the tracked topics already present in the target doc, typically:

- instructions or rules
- skills
- agents or subagents
- MCP
- hooks
- plugins
- platform-specific extras already called out in the doc

## Integration rules

- Normalize wording across platforms after gathering findings.
- Prefer short factual bullets over speculative interpretation.
- When a doc page disappears, replace the link or add a short note explaining the gap instead of silently dropping the reference.
