package uninstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/agents"
	"github.com/gentleman-programming/gentle-ai/v4/internal/components/permissions"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
)

// These cases were exercised by an independent verifier against the pre-fix
// commit. They pin the end-to-end uninstall path through the real service, so a
// future change to the permission rewrite has to keep every one of them true.
func TestUninstallPermissionEdgeCases(t *testing.T) {
	t.Run("missing settings file is a no-op", func(t *testing.T) {
		home, svc, adapter := permissionTestEnv(t, model.AgentAntigravityCLI)
		path := adapter.SettingsPath(home)

		result, err := svc.PartialUninstall([]model.AgentID{model.AgentAntigravityCLI}, []model.ComponentID{model.ComponentPermission})
		if err != nil {
			t.Fatalf("PartialUninstall() error = %v", err)
		}
		if len(result.ChangedFiles) != 0 {
			t.Fatalf("ChangedFiles = %#v, want none", result.ChangedFiles)
		}
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("uninstall created %q; stat err = %v", path, statErr)
		}
	})

	t.Run("a file holding only our entries is removed", func(t *testing.T) {
		for _, agent := range []model.AgentID{model.AgentClaudeCode, model.AgentAntigravityCLI} {
			home, svc, adapter := permissionTestEnv(t, agent)
			path := adapter.SettingsPath(home)
			writeUserSettings(t, path, `{}`)
			if _, err := permissions.Inject(home, adapter); err != nil {
				t.Fatalf("permissions.Inject() error = %v", err)
			}

			if _, err := svc.PartialUninstall([]model.AgentID{agent}, []model.ComponentID{model.ComponentPermission}); err != nil {
				t.Fatalf("PartialUninstall(%q) error = %v", agent, err)
			}
			if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
				t.Fatalf("%q: settings file survived an uninstall that left nothing: stat err = %v", agent, statErr)
			}
		}
	})

	t.Run("unrelated keys and the file mode survive", func(t *testing.T) {
		home, svc, adapter := permissionTestEnv(t, model.AgentAntigravityCLI)
		path := adapter.SettingsPath(home)
		writeUserSettings(t, path, `{"permissions":{"deny":["read_file(notes.md)"]},"unrelated":"keep me"}`)
		if err := os.Chmod(path, 0o640); err != nil {
			t.Fatalf("Chmod() error = %v", err)
		}
		if _, err := permissions.Inject(home, adapter); err != nil {
			t.Fatalf("permissions.Inject() error = %v", err)
		}

		if _, err := svc.PartialUninstall([]model.AgentID{model.AgentAntigravityCLI}, []model.ComponentID{model.ComponentPermission}); err != nil {
			t.Fatalf("PartialUninstall() error = %v", err)
		}

		document := readJSONFileForTest(t, path)
		if document["unrelated"] != "keep me" {
			t.Errorf("unrelated = %#v, want it preserved", document["unrelated"])
		}
		deny, _ := valueAtPath(document, []string{"permissions", "deny"})
		if got, _ := deny.([]any); len(got) != 1 || got[0] != "read_file(notes.md)" {
			t.Errorf("permissions.deny = %#v, want only the user's own rule", deny)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat() error = %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o640 {
			t.Errorf("settings mode = %04o, want 0640", perm)
		}
	})

	t.Run("crlf settings keep crlf and mode", func(t *testing.T) {
		home, svc, adapter := permissionTestEnv(t, model.AgentAntigravityCLI)
		path := adapter.SettingsPath(home)
		// Multi-line CRLF document with no trailing newline: one entry is ours,
		// one belongs to the user.
		withCRLF := "{\r\n  \"permissions\": {\r\n    \"deny\": [\r\n      \"read_file(notes.md)\",\r\n      \"read_file(.env)\"\r\n    ]\r\n  }\r\n}"
		if err := os.WriteFile(path, []byte(withCRLF), 0o640); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		if _, err := svc.PartialUninstall([]model.AgentID{model.AgentAntigravityCLI}, []model.ComponentID{model.ComponentPermission}); err != nil {
			t.Fatalf("PartialUninstall() error = %v", err)
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if !strings.Contains(string(raw), "\r\n") {
			t.Errorf("settings lost their CRLF line endings:\n%q", raw)
		}
		if lone := strings.Count(strings.ReplaceAll(string(raw), "\r\n", ""), "\n"); lone != 0 {
			t.Errorf("settings gained %d bare LF line endings:\n%q", lone, raw)
		}
		document := readJSONFileForTest(t, path)
		deny, _ := valueAtPath(document, []string{"permissions", "deny"})
		if got, _ := deny.([]any); len(got) != 1 || got[0] != "read_file(notes.md)" {
			t.Errorf("permissions.deny = %#v, want the user's own rule only", deny)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat() error = %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o640 {
			t.Errorf("settings mode = %04o, want 0640", perm)
		}
	})

	t.Run("malformed settings fail closed and stay untouched", func(t *testing.T) {
		home, svc, adapter := permissionTestEnv(t, model.AgentAntigravityCLI)
		path := adapter.SettingsPath(home)
		broken := `{"permissions":`
		writeUserSettings(t, path, broken)

		result, err := svc.PartialUninstall([]model.AgentID{model.AgentAntigravityCLI}, []model.ComponentID{model.ComponentPermission})
		if err == nil {
			t.Fatal("PartialUninstall() accepted a malformed settings file")
		}
		if !strings.Contains(err.Error(), "unmarshal json object") {
			t.Errorf("error = %v, want an unmarshal failure", err)
		}
		if len(result.ManualActions) == 0 {
			t.Error("a failed permission cleanup must record a manual action")
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("ReadFile() error = %v", readErr)
		}
		if string(raw) != broken {
			t.Errorf("malformed settings were rewritten:\n%s", raw)
		}
	})

	t.Run("declared list is emptied while undeclared data stays", func(t *testing.T) {
		home, svc, adapter := permissionTestEnv(t, model.AgentAntigravityCLI)
		path := adapter.SettingsPath(home)
		writeUserSettings(t, path, `{"permissions":{"deny":["read_file(notes.md)"]},"keep":1}`)
		if _, err := permissions.Inject(home, adapter); err != nil {
			t.Fatalf("permissions.Inject() error = %v", err)
		}

		first, err := svc.PartialUninstall([]model.AgentID{model.AgentAntigravityCLI}, []model.ComponentID{model.ComponentPermission})
		if err != nil {
			t.Fatalf("PartialUninstall() error = %v", err)
		}
		if len(first.ChangedFiles) == 0 {
			t.Fatal("first uninstall reported no change")
		}

		document := readJSONFileForTest(t, path)
		if document["keep"] != float64(1) {
			t.Errorf("keep = %#v, want 1", document["keep"])
		}
		if _, present := valueAtPath(document, []string{"permissions", "allow"}); present {
			t.Error("permissions.allow survived although every entry in it was ours")
		}

		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		second, err := svc.PartialUninstall([]model.AgentID{model.AgentAntigravityCLI}, []model.ComponentID{model.ComponentPermission})
		if err != nil {
			t.Fatalf("second PartialUninstall() error = %v", err)
		}
		if len(second.ChangedFiles) != 0 {
			t.Errorf("second uninstall changed %#v, want nothing", second.ChangedFiles)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if string(before) != string(after) {
			t.Errorf("second uninstall rewrote the file:\n%s\n---\n%s", before, after)
		}
	})

	// S4 scopes the fix to the agents the AGY work touches. This documents the
	// behavior of the agents it deliberately leaves alone, so widening the scope
	// later is a visible decision rather than an accident. It is not an
	// endorsement: OpenCode still drops the user's own rules with the key.
	t.Run("out of scope agents keep their current behavior", func(t *testing.T) {
		cases := []struct {
			agent   model.AgentID
			user    string
			keyPath []string
			wantKey bool
		}{
			{agent: model.AgentGeminiCLI, user: `{"general":{"defaultApprovalMode":"auto_edit","keep":1}}`, keyPath: []string{"general", "defaultApprovalMode"}, wantKey: false},
			{agent: model.AgentVSCodeCopilot, user: `{"chat.tools.autoApprove":true,"other":1}`, keyPath: []string{"chat.tools.autoApprove"}, wantKey: false},
			{agent: model.AgentOpenCode, user: `{"permission":{"bash":{"*":"ask"}},"other":1}`, keyPath: []string{"permission"}, wantKey: false},
		}
		for _, tc := range cases {
			home, svc, adapter := permissionTestEnv(t, tc.agent)
			path := adapter.SettingsPath(home)
			writeUserSettings(t, path, tc.user)

			if _, err := svc.PartialUninstall([]model.AgentID{tc.agent}, []model.ComponentID{model.ComponentPermission}); err != nil {
				t.Fatalf("PartialUninstall(%q) error = %v", tc.agent, err)
			}
			document := readJSONFileForTest(t, path)
			if _, present := valueAtPath(document, tc.keyPath); present != tc.wantKey {
				t.Errorf("%q: presence of %v = %v, want %v (document %#v)", tc.agent, tc.keyPath, present, tc.wantKey, document)
			}
		}
	})
}

func permissionTestEnv(t *testing.T, agent model.AgentID) (string, *Service, agents.Adapter) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)

	svc, err := NewService(home, t.TempDir(), "dev")
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	svc.snapshotter = stubSnapshotter{}

	adapter, ok := svc.registry.Get(agent)
	if !ok {
		t.Fatalf("registry has no adapter for %q", agent)
	}
	if err := os.MkdirAll(filepath.Dir(adapter.SettingsPath(home)), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	return home, svc, adapter
}

func writeUserSettings(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
