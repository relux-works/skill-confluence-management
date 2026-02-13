# confluence-mgmt

CLI tool for interacting with Confluence REST API. Designed as an agent skill for Claude Code and Codex CLI.

Supports both **Confluence Cloud** (v2 API) and **Server/DC** (v1 API) with automatic routing.

## Features

- **Dual API**: v2 for Cloud CRUD, v1 for CQL search and Server/DC
- **DSL query layer**: Token-efficient reads with field presets (`minimal`, `default`, `overview`, `full`)
- **Shared keychain**: Uses `atlassian-mgmt` OS keychain — same credentials as `jira-mgmt`
- **Auto-detection**: Cloud vs Server/DC based on URL pattern
- **VPN hint**: Network errors suggest checking corporate VPN
- **Batch queries**: Multiple operations in one call via `;` separator

## Setup

```bash
./scripts/setup.sh
```

This builds the binary and symlinks to `~/.local/bin/confluence-mgmt`. Also creates skill symlinks for Claude Code and Codex CLI.

### Manual build

```bash
go build -o confluence-mgmt ./cmd/confluence-mgmt/
```

## Authentication

```bash
# Cloud (email + API token)
confluence-mgmt auth --instance https://company.atlassian.net/wiki --email user@co.com --token TOKEN

# Server/DC (Personal Access Token)
confluence-mgmt auth --instance https://confluence.company.com --token PAT

# Interactive
confluence-mgmt auth
```

Credentials are stored in the OS keychain under `atlassian-mgmt` service name. If you already authenticated via `jira-mgmt`, the same credentials work automatically.

## Commands

### DSL Query (primary for agents)

```bash
# Get page by ID
confluence-mgmt q 'get(12345){minimal}'

# List pages in space
confluence-mgmt q 'list(space=DEV){default}'

# CQL search
confluence-mgmt q 'search("type=page AND text~\"migration\""){default}'

# Children, ancestors, tree
confluence-mgmt q 'children(12345){minimal}'
confluence-mgmt q 'ancestors(12345){minimal}'
confluence-mgmt q 'tree(12345, depth=5){minimal}'

# Spaces
confluence-mgmt q 'spaces(){minimal}'

# Batch
confluence-mgmt q 'spaces(){minimal}; get(12345){overview}'
```

### Page operations

```bash
confluence-mgmt page get 12345 --body
confluence-mgmt page create --space DEV --title "Title" --body "<p>Content</p>" --parent 67890
confluence-mgmt page update 12345 --title "New Title" --body "<p>Updated</p>" --message "fix typo"
confluence-mgmt page delete 12345
```

### Labels

```bash
confluence-mgmt label add 12345 --labels "api-docs,v2"
confluence-mgmt label remove 12345 --labels "draft"
```

### Spaces

```bash
confluence-mgmt space list
```

### Config

```bash
confluence-mgmt config show
confluence-mgmt config set space DEV
```

## Field presets

| Preset | Fields |
|--------|--------|
| `minimal` | id, title, status |
| `default` | id, title, status, spaceKey, version, url |
| `overview` | + ancestors, labels |
| `full` | + body, created, updated, author |

## Project structure

```
cmd/confluence-mgmt/         CLI entry point (Cobra)
internal/config/             Auth (keychain) and config (YAML)
internal/confluence/         HTTP client, types, operations
internal/query/              DSL parser and executor
agents/skills/confluence-management/  Skill packaging (SKILL.md + references)
scripts/                     Build and setup scripts
.spec/                       Project specification
.research/                   API research documents
```

## Testing

```bash
go test ./...
```

## Tools

| Tool | Purpose | Location |
|------|---------|----------|
| `confluence-mgmt` | CLI binary | `~/.local/bin/confluence-mgmt` |
| `go test` | Unit tests | `internal/*/` |
| `scripts/setup.sh` | Build + install + symlink | project root |
| `task-board` | Project management | `.task-board/` |
