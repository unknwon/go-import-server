package main

import (
	"encoding/json"
	"os"
	"sync/atomic"
	"time"

	log "unknwon.dev/clog/v2"
)

// NOTE: all int64 must be manipulated atomically.
type stats struct {
	totalView int64
	totalGet  int64
	pkgsView  map[string]*int64
	pkgsGet   map[string]*int64

	lastUpdated int64
	lastSynced  int64
}

func (s *stats) TotalView() int64 {
	return atomic.LoadInt64(&s.totalView)
}

func (s *stats) TotalGet() int64 {
	return atomic.LoadInt64(&s.totalGet)
}

func (s *stats) PkgView(improtPath string) int64 {
	return atomic.LoadInt64(s.pkgsView[improtPath])
}

func (s *stats) PkgGet(improtPath string) int64 {
	return atomic.LoadInt64(s.pkgsGet[improtPath])
}

func (s *stats) PkgViewIncr(improtPath string, n int64) {
	atomic.AddInt64(s.pkgsView[improtPath], n)
	atomic.AddInt64(&s.totalView, n)
	atomic.StoreInt64(&s.lastUpdated, time.Now().Unix())
}

func (s *stats) PkgGetIncr(improtPath string, n int64) {
	atomic.AddInt64(s.pkgsGet[improtPath], n)
	atomic.AddInt64(&s.totalGet, n)
	atomic.StoreInt64(&s.lastUpdated, time.Now().Unix())
}

// statsData is the structure for JSON serialization
type statsData struct {
	TotalView int64            `json:"total_view"`
	TotalGet  int64            `json:"total_get"`
	PkgsView  map[string]int64 `json:"pkgs_view"`
	PkgsGet   map[string]int64 `json:"pkgs_get"`
}

// NOTE: atomic operation is not needed in this method since it is currently only
// being called at init time.
func (s *stats) loadFromJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, which is fine
			return nil
		}
		return err
	}

	var sd statsData
	if err := json.Unmarshal(data, &sd); err != nil {
		return err
	}

	s.totalView = sd.TotalView
	s.totalGet = sd.TotalGet

	for importPath, view := range sd.PkgsView {
		v := view
		s.pkgsView[importPath] = &v
	}

	for importPath, get := range sd.PkgsGet {
		g := get
		s.pkgsGet[importPath] = &g
	}

	return nil
}

func (s *stats) start(path string, done chan struct{}) {
	defer func() {
		log.Info("Exiting stats syncing goroutine...")
		done <- struct{}{}
	}()

	t := time.NewTicker(time.Minute)
	for {
		select {
		case <-t.C:
			s.syncToJSON(path)
		case <-done:
			s.syncToJSON(path)
			return
		}
	}
}

func (s *stats) syncToJSON(path string) {
	lastSynced := atomic.LoadInt64(&s.lastSynced)
	lastUpdated := atomic.LoadInt64(&s.lastUpdated)
	if lastSynced == lastUpdated {
		log.Trace("stats.syncToJSON: nothing changed, file is up-to-date")
		return
	}

	sd := statsData{
		TotalView: s.TotalView(),
		TotalGet:  s.TotalGet(),
		PkgsView:  make(map[string]int64),
		PkgsGet:   make(map[string]int64),
	}

	for p := range s.pkgsView {
		sd.PkgsView[p] = s.PkgView(p)
		sd.PkgsGet[p] = s.PkgGet(p)
	}

	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		log.Error("Failed to marshal stats: %v", err)
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Error("Failed to write stats file: %v", err)
		return
	}

	atomic.StoreInt64(&s.lastSynced, lastUpdated)
}
