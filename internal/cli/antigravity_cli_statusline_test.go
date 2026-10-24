package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/agents/antigravitycli"
	"github.com/gentleman-programming/gentle-ai/v4/internal/system"
)

func TestAntigravityCLIInstallStatusLineOptional(t *testing.T) {
	home := installTestHome(t)
	agy := antigravitycli.NewAdapter()
	settingsPath := agy.SettingsPath(home)

	// Default install should NOT configure statusLine (it is optional visual polish/theme)
	if _, err := RunInstall([]string{"--agent", "antigravity-cli"}, system.DetectionResult{}); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	raw, err := os.ReadFile(settingsPath)
	if err == nil && strings.Contains(string(raw), "statusLine") {
		t.Errorf("default install should not register statusLine in settings.json, got:\n%s", raw)
	}
}

func TestAntigravityCLIInstallStatusLineWithIncludeTheme(t *testing.T) {
	home := installTestHome(t)
	agy := antigravitycli.NewAdapter()
	settingsPath := agy.SettingsPath(home)

	// Install with --include-theme MUST configure statusLine
	if _, err := RunInstall([]string{"--agent", "antigravity-cli", "--include-theme"}, system.DetectionResult{}); err != nil {
		t.Fatalf("RunInstall(--include-theme) error = %v", err)
	}

	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("expected settings.json at %s: %v", settingsPath, err)
	}
	content := string(raw)
	if !strings.Contains(content, "statusLine") || !strings.Contains(content, antigravitycli.ManagedStatusLineCommand) {
		t.Errorf("expected managed statusLine command in settings.json, got:\n%s", content)
	}
	if !strings.Contains(content, "title") || !strings.Contains(content, antigravitycli.ManagedTitleCommand) {
		t.Errorf("expected managed title command in settings.json, got:\n%s", content)
	}
}

func TestAntigravityCLIInstallStatusLineWithComponentTheme(t *testing.T) {
	home := installTestHome(t)
	agy := antigravitycli.NewAdapter()
	settingsPath := agy.SettingsPath(home)

	// Explicit component selection including "theme"
	if _, err := RunInstall([]string{"--agent", "antigravity-cli", "--component", "theme"}, system.DetectionResult{}); err != nil {
		t.Fatalf("RunInstall(--component theme) error = %v", err)
	}

	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("expected settings.json at %s: %v", settingsPath, err)
	}
	content := string(raw)
	if !strings.Contains(content, "statusLine") || !strings.Contains(content, antigravitycli.ManagedStatusLineCommand) {
		t.Errorf("expected managed statusLine command in settings.json, got:\n%s", content)
	}
}

func TestAntigravityCLISyncStatusLineWithIncludeTheme(t *testing.T) {
	home := installTestHome(t)
	agy := antigravitycli.NewAdapter()
	settingsPath := agy.SettingsPath(home)

	// Initial install without theme
	if _, err := RunInstall([]string{"--agent", "antigravity-cli"}, system.DetectionResult{}); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}

	raw, err := os.ReadFile(settingsPath)
	if err == nil && strings.Contains(string(raw), "statusLine") {
		t.Fatalf("statusLine unexpectedly present after initial install without theme")
	}

	// Sync with --include-theme configures statusLine
	if _, err := RunSync([]string{"--include-theme"}); err != nil {
		t.Fatalf("RunSync(--include-theme) error = %v", err)
	}

	raw, err = os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("expected settings.json after sync: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "statusLine") || !strings.Contains(content, antigravitycli.ManagedStatusLineCommand) {
		t.Errorf("sync with --include-theme did not configure statusLine, got:\n%s", content)
	}
}

func TestAntigravityCLIInstallStatusLineWorkspaceScopeRejected(t *testing.T) {
	installTestHome(t)

	_, err := RunInstall([]string{"--scope", "workspace", "--agent", "antigravity-cli", "--include-theme"}, system.DetectionResult{})
	if err == nil {
		t.Fatal("expected error installing theme with workspace scope for antigravity-cli, got nil")
	}
	if !strings.Contains(err.Error(), "scope global") {
		t.Errorf("error = %v, want error naming 'scope global'", err)
	}
}
