# go-import-server

HTTP server for canonical "go get" import path. It supports all versions of `go get` regardless of if it's Go Modules aware.

## Installation

The minimum requirement of Go is **1.13**.

```sh
$ go get unknwon.dev/go-import-server
```

### Configuration

Example for this tool itself (save as `app.toml`):

```toml
addr = "127.0.0.1:4333"

[[packages]]
import_path = "unknwon.dev/go-import-server"
subpath = "/go-import-server"
repo = "https://github.com/unknwon/go-import-server"
branch = "master"
```

Assuming `$GOPATH/bin` has been added to your `$PATH` environment variable.

```sh
$ go-import-server -config=./app.toml
YYYY/MM/DD 12:34:56 [ INFO] Listening on 127.0.0.1:4333...
```

## Reverse Proxy and HTTPS

I recommend use [Caddy](https://caddyserver.com) for automatic HTTPS in front of this tool:

```caddyfile
# Caddy 1
unknwon.dev {
    proxy / localhost:4333
}

# Caddy 2
unknwon.dev {
    reverse_proxy * localhost:4333
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

## License

This project is under MIT License. See the [LICENSE](LICENSE) file for the full license text.
