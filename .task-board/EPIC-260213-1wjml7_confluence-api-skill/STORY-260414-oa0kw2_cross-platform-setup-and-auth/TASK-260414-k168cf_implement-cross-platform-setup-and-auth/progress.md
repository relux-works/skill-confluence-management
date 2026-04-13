## Status
done

## Assigned To
codex

## Created
2026-04-13T22:57:06Z

## Last Update
2026-04-13T23:02:42Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Using the shared cross-platform setup and credentials pattern from skill-go-testing-tools as the target contract. Scope: root setup wrappers, scripts/setup.ps1, install metadata under os.UserConfigDir(), auto|keychain|env_or_file source policy, and auth commands set-access/whoami/resolve/clean/config-path.
Implemented cross-platform auth/config and setup alignment. Code changes: os.UserConfigDir() for config/auth/install-state, resolver with auto|keychain|env_or_file and desktop fallback, new auth set-access/whoami/resolve/clean/config-path flow, root setup.sh/setup.ps1 wrappers, scripts/setup.ps1, install metadata, and doc refresh. Verification: go test ./..., go run ./cmd/confluence-mgmt version, go run ./cmd/confluence-mgmt auth config-path, go run ./cmd/confluence-mgmt auth --help, go run ./cmd/confluence-mgmt auth whoami --help, ./setup.sh --help, ./setup.sh --install-only. Windows PowerShell script was checked statically only because pwsh/powershell is not installed on this macOS host.

## Precondition Resources
(none)

## Outcome Resources
(none)
