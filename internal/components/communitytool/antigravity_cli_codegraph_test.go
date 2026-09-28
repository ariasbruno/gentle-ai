package communitytool

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v3/internal/agents"
	"github.com/gentleman-programming/gentle-ai/v3/internal/model"
)

func agyPluginMCPConfigPath(t *testing.T, home string) string {
	t.Helper()
	reg, err := agents.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	adapter, ok := reg.Get(model.AgentAntigravityCLI)
	if !ok {
		t.Fatal("antigravity-cli adapter missing from registry")
	}
	return adapter.MCPConfigPath(home, "codegraph")
}

func TestReconcileAntigravityCLICodeGraphCreatesPluginServer(t *testing.T) {
	home := t.TempDir()
	mcpPath := agyPluginMCPConfigPath(t, home)

	res, err := ReconcileAntigravityCLICodeGraphWithAgents(home, []model.AgentID{model.AgentAntigravityCLI})
	if err != nil {
		t.Fatalf("Reconcile error = %v", err)
	}
	if !res.Changed || len(res.Files) != 1 || res.Files[0] != mcpPath {
		t.Fatalf("reconcile result = %+v, want one changed file %q", res, mcpPath)
	}

	var doc map[string]any
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("read %q: %v", mcpPath, err)
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	server, ok := doc["mcpServers"].(map[string]any)["codegraph"].(map[string]any)
	if !ok {
		t.Fatalf("codegraph server missing from %s", data)
	}
	if server["command"] != "codegraph" {
		t.Errorf("codegraph server command = %v, want codegraph", server["command"])
	}
	args, ok := server["args"].([]any)
	if !ok || len(args) != 2 || args[0] != "serve" || args[1] != "--mcp" {
		t.Errorf("codegraph server args = %v, want [serve --mcp]", server["args"])
	}
}

func TestReconcileAntigravityCLICodeGraphIsIdempotent(t *testing.T) {
	home := t.TempDir()
	mcpPath := agyPluginMCPConfigPath(t, home)
	selected := []model.AgentID{model.AgentAntigravityCLI}

	if _, err := ReconcileAntigravityCLICodeGraphWithAgents(home, selected); err != nil {
		t.Fatalf("first reconcile error = %v", err)
	}
	first, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}

	res, err := ReconcileAntigravityCLICodeGraphWithAgents(home, selected)
	if err != nil {
		t.Fatalf("second reconcile error = %v", err)
	}
	if res.Changed {
		t.Fatalf("second reconcile changed an already-wired config")
	}
	second, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("second reconcile rewrote bytes:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestReconcileAntigravityCLICodeGraphPreservesOtherServers(t *testing.T) {
	home := t.TempDir()
	mcpPath := agyPluginMCPConfigPath(t, home)
	if err := os.MkdirAll(filepath.Dir(mcpPath), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := `{
  "mcpServers": {
    "context7": {"command": "npx", "args": ["-y"]},
    "engram": {"command": "/bin/engram", "args": ["mcp"]}
  }
}`
	if err := os.WriteFile(mcpPath, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ReconcileAntigravityCLICodeGraphWithAgents(home, []model.AgentID{model.AgentAntigravityCLI}); err != nil {
		t.Fatalf("reconcile error = %v", err)
	}

	var doc map[string]any
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	servers := doc["mcpServers"].(map[string]any)
	for _, name := range []string{"codegraph", "context7", "engram"} {
		if _, ok := servers[name]; !ok {
			t.Errorf("server %q lost by reconcile: %s", name, data)
		}
	}
}

func TestReconcileAntigravityCLICodeGraphReplacesMalformedConfig(t *testing.T) {
	home := t.TempDir()
	mcpPath := agyPluginMCPConfigPath(t, home)
	if err := os.MkdirAll(filepath.Dir(mcpPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mcpPath, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ReconcileAntigravityCLICodeGraphWithAgents(home, []model.AgentID{model.AgentAntigravityCLI}); err != nil {
		t.Fatalf("reconcile over malformed config error = %v", err)
	}
	var doc map[string]any
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("reconcile left a malformed config: %s", data)
	}
	if _, ok := doc["mcpServers"].(map[string]any)["codegraph"]; !ok {
		t.Fatalf("codegraph server missing after malformed-config recovery: %s", data)
	}
}

func TestNeedsAntigravityCLICodeGraphReconcileDetection(t *testing.T) {
	home := t.TempDir()
	mcpPath := agyPluginMCPConfigPath(t, home)

	if NeedsAntigravityCLICodeGraphReconcileWithAgents(home, []model.AgentID{model.AgentClaudeCode}) {
		t.Fatal("needs reconcile although antigravity-cli is not selected")
	}
	if !NeedsAntigravityCLICodeGraphReconcileWithAgents(home, []model.AgentID{model.AgentAntigravityCLI}) {
		t.Fatal("needs reconcile = false for a selected, unwired antigravity-cli")
	}

	if err := os.MkdirAll(filepath.Dir(mcpPath), 0o755); err != nil {
		t.Fatal(err)
	}
	wired := `{"mcpServers": {"codegraph": {"command": "codegraph", "args": ["serve", "--mcp"]}}}`
	if err := os.WriteFile(mcpPath, []byte(wired), 0o644); err != nil {
		t.Fatal(err)
	}
	if NeedsAntigravityCLICodeGraphReconcileWithAgents(home, []model.AgentID{model.AgentAntigravityCLI}) {
		t.Fatal("needs reconcile = true although the plugin config is already wired")
	}
}

func TestAntigravityCLICodeGraphWiringPathsAndManagedPaths(t *testing.T) {
	home := t.TempDir()
	reg, err := agents.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	adapter, ok := reg.Get(model.AgentAntigravityCLI)
	if !ok {
		t.Fatal("antigravity-cli adapter missing from registry")
	}

	wiring := codeGraphToolWiringPaths(home, adapter)
	mcpPath := adapter.MCPConfigPath(home, "codegraph")
	if len(wiring) != 1 || wiring[0] != mcpPath {
		t.Fatalf("agy wiring paths = %v, want exactly [%q]", wiring, mcpPath)
	}
}
