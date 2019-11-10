package main

import (
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/v2"
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

// NOTE: atomic operation is not needed in this method since it is currently only
// being called at init time.
func (s *stats) loadFromDB(db *badger.DB) error {
	return db.View(func(tx *badger.Txn) error {
		iter := tx.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			k := item.Key()
			err := item.Value(func(v []byte) (err error) {
				ks := string(k)
				if ks == "view_total" {
					s.totalView, _ = strconv.ParseInt(string(v), 10, 64)
					return nil
				} else if ks == "get_total" {
					s.totalGet, _ = strconv.ParseInt(string(v), 10, 64)
					return nil
				}

				if strings.HasPrefix(ks, "view_") {
					importPath := strings.TrimPrefix(ks, "view_")
					pkgView, _ := strconv.ParseInt(string(v), 10, 64)
					s.pkgsView[importPath] = &pkgView
					return nil
				}

				if strings.HasPrefix(ks, "get_") {
					importPath := strings.TrimPrefix(ks, "get_")
					pkgGet, _ := strconv.ParseInt(string(v), 10, 64)
					s.pkgsGet[importPath] = &pkgGet
					return nil
				}

				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *stats) start(db *badger.DB, done chan struct{}) {
	defer func() {
		log.Info("Exiting stats syncing goroutine...")
		done <- struct{}{}
	}()

	t := time.NewTicker(time.Minute)
	for {
		select {
		case <-t.C:
			s.syncToDB(db)
		case <-done:
			s.syncToDB(db)
			return
		}
	}
}

func (s *stats) syncToDB(db *badger.DB) {
	lastSynced := atomic.LoadInt64(&s.lastSynced)
	lastUpdated := atomic.LoadInt64(&s.lastUpdated)
	if lastSynced == lastUpdated {
		log.Trace("stats.syncToDB: nothing changed, DB is up-to-date")
		return
	}

	err := db.Update(func(tx *badger.Txn) error {
		err := tx.Set([]byte("view_total"), []byte(strconv.FormatInt(s.TotalView(), 10)))
		if err != nil {
			return err
		}

		err = tx.Set([]byte("get_total"), []byte(strconv.FormatInt(s.TotalGet(), 10)))
		if err != nil {
			return err
		}

		for p := range s.pkgsView {
			err = tx.Set([]byte("view_"+p), []byte(strconv.FormatInt(s.PkgView(p), 10)))
			if err != nil {
				return err
			}

			err = tx.Set([]byte("get_"+p), []byte(strconv.FormatInt(s.PkgGet(p), 10)))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Error("Failed to update DB: %v", err)
		return
	}

	atomic.StoreInt64(&s.lastSynced, lastUpdated)
}
