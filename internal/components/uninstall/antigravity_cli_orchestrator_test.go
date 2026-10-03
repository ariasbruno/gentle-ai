package uninstall

import (
	"path/filepath"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/agents"
	"github.com/gentleman-programming/gentle-ai/v4/internal/agents/antigravitycli"
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

	// Static file removals that should always be present
	wanted := []string{
		filepath.Join(pluginDir, "rules", "gentle-ai-orchestrator.md"),
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

	// Routing parts are now removed dynamically by removeAntigravityCLIRoutingParts
	// which scans the rules directory for gentle-ai-routing*.md files with our marker.
	// Verify that operation exists.
	foundRoutingCleanup := false
	for _, op := range ops {
		if op.path == filepath.Join(pluginDir, "rules") && op.typeID == opRemoveFile {
			foundRoutingCleanup = true
			break
		}
	}
	if !foundRoutingCleanup {
		t.Errorf("uninstall operations missing dynamic routing parts cleanup for %s", filepath.Join(pluginDir, "rules"))
	}
}
