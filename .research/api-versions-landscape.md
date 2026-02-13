# Confluence API Versions & Landscape

Research date: 2026-02-13

---

## Overview

Confluence exposes multiple API surfaces depending on the deployment model (Cloud vs. Server/Data Center) and the integration approach (direct REST, Forge apps, Connect apps, GraphQL). The primary APIs are:

| API Surface | Base Path | Deployment | Status |
|---|---|---|---|
| REST API v1 | `/wiki/rest/api/` | Cloud | Actively being deprecated; some endpoints already removed |
| REST API v2 | `/wiki/api/v2/` | Cloud only | Current recommended API for new integrations |
| Server/DC REST API | `/rest/api/` (or `/confluence/rest/api/`) | Server & Data Center | Only API available on-prem; no v2 |
| Atlassian GraphQL Gateway (AGG) | `/gateway/api/graphql` | Cloud only | Public, partially beta; growing coverage |
| Forge `requestConfluence` | Wraps REST v1/v2 | Cloud only (Forge apps) | Managed auth; uses REST under the hood |
| Connect `AP.request` | Wraps REST v1/v2 | Cloud only (Connect apps) | JWT-based auth; legacy app framework |

---

## 1. REST API Versions

### REST API v1 (Cloud)

- **Base URL:** `https://{site}.atlassian.net/wiki/rest/api/{resource}`
- **Content model:** Uses a generic `Content` type. Pages, blog posts, comments, attachments are all flavors of `Content`.
- **Pagination:** Offset-based (`start` + `limit` parameters).
- **Expansions:** Uses `?expand=body.storage,space,version,...` to inline related objects in responses. Powerful but expensive -- each expansion triggers additional DB queries server-side.
- **Authentication:** Basic auth (email + API token), OAuth 2.0 (3LO), Connect JWT, Forge managed.

**Endpoint groups (v1):**

| Group | Example Endpoint |
|---|---|
| Content | `/rest/api/content`, `/rest/api/content/{id}` |
| Content - attachments | `/rest/api/content/{id}/child/attachment` |
| Content body | `/rest/api/contentbody/convert/{to}` |
| Content - children & descendants | `/rest/api/content/{id}/child`, `/rest/api/content/{id}/descendant` |
| Content - macro body | `/rest/api/content/{id}/history/{version}/macro/id/{macroId}` |
| Content labels | `/rest/api/content/{id}/label` |
| Content permissions | `/rest/api/content/{id}/permission/check` |
| Content restrictions | `/rest/api/content/{id}/restriction` |
| Content states | `/rest/api/content/{id}/state` |
| Content versions | `/rest/api/content/{id}/version` |
| Content watches | `/rest/api/content/{id}/notification/child-created` |
| Search (CQL) | `/rest/api/search`, `/rest/api/content/search` |
| Space | `/rest/api/space`, `/rest/api/space/{key}` |
| Space permissions | `/rest/api/space/{key}/permission` |
| Space settings | `/rest/api/space/{key}/settings` |
| Group | `/rest/api/group` |
| Users | `/rest/api/user` |
| User properties | `/rest/api/user/{accountId}/property` |
| Label info | `/rest/api/label` |
| Audit | `/rest/api/audit` |
| Analytics | `/rest/api/analytics/content/{contentId}/views` |
| Long-running task | `/rest/api/longtask` |
| Relation | `/rest/api/relation` |
| Settings | `/rest/api/settings/lookandfeel` |
| Template | `/rest/api/template` |
| Themes | `/rest/api/settings/theme` |
| Dynamic modules | `/rest/api/app/module/dynamic` |
| Experimental | `/rest/api/experimental/...` |

### REST API v2 (Cloud)

- **Base URL:** `https://{site}.atlassian.net/wiki/api/v2/{resource}`
- **Content model:** No generic `Content` type. Each entity (Page, Blog Post, Comment, Attachment, Custom Content, Whiteboard, Database, Folder) has its own dedicated endpoints.
- **Pagination:** Cursor-based (`cursor` + `limit` parameters). Substantially better latency than offset-based, especially at high offsets. Default limit: 50, max: 250 (varies by endpoint). Uses `Link: <...>; rel="next"` headers.
- **No expansions:** Responses contain only IDs referencing related objects (e.g., `spaceId` instead of a nested `space` object). You make separate calls to fetch related entities. This trade-off enables predictable DB access and lower latency.
- **Authentication:** Same as v1 -- Basic auth, OAuth 2.0 (3LO), Forge, Connect. Uses granular scopes like `read:page:confluence`, `write:page:confluence`, etc.

**Endpoint groups (v2):**

| Group | Example Endpoints |
|---|---|
| Page | `/api/v2/pages`, `/api/v2/pages/{id}` |
| Blog Post | `/api/v2/blogposts`, `/api/v2/blogposts/{id}` |
| Comment | `/api/v2/comments`, `/api/v2/pages/{id}/footer-comments` |
| Attachment | `/api/v2/attachments`, `/api/v2/pages/{id}/attachments` |
| Custom Content | `/api/v2/custom-content` |
| Whiteboard | `/api/v2/whiteboards`, `/api/v2/whiteboards/{id}` |
| Database | `/api/v2/databases` |
| Folder | `/api/v2/folders` |
| Space | `/api/v2/spaces`, `/api/v2/spaces/{id}` |
| Space Permissions | `/api/v2/spaces/{id}/permissions` |
| Space Properties | `/api/v2/spaces/{id}/properties` |
| Space Roles | `/api/v2/spaces/{id}/roles` |
| Label | `/api/v2/labels`, `/api/v2/pages/{id}/labels` |
| Like | `/api/v2/pages/{id}/likes` |
| Task | `/api/v2/tasks`, `/api/v2/tasks/{id}` |
| Content Properties | `/api/v2/pages/{id}/properties` |
| Children | `/api/v2/pages/{id}/children` |
| Ancestors | `/api/v2/pages/{id}/ancestors` |
| Descendants | `/api/v2/pages/{id}/descendants` |
| Version | `/api/v2/pages/{id}/versions` |
| User | `/api/v2/users/{accountId}` |
| Admin Key | Admin-level bypass for access restrictions |
| App Properties | `/api/v2/app/properties` |
| Classification Level | Content classification / data governance |
| Data Policies | Data policy management |
| Redactions | Content redaction |
| Smart Link | Inline card / smart link resolution |
| Operation | Permission operations |

### Key Architectural Differences: v1 vs v2

| Aspect | v1 | v2 |
|---|---|---|
| Content model | Generic `Content` type | Typed entities (Page, BlogPost, etc.) |
| Pagination | Offset-based (`start`/`limit`) | Cursor-based (`cursor`/`limit`) |
| Related data | `?expand=` inlines nested objects | IDs only; separate fetch calls |
| Latency | Higher (especially at high offsets, with expansions) | Lower, more predictable |
| Base path | `/wiki/rest/api/` | `/wiki/api/v2/` |
| Search (CQL) | Full CQL support (`/rest/api/search`) | No dedicated v2 search endpoint yet (use v1) |
| Content body conversion | `/rest/api/contentbody/convert/{to}` (deprecated, extended to June 2026) | Async conversion endpoint available |
| Audit, Analytics | Available | Not in v2 |
| Templates, Themes | Available | Not in v2 |
| Dynamic modules | Available | Not in v2 |
| Settings | Available | Not in v2 |

### What v2 Has That v1 Doesn't

- Whiteboard endpoints
- Database endpoints
- Folder endpoints
- Classification levels
- Data policies / redactions
- Smart links
- Space roles
- Admin key management

---

## 2. Deprecation Status & Timeline

### Chronology of v1 Deprecation (Cloud)

| Date | Event |
|---|---|
| Mar 2023 | Atlassian announces deprecation of v1 endpoints that have v2 equivalents |
| Aug 2023 | RFC-19 published; original removal target: Jan 2024 |
| Late 2023 | Timeline extended due to community pushback and gaps |
| Mid 2024 | Further extensions granted; many v1 endpoints still active |
| Apr 30, 2025 | Most deprecated v1 endpoints scheduled for removal |
| Sep 30, 2025 | Children/descendants v1 endpoints extended deadline |
| Jun 3, 2026 | `Convert content body` v1 endpoint deadline (latest extension) |
| Apr 15, 2026 | Internal `/download/attachments/` endpoint loses API token access |

### Current State (Feb 2026)

- **Most v1 endpoints with v2 equivalents:** Deprecated. Removal has been happening in waves, though Atlassian has repeatedly extended deadlines when gaps are found.
- **Search API (`/rest/api/search`):** Still v1-only. No v2 equivalent. Expected to remain.
- **Content body conversion:** Extended to June 2026. Use async v2 endpoint instead.
- **Audit, Analytics, Templates, Themes, Settings, Dynamic modules:** v1-only. No announced deprecation for endpoints that have NO v2 equivalent.
- **Some v1 endpoints likely still functional** despite "deprecated" status, as Atlassian has been cautious about hard removal.

### Practical Guidance

- For **new integrations:** Use v2 endpoints wherever possible.
- For **CQL search:** Must use v1 (`/rest/api/search`) -- no v2 equivalent exists.
- For **audit, analytics, templates, themes:** Must use v1 -- no v2 equivalent.
- For **content body conversion:** Use the async v2 endpoint; the v1 sync endpoint expires June 2026.
- Monitor the [Confluence Cloud changelog](https://developer.atlassian.com/cloud/confluence/changelog/) for removal announcements.

---

## 3. Atlassian GraphQL Gateway (AGG)

### Overview

AGG is a cross-product GraphQL API that federates data from multiple Atlassian services (Jira, Confluence, Bitbucket, Opsgenie, Compass, etc.) through a single endpoint.

### Endpoints

| Auth Method | URL |
|---|---|
| OAuth | `https://api.atlassian.com/graphql` |
| API token / session | `https://{site}.atlassian.net/gateway/api/graphql` |

### Confluence Coverage in GraphQL

- **Entities:** `ConfluencePage`, `ConfluenceBlogPost`, comments, spaces, databases, embeds, whiteboards, tasks, templates, macros, folders.
- **Status:** Many fields started in beta (2022), some have graduated to stable. Beta fields require `X-ExperimentalApi: <betaName>` header.
- **Legacy namespace:** `confluenceLegacy_*` queries exist (e.g., `confluenceLegacy_contentAnalyticsLastViewedAtByPage`) for analytics and data not yet in the modern schema.
- **Rate limiting:** Cost-based, 10,000 points per currency per minute. HTTP 429 with `RETRY-AFTER` on exceeded limits.

### GraphQL vs REST

- GraphQL is **not a replacement** for the REST API. It's a complementary access layer.
- Best for: cross-product queries, fetching specific fields without over-fetching, apps that need data from multiple Atlassian products.
- The official v2 REST API blog post notes: "Those needing expansions are directed toward the GraphQL API instead."
- Not suitable for: write-heavy workflows (mutation support is limited), bulk operations, anything requiring CQL search.

### Forge `requestGraph`

Forge apps can call AGG via `requestGraph()` from `@forge/api`. This is the recommended way for Forge apps to access GraphQL. Authentication is automatically managed by the Forge platform.

---

## 4. Cloud vs Server/Data Center Matrix

### Server/Data Center REST API

- **Base URL:** `http://{host}:{port}/confluence/rest/api/{resource}` (with context path) or `http://{host}:{port}/rest/api/{resource}` (without context path)
- **Version numbering:** Tied to product release versions (e.g., `v9214` = Confluence DC 9.2.14). Not the same as Cloud's v1/v2 scheme.
- **Content model:** Generic `Content` type (like Cloud v1).
- **Pagination:** Offset-based.
- **Authentication:** Basic auth (username + password or Personal Access Token from DC 7.9+).
- **No v2 API.** The v2 REST API is Cloud-only.
- **No GraphQL.** AGG is Cloud-only.
- **Server end-of-life:** Server products ended support Feb 15, 2024. Only Data Center is supported on-prem.

### Feature Matrix

| Feature | Cloud v1 | Cloud v2 | Server/DC |
|---|---|---|---|
| Pages CRUD | Yes | Yes | Yes |
| Blog Posts CRUD | Yes | Yes | Yes |
| Comments | Yes | Yes | Yes |
| Attachments | Yes | Yes | Yes |
| Spaces | Yes | Yes | Yes |
| Labels | Yes | Yes | Yes |
| CQL Search | Yes | No (use v1) | Yes |
| Content Properties | Yes | Yes | Yes |
| Space Permissions | Yes | Yes | Yes |
| Whiteboards | No | Yes | No |
| Databases | No | Yes | No |
| Folders | No | Yes | No |
| Classification | No | Yes | No |
| Smart Links | No | Yes | No |
| Audit | Yes | No (v1 only) | Yes |
| Analytics | Yes | No (v1 only) | No |
| Templates | Yes | No (v1 only) | Yes |
| Themes | Yes | No (v1 only) | Yes |
| Content Restrictions | Yes | No (v1 only) | Yes |
| Content Watches | Yes | No (v1 only) | Yes |
| Relations | Yes | No (v1 only) | Yes |
| Groups | Yes | No (v1 only) | Yes |
| Webhooks | No | No | Yes |
| Backups/Restore | No | No | Yes |
| GraphQL (AGG) | Yes | - | No |
| Cursor pagination | No | Yes | No |
| `?expand=` support | Yes | No | Yes |

### App Framework Matrix

| Framework | Cloud | Server/DC |
|---|---|---|
| Forge | Yes (recommended) | No |
| Connect | Yes (legacy, still supported) | No |
| P2 Plugins (Java) | No | Yes |
| REST API (direct) | Yes | Yes |

---

## 5. Authentication Summary

| Method | Cloud v1 | Cloud v2 | Server/DC | GraphQL |
|---|---|---|---|---|
| Basic auth (email + API token) | Yes | Yes | N/A | N/A |
| Basic auth (username + password) | No (deprecated) | No | Yes | No |
| Personal Access Token | No | No | Yes (DC 7.9+) | No |
| OAuth 2.0 (3LO) | Yes | Yes | No | Yes |
| Forge managed auth | Yes | Yes | No | Yes (`requestGraph`) |
| Connect JWT | Yes | Yes | No | No |
| API token (via header) | N/A | N/A | N/A | Yes (tenanted URL) |

---

## 6. Recommendations for New Integrations

1. **Cloud integrations -- use REST API v2** as the primary API. Fall back to v1 only for endpoints that don't exist in v2 (search, audit, analytics, templates, themes, content restrictions, watches, relations, groups).

2. **Search -- use v1 CQL** (`/wiki/rest/api/search`). There is no v2 search endpoint. This v1 endpoint is not deprecated.

3. **Forge apps** -- use `requestConfluence()` for REST calls and `requestGraph()` for GraphQL. Auth is managed. Prefer v2 REST endpoints.

4. **Cross-product data needs** -- consider AGG GraphQL. Good for fetching specific fields across Jira + Confluence without over-fetching.

5. **Server/Data Center** -- the only option is the Server REST API (`/rest/api/`). No v2, no GraphQL, no Forge.

6. **Don't build on deprecated v1 endpoints** that have v2 equivalents. The deprecation is slow but real. Content CRUD, space management, labels, attachments, comments -- all have v2 equivalents.

7. **Content format** -- v2 endpoints work with `atlas_doc_format` (ADF, Atlassian Document Format) as the primary body format. v1 uses `storage` format (XHTML-based). Both are available in both versions, but ADF is the forward-looking format.

8. **Rate limits** -- Cloud REST APIs have rate limits (varies by endpoint, generally documented per-endpoint). GraphQL uses cost-based limits (10k points/min). Plan for 429 responses and implement backoff.

---

## Sources & Links

### Official Documentation
- [Confluence Cloud REST API v2 Introduction](https://developer.atlassian.com/cloud/confluence/rest/v2/intro/)
- [Confluence Cloud REST API v2 Reference](https://developer.atlassian.com/cloud/confluence/rest/v2/)
- [Confluence Cloud REST API v1 Introduction](https://developer.atlassian.com/cloud/confluence/rest/v1/intro/)
- [Using the REST API (Cloud)](https://developer.atlassian.com/cloud/confluence/using-the-rest-api/)
- [Confluence Data Center REST API](https://developer.atlassian.com/server/confluence/confluence-server-rest-api/)
- [Confluence Data Center REST API Reference](https://developer.atlassian.com/server/confluence/rest/v9214/)
- [Confluence Cloud Changelog](https://developer.atlassian.com/cloud/confluence/changelog/)
- [Advanced Searching Using CQL](https://developer.atlassian.com/cloud/confluence/advanced-searching-using-cql/)

### GraphQL
- [Atlassian GraphQL API](https://developer.atlassian.com/platform/atlassian-graphql-api/)
- [GraphQL API Reference](https://developer.atlassian.com/platform/atlassian-graphql-api/graphql/)
- [Confluence GraphQL APIs Beta Announcement](https://community.developer.atlassian.com/t/confluence-graphql-apis-are-now-available-in-beta/56790)
- [GraphQL: New Fields and Types Update](https://community.developer.atlassian.com/t/confluence-graphql-api-new-fields-and-types-and-first-set-of-fields-moved-out-of-beta/60537)

### Forge & Connect
- [Forge requestConfluence](https://developer.atlassian.com/platform/forge/apis-reference/fetch-api-product.requestconfluence/)
- [Forge requestGraph](https://developer.atlassian.com/platform/forge/apis-reference/fetch-api-product.requestgraph/)
- [Confluence Scopes for OAuth 2.0 and Forge](https://developer.atlassian.com/cloud/confluence/scopes-for-oauth-2-3LO-and-forge-apps/)
- [Forge on Confluence](https://developer.atlassian.com/cloud/confluence/forge/)

### Deprecation & Migration
- [RFC-19: Deprecation of v1 Endpoints](https://community.developer.atlassian.com/t/rfc-19-deprecation-of-confluence-cloud-rest-api-v1-endpoints/71752)
- [Deprecating v1 APIs with v2 Equivalents](https://community.developer.atlassian.com/t/deprecating-many-confluence-v1-apis-that-have-v2-equivalents/66883)
- [Update to v1 Deprecation Timeline (Nov 2024)](https://community.developer.atlassian.com/t/update-to-confluence-v1-api-deprecation-timeline/79687)
- [v1 Deprecation Timeline Update (Jun 2024)](https://community.developer.atlassian.com/t/confluence-rest-api-v2-update-to-v1-deprecation-timeline/75126)
- [Search API Deprecation Notice](https://developer.atlassian.com/cloud/confluence/deprecation-notice-search-api/)
- [V2 Performance Improvements Blog Post](https://www.atlassian.com/blog/developer/the-confluence-cloud-rest-api-v2-brings-major-performance-improvements)

### Community Discussions
- [v1 vs v2 Discussion](https://community.atlassian.com/forums/Confluence-questions/Confluence-API-v1-versus-v2/qaq-p/2978171)
- [v2 Feature Gaps Discussion](https://community.atlassian.com/forums/Confluence-questions/the-funciton-lackness-of-REST-API-v2-compared-with-v1/qaq-p/3101941)
- [v1 to v2 Migration Discussion](https://community.developer.atlassian.com/t/confluence-cloud-rest-api-v1-to-v2-migration/73881)
- [Confluence Legacy GraphQL in Forge](https://community.developer.atlassian.com/t/how-to-use-confluence-legacy-graphql-in-forge-app/85492)
