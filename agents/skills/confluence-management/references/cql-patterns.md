# CQL Patterns for Agents

CQL (Confluence Query Language) is used for searching. Always goes through v1 API.

## Fields

| Field | Description | Example |
|-------|-------------|---------|
| `type` | Content type | `type=page`, `type=blogpost` |
| `space` | Space key | `space="DEV"` |
| `title` | Page title | `title="Architecture"` |
| `text` | Full-text body search | `text~"migration"` |
| `label` | Label name | `label="api-docs"` |
| `ancestor` | Under parent (recursive) | `ancestor=12345` |
| `parent` | Direct parent | `parent=12345` |
| `creator` | Created by user | `creator=currentUser()` |
| `contributor` | Any contributor | `contributor=currentUser()` |
| `created` | Creation date | `created >= "2026-01-01"` |
| `lastmodified` | Last modified date | `lastmodified >= now("-7d")` |
| `id` | Content ID | `id=12345` |

## Operators

| Operator | Meaning | Example |
|----------|---------|---------|
| `=` | Exact match | `type=page` |
| `!=` | Not equal | `status!=draft` |
| `~` | Contains (text) | `text~"migration"` |
| `!~` | Does not contain | `title!~"Draft"` |
| `>`, `>=`, `<`, `<=` | Comparison | `created >= "2026-01-01"` |
| `IN` | Set membership | `space IN ("DEV","DOCS")` |
| `NOT IN` | Not in set | `label NOT IN ("draft","wip")` |

## Keywords

| Keyword | Usage |
|---------|-------|
| `AND` | Combine conditions |
| `OR` | Alternative conditions |
| `NOT` | Negate |
| `ORDER BY` | Sort results |

## Date Functions

| Function | Description |
|----------|-------------|
| `now()` | Current timestamp |
| `now("-7d")` | 7 days ago |
| `now("-1M")` | 1 month ago |
| `startOfDay()` | Start of today |
| `startOfWeek()` | Start of this week |
| `startOfMonth()` | Start of this month |
| `startOfYear()` | Start of this year |

## Common Patterns

```cql
# All pages in a space
type=page AND space="DEV"

# Full-text search in space
type=page AND space="DEV" AND text~"API documentation"

# Pages by label
type=page AND label="release-notes"

# Pages under a parent (recursive)
type=page AND ancestor=12345

# Recently modified by me
type=page AND contributor=currentUser() AND lastmodified >= now("-7d")

# Pages created this month
type=page AND space="DEV" AND created >= startOfMonth()

# Search with ordering
type=page AND space="DEV" ORDER BY lastmodified DESC

# Multiple labels
type=page AND label="api-docs" AND label="v2"

# Exclude drafts
type=page AND space="DEV" AND status=current
```

## Limits

- Without body expansion: up to 1000 results per page
- With body.storage: max 50 results per page
- With body.export_view: max 25 results per page
- Recommendation: don't expand body in search. Fetch body separately for specific pages.
