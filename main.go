package main

import (
	"flag"
	"net/http"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/go-macaron/auth"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/macaron.v1"
	log "unknwon.dev/clog/v2"
)

var Version = "dev"

func main() {
	configPath := flag.String("config", "./app.toml", "The config file path")
	flag.Parse()

	if err := log.NewConsole(); err != nil {
		panic("error init logger: " + err.Error())
	}
	log.Info("go-import-server: %v", Version)

	var config struct {
		Addr     string
		Packages []struct {
			ImportPath string `toml:"import_path"`
			Subpath    string
			Repo       string
			Branch     string
		}
		Prometheus struct {
			AuthUsername string `toml:"auth_username"`
			AuthPassword string `toml:"auth_password"`
		}
	}

	_, err := toml.DecodeFile(*configPath, &config)
	if err != nil {
		log.Fatal("Failed to decode config file: %v", err)
	}

	t, err := template.New("go-import").Parse(`<!DOCTYPE html>
<html>
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
	<meta name="go-import" content="{{.ImportPath}} git {{.Repo}}">
	<meta name="go-source" content="{{.ImportPath}} _ {{.Repo}}/tree/{{.Branch}}{/dir} {{.Repo}}/blob/{{.Branch}}{/dir}/{file}#L{line}">
	<style>
		pre {
			tab-size: 4;
		}
	</style>
</head>
<body>
	<p>Install command:</p>
	<pre>
	<code>go get {{.ImportPath}}</code></pre>

	<p>Import in source code:</p>
	<pre>
	<code>import "{{.ImportPath}}"</code></pre>

	<p>View <a href="{{.Repo}}">source code</a>.</p>
	<p>View <a href="https://godoc.org/{{.ImportPath}}">GoDoc</a>.</p>
</body>
</html>`)
	if err != nil {
		log.Fatal("Failed to parse template: %v", err)
	}

	m := macaron.New()
	for _, pkg := range config.Packages {
		m.Get(pkg.Subpath, func(w http.ResponseWriter, r *http.Request) {
			if err = t.Execute(w, pkg); err != nil {
				log.Error("Failed to execute template: %v", err)
			}
			log.Trace("Page served: %s", r.URL.Path)
		})
	}
	m.Get("/-/metrics",
		func(r *http.Request) {
			log.Trace("Metrics requested from %q", r.RemoteAddr)
		},
		auth.BasicFunc(func(username, password string) bool {
			// Not configured, skip.
			if config.Prometheus.AuthUsername == "" && config.Prometheus.AuthPassword == "" {
				return true
			}

			return auth.SecureCompare(username, config.Prometheus.AuthUsername) &&
				auth.SecureCompare(password, config.Prometheus.AuthPassword)
		}), promhttp.Handler())

	log.Info("Listening on http://%s...", config.Addr)
	if err := http.ListenAndServe(config.Addr, m); err != nil {
		log.Fatal("Failed to start server: %v", err)
	}
}
