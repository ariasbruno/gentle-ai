package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	antigravitycliagent "github.com/gentleman-programming/gentle-ai/v4/internal/agents/antigravitycli"
	"github.com/gentleman-programming/gentle-ai/v4/internal/components/persona"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
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
	for _, want := range []string{
		"~/.gemini/config/plugins/gentle-ai", "<workspace>/.agents/plugins/gentle-ai",
		"Persona: `rules/AGENTS.md` at that root.",
		"Required rules (relative to plugin root's `rules/`)",
		"ODD and applicable test-first obligations are mandatory",
		"Before the associated phase or action",
		"`strict-tdd.md`", "`strict-tdd-verify.md`",
		"Every task follows ODD", "no writer or artifacts", "ask once and stop",
		"Explore before proposing or writing", "Clarify real uncertainty",
		"Track substantial authorized work", "injected language policy",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("rendered always_on core lost obligation or route %q", want)
		}
	}
	t.Logf("rendered core: %d bytes", len(data))
	if len(data) >= 24000 {
		t.Fatalf("orchestrator rule exceeds agy per-file cap of 24000 bytes: %d", len(data))
	}

	// Idempotent: the actual sync sequence redeploys then fills the core.
	if err := adapter.DeployPluginTree(home); err != nil {
		t.Fatalf("sync redeploy error = %v", err)
	}
	// A second fill must retain always_on and canonical RDD bytes.
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

func TestAntigravityOrchestratorRuleActualPersonaSelections(t *testing.T) {
	for _, selected := range []model.PersonaID{model.PersonaGentleman, model.PersonaNeutral, model.PersonaCustom} {
		t.Run(string(selected), func(t *testing.T) {
			home, adapter := t.TempDir(), antigravitycliagent.NewAdapter()
			if err := adapter.DeployPluginTree(home); err != nil {
				t.Fatal(err)
			}
			path := adapter.SystemPromptFile(home)
			if selected == model.PersonaCustom {
				if err := os.WriteFile(path, []byte("## Language\nReply in French.\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := persona.Inject(home, adapter, selected); err != nil {
				t.Fatal(err)
			}
			policy, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(policy, []byte("## Language")) || bytes.Contains(policy, []byte("Current persona mode:")) {
				t.Fatal("unexpected actual language policy")
			}
			if err := ensureAntigravityOrchestratorRule(home, adapter); err != nil {
				t.Fatal(err)
			}
			core, err := os.ReadFile(filepath.Join(adapter.SystemPromptDir(home), "gentle-ai-orchestrator.md"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(core, []byte("injected language policy")) || bytes.Contains(core, []byte("Current persona mode:")) {
				t.Fatal("core relies on nonexistent persona marker")
			}
		})
	}
}

func TestEnsureAntigravityOrchestratorRuleRejectsMissingDeploy(t *testing.T) {
	home := t.TempDir()
	adapter := antigravitycliagent.NewAdapter()
	if err := ensureAntigravityOrchestratorRule(home, adapter); err == nil {
		t.Fatalf("expected error when the plugin tree was never deployed")
	}
}
