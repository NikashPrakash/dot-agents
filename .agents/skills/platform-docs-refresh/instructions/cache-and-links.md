# Cache And Links

Use the cache and link registry to make future refreshes cheaper and more reliable.

## Cache workflow

1. Prefer the vendor-authored markdown source when it exists.
2. If the docs site renders markdown from a public repo, store both the docs URL and the source markdown URL in the cache file header.
3. If only rendered HTML exists, save a concise normalized summary rather than raw HTML.
4. Keep one cache file per logical page under `references/cache/<platform>/<slug>.md`.
5. For a new cache file, prefer `scripts/new-doc-cache.sh <platform> <slug> <canonical-url> [stale-url]` so parent directories and headers are created consistently.
6. Overwrite the cache file on refresh and update its checked date.

## Link registry workflow

Update `references/link-registry.md` whenever:

- a tracked official URL is stale
- a docs page moved
- a replacement page is discovered after a 404 or redirect chain

Record the old URL, new URL, topic, date checked, and any note that will help the next refresh.

If no stale or moved links were found, leave the registry unchanged and say that explicitly in the refresh summary.

Do not rewrite a working docs URL into a different "more canonical" form unless you verified that the new form actually works in the target markdown or docs workflow.
