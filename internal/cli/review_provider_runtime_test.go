package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v3/internal/model"
	"github.com/gentleman-programming/gentle-ai/v3/internal/reviewerprovider"
	"github.com/gentleman-programming/gentle-ai/v3/internal/state"
)

func TestReviewProviderAdapterWithRoutingAntigravityCLI(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := filepath.Join(home, ".gemini", "antigravity-cli", "gentle-ai")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "review-models.json"), []byte(`{"review-risk":"provider/model"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter, err := reviewProviderAdapterWithRouting(reviewProviderRoleLens, model.AgentAntigravityCLI, reviewerprovider.ReviewRoutingKeyRisk)
	if err != nil {
		t.Fatalf("with-routing adapter error = %v", err)
	}
	agy, ok := adapter.(*reviewerprovider.AntigravityCLIAdapter)
	if !ok {
		t.Fatalf("adapter = %T, want *reviewerprovider.AntigravityCLIAdapter", adapter)
	}
	if agy.Model != "provider/model" {
		t.Fatalf("adapter routing = %q, want provider/model", agy.Model)
	}
}

func TestReviewProviderAdapterWithRoutingCanonicalKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := filepath.Join(home, ".gemini", "antigravity-cli", "gentle-ai")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	fixture := `{
		"review-refuter":"refuter/model",
		"review-validator":"validator/model",
		"review-risk":"risk/model",
		"review-resilience":"resilience/model",
		"review-readability":{"model":"readability/model","effort":"high"},
		"review-reliability":"reliability/model"
	}`
	if err := os.WriteFile(filepath.Join(dir, "review-models.json"), []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		key, model string
	}{
		{reviewerprovider.ReviewRoutingKeyRefuter, "refuter/model"},
		{reviewerprovider.ReviewRoutingKeyValidator, "validator/model"},
		{reviewerprovider.ReviewRoutingKeyRisk, "risk/model"},
		{reviewerprovider.ReviewRoutingKeyResilience, "resilience/model"},
		{reviewerprovider.ReviewRoutingKeyReadability, "readability/model"},
		{reviewerprovider.ReviewRoutingKeyReliability, "reliability/model"},
	}
	for _, tc := range cases {
		adapter, err := reviewProviderAdapterWithRouting(reviewProviderRoleLens, model.AgentAntigravityCLI, tc.key)
		if err != nil {
			t.Fatalf("with-routing adapter error for %q = %v", tc.key, err)
		}
		agy, ok := adapter.(*reviewerprovider.AntigravityCLIAdapter)
		if !ok {
			t.Fatalf("adapter for %q = %T, want *reviewerprovider.AntigravityCLIAdapter", tc.key, adapter)
		}
		if agy.Model != tc.model {
			t.Fatalf("adapter routing for %q = %q, want %q", tc.key, agy.Model, tc.model)
		}
	}
}

func TestReviewProviderAdapterWithRoutingPreservesNativeModelAssignments(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	persisted := state.InstallState{
		ClaudePhaseAssignments: map[string]state.ClaudePhaseAssignmentState{
			"risk":      {Model: "opus"},
			"refuter":   {Model: "sonnet"},
			"validator": {Model: "haiku"},
		},
		CodexPhaseModelAssignments: map[string]string{
			"rdd-risk":      "gpt-6-astra",
			"rdd-refuter":   "gpt-5.4",
			"rdd-validator": "gpt-5.4-mini",
		},
	}
	if err := state.Write(home, persisted); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name, role, routingKey, claudeModel, codexModel string
	}{
		{"lens", reviewProviderRoleLens, reviewerprovider.ReviewRoutingKeyRisk, "opus", "gpt-6-astra"},
		{"refuter", reviewProviderRoleRefuter, reviewerprovider.ReviewRoutingKeyRefuter, "sonnet", "gpt-5.4"},
		{"validator", reviewProviderRoleTargetedValidator, reviewerprovider.ReviewRoutingKeyValidator, "haiku", "gpt-5.4-mini"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			claude, err := reviewProviderAdapterWithRouting(tc.role, model.AgentClaudeCode, tc.routingKey)
			if err != nil {
				t.Fatal(err)
			}
			if got := claude.(*reviewerprovider.ClaudeAdapter).Model; got != model.ClaudeModelAlias(tc.claudeModel) {
				t.Fatalf("Claude model = %q, want %q", got, tc.claudeModel)
			}

			codex, err := reviewProviderAdapterWithRouting(tc.role, model.AgentCodex, tc.routingKey)
			if err != nil {
				t.Fatal(err)
			}
			if got := codex.(*reviewerprovider.CodexAdapter).Model; got != tc.codexModel {
				t.Fatalf("Codex model = %q, want %q", got, tc.codexModel)
			}
		})
	}
}

func TestReviewProviderAdapterWithRoutingMissingKeyDiagnostic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := filepath.Join(home, ".gemini", "antigravity-cli", "gentle-ai")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "review-models.json"), []byte(`{"review-risk":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stderr
	os.Stderr = writer
	t.Cleanup(func() {
		os.Stderr = previous
		_ = reader.Close()
	})
	adapter, err := reviewProviderAdapterWithRouting(reviewProviderRoleLens, model.AgentAntigravityCLI, reviewerprovider.ReviewRoutingKeyReliability)
	if err != nil {
		t.Fatalf("with-routing adapter error = %v", err)
	}
	agy, ok := adapter.(*reviewerprovider.AntigravityCLIAdapter)
	if !ok {
		t.Fatalf("adapter = %T, want *reviewerprovider.AntigravityCLIAdapter", adapter)
	}
	if agy.Model != "" {
		t.Fatalf("adapter routing = %q, want zero values", agy.Model)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), `no assignment for "review-reliability"`) {
		t.Fatalf("stderr diagnostic = %q, want it to contain no assignment for \"review-reliability\"", output)
	}
}

func TestReviewProviderAdapterWithRoutingDelegatesOtherRuntimes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	previous := reviewProviderAdapterFor
	reviewProviderAdapterFor = func(contract reviewerprovider.Contract, agent model.AgentID) (reviewerprovider.Adapter, error) {
		return providerTestAdapter{raw: []byte("seam result")}, nil
	}
	t.Cleanup(func() { reviewProviderAdapterFor = previous })

	// Claude Code and Codex both capture in process, yet both must keep
	// delegating through the seam: neither owns the antigravity-cli home
	// routing assignment, so the AGY branch must never intercept them.
	for _, agent := range []model.AgentID{model.AgentClaudeCode, model.AgentCodex} {
		seam, err := reviewProviderAdapterWithRouting(reviewProviderRoleLens, agent, reviewerprovider.ReviewRoutingKeyRisk)
		if err != nil {
			t.Fatalf("delegated adapter error for %q = %v", agent, err)
		}
		if _, ok := seam.(providerTestAdapter); !ok {
			t.Fatalf("non-AGY adapter for %q = %T, want the reviewProviderAdapterFor seam result", agent, seam)
		}
	}

	// The AGY branch must bypass the seam entirely: with the same stub active
	// and no assignment file, it still returns the compiled adapter with the
	// zero routing (runtime default), never the seam result and never an error.
	agy, err := reviewProviderAdapterWithRouting(reviewProviderRoleLens, model.AgentAntigravityCLI, reviewerprovider.ReviewRoutingKeyRisk)
	if err != nil {
		t.Fatalf("AGY adapter error = %v", err)
	}
	branch, ok := agy.(*reviewerprovider.AntigravityCLIAdapter)
	if !ok {
		t.Fatalf("AGY adapter = %T, want *reviewerprovider.AntigravityCLIAdapter (seam must not be consulted)", agy)
	}
	if branch.Model != "" {
		t.Fatalf("AGY adapter routing without assignment = %q, want zero values", branch.Model)
	}
}
