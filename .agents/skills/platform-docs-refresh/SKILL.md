---
name: "platform-docs-refresh"
description: "Use when updating `docs/PLATFORM_DIRS_DOCS.md` or similar platform-compat docs against current official online docs, especially when links may be stale or vendor docs may have moved."
---

# Platform Docs Refresh

Refresh platform-location docs from current vendor sources without re-discovering the workflow each time.

## Workflow

1. **Scope the refresh**
   Load -> `instructions/scope.md`
   Identify the target doc, the platform/topic inventory, and whether this run should fan out by platform.

2. **Run the platform refresh**
   Load -> `instructions/workflow.md`
   Verify each platform against current official docs and update the repo doc from those findings.

3. **Refresh caches and moved links**
   Load -> `instructions/cache-and-links.md`
   Save markdown-friendly source snapshots when available and record stale-link replacements for future runs.

4. **Check failure points**
   Load -> `instructions/gotchas.md`
   Review the common ways this skill can silently drift or overstate vendor behavior.

5. **Summarize the refresh**
   Load -> `templates/refresh-summary.md`
   Produce a concise update covering factual changes, moved links, cached sources, and remaining uncertainty.
