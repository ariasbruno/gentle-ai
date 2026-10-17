package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestParseODDTasklist pins the ## Tasks parser: counting, the in-progress
// pick, the pending fallback, blocked counting, and section boundaries.
func TestParseODDTasklist(t *testing.T) {
	doc := `# Feature

## Specs

S1. Something with a - T99 | inline | done | decoy line.

## Tasks

- T1 | S1 | inline | done | commit ` + "`abc`" + `
- T2 | S1 | inline | in_progress | evidence
- T3 | S1 | inline | pending |
- T4 | S1 | inline | blocked | waits on relay
- prose line without pipes

## Log

- T9 | S1 | inline | done | must not count
`
	stats := parseODDTasklist(doc)
	if stats.Total != 4 || stats.Done != 1 || stats.Blocked != 1 {
		t.Errorf("counts = %+v, want total 4 done 1 blocked 1", stats)
	}
	if stats.CurrentID != "T2" || stats.CurrentStatus != "in_progress" {
		t.Errorf("current = %q %q, want T2 in_progress", stats.CurrentID, stats.CurrentStatus)
	}

	pendingOnly := "## Tasks\n\n- T1 | S1 | inline | done | x\n- T2 | S1 | inline | pending | y\n"
	stats = parseODDTasklist(pendingOnly)
	if stats.CurrentID != "T2" || stats.CurrentStatus != "pending" {
		t.Errorf("pending fallback = %q %q, want T2 pending", stats.CurrentID, stats.CurrentStatus)
	}

	if stats := parseODDTasklist("# no sections here\n- T1 | S1 | inline | done | x\n"); stats.Total != 0 {
		t.Errorf("tasks outside the section counted: %+v", stats)
	}
}

// TestStatusLineTasklistResolvesRepo pins the repo-root resolution, the
// newest-document pick, and the empty fallbacks.
func TestStatusLineTasklistResolvesRepo(t *testing.T) {
	repo := initReviewCLIRepo(t)
	tasksDir := filepath.Join(repo, "odd", "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	older := filepath.Join(tasksDir, "older-feature.md")
	newer := filepath.Join(tasksDir, "agy-statusline.md")
	if err := os.WriteFile(older, []byte("## Tasks\n\n- T1 | S1 | inline | done | x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newer, []byte("## Tasks\n\n- T5 | S1 | inline | in_progress | x\n- T6 | S1 | inline | pending | y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale := time.Now().Add(-time.Hour)
	if err := os.Chtimes(older, stale, stale); err != nil {
		t.Fatal(err)
	}

	stats := statusLineTasklist(repo)
	if stats.Total != 2 || stats.CurrentID != "T5" {
		t.Errorf("statusLineTasklist = %+v, want the newest doc's tasks", stats)
	}

	// A child directory resolves to the same repository root.
	child := filepath.Join(repo, "sub")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if stats := statusLineTasklist(child); stats.Total != 2 {
		t.Errorf("child resolution lost the tasks: %+v", stats)
	}

	// No odd/ directory: hidden, not an error.
	bare := initReviewCLIRepo(t)
	if stats := statusLineTasklist(bare); stats.Total != 0 {
		t.Errorf("repo without odd/tasks rendered tasks: %+v", stats)
	}

	// Not a repository at all: hidden.
	if stats := statusLineTasklist(t.TempDir()); stats.Total != 0 {
		t.Errorf("non-repo rendered tasks: %+v", stats)
	}
}
