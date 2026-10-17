package cli

import (
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
	if got := rddStateLabel("mystery"); got != "" {
		t.Errorf("unknown state mapped to %q, want empty fallback", got)
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
