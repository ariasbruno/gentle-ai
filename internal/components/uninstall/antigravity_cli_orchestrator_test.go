package uninstall

import (
	"path/filepath"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v3/internal/agents"
	"github.com/gentleman-programming/gentle-ai/v3/internal/agents/antigravitycli"
)

func TestRetainedAntigravityCLIPluginOperationsCoverOrchestratorRuleAndLegacyRoot(t *testing.T) {
	adapter, err := agents.NewAdapter("antigravity-cli")
	if err != nil {
		t.Fatalf("NewAdapter error = %v", err)
	}
	agy, ok := adapter.(*antigravitycli.Adapter)
	if !ok {
		t.Fatalf("adapter is not *antigravitycli.Adapter")
	}
	homeDir := t.TempDir()
	ops := retainedAntigravityCLIPluginOperations(adapter, homeDir)
	pluginDir := agy.PluginDir(homeDir)

	wanted := []string{
		filepath.Join(pluginDir, "rules", "gentle-ai-orchestrator.md"),
		filepath.Join(pluginDir, "rules", "gentle-ai-routing.md"),
		filepath.Join(pluginDir, "orchestrator.md"),
	}
	for _, want := range wanted {
		found := false
		for _, op := range ops {
			if op.path == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("uninstall operations missing %q (legacy root copies would survive reinstall otherwise)", want)
		}
	}
}
