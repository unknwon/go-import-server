package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/flamego/auth"
	"github.com/flamego/flamego"
	"github.com/flamego/template"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	stats, err := getStatsFromJSON(config.StatsPath)
	if err != nil {
		log.Fatal("Failed to load stats: %v", err)
	}

	fs, err := template.EmbedFS(templates, "templates", []string{".tmpl"})
	if err != nil {
		log.Fatal("Failed to load templates: %v", err)
	}

	f := flamego.New()
	f.Use(template.Templater(
		template.Options{
			FileSystem: fs,
		},
	))

	f.Get("/", func(t template.Template, data template.Data) {
		data["Packages"] = config.Packages
		t.HTML(http.StatusOK, "home")
	})

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
	}

	// Requests for any path under a package (e.g., a nested module or a
	// subpackage) are served with the meta tags of the package itself, as
	// required by "go get" for remote import paths.
	f.Get("/{**: **}", func(w http.ResponseWriter, r *http.Request, t template.Template, data template.Data) {
		pkg, ok := matchPackage(config.Packages, r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}

		data["Package"] = pkg
		t.HTML(http.StatusOK, "page")

		log.Trace("Page served: %s", r.URL.Path)

		if r.URL.Query().Get("go-get") == "1" {
			stats.PkgGetIncr(pkg.ImportPath, 1)
		} else {
			stats.PkgViewIncr(pkg.ImportPath, 1)
		}
	})

	f.Get("/-/metrics",
		func(c flamego.Context) {
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
	go stats.start(config.StatsPath, done)

	s := newServer(config.Addr, f)
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
	Addr       string
	StatsPath  string `toml:"stats_path"`
	Packages   []packageConfig
	Prometheus struct {
		AuthUsername string `toml:"auth_username"`
		AuthPassword string `toml:"auth_password"`
	}
}

type packageConfig struct {
	ImportPath string `toml:"import_path"`
	Subpath    string
	Repo       string
	Branch     string
}

// matchPackage returns the package whose subpath is the longest match of the
// given path on path segment boundaries, e.g., "/x/foo" matches "/x" but
// "/xfoo" does not.
func matchPackage(pkgs []packageConfig, path string) (packageConfig, bool) {
	var matched packageConfig
	var ok bool
	for _, pkg := range pkgs {
		if path != pkg.Subpath && !strings.HasPrefix(path, strings.TrimSuffix(pkg.Subpath, "/")+"/") {
			continue
		}
		if !ok || len(pkg.Subpath) > len(matched.Subpath) {
			matched, ok = pkg, true
		}
	}
	return matched, ok
}

func loadConfig(path string) (*config, error) {
	var c config
	_, err := toml.DecodeFile(path, &c)
	if err != nil {
		return nil, fmt.Errorf("decode file: %v", err)
	}
	return &c, nil
}

func getStatsFromJSON(path string) (*stats, error) {
	s := &stats{
		pkgsView: make(map[string]*int64),
		pkgsGet:  make(map[string]*int64),
	}
	if err := s.loadFromJSON(path); err != nil {
		return nil, fmt.Errorf("load stats from JSON: %v", err)
	}

	return s, nil
}

func newServer(addr string, f *flamego.Flame) *http.Server {
	s := &http.Server{
		Addr:    addr,
		Handler: f,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit

		if err := s.Shutdown(context.Background()); err != nil {
			log.Fatal("Failed to shutdown server: %v", err)
		}
	}()

	return s
}
