package antigravitycli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gentleman-programming/gentle-ai/v4/internal/assets"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
	"github.com/gentleman-programming/gentle-ai/v4/internal/system"
	"gopkg.in/yaml.v3"
)

func TestDeployPluginPhaseReferences(t *testing.T) {
	home := t.TempDir()
	a := NewAdapter()
	if err := a.DeployPluginTree(home); err != nil {
		t.Fatal(err)
	}
	const pluginRoot = "~/.gemini/config/plugins/gentle-ai/"
	for _, role := range []string{"gentle-ai-worker", "gentle-ai-verify"} {
		t.Run(role, func(t *testing.T) {
			relative := "agents/" + role + "/agent.md"
			data, err := os.ReadFile(filepath.Join(a.PluginDir(home), relative))
			if err != nil {
				t.Fatal(err)
			}
			source, err := assets.Read("antigravitycli/" + relative)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != source {
				t.Fatal("role body changed at deployment; dynamic system_prompt must use the same role instructions")
			}
			module := "strict-tdd.md"
			if role == "gentle-ai-verify" {
				module = "strict-tdd-verify.md"
			}
			if !strings.Contains(string(data), "`"+pluginRoot+"rules/"+module+"`") {
				t.Fatalf("deployed role must point directly to native rules/%s", module)
			}
			rule, err := os.ReadFile(filepath.Join(a.PluginDir(home), "rules", module))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(string(rule), "---\ntrigger: model_decision\n") {
				t.Fatal("referenced TDD module is not a native conditional rule")
			}
		})
	}
	data, err := os.ReadFile(filepath.Join(a.PluginDir(home), "rules", "gentle-ai-orchestrator.md"))
	if err != nil {
		t.Fatal(err)
	}
	// Bare filenames in the parent resolve beside the deployed core, not at
	// the plugin root. Exercise every phase module, not just source text.
	for module := range nativeRuleDescriptions {
		if module == "orchestrator-delegation-reference.md" {
			// The delegation reference is a lazy sub-reference, not a core
			// phase: the core routes to the delegation card, and the card
			// names the reference. Assert that reachable chain.
			card, err := os.ReadFile(filepath.Join(a.PluginDir(home), "rules", "orchestrator-delegation.md"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(card), "`"+module+"`") {
				t.Errorf("delegation card lacks explicit reference route to %s", module)
			}
			if _, err := os.Stat(filepath.Join(a.PluginDir(home), "rules", module)); err != nil {
				t.Errorf("broken deployed reference route %s: %v", module, err)
			}
			continue
		}
		if !strings.Contains(string(data), "`"+module+"`") {
			t.Errorf("deployed core lacks explicit phase route to %s", module)
		}
		if _, err := os.Stat(filepath.Join(a.PluginDir(home), "rules", module)); err != nil {
			t.Errorf("broken deployed phase route %s: %v", module, err)
		}
	}
}

func TestDetect(t *testing.T) {
	tests := []struct {
		name            string
		lookPathPath    string
		lookPathErr     error
		stat            statResult
		wantInstalled   bool
		wantBinaryPath  string
		wantConfigFound bool
		wantErr         bool
	}{
		{
			name:            "binary and config directory found",
			lookPathPath:    "/usr/local/bin/agy",
			stat:            statResult{isDir: true},
			wantInstalled:   true,
			wantBinaryPath:  "/usr/local/bin/agy",
			wantConfigFound: true,
		},
		{
			name:            "binary missing config missing",
			lookPathErr:     errors.New("missing"),
			stat:            statResult{err: os.ErrNotExist},
			wantInstalled:   false,
			wantBinaryPath:  "",
			wantConfigFound: false,
		},
		{
			name:            "binary found config missing",
			lookPathPath:    "/usr/local/bin/agy",
			stat:            statResult{err: os.ErrNotExist},
			wantInstalled:   true,
			wantBinaryPath:  "/usr/local/bin/agy",
			wantConfigFound: false,
		},
		{
			name:            "binary missing config found",
			lookPathErr:     errors.New("missing"),
			stat:            statResult{isDir: true},
			wantInstalled:   false,
			wantBinaryPath:  "",
			wantConfigFound: true,
		},
		{
			name:    "stat error propagates",
			stat:    statResult{err: errors.New("permission denied")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Adapter{
				lookPath: func(string) (string, error) {
					return tt.lookPathPath, tt.lookPathErr
				},
				statPath: func(string) statResult {
					return tt.stat
				},
			}
			homeDir := filepath.Join(string(filepath.Separator), "home", "test")

			installed, binaryPath, configPath, configFound, err := a.Detect(context.Background(), homeDir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Detect() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if installed != tt.wantInstalled {
				t.Fatalf("Detect() installed = %v, want %v", installed, tt.wantInstalled)
			}

			if binaryPath != tt.wantBinaryPath {
				t.Fatalf("Detect() binaryPath = %q, want %q", binaryPath, tt.wantBinaryPath)
			}

			wantConfigPath := filepath.Join(homeDir, ".gemini", "antigravity-cli")
			if configPath != wantConfigPath {
				t.Fatalf("Detect() configPath = %q, want %q", configPath, wantConfigPath)
			}

			if configFound != tt.wantConfigFound {
				t.Fatalf("Detect() configFound = %v, want %v", configFound, tt.wantConfigFound)
			}
		})
	}
}

func TestSupportsAutoInstall(t *testing.T) {
	a := NewAdapter()
	if a.SupportsAutoInstall() {
		t.Fatalf("SupportsAutoInstall() = true, want false")
	}
}

func TestInstallCommand(t *testing.T) {
	a := NewAdapter()

	commands, err := a.InstallCommand(system.PlatformProfile{})
	if err == nil {
		t.Fatalf("InstallCommand() error = nil, want non-installable error")
	}
	if commands != nil {
		t.Fatalf("InstallCommand() commands = %v, want nil", commands)
	}

	var notInstallable AgentNotInstallableError
	if !errors.As(err, &notInstallable) {
		t.Fatalf("InstallCommand() error type = %T, want AgentNotInstallableError", err)
	}
	if got := err.Error(); !strings.Contains(got, "must be installed manually") {
		t.Fatalf("InstallCommand() error = %q, want message containing 'must be installed manually'", got)
	}
}

func TestConfigPaths(t *testing.T) {
	a := NewAdapter()
	homeDir := filepath.Join(string(filepath.Separator), "home", "test")
	configDir := filepath.Join(homeDir, ".gemini", "antigravity-cli")
	settingsFile := filepath.Join(configDir, "settings.json")
	pluginDir := filepath.Join(homeDir, ".gemini", "config", "plugins", "gentle-ai")
	pluginManifestFile := filepath.Join(pluginDir, "plugin.json")
	importManifestFile := filepath.Join(homeDir, ".gemini", "config", "import_manifest.json")
	mcpConfigFile := filepath.Join(pluginDir, "mcp_config.json")
	systemPromptDir := filepath.Join(pluginDir, "rules")
	systemPromptFile := filepath.Join(systemPromptDir, "AGENTS.md")

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"GlobalConfigDir", a.GlobalConfigDir(homeDir), configDir},
		{"SettingsPath", a.SettingsPath(homeDir), settingsFile},
		{"PluginDir", a.PluginDir(homeDir), pluginDir},
		{"PluginManifestPath", a.PluginManifestPath(homeDir), pluginManifestFile},
		{"ImportManifestPath", a.ImportManifestPath(homeDir), importManifestFile},
		{"MCPConfigPath", a.MCPConfigPath(homeDir, "engram"), mcpConfigFile},
		{"SystemPromptDir", a.SystemPromptDir(homeDir), systemPromptDir},
		{"SystemPromptFile", a.SystemPromptFile(homeDir), systemPromptFile},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestAntigravityConfigPathsGlobalScopeWithHomeAgents(t *testing.T) {
	// This test demonstrates the bug: when homeDir contains .agents,
	// PluginDir incorrectly resolves to workspace plugin dir for global scope.
	a := NewAdapter()
	homeDir := t.TempDir()
	// Create .agents in home to simulate the bug condition
	if err := os.MkdirAll(filepath.Join(homeDir, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}

	// For global scope, all plugin paths MUST resolve to GlobalPluginDir
	wantGlobalPlugin := filepath.Join(homeDir, ".gemini", "config", "plugins", "gentle-ai")
	wantGlobalRules := filepath.Join(wantGlobalPlugin, "rules")
	wantGlobalAgents := filepath.Join(wantGlobalPlugin, "agents")
	wantGlobalSkills := filepath.Join(wantGlobalPlugin, "skills")
	wantGlobalMCP := filepath.Join(wantGlobalPlugin, "mcp_config.json")

	// These are the scope-aware methods we will add; they should return global paths
	if got := a.GlobalPluginDir(homeDir); got != wantGlobalPlugin {
		t.Fatalf("GlobalPluginDir() = %q, want %q", got, wantGlobalPlugin)
	}
	if got := a.GlobalSystemPromptDir(homeDir); got != wantGlobalRules {
		t.Fatalf("GlobalSystemPromptDir() = %q, want %q", got, wantGlobalRules)
	}
	if got := a.GlobalSystemPromptFile(homeDir); got != filepath.Join(wantGlobalRules, "AGENTS.md") {
		t.Fatalf("GlobalSystemPromptFile() = %q, want %q", got, filepath.Join(wantGlobalRules, "AGENTS.md"))
	}
	if got := a.GlobalSkillsDir(homeDir); got != wantGlobalSkills {
		t.Fatalf("GlobalSkillsDir() = %q, want %q", got, wantGlobalSkills)
	}
	if got := a.GlobalSubAgentsDir(homeDir); got != wantGlobalAgents {
		t.Fatalf("GlobalSubAgentsDir() = %q, want %q", got, wantGlobalAgents)
	}
	if got := a.GlobalMCPConfigPath(homeDir); got != wantGlobalMCP {
		t.Fatalf("GlobalMCPConfigPath() = %q, want %q", got, wantGlobalMCP)
	}

	// The legacy PluginDir (without scope) should still work for backward compat,
	// but call sites must use scope-aware methods.
	_ = a.PluginDir(homeDir) // legacy, not asserted
}

func TestWorkspaceConfigPaths(t *testing.T) {
	a := NewAdapter()
	workspaceDir := t.TempDir()

	wantWorkspacePlugin := filepath.Join(workspaceDir, ".agents", "plugins", "gentle-ai")
	if got := a.WorkspacePluginDir(workspaceDir); got != wantWorkspacePlugin {
		t.Fatalf("WorkspacePluginDir() = %q, want %q", got, wantWorkspacePlugin)
	}
	if got := a.WorkspaceSystemPromptDir(workspaceDir); got != filepath.Join(wantWorkspacePlugin, "rules") {
		t.Fatalf("WorkspaceSystemPromptDir() = %q, want %q", got, filepath.Join(wantWorkspacePlugin, "rules"))
	}
	if got := a.WorkspaceSystemPromptFile(workspaceDir); got != filepath.Join(wantWorkspacePlugin, "rules", "AGENTS.md") {
		t.Fatalf("WorkspaceSystemPromptFile() = %q, want %q", got, filepath.Join(wantWorkspacePlugin, "rules", "AGENTS.md"))
	}
	if got := a.WorkspaceSkillsDir(workspaceDir); got != filepath.Join(wantWorkspacePlugin, "skills") {
		t.Fatalf("WorkspaceSkillsDir() = %q, want %q", got, filepath.Join(wantWorkspacePlugin, "skills"))
	}
	if got := a.WorkspaceSubAgentsDir(workspaceDir); got != filepath.Join(wantWorkspacePlugin, "agents") {
		t.Fatalf("WorkspaceSubAgentsDir() = %q, want %q", got, filepath.Join(wantWorkspacePlugin, "agents"))
	}
	if got := a.WorkspaceMCPConfigPath(workspaceDir); got != filepath.Join(wantWorkspacePlugin, "mcp_config.json") {
		t.Fatalf("WorkspaceMCPConfigPath() = %q, want %q", got, filepath.Join(wantWorkspacePlugin, "mcp_config.json"))
	}

	// When .git directory exists in workspace (even without .agents yet)
	workspaceWithGit := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspaceWithGit, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	wantGitPlugin := filepath.Join(workspaceWithGit, ".agents", "plugins", "gentle-ai")
	if got := a.PluginDir(workspaceWithGit); got != wantGitPlugin {
		t.Fatalf("PluginDir() with .git = %q, want %q", got, wantGitPlugin)
	}

	// When .agents directory exists in workspace
	if err := os.MkdirAll(filepath.Join(workspaceDir, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := a.PluginDir(workspaceDir); got != wantWorkspacePlugin {
		t.Fatalf("PluginDir() with .agents = %q, want %q", got, wantWorkspacePlugin)
	}
	if got := a.SystemPromptDir(workspaceDir); got != filepath.Join(wantWorkspacePlugin, "rules") {
		t.Fatalf("SystemPromptDir() with .agents = %q, want %q", got, filepath.Join(wantWorkspacePlugin, "rules"))
	}
	if got := a.SystemPromptFile(workspaceDir); got != filepath.Join(wantWorkspacePlugin, "rules", "AGENTS.md") {
		t.Fatalf("SystemPromptFile() with .agents = %q, want %q", got, filepath.Join(wantWorkspacePlugin, "rules", "AGENTS.md"))
	}
	if got := a.SkillsDir(workspaceDir); got != filepath.Join(wantWorkspacePlugin, "skills") {
		t.Fatalf("SkillsDir() with .agents = %q, want %q", got, filepath.Join(wantWorkspacePlugin, "skills"))
	}
	if got := a.SubAgentsDir(workspaceDir); got != filepath.Join(wantWorkspacePlugin, "agents") {
		t.Fatalf("SubAgentsDir() with .agents = %q, want %q", got, filepath.Join(wantWorkspacePlugin, "agents"))
	}
	if got := a.MCPConfigPath(workspaceDir, "context7"); got != filepath.Join(wantWorkspacePlugin, "mcp_config.json") {
		t.Fatalf("MCPConfigPath() with .agents = %q, want %q", got, filepath.Join(wantWorkspacePlugin, "mcp_config.json"))
	}
}

func TestCapabilities(t *testing.T) {
	a := NewAdapter()

	if got := a.Agent(); got != model.AgentAntigravityCLI {
		t.Fatalf("Agent() = %q, want %q", got, model.AgentAntigravityCLI)
	}

	if got := a.Tier(); got != model.TierFull {
		t.Fatalf("Tier() = %v, want TierFull", got)
	}

	features := a.CapabilityManifest().Features
	if !features.Skills {
		t.Fatalf("Features.Skills = false, want true")
	}

	if !a.SupportsSkills() {
		t.Fatalf("SupportsSkills() = false, want true")
	}

	if !a.SupportsSystemPrompt() {
		t.Fatalf("SupportsSystemPrompt() = false, want true")
	}

	if !a.SupportsMCP() {
		t.Fatalf("SupportsMCP() = false, want true")
	}

	if got := a.MCPStrategy(); got != model.StrategyMCPConfigFile {
		t.Fatalf("MCPStrategy() = %v, want StrategyMCPConfigFile", got)
	}
}

func TestSubAgents(t *testing.T) {
	a := NewAdapter()
	homeDir := filepath.Join(string(filepath.Separator), "home", "test")

	if !a.SupportsSubAgents() {
		t.Fatalf("SupportsSubAgents() = false, want true")
	}

	if !a.CapabilityManifest().Features.FileSubAgents {
		t.Fatalf("CapabilityManifest().Features.FileSubAgents = false, want true")
	}

	wantEmbedded := "antigravitycli/agents"
	if got := a.EmbeddedSubAgentsDir(); got != wantEmbedded {
		t.Fatalf("EmbeddedSubAgentsDir() = %q, want %q", got, wantEmbedded)
	}

	wantSubAgentsDir := filepath.Join(homeDir, ".gemini", "config", "plugins", "gentle-ai", "agents")
	if got := a.SubAgentsDir(homeDir); got != wantSubAgentsDir {
		t.Fatalf("SubAgentsDir() = %q, want %q", got, wantSubAgentsDir)
	}
}

func TestSystemPromptStrategy(t *testing.T) {
	a := NewAdapter()
	if got := a.SystemPromptStrategy(); got != model.StrategyAppendToFile {
		t.Fatalf("SystemPromptStrategy() = %v, want StrategyAppendToFile", got)
	}
}

func TestBundleAssets(t *testing.T) {
	a := NewAdapter()
	assetsList := a.BundleAssets()
	if len(assetsList) == 0 {
		t.Fatalf("BundleAssets() returned 0 assets")
	}
	for _, pair := range assetsList {
		embedPath := pair[0]
		relDest := pair[1]
		if embedPath == "" || relDest == "" {
			t.Errorf("empty asset pair: %v", pair)
		}
	}
	// agy never reads the plugin root; the orchestrator must deploy as an
	// always-on rule file under rules/, never at the old root location.
	sawOrchestratorRule := false
	for _, pair := range assetsList {
		if pair[0] == "antigravitycli/orchestrator.md" {
			if pair[1] != filepath.Join("rules", "gentle-ai-orchestrator.md") {
				t.Errorf("orchestrator asset must map to rules/gentle-ai-orchestrator.md, got %q", pair[1])
			}
			sawOrchestratorRule = true
		}
		if pair[1] == "orchestrator.md" {
			t.Errorf("no asset may deploy to the unread plugin root: %q -> %q", pair[0], pair[1])
		}
	}
	if !sawOrchestratorRule {
		t.Errorf("BundleAssets() lost the orchestrator rule mapping")
	}
}

func TestEnsurePluginImportedMergesDestructiveManifest(t *testing.T) {
	homeDir := t.TempDir()
	adapter := NewAdapter()
	pluginDir := adapter.PluginDir(homeDir)
	hooksDir := filepath.Join(pluginDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(hooksDir, 0o710); err != nil {
		t.Fatal(err)
	}
	writeModeFile(t, filepath.Join(pluginDir, "plugin.json"), []byte(`{"name":"gentle-ai"}`), 0o640)
	writeModeFile(t, filepath.Join(hooksDir, "hook.sh"), []byte("#!/bin/sh\n"), 0o750)

	manifestPath := adapter.ImportManifestPath(homeDir)
	original := []byte(`{"imports":[{"name":"unrelated-plugin","source":"external","importedAt":"2026-01-02T03:04:05Z"}],"unrelatedRoot":{"keep":true},"keepRoot":"value"}`)
	writeModeFile(t, manifestPath, original, 0o640)
	originalInfo, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	created := []byte(`{"name":"gentle-ai","source":"antigravity","importedAt":"2026-09-23T23:55:18Z","components":["skills","agents"]}`)

	var staging string
	adapter.lookPath = func(string) (string, error) { return "/usr/local/bin/agy", nil }
	adapter.runPluginImport = func(_ context.Context, command string, args []string, env []string) ([]byte, error) {
		staging = args[2]
		if command != "/usr/local/bin/agy" || len(args) != 3 || args[0] != "plugin" || args[1] != "install" {
			t.Fatalf("command/args = %q %v, want agy plugin install <staging>", command, args)
		}
		if filepath.Clean(staging) == filepath.Clean(pluginDir) {
			t.Fatalf("staging source equals canonical destination %q", pluginDir)
		}
		if !slices.Contains(env, "HOME="+homeDir) || !slices.Contains(env, "USERPROFILE="+homeDir) {
			t.Fatalf("HOME/USERPROFILE do not select %q", homeDir)
		}
		for path, want := range map[string]os.FileMode{
			filepath.Join(staging, "plugin.json"):      0o640,
			filepath.Join(staging, "hooks"):            0o710,
			filepath.Join(staging, "hooks", "hook.sh"): 0o750,
		} {
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm() != want {
				t.Fatalf("staged mode %s = %v, %v; want %04o", path, info, err, want)
			}
		}
		destructive := []byte(`{"imports":[` + string(created) + `],"agyMutation":"discard"}`)
		writeModeFile(t, manifestPath, destructive, 0o666)
		return []byte("installed gentle-ai"), nil
	}

	if err := adapter.EnsurePluginImported(context.Background(), homeDir); err != nil {
		t.Fatal(err)
	}
	after, err := loadPluginImportManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.imports) != 2 || compactJSON(t, after.imports[0].raw) != compactJSON(t, []byte(`{"name":"unrelated-plugin","source":"external","importedAt":"2026-01-02T03:04:05Z"}`)) || compactJSON(t, after.imports[1].raw) != compactJSON(t, created) {
		t.Fatalf("merged imports = %s, %s; want original entry plus exact AGY entry", after.imports[0].raw, after.imports[1].raw)
	}
	if compactJSON(t, after.root["unrelatedRoot"]) != `{"keep":true}` || compactJSON(t, after.root["keepRoot"]) != `"value"` {
		t.Fatalf("original root fields were not preserved: %#v", after.root)
	}
	if _, ok := after.root["agyMutation"]; ok {
		t.Fatalf("AGY-only root mutation was retained: %#v", after.root)
	}
	afterInfo, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if afterInfo.Mode() != originalInfo.Mode() {
		t.Fatalf("merged manifest mode = %v, want original %v", afterInfo.Mode(), originalInfo.Mode())
	}
	assertStagingRemoved(t, staging)
}

func TestEnsurePluginImportedManifestGate(t *testing.T) {
	tests := []struct {
		name, raw string
		wantErr   bool
	}{
		{name: "already imported", raw: `{"imports":[{"name":"gentle-ai","importedAt":"2026-01-02T03:04:05Z"}],"unrelated":{"keep":true}}`},
		{name: "invalid JSON", raw: `{"imports":[`, wantErr: true},
		{name: "imports object", raw: `{"imports":{"gentle-ai":{}}}`, wantErr: true},
		{name: "entry missing name", raw: `{"imports":[{"source":"antigravity"}]}`, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			homeDir := t.TempDir()
			adapter := NewAdapter()
			manifestPath := adapter.ImportManifestPath(homeDir)
			original := []byte(test.raw)
			writeModeFile(t, manifestPath, original, 0o600)
			calls := 0
			adapter.lookPath = func(string) (string, error) { return "", errors.New("must not resolve agy") }
			adapter.runPluginImport = func(context.Context, string, []string, []string) ([]byte, error) {
				calls++
				return nil, nil
			}

			err := adapter.EnsurePluginImported(context.Background(), homeDir)
			if (err != nil) != test.wantErr || (test.wantErr && !strings.Contains(err.Error(), "import manifest")) {
				t.Fatalf("EnsurePluginImported() error = %v, wantErr=%t", err, test.wantErr)
			}
			after, readErr := os.ReadFile(manifestPath)
			info, statErr := os.Stat(manifestPath)
			if readErr != nil || string(after) != test.raw || statErr != nil || info.Mode().Perm() != 0o600 || calls != 0 {
				t.Fatalf("manifest gate mutated state: bytes=%q mode=%v calls=%d err=%v", after, info, calls, readErr)
			}
		})
	}
}

func TestEnsurePluginImportedFailures(t *testing.T) {
	tests := []struct {
		name, want string
		tree       bool
		noBinary   bool
		timeout    bool
		run        func(context.Context) ([]byte, error)
	}{
		{name: "missing binary", tree: true, noBinary: true, want: "install Antigravity CLI 1.2.9"},
		{name: "copy failure", want: "inspect completed Antigravity CLI plugin tree"},
		{name: "install failure", tree: true, want: "install exploded", run: func(context.Context) ([]byte, error) { return []byte("install exploded"), errors.New("exit status 1") }},
		{name: "missing registration", tree: true, want: "without registering", run: func(context.Context) ([]byte, error) { return nil, nil }},
		{name: "timeout", tree: true, timeout: true, want: "timed out", run: func(ctx context.Context) ([]byte, error) { <-ctx.Done(); return nil, ctx.Err() }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			homeDir := t.TempDir()
			adapter := NewAdapter()
			if test.tree {
				writeModeFile(t, adapter.PluginManifestPath(homeDir), []byte(`{"name":"gentle-ai"}`), 0o644)
			}
			adapter.lookPath = func(string) (string, error) {
				if test.noBinary {
					return "", errors.New("not installed")
				}
				return "/usr/local/bin/agy", nil
			}
			var staging string
			if test.run != nil {
				adapter.pluginImportTimeout = 20 * time.Millisecond
				adapter.runPluginImport = func(ctx context.Context, _ string, args []string, _ []string) ([]byte, error) {
					staging = args[2]
					return test.run(ctx)
				}
			}
			err := adapter.EnsurePluginImported(context.Background(), homeDir)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
			if test.timeout && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("error = %v, want deadline exceeded", err)
			}
			assertStagingRemoved(t, staging)
		})
	}
}

func TestEnsurePluginImportedRestoresAfterDestructiveFailure(t *testing.T) {
	original := []byte(`{"imports":[{"name":"unrelated-plugin","source":"external"}],"unrelatedRoot":{"keep":true}}`)
	tests := []struct {
		name       string
		postImport string
		run        func(context.Context) ([]byte, error)
		timeout    bool
	}{
		{
			name:       "command failure",
			postImport: `{"imports":[{"name":"gentle-ai","importedAt":"2026-09-23T23:55:18Z"}],"agyMutation":"discard"}`,
			run:        func(context.Context) ([]byte, error) { return []byte("install exploded"), errors.New("exit status 1") },
		},
		{
			name:       "missing registration",
			postImport: `{"imports":[],"agyMutation":"discard"}`,
			run:        func(context.Context) ([]byte, error) { return nil, nil },
		},
		{
			name:       "duplicate registration",
			postImport: `{"imports":[{"name":"gentle-ai"},{"name":"gentle-ai"}]}`,
			run:        func(context.Context) ([]byte, error) { return nil, nil },
		},
		{
			name:       "invalid manifest",
			postImport: `{"imports":[`,
			run:        func(context.Context) ([]byte, error) { return nil, nil },
		},
		{
			name:       "timeout",
			postImport: `{"imports":[{"name":"gentle-ai","importedAt":"2026-09-23T23:55:18Z"}]}`,
			run:        func(ctx context.Context) ([]byte, error) { <-ctx.Done(); return nil, ctx.Err() },
			timeout:    true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			adapter, homeDir, manifestPath := prepareImportTest(t, original, 0o640)
			adapter.pluginImportTimeout = 20 * time.Millisecond
			adapter.runPluginImport = func(ctx context.Context, _ string, _ []string, _ []string) ([]byte, error) {
				writeModeFile(t, manifestPath, []byte(test.postImport), 0o666)
				return test.run(ctx)
			}
			err := adapter.EnsurePluginImported(context.Background(), homeDir)
			if err == nil {
				t.Fatal("EnsurePluginImported() error = nil")
			}
			if test.timeout && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("error = %v, want deadline exceeded", err)
			}
			after, readErr := os.ReadFile(manifestPath)
			if readErr != nil || !bytes.Equal(after, original) {
				t.Fatalf("restored bytes = %q, %v; want %q", after, readErr, original)
			}
			info, statErr := os.Stat(manifestPath)
			if statErr != nil || info.Mode().Perm() != 0o640 {
				t.Fatalf("restored mode = %v, %v; want 0640", info, statErr)
			}
		})
	}
}

func TestEnsurePluginImportedRestoresOriginalAfterMergeWriteFailure(t *testing.T) {
	original := []byte(`{"imports":[{"name":"unrelated-plugin","source":"external"}],"unrelatedRoot":{"keep":true}}`)
	adapter, homeDir, manifestPath := prepareImportTest(t, original, 0o640)
	adapter.lookPath = func(string) (string, error) { return "/usr/local/bin/agy", nil }
	adapter.runPluginImport = func(context.Context, string, []string, []string) ([]byte, error) {
		writeModeFile(t, manifestPath, []byte(`{"imports":[{"name":"gentle-ai","importedAt":"2026-09-23T23:55:18Z"}]}`), 0o666)
		return nil, nil
	}
	mergeErr := errors.New("merge write failed")
	writeCalls := 0
	adapter.writePluginImportManifest = func(path string, data []byte, mode os.FileMode) error {
		writeCalls++
		if writeCalls == 1 {
			return mergeErr
		}
		return defaultPluginImportManifestWriter(path, data, mode)
	}

	err := adapter.EnsurePluginImported(context.Background(), homeDir)
	if !errors.Is(err, mergeErr) {
		t.Fatalf("error = %v, want merge error", err)
	}
	after, readErr := os.ReadFile(manifestPath)
	if readErr != nil || !bytes.Equal(after, original) {
		t.Fatalf("restored bytes = %q, %v; want %q", after, readErr, original)
	}
	info, statErr := os.Stat(manifestPath)
	if statErr != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("restored mode = %v, %v; want 0640", info, statErr)
	}
}

func TestEnsurePluginImportedRemovesCreatedManifestAfterCommandFailure(t *testing.T) {
	adapter, homeDir, manifestPath := prepareImportTest(t, nil, 0)
	adapter.runPluginImport = func(context.Context, string, []string, []string) ([]byte, error) {
		writeModeFile(t, manifestPath, []byte(`{"imports":[{"name":"gentle-ai"}]}`), 0o644)
		return nil, errors.New("install exploded")
	}

	if err := adapter.EnsurePluginImported(context.Background(), homeDir); err == nil {
		t.Fatal("EnsurePluginImported() error = nil")
	}
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Fatalf("AGY-created manifest still exists: %v", err)
	}
}

func prepareImportTest(t *testing.T, manifest []byte, mode os.FileMode) (*Adapter, string, string) {
	t.Helper()
	homeDir := t.TempDir()
	adapter := NewAdapter()
	writeModeFile(t, adapter.PluginManifestPath(homeDir), []byte(`{"name":"gentle-ai"}`), 0o644)
	manifestPath := adapter.ImportManifestPath(homeDir)
	if manifest != nil {
		writeModeFile(t, manifestPath, manifest, mode)
	}
	adapter.lookPath = func(string) (string, error) { return "/usr/local/bin/agy", nil }
	return adapter, homeDir, manifestPath
}

func TestEnsurePluginImportedRefusesPluginTreeSymlink(t *testing.T) {
	homeDir := t.TempDir()
	adapter := NewAdapter()
	adapter.lookPath = func(string) (string, error) { return "/usr/local/bin/agy", nil }
	pluginDir := adapter.PluginDir(homeDir)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	writeModeFile(t, outside, []byte("outside"), 0o644)
	if err := os.Symlink(outside, filepath.Join(pluginDir, "plugin.json")); err != nil {
		t.Skipf("symlink test unsupported: %v", err)
	}
	adapter.runPluginImport = func(context.Context, string, []string, []string) ([]byte, error) {
		t.Fatal("symlink refusal must happen before installer")
		return nil, nil
	}

	err := adapter.EnsurePluginImported(context.Background(), homeDir)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("error = %v, want symlink refusal", err)
	}
	contents, readErr := os.ReadFile(outside)
	if readErr != nil || string(contents) != "outside" {
		t.Fatalf("outside target changed: %q, %v", contents, readErr)
	}
}

func compactJSON(t *testing.T, raw []byte) string {
	t.Helper()
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		t.Fatalf("compact JSON %q: %v", raw, err)
	}
	return compact.String()
}

func assertStagingRemoved(t *testing.T, path string) {
	t.Helper()
	if path == "" {
		return
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("staging directory %q was not cleaned: %v", path, err)
	}
}

func writeModeFile(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func TestDeployNativeRules(t *testing.T) {
	a := NewAdapter()
	home := t.TempDir()
	plugin := a.PluginDir(home)
	names := []string{"orchestrator-delegation", "orchestrator-delegation-reference", "orchestrator-memory", "orchestrator-skills", "orchestrator-prompts", "orchestrator-tracking", "orchestrator-verification", "orchestrator-writer", "strict-tdd", "strict-tdd-verify"}
	for _, name := range names {
		source := "antigravitycli/" + name + ".md"
		legacy := name + ".md"
		if strings.HasPrefix(name, "strict-tdd") {
			source = "antigravitycli/support/" + name + ".md"
			legacy = filepath.Join("support", legacy)
		}
		content, err := assets.Read(source)
		if err != nil {
			t.Fatal(err)
		}
		writeModeFile(t, filepath.Join(plugin, legacy), []byte(content), 0o644)
	}
	persona := filepath.Join(plugin, "rules", "AGENTS.md")
	writeModeFile(t, persona, []byte("User persona\n"), 0o644)
	if err := a.DeployPluginTree(home); err != nil {
		t.Fatal(err)
	}
	first := make(map[string]string)
	for _, name := range names {
		path := filepath.Join(plugin, "rules", name+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		if !strings.HasPrefix(content, "---\ntrigger: model_decision\ndescription: \"") || !strings.Contains(content, "\"\n---\n") {
			t.Errorf("%s: invalid descriptive native metadata", name)
		}
		var metadata struct {
			Trigger     string `yaml:"trigger"`
			Description string `yaml:"description"`
		}
		end := strings.Index(content[4:], "\n---\n")
		if end < 0 || yaml.Unmarshal([]byte(content[4:4+end]), &metadata) != nil || metadata.Trigger != "model_decision" || len(metadata.Description) < 40 {
			t.Errorf("%s: metadata is not valid activation YAML", name)
		}
		if len(data) >= 24000 {
			t.Errorf("%s: exceeds individual rule budget (%d)", name, len(data))
		}
		for _, old := range []string{"`orchestrator.md`", "gentle-ai/orchestrator.md", "support/strict-tdd"} {
			if strings.Contains(content, old) {
				t.Errorf("%s: broken legacy reference %q", name, old)
			}
		}
		for _, target := range append(names, "gentle-ai-orchestrator") {
			if strings.Contains(content, "`"+target+".md`") {
				if _, err := os.Stat(filepath.Join(plugin, "rules", target+".md")); err != nil {
					t.Errorf("%s: unresolved relative reference %s: %v", name, target, err)
				}
			}
		}
		first[path] = content
		legacy := name + ".md"
		if strings.HasPrefix(name, "strict-tdd") {
			legacy = filepath.Join("support", legacy)
		}
		if _, err := os.Stat(filepath.Join(plugin, legacy)); !os.IsNotExist(err) {
			t.Errorf("managed legacy %s remains: %v", legacy, err)
		}
	}
	core, err := os.ReadFile(filepath.Join(plugin, "rules", "gentle-ai-orchestrator.md"))
	if err != nil || !bytes.Contains(core, []byte(ReviewContractInsertMarker)) || bytes.Contains(core, []byte("trigger: model_decision")) {
		t.Fatalf("core authority changed: %v", err)
	}
	if err := a.DeployPluginTree(home); err != nil {
		t.Fatal(err)
	}
	for path, want := range first {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Errorf("second deployment changed %s: %v", path, err)
		}
	}
	if data, err := os.ReadFile(persona); err != nil || string(data) != "User persona\n" {
		t.Errorf("persona changed: %v", err)
	}
}

func TestDeployNativeRulesPreservesConflicts(t *testing.T) {
	for _, relative := range []string{"orchestrator.md", "orchestrator-memory.md", "support/strict-tdd.md", "rules/orchestrator-memory.md"} {
		t.Run(relative, func(t *testing.T) {
			a := NewAdapter()
			home := t.TempDir()
			path := filepath.Join(a.PluginDir(home), relative)
			writeModeFile(t, path, []byte("User customization\n"), 0o644)
			if err := a.DeployPluginTree(home); err == nil || !strings.Contains(err.Error(), "conflict") {
				t.Fatalf("want explicit migration conflict, got %v", err)
			}
			if data, err := os.ReadFile(path); err != nil || string(data) != "User customization\n" {
				t.Fatalf("user content changed: %v", err)
			}
			if _, err := os.Stat(filepath.Join(a.PluginDir(home), "plugin.json")); !os.IsNotExist(err) {
				t.Fatalf("deployment wrote before conflict preflight: %v", err)
			}
		})
	}
}

func TestDeployNativeRulesRejectsSymlinks(t *testing.T) {
	for _, relative := range []string{"rules", "support", "orchestrator-memory.md", "rules/strict-tdd.md", ".native-rule-ownership.json"} {
		t.Run(relative, func(t *testing.T) {
			a := NewAdapter()
			home := t.TempDir()
			path := filepath.Join(a.PluginDir(home), relative)
			target := t.TempDir()
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, path); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			if err := a.DeployPluginTree(home); err == nil || !strings.Contains(err.Error(), "conflict") {
				t.Fatalf("want symlink conflict, got %v", err)
			}
			if entries, err := os.ReadDir(target); err != nil || len(entries) != 0 {
				t.Fatalf("symlink target changed: %v", err)
			}
		})
	}
}

func TestDeployNativeRulesStaleLedgerRetry(t *testing.T) {
	for _, tt := range []struct {
		name     string
		userEdit bool
	}{
		{name: "current managed bytes self-heal"},
		{name: "user edit still conflicts", userEdit: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a, home := NewAdapter(), t.TempDir()
			plugin := a.PluginDir(home)
			relative := "rules/orchestrator-memory.md"
			source, err := assets.Read("antigravitycli/orchestrator-memory.md")
			if err != nil {
				t.Fatal(err)
			}
			current := []byte(renderNativeRule("antigravitycli/orchestrator-memory.md", source))
			installed := current
			if tt.userEdit {
				installed = []byte("User edit\n")
			}
			path := filepath.Join(plugin, relative)
			writeModeFile(t, path, installed, 0o644)
			record := nativeOwnership{Version: 1, Rules: map[string]string{
				relative: ruleDigest([]byte("Previous generated bytes\n")),
			}}
			raw, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			writeModeFile(t, filepath.Join(plugin, nativeOwnershipFile), raw, 0o644)
			err = a.DeployPluginTree(home)
			if tt.userEdit {
				if err == nil || !strings.Contains(err.Error(), "reconcile explicitly before retrying") {
					t.Fatalf("want explicit conflict, got %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("retry with current bytes: %v", err)
				}
				refreshed, err := loadNativeOwnership(plugin)
				if err != nil || refreshed.Rules[relative] != ruleDigest(current) {
					t.Fatalf("ledger did not refresh: %+v, %v", refreshed, err)
				}
			}
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, installed) {
				t.Fatalf("installed bytes changed: %q, %v", got, err)
			}
		})
	}
}

func TestDeployNativeRulesSuccessiveUpgrades(t *testing.T) {
	a, home := NewAdapter(), t.TempDir()
	name := "orchestrator-memory.md"
	original := nativeRuleDescriptions[name]
	defer func() { nativeRuleDescriptions[name] = original }()
	for _, description := range []string{"First generated description", "Second generated description"} {
		nativeRuleDescriptions[name] = description
		if err := a.DeployPluginTree(home); err != nil {
			t.Fatalf("upgrade %q: %v", description, err)
		}
	}
	plugin := a.PluginDir(home)
	path := filepath.Join(plugin, "rules", name)
	// A prior successful generator may change both description and body.
	prior := renderNativeRule("antigravitycli/"+name, "Previous generated body\n")
	writeModeFile(t, path, []byte(prior), 0o644)
	metadataPath := filepath.Join(plugin, ".native-rule-ownership.json")
	raw, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Version int               `json:"version"`
		Rules   map[string]string `json:"rules"`
	}
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatal(err)
	}
	record.Rules["rules/"+name] = fmt.Sprintf("%x", sha256.Sum256([]byte(prior)))
	raw, err = json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	writeModeFile(t, metadataPath, raw, 0o644)
	nativeRuleDescriptions[name] = original
	if err := a.DeployPluginTree(home); err != nil {
		t.Fatalf("body upgrade: %v", err)
	}
	current, err := os.ReadFile(path)
	if err != nil || bytes.Equal(current, []byte(prior)) {
		t.Fatalf("body was not upgraded: %v", err)
	}
	writeModeFile(t, path, []byte("User edit\n"), 0o644)
	if err := a.DeployPluginTree(home); err == nil {
		t.Fatal("accepted edited managed rule")
	}
	got, _ := os.ReadFile(path)
	if string(got) != "User edit\n" {
		t.Fatal("overwrote user edit")
	}
}

func TestDeployNativeRulesInvalidProvenance(t *testing.T) {
	for _, record := range []string{`{`, `{}`, `null`, `{"version":2,"rules":{}}`, `{"version":1,"rules":{},"unknown":true}`, `{"version":1,"rules":{}} {}`, `{"version":1,"rules":{"../outside":"abc"}}`, `{"version":1,"rules":{"rules/orchestrator-memory.md":"bad"}}`} {
		t.Run(record, func(t *testing.T) {
			a, home := NewAdapter(), t.TempDir()
			plugin := a.PluginDir(home)
			writeModeFile(t, filepath.Join(plugin, ".native-rule-ownership.json"), []byte(record), 0o644)
			writeModeFile(t, filepath.Join(plugin, "plugin.json"), []byte("Existing manifest\n"), 0o644)
			if err := a.DeployPluginTree(home); err == nil {
				t.Fatal("accepted invalid provenance")
			}
			got, _ := os.ReadFile(filepath.Join(plugin, "plugin.json"))
			if string(got) != "Existing manifest\n" {
				t.Fatal("wrote before preflight")
			}
		})
	}
}

func TestDeployCoreRuleTypePreflight(t *testing.T) {
	for _, symlink := range []bool{true, false} {
		t.Run(fmt.Sprint(symlink), func(t *testing.T) {
			a, home := NewAdapter(), t.TempDir()
			plugin := a.PluginDir(home)
			path := filepath.Join(plugin, "rules", "gentle-ai-orchestrator.md")
			writeModeFile(t, filepath.Join(plugin, "plugin.json"), []byte("Existing manifest\n"), 0o644)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if symlink {
				target := filepath.Join(t.TempDir(), "core.md")
				writeModeFile(t, target, []byte("Target\n"), 0o644)
				if err := os.Symlink(target, path); err != nil {
					t.Skip(err)
				}
			} else if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := a.DeployPluginTree(home); err == nil {
				t.Fatal("accepted nonregular core")
			}
			got, _ := os.ReadFile(filepath.Join(plugin, "plugin.json"))
			if string(got) != "Existing manifest\n" {
				t.Fatal("manifest changed before core preflight")
			}
		})
	}
}

func TestDeployPluginTree(t *testing.T) {
	homeDir := t.TempDir()
	adapter := NewAdapter()
	pluginDir := adapter.PluginDir(homeDir)

	// Pre-create stale SDD artifacts and arbitrary unmanaged agent
	staleAgentDir := filepath.Join(pluginDir, "agents", "sdd-apply")
	if err := os.MkdirAll(staleAgentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staleAgentDir, "agent.md"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	arbitraryAgentDir := filepath.Join(pluginDir, "agents", "custom-deprecated-agent")
	if err := os.MkdirAll(arbitraryAgentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(arbitraryAgentDir, "agent.md"), []byte("deprecated"), 0o644); err != nil {
		t.Fatal(err)
	}
	staleSkillDir := filepath.Join(pluginDir, "skills", "sdd-apply")
	if err := os.MkdirAll(staleSkillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	staleChain := filepath.Join(pluginDir, "chains", "sdd-full.chain.md")
	writeModeFile(t, staleChain, []byte("stale chain"), 0o644)
	// Only exact embedded bytes prove ownership of the old core location.
	staleRootOrchestrator := filepath.Join(pluginDir, "orchestrator.md")
	oldCore, err := assets.Read("antigravitycli/orchestrator.md")
	if err != nil {
		t.Fatal(err)
	}
	writeModeFile(t, staleRootOrchestrator, []byte(oldCore), 0o644)

	// Pre-create AGENTS.md with duplicate agent-routing
	agentsMD := filepath.Join(pluginDir, "rules", "AGENTS.md")
	contentWithRouting := "<!-- gentle-ai:persona -->\nPersona text\n<!-- /gentle-ai:persona -->\n<!-- gentle-ai:agent-routing -->\nRouting text\n<!-- /gentle-ai:agent-routing -->\n"
	writeModeFile(t, agentsMD, []byte(contentWithRouting), 0o644)

	if err := adapter.DeployPluginTree(homeDir); err != nil {
		t.Fatalf("DeployPluginTree() error = %v", err)
	}

	// Verify Bundle Assets
	for _, pair := range adapter.BundleAssets() {
		dest := filepath.Join(pluginDir, pair[1])
		if _, err := os.Stat(dest); err != nil {
			t.Errorf("missing deployed bundle asset %s: %v", pair[1], err)
		}
	}
	orchestratorRule := filepath.Join(pluginDir, "rules", "gentle-ai-orchestrator.md")
	if data, err := os.ReadFile(orchestratorRule); err != nil {
		t.Errorf("missing orchestrator rule file %s: %v", orchestratorRule, err)
	} else if !bytes.Contains(data, []byte(ReviewContractInsertMarker)) {
		t.Errorf("orchestrator rule file lost its deterministic review-contract marker")
	}
	if _, err := os.Stat(staleRootOrchestrator); !os.IsNotExist(err) {
		t.Errorf("legacy root orchestrator.md was not pruned")
	}

	reviewModelsFile := filepath.Join(pluginDir, "review-models.json")
	if _, err := os.Stat(reviewModelsFile); err != nil {
		t.Errorf("missing deployed review-models.json: %v", err)
	}

	// Verify 10 ODD subagents
	expectedSubagents := []string{
		"gentle-ai-explore",
		"gentle-ai-worker",
		"gentle-ai-verify",
		"jd-fix-agent",
		"jd-judge-a",
		"jd-judge-b",
		"review-readability",
		"review-reliability",
		"review-resilience",
		"review-risk",
	}
	for _, name := range expectedSubagents {
		dest := filepath.Join(pluginDir, "agents", name, "agent.md")
		if info, err := os.Stat(dest); err != nil || info.Size() == 0 {
			t.Errorf("missing or empty deployed subagent %s", name)
		}
	}

	// Verify hooks
	if info, err := os.Stat(filepath.Join(pluginDir, "hooks", "hook.sh")); err != nil || info.Mode().Perm() != 0o755 {
		t.Errorf("invalid hook.sh: %v, info: %v", err, info)
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "hooks", "hook.cmd")); err != nil {
		t.Errorf("missing hook.cmd: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "hooks.json")); err != nil {
		t.Errorf("missing hooks.json: %v", err)
	}

	// Verify stale SDD artifacts and arbitrary unmanaged agent pruned
	if _, err := os.Stat(staleAgentDir); !os.IsNotExist(err) {
		t.Errorf("stale sdd agent directory %q was not removed", staleAgentDir)
	}
	if _, err := os.Stat(arbitraryAgentDir); !os.IsNotExist(err) {
		t.Errorf("unmanaged agent directory %q was not removed", arbitraryAgentDir)
	}
	if _, err := os.Stat(staleSkillDir); !os.IsNotExist(err) {
		t.Errorf("stale sdd skill directory %q was not removed", staleSkillDir)
	}
	if _, err := os.Stat(staleChain); !os.IsNotExist(err) {
		t.Errorf("stale sdd chain %q was not removed", staleChain)
	}

	// Verify AGENTS.md duplicate routing stripped
	cleanedData, err := os.ReadFile(agentsMD)
	if err != nil {
		t.Fatalf("read %s: %v", agentsMD, err)
	}
	if strings.Contains(string(cleanedData), "agent-routing") {
		t.Errorf("AGENTS.md still contains agent-routing:\n%s", string(cleanedData))
	}
	if !strings.Contains(string(cleanedData), "persona") {
		t.Errorf("AGENTS.md lost persona section:\n%s", string(cleanedData))
	}

	// The statusline script is no longer deployed (in-process subcommand):
	// a stale copy from an older sync is pruned with the legacy files.
	if _, err := os.Stat(filepath.Join(pluginDir, "statusline.sh")); !os.IsNotExist(err) {
		t.Errorf("legacy statusline.sh was not pruned from the plugin root")
	}
}

func TestDeployPluginTreePrunesStaleSkills(t *testing.T) {
	homeDir := t.TempDir()
	adapter := NewAdapter()
	skillsDir := adapter.SkillsDir(homeDir)

	// Pre-create a stale skill that was deployed by the old selection-driven flow
	// (e.g., go-testing on real machines) but is NOT in the current bundle.
	staleSkillDir := filepath.Join(skillsDir, "go-testing")
	if err := os.MkdirAll(filepath.Join(staleSkillDir, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staleSkillDir, "SKILL.md"), []byte("stale go-testing skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staleSkillDir, "references", "examples.md"), []byte("stale examples"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Also pre-create a stale SDD skill (legacy per-name prune should still catch it)
	staleSDDSkillDir := filepath.Join(skillsDir, "sdd-apply")
	if err := os.MkdirAll(staleSDDSkillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staleSDDSkillDir, "SKILL.md"), []byte("stale sdd-apply"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := adapter.DeployPluginTree(homeDir); err != nil {
		t.Fatalf("DeployPluginTree() error = %v", err)
	}

	// Verify stale skills are pruned
	if _, err := os.Stat(staleSkillDir); !os.IsNotExist(err) {
		t.Errorf("stale go-testing skill directory %q was not removed", staleSkillDir)
	}
	if _, err := os.Stat(staleSDDSkillDir); !os.IsNotExist(err) {
		t.Errorf("stale sdd-apply skill directory %q was not removed", staleSDDSkillDir)
	}

	// Verify bundle skills are deployed (spot-check a few known bundle skills).
	// _shared is deliberately excluded: it is a fragments directory referenced by
	// skills, not a skill itself, and carries no SKILL.md in the gentle-shell canon.
	bundleSkills := []string{
		"branch-pr",
		"chained-pr",
		"cognitive-doc-design",
		"comment-writer",
		"gentle-ai",
		"issue-creation",
		"judgment-day",
		"rdd-defect-workflow",
		"skill-creator",
		"skill-improver",
		"skill-registry",
		"work-unit-commits",
	}
	for _, name := range bundleSkills {
		skillPath := filepath.Join(skillsDir, name, "SKILL.md")
		if info, err := os.Stat(skillPath); err != nil || info.Size() == 0 {
			t.Errorf("missing or empty deployed bundle skill %s at %s", name, skillPath)
		}
	}
	// Verify the Pi-runtime review ledger is NOT deployed — it was stripped
	// from the bundle (describes Pi's extension system, not agy's runtime).
	// _shared may be absent entirely (empty in the bundle = nothing to deploy).
	sharedDir := filepath.Join(skillsDir, "_shared")
	if _, err := os.Stat(sharedDir); err == nil {
		// _shared exists — verify the review ledger is NOT in it
		if _, err := os.Stat(filepath.Join(sharedDir, "review-ledger-contract.md")); !os.IsNotExist(err) {
			t.Errorf("_shared/review-ledger-contract.md should be pruned (Pi-runtime content removed from the bundle)")
		}
	} else if !os.IsNotExist(err) {
		t.Errorf("unexpected stat error for _shared: %v", err)
	}
	// The Pi-runtime review ledger must NEVER exist in the plugin, regardless.
	if _, err := os.Stat(filepath.Join(skillsDir, "_shared", "review-ledger-contract.md")); !os.IsNotExist(err) {
		t.Errorf("_shared/review-ledger-contract.md was deployed despite the transpiler strip")
	}
}
