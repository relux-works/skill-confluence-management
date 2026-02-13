# EPIC-260213-1wjml7: confluence-api-skill

## Description
Skill для агентов: CLI-обёртка над Confluence REST API + SKILL.md. Чтение, поиск, создание и обновление страниц. Go, token-efficient output, Cloud + Server/DC support.

## Scope
CLI tool (Go) wrapping Confluence REST API. SKILL.md for agent integration. Query DSL for token-efficient reads. Support Cloud (v2 API + token) and Server/DC (v1 API + PAT). Read: page content, search (CQL), page tree, metadata. Write: create page, update page, manage labels. Navigation: spaces list, space tree, breadcrumbs.

## Acceptance Criteria
1. Agent can search and read a Confluence page with a single CLI command
2. Agent can create and update pages from CLI
3. Output has --format compact mode optimized for LLM consumption
4. Works with both Confluence Cloud and Server/DC
5. Auth configured via env vars or config file
6. SKILL.md documents all commands, triggers, and usage patterns
7. Skill follows standard structure (SKILL.md + scripts/ + references/)
