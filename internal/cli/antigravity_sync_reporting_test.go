package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/agents/antigravitycli"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
)

func TestAntigravitySyncReportsCompleteFinalPluginDelta(t *testing.T) {
	for _, scope := range []InstallScope{ScopeGlobal, ScopeWorkspace} {
		t.Run(string(scope), func(t *testing.T) {
			home, workspace := t.TempDir(), t.TempDir()
			t.Setenv("GENTLE_AI_HOME", filepath.Join(home, ".gentle-ai"))
			t.Chdir(workspace)
			adapter := antigravitycli.NewAdapter()
			root := adapter.GlobalPluginDir(home)
			if scope == ScopeWorkspace {
				root = adapter.WorkspacePluginDir(workspace)
			} else {
				mustWriteFile(t, adapter.ImportManifestPath(home), []byte(`{"imports":[{"name":"gentle-ai","source":"antigravity"}]}`))
			}
			selection := model.Selection{Agents: []model.AgentID{model.AgentAntigravityCLI}, Persona: model.PersonaNeutral,
				Components: []model.ComponentID{model.ComponentPersona, model.ComponentEngram}}
			// Initial creation must report every file actually materialized, including
			// native assets not injected by a selected component.
			before := pluginReportingContents(t, root)
			result, err := RunSyncWithSelectionScope(home, selection, scope)
			if err != nil {
				t.Fatal(err)
			}
			assertPluginReport(t, root, before, result)

			// Simulate the user's thirteen-file upgrade from a prior owned bundle.
			previous := []string{
				"agents/gentle-ai-explore/agent.md", "agents/gentle-ai-verify/agent.md", "agents/gentle-ai-worker/agent.md",
				"rules/AGENTS.md", "rules/gentle-ai-orchestrator.md", "rules/gentle-ai-routing.md",
				"rules/orchestrator-delegation.md", "rules/orchestrator-memory.md", "rules/orchestrator-prompts.md", "rules/orchestrator-verification.md",
				"skills/skill-registry/SKILL.md", "skills/work-unit-commits/SKILL.md",
			}
			ledgerPath := filepath.Join(root, ".native-rule-ownership.json")
			var ledger struct {
				Version int               `json:"version"`
				Rules   map[string]string `json:"rules"`
			}
			if err := json.Unmarshal(mustReadReportingFile(t, ledgerPath), &ledger); err != nil {
				t.Fatal(err)
			}
			for _, rel := range previous {
				path := filepath.Join(root, filepath.FromSlash(rel))
				data := append(mustReadReportingFile(t, path), []byte("\n<!-- previous managed bundle -->\n")...)
				if rel == "rules/AGENTS.md" {
					// AGENTS is section-merged, not overwritten: age a managed
					// Engram rule rather than adding user-owned text outside it.
					data = []byte(strings.Replace(string(mustReadReportingFile(t, path)), "Resolve Project Before Initial Memory Access", "Detect Project Name", 1))
				}
				mustWriteFile(t, path, data)
				if _, owned := ledger.Rules[rel]; owned {
					ledger.Rules[rel] = fmt.Sprintf("%x", sha256.Sum256(data))
				}
			}
			encoded, err := json.Marshal(ledger)
			if err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, ledgerPath, encoded)
			before = pluginReportingContents(t, root)
			result, err = RunSyncWithSelectionScope(home, selection, scope)
			if err != nil {
				t.Fatal(err)
			}
			assertPluginReport(t, root, before, result)
			if result.FilesChanged != 13 {
				t.Fatalf("upgrade reported %d files, want 13: %v", result.FilesChanged, result.ChangedFiles)
			}
			report := RenderSyncReport(result)
			if !strings.Contains(report, "Sync actions executed: 13 files changed\n") {
				t.Fatalf("incorrect public summary: %s", report)
			}
			for _, path := range result.ChangedFiles {
				if strings.Count(report+"\n", "  - "+path+"\n") != 1 {
					t.Fatalf("path missing or duplicated in report: %s", path)
				}
			}

			// A missing embedded resource and a stale pruned skill exercise creation
			// and deletion; the unrelated custom file must remain unchanged.
			if err := os.Remove(filepath.Join(root, "agents", "gentle-ai-worker", "agent.md")); err != nil {
				t.Fatal(err)
			}
			stale := filepath.Join(root, "skills", "obsolete", "SKILL.md")
			mustWriteFile(t, stale, []byte("obsolete managed skill"))
			custom := filepath.Join(root, "custom.txt")
			mustWriteFile(t, custom, []byte("keep user content"))
			before = pluginReportingContents(t, root)
			result, err = RunSyncWithSelectionScope(home, selection, scope)
			if err != nil {
				t.Fatal(err)
			}
			assertPluginReport(t, root, before, result)
			if _, err := os.Stat(stale); !os.IsNotExist(err) {
				t.Fatalf("stale skill not pruned: %v", err)
			}
			if string(mustReadReportingFile(t, custom)) != "keep user content" {
				t.Fatal("custom file changed")
			}

			before = pluginReportingContents(t, root)
			result, err = RunSyncWithSelectionScope(home, selection, scope)
			if err != nil {
				t.Fatal(err)
			}
			assertPluginReport(t, root, before, result)
			if result.FilesChanged != 0 || len(result.ChangedFiles) != 0 {
				t.Fatalf("converged sync reported changes: %#v", result.ChangedFiles)
			}
		})
	}
}

func TestAntigravitySyncRejectsLinkedPluginAncestorsWithoutMutation(t *testing.T) {
	if _, err := ParseSyncFlags([]string{"--help"}); err == nil || !strings.Contains(err.Error(), "Antigravity CLI rejects linked plugin paths below the selected home/workspace root") {
		t.Fatalf("sync help omits plugin path refusal: %v", err)
	}
	for _, tc := range []struct {
		scope    InstallScope
		relative string
	}{
		{ScopeGlobal, ".gemini"}, {ScopeGlobal, ".gemini/config"},
		{ScopeGlobal, ".gemini/config/plugins"}, {ScopeGlobal, ".gemini/config/plugins/gentle-ai"},
		{ScopeWorkspace, ".agents"}, {ScopeWorkspace, ".agents/plugins"},
		{ScopeWorkspace, ".agents/plugins/gentle-ai"},
	} {
		t.Run(string(tc.scope)+"/"+tc.relative, func(t *testing.T) {
			home, workspace, external := t.TempDir(), t.TempDir(), t.TempDir()
			t.Setenv("GENTLE_AI_HOME", filepath.Join(home, ".gentle-ai"))
			t.Chdir(workspace)
			base := home
			if tc.scope == ScopeWorkspace {
				base = workspace
			}
			link := filepath.Join(base, filepath.FromSlash(tc.relative))
			if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(external, link); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			sentinel := filepath.Join(external, "sentinel.json")
			mustWriteFile(t, sentinel, []byte(`{"counter":7,"content":"preserve"}`))
			if err := os.Chmod(sentinel, 0o600); err != nil {
				t.Fatal(err)
			}
			beforeHome, beforeWorkspace := pluginReportingContents(t, home), pluginReportingContents(t, workspace)
			beforeExternal := pluginReportingContents(t, external)
			selection := model.Selection{Agents: []model.AgentID{model.AgentAntigravityCLI}, Persona: model.PersonaNeutral}
			result, err := RunSyncWithSelectionScope(home, selection, tc.scope)
			want := "enumerate pre-sync Antigravity plugin files: refuse linked Antigravity plugin path: " + link
			if err == nil || err.Error() != want {
				t.Errorf("error = %v, want %q", err, want)
			}
			if result.FilesChanged != 0 || len(result.ChangedFiles) != 0 {
				t.Errorf("rejection reported changes: %v", result.ChangedFiles)
			}
			if !reflect.DeepEqual(beforeHome, pluginReportingContents(t, home)) || !reflect.DeepEqual(beforeWorkspace, pluginReportingContents(t, workspace)) || !reflect.DeepEqual(beforeExternal, pluginReportingContents(t, external)) {
				t.Error("rejected linked path changed stored data or counters")
			}
			info, err := os.Stat(sentinel)
			if err != nil || info.Mode().Perm() != 0o600 {
				t.Errorf("sentinel mode changed: %v, %v", info, err)
			}
			if target, err := os.Readlink(link); err != nil || target != external {
				t.Errorf("link changed: %q, %v", target, err)
			}
		})
	}
}

func pluginReportingContents(t *testing.T, root string) map[string]string {
	t.Helper()
	contents := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if os.IsNotExist(err) && path == root {
			return nil
		}
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			contents[path] = string(mustReadReportingFile(t, path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

func mustReadReportingFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertPluginReport(t *testing.T, root string, before map[string]string, result SyncResult) {
	t.Helper()
	after := pluginReportingContents(t, root)
	expectedSet := map[string]bool{}
	for path, old := range before {
		if current, ok := after[path]; !ok || current != old {
			expectedSet[path] = true
		}
	}
	for path, current := range after {
		if old, ok := before[path]; !ok || current != old {
			expectedSet[path] = true
		}
	}
	var expected, actual []string
	for path := range expectedSet {
		expected = append(expected, path)
	}
	for _, path := range result.ChangedFiles {
		if pathUnder(path, root) {
			actual = append(actual, path)
		}
	}
	sort.Strings(expected)
	sort.Strings(actual)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("plugin delta omitted or overreported files:\nexpected %v\nactual %v", expected, actual)
	}
	if result.FilesChanged != len(result.ChangedFiles) {
		t.Fatalf("count %d differs from paths %d", result.FilesChanged, len(result.ChangedFiles))
	}
}
