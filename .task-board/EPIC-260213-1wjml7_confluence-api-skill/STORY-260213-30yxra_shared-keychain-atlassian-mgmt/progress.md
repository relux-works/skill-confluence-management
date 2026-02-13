## Status
done

## Assigned To
claude

## Created
2026-02-13T13:11:25Z

## Last Update
2026-02-13T13:15:51Z

## Blocked By
- (none)

## Blocks
- STORY-260213-24x0jt

## Checklist
(empty)

## Notes
jira-mgmt auth.go: serviceName changed to 'atlassian-mgmt', legacyServiceName='jira-mgmt'. Load() does fallback + one-time migration. 2 new tests (LegacyMigration, LegacyNotUsedWhenNewExists). All config tests PASS. Pre-existing failure in query/parser_test.go (unrelated — subtasks field count mismatch).
Verified against real instance (jira.mts.ru, Server/DC). Migration worked transparently — credentials loaded from legacy 'jira-mgmt', migrated to 'atlassian-mgmt', query succeeded.
