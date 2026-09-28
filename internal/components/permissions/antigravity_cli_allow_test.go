package permissions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v3/internal/agents/antigravitycli"
)

func TestInjectAntigravityCLICarriesCodeGraphInitAndReviewAllowEntries(t *testing.T) {
	home := t.TempDir()
	adapter := antigravitycli.NewAdapter()
	settingsPath := adapter.SettingsPath(home)
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// A user who already approved other commands must never lose them.
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

	if _, err := Inject(home, adapter); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings file: %v", err)
	}
	var settings struct {
		Permissions struct {
			Allow []string `json:"allow"`
			Deny  []string `json:"deny"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(content, &settings); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	got := map[string]bool{}
	for _, entry := range settings.Permissions.Allow {
		got[entry] = true
	}
	for _, want := range []string{
		"command(gentle-ai codegraph init)",
		"command(gentle-ai review assess)",
		"command(gentle-ai review status)",
		"command(git status)",
	} {
		if !got[want] {
			t.Errorf("allow list missing %q; got %v", want, settings.Permissions.Allow)
		}
	}
	if len(settings.Permissions.Deny) == 0 {
		t.Errorf("deny list lost its security overlay entries")
	}
}

func TestInjectAntigravityCLIAllowUnionIsIdempotent(t *testing.T) {
	home := t.TempDir()
	adapter := antigravitycli.NewAdapter()
	if _, err := Inject(home, adapter); err != nil {
		t.Fatalf("first Inject() error = %v", err)
	}
	first, err := os.ReadFile(adapter.SettingsPath(home))
	if err != nil {
		t.Fatalf("read after first: %v", err)
	}
	if _, err := Inject(home, adapter); err != nil {
		t.Fatalf("second Inject() error = %v", err)
	}
	second, err := os.ReadFile(adapter.SettingsPath(home))
	if err != nil {
		t.Fatalf("read after second: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("second Inject changed the allow overlay:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}
