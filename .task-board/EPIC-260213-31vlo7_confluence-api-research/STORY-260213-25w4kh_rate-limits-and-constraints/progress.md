## Status
done

## Assigned To
(none)

## Created
2026-02-13T12:46:42Z

## Last Update
2026-02-13T12:54:39Z

## Blocked By
- (none)

## Blocks
- STORY-260213-1rdtug

## Checklist
(empty)

## Notes
Results: .research/rate-limits-and-constraints.md. Key: 3 independent layers on Cloud (points-based quota, burst per-sec, API token limits). For CLI with API tokens — burst limits main constraint (100 GET/sec, 50 PUT/sec). Points quota NOT applies to API tokens. Pagination: v1 max 25-1000, v2 cursor-based max 250. DC: configurable token bucket. Server: no rate limits.
