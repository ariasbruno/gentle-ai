package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	antigravitycliagent "github.com/gentleman-programming/gentle-ai/v3/internal/agents/antigravitycli"
)

func TestEnsureAntigravityOrchestratorRuleFillsContractAndFrontmatter(t *testing.T) {
	home := t.TempDir()
	adapter := antigravitycliagent.NewAdapter()
	if err := adapter.DeployPluginTree(home); err != nil {
		t.Fatalf("DeployPluginTree error = %v", err)
	}

	if err := ensureAntigravityOrchestratorRule(home, adapter); err != nil {
		t.Fatalf("ensureAntigravityOrchestratorRule error = %v", err)
	}

	rulePath := filepath.Join(adapter.PluginDir(home), "rules", "gentle-ai-orchestrator.md")
	data, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatalf("read %q: %v", rulePath, err)
	}
	content := string(data)
	if !strings.HasPrefix(content, "---\ntrigger: always_on\n") {
		t.Fatalf("orchestrator rule must start with always_on frontmatter, got prefix:\n%.200q", content)
	}
	if bytes.Contains(data, []byte(antigravitycliagent.ReviewContractInsertMarker)) {
		t.Fatalf("orchestrator rule still contains the unfilled insert marker")
	}
	if !strings.Contains(content, "## Entry rule") {
		t.Fatalf("orchestrator rule missing rendered review execution contract")
	}
	if !strings.Contains(content, "Concurrent Reviewer Group") {
		t.Fatalf("orchestrator rule missing the concurrent reviewer group contract section")
	}
	if !strings.Contains(content, "#### Review Execution Contract") {
		t.Fatalf("orchestrator rule lost its Review Execution Contract heading")
	}
	if len(data) >= 24000 {
		t.Fatalf("orchestrator rule exceeds agy per-file cap of 24000 bytes: %d", len(data))
	}

	// Idempotent: a second fill (as every sync performs) must be byte-identical.
	if err := ensureAntigravityOrchestratorRule(home, adapter); err != nil {
		t.Fatalf("second ensureAntigravityOrchestratorRule error = %v", err)
	}
	again, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatalf("re-read %q: %v", rulePath, err)
	}
	if !bytes.Equal(data, again) {
		t.Fatalf("second fill changed the orchestrator rule bytes")
	}
}

func TestEnsureAntigravityOrchestratorRuleRejectsMissingDeploy(t *testing.T) {
	home := t.TempDir()
	adapter := antigravitycliagent.NewAdapter()
	if err := ensureAntigravityOrchestratorRule(home, adapter); err == nil {
		t.Fatalf("expected error when the plugin tree was never deployed")
	}
}
