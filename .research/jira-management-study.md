# jira-management Skill Study

Research findings for reusing jira-management patterns in the confluence-manager skill.

**Source:** `/Users/aagrigore1/src/skill-jira-management/`
**Skill deployed to:** `~/.claude/skills/jira-management/` (symlink)
**Date:** 2026-02-13

---

## 1. Overview

The jira-management skill is a **monorepo** containing:

1. A **Go CLI tool** (`jira-mgmt`) that wraps the Jira REST API
2. A **SKILL.md** file with agent-facing instructions (triggers, commands, usage patterns)
3. **Reference docs** for agents (CLI commands, DSL examples, JQL patterns, workflows, troubleshooting, dev notes)
4. **Scripts** for setup/teardown (build, symlink creation/removal)

The architecture follows a clean layered pattern:

```
Agent (Claude Code / Codex CLI)
  |
  v
SKILL.md (instructions + triggers)
  |
  v
CLI tool (jira-mgmt) — Cobra commands + DSL query parser
  |
  v
Go library (internal/) — HTTP client, types, field selection
  |
  v
Jira REST API (v2 for Server/DC, v3 for Cloud)
```

Key design principle: **DSL for reads (token-efficient), CLI commands for writes**. The DSL is a compact query language parsed by the tool, not raw JQL. This saves tokens in agent conversations.

---

## 2. Skill Structure (File Tree)

### Source repository

```
skill-jira-management/
|-- cmd/jira-mgmt/             # CLI entry point + Cobra commands
|   |-- main.go                # Root command, global flags, config loading
|   |-- helpers.go             # buildJiraClientFromConfig(), credential store setup
|   |-- locale.go              # Locale handling
|   |-- cmd_auth.go            # `auth` command
|   |-- cmd_config.go          # `config set/show` commands
|   |-- cmd_query.go           # `q` command (DSL query execution)
|   |-- cmd_grep.go            # `grep` command
|   |-- cmd_create.go          # `create` command
|   |-- cmd_update.go          # `update` command
|   |-- cmd_transition.go      # `transition` command
|   |-- cmd_comment.go         # `comment` command
|   |-- cmd_dod.go             # `dod` command (Definition of Done)
|   `-- main_test.go           # CLI integration tests
|-- internal/
|   |-- jira/                  # Jira REST API client library
|   |   |-- client.go          # HTTP client with retry, auth, API path selection
|   |   |-- types.go           # Domain types (Issue, Project, Sprint, Board, ADF, etc.)
|   |   |-- issues.go          # Issue CRUD + list
|   |   |-- search.go          # JQL search with pagination (Cloud cursor / Server offset)
|   |   |-- projects.go        # Project listing
|   |   |-- boards.go          # Board operations
|   |   |-- transitions.go     # Workflow transitions
|   |   |-- comments.go        # Comment operations
|   |   `-- client_test.go     # Client unit tests
|   |-- config/                # Config + credentials
|   |   |-- config.go          # YAML config manager (~/.config/jira-mgmt/config.yaml)
|   |   |-- auth.go            # Keychain-based credential store (go-keyring)
|   |   |-- config_test.go
|   |   `-- auth_test.go
|   |-- query/                 # DSL parser & executor
|   |   |-- parser.go          # Tokenizer + recursive descent parser + field presets
|   |   |-- ops.go             # Operation handlers (get, list, summary, search)
|   |   `-- parser_test.go
|   |-- fields/                # Field selection & projection
|   |   |-- selector.go        # Selector with presets, API field mapping, Apply/ApplyMany
|   |   `-- selector_test.go
|   `-- search/                # Scoped grep
|       |-- grep.go            # Regex search across issues/comments
|       `-- grep_test.go
|-- agents/skills/jira-management/   # Skill definition (source of truth)
|   |-- SKILL.md               # Main skill file with frontmatter + instructions
|   `-- references/
|       |-- cli-commands.md    # Complete CLI command reference
|       |-- dsl-examples.md    # DSL query patterns and examples
|       |-- jql-patterns.md    # JQL query patterns (comprehensive)
|       |-- workflows.md       # Multi-step workflow patterns
|       |-- troubleshooting.md # Common issues and solutions
|       `-- dev-notes.md       # Architecture notes for agents modifying CLI
|-- scripts/
|   |-- setup.sh               # Build binary + create symlinks
|   `-- deinit.sh              # Remove symlinks (+ optional config purge)
|-- go.mod
|-- go.sum
|-- README.md
|-- LICENSE
|-- .gitignore
|-- .spec/                     # Project specifications
|-- .task-board/               # Project management board
|-- .research/                 # Research artifacts
`-- .planning/                 # Planning artifacts
```

### Deployed skill (symlinked to `~/.claude/skills/jira-management/`)

```
jira-management/
|-- SKILL.md
`-- references/
    |-- cli-commands.md
    |-- dsl-examples.md
    |-- jql-patterns.md
    |-- workflows.md
    |-- troubleshooting.md
    `-- dev-notes.md
```

---

## 3. CLI Architecture

### Framework

- **Language:** Go 1.25.5
- **CLI framework:** [spf13/cobra](https://github.com/spf13/cobra)
- **Config format:** YAML (via `gopkg.in/yaml.v3`)
- **Credential storage:** OS keychain via [zalando/go-keyring](https://github.com/zalando/go-keyring)

### Go module path

```
github.com/ivalx1s/skill-jira-management
```

### Dependencies (minimal)

```
gopkg.in/yaml.v3         # Config serialization
github.com/spf13/cobra   # CLI framework
github.com/spf13/pflag   # CLI flag parsing (cobra dependency)
github.com/zalando/go-keyring  # OS keychain access
al.essio.dev/pkg/shellescape   # Shell escaping (keyring dep)
```

### Command structure

```
jira-mgmt
|-- auth                  # Authentication setup (interactive or flags)
|-- config
|   |-- set <key> <value> # Set config values (project, board, locale)
|   `-- show              # Display current config
|-- q '<dsl-query>'       # DSL query execution (read operations)
|-- grep <pattern>        # Scoped text search across issues/comments
|-- create                # Create new issue (--type, --summary, --project, etc.)
|-- update <key>          # Update issue fields (--summary, --description)
|-- transition <key>      # Move issue to different status (--to)
|-- comment <key>         # Add comment (--body)
|-- dod <key>             # Set Definition of Done (--set)
`-- version               # Print version
```

### Global flags (all commands)

| Flag | Type | Default | Purpose |
|------|------|---------|---------|
| `--project KEY` | string | from config | Override default project |
| `--board ID` | int | from config | Override default board |
| `--format json\|text` | string | `json` | Output format |

### Build-time variables

Set via `ldflags`:
- `version`
- `commit`
- `date`

### Binary deployment

- Built to `$PROJECT_ROOT/jira-mgmt` (gitignored)
- Symlinked to `~/.local/bin/jira-mgmt`
- Expects `~/.local/bin` in PATH

---

## 4. DSL / Query Layer

This is the core innovation. Instead of having agents construct raw JQL queries (verbose, error-prone, token-expensive), the CLI provides a compact DSL.

### Grammar

```
batch     = query (";" query)*
query     = operation "(" args ")" [ "{" fields "}" ]
args      = arg ("," arg)*  |  empty
arg       = ident "=" value  |  value
fields    = ident+
value     = ident | quoted_string
```

### Operations

| Operation | Purpose | Example |
|-----------|---------|---------|
| `get(KEY)` | Single issue lookup | `get(PROJ-123){full}` |
| `list(filters)` | Filtered multi-issue listing | `list(sprint=current,type=epic){overview}` |
| `summary()` | Board/project statistics | `summary()` |
| `search(jql="...")` | JQL passthrough | `search(jql="assignee=currentUser()"){default}` |

### Field presets (token efficiency)

| Preset | Fields |
|--------|--------|
| `minimal` | key, status |
| `default` | key, summary, status, assignee |
| `overview` | + type, priority, parent |
| `full` | all fields including subtasks |

### Batch support

Multiple queries separated by `;`:
```bash
jira-mgmt q 'summary(); list(sprint=current){default}'
```

### Implementation details

1. **Tokenizer** (`parser.go`): Hand-written lexer that handles idents, strings, parens, braces, equals, commas, semicolons
2. **Parser** (`parser.go`): Recursive descent parser producing an AST (`Query` -> `Statement[]` -> `Operation`, `Args[]`, `Fields[]`)
3. **Executor** (`ops.go`): Takes parsed AST, executes against Jira client. Each operation maps to specific API calls
4. **Selector** (`selector.go`): Handles field projection. Maps DSL field names to Jira API field names. Apply() builds response maps with only requested fields

### IMPORTANT: Dual field definitions

Field names and presets are defined in **two places** that must stay in sync:
- `internal/query/parser.go`: `ValidFields`, `FieldPresets` (parser validation)
- `internal/fields/selector.go`: `ValidFields`, `Presets`, `JiraAPIFields()` (projection + API mapping)

---

## 5. Auth / Config Pattern

### Two-layer storage

| What | Where | Format |
|------|-------|--------|
| Configuration | `~/.config/jira-mgmt/config.yaml` | YAML |
| Credentials | OS keychain | JSON serialized to keychain value |

### Config file structure

```yaml
active_project: SEARCH
active_board: 0
locale: en
instance_url: https://jira.mts.ru
instance_type: server    # "cloud" or "server"
auth_type: bearer        # "basic" or "bearer"
```

### Auth flow

1. User provides instance URL + token (+ optional email)
2. CLI auto-detects auth type:
   - Email provided -> Basic auth (Cloud): `base64(email:token)`
   - No email -> Bearer auth (Server/DC PAT): `Bearer <token>`
3. CLI probes `/rest/api/2/serverInfo` to detect instance type (Cloud vs Server/DC)
4. Credentials serialized to JSON, stored in OS keychain under service name `jira-mgmt`
5. Instance URL, type, auth type saved to config YAML
6. On subsequent runs: read config for instance URL, load credentials from keychain, construct client

### Auth type detection

```
Email provided?
  Yes -> Basic auth (Cloud: email + API token)
  No  -> Bearer auth (Server/DC: Personal Access Token)
```

### Instance type detection

Probes `/rest/api/2/serverInfo`:
- `deploymentType: "Cloud"` -> Cloud (uses API v3, cursor pagination)
- Other -> Server/DC (uses API v2, offset pagination)

### Credential store interface

```go
type CredentialStore interface {
    Save(creds Credentials) error
    Load(instanceURL string) (Credentials, error)
    Delete(instanceURL string) error
}
```

Implemented by `KeychainStore` with function pointers for keyring operations (allows test mocking).

---

## 6. Output Formats

### Global `--format` flag

| Format | Default? | Use case |
|--------|----------|----------|
| `json` | Yes | Machine-readable, agent parsing |
| `text` | No | Human-readable display |

### DSL query output

- Single result: pretty-printed JSON
- Multiple results (batch): JSON array
- Uses `json.Encoder` with `SetIndent("", "  ")`

### Grep output

- JSON mode: array of Match objects `{issue_key, field, content, line}`
- Text mode: grep-style `issue_key:field:line:content`

### Selector-based projection

All read operations go through `fields.Selector.Apply()` which builds a `map[string]interface{}` with only the requested fields. This is the primary mechanism for token efficiency.

---

## 7. SKILL.md Pattern

### Frontmatter

```yaml
---
name: jira-management
description: Drive jira-mgmt CLI for Jira operations (Cloud & Server/DC, auto-detected). Translates natural language intent to CLI commands. Uses DSL for reads (token-efficient), CLI for writes. Handles multi-step workflows (create epic with stories, bulk transitions, sprint reviews).
triggers:
  - jira
  - ticket
  - issue
  - epic
  - story
  - board
  - sprint
  - create issue
  - search
  - jql
  - ... (56 triggers total, bilingual EN/RU)
---
```

### Structure

1. **Purpose statement** (1 line: what + how + tool name)
2. **Quick Start** (auth, config, basic ops)
3. **Commands Overview** (auth, queries, search, create, update, global flags)
4. **Agent Usage Patterns** (natural language -> CLI mapping examples)
5. **Read/Write separation** (DSL for reads, CLI for writes, when to use JQL)
6. **Cloud vs Server/DC differences table**
7. **References section** (links to reference docs in `references/`)
8. **Version footer**

### Trigger design

- Bilingual: English + Russian
- Short phrases: "jira", "ticket", "issue"
- Action phrases: "create issue", "move issue", "show board"
- Domain terms: "sprint", "epic", "story", "jql", "dod"
- Total: 56 triggers covering the full vocabulary

### Reference doc pattern

Each reference doc is self-contained:
- `cli-commands.md`: Every command with all flags, examples, notes
- `dsl-examples.md`: Every query pattern with output examples
- `jql-patterns.md`: Comprehensive JQL reference (1119 lines)
- `workflows.md`: Multi-step workflow scripts (epic creation, bulk transitions, sprint review, etc.)
- `troubleshooting.md`: Error -> cause -> solution
- `dev-notes.md`: Architecture notes for modifying the CLI codebase

---

## 8. Reusable Patterns (directly applicable to confluence-manager)

### 8.1 Monorepo structure

The same layout works perfectly:
```
skill-confluence-manager/
|-- cmd/confluence-mgmt/      # CLI entry point
|-- internal/
|   |-- confluence/           # Confluence REST API client
|   |-- config/               # Auth + config (reuse pattern verbatim)
|   |-- query/                # DSL parser (adapt operations)
|   |-- fields/               # Field selection (adapt fields)
|   `-- search/               # Content search (CQL instead of JQL)
|-- agents/skills/confluence-management/
|   |-- SKILL.md
|   `-- references/
|-- scripts/
|   |-- setup.sh
|   `-- deinit.sh
```

### 8.2 Auth/config pattern (reuse ~90%)

Nearly identical:
- Config YAML at `~/.config/confluence-mgmt/config.yaml`
- Credentials in OS keychain under service name `confluence-mgmt`
- Same auth type detection: email -> Basic (Cloud), no email -> Bearer (Server/DC PAT)
- Same instance type detection via serverInfo endpoint
- Same Atlassian API versioning (Cloud v2 vs Server/DC v1 for Confluence)

Config fields would be:
```yaml
active_space: SPACEKEY
locale: en
instance_url: https://company.atlassian.net/wiki
instance_type: cloud
auth_type: basic
```

### 8.3 DSL query layer (reuse architecture, adapt grammar)

The tokenizer and parser are domain-agnostic. Reuse the tokenizer verbatim, adapt:
- `ValidOperations`: change from Jira ops to Confluence ops
- `ValidFields`: change from issue fields to page/space fields
- `FieldPresets`: define Confluence-appropriate presets
- `Executor`: new operation handlers for Confluence API calls

### 8.4 Field selection pattern (reuse architecture)

`Selector` pattern works perfectly:
- Define Confluence field mappings (page title, body, space, labels, version, etc.)
- Same Apply/ApplyMany projection
- Same JiraAPIFields() -> ConfluenceAPIFields() concept

### 8.5 CLI framework (reuse verbatim)

- Cobra command structure
- Global flags pattern
- `buildClientFromConfig()` helper
- `persistentPreRun` for auth checks
- Build-time ldflags

### 8.6 Scripts (reuse ~95%)

Setup/deinit scripts just change the tool name.

### 8.7 SKILL.md structure (reuse template)

Same frontmatter format, same section organization, same reference doc pattern.

### 8.8 Grep pattern (reuse with adaptation)

Search across page titles, bodies, comments. Same Match struct, same regex engine.

### 8.9 HTTP client (reuse ~80%)

Same retry logic, auth header construction, API path selection. Different base paths.

---

## 9. Adaptations Needed

### 9.1 Domain model (completely new)

Jira domain: Issue, Sprint, Board, Transition, IssueType, Priority
Confluence domain: Page, Space, Attachment, Label, Version, Ancestor, Comment, Template

Key types to define:
```go
type Page struct {
    ID       string
    Title    string
    SpaceKey string
    Body     PageBody    // storage format or view format
    Version  Version
    Ancestors []PageRef
    Labels   []Label
    Status   string      // current, draft, archived
    // ...
}

type Space struct {
    ID   int
    Key  string
    Name string
    Type string  // global, personal
    // ...
}
```

### 9.2 DSL operations (new set)

Replace Jira operations with Confluence operations:

| Jira op | Confluence equivalent |
|---------|----------------------|
| `get(KEY)` | `get(PAGE-ID)` or `get(space=X,title="Page Title")` |
| `list(filters)` | `list(space=X,type=page,status=current)` |
| `summary()` | `summary(space=X)` - space statistics |
| `search(jql="...")` | `search(cql="...")` - CQL search |

New operations specific to Confluence:
- `tree(PAGE-ID)` - get page hierarchy/tree
- `children(PAGE-ID)` - list child pages
- `history(PAGE-ID)` - version history

### 9.3 Query language: CQL instead of JQL

Confluence uses CQL (Confluence Query Language), not JQL:
```
type = "page" AND space = "DEV" AND title ~ "architecture"
```

Reference doc should be `cql-patterns.md` instead of `jql-patterns.md`.

### 9.4 Content format: Confluence Storage Format

Jira uses ADF (Atlassian Document Format) for Cloud and wiki markup for older Server.
Confluence uses:
- **Storage format** (XHTML-based) for Cloud and newer Server
- **View format** (rendered HTML)
- **Atlas Doc Format** (newer Confluence Cloud)

Need `StorageFormatText()` equivalent of `DescriptionText()`.

### 9.5 Write operations (different verbs)

| Jira writes | Confluence writes |
|-------------|-------------------|
| `create --type story` | `create --space X --title "..." --parent PAGE-ID` |
| `transition PROJ-123 --to "Done"` | `publish PAGE-ID` / `archive PAGE-ID` |
| `comment PROJ-123 --body "..."` | `comment PAGE-ID --body "..."` |
| `update PROJ-123 --summary "..."` | `update PAGE-ID --title "..." --body "..."` |
| `dod PROJ-123 --set "..."` | N/A (no DoD concept) |

New write operations:
- `move PAGE-ID --to-space X --under PARENT-ID`
- `label PAGE-ID --add "label1,label2"` / `--remove "label"`
- `attach PAGE-ID --file path/to/file`

### 9.6 Field presets (new set)

```go
var FieldPresets = map[string][]string{
    "minimal":  {"id", "title", "status"},
    "default":  {"id", "title", "status", "space", "version"},
    "overview": {"id", "title", "status", "space", "version", "ancestors", "labels"},
    "full":     {"id", "title", "status", "space", "version", "ancestors", "labels", "body", "created", "updated", "creator", "lastModifier"},
}
```

### 9.7 API paths

| Jira | Confluence |
|------|------------|
| `/rest/api/3/` (Cloud) | `/wiki/api/v2/` (Cloud) or `/rest/api/content` (Cloud legacy / Server) |
| `/rest/api/2/` (Server) | `/rest/api/content` (Server) |
| `/rest/agile/1.0/` | N/A |

Confluence Cloud has two API versions:
- **v1** (legacy): `/wiki/rest/api/content/...`
- **v2** (new): `/wiki/api/v2/pages/...`, `/wiki/api/v2/spaces/...`

### 9.8 Pagination

Jira: cursor-based (Cloud) or offset-based (Server)
Confluence: cursor-based (Cloud v2) or offset-based (v1 / Server). Same pattern, different field names.

### 9.9 Reference docs adaptation

| Jira ref | Confluence equivalent |
|----------|----------------------|
| `cli-commands.md` | `cli-commands.md` (new commands) |
| `dsl-examples.md` | `dsl-examples.md` (new operations) |
| `jql-patterns.md` | `cql-patterns.md` (CQL reference) |
| `workflows.md` | `workflows.md` (page creation, space management, doc migration) |
| `troubleshooting.md` | `troubleshooting.md` (Confluence-specific issues) |
| `dev-notes.md` | `dev-notes.md` (same pattern) |

### 9.10 Triggers (new vocabulary)

Confluence-specific triggers:
```yaml
triggers:
  - confluence
  - конфлюенс
  - wiki
  - вики
  - page
  - страница
  - space
  - спейс
  - пространство
  - create page
  - создай страницу
  - find page
  - найди страницу
  - update page
  - обнови страницу
  - search docs
  - поиск документов
  - cql
  - template
  - шаблон
  - label
  - метка
  - attachment
  - вложение
```

---

## 10. Implementation Priority

Recommended build order (based on jira-management's architecture):

1. **Scaffold project** (go mod, directory structure, scripts)
2. **Config + auth** (copy from jira-management, change service name and config keys)
3. **HTTP client** (copy from jira-management, change API paths for Confluence)
4. **Domain types** (Page, Space, Label, Version, etc.)
5. **Basic API operations** (get page, list pages, search CQL)
6. **DSL parser** (copy tokenizer, define Confluence operations and field presets)
7. **Field selector** (define Confluence field mappings)
8. **CLI commands** (auth, config, q, grep, create, update)
9. **Skill files** (SKILL.md + references/)
10. **Workflow patterns** (multi-step operations)

---

## 11. Technical Decisions to Carry Forward

1. **JSON as default output format** - agents parse JSON better than text
2. **DSL over raw CQL for reads** - token efficiency is critical
3. **Presets for field selection** - reduces token count dramatically
4. **OS keychain for credentials** - never store tokens in plain files
5. **Auto-detection of instance type** - one tool works for both Cloud and Server
6. **Batch queries via semicolons** - reduces round-trips
7. **Selector.Apply() projection** - only return what was requested
8. **Separate read/write paths** - DSL for reads, explicit CLI for writes
9. **Comprehensive reference docs** - agents need examples, not just flag descriptions
10. **Bilingual triggers** - support both EN and RU

---

## 12. Key Source Files Reference

| File | Lines | Purpose | Reuse level |
|------|-------|---------|-------------|
| `internal/jira/client.go` | 278 | HTTP client, retry, auth, path selection | ~80% (change paths) |
| `internal/jira/types.go` | 410 | All domain types | ~10% (new domain) |
| `internal/config/config.go` | 184 | Config manager (YAML) | ~90% (change keys) |
| `internal/config/auth.go` | 120 | Keychain credential store | ~95% (change service name) |
| `internal/query/parser.go` | 365 | DSL tokenizer + parser + field defs | ~70% (new ops/fields) |
| `internal/query/ops.go` | 232 | Operation handlers | ~20% (completely new ops) |
| `internal/fields/selector.go` | 213 | Field selection + projection | ~50% (new field mappings) |
| `internal/search/grep.go` | 174 | Regex search | ~60% (adapt for pages) |
| `cmd/jira-mgmt/main.go` | 107 | Root command, global flags | ~90% (change name) |
| `cmd/jira-mgmt/helpers.go` | 50 | Client builder | ~90% |
| `cmd/jira-mgmt/cmd_query.go` | 78 | Query command | ~95% |
| `scripts/setup.sh` | 51 | Build + symlink | ~95% (change names) |
| `scripts/deinit.sh` | 64 | Cleanup | ~95% (change names) |

---

**Document Version:** 1.0
**Last Updated:** 2026-02-13
