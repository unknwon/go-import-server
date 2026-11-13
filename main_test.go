package main

import (
	"testing"
)

func TestMatchPackage(t *testing.T) {
	pkgs := []packageConfig{
		{ImportPath: "unknwon.dev/x", Subpath: "/x"},
		{ImportPath: "unknwon.dev/x/y", Subpath: "/x/y"},
		{ImportPath: "unknwon.dev/clog/v2", Subpath: "/clog/v2"},
		{ImportPath: "unknwon.dev/git-lfs-server", Subpath: "/git-lfs-server"},
	}

	tests := []struct {
		name           string
		path           string
		wantImportPath string
		wantOK         bool
	}{
		{name: "exact root", path: "/x", wantImportPath: "unknwon.dev/x", wantOK: true},
		{name: "exact multi-segment root", path: "/clog/v2", wantImportPath: "unknwon.dev/clog/v2", wantOK: true},
		{name: "trailing slash", path: "/x/", wantImportPath: "unknwon.dev/x", wantOK: true},
		{name: "subpath", path: "/x/foo", wantImportPath: "unknwon.dev/x", wantOK: true},
		{name: "deep subpath", path: "/clog/v2/internal/errutil", wantImportPath: "unknwon.dev/clog/v2", wantOK: true},
		{name: "nested module", path: "/git-lfs-server/lfs", wantImportPath: "unknwon.dev/git-lfs-server", wantOK: true},
		{name: "longest prefix wins for exact match", path: "/x/y", wantImportPath: "unknwon.dev/x/y", wantOK: true},
		{name: "longest prefix wins for subpath", path: "/x/y/z", wantImportPath: "unknwon.dev/x/y", wantOK: true},
		{name: "sibling of longer prefix", path: "/x/yz", wantImportPath: "unknwon.dev/x", wantOK: true},
		{name: "not on segment boundary", path: "/xfoo", wantOK: false},
		{name: "not on segment boundary with dash", path: "/git-lfs-serverx", wantOK: false},
		{name: "parent of root", path: "/clog", wantOK: false},
		{name: "no match", path: "/foo/bar", wantOK: false},
		{name: "home", path: "/", wantOK: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := matchPackage(pkgs, test.path)
			if ok != test.wantOK {
				t.Fatalf("ok: want %v but got %v", test.wantOK, ok)
			}
			if got.ImportPath != test.wantImportPath {
				t.Fatalf("import path: want %q but got %q", test.wantImportPath, got.ImportPath)
			}
		})
	}
}

func TestRequestedImportPath(t *testing.T) {
	pkg := packageConfig{ImportPath: "unknwon.dev/git-lfs-server", Subpath: "/git-lfs-server"}
	tests := []struct {
		path string
		want string
	}{
		{path: "/git-lfs-server", want: "unknwon.dev/git-lfs-server"},
		{path: "/git-lfs-server/", want: "unknwon.dev/git-lfs-server"},
		{path: "/git-lfs-server/lfs", want: "unknwon.dev/git-lfs-server/lfs"},
		{path: "/git-lfs-server/lfs/storage/", want: "unknwon.dev/git-lfs-server/lfs/storage"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			if got := requestedImportPath(pkg, test.path); got != test.want {
				t.Fatalf("want %q but got %q", test.want, got)
			}
		})
	}
}
