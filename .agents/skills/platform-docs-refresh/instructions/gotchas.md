# Gotchas: Platform Docs Refresh

Common failure points:

## Mixing official behavior with repo behavior

- Rewriting "official docs" sections with implementation details from this repo makes the matrix untrustworthy. Keep vendor behavior and repo behavior in separate sections.
- Reusing repo wording without re-checking the current source can preserve stale assumptions after the vendor docs moved.

## Bad source capture

- Saving raw HTML dumps makes the cache noisy and hard to reuse. Prefer markdown sources or concise normalized summaries.
- Caching a page without the canonical URL and checked date makes the snapshot nearly useless on the next refresh.
- Creating a new cache file ad hoc instead of using the helper script can lead to missing parent directories or inconsistent headers.

## Weak stale-link handling

- Replacing a stale URL in the main doc but not logging it in `references/link-registry.md` forces the next refresh to rediscover the move.
- Accepting redirects blindly can pin the cache to a transitional URL instead of the new canonical page.
- Rewriting a working link into a different URL shape because it looks more canonical can break the actual docs workflow if that exact form was never runtime-verified.

## Overstating certainty

- Some vendor docs imply behavior without saying it directly. Mark those statements as inference instead of writing them as confirmed facts.
- Cross-platform comparisons can become editorialized if the source pages use different terms. Keep comparisons factual and minimal.
- Updating a global "checked on" date after refreshing only one platform makes the matrix look fresher than it is. Mark partial refreshes explicitly.
