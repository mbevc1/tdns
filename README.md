[![Build](https://github.com/mbevc1/tdns/actions/workflows/build.yaml/badge.svg)](https://github.com/mbevc1/tdns/actions/workflows/build.yaml)

# tdns CLI

A powerful, lightweight CLI to manage Technitium DNS server via an HTTP API.

## Installing

1. Download from the [releases](https://github.com/mbevc1/tdns/releases)
2. Run `tdns -v` to check if it's working correctly.
3. Enjoy!

## Usage and 🛠 Setup

Generate a config file with:

```bash
tdns init
```

Or manually create `config.json` or `~/.tdns/config.json`:

```json
{
  "token": "your-api-token",
  "host": "http://localhost:5380"
}
```

You can also use:
- `--token` (`-t`) and `--endpoint` (`-e`) flags
- Environment variable: `TDNS_API_TOKEN`

## 💡 Commands

### Zones

```bash
tdns list [--json]
tdns import <zone> --file zone.txt [--json]
tdns export <zone> [--output-dir dir] [--json]
tdns delete <zone>
```

### Records

```bash
tdns get-records <zone> [--filter A] [--json]
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
```

## Building and 🧪 Dev

If you want to build your own binarly locally, you can do that by running:

```shell
make build
```

Which should produce a locally binary to run. You'll need Golang compiler.

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
