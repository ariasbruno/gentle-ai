package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	antigravityDesktopCanary   = `{"antigravity_desktop":"canary_value"}`
	antigravityImportTimestamp = "2026-09-23T23:55:18Z"
	antigravityUnrelatedPlugin = "unrelated-plugin"
)

func antigravityCLIRuntimeFixture(sandbox *Sandbox) error {
	if err := baseRepo(sandbox); err != nil {
		return err
	}

	binDir := filepath.Join(sandbox.Root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	mockAgy := filepath.Join(binDir, "agy")
	script := `#!/bin/sh
set -eu
if [ "$1" = "--version" ]; then
  printf 'agy 1.2.9\n'
  exit 0
fi
if [ "$1" != "plugin" ] || [ "$2" != "install" ]; then
  exit 0
fi
source_dir=$3
destination="$HOME/.gemini/config/plugins/gentle-ai"
if [ "$source_dir" = "$destination" ]; then
  printf 'fake agy: source and destination are identical\n' >&2
  exit 20
fi
if [ ! -f "$source_dir/plugin.json" ]; then
  printf 'fake agy: staged plugin manifest missing\n' >&2
  exit 21
fi
printf 'x' >> "$HOME/.fake-agy-install-count"
# AGY 1.2.9 replaces the import manifest instead of merging it. The product
# must derive the final manifest from its pre-install snapshot.
cat > "$HOME/.gemini/config/import_manifest.json" <<EOF
{
  "imports": [
    {
      "name": "gentle-ai",
      "source": "antigravity",
      "importedAt": "2026-09-23T23:55:18Z",
      "components": ["skills", "agents", "mcpServers", "hooks"]
    }
  ],
  "agyMutation": {"discard": true}
}
EOF
`
	if err := os.WriteFile(mockAgy, []byte(script), 0o755); err != nil {
		return err
	}
	sandbox.PathOverride = binDir

	configDir := filepath.Join(sandbox.Home, ".gemini", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	initialManifest := `{"imports":[{"name":"unrelated-plugin","source":"external","importedAt":"2026-01-02T03:04:05Z"}],"unrelatedRoot":{"keep":true}}`
	if err := os.WriteFile(filepath.Join(configDir, "import_manifest.json"), []byte(initialManifest), 0o644); err != nil {
		return err
	}
	desktopDir := filepath.Join(sandbox.Home, ".config", "antigravity")
	if err := os.MkdirAll(desktopDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(desktopDir, "settings.json"), []byte(antigravityDesktopCanary), 0o644); err != nil {
		return err
	}

	return nil
}

var antigravityCLISyncCapability = &Capability{
	Verb:  []string{"sync"},
	Flags: []string{"--agents"},
}

func antigravityCLISyncArgs(*Sandbox) ([]string, error) {
	return []string{"sync", "--agents", "antigravity-cli"}, nil
}

func antigravityCLIUninstallArgs(*Sandbox) ([]string, error) {
	return []string{"uninstall", "--agents", "antigravity-cli", "--components", "sdd", "--yes"}, nil
}

type antigravityImportManifest struct {
	Imports []struct {
		Name       string `json:"name"`
		ImportedAt string `json:"importedAt"`
	} `json:"imports"`
	UnrelatedRoot map[string]bool `json:"unrelatedRoot"`
	AgyMutation   map[string]bool `json:"agyMutation"`
}

func readAntigravityImportManifest(home string) ([]byte, antigravityImportManifest, error) {
	path := filepath.Join(home, ".gemini", "config", "import_manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, antigravityImportManifest{}, err
	}
	var manifest antigravityImportManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, antigravityImportManifest{}, err
	}
	return raw, manifest, nil
}

func assertAntigravityCLIFirstSync(sandbox *Sandbox, observation Observation) error {
	if observation.ExitCode != 0 {
		return fmt.Errorf("sync --agents antigravity-cli failed: exit=%d stdout=%q stderr=%q", observation.ExitCode, observation.Stdout, observation.Stderr)
	}

	pluginDir := filepath.Join(sandbox.Home, ".gemini", "config", "plugins", "gentle-ai")
	rulesPath := filepath.Join(pluginDir, "rules", "AGENTS.md")
	rulesBytes, err := os.ReadFile(rulesPath)
	if err != nil {
		return fmt.Errorf("missing rules file %s: %w", rulesPath, err)
	}
	rulesStr := string(rulesBytes)
	if !strings.Contains(rulesStr, "<!-- gentle-ai:persona -->") || !strings.Contains(rulesStr, "<!-- gentle-ai:sdd-orchestrator -->") {
		return fmt.Errorf("rules file missing persona or sdd-orchestrator markers: %s", rulesStr)
	}
	sandbox.Scratch["j4500-rules-first-sync"] = rulesStr

	pluginManifest := filepath.Join(pluginDir, "plugin.json")
	if _, err := os.Stat(pluginManifest); err != nil {
		return fmt.Errorf("missing plugin manifest: %w", err)
	}

	manifestBytes, manifest, err := readAntigravityImportManifest(sandbox.Home)
	if err != nil {
		return fmt.Errorf("read import manifest after first sync: %w", err)
	}
	if len(manifest.Imports) != 2 || manifest.Imports[0].Name != antigravityUnrelatedPlugin || manifest.Imports[1].Name != "gentle-ai" {
		return fmt.Errorf("unexpected imports after first sync: %+v", manifest.Imports)
	}
	if manifest.Imports[1].ImportedAt != antigravityImportTimestamp || !manifest.UnrelatedRoot["keep"] || manifest.AgyMutation != nil {
		return fmt.Errorf("import metadata/unrelated fields not preserved: %+v", manifest)
	}
	sandbox.Scratch["j4500-import-manifest-first-sync"] = string(manifestBytes)
	count, err := os.ReadFile(filepath.Join(sandbox.Home, ".fake-agy-install-count"))
	if err != nil || string(count) != "x" {
		return fmt.Errorf("first sync install count = %q, %v; want one import", count, err)
	}

	// Spot-check key subagents
	for _, agent := range []string{"sdd-init", "sdd-apply", "sdd-verify", "sdd-remediate"} {
		agentFile := filepath.Join(pluginDir, "agents", agent, "agent.md")
		if _, err := os.Stat(agentFile); err != nil {
			return fmt.Errorf("missing subagent file %s: %w", agentFile, err)
		}
	}

	// Verify satellites
	for _, sat := range []string{"orchestrator-delegation.md", "orchestrator-memory.md", "orchestrator-skills.md", "sdd-orchestrator-workflow.md"} {
		satFile := filepath.Join(pluginDir, sat)
		if _, err := os.Stat(satFile); err != nil {
			return fmt.Errorf("missing satellite file %s: %w", satFile, err)
		}
	}

	// Verify support contracts
	for _, sup := range []string{"strict-tdd.md", "strict-tdd-verify.md", "sdd-status-contract.md"} {
		supFile := filepath.Join(pluginDir, "support", sup)
		if _, err := os.Stat(supFile); err != nil {
			return fmt.Errorf("missing support contract file %s: %w", supFile, err)
		}
	}

	// Verify chains
	for _, ch := range []string{"sdd-full.chain.md", "4r-review.chain.md"} {
		chFile := filepath.Join(pluginDir, "chains", ch)
		if _, err := os.Stat(chFile); err != nil {
			return fmt.Errorf("missing chain file %s: %w", chFile, err)
		}
	}

	// Verify desktop canary is untouched
	desktopSettings := filepath.Join(sandbox.Home, ".config", "antigravity", "settings.json")
	desktopBytes, err := os.ReadFile(desktopSettings)
	if err != nil {
		return fmt.Errorf("missing desktop settings canary: %w", err)
	}
	if string(desktopBytes) != antigravityDesktopCanary {
		return fmt.Errorf("desktop settings canary was corrupted: got %q, want %q", string(desktopBytes), antigravityDesktopCanary)
	}

	return nil
}

func assertAntigravityCLISecondSync(sandbox *Sandbox, observation Observation) error {
	if observation.ExitCode != 0 {
		return fmt.Errorf("second sync failed: exit=%d stdout=%q stderr=%q", observation.ExitCode, observation.Stdout, observation.Stderr)
	}

	rulesPath := filepath.Join(sandbox.Home, ".gemini", "config", "plugins", "gentle-ai", "rules", "AGENTS.md")
	rulesBytes, err := os.ReadFile(rulesPath)
	if err != nil {
		return fmt.Errorf("missing rules file after second sync: %w", err)
	}
	if string(rulesBytes) != sandbox.Scratch["j4500-rules-first-sync"] {
		return fmt.Errorf("second sync modified rules file (idempotency violated)")
	}
	manifestBytes, manifest, err := readAntigravityImportManifest(sandbox.Home)
	if err != nil {
		return fmt.Errorf("read import manifest after second sync: %w", err)
	}
	if string(manifestBytes) != sandbox.Scratch["j4500-import-manifest-first-sync"] {
		return fmt.Errorf("second sync rewrote import manifest or importedAt")
	}
	if len(manifest.Imports) != 2 || manifest.Imports[1].ImportedAt != antigravityImportTimestamp {
		return fmt.Errorf("second sync changed import registration: %+v", manifest.Imports)
	}
	count, err := os.ReadFile(filepath.Join(sandbox.Home, ".fake-agy-install-count"))
	if err != nil || string(count) != "x" {
		return fmt.Errorf("second sync reinstalled plugin: count=%q err=%v", count, err)
	}

	desktopSettings := filepath.Join(sandbox.Home, ".config", "antigravity", "settings.json")
	desktopBytes, err := os.ReadFile(desktopSettings)
	if err != nil || string(desktopBytes) != antigravityDesktopCanary {
		return fmt.Errorf("desktop settings canary corrupted on second sync")
	}

	return nil
}

func assertAntigravityCLIUninstall(sandbox *Sandbox, observation Observation) error {
	if observation.ExitCode != 0 {
		return fmt.Errorf("uninstall failed: exit=%d stdout=%q stderr=%q", observation.ExitCode, observation.Stdout, observation.Stderr)
	}

	pluginDir := filepath.Join(sandbox.Home, ".gemini", "config", "plugins", "gentle-ai")
	manifestBytes, manifest, err := readAntigravityImportManifest(sandbox.Home)
	if err != nil {
		return fmt.Errorf("read import manifest after uninstall: %w", err)
	}
	if len(manifest.Imports) != 1 || manifest.Imports[0].Name != antigravityUnrelatedPlugin || !manifest.UnrelatedRoot["keep"] {
		return fmt.Errorf("uninstall damaged unrelated imports/fields: %s", manifestBytes)
	}
	pluginManifest := filepath.Join(pluginDir, "plugin.json")
	if _, err := os.Stat(pluginManifest); !os.IsNotExist(err) {
		return fmt.Errorf("plugin manifest was not uninstalled: %v", err)
	}

	// Verify satellites are removed
	for _, sat := range []string{"orchestrator-delegation.md", "orchestrator-memory.md", "orchestrator-skills.md", "sdd-orchestrator-workflow.md"} {
		satFile := filepath.Join(pluginDir, sat)
		if _, err := os.Stat(satFile); !os.IsNotExist(err) {
			return fmt.Errorf("satellite %s was not uninstalled", sat)
		}
	}

	// Verify support contracts are removed
	for _, sup := range []string{"strict-tdd.md", "strict-tdd-verify.md", "sdd-status-contract.md"} {
		supFile := filepath.Join(pluginDir, "support", sup)
		if _, err := os.Stat(supFile); !os.IsNotExist(err) {
			return fmt.Errorf("support contract %s was not uninstalled", sup)
		}
	}

	// Verify chains are removed
	for _, ch := range []string{"sdd-full.chain.md", "4r-review.chain.md"} {
		chFile := filepath.Join(pluginDir, "chains", ch)
		if _, err := os.Stat(chFile); !os.IsNotExist(err) {
			return fmt.Errorf("chain %s was not uninstalled", ch)
		}
	}

	// Verify desktop canary is still untouched
	desktopSettings := filepath.Join(sandbox.Home, ".config", "antigravity", "settings.json")
	desktopBytes, err := os.ReadFile(desktopSettings)
	if err != nil || string(desktopBytes) != antigravityDesktopCanary {
		return fmt.Errorf("desktop settings canary corrupted on uninstall")
	}

	return nil
}

func antigravityCLILifecycleJourneys() []Journey {
	return []Journey{{
		ID:     "j4500-antigravity-cli-lifecycle-parity",
		Review: reviewUntouched,
		Title:  "Antigravity CLI lifecycle: driven import, strict idempotency, uninstallation cleanup, and desktop coexistence",
		Source: "https://github.com/Gentleman-Programming/gentle-ai/issues/antigravity-cli",
		Steps: []Step{
			{Name: "fixture: repository, mock agy binary, and desktop canary", Fixture: antigravityCLIRuntimeFixture},
			{Name: "public sync deploys and imports the Antigravity CLI plugin", Requires: antigravityCLISyncCapability, Args: antigravityCLISyncArgs, After: assertAntigravityCLIFirstSync},
			{Name: "second public sync preserves import bytes and timestamp", Requires: antigravityCLISyncCapability, Args: antigravityCLISyncArgs, After: assertAntigravityCLISecondSync},
			{Name: "public uninstall removes only Gentle AI import state", Args: antigravityCLIUninstallArgs, After: assertAntigravityCLIUninstall},
		},
	}}
}
