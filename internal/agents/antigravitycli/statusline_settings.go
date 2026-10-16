package antigravitycli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gentleman-programming/gentle-ai/v4/internal/components/filemerge"
)

// Managed statusline registration for the Antigravity CLI (agy).
//
// agy renders a custom status line when `statusLine` (and `title`) in the
// user-level `~/.gemini/antigravity-cli/settings.json` points at a command.
// The managed command is the bare in-process subcommand — PATH-resolved by
// agy and Windows-portable, with no script file and no jq. Gentle AI owns
// only `type`+`command` of a block whose command is empty, already the
// managed subcommand, or the legacy deployed-script registration (rewritten
// to the subcommand on upgrade); every other key (`enabled`, `padding`,
// `stack_with_default`) belongs to the user and is preserved verbatim. A
// block whose command points somewhere else is user-owned and never touched.

const managedStatusLineType = "command"

// ManagedStatusLineCommand is the registered statusLine block command.
const ManagedStatusLineCommand = "gentle-ai statusline"

// ManagedTitleCommand is the registered title block command.
const ManagedTitleCommand = "gentle-ai statusline --title"

// StatusLineScriptPath returns the legacy deployed-script location. It is
// not deployed anymore (the render is in-process); it survives as the upgrade
// identity: registrations and uninstall cleanup that still point here are
// ours and are re-owned or retired.
func (a *Adapter) StatusLineScriptPath(homeDir string) string {
	return filepath.Join(a.GlobalPluginDir(homeDir), "statusline.sh")
}

// EnsureStatusLineSettings registers the managed `statusLine` and `title`
// blocks in the user-level Antigravity CLI settings.json under the S8
// ownership rules. A missing settings.json is created; a malformed one fails
// closed and is left byte-identical; an already-managed configuration is not
// rewritten; a legacy script-path registration is upgraded in place.
func (a *Adapter) EnsureStatusLineSettings(homeDir string) error {
	settingsPath := a.SettingsPath(homeDir)
	legacyStatusLineCmd := a.StatusLineScriptPath(homeDir)

	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read Antigravity CLI settings %q: %w", settingsPath, err)
		}
		root := map[string]any{}
		applyManagedStatusLineBlocks(root, legacyStatusLineCmd)
		return writeStatusLineSettings(settingsPath, root)
	}

	root, err := filemerge.UnmarshalJSONObject(raw)
	if err != nil {
		return fmt.Errorf("parse Antigravity CLI settings %q: %w", settingsPath, err)
	}
	if !applyManagedStatusLineBlocks(root, legacyStatusLineCmd) {
		return nil
	}
	return writeStatusLineSettings(settingsPath, root)
}

// applyManagedStatusLineBlocks registers both managed blocks on the decoded
// root and reports whether anything changed. Each block follows the same
// ownership rule, independently.
func applyManagedStatusLineBlocks(root map[string]any, legacyStatusLineCmd string) bool {
	changed := ensureManagedStatusLineBlock(root, "statusLine", ManagedStatusLineCommand, legacyStatusLineCmd)
	if ensureManagedStatusLineBlock(root, "title", ManagedTitleCommand, legacyStatusLineCmd+" --title") {
		changed = true
	}
	return changed
}

// ensureManagedStatusLineBlock owns type+command of one settings block when
// its command is empty, already the managed one, or the legacy registration.
// Non-object blocks, foreign or malformed commands, and any other keys stay
// untouched; it reports whether the block changed.
func ensureManagedStatusLineBlock(root map[string]any, key, managedCmd, legacyCmd string) bool {
	existing, present := root[key]
	if !present {
		root[key] = map[string]any{
			"type":    managedStatusLineType,
			"command": managedCmd,
		}
		return true
	}
	block, isObject := existing.(map[string]any)
	if !isObject {
		// A shape we do not understand is user-owned: never replace it.
		return false
	}
	command, isString := block["command"].(string)
	if present && !isString {
		return false
	}
	if command != "" && command != managedCmd && command != legacyCmd {
		return false
	}
	changed := false
	if block["type"] != managedStatusLineType {
		block["type"] = managedStatusLineType
		changed = true
	}
	if command != managedCmd {
		block["command"] = managedCmd
		changed = true
	}
	return changed
}

func writeStatusLineSettings(settingsPath string, root map[string]any) error {
	encoded, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("encode Antigravity CLI settings %q: %w", settingsPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return fmt.Errorf("create Antigravity CLI settings dir: %w", err)
	}
	if _, err := filemerge.WriteFileAtomic(settingsPath, append(encoded, '\n'), 0o644); err != nil {
		return fmt.Errorf("write Antigravity CLI settings %q: %w", settingsPath, err)
	}
	return nil
}
