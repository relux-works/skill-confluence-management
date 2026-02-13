---
name: confluence-management
description: >
  Agent-facing CLI for Confluence Cloud and Server/DC.
  DSL queries for reads, explicit commands for writes.
  Supports pages, spaces, labels, CQL search, page tree navigation.
triggers:
  - confluence
  - confluence page
  - confluence search
  - confluence space
  - read confluence
  - write confluence
  - confluence api
  - confluence content
  - конфлюенс
  - конфлю
  - страница в конфлю
  - найди в конфлю
  - прочитай из конфлю
  - создай страницу
  - обнови страницу
  - поиск в конфлю
---

# Confluence Management Skill

Agent-facing CLI (`confluence-mgmt`) for Confluence Cloud and Server/DC.

## Quick Start

```bash
# Auth (Cloud — same token as Jira)
confluence-mgmt auth --instance https://company.atlassian.net/wiki --email user@company.com --token API_TOKEN

# Auth (Server/DC — separate PAT)
confluence-mgmt auth --instance https://confluence.company.com --token PAT_TOKEN

# Set active space
confluence-mgmt config set space DEV

# Read page
confluence-mgmt q 'get(12345){full}'

# Search
confluence-mgmt q 'search("type=page AND space=DEV AND text~\"API\""){default}'

# List spaces
confluence-mgmt q 'spaces(){minimal}'
```

## Commands Overview

### Reads (DSL via `q`)

| Operation | Description | Example |
|-----------|-------------|---------|
| `get(ID)` | Page by ID | `q 'get(12345){full}'` |
| `get(space=KEY,title="Title")` | Page by title | `q 'get(space=DEV,title="Architecture"){default}'` |
| `list(space=KEY)` | Pages in space | `q 'list(space=DEV){minimal}'` |
| `list(space=KEY,label=NAME)` | Pages by label | `q 'list(space=DEV,label=api-docs){default}'` |
| `search("CQL")` | CQL search | `q 'search("text~\"migration\""){default}'` |
| `children(ID)` | Direct children | `q 'children(12345){minimal}'` |
| `ancestors(ID)` | Breadcrumb chain | `q 'ancestors(12345){minimal}'` |
| `tree(ID)` | Recursive tree | `q 'tree(12345,depth=3){minimal}'` |
| `spaces()` | List spaces | `q 'spaces(){default}'` |

### Writes (explicit commands)

```bash
# Create page
confluence-mgmt page create --space DEV --title "New Page" --body "<p>Content</p>" --parent 12345

# Create from file
confluence-mgmt page create --space DEV --title "New Page" --body-file content.html

# Update page (auto-increments version)
confluence-mgmt page update 12345 --title "Updated" --body "<p>New content</p>" --message "Updated via CLI"

# Delete (trash) page
confluence-mgmt page delete 12345

# Labels
confluence-mgmt label add 12345 --labels "api-docs,v2"
confluence-mgmt label remove 12345 --labels "draft"
```

### Config

```bash
confluence-mgmt config show
confluence-mgmt config set space DEV
```

## Field Presets

| Preset | Fields |
|--------|--------|
| `minimal` | id, title, status |
| `default` | id, title, status, spaceKey, version, url |
| `overview` | id, title, status, spaceKey, version, ancestors, labels, url |
| `full` | id, title, status, spaceKey, version, ancestors, labels, body, created, updated, author, url |

## Batch Queries

Semicolons separate multiple queries:

```bash
confluence-mgmt q 'spaces(){minimal}; list(space=DEV){default}; get(12345){full}'
```

## Auth Notes

- **Cloud:** Uses Atlassian API token (same as Jira). Email + token pair.
- **Server/DC:** Uses Personal Access Token (product-specific, NOT the same as Jira PAT).
- Credentials stored in OS keychain under service `atlassian-mgmt` (shared with jira-mgmt).
- Auto-detection: `*.atlassian.net` = Cloud, else Server/DC.

## CQL Quick Reference

Common CQL patterns for agents:

```
type=page AND space="DEV"                          # all pages in space
type=page AND space="DEV" AND title="Architecture" # exact title
type=page AND text~"migration"                     # full-text search
type=page AND label="api-docs"                     # by label
type=page AND ancestor=12345                       # under parent
type=page AND creator=currentUser()                # my pages
type=page AND lastmodified >= now("-7d")            # recently modified
```

## References

- [CLI Commands](references/cli-commands.md)
- [DSL Examples](references/dsl-examples.md)
- [CQL Patterns](references/cql-patterns.md)
