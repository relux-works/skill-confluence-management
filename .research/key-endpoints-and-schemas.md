# Confluence API: Key Endpoints and Schemas

Reference for building a Confluence CLI tool. Covers Cloud (v1 + v2) and Server/Data Center APIs.

**API base paths:**
- Cloud v2: `https://{domain}.atlassian.net/wiki/api/v2/`
- Cloud v1: `https://{domain}.atlassian.net/wiki/rest/api/`
- Server/DC: `http://{host}:{port}/confluence/rest/api/`

**Auth:**
- Cloud: Basic Auth (`email:api-token`), header `Authorization: Basic base64(email:token)`
- Server/DC: Basic Auth (`username:password`) or Bearer PAT, header `Authorization: Bearer {pat}`

---

## 1. Read Page

### v2: GET /pages/{id}

**URL:** `/wiki/api/v2/pages/{id}`

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `body-format` | string | _(none)_ | **Required to get body.** Values: `storage`, `atlas_doc_format`. Without this, `body` returns `{}`. |
| `version` | integer | _(current)_ | Retrieve a specific previously published version. |
| `include-labels` | boolean | false | Include labels in response. |
| `include-properties` | boolean | false | Include content properties. |
| `include-operations` | boolean | false | Include available operations. |
| `include-versions` | boolean | false | Include version history. |
| `include-version` | boolean | true | Include current version object. |
| `include-collaborators` | boolean | false | Include collaborator info. |

**Response (200):**

```json
{
  "id": "123456",
  "status": "current",
  "title": "Page Title",
  "spaceId": "789",
  "parentId": "111222",
  "parentType": "page",
  "position": 0,
  "authorId": "5a1234...",
  "ownerId": "5a1234...",
  "lastOwnerId": "5a1234...",
  "createdAt": "2024-01-15T10:30:00.000Z",
  "version": {
    "number": 5,
    "message": "Updated section 3",
    "createdAt": "2024-06-01T14:20:00.000Z",
    "authorId": "5a1234..."
  },
  "body": {
    "storage": {
      "value": "<p>Page content in XHTML storage format</p>",
      "representation": "storage"
    }
  },
  "_links": {
    "webui": "/spaces/SPACEKEY/pages/123456/Page+Title",
    "editui": "/pages/resumedraft.action?draftId=123456",
    "tinyui": "/x/abc123"
  }
}
```

**Key notes:**
- `body-format=storage` is the most useful for CLI read/write. Returns XHTML storage format.
- `atlas_doc_format` returns Atlassian Document Format (ADF JSON). Useful for new editor content.
- Without `body-format`, the `body` field is an empty object `{}`.

### v2: GET /pages (list/filter)

**URL:** `/wiki/api/v2/pages`

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `title` | string | | Filter by exact title. |
| `space-id` | string | | Filter by space ID. |
| `status` | array[string] | `[current, archived]` | Filter: `current`, `archived`, `trashed`. |
| `sort` | string | | Sort field (e.g., `title`, `-modified-date`). |
| `body-format` | string | | Same as above. |
| `cursor` | string | | Pagination cursor (from `Link` header). |
| `limit` | integer | 50 | Results per page (1-250). |

**Response (200):**

```json
{
  "results": [ /* array of Page objects (same schema as GET /pages/{id}) */ ],
  "_links": {
    "next": "/wiki/api/v2/pages?cursor=abc123&limit=50",
    "base": "https://domain.atlassian.net/wiki"
  }
}
```

**Pagination:** Cursor-based. The `Link` response header contains `rel="next"` URL. Follow it until no `next` link is returned.

**Use case: Get page by title:**
```
GET /wiki/api/v2/pages?title=My+Page+Title&space-id=12345&body-format=storage
```

### v1: GET /content/{id}

**URL:** `/wiki/rest/api/content/{id}`

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `expand` | string | | Comma-separated list of properties to expand: `body.storage`, `body.view`, `version`, `history`, `space`, `ancestors`, `children`, `metadata.labels`. |
| `status` | string | | Content status filter. |
| `version` | integer | | Specific version number. |

**Response (200):**

```json
{
  "id": "3604482",
  "type": "page",
  "status": "current",
  "title": "Page Title",
  "space": {
    "id": 789,
    "key": "TST",
    "name": "Test Space",
    "type": "global"
  },
  "version": {
    "by": {
      "type": "known",
      "accountId": "5a1234...",
      "displayName": "John Doe"
    },
    "number": 5,
    "when": "2024-06-01T14:20:00.000Z",
    "message": "Updated section 3",
    "minorEdit": false
  },
  "body": {
    "storage": {
      "value": "<p>Content in XHTML</p>",
      "representation": "storage"
    }
  },
  "ancestors": [
    { "id": "111000", "type": "page", "title": "Parent Page" },
    { "id": "111222", "type": "page", "title": "Grandparent Page" }
  ],
  "history": {
    "createdBy": { "displayName": "John Doe", "accountId": "5a1234..." },
    "createdDate": "2024-01-15T10:30:00.000Z",
    "lastUpdated": {
      "by": { "displayName": "Jane Doe", "accountId": "5b5678..." },
      "when": "2024-06-01T14:20:00.000Z",
      "number": 5
    }
  },
  "_links": {
    "webui": "/display/TST/Page+Title",
    "self": "https://domain.atlassian.net/wiki/rest/api/content/3604482"
  },
  "_expandable": {
    "children": "/rest/api/content/3604482/child",
    "descendants": "/rest/api/content/3604482/descendant",
    "metadata": ""
  }
}
```

**Key notes:**
- v1 uses `expand` parameter (comma-separated, dot-notation for nesting).
- Without `expand=body.storage`, body is not returned.
- v1 returns `space` as a nested object with `key` (v2 returns `spaceId` as a flat string).
- `ancestors` returns an ordered array from root to immediate parent.
- `_expandable` shows what additional properties are available.

**Common expand combinations:**
- Read page with content: `?expand=body.storage,version`
- Full page details: `?expand=body.storage,version,space,history,ancestors,metadata.labels`
- Page tree navigation: `?expand=ancestors,children.page`

### Server/DC: Same as v1

Server/DC uses the same v1 endpoint pattern at `/rest/api/content/{id}`. Same expand parameters. Same response schema.

```bash
curl -u admin:admin \
  "http://localhost:8080/confluence/rest/api/content/3965072?expand=body.storage,version"
```

---

## 2. Search (CQL)

CQL search is **v1 only** (no v2 equivalent). Two endpoints exist:

### GET /search (primary)

**URL:** `/wiki/rest/api/search`

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `cql` | string | **required** | CQL query string. |
| `cqlcontext` | string | | JSON context for CQL (e.g., `{"spaceKey":"TST"}`). |
| `cursor` | string | | Cursor-based pagination token (preferred). |
| `start` | integer | 0 | Offset-based pagination start index. |
| `limit` | integer | 25 | Max results (reduced to max 25 when expanding body.export_view or body.styled_view). |
| `expand` | string | | Properties to expand on each result's `content` object. |
| `excerpt` | string | | Controls excerpt generation (`highlight`, `indexed`, `none`). |
| `includeArchivedSpaces` | boolean | false | Include results from archived spaces. |
| `excludeCurrentSpaces` | boolean | false | Exclude current spaces. |
| `sitePermissionTypeFilter` | string | | Filter by permission type. |

**Response (200):**

```json
{
  "results": [
    {
      "title": "Page Title",
      "excerpt": "...matched <b>keyword</b> in context...",
      "url": "/spaces/TST/pages/123456/Page+Title",
      "resultParentContainer": {
        "title": "Parent Space",
        "displayUrl": "/spaces/TST"
      },
      "content": {
        "id": "123456",
        "type": "page",
        "status": "current",
        "title": "Page Title",
        "space": { "key": "TST", "name": "Test Space" },
        "version": { "number": 5 },
        "body": { /* only if expanded */ },
        "_links": { "webui": "/spaces/TST/pages/123456/Page+Title" }
      },
      "space": {
        "key": "TST",
        "name": "Test Space"
      },
      "lastModified": "2024-06-01T14:20:00.000Z",
      "friendlyLastModified": "Jun 01, 2024",
      "score": 1.234
    }
  ],
  "start": 0,
  "limit": 25,
  "size": 25,
  "totalSize": 142,
  "_links": {
    "base": "https://domain.atlassian.net/wiki",
    "context": "/wiki",
    "next": "/rest/api/search?cql=type%3Dpage&cursor=abc123&limit=25",
    "self": "https://domain.atlassian.net/wiki/rest/api/search?cql=type%3Dpage"
  }
}
```

### GET /content/search (alternative)

**URL:** `/wiki/rest/api/content/search`

Same `cql` and pagination parameters. Returns `Content` objects directly (not wrapped in `SearchResult`). Simpler schema when you only need content, not search metadata (excerpt, score).

**Response:**

```json
{
  "results": [
    {
      "id": "123456",
      "type": "page",
      "title": "Page Title",
      "space": { "key": "TST" },
      "_links": { "webui": "..." }
    }
  ],
  "start": 0,
  "limit": 25,
  "size": 10,
  "_links": { "next": "..." }
}
```

### CQL Syntax Reference

**Fields:**

| Field | Description | Operators |
|-------|-------------|-----------|
| `type` | Content type: `page`, `blogpost`, `comment`, `attachment` | `=`, `!=`, `IN` |
| `space` | Space key | `=`, `!=`, `IN`, `NOT IN` |
| `space.type` | Space type: `personal`, `global` | `=`, `!=` |
| `title` | Page title | `=`, `!=`, `~` |
| `text` | Full-text (title + body + labels) | `~`, `!~` |
| `label` | Label name | `=`, `!=`, `IN`, `NOT IN` |
| `ancestor` | Ancestor page ID (finds all descendants) | `=` |
| `parent` | Direct parent page ID | `=` |
| `creator` | Content creator (user) | `=`, `!=` |
| `contributor` | Creator or editor | `=`, `!=` |
| `created` | Creation date | `=`, `!=`, `>`, `>=`, `<`, `<=` |
| `lastmodified` | Last modification date | `=`, `!=`, `>`, `>=`, `<`, `<=` |
| `id` | Content ID | `=`, `!=`, `IN` |
| `mention` | Mentioned user | `=` |
| `macro` | Macro name in content | `=` |
| `favourite` / `favorite` | Favorited by user | `=` |
| `watcher` | Watched by user | `=` |

**Operators:**

| Operator | Meaning | Example |
|----------|---------|---------|
| `=` | Exact match | `space = "TST"` |
| `!=` | Not equal | `type != "blogpost"` |
| `~` | Contains (text search) | `title ~ "API"` |
| `!~` | Does not contain | `text !~ "draft"` |
| `>`, `>=`, `<`, `<=` | Comparison (dates/numbers) | `created > "2024-01-01"` |
| `IN` | Multiple values | `label IN ("api", "docs")` |
| `NOT IN` | Exclude multiple values | `space NOT IN ("TEMP", "TRASH")` |

**Keywords:** `AND`, `OR`, `NOT`, `ORDER BY` (with `asc`/`desc`)

**Date functions:** `now()`, `startOfDay()`, `endOfDay()`, `startOfWeek()`, `endOfWeek()`, `startOfMonth()`, `endOfMonth()`, `startOfYear()`, `endOfYear()`. All accept optional increment, e.g., `now("-4w")`, `startOfDay("+1d")`.

**User function:** `currentUser()`

**Text search modifiers (~ operator):**
- Wildcards: `?` (single char), `*` (multi char). E.g., `title ~ "win*"`
- Phrase search: `text ~ "\"exact phrase\""`
- Fuzzy search: `text ~ "roam~"` (finds "foam", "roams")
- All queries are case-insensitive.

**Useful CQL queries for a CLI tool:**

```
# Find page by title in space
type = page AND space = "TST" AND title = "My Page"

# Full-text search in a space
type = page AND space = "TST" AND text ~ "search term"

# Recently modified pages
type = page AND space = "TST" AND lastmodified > now("-7d") ORDER BY lastmodified DESC

# Pages with specific label
type = page AND label = "release-notes"

# Child pages of a parent
type = page AND parent = "123456"

# All descendants under an ancestor
type = page AND ancestor = "123456"

# Pages modified by current user
type = page AND contributor = currentUser() ORDER BY lastmodified DESC
```

### Server/DC: Same endpoints

Server/DC uses the same v1 search endpoints at `/rest/api/search` and `/rest/api/content/search`. Same CQL syntax.

---

## 3. Page Tree / Children

### v2: GET /pages/{id}/children

**URL:** `/wiki/api/v2/pages/{id}/children`

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `cursor` | string | | Pagination cursor. |
| `limit` | integer | 50 | Results per page (1-250). |
| `sort` | string | | Sort field. |
| `status` | string | | Filter: `current`, `archived`, `trashed`. |

**Response (200):**

```json
{
  "results": [
    {
      "id": "234567",
      "status": "current",
      "title": "Child Page 1",
      "spaceId": "789",
      "parentId": "123456",
      "parentType": "page",
      "position": 0,
      "authorId": "5a1234...",
      "createdAt": "2024-03-01T09:00:00.000Z",
      "version": { "number": 2, "createdAt": "2024-05-01T12:00:00.000Z" }
    },
    {
      "id": "234568",
      "status": "current",
      "title": "Child Page 2",
      "parentId": "123456",
      "position": 1
    }
  ],
  "_links": {
    "next": "/wiki/api/v2/pages/123456/children?cursor=xyz789"
  }
}
```

**Pagination:** Cursor-based, via `Link` header.

### v2: GET /pages/{id}/ancestors

**URL:** `/wiki/api/v2/pages/{id}/ancestors`

Returns the ancestor chain (breadcrumbs) for a page.

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `cursor` | string | | Pagination cursor. |
| `limit` | integer | 25 | Results per page (1-250). |
| `sort` | string | | Sort field. |

**Response (200):**

```json
{
  "results": [
    { "id": "100000", "title": "Space Home", "parentType": "space" },
    { "id": "111000", "title": "Section A", "parentId": "100000", "parentType": "page" },
    { "id": "111222", "title": "Subsection B", "parentId": "111000", "parentType": "page" }
  ],
  "_links": {}
}
```

**Note:** Ancestors are ordered from the root (space home) to the immediate parent.

### v1: GET /content/{id}/child/page

**URL:** `/wiki/rest/api/content/{id}/child/page`

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `expand` | string | | Expand properties on children. |
| `start` | integer | 0 | Offset pagination start. |
| `limit` | integer | 25 | Results per page. |

**Response (200):**

```json
{
  "results": [
    {
      "id": "234567",
      "type": "page",
      "title": "Child Page 1",
      "status": "current",
      "_links": { "webui": "/display/TST/Child+Page+1" }
    }
  ],
  "start": 0,
  "limit": 25,
  "size": 2,
  "_links": { "next": "/rest/api/content/123456/child/page?start=25" }
}
```

**Pagination:** Offset-based (`start` + `limit`).

### v1: Ancestors via expand

In v1, ancestors are retrieved by expanding the `ancestors` property on a content GET:

```
GET /wiki/rest/api/content/{id}?expand=ancestors
```

Returns `ancestors` as an array from root to immediate parent.

### Server/DC: Same as v1

Same endpoint: `/rest/api/content/{id}/child/page`.

---

## 4. Create Page

### v2: POST /pages

**URL:** `/wiki/api/v2/pages`

**Request body:**

```json
{
  "spaceId": "789",
  "status": "current",
  "title": "New Page Title",
  "parentId": "123456",
  "body": {
    "representation": "storage",
    "value": "<p>Page content in XHTML storage format</p>"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `spaceId` | string | **yes** | Space ID (not key). |
| `title` | string | **yes** | Page title. |
| `status` | string | **yes** | `current` (published) or `draft`. |
| `parentId` | string | no | Parent page ID. If omitted, created at space root. |
| `body` | object | no | Page body content. |
| `body.representation` | string | yes (if body) | `storage` or `atlas_doc_format`. |
| `body.value` | string | yes (if body) | Content in the specified format. |

**Response (200):** Full Page object (same schema as GET /pages/{id}).

**Key notes:**
- v2 uses `spaceId` (numeric/string ID), NOT `spaceKey`. You may need to look up the space ID first.
- `parentId` lets you create child pages. Without it, the page is created at space root level.
- `storage` representation is XHTML-based Confluence storage format.
- `atlas_doc_format` is Atlassian Document Format (ADF) JSON — used by the new editor.

### v1: POST /content

**URL:** `/wiki/rest/api/content`

**Request body:**

```json
{
  "type": "page",
  "title": "New Page Title",
  "space": {
    "key": "TST"
  },
  "ancestors": [
    { "id": "123456" }
  ],
  "body": {
    "storage": {
      "value": "<p>Page content in XHTML storage format</p>",
      "representation": "storage"
    }
  },
  "status": "current"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | **yes** | `page` or `blogpost`. |
| `title` | string | **yes** | Page title. |
| `space.key` | string | **yes** | Space key (e.g., `"TST"`). |
| `ancestors` | array | no | Array with single object `{ "id": "parentId" }`. Creates child page. |
| `body.storage.value` | string | yes | XHTML content. |
| `body.storage.representation` | string | yes | `"storage"`. |
| `status` | string | no | `"current"` (default) or `"draft"`. |

**Response (200):** Full Content object.

**Key difference v1 vs v2:**
- v1 uses `space.key` (string key like `"TST"`), v2 uses `spaceId` (numeric ID).
- v1 uses `ancestors: [{ "id": "..." }]` for parent, v2 uses `parentId`.
- v1 body is nested: `body.storage.value`, v2 body is flat: `body.value` + `body.representation`.

### Server/DC: Same as v1

```bash
curl -u admin:admin -X POST -H 'Content-Type: application/json' \
  -d '{"type":"page","title":"New Page","space":{"key":"TST"},"body":{"storage":{"value":"<p>Content</p>","representation":"storage"}}}' \
  http://localhost:8080/confluence/rest/api/content/
```

---

## 5. Update Page

### v2: PUT /pages/{id}

**URL:** `/wiki/api/v2/pages/{id}`

**Request body:**

```json
{
  "id": "123456",
  "status": "current",
  "title": "Updated Page Title",
  "body": {
    "representation": "storage",
    "value": "<p>Updated content</p>"
  },
  "version": {
    "number": 6,
    "message": "Updated via CLI"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | **yes** | Page ID (must match path param). |
| `status` | string | **yes** | `current` or `draft`. |
| `title` | string | **yes** | Page title (can be changed). |
| `body` | object | no | Updated body content. |
| `body.representation` | string | yes (if body) | `storage` or `atlas_doc_format`. |
| `body.value` | string | yes (if body) | Updated content. |
| `version` | object | **yes** | Version info. |
| `version.number` | integer | **yes** | Must be **current version + 1**. |
| `version.message` | string | no | Change description / commit message. |

**Response (200):** Full Page object with incremented version.

### v1: PUT /content/{id}

**URL:** `/wiki/rest/api/content/{id}`

**Request body:**

```json
{
  "id": "3604482",
  "type": "page",
  "title": "Updated Page Title",
  "space": {
    "key": "TST"
  },
  "body": {
    "storage": {
      "value": "<p>Updated content</p>",
      "representation": "storage"
    }
  },
  "version": {
    "number": 6
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | **yes** | Page ID. |
| `type` | string | **yes** | `page` or `blogpost`. |
| `title` | string | **yes** | Page title. |
| `space.key` | string | no | Space key (usually included). |
| `body.storage.value` | string | yes | Updated XHTML content. |
| `body.storage.representation` | string | yes | `"storage"`. |
| `version.number` | integer | **yes** | **Current version + 1**. |
| `version.message` | string | no | Change message. |

**Version increment workflow:**
1. GET the page to read `version.number` (e.g., returns `5`).
2. PUT with `version.number` set to `6`.
3. If you send a stale version number, the API returns `409 Conflict`.

### Server/DC: Same as v1

```bash
curl -u admin:admin -X PUT -H 'Content-Type: application/json' \
  -d '{"id":"3604482","type":"page","title":"Updated","space":{"key":"TST"},"body":{"storage":{"value":"<p>New content</p>","representation":"storage"}},"version":{"number":2}}' \
  http://localhost:8080/confluence/rest/api/content/3604482
```

---

## 6. Labels

### v2: Label Endpoints

#### GET labels on a page

**URL:** `GET /wiki/api/v2/pages/{id}/labels`

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `prefix` | string | | Filter: `my`, `team`, `global`, `system`. |
| `sort` | string | | Sort field. |
| `cursor` | string | | Pagination cursor. |
| `limit` | integer | 25 | Results per page (1-250). |

**Response (200):**

```json
{
  "results": [
    { "id": "label-1", "prefix": "global", "name": "release-notes" },
    { "id": "label-2", "prefix": "global", "name": "api-docs" }
  ],
  "_links": {}
}
```

#### Add labels to a page

**URL:** `POST /wiki/api/v2/pages/{id}/labels`

**Request body:**

```json
[
  { "prefix": "global", "name": "new-label" },
  { "prefix": "global", "name": "another-label" }
]
```

**Response (200):** Updated label collection.

#### Remove a label from a page

**URL:** `DELETE /wiki/api/v2/pages/{id}/labels/{label-id}`

**Response:** `204 No Content`.

### v1: Label Endpoints

#### GET labels

**URL:** `GET /wiki/rest/api/content/{id}/label`

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `prefix` | string | | Filter by prefix. |
| `start` | integer | 0 | Offset pagination. |
| `limit` | integer | 200 | Max results. |

**Response:**

```json
{
  "results": [
    { "id": 12345, "name": "release-notes", "prefix": "global" },
    { "id": 12346, "name": "api-docs", "prefix": "global" }
  ],
  "start": 0,
  "limit": 200,
  "size": 2
}
```

#### Add labels

**URL:** `POST /wiki/rest/api/content/{id}/label`

**Request body:**

```json
[
  { "prefix": "global", "name": "new-label" },
  { "prefix": "global", "name": "another-label" }
]
```

#### Remove a label

Two approaches:

**By query param (if label name contains `/`):**
```
DELETE /wiki/rest/api/content/{id}/label?name={labelName}
```

**By path param:**
```
DELETE /wiki/rest/api/content/{id}/label/{labelName}
```

**Response:** `204 No Content`.

### Label Object Schema

```
{
  "id": string | integer,    // unique identifier
  "prefix": string,          // "global" | "my" | "team" | "system"
  "name": string,            // the label text (e.g., "release-notes")
  "owner"?: object           // (v1 only) user who created the label
}
```

**Label prefixes:**
- `global` — visible to all users (most common for CLI use).
- `my` — personal label, visible only to the creator.
- `team` — team-scoped label.
- `system` — system-managed label.

### Server/DC: Same as v1

Same endpoints at `/rest/api/content/{id}/label`.

---

## 7. Spaces

### v2: GET /spaces (list)

**URL:** `/wiki/api/v2/spaces`

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `status` | string | | Filter: `current`, `archived`, `trashed`. |
| `sort` | string | | Sort field (e.g., `name`, `-key`). |
| `cursor` | string | | Pagination cursor. |
| `limit` | integer | 50 | Results per page (1-250). |

**Response (200):**

```json
{
  "results": [
    {
      "id": "789",
      "key": "TST",
      "name": "Test Space",
      "type": "global",
      "status": "current",
      "description": { "plain": { "value": "Space description" } },
      "homepageId": "100000",
      "icon": { "path": "/wiki/images/logo/default-space-logo.png" },
      "_links": {
        "webui": "/spaces/TST"
      }
    }
  ],
  "_links": {
    "next": "/wiki/api/v2/spaces?cursor=abc123"
  }
}
```

### v2: GET /spaces/{id}

**URL:** `/wiki/api/v2/spaces/{id}`

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `include-properties` | boolean | false | Include space properties. |
| `include-operations` | boolean | false | Include available operations. |

**Response (200):** Single Space object (same schema as list item).

**Scopes required:** `read:space:confluence`

### v1: GET /space (list)

**URL:** `/wiki/rest/api/space`

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `spaceKey` | string | | Filter by key. |
| `type` | string | | `global` or `personal`. |
| `status` | string | | `current` or `archived`. |
| `expand` | string | | `description`, `homepage`, `metadata.labels`. |
| `start` | integer | 0 | Offset pagination. |
| `limit` | integer | 25 | Results per page. |

**Response:**

```json
{
  "results": [
    {
      "id": 789,
      "key": "TST",
      "name": "Test Space",
      "type": "global",
      "status": "current",
      "_links": { "webui": "/spaces/TST" }
    }
  ],
  "start": 0,
  "limit": 25,
  "size": 5,
  "_links": {}
}
```

### Mapping space key to space ID

v2 uses `spaceId` everywhere (numeric), v1 uses `spaceKey` (string). To bridge:

```
# Get space ID from key (v2)
GET /wiki/api/v2/spaces?keys=TST
# Or via v1
GET /wiki/rest/api/space/TST
```

### Server/DC: Same as v1

Same endpoint: `/rest/api/space`.

---

## 8. Page Metadata

### Version Info

**v2:** `GET /wiki/api/v2/pages/{id}/versions`

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `cursor` | string | | Pagination cursor. |
| `limit` | integer | 50 | Results per page (1-250). |
| `sort` | string | | Sort field (e.g., `-modified-date`). |

**Response:**

```json
{
  "results": [
    {
      "number": 5,
      "message": "Updated section 3",
      "createdAt": "2024-06-01T14:20:00.000Z",
      "authorId": "5a1234..."
    },
    {
      "number": 4,
      "message": "Added diagrams",
      "createdAt": "2024-05-15T09:00:00.000Z",
      "authorId": "5b5678..."
    }
  ]
}
```

**v2 single version:** `GET /wiki/api/v2/pages/{id}/versions/{version-number}`

**v1:** Use `?expand=version,history` on the content GET:

```
GET /wiki/rest/api/content/{id}?expand=version,history,history.lastUpdated,history.previousVersion
```

### Last Author

**v2:** The `version.authorId` on the page object gives the last editor. The `authorId` at root level is the page creator. Use `ownerId` for the current owner.

**v1:** Expand `history.lastUpdated.by` or `version.by` to get the last author with `displayName` and `accountId`.

### Ancestors (Breadcrumbs)

**v2:** `GET /wiki/api/v2/pages/{id}/ancestors`

Returns ordered array from root to immediate parent. See Section 3 for full details.

**v1:** `GET /wiki/rest/api/content/{id}?expand=ancestors`

Returns `ancestors` array in the same root-to-parent order.

### Combining metadata in a single call

**v2:** Use `include-*` params on GET /pages/{id}:
```
GET /wiki/api/v2/pages/{id}?body-format=storage&include-labels=true&include-version=true
```

**v1:** Use expand for everything in one call:
```
GET /wiki/rest/api/content/{id}?expand=body.storage,version,history,history.lastUpdated,space,ancestors,metadata.labels
```

---

## Pagination Summary

| API | Approach | Parameters | How to paginate |
|-----|----------|------------|-----------------|
| v2 | **Cursor-based** | `cursor`, `limit` (1-250, default 50) | Follow `next` URL from `_links` or `Link` header. |
| v1 (content) | **Offset-based** | `start`, `limit` (default 25) | Increment `start` by `limit`. Stop when `size < limit`. |
| v1 (search) | **Cursor or offset** | `cursor` or `start`, `limit` | Prefer cursor from `_links.next`. Fallback to offset. |
| Server/DC | **Offset-based** | `start`, `limit` | Same as v1. |

---

## Quick Reference: v1 vs v2 Differences

| Aspect | v1 | v2 |
|--------|----|----|
| Base path | `/wiki/rest/api/` | `/wiki/api/v2/` |
| Space identifier | `space.key` (string, e.g., `"TST"`) | `spaceId` (string ID, e.g., `"789"`) |
| Parent page | `ancestors: [{ "id": "..." }]` | `parentId: "..."` |
| Body in response | `expand=body.storage` | `body-format=storage` |
| Body in request | `body.storage.value` + `body.storage.representation` | `body.value` + `body.representation` |
| Content type field | `type: "page"` | Not needed (separate endpoints per type) |
| Pagination | Offset (`start` + `limit`) | Cursor (`cursor` + `limit`) |
| CQL search | Yes (`/search`, `/content/search`) | Not available |
| Labels CRUD | Yes | Yes |
| Ancestors | Via `expand=ancestors` | Dedicated endpoint `/pages/{id}/ancestors` |
| Versions | Via `expand=version,history` | Dedicated endpoint `/pages/{id}/versions` |
