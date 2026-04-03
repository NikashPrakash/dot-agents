# Scope

Start from [docs/PLATFORM_DIRS_DOCS.md](/Users/nikashp/Documents/dot-agents/docs/PLATFORM_DIRS_DOCS.md). Treat its platform sections and topic buckets as the refresh inventory unless the user narrows the task.

## Scope the run

1. Identify the target repo doc to update.
2. Identify which platforms and topic buckets are in scope.
3. Decide whether the work should be done sequentially or one platform at a time in parallel when subagents are available.
4. Open only the reference files you need:
   - `references/cache-policy.md`
   - `references/link-registry.md`
   - existing `references/cache/<platform>/...` files for the pages you are revisiting

## Fan-out guidance

- If the task spans multiple platforms and delegation is allowed, split by platform.
- Keep each worker scoped to one platform with exact ownership of the related sections in the target doc.
- Ask each worker for current official URLs, factual deltas, cache candidates, and stale-link replacements.
- Integrate all results centrally so wording and cross-platform comparisons stay consistent.

## Source rule

Use official vendor docs only. If a statement is an inference rather than directly documented, label it as an inference in the repo doc or refresh summary.
