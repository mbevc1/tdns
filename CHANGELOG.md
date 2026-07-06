# Changelog

## [0.7.0](https://github.com/mbevc1/tdns/compare/v0.6.2...v0.7.0) (2026-07-06)


### Features

* **list:** support zones list filtering and pagination (API v15.3) ([168c5ea](https://github.com/mbevc1/tdns/commit/168c5ea78b30efd6270cb2a380de81dbc2a81788))
* **list:** support zones list filtering and pagination (API v15.3) ([0d00433](https://github.com/mbevc1/tdns/commit/0d00433010d18a1257dc874791ccbce1bb8029ec))


### Miscellaneous Chores

* release 0.7.0 ([477cf47](https://github.com/mbevc1/tdns/commit/477cf47c29013933f516865fb0680febb5a624ca))

## [0.6.2](https://github.com/mbevc1/tdns/compare/v0.6.1...v0.6.2) (2026-06-09)


### Miscellaneous Chores

* release 0.6.2 ([4e5cea8](https://github.com/mbevc1/tdns/commit/4e5cea8e118f4d64935f5265620cd9da538b3c3a))

## [0.6.1](https://github.com/mbevc1/tdns/compare/v0.6.0...v0.6.1) (2026-05-04)


### Features

* align change-password with upstream API ([b8480f0](https://github.com/mbevc1/tdns/commit/b8480f0432a61bd05d03bcc49444c3c431e420ae))


### Bug Fixes

* code formatting ([47b7b3b](https://github.com/mbevc1/tdns/commit/47b7b3b43ab8a5abf4cd1890b52e5f8f3676e406))
* remove unsupported Release Please parameter ([a51cf50](https://github.com/mbevc1/tdns/commit/a51cf5052e34da376e86c6fb38d3b650f79dfc9b))


### Miscellaneous Chores

* release 0.6.1 ([f2d4294](https://github.com/mbevc1/tdns/commit/f2d4294a25c1fe97d85602ce69aee3988aaff2c8))

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
- `admin change-password`: align with upstream Technitium API. Now requires
  current password (`-c`/`--current`) plus new password (`-n`/`--new`), with
  optional `-o`/`--totp` and `--iterations` flags. Replaces the previous
  single `-p`/`--pass` flag.

### Fixed
- Removed redundant fallback checks across command files
- Fully repopulated all command logic
