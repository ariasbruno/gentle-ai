package antigravitycli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeAGYSettings writes a raw settings.json under the injected home's
// Antigravity CLI config directory.
func writeAGYSettings(t *testing.T, homeDir, content string) {
	t.Helper()
	adapter := NewAdapter()
	path := adapter.SettingsPath(homeDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readAGYSettings(t *testing.T, homeDir string) (string, []byte) {
	t.Helper()
	adapter := NewAdapter()
	raw, err := os.ReadFile(adapter.SettingsPath(homeDir))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	return string(raw), raw
}

func decodeAGYSettings(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	root := map[string]any{}
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	return root
}

// TestEnsureStatusLineSettings_FillsEmptyStub verifies the real-world empty
// stub ({"type":"","command":"","enabled":true}) gets type+command managed
// while enabled is preserved verbatim, and the title block is registered.
func TestEnsureStatusLineSettings_FillsEmptyStub(t *testing.T) {
	homeDir := t.TempDir()
	adapter := NewAdapter()
	writeAGYSettings(t, homeDir, `{
  "allowNonWorkspaceAccess": true,
  "statusLine": {
    "type": "",
    "command": "",
    "enabled": true
  }
}`)

	if err := adapter.EnsureStatusLineSettings(homeDir); err != nil {
		t.Fatalf("EnsureStatusLineSettings() error = %v", err)
	}

	_, raw := readAGYSettings(t, homeDir)
	root := decodeAGYSettings(t, raw)

	statusLine, ok := root["statusLine"].(map[string]any)
	if !ok {
		t.Fatalf("statusLine block missing or not an object: %s", raw)
	}
	wantCmd := adapter.StatusLineScriptPath(homeDir)
	if statusLine["type"] != "command" {
		t.Errorf("statusLine.type = %v, want command", statusLine["type"])
	}
	if statusLine["command"] != wantCmd {
		t.Errorf("statusLine.command = %v, want %s", statusLine["command"], wantCmd)
	}
	if statusLine["enabled"] != true {
		t.Errorf("statusLine.enabled = %v, want preserved true", statusLine["enabled"])
	}
	if root["allowNonWorkspaceAccess"] != true {
		t.Errorf("unknown top-level key not preserved: %s", raw)
	}

	title, ok := root["title"].(map[string]any)
	if !ok {
		t.Fatalf("title block missing or not an object: %s", raw)
	}
	if title["type"] != "command" {
		t.Errorf("title.type = %v, want command", title["type"])
	}
	if title["command"] != wantCmd+" --title" {
		t.Errorf("title.command = %v, want %s --title", title["command"], wantCmd)
	}
}

// TestEnsureStatusLineSettings_PreservesForeignBlocks verifies user-owned
// blocks (non-empty foreign command) are left completely untouched and the
// file is not rewritten when nothing changes.
func TestEnsureStatusLineSettings_PreservesForeignBlocks(t *testing.T) {
	homeDir := t.TempDir()
	adapter := NewAdapter()
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
	writeAGYSettings(t, homeDir, original)

	if err := adapter.EnsureStatusLineSettings(homeDir); err != nil {
		t.Fatalf("EnsureStatusLineSettings() error = %v", err)
	}

	got, _ := readAGYSettings(t, homeDir)
	if got != original {
		t.Errorf("user-owned blocks were modified:\n%s", got)
	}
}

// TestEnsureStatusLineSettings_ReownsOursAndPreservesUserKnobs verifies we
// re-own type+command when the command already points at our script while
// user-owned knobs (enabled/padding/stack_with_default) are preserved.
func TestEnsureStatusLineSettings_ReownsOursAndPreservesUserKnobs(t *testing.T) {
	homeDir := t.TempDir()
	adapter := NewAdapter()
	ourCmd := adapter.StatusLineScriptPath(homeDir)
	settings := `{
  "statusLine": {
    "type": "broken",
    "command": "` + ourCmd + `",
    "enabled": false,
    "padding": 2,
    "stack_with_default": true
  }
}`
	writeAGYSettings(t, homeDir, settings)

	if err := adapter.EnsureStatusLineSettings(homeDir); err != nil {
		t.Fatalf("EnsureStatusLineSettings() error = %v", err)
	}

	_, raw := readAGYSettings(t, homeDir)
	root := decodeAGYSettings(t, raw)
	statusLine, ok := root["statusLine"].(map[string]any)
	if !ok {
		t.Fatalf("statusLine block missing: %s", raw)
	}
	if statusLine["type"] != "command" {
		t.Errorf("statusLine.type = %v, want command", statusLine["type"])
	}
	if statusLine["command"] != ourCmd {
		t.Errorf("statusLine.command = %v, want %s", statusLine["command"], ourCmd)
	}
	if statusLine["enabled"] != false {
		t.Errorf("statusLine.enabled = %v, want preserved false", statusLine["enabled"])
	}
	if statusLine["padding"] != float64(2) {
		t.Errorf("statusLine.padding = %v, want preserved 2", statusLine["padding"])
	}
	if statusLine["stack_with_default"] != true {
		t.Errorf("statusLine.stack_with_default = %v, want preserved true", statusLine["stack_with_default"])
	}
}

// TestEnsureStatusLineSettings_CreatesMissingFile verifies registration on a
// fresh machine without an existing settings.json.
func TestEnsureStatusLineSettings_CreatesMissingFile(t *testing.T) {
	homeDir := t.TempDir()
	adapter := NewAdapter()

	if err := adapter.EnsureStatusLineSettings(homeDir); err != nil {
		t.Fatalf("EnsureStatusLineSettings() error = %v", err)
	}

	_, raw := readAGYSettings(t, homeDir)
	root := decodeAGYSettings(t, raw)
	statusLine, ok := root["statusLine"].(map[string]any)
	if !ok || statusLine["command"] != adapter.StatusLineScriptPath(homeDir) {
		t.Errorf("statusLine block not registered on fresh settings: %s", raw)
	}
	title, ok := root["title"].(map[string]any)
	if !ok || title["command"] != adapter.StatusLineScriptPath(homeDir)+" --title" {
		t.Errorf("title block not registered on fresh settings: %s", raw)
	}
}

// TestEnsureStatusLineSettings_MalformedFileFailsClosed verifies a malformed
// settings.json returns an error and leaves the file byte-identical.
func TestEnsureStatusLineSettings_MalformedFileFailsClosed(t *testing.T) {
	homeDir := t.TempDir()
	adapter := NewAdapter()
	malformed := "{ not json"
	writeAGYSettings(t, homeDir, malformed)

	if err := adapter.EnsureStatusLineSettings(homeDir); err == nil {
		t.Fatal("EnsureStatusLineSettings() expected error for malformed settings.json")
	}

	got, _ := readAGYSettings(t, homeDir)
	if got != malformed {
		t.Errorf("malformed settings.json was modified on failure:\n%s", got)
	}
}

// TestEnsureStatusLineSettings_NonObjectBlockLeftUntouched verifies a
// non-object block value (user-owned shape we do not understand) is never
// modified or replaced.
func TestEnsureStatusLineSettings_NonObjectBlockLeftUntouched(t *testing.T) {
	homeDir := t.TempDir()
	adapter := NewAdapter()
	writeAGYSettings(t, homeDir, `{
  "statusLine": "user-scalar"
}`)

	if err := adapter.EnsureStatusLineSettings(homeDir); err != nil {
		t.Fatalf("EnsureStatusLineSettings() error = %v", err)
	}

	_, raw := readAGYSettings(t, homeDir)
	root := decodeAGYSettings(t, raw)
	if root["statusLine"] != "user-scalar" {
		t.Errorf("non-object statusLine was modified: %s", raw)
	}
}

// TestEnsureStatusLineSettings_NoChangeKeepsBytes verifies an already managed
// configuration is not rewritten: bytes stay identical.
func TestEnsureStatusLineSettings_NoChangeKeepsBytes(t *testing.T) {
	homeDir := t.TempDir()
	adapter := NewAdapter()
	ourCmd := adapter.StatusLineScriptPath(homeDir)
	settings := `{
  "statusLine": {
    "command": "` + ourCmd + `",
    "type": "command",
    "enabled": true
  },
  "title": {
    "type": "command",
    "command": "` + ourCmd + ` --title"
  }
}`
	writeAGYSettings(t, homeDir, settings)

	if err := adapter.EnsureStatusLineSettings(homeDir); err != nil {
		t.Fatalf("EnsureStatusLineSettings() error = %v", err)
	}

	got, _ := readAGYSettings(t, homeDir)
	if got != settings {
		t.Errorf("managed settings were rewritten without a change:\n%s", got)
	}
}

// TestEnsureStatusLineSettings_TitleManagedIndependently verifies per-block
// ownership: a user-owned statusLine does not stop the managed title block
// from being registered.
func TestEnsureStatusLineSettings_TitleManagedIndependently(t *testing.T) {
	homeDir := t.TempDir()
	adapter := NewAdapter()
	writeAGYSettings(t, homeDir, `{
  "statusLine": {
    "type": "command",
    "command": "/usr/local/bin/user-statusline.sh"
  }
}`)

	if err := adapter.EnsureStatusLineSettings(homeDir); err != nil {
		t.Fatalf("EnsureStatusLineSettings() error = %v", err)
	}

	_, raw := readAGYSettings(t, homeDir)
	root := decodeAGYSettings(t, raw)
	statusLine, ok := root["statusLine"].(map[string]any)
	if !ok || statusLine["command"] != "/usr/local/bin/user-statusline.sh" {
		t.Errorf("user-owned statusLine was touched: %s", raw)
	}
	title, ok := root["title"].(map[string]any)
	if !ok {
		t.Fatalf("managed title block missing: %s", raw)
	}
	if title["command"] != adapter.StatusLineScriptPath(homeDir)+" --title" {
		t.Errorf("title.command = %v, want managed path", title["command"])
	}
}
