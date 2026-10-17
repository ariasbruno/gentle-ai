package cli

import (
	"bytes"
	"context"
	"github.com/gentleman-programming/gentle-ai/v4/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/reviewtransaction"
)

// TestRDDStateLabel pins the authority-state → display-label map (S15).
func TestRDDStateLabel(t *testing.T) {
	cases := map[reviewtransaction.State]string{
		reviewtransaction.StateUnreviewed:             "Not reviewed yet",
		reviewtransaction.StateReviewing:              "Review in progress",
		reviewtransaction.StateFindingsFrozen:         "Review in progress",
		reviewtransaction.StateEvidenceClassified:     "Review in progress",
		reviewtransaction.StateFixRequired:            "Fixing findings…",
		reviewtransaction.StateFixing:                 "Fixing findings…",
		reviewtransaction.StateFixValidating:          "Fixing findings…",
		reviewtransaction.StateReadyFinalVerification: "Verifying…",
		reviewtransaction.StateFinalVerifying:         "Verifying…",
		reviewtransaction.StateApproved:               "✓ Approved",
		reviewtransaction.StateEscalated:              "Needs your decision",
		reviewtransaction.StateInvalidated:            "Outdated · code changed",
	}
	for state, want := range cases {
		if got := rddStateLabel(state); got != want {
			t.Errorf("rddStateLabel(%q) = %q, want %q", state, got, want)
		}
	}
	if got := rddStateLabel("mystery"); got != "Review unavailable" {
		t.Errorf("unknown state mapped to %q, want Review unavailable", got)
	}
}

// TestResolveStatusLineRDDNoRepoShowsSwitch pins the plain switch display
// when the cwd is not a reviewable repository: no authority → on.
func TestResolveStatusLineRDDNoRepoShowsSwitch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	plain := t.TempDir()
	if got := resolveStatusLineRDD(home, plain); got != "on" {
		t.Errorf("resolveStatusLineRDD(no repo) = %q, want on", got)
	}
	// The result is cached per cwd: a second call is served from the cache.
	if got := resolveStatusLineRDD(home, plain); got != "on" {
		t.Errorf("cached resolveStatusLineRDD = %q, want on", got)
	}
}

func TestRunStatusLineRDDOffAndUnavailable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := initReviewCLIRepo(t)
	var out bytes.Buffer
	if err := RunReviewMode([]string{"disable", "--cwd", repo}, &out); err != nil {
		t.Fatal(err)
	}
	payload := `{"cwd":"` + repo + `","terminal_width":200}`
	line := runStatusLineTest(t, nil, payload, home)
	if !strings.Contains(line, "🌹 off") {
		t.Fatal(line)
	}
	// A malformed persisted mode must be visible as unavailable, not on/off.
	if err := os.WriteFile(state.Path(home), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	line = runStatusLineTest(t, nil, payload, home)
	if !strings.Contains(line, "🌹 Review unavailable") {
		t.Fatal(line)
	}
	after, err := os.ReadFile(state.Path(home))
	if err != nil || string(after) != "{" {
		t.Fatalf("read changed state: %q, %v", after, err)
	}
}

func TestRDDCacheSharedByRepositoryRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := initReviewCLIRepo(t)
	root, err := (reviewtransaction.SnapshotBuilder{Repo: repo}).ResolveRepositoryRoot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(repo, "child")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	writeRDDStateCache(home, root, "Review in progress")
	if got := resolveStatusLineRDD(home, child); got != "Review in progress" {
		t.Fatalf("child cache: %q", got)
	}
}
