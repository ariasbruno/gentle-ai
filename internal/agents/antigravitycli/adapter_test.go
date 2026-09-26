package antigravitycli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gentleman-programming/gentle-ai/v3/internal/model"
	"github.com/gentleman-programming/gentle-ai/v3/internal/system"
)

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
}
