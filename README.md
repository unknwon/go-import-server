# go-import-server

HTTP server for canonical "go get" import path. It supports all versions of `go get` regardless of if it's Go Modules aware.

## Installation

Install from source or download binaries on [GitHub Releases](https://github.com/unknwon/go-import-server/releases).

The minimum requirement of Go is **1.16**, and 64-bit system is required because of [a bug in BadgerDB](https://github.com/dgraph-io/badger/issues/953).

```sh
$ go get unknwon.dev/go-import-server
```

### Configuration

Example for this tool itself (save as `app.toml`):

```toml
addr = "127.0.0.1:4333"
db_path = "app.db"

[[packages]]
import_path = "unknwon.dev/go-import-server"
subpath = "/go-import-server"
repo = "https://github.com/unknwon/go-import-server"
branch = "main"
```

Assuming `$GOPATH/bin` has been added to your `$PATH` environment variable.

```sh
$ go-import-server -config=./app.toml
YYYY/MM/DD 12:34:56 [ INFO] Listening on http://127.0.0.1:4333...
```

## Reverse proxy and HTTPS

I recommend use [Caddy](https://caddyserver.com) for automatic HTTPS in front of this tool:

```caddyfile
# Caddy 1
unknwon.dev {
    proxy / localhost:4333 {
        transparent
    }
}

# Caddy 2
unknwon.dev {
    reverse_proxy * localhost:4333 {
        header_up Host {http.request.host}
        header_up X-Real-IP {http.request.remote}
        header_up X-Forwarded-For {http.request.remote}
        header_up X-Forwarded-Port {http.request.port}
        header_up X-Forwarded-Proto {http.request.scheme}
    }
}
```

## Metrics

This tool exposes [Prometheus](https://prometheus.io/) metrics via endpoint `/-/metrics`.

You can set HTTP Basic Authentication for this endpoint via your `app.toml`:

```toml
[prometheus]
auth_username = "superuser"
auth_password = "supersecure"
```

The [BadgerDB](https://github.com/dgraph-io/badger) is used to store total page views and number of `go get`s.

Here is an example dump:

```
go_import_server_stats_view_total 20
go_import_server_stats_view_unknwon_dev_go_import_server 20
go_import_server_stats_get_total 16
go_import_server_stats_get_unknwon_dev_go_import_server 16
```

## Development

This project uses the [Task](https://taskfile.dev/) as the build tool, it is not required as all the commands are listed in the `Taskfile.yml` in plaintext.

The source files of templates are located in the `templates` directory but uses [Go embed](https://blog.jetbrains.com/go/2021/06/09/how-to-use-go-embed-in-go-1-16/) to embed into the binary. Due to the nature limitation of the Go embed, templates cannot be hot-reloaded after modifications, so the following command needs to be used for re-packing templates and re-compiling the bianry:

```sh
$ task web --force
```

## Open-source, not open-contribution

_Quote from [benbjohnson/litestream](https://github.com/benbjohnson/litestream#open-source-not-open-contribution):_

> I am grateful for community involvement, bug reports, & feature requests. I do not wish to come off as anything but welcoming, however, I've made the decision to keep this project closed to contributions for my own mental health and long term viability of the project.

## License

This project is under MIT License. See the [LICENSE](LICENSE) file for the full license text.
