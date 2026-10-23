package permissions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/agents/antigravitycli"
)

// Antigravity CLI security guards are handled dynamically in-process by the
// PreToolUse hook (RunAntigravityCLIHook / evaluateAntigravityCLIToolCall).
// settings.json must remain clean and unmutated so it can be enabled/disabled
// via plugin management without leaving orphaned settings or breaking the sandbox.
func TestInjectAntigravityCLIDoesNotMutateSettings(t *testing.T) {
	home := t.TempDir()
	adapter := antigravitycli.NewAdapter()
	settingsPath := adapter.SettingsPath(home)
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	existing := `{
  "permissions": {
    "allow": [
      "command(git status)"
    ]
  }
}`
	if err := os.WriteFile(settingsPath, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	res, err := Inject(home, adapter)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if res.Changed {
		t.Errorf("Inject() result.Changed = true, want false (no static overlay for Antigravity CLI)")
	}

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings file: %v", err)
	}
	if string(content) != existing {
		t.Errorf("settings file was mutated; got:\n%s\nwant:\n%s", string(content), existing)
	}
}
