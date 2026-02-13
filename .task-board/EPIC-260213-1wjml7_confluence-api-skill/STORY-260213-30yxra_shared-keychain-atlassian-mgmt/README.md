# STORY-260213-30yxra: shared-keychain-atlassian-mgmt

## Description
Единый keychain service name 'atlassian-mgmt' для обоих CLI. В jira-mgmt: сменить serviceName на 'atlassian-mgmt'. При Load: 1) ищем в 'atlassian-mgmt', 2) если нет — ищем в старом 'jira-mgmt', перекладываем в 'atlassian-mgmt', удаляем из 'jira-mgmt' (one-time migration on the fly). В confluence-mgmt: сразу 'atlassian-mgmt', без fallback.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
