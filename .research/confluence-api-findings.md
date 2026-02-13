# Confluence API Research Findings & Architectural Recommendations

Research synthesis date: 2026-02-13

---

## 1. Executive Summary

Research across five documents (API landscape, authentication, endpoints/schemas, rate limits, and the jira-management skill study) confirms that building a Confluence CLI tool is highly feasible by reusing 70-90% of the jira-management skill architecture. The approach is: **Go CLI with Cobra, DSL for reads, explicit commands for writes, v2 API as primary for Cloud with v1 fallback for CQL search and Server/DC, same auth/config/keychain pattern, same token reuse (one Atlassian API token covers both Jira and Confluence Cloud)**. The main new challenges are Confluence storage format (XHTML-based), page tree/hierarchy navigation, CQL instead of JQL, space key-to-ID mapping for v2, and dual-API routing (v1 for search, v2 for everything else on Cloud).

---

## 2. Architecture Decision: API Strategy

### Principle

Use v2 as default for Cloud. Fall back to v1 only for operations that have no v2 equivalent (CQL search, content body conversion). Server/DC gets v1-style endpoints only (no v2 exists there).

### Operation-to-API Mapping

| Operation | Cloud API | Endpoint | Notes |
|-----------|-----------|----------|-------|
| Get page by ID | v2 | `GET /wiki/api/v2/pages/{id}?body-format=storage` | Must pass `body-format` or body is `{}` |
| List pages (filtered) | v2 | `GET /wiki/api/v2/pages?space-id=X&title=Y` | Cursor-based pagination, max 250/page |
| Create page | v2 | `POST /wiki/api/v2/pages` | Uses `spaceId` (not key), `parentId` |
| Update page | v2 | `PUT /wiki/api/v2/pages/{id}` | Requires `version.number = current + 1` |
| Delete/trash page | v2 | `DELETE /wiki/api/v2/pages/{id}` | Moves to trash |
| Get children | v2 | `GET /wiki/api/v2/pages/{id}/children` | Cursor pagination |
| Get ancestors | v2 | `GET /wiki/api/v2/pages/{id}/ancestors` | Root-to-parent order |
| Get versions | v2 | `GET /wiki/api/v2/pages/{id}/versions` | Cursor pagination |
| Labels CRUD | v2 | `GET/POST/DELETE /wiki/api/v2/pages/{id}/labels` | v2 delete uses label ID |
| List spaces | v2 | `GET /wiki/api/v2/spaces` | Returns `id`, `key`, `name` |
| Get space by ID | v2 | `GET /wiki/api/v2/spaces/{id}` | |
| **CQL search** | **v1** | `GET /wiki/rest/api/search?cql=...` | **No v2 equivalent. Not deprecated.** |
| CQL content search | v1 | `GET /wiki/rest/api/content/search?cql=...` | Simpler response (Content objects only) |
| Space key lookup | v2 | `GET /wiki/api/v2/spaces?keys=TST` | Needed to resolve key -> ID for v2 writes |
| **Server/DC: all ops** | **v1** | `/rest/api/content/...`, `/rest/api/space/...` | Same as Cloud v1 but different base path |

### Dual-API Router Logic

```
if Cloud:
    if operation == CQL search:
        use /wiki/rest/api/search (v1)
    else:
        use /wiki/api/v2/ (v2)
else (Server/DC):
    use /rest/api/ or /confluence/rest/api/ (v1-style)
```

This is more nuanced than jira-management's Cloud v3 / Server v2 split because Confluence Cloud itself needs both v1 and v2.

---

## 3. Architecture Decision: Auth Strategy

### Cloud: Basic Auth (email + API token)

- Header: `Authorization: Basic base64(email:token)`
- **Same token works for both Jira and Confluence Cloud**. Users with existing jira-mgmt setup don't need a new token.
- Token created at `id.atlassian.com` -- tied to Atlassian account, not individual products.
- Config values: `email`, `api_token`, `instance_url` (e.g., `https://company.atlassian.net/wiki`)
- Confluence Cloud base URL includes `/wiki` prefix.

### Server/DC: Bearer PAT

- Header: `Authorization: Bearer <token>`
- No email needed -- PAT is self-contained.
- PATs are product-specific on Server/DC (Jira PAT does NOT work for Confluence).
- Config values: `pat`, `instance_url` (e.g., `https://confluence.company.com`)

### Auto-Detection

Reuse jira-management pattern with adaptation:

1. **Auth type**: email provided -> Basic (Cloud), no email -> Bearer (Server/DC PAT)
2. **Instance type**: Probe endpoint to confirm:
   - Cloud: `GET /wiki/rest/api/settings/systemInfo` or `/_edge/tenant_info` -- look for Cloud indicators
   - Server/DC: same endpoint returns on-prem indicators
   - URL heuristic as fast-path: `*.atlassian.net` = Cloud, else Server/DC
3. **Store**: credentials in OS keychain (go-keyring), config in YAML

### Config File

Path: `~/.config/confluence-mgmt/config.yaml`

```yaml
active_space: SPACEKEY
locale: en
instance_url: https://company.atlassian.net/wiki
instance_type: cloud      # "cloud" or "server"
auth_type: basic           # "basic" or "bearer"
```

### Credential Store

Service name: `confluence-mgmt` in OS keychain. Same `CredentialStore` interface as jira-management.

---

## 4. Architecture Decision: CLI Structure

### Framework

- Language: Go (same as jira-management, task-board)
- CLI framework: Cobra
- Module path: `github.com/ivalx1s/skill-confluence-manager`
- Binary name: `confluence-mgmt`
- Deployed to: `~/.local/bin/confluence-mgmt`

### Command Tree

```
confluence-mgmt
|-- auth                        # Auth setup (--url, --email, --token / --pat)
|-- config
|   |-- set <key> <value>       # Set config (space, locale)
|   `-- show                    # Show current config
|-- q '<dsl-query>'             # DSL query execution (reads)
|-- grep <pattern>              # Regex search across page titles/bodies
|-- page
|   |-- create                  # Create page (--space, --title, --body, --parent, --body-file)
|   |-- update <PAGE-ID>        # Update page (--title, --body, --body-file, --message)
|   |-- move <PAGE-ID>          # Move page (--to-space, --under)
|   |-- archive <PAGE-ID>       # Archive page
|   `-- delete <PAGE-ID>        # Trash page
|-- label
|   |-- add <PAGE-ID>           # Add labels (--labels "a,b,c")
|   `-- remove <PAGE-ID>        # Remove labels (--labels "a,b")
|-- space
|   `-- list                    # List accessible spaces
`-- version                     # Print version
```

### Output Formats

| Format | Flag | Use case |
|--------|------|----------|
| `json` | `--format json` | Agent/programmatic parsing (default) |
| `compact` | `--format compact` | Minimal token-efficient for agents |
| `text` | `--format text` | Human-readable |

JSON as default (same as jira-management) -- agents parse JSON best.

### Global Flags

| Flag | Type | Default | Purpose |
|------|------|---------|---------|
| `--space KEY` | string | from config | Override active space |
| `--format json\|compact\|text` | string | `json` | Output format |

### Read/Write Separation

- **Reads**: DSL queries via `q` command (token-efficient, batch-capable)
- **Writes**: Explicit subcommands (`page create`, `page update`, `label add`, etc.)
- **Search**: CQL passthrough via DSL `search()` operation or `grep` for regex

---

## 5. Key Technical Gotchas

### 5.1 v2 `body-format` Parameter

v2 GET endpoints return `body: {}` (empty) unless `body-format=storage` or `body-format=atlas_doc_format` is explicitly passed. This is the most common trap. The CLI must always include `body-format=storage` when body content is requested.

### 5.2 Version Increment on Update (409 Conflict)

Every page update requires `version.number = current_version + 1`. The CLI must:
1. GET the page to read current `version.number`
2. PUT with `version.number + 1`
3. Handle `409 Conflict` (stale version) with a retry: re-read version, re-attempt

### 5.3 Space Key vs Space ID

v2 uses `spaceId` (numeric string) everywhere. v1 uses `spaceKey` (e.g., "TST"). The CLI needs a space key-to-ID resolver:
- Cache: `GET /wiki/api/v2/spaces?keys=TST` to get the ID
- Store mapping in memory (or config) to avoid repeated lookups
- v1 (Server/DC) uses space key directly -- no mapping needed

### 5.4 CQL Search is v1-Only

No v2 search endpoint exists. CQL search must always go through `/wiki/rest/api/search`. This endpoint is NOT deprecated and expected to remain. The CLI must route search operations to v1 even on Cloud.

### 5.5 Pagination Differences

| API | Method | Params | How to detect end |
|-----|--------|--------|-------------------|
| v2 (Cloud) | Cursor-based | `cursor` + `limit` (max 250) | No `next` link |
| v1 (Cloud/Server) | Offset-based | `start` + `limit` (max varies) | `size < limit` |
| v1 search | Both available | `cursor` (preferred) or `start` | No `next` link / `size < limit` |

The HTTP client needs both pagination strategies.

### 5.6 Rate Limit Headers

Parse on every response:

| Header | Action |
|--------|--------|
| `X-RateLimit-Remaining` | Monitor remaining capacity |
| `X-RateLimit-NearLimit` | Proactive backoff when `true` |
| `Retry-After` | Wait this many seconds on 429 |
| `RateLimit-Reason` | Log which limit was hit |

Retry strategy: exponential backoff with jitter, initial 1-5s, max 30s, max 4 retries.

### 5.7 Confluence Storage Format

Page bodies use XHTML-based storage format. It's not raw HTML -- it uses Confluence-specific macros and elements. The CLI should:
- Accept raw storage format for create/update (from `--body` or `--body-file`)
- Return storage format in reads
- Not attempt to render or convert it -- let agents work with raw storage format
- Optionally support `--body-format atlas_doc_format` for ADF JSON

### 5.8 Confluence Base URL Prefix

Cloud Confluence URLs include `/wiki` prefix: `https://domain.atlassian.net/wiki/api/v2/...`. This differs from Jira which has no prefix. The HTTP client must handle this correctly.

### 5.9 Search Result Limits with Body Expansion

CQL search has hardcoded backend limits based on what's expanded:
- No body: up to 1000 results/page
- With body: max 50 results/page
- With `body.export_view`: max 25 results/page

The CLI should avoid expanding body in search results. Fetch body separately for specific pages.

---

## 6. Reuse from jira-management

| Component | Source File(s) | Reuse Level | Adaptation Needed |
|-----------|---------------|-------------|-------------------|
| Project structure & layout | entire repo | 95% | Change directory/module names |
| Makefile / scripts | `scripts/setup.sh`, `scripts/deinit.sh` | 95% | Change binary name, symlink targets |
| Config manager | `internal/config/config.go` | 90% | Change config keys (`active_space` instead of `active_project`), config path |
| Credential store | `internal/config/auth.go` | 95% | Change service name to `confluence-mgmt` |
| HTTP client (core) | `internal/jira/client.go` | 80% | Change base paths, add v1/v2 routing, add `/wiki` prefix for Cloud |
| Retry logic | `internal/jira/client.go` | 95% | Same exponential backoff, same 429 handling |
| DSL tokenizer | `internal/query/parser.go` (tokenizer portion) | 95% | Domain-agnostic, reuse verbatim |
| DSL parser | `internal/query/parser.go` (parser portion) | 70% | New operations, new field names, new presets |
| DSL executor | `internal/query/ops.go` | 20% | Completely new operation handlers |
| Field selector | `internal/fields/selector.go` | 50% | New field mappings, same Apply/ApplyMany pattern |
| Grep search | `internal/search/grep.go` | 60% | Adapt for pages (title + body instead of issue fields) |
| CLI root command | `cmd/jira-mgmt/main.go` | 90% | Change name, global flags |
| CLI helpers | `cmd/jira-mgmt/helpers.go` | 90% | Change client builder |
| CLI query command | `cmd/jira-mgmt/cmd_query.go` | 95% | Same pattern, different executor |
| SKILL.md template | `agents/skills/jira-management/SKILL.md` | 90% | New triggers, new commands, new examples |
| Reference docs structure | `agents/skills/jira-management/references/` | 90% | New content, same organization |
| Tests structure | `*_test.go` files throughout | 80% | New test cases, same patterns |

### Estimated Total Effort

With this reuse level, the project is roughly **40-50% new code** (domain types, API operations, field mappings, CQL handling, v1/v2 routing) and **50-60% adapted code** from jira-management.

---

## 7. What's New (Not in jira-management)

### 7.1 Confluence Storage Format (XHTML-based)

Jira uses ADF (JSON) for rich text. Confluence uses XHTML storage format with Confluence-specific elements like `<ac:structured-macro>`, `<ac:rich-text-body>`, `<ac:link>`, etc. The CLI needs to:
- Pass through storage format without modification for reads/writes
- Provide a `StorageFormatToText()` utility for stripping tags when showing plain-text summaries
- Handle proper XML escaping in write payloads

### 7.2 Page Tree / Hierarchy Navigation

Confluence has a deep page tree structure that Jira lacks. New operations:
- **Children**: list direct children of a page
- **Ancestors**: get breadcrumb chain (root to parent)
- **Tree**: recursive traversal (children of children) -- client-side recursive fetch
- These require dedicated v2 endpoints (`/pages/{id}/children`, `/pages/{id}/ancestors`)

### 7.3 CQL Instead of JQL

Different query language with different fields and operators. Key differences:
- `space` field (not `project`)
- `ancestor` / `parent` fields for hierarchy queries
- `type` values: `page`, `blogpost`, `comment`, `attachment` (not issue types)
- `text ~ "..."` for full-text search (same operator, different index)
- Date functions: same `now()`, `startOfDay()`, etc.
- `label` field for filtering by labels

Need a `cql-patterns.md` reference doc for agents.

### 7.4 Space Key to ID Resolution

v2 API uses numeric space IDs everywhere. Users think in space keys (e.g., "DEV", "DOCS"). The CLI needs:
- A resolver that translates key -> ID via `GET /wiki/api/v2/spaces?keys=KEY`
- In-memory caching (spaces don't change often)
- Config stores `active_space` as key (human-friendly), resolver handles the rest

### 7.5 v1/v2 API Routing Logic

jira-management has a simpler split: Cloud v3 or Server v2. Confluence Cloud needs BOTH v1 and v2 simultaneously:
- v2 for all CRUD operations
- v1 for CQL search (no v2 equivalent)
- The HTTP client needs a method/path-based router, not just a base URL switch

### 7.6 Version-Aware Updates

Page updates require reading current version first, then incrementing. The update flow is:
```
GET page -> extract version.number -> PUT with version.number + 1
```
This is more rigid than Jira issue updates which don't require version tracking.

---

## 8. Proposed DSL Operations

### Grammar (inherited from jira-management)

```
batch     = query (";" query)*
query     = operation "(" args ")" [ "{" fields "}" ]
args      = arg ("," arg)*  |  empty
arg       = ident "=" value  |  value
fields    = ident+
value     = ident | quoted_string
```

### Operations

| Operation | Purpose | Example | API Call(s) |
|-----------|---------|---------|-------------|
| `get(PAGE-ID)` | Single page by ID | `get(12345){full}` | `GET /wiki/api/v2/pages/12345?body-format=storage` |
| `get(space=KEY,title="Page Title")` | Page by title in space | `get(space=DEV,title="Architecture"){default}` | Resolve space key -> ID, `GET /wiki/api/v2/pages?space-id=X&title=Y` |
| `list(space=KEY)` | List pages in space | `list(space=DEV,status=current){minimal}` | `GET /wiki/api/v2/pages?space-id=X` |
| `list(space=KEY,type=page)` | Filtered page list | `list(space=DEV,label=api-docs){overview}` | CQL search via v1 (labels require CQL) |
| `tree(PAGE-ID)` | Page tree (children recursive) | `tree(12345){minimal}` | `GET /wiki/api/v2/pages/{id}/children` (recursive) |
| `children(PAGE-ID)` | Direct children only | `children(12345){default}` | `GET /wiki/api/v2/pages/{id}/children` |
| `ancestors(PAGE-ID)` | Breadcrumb chain | `ancestors(12345){minimal}` | `GET /wiki/api/v2/pages/{id}/ancestors` |
| `search("CQL query")` | CQL search passthrough | `search("type=page AND space=DEV AND text~\"API\""){default}` | `GET /wiki/rest/api/search?cql=...` (v1) |
| `spaces()` | List all spaces | `spaces(){default}` | `GET /wiki/api/v2/spaces` |
| `history(PAGE-ID)` | Version history | `history(12345)` | `GET /wiki/api/v2/pages/{id}/versions` |

### Field Presets

#### Page fields

| Preset | Fields | When to use |
|--------|--------|-------------|
| `minimal` | `id`, `title`, `status` | Listings, tree views, batch operations |
| `default` | `id`, `title`, `status`, `spaceKey`, `version`, `url` | Standard reads |
| `overview` | `id`, `title`, `status`, `spaceKey`, `version`, `ancestors`, `labels`, `url` | Navigation, context |
| `full` | `id`, `title`, `status`, `spaceKey`, `version`, `ancestors`, `labels`, `body`, `created`, `updated`, `creator`, `lastModifier`, `url` | Full page read |

#### Space fields

| Preset | Fields |
|--------|--------|
| `minimal` | `id`, `key`, `name` |
| `default` | `id`, `key`, `name`, `type`, `status` |
| `full` | `id`, `key`, `name`, `type`, `status`, `description`, `homepageId` |

### Batch Support

Multiple queries separated by `;`:
```bash
confluence-mgmt q 'spaces(){minimal}; list(space=DEV){default}; get(12345){full}'
```

### Label Filtering via DSL

When `list()` includes `label=X`, the operation must route through CQL search (v1) since v2 page listing doesn't support label filtering:
```
list(space=DEV, label=release-notes) -> CQL: type=page AND space="DEV" AND label="release-notes"
```

---

## 9. Open Questions / Risks

### 9.1 ADF vs Storage Format

Should the CLI support both `storage` (XHTML) and `atlas_doc_format` (ADF JSON) for body content? ADF is the forward-looking format but harder to work with. **Recommendation:** Default to `storage` format -- it's simpler, widely supported, and what most existing content uses. Add `--body-format adf` flag for future use.

### 9.2 Tree Depth Limits

Recursive tree traversal (`tree()` operation) could be expensive for deep hierarchies. Need to decide:
- Max depth parameter? (e.g., `tree(12345, depth=3)`)
- Default depth limit?
- **Recommendation:** Default depth 3, configurable via `depth=N` arg, max 10.

### 9.3 Large Page Bodies

No documented hard limit on `body.storage` size. Very large pages may cause issues. The CLI should handle large responses gracefully but doesn't need to impose artificial limits.

### 9.4 Space Key Uniqueness

Space keys are globally unique within an instance. The config stores `active_space` as a key. If a user works across multiple instances, they need to switch config. This is the same pattern as jira-management's `active_project`. No change needed.

### 9.5 API Token Rate Limits (Nov 2025)

Atlassian enforces rate limits on API token traffic since November 2025 but hasn't published exact numbers. The CLI must handle 429 responses gracefully. The retry logic from jira-management handles this, but we should monitor Atlassian's announcements for concrete numbers.

### 9.6 v1 Deprecation Risk

Most v1 endpoints with v2 equivalents are being deprecated. The CQL search endpoint (`/rest/api/search`) has NO v2 equivalent and is expected to remain. However, Atlassian's deprecation timeline has been unpredictable. Monitor the Confluence Cloud changelog.

### 9.7 Server/DC PAT Scope

On Server/DC, PATs are product-specific. A Jira PAT does NOT work for Confluence. Users with both products need separate tokens. The auth setup flow should make this clear.

### 9.8 Scoped API Tokens and Base URL

Scoped API tokens use a different base URL: `https://api.atlassian.com/ex/confluence/{cloudId}/...` instead of `https://domain.atlassian.net/wiki/...`. The CLI could support this in v2 but it adds complexity (cloudId lookup). **Recommendation:** Support classic (unscoped) tokens only in v1. Add scoped token support later if demanded.

---

## 10. Recommended Implementation Order

### Phase 1: Foundation (Days 1-2)

1. **Project scaffold**: `go mod init`, directory structure, `.gitignore`, Makefile
2. **Config module**: Copy from jira-management, adapt keys (`active_space`, etc.), change config path to `~/.config/confluence-mgmt/`
3. **Auth module**: Copy credential store, change service name to `confluence-mgmt`
4. **Auth CLI command**: `confluence-mgmt auth --url X --email Y --token Z` (and `--pat` for Server/DC)
5. **Config CLI commands**: `config set`, `config show`
6. **Scripts**: `setup.sh`, `deinit.sh` (adapt from jira-management)

### Phase 2: HTTP Client & Domain Types (Days 2-3)

7. **Domain types**: `Page`, `Space`, `Label`, `Version`, `Ancestor`, `SearchResult`
8. **HTTP client**: Copy from jira-management, add v1/v2 routing, `/wiki` prefix, Confluence base paths
9. **Space key resolver**: Translate space key -> space ID for v2 operations, with caching
10. **Pagination helpers**: Both cursor-based (v2) and offset-based (v1)

### Phase 3: Core Read Operations (Days 3-4)

11. **Get page by ID**: v2 endpoint, with `body-format=storage`
12. **List pages**: v2 endpoint, with space/title filtering
13. **Get children**: v2 endpoint
14. **Get ancestors**: v2 endpoint
15. **CQL search**: v1 endpoint, with result projection
16. **List spaces**: v2 endpoint

### Phase 4: DSL Layer (Days 4-5)

17. **DSL tokenizer**: Copy from jira-management (domain-agnostic)
18. **DSL parser**: Adapt operations (`get`, `list`, `tree`, `children`, `ancestors`, `search`, `spaces`, `history`), field names, presets
19. **DSL executor**: New operation handlers mapping DSL to API calls
20. **Field selector**: New Confluence field mappings, same Apply/ApplyMany
21. **Query CLI command**: `confluence-mgmt q '<query>'`

### Phase 5: Write Operations (Days 5-6)

22. **Create page**: `page create --space X --title Y --body Z --parent P`
23. **Update page**: `page update PAGE-ID --title Y --body Z --message M` (with version increment)
24. **Label management**: `label add`, `label remove`
25. **Grep command**: Regex search across pages

### Phase 6: Skill Files & Polish (Days 6-7)

26. **SKILL.md**: Write agent-facing instructions with triggers, command overview, usage patterns
27. **Reference docs**: `cli-commands.md`, `dsl-examples.md`, `cql-patterns.md`, `workflows.md`, `troubleshooting.md`, `dev-notes.md`
28. **Symlink setup**: `agents/skills/confluence-management/` -> `.claude/skills/` and `.codex/skills/`
29. **README.md**: Project documentation
30. **End-to-end testing**: Full workflow tests against a real Confluence instance

### Phase 7: Advanced (Post-MVP)

- `page move` command
- `page archive` command
- Tree traversal with depth control
- `compact` output format tuning
- Scoped API token support
- `--body-file` flag for large content
- Attachment support (out of scope for v1 per spec)

---

**Document Version:** 1.0
**Last Updated:** 2026-02-13
**Sources:** api-versions-landscape.md, authentication-mechanisms.md, key-endpoints-and-schemas.md, rate-limits-and-constraints.md, jira-management-study.md, confluence-api-skill.md (spec)
