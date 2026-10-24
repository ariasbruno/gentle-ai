package uninstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/agents"
	antigravitycli "github.com/gentleman-programming/gentle-ai/v4/internal/agents/antigravitycli"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
)

// statusLineAGYAdapter builds the Antigravity CLI adapter for an injected
// home without retaining operations.
func statusLineAGYAdapter(t *testing.T) *antigravitycli.Adapter {
	t.Helper()
	adapter, err := agents.NewAdapter("antigravity-cli")
	if err != nil {
		t.Fatalf("agents.NewAdapter error = %v", err)
	}
	agy, ok := adapter.(*antigravitycli.Adapter)
	if !ok {
		t.Fatalf("adapter is not *antigravitycli.Adapter")
	}
	return agy
}

// statusLineOps builds the retained Antigravity CLI operations for an
// injected home.
func statusLineOps(t *testing.T, homeDir string) (*antigravitycli.Adapter, []operation) {
	t.Helper()
	agy := statusLineAGYAdapter(t)
	return agy, retainedAntigravityCLIPluginOperations(agy, homeDir)
}

func findOperationByPath(ops []operation, path string) *operation {
	for i := range ops {
		if ops[i].path == path {
			return &ops[i]
		}
	}
	return nil
}

func writeStatusLineSettings(t *testing.T, agy *antigravitycli.Adapter, homeDir, content string) {
	t.Helper()
	settingsPath := agy.SettingsPath(homeDir)
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readStatusLineSettings(t *testing.T, agy *antigravitycli.Adapter, homeDir string) string {
	t.Helper()
	raw, err := os.ReadFile(agy.SettingsPath(homeDir))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	return string(raw)
}

// TestRetainedAntigravityCLIUninstallRemovesManagedStatusLine verifies the
// managed statusLine/title blocks are removed while user keys survive.
func TestRetainedAntigravityCLIUninstallRemovesManagedStatusLine(t *testing.T) {
	homeDir := t.TempDir()
	agy, ops := statusLineOps(t, homeDir)
	ourCmd := "gentle-ai statusline"
	settings := `{
  "allowNonWorkspaceAccess": true,
  "statusLine": {
    "type": "command",
    "command": "` + ourCmd + `",
    "enabled": false
  },
  "title": {
    "type": "command",
    "command": "gentle-ai statusline --title"
  }
}`
	writeStatusLineSettings(t, agy, homeDir, settings)

	op := findOperationByPath(ops, agy.SettingsPath(homeDir))
	if op == nil {
		t.Fatal("retained operations missing the settings.json statusline cleanup")
	}
	changed, removed, err := op.apply(op.path)
	if err != nil {
		t.Fatalf("apply statusline cleanup: %v", err)
	}
	if !changed || removed {
		t.Errorf("apply = changed %v removed %v, want changed true removed false", changed, removed)
	}

	got := readStatusLineSettings(t, agy, homeDir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "statusLine") || strings.Contains(got, "title") {
		t.Errorf("managed blocks survived uninstall:\n%s", got)
	}
	if !strings.Contains(got, "allowNonWorkspaceAccess") {
		t.Errorf("user top-level key was dropped:\n%s", got)
	}
}

// TestRetainedAntigravityCLIUninstallPreservesUserStatusLine verifies a
// user-owned status line (foreign command) is a no-op, byte-identical.
func TestRetainedAntigravityCLIUninstallPreservesUserStatusLine(t *testing.T) {
	homeDir := t.TempDir()
	agy, ops := statusLineOps(t, homeDir)
	original := `{
  "statusLine": {
    "type": "command",
    "command": "/usr/local/bin/user-statusline.sh",
    "padding": 2
  },
  "title": {
    "type": "command",
    "command": "/usr/local/bin/user-title.sh"
  }
}`
	writeStatusLineSettings(t, agy, homeDir, original)

	op := findOperationByPath(ops, agy.SettingsPath(homeDir))
	if op == nil {
		t.Fatal("retained operations missing the settings.json statusline cleanup")
	}
	changed, _, err := op.apply(op.path)
	if err != nil {
		t.Fatalf("apply statusline cleanup: %v", err)
	}
	if changed {
		got := readStatusLineSettings(t, agy, homeDir)
		t.Errorf("user-owned statusline was modified:\n%s", got)
	}
	got := readStatusLineSettings(t, agy, homeDir)
	if got != original {
		t.Errorf("user-owned settings bytes changed:\n%s", got)
	}
}

// TestRetainedAntigravityCLIUninstallMixedOwnership verifies only the managed
// block is removed when the sibling block is user-owned.
func TestRetainedAntigravityCLIUninstallMixedOwnership(t *testing.T) {
	homeDir := t.TempDir()
	agy, ops := statusLineOps(t, homeDir)
	ourCmd := "gentle-ai statusline"
	writeStatusLineSettings(t, agy, homeDir, `{
  "statusLine": {
    "type": "command",
    "command": "`+ourCmd+`"
  },
  "title": {
    "type": "command",
    "command": "/usr/local/bin/user-title.sh"
  }
}`)

	op := findOperationByPath(ops, agy.SettingsPath(homeDir))
	if op == nil {
		t.Fatal("retained operations missing the settings.json statusline cleanup")
	}
	if _, _, err := op.apply(op.path); err != nil {
		t.Fatalf("apply statusline cleanup: %v", err)
	}

	got := readStatusLineSettings(t, agy, homeDir)
	if strings.Contains(got, `"statusLine"`) {
		t.Errorf("managed statusLine survived:\n%s", got)
	}
	if !strings.Contains(got, "/usr/local/bin/user-title.sh") {
		t.Errorf("user-owned title was removed:\n%s", got)
	}
}

// TestRetainedAntigravityCLIUninstallRemovesEmptySettingsFile verifies a
// settings.json holding only the managed blocks is deleted entirely.
func TestRetainedAntigravityCLIUninstallRemovesEmptySettingsFile(t *testing.T) {
	homeDir := t.TempDir()
	agy, ops := statusLineOps(t, homeDir)
	ourCmd := "gentle-ai statusline"
	writeStatusLineSettings(t, agy, homeDir, `{
  "statusLine": {
    "command": "`+ourCmd+`"
  },
  "title": {
    "command": "`+ourCmd+` --title"
  }
}`)

	op := findOperationByPath(ops, agy.SettingsPath(homeDir))
	if op == nil {
		t.Fatal("retained operations missing the settings.json statusline cleanup")
	}
	changed, removed, err := op.apply(op.path)
	if err != nil {
		t.Fatalf("apply statusline cleanup: %v", err)
	}
	if !changed || !removed {
		t.Errorf("apply = changed %v removed %v, want both true for empty result", changed, removed)
	}
	if _, statErr := os.Stat(agy.SettingsPath(homeDir)); !os.IsNotExist(statErr) {
		t.Errorf("empty settings.json was not removed")
	}
}

// TestRetainedAntigravityCLIUninstallMissingSettingsNoop verifies the
// cleanup is a no-op when settings.json does not exist.
func TestRetainedAntigravityCLIUninstallMissingSettingsNoop(t *testing.T) {
	homeDir := t.TempDir()
	agy, ops := statusLineOps(t, homeDir)

	op := findOperationByPath(ops, agy.SettingsPath(homeDir))
	if op == nil {
		t.Fatal("retained operations missing the settings.json statusline cleanup")
	}
	changed, removed, err := op.apply(op.path)
	if err != nil {
		t.Fatalf("apply statusline cleanup: %v", err)
	}
	if changed || removed {
		t.Errorf("apply = changed %v removed %v, want no-op", changed, removed)
	}
}

// TestRetainedAntigravityCLIUninstallMalformedSettingsFails verifies a
// malformed settings.json fails the operation without touching the file.
func TestRetainedAntigravityCLIUninstallMalformedSettingsFails(t *testing.T) {
	homeDir := t.TempDir()
	agy, ops := statusLineOps(t, homeDir)
	malformed := "{ not json"
	writeStatusLineSettings(t, agy, homeDir, malformed)

	op := findOperationByPath(ops, agy.SettingsPath(homeDir))
	if op == nil {
		t.Fatal("retained operations missing the settings.json statusline cleanup")
	}
	if _, _, err := op.apply(op.path); err == nil {
		t.Fatal("apply expected error for malformed settings.json")
	}
	got := readStatusLineSettings(t, agy, homeDir)
	if got != malformed {
		t.Errorf("malformed settings.json was modified:\n%s", got)
	}
}

// TestRetainedAntigravityCLIUninstallRemovesStatuslineScript verifies the
// deployed statusline.sh is part of the retained file removals.
func TestRetainedAntigravityCLIUninstallRemovesStatuslineScript(t *testing.T) {
	homeDir := t.TempDir()
	agy, ops := statusLineOps(t, homeDir)
	script := filepath.Join(agy.GlobalPluginDir(homeDir), "statusline.sh")
	if op := findOperationByPath(ops, script); op == nil {
		t.Errorf("retained operations missing statusline.sh removal (%s)", script)
	}
}

// TestRetainedAntigravityCLIUninstallRemovesLegacyScriptRegistration verifies
// a registration left by an older sync (the deployed script path) is retired
// by uninstall, including its user-owned extras surviving nothing else.
func TestRetainedAntigravityCLIUninstallRemovesLegacyScriptRegistration(t *testing.T) {
	homeDir := t.TempDir()
	agy, ops := statusLineOps(t, homeDir)
	legacyCmd := agy.StatusLineScriptPath(homeDir)
	writeStatusLineSettings(t, agy, homeDir, `{
  "statusLine": {
    "type": "command",
    "command": "`+legacyCmd+`",
    "enabled": false
  }
}`)

	op := findOperationByPath(ops, agy.SettingsPath(homeDir))
	if op == nil {
		t.Fatal("retained operations missing the settings.json statusline cleanup")
	}
	changed, _, err := op.apply(op.path)
	if err != nil {
		t.Fatalf("apply statusline cleanup: %v", err)
	}
	if !changed {
		t.Fatal("legacy script registration was not removed")
	}
	// The settings file held only the managed block: it is removed entirely.
	if _, statErr := os.Stat(agy.SettingsPath(homeDir)); !os.IsNotExist(statErr) {
		t.Errorf("empty settings.json was not removed after retiring the legacy block")
	}
}

// TestPartialUninstallAntigravityCLIThemeRemovesStatusLine verifies that
// uninstalling ComponentTheme for Antigravity CLI cleans up statusLine and
// title blocks while preserving user settings.
func TestPartialUninstallAntigravityCLIThemeRemovesStatusLine(t *testing.T) {
	homeDir := t.TempDir()
	agy := statusLineAGYAdapter(t)
	writeStatusLineSettings(t, agy, homeDir, `{
  "customKey": "keepMe",
  "statusLine": {
    "type": "command",
    "command": "gentle-ai statusline",
    "enabled": true
  },
  "title": {
    "type": "command",
    "command": "gentle-ai statusline --title"
  }
}`)

	svc, err := NewService(homeDir, t.TempDir(), "dev")
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, err := svc.PartialUninstall([]model.AgentID{model.AgentAntigravityCLI}, []model.ComponentID{model.ComponentTheme})
	if err != nil {
		t.Fatalf("PartialUninstall error = %v", err)
	}
	if len(result.RemovedFiles) == 0 && len(result.ChangedFiles) == 0 {
		t.Errorf("PartialUninstall had no effect")
	}

	got := readStatusLineSettings(t, agy, homeDir)
	if strings.Contains(got, "statusLine") || strings.Contains(got, "title") {
		t.Errorf("managed statusLine or title survived partial theme uninstall:\n%s", got)
	}
	if !strings.Contains(got, "customKey") || !strings.Contains(got, "keepMe") {
		t.Errorf("user customKey was lost during partial theme uninstall:\n%s", got)
	}
}

