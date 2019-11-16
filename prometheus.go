package main

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func setupPrometheusMetrics(s *stats) {
	cs := []prometheus.Collector{
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Namespace: "go_import_server",
			Subsystem: "stats",
			Name:      "view_total",
			Help:      "Number of total page views",
		}, func() float64 {
			return float64(s.TotalView())
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Namespace: "go_import_server",
			Subsystem: "stats",
			Name:      "get_total",
			Help:      "Number of total 'go-get's",
		}, func() float64 {
			return float64(s.TotalGet())
		}),
	}

	sr := strings.NewReplacer(".", "_", "/", "_", "-", "_")
	for p := range s.pkgsView {
		p := p
		cs = append(cs, prometheus.NewCounterFunc(prometheus.CounterOpts{
			Namespace: "go_import_server",
			Subsystem: "stats",
			Name:      "view_" + sr.Replace(p),
			Help:      "Number of page views for " + p,
		}, func() float64 {
			return float64(s.PkgView(p))
		}))
	}

	for p := range s.pkgsGet {
		p := p
		cs = append(cs, prometheus.NewCounterFunc(prometheus.CounterOpts{
			Namespace: "go_import_server",
			Subsystem: "stats",
			Name:      "get_" + sr.Replace(p),
			Help:      "Number of 'go-get's for " + p,
		}, func() float64 {
			return float64(s.PkgGet(p))
		}))
	}

	prometheus.MustRegister(cs...)
}
