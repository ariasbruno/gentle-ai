package antigravitycli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gentleman-programming/gentle-ai/v3/internal/agents/capabilitymanifest"
	"github.com/gentleman-programming/gentle-ai/v3/internal/assets"
	"github.com/gentleman-programming/gentle-ai/v3/internal/components/filemerge"
	"github.com/gentleman-programming/gentle-ai/v3/internal/model"
	"github.com/gentleman-programming/gentle-ai/v3/internal/system"
)

type statResult struct {
	isDir bool
	err   error
}

const (
	gentleAIPluginName = "gentle-ai"

	// pluginImportTimeout bounds the external AGY import. A healthy 1.2.9
	// install completes locally in seconds; this larger safety bound still
	// prevents sync from hanging indefinitely on a broken runtime.
	pluginImportTimeout = 20 * time.Second
	// pluginImportWaitDelay bounds pipe collection after context cancellation.
	pluginImportWaitDelay = time.Second
)

type pluginImportRunner func(ctx context.Context, command string, args []string, env []string) ([]byte, error)
type pluginImportManifestWriter func(path string, data []byte, mode os.FileMode) error
type pluginImportManifestRemover func(path string) error

type Adapter struct {
	lookPath                   func(string) (string, error)
	statPath                   func(string) statResult
	runPluginImport            pluginImportRunner
	pluginImportTimeout        time.Duration
	writePluginImportManifest  pluginImportManifestWriter
	removePluginImportManifest pluginImportManifestRemover
}

func NewAdapter() *Adapter {
	return &Adapter{
		lookPath:                   exec.LookPath,
		statPath:                   defaultStat,
		runPluginImport:            runPluginInstall,
		pluginImportTimeout:        pluginImportTimeout,
		writePluginImportManifest:  defaultPluginImportManifestWriter,
		removePluginImportManifest: os.Remove,
	}
}

// --- Identity ---

func (a *Adapter) Agent() model.AgentID {
	return model.AgentAntigravityCLI
}

func (a *Adapter) Tier() model.SupportTier {
	return model.TierFull
}

// --- Detection ---

func (a *Adapter) Detect(_ context.Context, homeDir string) (bool, string, string, bool, error) {
	binPath, err := a.lookPath("agy")
	installed := err == nil && binPath != ""

	configDir := a.GlobalConfigDir(homeDir)
	stat := a.statPath(configDir)
	if stat.err != nil && !os.IsNotExist(stat.err) {
		return false, "", "", false, stat.err
	}

	configFound := stat.err == nil && stat.isDir
	return installed, binPath, configDir, configFound, nil
}

// --- Installation ---

func (a *Adapter) SupportsAutoInstall() bool {
	return false
}

func (a *Adapter) InstallCommand(_ system.PlatformProfile) ([][]string, error) {
	return nil, AgentNotInstallableError{Agent: model.AgentAntigravityCLI}
}

func (a *Adapter) CapabilityManifest() capabilitymanifest.AgentCapabilityManifest {
	return capabilitymanifest.MustForAgent(model.AgentAntigravityCLI)
}

// --- Config paths ---

func (a *Adapter) GlobalConfigDir(homeDir string) string {
	return filepath.Join(homeDir, ".gemini", "antigravity-cli")
}

func (a *Adapter) PluginDir(homeDir string) string {
	return filepath.Join(homeDir, ".gemini", "config", "plugins", "gentle-ai")
}

func (a *Adapter) PluginManifestPath(homeDir string) string {
	return filepath.Join(a.PluginDir(homeDir), "plugin.json")
}

func (a *Adapter) ImportManifestPath(homeDir string) string {
	return filepath.Join(homeDir, ".gemini", "config", "import_manifest.json")
}

func (a *Adapter) SystemPromptDir(homeDir string) string {
	return filepath.Join(a.PluginDir(homeDir), "rules")
}

func (a *Adapter) SystemPromptFile(homeDir string) string {
	return filepath.Join(a.SystemPromptDir(homeDir), "AGENTS.md")
}

func (a *Adapter) SkillsDir(homeDir string) string {
	return filepath.Join(a.PluginDir(homeDir), "skills")
}

func (a *Adapter) SettingsPath(homeDir string) string {
	return filepath.Join(a.GlobalConfigDir(homeDir), "settings.json")
}

// --- Config strategies ---

func (a *Adapter) SystemPromptStrategy() model.SystemPromptStrategy {
	return model.StrategyAppendToFile
}

func (a *Adapter) MCPStrategy() model.MCPStrategy {
	return model.StrategyMCPConfigFile
}

// --- MCP ---

func (a *Adapter) MCPConfigPath(homeDir string, _ string) string {
	return filepath.Join(a.PluginDir(homeDir), "mcp_config.json")
}

// --- Optional capabilities ---

func (a *Adapter) SupportsOutputStyles() bool {
	return a.CapabilityManifest().Features.OutputStyles
}

func (a *Adapter) OutputStyleDir(_ string) string {
	return ""
}

func (a *Adapter) SupportsSlashCommands() bool {
	return a.CapabilityManifest().Features.SlashCommands
}

func (a *Adapter) CommandsDir(_ string) string {
	return ""
}

func (a *Adapter) SupportsSubAgents() bool {
	return a.CapabilityManifest().Features.FileSubAgents
}

func (a *Adapter) SubAgentsDir(homeDir string) string {
	return filepath.Join(a.PluginDir(homeDir), "agents")
}

func (a *Adapter) EmbeddedSubAgentsDir() string {
	return "antigravitycli/agents"
}

// BundleAssets returns the embedded asset path to destination relative path
// mappings for the Antigravity CLI plugin manifest and managed bundle assets.
func (a *Adapter) BundleAssets() [][2]string {
	return [][2]string{
		{"antigravitycli/plugin.json", "plugin.json"},
		{"antigravitycli/orchestrator.md", "orchestrator.md"},
		{"antigravitycli/orchestrator-delegation.md", "orchestrator-delegation.md"},
		{"antigravitycli/orchestrator-memory.md", "orchestrator-memory.md"},
		{"antigravitycli/orchestrator-skills.md", "orchestrator-skills.md"},
		{"antigravitycli/support/strict-tdd.md", filepath.Join("support", "strict-tdd.md")},
		{"antigravitycli/support/strict-tdd-verify.md", filepath.Join("support", "strict-tdd-verify.md")},
		{"antigravitycli/chains/4r-review.chain.md", filepath.Join("chains", "4r-review.chain.md")},
	}
}

func (a *Adapter) SupportsSkills() bool {
	return a.CapabilityManifest().Features.Skills
}

func (a *Adapter) SupportsSystemPrompt() bool {
	return a.CapabilityManifest().Features.SystemPrompt
}

func (a *Adapter) SupportsMCP() bool {
	return a.CapabilityManifest().Features.MCP
}

type pluginImportEntry struct {
	name string
	raw  json.RawMessage
}

type pluginImportManifest struct {
	root       map[string]json.RawMessage
	imports    []pluginImportEntry
	registered bool
}

type pluginImportSnapshot struct {
	manifest *pluginImportManifest
	data     []byte
	mode     os.FileMode
	exists   bool
}

// PluginImportRegistered reports whether the AGY import manifest contains a
// Gentle AI registration. A missing manifest is an unimported plugin, not an
// error; malformed manifests fail closed before any external command runs.
func PluginImportRegistered(manifestPath string) (bool, error) {
	manifest, err := loadPluginImportManifest(manifestPath)
	if err != nil || manifest == nil {
		return false, err
	}
	return manifest.registered, nil
}

// RemovePluginImportRegistration returns import-manifest bytes with only the
// Gentle AI registration removed. Unknown root fields and unrelated entries are
// retained as raw JSON. When no Gentle AI entry exists, changed is false.
func RemovePluginImportRegistration(manifestPath string) (updated []byte, changed bool, err error) {
	manifest, err := loadPluginImportManifest(manifestPath)
	if err != nil || manifest == nil {
		return nil, false, err
	}

	remaining := make([]json.RawMessage, 0, len(manifest.imports))
	for _, entry := range manifest.imports {
		if entry.name == gentleAIPluginName {
			changed = true
			continue
		}
		remaining = append(remaining, entry.raw)
	}
	if !changed {
		return nil, false, nil
	}

	var imports any = remaining
	if len(remaining) == 0 {
		imports = nil
	}
	encodedImports, err := json.Marshal(imports)
	if err != nil {
		return nil, false, fmt.Errorf("encode Antigravity CLI imports: %w", err)
	}
	manifest.root["imports"] = encodedImports
	updated, err = json.MarshalIndent(manifest.root, "", "  ")
	if err != nil {
		return nil, false, fmt.Errorf("encode Antigravity CLI import manifest: %w", err)
	}
	return append(updated, '\n'), true, nil
}

func loadPluginImportManifest(manifestPath string) (*pluginImportManifest, error) {
	manifest, _, _, err := readPluginImportManifest(manifestPath)
	return manifest, err
}

func snapshotPluginImportManifest(manifestPath string) (pluginImportSnapshot, error) {
	manifest, data, mode, err := readPluginImportManifest(manifestPath)
	if err != nil {
		return pluginImportSnapshot{}, err
	}
	return pluginImportSnapshot{manifest: manifest, data: data, mode: mode, exists: manifest != nil}, nil
}

func readPluginImportManifest(manifestPath string) (*pluginImportManifest, []byte, os.FileMode, error) {
	info, err := os.Lstat(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, 0, nil
		}
		return nil, nil, 0, fmt.Errorf("inspect Antigravity CLI import manifest %q: %w", manifestPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, nil, 0, fmt.Errorf("refuse Antigravity CLI import manifest symlink %q", manifestPath)
	}
	if !info.Mode().IsRegular() {
		return nil, nil, 0, fmt.Errorf("Antigravity CLI import manifest %q is not a regular file", manifestPath)
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("read Antigravity CLI import manifest %q: %w", manifestPath, err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil || root == nil {
		if err == nil {
			err = errors.New("root must be a JSON object")
		}
		return nil, nil, 0, fmt.Errorf("parse Antigravity CLI import manifest %q: %w", manifestPath, err)
	}

	manifest := &pluginImportManifest{root: root}
	rawImports, ok := root["imports"]
	if !ok || strings.TrimSpace(string(rawImports)) == "null" {
		return manifest, raw, info.Mode(), nil
	}
	var rawEntries []json.RawMessage
	if err := json.Unmarshal(rawImports, &rawEntries); err != nil {
		return nil, nil, 0, fmt.Errorf("parse Antigravity CLI import manifest %q imports: %w", manifestPath, err)
	}
	for index, raw := range rawEntries {
		var entry struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &entry); err != nil {
			return nil, nil, 0, fmt.Errorf("parse Antigravity CLI import manifest %q imports[%d]: %w", manifestPath, index, err)
		}
		if strings.TrimSpace(entry.Name) == "" {
			return nil, nil, 0, fmt.Errorf("parse Antigravity CLI import manifest %q imports[%d]: name must be a non-empty string", manifestPath, index)
		}
		manifest.imports = append(manifest.imports, pluginImportEntry{name: entry.Name, raw: raw})
		manifest.registered = manifest.registered || entry.Name == gentleAIPluginName
	}
	return manifest, raw, info.Mode(), nil
}

func mergePluginImportManifest(original *pluginImportManifest, created json.RawMessage) ([]byte, error) {
	if original == nil {
		return nil, errors.New("original Antigravity CLI import manifest is missing")
	}
	root := make(map[string]json.RawMessage, len(original.root)+1)
	for key, value := range original.root {
		root[key] = value
	}
	imports := make([]json.RawMessage, 0, len(original.imports)+1)
	for _, entry := range original.imports {
		imports = append(imports, entry.raw)
	}
	imports = append(imports, created)
	encodedImports, err := json.Marshal(imports)
	if err != nil {
		return nil, fmt.Errorf("encode merged Antigravity CLI imports: %w", err)
	}
	root["imports"] = encodedImports
	updated, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode merged Antigravity CLI import manifest: %w", err)
	}
	return append(updated, '\n'), nil
}

func (a *Adapter) writeManifest(path string, data []byte, mode os.FileMode) error {
	writer := a.writePluginImportManifest
	if writer == nil {
		writer = defaultPluginImportManifestWriter
	}
	return writer(path, data, mode)
}

func (a *Adapter) removeManifest(path string) error {
	remover := a.removePluginImportManifest
	if remover == nil {
		remover = os.Remove
	}
	return remover(path)
}

func (a *Adapter) restorePluginImportManifest(path string, snapshot pluginImportSnapshot) error {
	if snapshot.exists {
		if err := a.writeManifest(path, snapshot.data, snapshot.mode); err != nil {
			return fmt.Errorf("restore Antigravity CLI import manifest: %w", err)
		}
		return nil
	}
	if err := a.removeManifest(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove AGY-created Antigravity CLI import manifest: %w", err)
	}
	return nil
}

func (a *Adapter) failPluginImport(path string, snapshot pluginImportSnapshot, cause error) error {
	if restoreErr := a.restorePluginImportManifest(path, snapshot); restoreErr != nil {
		return errors.Join(cause, restoreErr)
	}
	return cause
}

func defaultPluginImportManifestWriter(path string, data []byte, mode os.FileMode) error {
	_, err := filemerge.WriteFileAtomic(path, data, mode)
	return err
}

const antigravityCLIPluginHooksJSON = `{
  "gentle-ai": {
    "PostInvocation": [
      {
        "command": "gentle-ai hook run --agent=antigravity-cli --event PostInvocation",
        "timeout": 30,
        "type": "command"
      }
    ],
    "PostToolUse": [
      {
        "hooks": [
          {
            "command": "gentle-ai hook run --agent=antigravity-cli --event PostToolUse",
            "timeout": 10,
            "type": "command"
          }
        ],
        "matcher": "*"
      }
    ],
    "PreInvocation": [
      {
        "command": "gentle-ai hook run --agent=antigravity-cli --event PreInvocation",
        "timeout": 30,
        "type": "command"
      }
    ],
    "PreToolUse": [
      {
        "hooks": [
          {
            "command": "gentle-ai hook run --agent=antigravity-cli --event PreToolUse",
            "timeout": 10,
            "type": "command"
          }
        ],
        "matcher": "*"
      }
    ],
    "Stop": [
      {
        "command": "gentle-ai hook run --agent=antigravity-cli --event Stop",
        "timeout": 60,
        "type": "command"
      }
    ]
  }
}`

// DeployPluginTree deploys embedded ODD bundle assets, native subagents, and hooks
// into the Antigravity CLI plugin directory, and cleans up any legacy SDD assets.
func (a *Adapter) DeployPluginTree(homeDir string) error {
	pluginDir := a.PluginDir(homeDir)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return fmt.Errorf("create plugin dir %q: %w", pluginDir, err)
	}

	// 1. Write Bundle Assets
	for _, pair := range a.BundleAssets() {
		content, err := assets.Read(pair[0])
		if err != nil {
			return fmt.Errorf("read bundle asset %s: %w", pair[0], err)
		}
		outPath := filepath.Join(pluginDir, pair[1])
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if _, err := filemerge.WriteFileAtomic(outPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write bundle asset %s: %w", pair[1], err)
		}
	}

	// 2. Deploy Native Subagents
	agentEntries, err := assets.FS.ReadDir("antigravitycli/agents")
	if err != nil {
		return fmt.Errorf("read embedded subagents: %w", err)
	}
	validSubagents := make(map[string]bool)
	for _, entry := range agentEntries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		validSubagents[name] = true
		agentMDPath := filepath.Join("antigravitycli", "agents", name, "agent.md")
		content, err := assets.Read(agentMDPath)
		if err != nil {
			return fmt.Errorf("read subagent %s: %w", agentMDPath, err)
		}
		outPath := filepath.Join(pluginDir, "agents", name, "agent.md")
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if _, err := filemerge.WriteFileAtomic(outPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write subagent %s: %w", name, err)
		}
	}

	// 3. Write Fail-Open Hooks
	hooksDir := filepath.Join(pluginDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}
	hookShPath := filepath.Join(hooksDir, "hook.sh")
	hookShContent := "#!/bin/sh\nexec gentle-ai hook run --agent=antigravity-cli --event \"$1\"\n"
	if _, err := filemerge.WriteFileAtomic(hookShPath, []byte(hookShContent), 0o755); err != nil {
		return err
	}
	_ = os.Chmod(hookShPath, 0o755)

	hookCmdPath := filepath.Join(hooksDir, "hook.cmd")
	hookCmdContent := "@echo off\r\ngentle-ai.exe hook run --agent=antigravity-cli --event %1\r\n"
	if _, err := filemerge.WriteFileAtomic(hookCmdPath, []byte(hookCmdContent), 0o644); err != nil {
		return err
	}

	hooksJSONPath := filepath.Join(pluginDir, "hooks.json")
	if _, err := filemerge.WriteFileAtomic(hooksJSONPath, []byte(antigravityCLIPluginHooksJSON), 0o644); err != nil {
		return err
	}

	// 4. Prune Stale Subagents and Legacy Assets
	if existingEntries, err := os.ReadDir(filepath.Join(pluginDir, "agents")); err == nil {
		for _, entry := range existingEntries {
			if entry.IsDir() && !validSubagents[entry.Name()] {
				_ = os.RemoveAll(filepath.Join(pluginDir, "agents", entry.Name()))
			}
		}
	}
	legacyFiles := []string{
		filepath.Join(pluginDir, "sdd-orchestrator-workflow.md"),
		filepath.Join(pluginDir, "support", "sdd-status-contract.md"),
		filepath.Join(pluginDir, "chains", "sdd-full.chain.md"),
		filepath.Join(pluginDir, "chains", "sdd-plan.chain.md"),
		filepath.Join(pluginDir, "chains", "sdd-verify.chain.md"),
		filepath.Join(pluginDir, "rules", "gentle-ai-codegraph.md"),
		filepath.Join(pluginDir, "rules", "gentle-ai-orchestrator-details.md"),
	}
	for _, path := range legacyFiles {
		_ = os.Remove(path)
	}
	legacySkills := []string{
		"sdd-apply", "sdd-archive", "sdd-design", "sdd-explore",
		"sdd-init", "sdd-onboard", "sdd-propose", "sdd-research",
		"sdd-spec", "sdd-tasks", "sdd-verify",
	}
	for _, name := range legacySkills {
		_ = os.RemoveAll(filepath.Join(pluginDir, "skills", name))
	}

	// 5. Clean up duplicate agent-routing section from AGENTS.md if present
	agentsMDPath := filepath.Join(pluginDir, "rules", "AGENTS.md")
	if data, err := os.ReadFile(agentsMDPath); err == nil {
		cleaned := filemerge.InjectMarkdownSection(string(data), "agent-routing", "")
		if cleaned != string(data) {
			_, _ = filemerge.WriteFileAtomic(agentsMDPath, []byte(cleaned), 0o644)
		}
	}

	return nil
}

// EnsurePluginImported performs the first native AGY import only when the
// manifest has no Gentle AI entry. The canonical plugin tree is copied to a
// distinct temporary source because AGY rejects source == destination.
func (a *Adapter) EnsurePluginImported(ctx context.Context, homeDir string) error {
	manifestPath := a.ImportManifestPath(homeDir)
	snapshot, err := snapshotPluginImportManifest(manifestPath)
	if err != nil {
		return err
	}
	if snapshot.manifest != nil && snapshot.manifest.registered {
		return nil
	}

	pluginDir := a.PluginDir(homeDir)
	pluginInfo, err := os.Lstat(pluginDir)
	if err != nil {
		return fmt.Errorf("inspect completed Antigravity CLI plugin tree %q before import: %w", pluginDir, err)
	}
	if pluginInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refuse Antigravity CLI plugin tree symlink %q", pluginDir)
	}
	if !pluginInfo.IsDir() {
		return fmt.Errorf("completed Antigravity CLI plugin tree %q is not a directory", pluginDir)
	}

	agyPath, err := a.lookPath("agy")
	if err != nil || agyPath == "" {
		if err == nil {
			err = errors.New("empty executable path")
		}
		return fmt.Errorf("locate agy for Antigravity CLI plugin import: %w; install Antigravity CLI 1.2.9 or later and retry sync", err)
	}

	stagingDir, err := os.MkdirTemp("", "gentle-ai-antigravity-cli-import-")
	if err != nil {
		return fmt.Errorf("create Antigravity CLI plugin staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(stagingDir) }()

	if err := copyPluginTree(pluginDir, stagingDir); err != nil {
		return fmt.Errorf("stage completed Antigravity CLI plugin tree %q: %w", pluginDir, err)
	}

	importCtx, cancel := context.WithTimeout(ctx, a.pluginImportTimeout)
	defer cancel()

	env := slices.DeleteFunc(os.Environ(), func(entry string) bool {
		key, _, _ := strings.Cut(entry, "=")
		return key == "HOME" || key == "USERPROFILE"
	})
	env = append(env, "HOME="+homeDir, "USERPROFILE="+homeDir)
	output, runErr := a.runPluginImport(importCtx, agyPath, []string{"plugin", "install", stagingDir}, env)
	if runErr != nil {
		if errors.Is(importCtx.Err(), context.DeadlineExceeded) {
			return a.failPluginImport(manifestPath, snapshot, fmt.Errorf("import Antigravity CLI plugin timed out after %s: %w%s", a.pluginImportTimeout, errors.Join(runErr, importCtx.Err()), formatPluginImportOutput(output)))
		}
		return a.failPluginImport(manifestPath, snapshot, fmt.Errorf("import Antigravity CLI plugin with agy: %w%s", runErr, formatPluginImportOutput(output)))
	}

	postImport, err := loadPluginImportManifest(manifestPath)
	if err != nil {
		return a.failPluginImport(manifestPath, snapshot, fmt.Errorf("verify Antigravity CLI plugin import: %w", err))
	}
	var created json.RawMessage
	if postImport != nil {
		for _, entry := range postImport.imports {
			if entry.name != gentleAIPluginName {
				continue
			}
			if created != nil {
				return a.failPluginImport(manifestPath, snapshot, fmt.Errorf("agy plugin install must produce exactly one %q registration in %s%s", gentleAIPluginName, manifestPath, formatPluginImportOutput(output)))
			}
			created = entry.raw
		}
	}
	if created == nil {
		return a.failPluginImport(manifestPath, snapshot, fmt.Errorf("agy plugin install completed without registering %q in %s%s", gentleAIPluginName, manifestPath, formatPluginImportOutput(output)))
	}
	if snapshot.manifest == nil {
		return nil
	}

	merged, err := mergePluginImportManifest(snapshot.manifest, created)
	if err != nil {
		return a.failPluginImport(manifestPath, snapshot, fmt.Errorf("merge Antigravity CLI import manifest: %w", err))
	}
	if err := a.writeManifest(manifestPath, merged, snapshot.mode); err != nil {
		return a.failPluginImport(manifestPath, snapshot, fmt.Errorf("merge Antigravity CLI import manifest: %w", err))
	}
	return nil
}

func runPluginInstall(ctx context.Context, command string, args []string, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = env
	cmd.WaitDelay = pluginImportWaitDelay
	return cmd.CombinedOutput()
}

func formatPluginImportOutput(output []byte) string {
	if len(strings.TrimSpace(string(output))) == 0 {
		return ""
	}
	return "; output: " + strings.TrimSpace(string(output))
}

func copyPluginTree(source, destination string) error {
	root, err := os.OpenRoot(source)
	if err != nil {
		return err
	}
	defer root.Close()
	sourceFS := root.FS()

	if err := fs.WalkDir(sourceFS, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse plugin-tree symlink %q", path)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refuse non-regular plugin-tree entry %q", path)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := os.CopyFS(destination, sourceFS); err != nil {
		return err
	}
	return fs.WalkDir(sourceFS, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		target := destination
		if path != "." {
			target = filepath.Join(destination, path)
		}
		return os.Chmod(target, info.Mode().Perm())
	})
}

type AgentNotInstallableError struct {
	Agent model.AgentID
}

func (e AgentNotInstallableError) Error() string {
	return "agent " + string(e.Agent) + " must be installed manually via Google's official instructions"
}

func defaultStat(path string) statResult {
	info, err := os.Stat(path)
	if err != nil {
		return statResult{err: err}
	}
	return statResult{isDir: info.IsDir()}
}
