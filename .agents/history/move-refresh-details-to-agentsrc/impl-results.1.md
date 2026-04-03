## Result

- Added a `refresh` object to `.agentsrc.json` via `internal/config/agentsrc.go`, including `version`, `commit`, `describe`, and `refreshedAt`.
- Updated Go `refresh` and `install` flows to persist refresh metadata into `.agentsrc.json`, generate a minimal manifest when missing during refresh, and remove any legacy `.agents-refresh` file after writing.
- Updated `status` to read refresh data from the manifest first and fall back to legacy `.agents-refresh` markers for older projects.
- Removed the old `add` behavior that inserted `.agents-refresh` into project `.gitignore`.
- Brought the shell `refresh.sh` and `install.sh` paths into parity by updating `.agentsrc.json` directly.

## Verification

- `go test ./internal/config`
- `go test ./commands -run 'Test(WriteRefreshMetadataStoresDetailsInAgentsRC|ReadRefreshTimestampPrefersAgentsRCMetadata|ReadRefreshTimestampFallsBackToLegacyMarker)$'`
