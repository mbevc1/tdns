# Changelog

## [0.8.1](https://github.com/mbevc1/tdns/compare/v0.8.0...v0.8.1) (2026-08-20)


### Bug Fixes

* **list:** warn when filtering paginated results needs a newer server ([03c6485](https://github.com/mbevc1/tdns/commit/03c648559c8a4c22e40cdeb2c53c9fba280aec0d))
* **list:** warn when filtering paginated results needs a newer server ([18cf668](https://github.com/mbevc1/tdns/commit/18cf66800ab0f45691b20cd89e91c1a8a17344ac))

## [0.8.0](https://github.com/mbevc1/tdns/compare/v0.7.1...v0.8.0) (2026-08-19)


### Features

* **import:** add overwrite-zone, overwrite and create flags ([1386b9c](https://github.com/mbevc1/tdns/commit/1386b9ce9188184e5ec42031aa05b4427664ff06))
* **import:** add overwrite-zone, overwrite and create flags ([8ab2e08](https://github.com/mbevc1/tdns/commit/8ab2e081b898c369aa6c29f8a143445b1d68208d)), closes [#72](https://github.com/mbevc1/tdns/issues/72)
* **import:** require --file and accept the zone file on stdin ([74ab3c4](https://github.com/mbevc1/tdns/commit/74ab3c4f409f7266f37783835f93e51189b5c4b1))

## [0.7.1](https://github.com/mbevc1/tdns/compare/v0.7.0...v0.7.1) (2026-07-14)


### Miscellaneous Chores

* release 0.7.1 ([afa0aed](https://github.com/mbevc1/tdns/commit/afa0aed6590eec1ce9c1703e8463e33d8635c6fd))

## [0.7.0](https://github.com/mbevc1/tdns/compare/v0.6.2...v0.7.0) (2026-07-06)


### Features

* **list:** support zones list filtering and pagination (API v15.3) ([168c5ea](https://github.com/mbevc1/tdns/commit/168c5ea78b30efd6270cb2a380de81dbc2a81788))


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
