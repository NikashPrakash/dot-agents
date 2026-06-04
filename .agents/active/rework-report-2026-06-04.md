# Rework Report — 2026-06-04

Rework-round review of four delegated tasks across two tracks. Each verdict is a
3-lens decision (phase-1 design + phase-2 implementation → overall) recorded via
`da workflow verify record`. The canonical per-task record is the
`review-decision.yaml` under `.agents/active/verification/<task>/`; no separate
numeric score is persisted, so the 3-lens verdict is the score of record.

## Track A — layered-pr-fanout

### lpf-pr-producer — PR #22 — ACCEPT

- **PR:** https://github.com/AGOrcha/dot-agents/pull/22
- **3-lens decision:** phase-1 `accept` / phase-2 `accept` → overall **accept**
- **Score of record:** 3-lens accept (no failed gates)
- **Advanced to:** `awaiting_owner_review`
- **Blocking findings:** none.

Rework resolved 3 previously-blocking findings: (1) `pr_source` config was dead —
now bridged `AgentsRCPRSource` → engine `PRSourceConfig` via `ToEngineConfig` and
driven through `AgentsRC.NewPRListProducer` so configured sources build a real
producer; (2) `DeriveRollupState` was off the live path — `PRProducer.Cycle` now
derives GREEN/FAILING/PENDING per cycle, empty checks derive explicit GREEN;
(3) `AuthRoundTripper` proxy dropped request body metadata —
`ContentLength`/`GetBody`/`TransferEncoding`/`Trailer`/`Host` now preserved so
POST/PATCH bodies forward intact. Each fix has end-to-end test coverage.

### lpf-d-base-resolution — PR #23 — ACCEPT

- **PR:** https://github.com/AGOrcha/dot-agents/pull/23
- **3-lens decision:** phase-1 `accept` / phase-2 `accept` → overall **accept**
- **Score of record:** 3-lens accept (no failed gates)
- **Advanced to:** `awaiting_owner_review`
- **Blocking findings:** none.

This is the re-cut of the layered-pr-fanout base-resolution slice
(base-resolution via the `internal/events` PR producer), accepted clean.

## Track B — config-relevance-profiles

### t3-noise-suppression — PR #20 — ACCEPT

- **PR:** https://github.com/AGOrcha/dot-agents/pull/20
- **3-lens decision:** phase-1 `accept` / phase-2 `accept` → overall **accept**
- **Score of record:** 3-lens accept (no failed gates)
- **Advanced to:** `awaiting_owner_review`
- **Blocking findings:** none.

Rework verdict resolved the prior blockers: (1) `default_class=situational` so
units not explicitly classed noise are never silently dropped from the working
set; (2) noise suppression is a reversible filtered view over the resolved set,
not a deletion; (3) pure function with table-driven tests covering
core/situational/noise classification.

### t4-relevance-recompute — PR #21 — REJECT

- **PR:** https://github.com/AGOrcha/dot-agents/pull/21
- **3-lens decision:** phase-1 `reject` / phase-2 `reject` → overall **reject**
- **Score of record:** 3-lens reject
- **Advanced to:** `in_progress` (returned for further rework)
- **Blocking findings (verbatim):**

  > acceptance-invariants: commands/config/relevance_recompute.go:545 proposedLayer
  > (+ buildRecomputeResult:330 / recomputeStages:372) — ROUND-TRIP LOSS:
  > --stage-scoped --write drops every non-targeted stage's relevance from the
  > target app_type

## #15 / lpf-d mergeability

#15 (`lpf-d-rebase` → `master`) is **CLOSED** and was superseded by the re-cut
slice. With **lpf-d-base-resolution accepted as PR #23**, the accepted swap now
exists: lpf-d is **mergeable via #23** — #23 carries the accepted base-resolution
work that #15 was holding. #15 itself stays closed; merge #23 in its place.

## Merge queue (dependency order)

1. **#22 (lpf-pr-producer)** — merge first; #23 depends on it.
2. **#23 (lpf-d-base-resolution)** — merge after #22 (the accepted lpf-d swap).
3. **#20 (t3-noise-suppression)** — independent of the Track-A queue and of #21;
   mergeable any time.

- #20 (t3) and #21 (t4) are mutually independent.
- Within Track A, **#22 must land before #23**.
- **#21 (t4-relevance-recompute) is NOT in the merge queue** — rejected, returned
  to `in_progress` pending resolution of the round-trip-loss blocker above.
