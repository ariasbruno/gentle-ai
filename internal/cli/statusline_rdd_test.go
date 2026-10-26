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
		reviewtransaction.StateCorrectionRequired:     "Fixing findings…",
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
	if !strings.Contains(stripStatusLineANSI(line), "🌹 off") {
		t.Fatal(line)
	}
	// A malformed persisted mode must be visible as unavailable, not on/off.
	if err := os.WriteFile(state.Path(home), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	line = runStatusLineTest(t, nil, payload, home)
	if !strings.Contains(stripStatusLineANSI(line), "🌹 Review unavailable") {
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

func TestResolveStatusLineRDDActiveReview(t *testing.T) {
	home := reviewEnabledHome(t)
	t.Setenv("HOME", home)
	repo := initReviewCLIRepo(t)
	writeReviewStartCandidate(t, repo, "candidate.go", "package candidate\nfunc f() {}\n", 0o644)
	runReviewCLIGit(t, repo, "add", "candidate.go")

	_, err := runLegacyFacadeStartForTestBytes(t, []string{"--cwd", repo, "--lineage", "test-active-review"})
	if err != nil {
		t.Fatal(err)
	}

	got := resolveStatusLineRDD(home, repo)
	if got != "Review in progress" {
		t.Fatalf("resolveStatusLineRDD = %q, want %q", got, "Review in progress")
	}
}

func TestResolveStatusLineRDDAcknowledgedApproved(t *testing.T) {
	home := reviewEnabledHome(t)
	t.Setenv("HOME", home)
	t.Setenv(reviewPiHostRelayContractEnvironment, reviewPiHostRelayContract)
	repo := initReviewCLIRepo(t)
	writeReviewStartCandidate(t, repo, "candidate.go", "package candidate\nfunc f() {}\n", 0o644)
	runReviewCLIGit(t, repo, "add", "candidate.go")

	startedBytes, err := runLegacyFacadeStartForTestBytes(t, []string{"--cwd", repo, "--lineage", "test-approved-review"})
	if err != nil {
		t.Fatal(err)
	}
	var started ReviewFacadeStartResult
	decodeStrictReviewJSON(t, startedBytes, &started)

	for order := 0; order < len(started.SelectedLenses)-1; order++ {
		captureCleanCLIReviewerResult(t, repo, started, order, &bytes.Buffer{})
	}
	var closureOutput bytes.Buffer
	captureCleanCLIReviewerResult(t, repo, started, len(started.SelectedLenses)-1, &closureOutput)
	var closure reviewLastEventClosureResult
	decodeStrictReviewJSON(t, closureOutput.Bytes(), &closure)

	_ = os.Remove(rddStateCachePath(home))

	if got := resolveStatusLineRDD(home, repo); got != "✓ Approved" {
		t.Fatalf("resolveStatusLineRDD(pending ack) = %q, want %q", got, "✓ Approved")
	}

	ackArgs := []string{strings.TrimPrefix(closure.Acknowledgement.Operation, "review.")}
	for _, argument := range closure.Acknowledgement.Arguments {
		ackArgs = append(ackArgs, argument.Token)
	}
	if err := RunReview(ackArgs, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	_ = os.Remove(rddStateCachePath(home))

	if got := resolveStatusLineRDD(home, repo); got != "✓ Approved" {
		t.Fatalf("resolveStatusLineRDD(consumed) = %q, want %q", got, "✓ Approved")
	}
}

func TestResolveStatusLineRDDUnrelatedLeafDoesNotMaskTarget(t *testing.T) {
	home := reviewEnabledHome(t)
	t.Setenv("HOME", home)
	repo := initReviewCLIRepo(t)

	writeReviewStartCandidate(t, repo, "candidate_a.go", "package candidate\nfunc A() {}\n", 0o644)
	runReviewCLIGit(t, repo, "add", "candidate_a.go")
	_, err := runLegacyFacadeStartForTestBytes(t, []string{"--cwd", repo, "--lineage", "test-leaf-a"})
	if err != nil {
		t.Fatal(err)
	}

	runReviewCLIGit(t, repo, "rm", "-f", "candidate_a.go")
	writeReviewStartCandidate(t, repo, "candidate_b.go", "package candidate\nfunc B() {}\n", 0o644)
	runReviewCLIGit(t, repo, "add", "candidate_b.go")

	_ = os.Remove(rddStateCachePath(home))

	if got := resolveStatusLineRDD(home, repo); got != "on" {
		t.Fatalf("resolveStatusLineRDD(unrelated leaf present) = %q, want %q", got, "on")
	}
}

func TestResolveStatusLineRDDNestedChildRepository(t *testing.T) {
	home := reviewEnabledHome(t)
	t.Setenv("HOME", home)
	parent := initReviewCLIRepo(t)
	childRepo := filepath.Join(parent, "child-repo")
	if err := os.MkdirAll(childRepo, 0o755); err != nil {
		t.Fatal(err)
	}
	runReviewCLIGit(t, childRepo, "init")
	runReviewCLIGit(t, childRepo, "config", "user.email", "tester@gentle-ai.dev")
	runReviewCLIGit(t, childRepo, "config", "user.name", "Gentle AI Tester")
	runReviewCLIGit(t, childRepo, "commit", "--allow-empty", "-m", "init")

	writeReviewStartCandidate(t, childRepo, "candidate.go", "package candidate\nfunc f() {}\n", 0o644)
	runReviewCLIGit(t, childRepo, "add", "candidate.go")

	_, err := runLegacyFacadeStartForTestBytes(t, []string{"--cwd", childRepo, "--lineage", "test-child-review"})
	if err != nil {
		t.Fatal(err)
	}

	_ = os.Remove(rddStateCachePath(home))

	if got := resolveStatusLineRDD(home, parent); got != "Review in progress" {
		t.Fatalf("resolveStatusLineRDD(parent workspace) = %q, want %q", got, "Review in progress")
	}
}
