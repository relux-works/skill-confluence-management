## Status
done

## Assigned To
(none)

## Created
2026-02-13T12:46:59Z

## Last Update
2026-02-13T12:55:39Z

## Blocked By
- (none)

## Blocks
- STORY-260213-1rdtug

## Checklist
(empty)

## Notes
IMPORTANT: изучить auto-detection паттерн (Cloud vs Server/DC) в jira-management CLI — нужно слизать для confluence-manager
Results: .research/jira-management-study.md. Key: layered arch (SKILL.md → CLI → Go lib → API). DSL for reads (tokenizer+parser, 365 lines), CLI for writes. Auth: YAML config + OS keychain, auto-detect Cloud vs Server via /rest/api/2/serverInfo. Reusable: project structure (95%), auth/config (90%), HTTP client (80%), DSL parser (70%), SKILL.md template (90%). New: domain types, CQL ops, Confluence storage format, write commands.
