# Confluence API Access Skill

## Overview

Skill для агентов (Claude Code / Codex CLI), дающий структурированный доступ к Confluence через API. Агент получает возможность читать, искать, создавать и обновлять страницы в Confluence без ручного копирования контента.

## Problem

- Агенты не имеют прямого доступа к Confluence — приходится копипастить контент руками.
- Нет стандартизированного способа для агента запросить данные из Confluence (поиск, чтение страницы, дерево пространства).
- Нет инструмента для агента, чтобы публиковать результаты работы обратно в Confluence.

## Goals

1. **CLI-инструмент** — обёртка над Confluence REST API, оптимизированная для агентов (компактный вывод, token-efficient форматы).
2. **Skill** — SKILL.md с инструкциями для агента: когда и как использовать CLI, паттерны работы.
3. **Agent-facing query layer** — DSL или structured commands для минимизации токенов при чтении.

## Capabilities (Target)

### Read
- Получить содержимое страницы по ID или title
- Поиск страниц (CQL)
- Дерево дочерних страниц
- Получить метаданные (labels, space, version, last author)

### Write
- Создать страницу (title, body, parent, space)
- Обновить существующую страницу (append / replace)
- Добавить/удалить labels

### Navigation
- Список пространств
- Дерево пространства
- Breadcrumb (ancestors) страницы

## Technical Approach

- **Language:** Go (единый стек с task-board CLI)
- **API:** Confluence REST API v2 (Cloud) + fallback на v1 для Server/DC
- **Auto-detection:** CLI автоматически определяет тип инстанса (Cloud vs Server/DC) и выбирает нужный API/auth flow. Паттерн слизать из jira-management skill.
- **Auth:** API token (Cloud) / PAT (Server/DC). Shared keychain service `atlassian-mgmt` (общий с jira-mgmt). В jira-mgmt добавить fallback на старый `jira-mgmt` service для бесшовной миграции.
- **Output:** `--format compact` для агентов, human-readable по умолчанию
- **Skill structure:** стандартная (SKILL.md + scripts/ + references/)

## Out of Scope (v1)

- Attachments (upload/download)
- Comments (read/write)
- Page permissions management
- Real-time collaboration / watching
- Confluence macros rendering

## Success Criteria

- Агент может найти и прочитать страницу из Confluence одной командой
- Агент может создать/обновить страницу из CLI
- Вывод оптимизирован для LLM (минимум мусора, максимум контента)
- Работает и с Cloud, и с Server/DC
