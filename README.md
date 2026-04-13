# confluence-mgmt

CLI tool for interacting with Confluence REST API. Designed as an agent skill for Claude Code and Codex CLI.

Supports both **Confluence Cloud** (v2 API) and **Server/DC** (v1 API) with automatic routing.

## Features

- **Dual API**: v2 for Cloud CRUD, v1 for CQL search and Server/DC
- **DSL query layer**: Token-efficient reads with field presets (`minimal`, `default`, `overview`, `full`)
- **Cross-platform auth**: `auto | keychain | env_or_file` with desktop-first defaults
- **Auto-detection**: Cloud vs Server/DC based on URL pattern
- **VPN hint**: Network errors suggest checking corporate VPN
- **Batch queries**: Multiple operations in one call via `;` separator

## Setup

```bash
# macOS / Linux shells
./setup.sh

# Windows PowerShell
.\setup.ps1

# Verify installed binary
confluence-mgmt version
```

This builds the binary, installs it to the user-local bin dir, refreshes the installed skill artifact, writes install metadata, and refreshes Claude/Codex skill links.

### Manual build

```bash
go build -o confluence-mgmt ./cmd/confluence-mgmt/
```

## Authentication

```bash
# Cloud (email + API token)
confluence-mgmt auth set-access --instance https://company.atlassian.net/wiki --email user@co.com --token TOKEN

# Server/DC (Personal Access Token)
confluence-mgmt auth set-access --instance https://confluence.company.com --token PAT

# Canonical live auth probe
confluence-mgmt auth whoami
```

Credential source names are stable across platforms: `auto`, `keychain`, `env_or_file`.
`auto` prefers system secret storage on macOS and Windows, with `env_or_file` as explicit fallback.

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
setup.sh / setup.ps1         Root setup wrappers
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
| `setup.sh` / `setup.ps1` | Build + install + verify | project root |
| `task-board` | Project management | `.task-board/` |

## Runtime Paths

- Config: `os.UserConfigDir()/confluence-mgmt/config.yaml`
- Auth fallback file: `os.UserConfigDir()/confluence-mgmt/auth.json`
- Install state: `os.UserConfigDir()/confluence-mgmt/install.json`
- Installed binary: `~/.local/bin/confluence-mgmt`
- Installed skill artifact: `~/.agents/skills/confluence-management`
