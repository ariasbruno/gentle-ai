package uninstall

import (
	"os"
	"path/filepath"
	"strings"
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
	if err := os.MkdirAll(filepath.Join(homeDir, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	ops := retainedAntigravityCLIPluginOperations(adapter, homeDir)
	pluginDir := agy.GlobalPluginDir(homeDir)
	for _, op := range ops {
		if !strings.HasPrefix(op.path, pluginDir+string(filepath.Separator)) && op.path != pluginDir && op.path != agy.ImportManifestPath(homeDir) && op.path != agy.SettingsPath(homeDir) {
			t.Errorf("global uninstall escaped plugin root: %s", op.path)
		}
	}

	// Static file removals that should always be present
	wanted := []string{
		filepath.Join(pluginDir, "rules", "gentle-ai-orchestrator.md"),
		filepath.Join(pluginDir, "orchestrator.md"),
		filepath.Join(pluginDir, "review-models.json"),
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
