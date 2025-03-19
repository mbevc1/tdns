# Changelog

## [dev] - In Development

### Added
- Zone commands: `list`, `import`, `export`, `delete`
- Record listing: `get-records` with filtering
- Logs: `logs list`, `download`, `delete`, `deleteAll`
- Admin: `list-sessions`, `delete-session`, `create-token`
- Support for config file, env vars, CLI flags
- Optional JSON output for `list`, `import`, `export`, `get-records`
- Colorized console output using `colorama` or `fatih/color`
- Config fallback & default values
- Makefile with helpful targets

### Changed
- Renamed `--type` to `--filter` for consistency
- Added confirmation for destructive actions

### Fixed
- Removed redundant fallback checks across command files
- Fully repopulated all command logic
