package statusline

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfigDefaults pins the defaults for a missing config file.
func TestLoadConfigDefaults(t *testing.T) {
	home := t.TempDir()
	config := LoadConfig(home)
	if config.Density != DensityComfortable {
		t.Errorf("default density = %q", config.Density)
	}
	if !config.Visibility.Changes {
		t.Error("changes defaults to visible")
	}
	if !config.Visibility.Tasklist || !config.Visibility.Engram {
		t.Error("tasklist/engram default to visible")
	}
	if !config.Visibility.RDD || !config.Visibility.Agents || !config.Visibility.UsageCost ||
		!config.Visibility.ModelDetails || !config.Visibility.ContextWarning || !config.Visibility.Shortcuts {
		t.Errorf("visibility flags not all visible by default: %+v", config.Visibility)
	}
}

// TestLoadConfigMalformedFallsBack pins that a broken config degrades to the
// defaults instead of failing the render.
func TestLoadConfigMalformedFallsBack(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".gentle-ai")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "statusline.json"), []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	config := LoadConfig(home)
	if config.Density != DensityComfortable || !config.Visibility.RDD {
		t.Errorf("malformed config did not fall back to defaults: %+v", config)
	}
}

// TestLoadConfigValues pins that every documented key is honored.
func TestLoadConfigValues(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".gentle-ai")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{
		"schema": "gentle-ai.statusline/v1",
		"density": "minimal",
		"visibility": {
			"changes": false,
			"rdd": false,
			"agents": false,
			"usageCost": false,
			"modelDetails": false,
			"contextWarning": false,
			"shortcuts": false
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "statusline.json"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	config := LoadConfig(home)
	if config.Density != DensityMinimal {
		t.Errorf("density = %q, want minimal", config.Density)
	}
	if config.Visibility.Changes {
		t.Error("changes=false not honored")
	}
	if config.Visibility.RDD || config.Visibility.Agents || config.Visibility.UsageCost ||
		config.Visibility.ModelDetails || config.Visibility.ContextWarning || config.Visibility.Shortcuts {
		t.Errorf("visibility not honored: %+v", config.Visibility)
	}
}

// TestLoadConfigPartialVisibility pins that absent keys keep their defaults.
func TestLoadConfigPartialVisibility(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".gentle-ai")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{"density": "compact", "visibility": {"rdd": false}}`
	if err := os.WriteFile(filepath.Join(dir, "statusline.json"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	config := LoadConfig(home)
	if config.Density != DensityCompact {
		t.Errorf("density = %q, want compact", config.Density)
	}
	if config.Visibility.RDD {
		t.Error("rdd=false not honored")
	}
	if !config.Visibility.ModelDetails || !config.Visibility.Changes {
		t.Errorf("absent keys did not keep defaults: %+v", config.Visibility)
	}
}
