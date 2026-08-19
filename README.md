[![Build](https://github.com/mbevc1/tdns/actions/workflows/build.yaml/badge.svg)](https://github.com/mbevc1/tdns/actions/workflows/build.yaml)

# tdns CLI

A powerful, lightweight CLI to manage Technitium DNS server via HTTP API endpoint.

> [!NOTE]
`tnds` doesn't support full set of API calls yet. Contributions are welcome and feel free to check the upstream
guide or open an issue/PR!

> [!TIP]
Full list of API docs and spec is available [here](https://github.com/TechnitiumSoftware/DnsServer/blob/master/APIDOCS.md).

> [!IMPORTANT]
Because this is the initial (v0) iteration of the CLI, some features might
change and the code quality will improve over time (e.g. tests, re-use, ...).

## Installing

1. Download from the [releases](https://github.com/mbevc1/tdns/releases)
2. Run `tdns -v` to check if it's working correctly.
3. Enjoy!

## Usage and 🛠 Setup

Generate a config file with:

```bash
tdns init
```

and update the values for your endpoint!

Or manually create `config.json` or `~/.tdns/config.json`:

```json
{
  "token": "your-api-token",
  "host": "http://localhost:5380"
}
```

> [!TIP]
You can also use:
- `--token` (`-t`) and `--endpoint` (`-e`) flags
- Environment variable: `TDNS_API_TOKEN`

## 💡 Useful commands

### Zones

```bash
tdns list [--name 'example.*'] [--type Primary] [--page 1 --per-page 10] [--json]
tdns import <zone> --file zone.txt [--overwrite-zone] [--create] [--json]
tdns export <zone> [--output-dir dir] [--json]
tdns create <zone>... [--type Primary]
tdns delete <zone>...
```

#### Importing zones

`tdns import` posts an RFC 1035 (BIND style) zone file to an existing Primary or
Forwarder zone. Add `--create` to create the zone first if it isn't there yet
(`--type Forwarder` to create a Forwarder zone instead of a Primary one); it's a
no-op when the zone already exists, so it's safe to leave on in automation.

Import behaviour is controlled by three flags mapping to the API parameters:

| Flag | Default | Effect |
| --- | --- | --- |
| `--overwrite` | `true` | Overwrite existing record sets for the records being imported |
| `--overwrite-zone` | `false` | Delete **all** existing records in the zone first, so only the imported ones remain (needs Technitium v15.0+) |
| `--overwrite-soa-serial` | `true` | Use the SOA serial from the file instead of bumping the current one |

So a full replace from a generated zone file is:

```bash
tdns import example.com --file example.com.zone --create --overwrite-zone --yes
```

> [!WARNING]
`--overwrite-zone` also deletes the zone's apex `NS` records, so your zone file
must contain them or the zone will be left with none. The zone's `SOA` record is
kept (and is optional in the file), and DNSSEC records are always managed by the
server — it ignores `DNSKEY`/`RRSIG`/`NSEC`/`NSEC3`/`NSEC3PARAM` on import and
keeps the existing ones. `tdns export` output is safe to feed straight back in.
You'll be asked to confirm unless you pass `--yes` (`-y`).

> [!NOTE]
Because servers older than v15.0 silently ignore unknown parameters — and would
therefore import *without* clearing the zone — `--overwrite-zone` checks the
server version first and refuses to run against an older server. If the version
can't be read (for example when the API token has no permission to read
settings) you get a warning and the import proceeds.

> [!NOTE]
`--overwrite-soa-serial` defaults to `true` to match earlier `tdns` releases,
which differs from the API's own default of `false`. Importing a file whose
serial is *lower* than the zone's current serial will make secondary zones fail
to sync — pass `--overwrite-soa-serial=false` to let the server bump the serial
itself instead.

### Records

```bash
tdns records get <zone> [--filter A] [--json]
```

### Logs

```bash
tdns logs list
tdns logs download <filename> [--output log.txt]
tdns logs delete <filename>
tdns logs deleteAll
```

### Admin (Sessions)

```bash
tdns admin list-sessions
tdns admin delete-session --id <partialToken>
tdns admin create-token --user admin --token-name mytoken
tdns admin change-password -i [-c <current>] [-n <new>] [-o <totp>] [--iterations <n>]
tdns admin check-update [--json]
```

## Building and 🧪 Dev

If you want to build your own binarly locally, you can do that by running:

```shell
make build
```

Which should produce a locally executable binary.

> [!NOTE]
You'll need Golang 1.x compiler and Make.

To run tests there is a Makefile target for that as well:

```shell
make test
```

## Contributing

Report issues/questions/feature requests on in the [issues](https://github.com/mbevc1/tdns/issues/new) section.

Full contributing [guidelines are covered here](.github/CONTRIBUTING.md).

## Authors

* [Marko Bevc](https://github.com/mbevc1)
* Full [contributors list](https://github.com/mbevc1/tdns/graphs/contributors)

## License 🏷

MPL-2.0 Licensed. See [LICENSE](LICENSE) for full details.
<!-- https://choosealicense.com/licenses/ -->
