package uninstall

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/components/filemerge"
	"github.com/gentleman-programming/gentle-ai/v4/internal/components/permissions"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
)

// An uninstall removes the permission entries gentle-ai injected and nothing
// else. Claude Code used to lose its entire `permissions` object, taking the
// user's own rules and default mode with it, and Antigravity CLI had no
// permission case at all, so its injected overlay survived the uninstall.
func TestUninstallPermissionSubtractsOnlyManagedEntries(t *testing.T) {
	cases := []struct {
		name  string
		agent model.AgentID
		// userSettings is the file as it looked before install: exactly the
		// content gentle-ai does not own.
		userSettings string
		// want contains the assertions run after the uninstall, expressed as
		// value expectations per path. A nil value means the path must be gone.
		want map[string]any
	}{
		{
			name:  "claude-code",
			agent: model.AgentClaudeCode,
			userSettings: `{
  "model": "opus",
  "permissions": {
    "defaultMode": "acceptEdits",
    "deny": ["Bash(docker rm)"],
    "allow": ["Bash(ls)"]
  }
}`,
			want: map[string]any{
				"model":                   "opus",
				"permissions.defaultMode": nil,
				"permissions.allow":       []any{"Bash(ls)"},
				"permissions.deny":        []any{"Bash(docker rm)"},
			},
		},
		{
			name:  "antigravity-cli",
			agent: model.AgentAntigravityCLI,
			userSettings: `{
  "custom": true,
  "permissions": {
    "deny": ["read_file(notes.md)"]
  }
}`,
			want: map[string]any{
				"custom":            true,
				"permissions.deny":  []any{"read_file(notes.md)"},
				"permissions.allow": nil,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			svc, err := NewService(home, t.TempDir(), "dev")
			if err != nil {
				t.Fatalf("NewService() error = %v", err)
			}
			svc.snapshotter = stubSnapshotter{}

			adapter, ok := svc.registry.Get(tc.agent)
			if !ok {
				t.Fatalf("registry has no adapter for %q", tc.agent)
			}
			settingsPath := adapter.SettingsPath(home)
			if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}

			overlay := permissions.ManagedOverlay(tc.agent, home)
			if len(overlay) == 0 {
				t.Fatalf("permissions.ManagedOverlay(%q) returned no overlay", tc.agent)
			}
			// Reproduce the installed state through the same merge install uses.
			installed, err := filemerge.MergeJSONObjects([]byte(tc.userSettings), overlay)
			if err != nil {
				t.Fatalf("MergeJSONObjects() error = %v", err)
			}
			if err := os.WriteFile(settingsPath, installed, 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			ops, targets, err := svc.componentOperations(adapter, model.ComponentPermission)
			if err != nil {
				t.Fatalf("componentOperations() error = %v", err)
			}
			if !containsString(targets, settingsPath) {
				t.Fatalf("targets = %#v, want %q", targets, settingsPath)
			}
			if len(ops) == 0 {
				t.Fatalf("componentOperations(%q, permission) queued no cleanup op", tc.agent)
			}

			// The injected entries really are present before the uninstall, so
			// a passing assertion below cannot come from an empty file.
			before := readJSONFileForTest(t, settingsPath)
			if injected := managedEntryCount(tc.agent, overlay, before); injected == 0 {
				t.Fatalf("installed settings carry none of the injected entries: %#v", before)
			}

			runPermissionOps(t, ops)

			after := readJSONFileForTest(t, settingsPath)
			for path, want := range tc.want {
				got, present := valueAtPath(after, strings.Split(path, "."))
				if want == nil {
					if present {
						t.Errorf("%s survived the uninstall: %#v", path, got)
					}
					continue
				}
				if !present {
					t.Errorf("%s is missing after the uninstall, want %#v", path, want)
					continue
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("%s = %#v, want %#v", path, got, want)
				}
			}

			info, err := os.Stat(settingsPath)
			if err != nil {
				t.Fatalf("Stat() after uninstall error = %v", err)
			}
			if perm := info.Mode().Perm(); perm != 0o600 {
				t.Errorf("settings mode = %04o, want 0600 (an uninstall must not widen it)", perm)
			}

			// Idempotence: a second uninstall finds nothing left to remove and
			// must not rewrite the file.
			firstCleanup, err := os.ReadFile(settingsPath)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			changed := runPermissionOps(t, ops)
			if changed {
				t.Error("second uninstall reported a change")
			}
			secondCleanup, err := os.ReadFile(settingsPath)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			if string(firstCleanup) != string(secondCleanup) {
				t.Errorf("second uninstall rewrote the file:\n%s\n---\n%s", firstCleanup, secondCleanup)
			}
		})
	}
}

func runPermissionOps(t *testing.T, ops []operation) bool {
	t.Helper()
	changedAny := false
	for _, op := range ops {
		changed, _, err := op.apply(op.path)
		if err != nil {
			t.Fatalf("operation %v on %q error = %v", op.typeID, op.path, err)
		}
		changedAny = changedAny || changed
	}
	return changedAny
}

// managedEntryCount counts how many entries of the injected overlay are present
// in the document, so the test proves the uninstall had something to remove.
func managedEntryCount(agent model.AgentID, overlay []byte, document map[string]any) int {
	owned := map[string]any{}
	if err := json.Unmarshal(overlay, &owned); err != nil {
		return 0
	}
	count := 0
	forEachOverlayValue(owned, nil, func(path []string, value any) {
		found, present := valueAtPath(document, path)
		if !present {
			return
		}
		switch want := value.(type) {
		case []any:
			list, isList := found.([]any)
			if !isList {
				return
			}
			for _, item := range want {
				for _, existing := range list {
					if reflect.DeepEqual(item, existing) {
						count++
					}
				}
			}
		default:
			if reflect.DeepEqual(found, value) {
				count++
			}
		}
	})
	return count
}

func forEachOverlayValue(object map[string]any, prefix []string, visit func(path []string, value any)) {
	for key, value := range object {
		path := append(append([]string{}, prefix...), key)
		switch nested := value.(type) {
		case map[string]any:
			forEachOverlayValue(nested, path, visit)
		default:
			visit(path, value)
		}
	}
}

func valueAtPath(document map[string]any, path []string) (any, bool) {
	var current any = document
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func containsString(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}
