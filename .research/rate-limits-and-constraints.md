# Confluence API Rate Limits and Constraints

Research date: 2026-02-13

---

## Overview

Confluence API rate limiting differs significantly between Cloud and Server/Data Center deployments. Cloud uses a multi-layered system combining points-based quotas, burst rate limits, and (as of late 2025) API token rate limits. Data Center uses a configurable token-bucket approach. Server has no built-in rate limiting.

Key takeaway for CLI tools: **API token-based traffic** (basic auth with email + API token) is governed by burst rate limits and the newer API token rate limiting (since November 2025), but is **not subject to the points-based quota system** that targets Forge/Connect/OAuth 2.0 apps. This is the most likely authentication method for a CLI tool.

---

## Cloud Rate Limits

### Three Independent Rate Limiting Layers

Confluence Cloud enforces three concurrent and independent mechanisms:

1. **Points-based quota (hourly)** -- measures total API "work" per hour
2. **Request rate limits (per-second burst)** -- restricts requests per second per endpoint
3. **API token rate limits** -- additional layer for API-token-based traffic (since Nov 2025)

### 1. Points-Based Quota (Hourly)

**Enforcement date:** March 2, 2026

**Applies to:** Forge, Connect, and OAuth 2.0 (3LO) apps only.
**Does NOT apply to:** API token-based traffic (basic auth).

#### Points Calculation

- **Base cost:** 1 point per request
- **Core domain objects** (Pages, Spaces, Attachments): +1 point (total 2 per GET)
- **Identity & access** (Users, Groups, Permissions): +2 points (total 3 per GET)
- **Write operations** (POST, PUT, PATCH, DELETE): 1 point base only

Examples:
- Fetch a page: 1 (base) + 1 (Page) = **2 points**
- Lookup a user: 1 (base) + 2 (User) = **3 points**
- Create/update a page: **1 point**

#### Quota Tiers

**Tier 1 -- Global Pool (default for most apps):**
- 65,000 points/hour shared across ALL tenants

**Tier 2 -- Per-Tenant Pool (requires Atlassian review):**

| Edition    | Formula                        | Cap        |
|------------|--------------------------------|------------|
| Free       | 65,000 pts/hr                  | 65,000     |
| Standard   | 100,000 + (10 x users) pts/hr | 500,000    |
| Premium    | 130,000 + (20 x users) pts/hr | 500,000    |
| Enterprise | 150,000 + (30 x users) pts/hr | 500,000    |

- Resets at the **top of each UTC hour**
- **No partial throttling** -- once quota is exhausted, ALL requests are blocked until reset
- **No carryover** -- unused quota does not accumulate

### 2. Request Rate Limits (Per-Second Burst)

Enforced independently of hourly quota. Applies to all traffic including API tokens.

**Default steady-state limits by HTTP method:**

| Method | Limit              |
|--------|--------------------|
| GET    | 100 requests/sec   |
| POST   | 100 requests/sec   |
| PUT    | 50 requests/sec    |
| DELETE | 50 requests/sec    |

- Uses a **token bucket** model that allows temporary bursts above steady-state
- **High-impact endpoints** (Permissions, Search, Admin operations) have additional/lower burst protections
- Burst limits reset quickly (within seconds)

### 3. Per-Resource Write Limits

(Documented for Jira, likely similar pattern for Confluence)

| Window          | Limit                    |
|-----------------|--------------------------|
| Short (2 sec)   | 20 write operations      |
| Long (30 sec)   | 100 write operations     |

### 4. API Token Rate Limits

**Enforcement date:** November 22, 2025

- Applies to all API calls using API tokens (basic auth)
- Atlassian has not published exact numbers; stated "we don't expect the majority of customers to be affected"
- Returns beta-phase headers matching Marketplace app headers
- Atlassian reserves the right to enforce limits earlier for integrations impacting stability

---

## Rate Limit Headers

### Standard Headers (returned on all responses)

| Header                   | Description                                        |
|--------------------------|----------------------------------------------------|
| `X-RateLimit-Limit`     | Maximum request rate for current scope             |
| `X-RateLimit-Remaining` | Remaining capacity in current window               |
| `X-RateLimit-Reset`     | ISO 8601 timestamp when limit resets               |
| `X-RateLimit-NearLimit` | `true` when less than 20% of quota remains         |
| `Retry-After`           | Seconds to wait before retrying (on 429 responses) |
| `RateLimit-Reason`      | Which limit was exceeded (see below)               |

### RateLimit-Reason Values

| Value                              | Meaning                          |
|------------------------------------|----------------------------------|
| `confluence-quota-global-based`    | Global hourly points quota hit   |
| `confluence-quota-tenant-based`    | Per-tenant hourly quota hit      |
| `confluence-burst-based`           | Per-second burst limit hit       |

### Beta Headers (Points-Based Quota, pre-enforcement)

| Header                 | Description                                       |
|------------------------|---------------------------------------------------|
| `Beta-RateLimit-Policy`| `q=<total quota>; w=<time window>`                |
| `Beta-RateLimit`       | `r=<remaining>; t=<seconds until reset>`          |

---

## Throttling Behavior

### When a limit is hit:

1. API returns **HTTP 429 Too Many Requests**
2. Response includes `Retry-After` header with seconds to wait
3. Response body is JSON with error details
4. **Quota limits:** all requests blocked until next UTC hour reset
5. **Burst limits:** reset within seconds, normal operations resume quickly

### Recommended retry strategy:

- Only retry **idempotent operations** that return `Retry-After`
- Use **exponential backoff with jitter**
- Double the delay after each successive 429
- Initial delay: up to 5 seconds
- Maximum delay: 30 seconds
- Maximum retries: 4
- Add random jitter to avoid thundering herd

---

## Server/Data Center Rate Limits

### Confluence Server

- **No rate limiting** -- Server has no built-in rate limiting at all
- Server is deprecated by Atlassian

### Confluence Data Center

- **Configurable rate limiting** -- admin-controlled, not enforced by default
- Uses **token bucket** technique (tokens exchanged for requests, 1 token = 1 request)
- Targets **only external REST API requests** -- internal UI requests are NOT limited
- Configuration path: Administration > General Configuration > Rate limiting
- Options: Allow unlimited, Block all, or Limit requests (configurable rate)
- Example: 10 tokens per minute per user
- **Per-user** scoping -- each user gets their own token bucket
- **Allowlisting** available for specific consumer keys (AppLinks integrations)
- Returns **HTTP 429** when limit exceeded

---

## Pagination Limits

### REST API v1 (offset-based)

Uses `start` and `limit` parameters.

**Search endpoint** (`/wiki/rest/api/content/search`):

| Expansion requested                       | Max results per page |
|-------------------------------------------|----------------------|
| No expansions                             | 1,000                |
| Expansions excluding `body`               | 200                  |
| `body` expansion included                 | 50                   |
| `body.export_view` or `body.styled_view`  | 25                   |

These limits are **hardcoded in the backend** and override the `limit` parameter.

**Content endpoints** (`/wiki/rest/api/content`):

- Default limit: 25
- Maximum limit: 250 (varies by endpoint)

### REST API v2 (cursor-based)

Uses `cursor` and `limit` parameters.

| Parameter | Value   |
|-----------|---------|
| Default   | 50      |
| Minimum   | 1       |
| Maximum   | 250     |

- Pagination via `Link` header with opaque cursor token
- Also available in `_links.next` property of response body
- No `total` count is guaranteed in responses -- must paginate until no `next` link

---

## Payload Limits

### Request Body Size

| Deployment    | Default Max     | Configurable?                                    |
|---------------|-----------------|--------------------------------------------------|
| Cloud         | ~5 MB (est.)    | No                                               |
| Data Center   | 5,242,880 bytes (5 MB) | Yes, via `atlassian.rest.request.maxsize` |

- HTTP 413 "Request Entity Too Large" returned when exceeded

### Page Content Size (body.storage)

- **No officially documented hard limit** on body.storage content size
- Community reports suggest content works reliably up to ~64KB+ but very large pages may encounter issues
- Issues at large sizes are often caused by malformed HTML/XML (unescaped characters) rather than size limits

### Attachment Size

| Deployment | Default Max | Configurable?                             |
|------------|-------------|-------------------------------------------|
| Cloud      | 100 MB      | Yes, via admin UI (General Configuration)  |
| DC/Server  | Varies      | Yes, via admin UI                          |

### Response Size

- No officially documented response body size limit from Confluence itself
- Third-party integrations (e.g., Zapier) may impose their own limits (~6 MB)
- Very large responses (pages with many expansions) may cause timeouts

---

## CQL-Specific Limits

### Result Limits

As described in Pagination Limits above, CQL search results are capped based on expansions:
- No body expansion: up to 1,000 results
- With non-body expansions: up to 200 results
- With body expansion: up to 50 results
- With export_view/styled_view: up to 25 results

### Query Constraints

- **No officially documented maximum clause count** for CQL queries
- **No officially documented query complexity limit**
- **No officially documented execution timeout** (though timeouts exist server-side)
- CQL supports: fields, operators, keywords (AND, OR, NOT, ORDER BY), and functions
- Text search uses Lucene syntax under the hood
- Wildcard and fuzzy search supported but may be slower

### CQL Search Points Cost

Under the points-based quota system, Search endpoints are classified as **high-impact** and may receive additional burst protections (lower per-second limits).

---

## Bulk Operation Limits

### Content Body Conversion

- Maximum **10 conversions per request** for content body conversion operations
- Maximum **50 task results per request** for async conversion task results

### No Native Bulk Create/Update

- Confluence REST API does **not** provide native bulk create/update endpoints for pages
- Each page create/update is a separate API call
- Bulk operations must be implemented client-side with sequential or controlled-concurrency requests
- Subject to all rate limits described above

---

## Practical Implications for a CLI Tool

### Authentication Method Matters

A CLI tool using **API tokens (basic auth)** is:
- Subject to **burst rate limits** (100 GET/sec, 50 PUT/sec, 50 DELETE/sec)
- Subject to **API token rate limits** (since Nov 2025, exact numbers TBD)
- **NOT** subject to points-based hourly quota (that targets Forge/Connect/OAuth apps)

### Recommended Client-Side Patterns

1. **Parse rate limit headers** on every response:
   - Monitor `X-RateLimit-Remaining` and `X-RateLimit-NearLimit`
   - Back off proactively when `NearLimit` is `true`

2. **Handle HTTP 429 gracefully:**
   - Read `Retry-After` header
   - Exponential backoff with jitter (initial: 1-5s, max: 30s, max retries: 4)

3. **Optimize pagination:**
   - Avoid `body` expansion in search/list requests when possible
   - Fetch body content separately for specific pages
   - Use v2 API cursor-based pagination when available (up to 250 per page)
   - Without body expansion, can get up to 1,000 search results per page

4. **Control concurrency:**
   - Limit concurrent requests to well under burst thresholds
   - For write operations: stay under 50/sec steady-state
   - For sequential CLI operations: natural request spacing is usually sufficient

5. **Payload management:**
   - Keep request bodies under 5 MB
   - For large page content, ensure proper HTML/XML escaping
   - Attachment uploads: respect 100 MB default (or site-configured limit)

6. **Search optimization:**
   - Use targeted CQL queries to minimize result sets
   - Prefer specific field queries over broad text search
   - Avoid requesting body expansion in search results -- fetch individually

---

## Sources

- [Rate limiting - Confluence Cloud](https://developer.atlassian.com/cloud/confluence/rate-limiting/)
- [Rate limiting - Jira Cloud platform](https://developer.atlassian.com/cloud/jira/platform/rate-limiting/) (shared infrastructure details)
- [Confluence Cloud REST API v2 Introduction](https://developer.atlassian.com/cloud/confluence/rest/v2/intro/)
- [Confluence Cloud REST API v1 Search](https://developer.atlassian.com/cloud/confluence/rest/v1/api-group-search/)
- [Advanced searching using CQL](https://developer.atlassian.com/cloud/confluence/advanced-searching-using-cql/)
- [CQL functions reference](https://developer.atlassian.com/cloud/confluence/cql-functions/)
- [Searching with CQL always limits results to 50](https://support.atlassian.com/confluence/kb/searching-for-content-with-the-rest-api-and-cql-always-limits-results-to-50/)
- [Request too large (5 MB limit)](https://support.atlassian.com/confluence/kb/rest-call-gives-request-too-large-requests-for-this-resource-can-be-at-most-5242880-bytes-error/)
- [Configure attachment size](https://support.atlassian.com/confluence-cloud/docs/configure-attachment-size/)
- [Improving instance stability with rate limiting (DC)](https://confluence.atlassian.com/doc/improving-instance-stability-with-rate-limiting-992679004.html)
- [Adjusting your code for rate limiting (DC)](https://confluence.atlassian.com/doc/adjusting-your-code-for-rate-limiting-992679008.html)
- [API Token Rate Limiting announcement](https://community.developer.atlassian.com/t/api-token-rate-limiting/92292/1)
- [2026 point-based rate limits discussion](https://community.developer.atlassian.com/t/2026-point-based-rate-limits/97828)
- [Confluence Cloud rate limits community thread](https://community.developer.atlassian.com/t/confluence-cloud-rate-limits/28062)
- [body.storage size limit discussion](https://community.developer.atlassian.com/t/is-there-a-size-limit-for-body-storge-when-updating-a-page-with-the-rest-api/58055)
