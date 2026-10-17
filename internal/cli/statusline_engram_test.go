package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeEngramOnPATH installs a fake `engram` binary on PATH that prints the
// standard stats text and counts invocations in a file; it returns the
// invocation counter path.
func fakeEngramOnPATH(t *testing.T, observations int) string {
	t.Helper()
	binDir := t.TempDir()
	counter := filepath.Join(binDir, "calls")
	script := "#!/bin/sh\necho 1 >> " + counter +
		"\nprintf 'Engram Memory Stats\\n  Sessions:     263\\n  Observations:  " +
		strconv.Itoa(observations) + "\\n  Projects:     gentle-ai\\n  Prompts:      1194\\n'\n"
	if err := os.WriteFile(filepath.Join(binDir, "engram"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
	return counter
}

func engramCallCount(t *testing.T, counter string) int {
	t.Helper()
	raw, err := os.ReadFile(counter)
	if err != nil {
		return 0
	}
	return len(strings.Split(strings.TrimSpace(string(raw)), "\n"))
}

// TestStatusLineEngramObservations pins the subprocess invocation and the
// Observations parse.
func TestStatusLineEngramObservations(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("HOME", home)
	counter := fakeEngramOnPATH(t, 918)

	if project, got := statusLineEngramStats(home, cwd); got != 918 || project != "gentle-ai" {
		t.Errorf("stats = (%q, %d), want (gentle-ai, 918)", project, got)
	}
	if calls := engramCallCount(t, counter); calls != 1 {
		t.Errorf("expected one engram invocation, got %d", calls)
	}

	// The second call is served from the cache: no extra invocation.
	if _, got := statusLineEngramStats(home, cwd); got != 918 {
		t.Errorf("cached observations = %d, want 918", got)
	}
	if calls := engramCallCount(t, counter); calls != 1 {
		t.Errorf("cache miss caused a second invocation (%d calls)", calls)
	}

	// Another cwd is a different cache entry: one more invocation.
	other := t.TempDir()
	if _, got := statusLineEngramStats(home, other); got != 918 {
		t.Errorf("other cwd observations = %d, want 918", got)
	}
	if calls := engramCallCount(t, counter); calls != 2 {
		t.Errorf("expected two invocations for two cwds, got %d", calls)
	}
}

// TestStatusLineEngramStaleCacheReinvokes pins the 30s TTL.
func TestStatusLineEngramStaleCacheReinvokes(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("HOME", home)
	counter := fakeEngramOnPATH(t, 42)

	if _, got := statusLineEngramStats(home, cwd); got != 42 {
		t.Fatalf("first read = %d", got)
	}

	// Age the cache past the TTL.
	path := engramCachePath(home)
	stale := time.Now().Add(-time.Minute).Unix()
	record := engramCacheRecord{Cwd: cwd, TS: stale, Observations: 7}
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, got := statusLineEngramStats(home, cwd); got != 42 {
		t.Errorf("stale cache served %d, want a fresh 42", got)
	}
	if calls := engramCallCount(t, counter); calls != 2 {
		t.Errorf("stale cache did not re-invoke engram (%d calls)", calls)
	}
}

// TestStatusLineEngramFailuresHideSegment pins that a missing binary or
// unparsable output yields zero without writing a poisoned cache entry that
// hides a later good run.
func TestStatusLineEngramFailuresHideSegment(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("HOME", home)

	// No engram on PATH at all.
	t.Setenv("PATH", t.TempDir())
	if _, got := statusLineEngramStats(home, cwd); got != 0 {
		t.Errorf("missing binary returned %d, want 0", got)
	}

	// A binary whose output has no Observations line.
	binDir := t.TempDir()
	script := "#!/bin/sh\nprintf 'nothing here\\n'\n"
	if err := os.WriteFile(filepath.Join(binDir, "engram"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	if project, got := statusLineEngramStats(home, cwd); got != 0 || project != "" {
		t.Errorf("unparsable output returned (%q, %d), want empty", project, got)
	}
}

// TestRunStatusLineRendersEngramSegment pins the end-to-end segment.
func TestRunStatusLineRendersEngramSegment(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	fakeEngramOnPATH(t, 314)
	if err := os.MkdirAll(filepath.Join(home, "project"), 0o755); err != nil {
		t.Fatal(err)
	}
	payload := `{
		"cwd": "` + home + `/project",
		"model": {"display_name": "Gemini 3.5 Pro"},
		"context_window": {"used_percentage": 10},
		"terminal_width": 200
	}`
	line := runStatusLineTest(t, nil, payload, home)
	if !strings.Contains(line, "🧠 gentle-ai · 314") {
		t.Errorf("engram segment missing: %s", line)
	}
}
