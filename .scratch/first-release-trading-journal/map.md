# First-release implementation map

Integration branch: `integration/first-release-journal`
Baseline: `0145569`

## Task graph

- 01 and 02 have no blockers.
- 03 follows 01 and 02.
- 04, 07, and 08 follow 03.
- 05 and 06 follow 04.
- 09 follows 05 and 06.
- 10 follows 08 and 09.
- 11 and 12 follow 07 and 10.
- 13 follows 12.

## Decisions

The approved test seam is the public Go library API, shared across CSV and an
isolated PostgreSQL database. Each ticket is implemented in its own worktree and
merged into the integration branch after its dependencies are resolved.
