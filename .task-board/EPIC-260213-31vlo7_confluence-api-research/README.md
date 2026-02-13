# EPIC-260213-31vlo7: confluence-api-research

## Description
Предварительный ресёрч Confluence API: доступные API (REST v1, v2, GraphQL?), схемы, аутентификация (общий ключ с Jira или отдельный), ограничения, rate limits. Результат — задокументированное понимание API landscape перед началом разработки CLI.

## Scope
Research-only epic. No code. Deliverable: documented findings in .research/ covering Confluence API versions, auth mechanisms, key endpoints, schemas, and rate limits.

## Acceptance Criteria
1. Documented: какие API доступны (REST v1, v2, другие) и чем отличаются
2. Documented: схема аутентификации — работает ли Jira API token для Confluence или нужен отдельный
3. Documented: ключевые эндпоинты для read/write/search операций
4. Documented: rate limits и ограничения
5. Documented: разница Cloud vs Server/DC по API
6. Все findings в .research/ с линками из борда
