package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestStatusLineChangesCached pins the S18 cache: a fresh record serves
// without re-running git, expiry recomputes, and a dirty repo renders.
func TestStatusLineChangesCached(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := initReviewCLIRepo(t)

	// Dirty the worktree so the summary is non-empty.
	dirty := filepath.Join(repo, "feature.txt")
	if err := os.WriteFile(dirty, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	first := statusLineGitChangesCached(home, repo)
	if !strings.HasPrefix(first, "±") || !strings.Contains(first, "+") {
		t.Fatalf("dirty repo summary empty: %q", first)
	}

	// Rewrite the fresh cache with a sentinel: a cached read serves it.
	sentinel := changesCacheRecord{Cwd: repo, TS: time.Now().Unix(), Changes: "±SENTINEL"}
	encoded, err := json.Marshal(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(changesCachePath(home), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := statusLineGitChangesCached(home, repo); got != "±SENTINEL" {
		t.Errorf("fresh cache not served: %q", got)
	}

	// A stale record recomputes: the sentinel disappears.
	stale := changesCacheRecord{Cwd: repo, TS: time.Now().Add(-time.Minute).Unix(), Changes: "±STALE"}
	if encoded, err = json.Marshal(stale); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(changesCachePath(home), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := statusLineGitChangesCached(home, repo); got == "±STALE" || !strings.HasPrefix(got, "±") {
		t.Errorf("stale cache served: %q", got)
	}

	// Another cwd is a different cache entry.
	other := initReviewCLIRepo(t)
	if got := statusLineGitChangesCached(home, other); got != "" {
		t.Errorf("clean repo rendered changes: %q", got)
	}
}
