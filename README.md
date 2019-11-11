# go-import-server

HTTP server for canonical "go get" import path. It supports all versions of `go get` regardless of if it's Go Modules aware.

## Installation

Install from source or download binaries on [GitHub Releases](https://github.com/unknwon/go-import-server/releases).

The minimum requirement of Go is **1.13**, and 64-bit system is required because of [a bug in BadgerDB](https://github.com/dgraph-io/badger/issues/953).

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
branch = "master"
```

Assuming `$GOPATH/bin` has been added to your `$PATH` environment variable.

```sh
$ go-import-server -config=./app.toml
YYYY/MM/DD 12:34:56 [ INFO] Listening on http://127.0.0.1:4333...
```

## Reverse Proxy and HTTPS

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

## License

This project is under MIT License. See the [LICENSE](LICENSE) file for the full license text.
