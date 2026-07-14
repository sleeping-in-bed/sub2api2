package datadir

import (
	"path/filepath"
	"testing"
)

func TestResolvePrefersExplicitDataDir(t *testing.T) {
	got := resolve(" /srv/sub2api-data ", filepath.Join(t.TempDir(), "missing"))
	if got != "/srv/sub2api-data" {
		t.Fatalf("resolve explicit data dir: got %q", got)
	}
}

func TestResolveUsesWritableContainerDir(t *testing.T) {
	writableDir := t.TempDir()
	got := resolve("", writableDir)
	if got != writableDir {
		t.Fatalf("resolve writable container dir: got %q, want %q", got, writableDir)
	}
}

func TestResolveFallsBackToCurrentDirectory(t *testing.T) {
	got := resolve("", filepath.Join(t.TempDir(), "missing"))
	if got != "." {
		t.Fatalf("resolve fallback data dir: got %q", got)
	}
}
