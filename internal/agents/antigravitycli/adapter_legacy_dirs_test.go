package antigravitycli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDeployPluginTreePrunesEmptyLegacyDirectory covers the tombstone a legacy
// prune leaves behind: removing the last file of a directory leaves the
// directory empty, and an empty directory is invisible to every other check.
func TestDeployPluginTreePrunesEmptyLegacyDirectory(t *testing.T) {
	home := t.TempDir()
	a := NewAdapter()
	pluginDir := a.PluginDir(home)
	legacyFile := filepath.Join(pluginDir, "support", "sdd-status-contract.md")
	if err := os.MkdirAll(filepath.Dir(legacyFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyFile, []byte("legacy contract"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := a.DeployPluginTree(home); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(legacyFile); !os.IsNotExist(err) {
		t.Fatalf("legacy file %q survived the deploy: %v", legacyFile, err)
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "support")); !os.IsNotExist(err) {
		t.Fatalf("empty legacy directory %q survived the deploy: %v", filepath.Join(pluginDir, "support"), err)
	}
}

// TestDeployPluginTreeKeepsNonEmptyLegacyDirectory pins the guard: the prune
// removes a legacy directory only when it is empty, never its contents.
func TestDeployPluginTreeKeepsNonEmptyLegacyDirectory(t *testing.T) {
	home := t.TempDir()
	a := NewAdapter()
	pluginDir := a.PluginDir(home)
	unknownFile := filepath.Join(pluginDir, "support", "operator-note.md")
	if err := os.MkdirAll(filepath.Dir(unknownFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unknownFile, []byte("not ours"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := a.DeployPluginTree(home); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(unknownFile); err != nil {
		t.Fatalf("unknown file %q must survive the deploy: %v", unknownFile, err)
	}
	if info, err := os.Stat(filepath.Join(pluginDir, "support")); err != nil || !info.IsDir() {
		t.Fatalf("non-empty legacy directory must survive the deploy: %v", err)
	}
}
