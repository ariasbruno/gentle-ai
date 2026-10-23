package permissions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gentleman-programming/gentle-ai/v4/internal/agents"
	"github.com/gentleman-programming/gentle-ai/v4/internal/components/filemerge"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
)

type InjectionResult struct {
	Changed bool
	Files   []string
}

// TargetPath returns the file path that permission injection creates or updates
// for the adapter, or an empty string when the agent has no supported
// permission injection target. Codex has no target: gentle-ai relies on Codex's
// built-in default permissions and does not write its permissions config at all.
func TargetPath(homeDir string, adapter agents.Adapter) string {
	if agentOverlay(adapter.Agent(), homeDir) == nil {
		return ""
	}
	return adapter.SettingsPath(homeDir)
}

// ManagedOverlay returns the permission overlay gentle-ai injects for the agent
// at this home directory, or nil when the agent has no permission target.
// Uninstall uses it as the authoritative list of the entries gentle-ai owns, so
// it can subtract exactly those entries instead of deleting a whole key the
// user also writes to.
func ManagedOverlay(id model.AgentID, homeDir string) []byte {
	return agentOverlay(id, homeDir)
}

// claudeCodeOverlayJSON sets Claude Code to bypassPermissions mode (auto-accept all).
// Valid modes: "acceptEdits", "bypassPermissions", "default", "dontAsk", "plan".
var claudeCodeOverlayJSON = []byte(`{
  "permissions": {
    "defaultMode": "bypassPermissions",
    "deny": [
      "Bash(rm -rf /)",
      "Bash(sudo rm -rf /)",
      "Bash(rm -rf ~)",
      "Bash(sudo rm -rf ~)",
      "Read(.env)",
      "Read(.env.*)",
      "Edit(.env)",
      "Edit(.env.*)",
      "Read(.ssh/*)",
      "Edit(.ssh/*)",
      "Read(.credentials/*)",
      "Edit(.credentials/*)",
      "Read(Library/Keychains/*)",
      "Edit(Library/Keychains/*)",
      "Read(.aws/credentials)",
      "Edit(.aws/credentials)",
      "Read(.config/gh/hosts.yml)",
      "Edit(.config/gh/hosts.yml)",
      "Read(**/*.pem)",
      "Edit(**/*.pem)",
      "Read(**/*.key)",
      "Edit(**/*.key)",
      "Read(**/secrets/*)",
      "Edit(**/secrets/*)"
    ]
  }
}
`)

// geminiCLIOverlayJSON sets Gemini CLI to "auto_edit" mode (auto-approve edit tools).
var geminiCLIOverlayJSON = []byte(`{
  "general": {
    "defaultApprovalMode": "auto_edit"
  }
}
`)

// qwenCodeOverlayJSON sets Qwen Code to "auto_edit" mode (auto-approve edits, manual approval for shell commands).
var qwenCodeOverlayJSON = []byte(`{
  "permissions": {
    "defaultMode": "auto_edit"
  }
}
`)

// vscodeCopilotOverlayJSON enables auto-approve for VS Code Copilot chat tools.
var vscodeCopilotOverlayJSON = []byte(`{
  "chat.tools.autoApprove": true
}
`)

// antigravityCLIEnvVariants lists the common .env suffixes: the vendor file
// targets support no globs, so .env.* cannot be expressed as a single rule.
var antigravityCLIEnvVariants = []string{".env.local", ".env.development", ".env.dev", ".env.production", ".env.prod", ".env.test", ".env.staging", ".env.qa"}

// antigravityCLIOverlayForHome builds the Antigravity CLI permission overlay
// for a concrete home directory, written to
// ~/.gemini/antigravity-cli/settings.json. The Antigravity CLI permission
// schema defines three lists — deny, ask, allow — whose entries are
// action(target) resources. command targets match word-by-word command
// prefixes; file targets are absolute or workspace-relative paths, a
// directory target covers its whole subtree, and the bare "*" is the only
// wildcard. Because bare file targets resolve against workspace roots only,
// the home-sensitive stores are denied by absolute path derived from the
// injected home directory; the deny-read rules also imply deny-write.
func antigravityCLIOverlayForHome(homeDir string) []byte {
	deny := []string{
		"command(rm -rf)",
		"command(sudo rm -rf)",
		"command(git reset --hard)",
		"command(git clean -fd)",
		"command(git push --force)",
		"command(chmod -R 777)",
		"command(chown -R)",
	}
	for _, name := range append([]string{".env"}, antigravityCLIEnvVariants...) {
		deny = append(deny, "read_file("+name+")", "write_file("+name+")")
	}
	for _, name := range []string{".ssh", ".credentials", ".aws"} {
		deny = append(deny, "read_file("+name+")", "write_file("+name+")")
	}
	for _, target := range antigravityCLIHomeSensitiveTargets(homeDir) {
		deny = append(deny, "read_file("+target+")", "write_file("+target+")")
	}
	encoded, err := json.MarshalIndent(map[string]any{
		"permissions": map[string]any{
			"deny":  deny,
			"allow": []string{"command(gentle-ai codegraph init)", "command(gentle-ai review assess)", "command(gentle-ai review status)"},
		},
	}, "", "  ")
	if err != nil {
		// Unreachable for these value types; nil keeps the injection a no-op
		// rather than writing malformed settings.
		return nil
	}
	return encoded
}

// antigravityCLIHomeSensitiveTargets returns absolute, forward-slashed deny
// targets for the sensitive stores under the injected home: SSH keys,
// credentials and AWS directories, the GitHub hosts token file and the
// macOS Keychains.
func antigravityCLIHomeSensitiveTargets(homeDir string) []string {
	if homeDir == "" {
		return nil
	}
	home := strings.TrimSuffix(filepath.ToSlash(homeDir), "/")
	targets := make([]string, 0, 5)
	for _, suffix := range []string{"/.ssh", "/.credentials", "/.aws", "/Library/Keychains"} {
		targets = append(targets, home+suffix)
	}
	return append(targets, home+"/.config/gh/hosts.yml")
}

// agentOverlay returns the correct permission overlay for the given agent,
// or nil if the agent does not support permission injection via settings.json.
func agentOverlay(id model.AgentID, homeDir string) []byte {
	switch id {
	case model.AgentClaudeCode:
		return claudeCodeOverlayJSON
	case model.AgentOpenCode, model.AgentKilocode:
		return openCodeOverlayJSON
	case model.AgentGeminiCLI:
		return geminiCLIOverlayJSON
	case model.AgentQwenCode:
		return qwenCodeOverlayJSON
	case model.AgentAntigravity, model.AgentAntigravityCLI:
		// Antigravity and Antigravity CLI manage permissions via IDE UI or
		// dynamically in-process via the PreToolUse hook. No static settings.json
		// permissions overlay is injected.
		return nil
	case model.AgentVSCodeCopilot:
		return vscodeCopilotOverlayJSON
	case model.AgentCursor:
		// Cursor manages permissions via cli-config.json, not settings.json.
		return nil
	case model.AgentCodex:
		// Codex relies on its built-in default permissions. gentle-ai writes
		// nothing to Codex's permissions config — not a profile, and not the
		// legacy cleanup that used to strip one. Codex refuses to load a config
		// that defines a [permissions.*] profile without default_permissions,
		// so a cleanup removing the pointer while a user profile survived left
		// Codex unable to start (#1794). An old gentle-dev profile stays until
		// its owner removes it.
		return nil
	case model.AgentHermes:
		// Hermes permission format is undocumented — no overlay is injected (§14).
		return nil
	default:
		return nil
	}
}

func Inject(homeDir string, adapter agents.Adapter) (InjectionResult, error) {
	return InjectAtPath(TargetPath(homeDir, adapter), homeDir, adapter)
}

// InjectAtPath writes the permission overlay to the caller-selected settings
// path while preserving each adapter's permission capability check. Only the
// OpenCode selected settings refuse symlinked, non-regular or locked files;
// other agents keep the shared writer behavior.
func InjectAtPath(settingsPath, homeDir string, adapter agents.Adapter) (InjectionResult, error) {
	if settingsPath == "" {
		return InjectionResult{}, nil
	}

	overlay := agentOverlay(adapter.Agent(), homeDir)
	if overlay == nil {
		return InjectionResult{}, nil
	}
	if adapter.Agent() == model.AgentOpenCode {
		if err := filemerge.RefuseLockedSettingsFile(settingsPath); err != nil {
			return InjectionResult{}, err
		}
	}

	merge := filemerge.MergeJSONObjectsForPath
	switch adapter.Agent() {
	case model.AgentOpenCode:
		merge = filemerge.MergeOpenCodeJSONDefaultsForPath
	case model.AgentKilocode:
		merge = filemerge.MergeJSONDefaultsForPath
	}
	if adapter.Agent() == model.AgentOpenCode || adapter.Agent() == model.AgentKilocode {
		defaultsMerge := merge
		merge = func(path string, base, overlay []byte) ([]byte, error) {
			return defaultsMerge(path, base, withoutRulesLooserThanUserDefault(base, overlay))
		}
	}
	writeResult, err := mergeJSONFile(settingsPath, overlay, merge)
	if err != nil {
		return InjectionResult{}, err
	}

	return InjectionResult{Changed: writeResult.Changed, Files: []string{settingsPath}}, nil
}

// withoutRulesLooserThanUserDefault drops overlay tool rules looser than the
// user's root permission "*" (deny is stricter than ask, ask than allow).
// Merged into a tool map the user scoped, such a rule follows the root "*" and,
// as the last match, would loosen the user's default.
func withoutRulesLooserThanUserDefault(base, overlay []byte) []byte {
	strictness := map[string]int{"allow": 0, "ask": 1, "deny": 2}
	root, err := filemerge.UnmarshalJSONObject(base)
	if err != nil {
		return overlay
	}
	permission, _ := root["permission"].(map[string]any)
	rootAction, _ := permission["*"].(string)
	if patterns, ok := permission["*"].(map[string]any); ok {
		rootAction, _ = patterns["*"].(string)
	}
	floor, ok := strictness[rootAction]
	if !ok || floor == 0 {
		return overlay
	}
	defaults, err := filemerge.UnmarshalJSONObject(overlay)
	if err != nil {
		return overlay
	}
	rules, _ := defaults["permission"].(map[string]any)
	for _, value := range rules {
		patterns, _ := value.(map[string]any)
		for pattern, action := range patterns {
			if name, _ := action.(string); strictness[name] < floor {
				delete(patterns, pattern)
			}
		}
	}
	encoded, err := json.Marshal(defaults)
	if err != nil {
		return overlay
	}
	return encoded
}

func mergeJSONFile(path string, overlay []byte, merge func(string, []byte, []byte) ([]byte, error)) (filemerge.WriteResult, error) {
	baseJSON, err := osReadFile(path)
	if err != nil {
		return filemerge.WriteResult{}, err
	}

	merged, err := merge(path, baseJSON, overlay)
	if err != nil {
		return filemerge.WriteResult{}, err
	}

	return filemerge.WriteFileAtomic(path, merged, filemerge.ExistingFileMode(path, 0o644))
}

var osReadFile = func(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read json file %q: %w", path, err)
	}

	return content, nil
}
