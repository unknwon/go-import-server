package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/dgraph-io/badger/v2"
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
	defer log.Stop()

	log.Info("go-import-server: %v", Version)

	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatal("Failed to load config: %v", err)
	}

	db, stats, err := getDBWithStats(config.DBPath)
	if err != nil {
		log.Fatal("Failed to get database with stats: %v", err)
	}
	defer db.Close()

	t, err := getTemplate()
	if err != nil {
		log.Fatal("Failed to get template: %v", err)
	}

	m := macaron.New()
	for i := range config.Packages {
		pkg := config.Packages[i]

		if stats.pkgsView[pkg.ImportPath] == nil {
			var pkgView int64
			stats.pkgsView[pkg.ImportPath] = &pkgView
		}
		if stats.pkgsGet[pkg.ImportPath] == nil {
			var pkgGet int64
			stats.pkgsGet[pkg.ImportPath] = &pkgGet
		}

		m.Get(pkg.Subpath, func(w http.ResponseWriter, r *http.Request) {
			if err = t.Execute(w, pkg); err != nil {
				log.Error("Failed to execute template: %v", err)
				return
			}
			log.Trace("Page served: %s", r.URL.Path)

			if r.URL.Query().Get("go-get") == "1" {
				stats.PkgGetIncr(pkg.ImportPath, 1)
			} else {
				stats.PkgViewIncr(pkg.ImportPath, 1)
			}
		})
	}
	m.Get("/-/metrics",
		func(c *macaron.Context) {
			log.Trace("Metrics requested from %q", c.RemoteAddr())
		},
		auth.BasicFunc(func(username, password string) bool {
			// Not configured, skip.
			if config.Prometheus.AuthUsername == "" && config.Prometheus.AuthPassword == "" {
				return true
			}

			return auth.SecureCompare(username, config.Prometheus.AuthUsername) &&
				auth.SecureCompare(password, config.Prometheus.AuthPassword)
		}),
		promhttp.Handler(),
	)
	setupPrometheusMetrics(stats)

	done := make(chan struct{})
	go stats.start(db, done)

	s := newServer(config.Addr, m)
	log.Info("Listening on http://%s...", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Info("Server closed gracefully")
			done <- struct{}{}
		} else {
			log.Fatal("Failed to start server: %v", err)
		}
	}

	<-done
}

type config struct {
	Addr     string
	DBPath   string `toml:"db_path"`
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

func loadConfig(path string) (*config, error) {
	var c config
	_, err := toml.DecodeFile(path, &c)
	if err != nil {
		return nil, fmt.Errorf("decode file: %v", err)
	}
	return &c, nil
}

func getTemplate() (*template.Template, error) {
	return template.New("go-import").Parse(`<!DOCTYPE html>
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
	
		<p>Repository: <a href="{{.Repo}}">{{.Repo}}</a></p>
		<p>GoDoc: <a href="https://pkg.go.dev/{{.ImportPath}}">https://pkg.go.dev/{{.ImportPath}}</a></p>
	</body>
	</html>`)
}

func getDBWithStats(path string) (*badger.DB, *stats, error) {
	opts := badger.DefaultOptions(path)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("open: %v", err)
	}

	// Retrieve current stats in database.
	s := &stats{
		pkgsView: make(map[string]*int64),
		pkgsGet:  make(map[string]*int64),
	}
	if err = s.loadFromDB(db); err != nil {
		return nil, nil, fmt.Errorf("load stats from DB: %v", err)
	}

	return db, s, nil
}

func newServer(addr string, m *macaron.Macaron) *http.Server {
	s := &http.Server{
		Addr:    addr,
		Handler: m,
	}

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit

		if err := s.Shutdown(context.Background()); err != nil {
			log.Fatal("Failed to shutdown server: %v", err)
		}
	}()

	return s
}
