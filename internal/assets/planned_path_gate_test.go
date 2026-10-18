package assets

import (
	"io/fs"
	"path"
	"strings"
	"testing"
)

// A gate that rejects unresolved paths must distinguish promised future paths
// from paths claimed to have already been created or read. ODD orchestrators
// without a path-existence gate have no such rejection to qualify.
const plannedPathCarveOut = "A path the artifact explicitly marks as planned (to be created by a later apply) is not required to exist yet; only paths claimed as already created or read must resolve."

// The Antigravity CLI assets are the only runtime directory this repository does
// not author. scripts/transpile-antigravitycli generates every file in it from
// the Gentle Shell source, and the drift gate compares the committed tree with a
// fresh transpilation byte for byte in both directions, so hand-editing anything
// there fails CI instead of correcting it. Two consequences keep the per-runtime
// guards out of that directory:
//
//   - Its orchestrator.md is the compact native core. It routes to per-rule
//     modules rather than carrying the shared ODD sections, so it holds no
//     {{GENTLE_AI_ODD_SECTION:...}} placeholder to assert, and that expansion only
//     runs for the runtimes agentguidance renders an orchestrator for, which
//     deliberately excludes Antigravity CLI.
//   - Its wording is owned upstream, so an assertion here would only pin a
//     revision this repository cannot fix.
//
// Coverage for the bundle therefore comes from the byte-equality drift gate and
// the transpiler's own tests. docs/antigravity-cli/08-decisions.md maps which
// guard covers what, and no guard skips the directory silently.
const transpiledBundleDir = "antigravitycli"

// isTranspiledBundleAsset reports whether an embedded asset path lives inside the
// generated Antigravity CLI bundle, at any depth.
func isTranspiledBundleAsset(assetPath string) bool {
	return assetPath == transpiledBundleDir || strings.HasPrefix(assetPath, transpiledBundleDir+"/")
}

// isTranspiledBundleRootAsset reports whether an embedded asset path sits at the
// bundle's own root, which is where its generated orchestrator.md lives. Bundle
// subdirectories are not roots and keep whatever a guard applies to them.
func isTranspiledBundleRootAsset(assetPath string) bool {
	return path.Dir(assetPath) == transpiledBundleDir
}

func rejectsPlannedPaths(content string) bool {
	return strings.Contains(content, "FAILS the gate.") && !strings.Contains(content, plannedPathCarveOut)
}

func TestEveryODDOrchestratorDoesNotRejectPlannedPaths(t *testing.T) {
	seen := map[string]bool{}
	err := fs.WalkDir(FS, ".", func(assetPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || path.Base(assetPath) != "orchestrator.md" || isTranspiledBundleRootAsset(assetPath) {
			return nil
		}
		seen[path.Dir(assetPath)] = true
		content := MustRead(assetPath)
		if !strings.Contains(content, "GENTLE_AI_ODD_SECTION:") {
			t.Errorf("%s no longer carries the ODD contract", assetPath)
		}
		if rejectsPlannedPaths(content) {
			t.Errorf("%s rejects unresolved paths without the planned-path carve-out", assetPath)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk ODD orchestrators: %v", err)
	}
	// Twelve repo-authored runtimes. Antigravity CLI ships a thirteenth
	// orchestrator asset that stays out of this inventory on purpose; see
	// transpiledBundleDir above.
	for _, runtime := range []string{"antigravity", "claude", "codex", "cursor", "gemini", "generic", "hermes", "kimi", "kiro", "opencode", "qwen", "windsurf"} {
		if !seen[runtime] {
			t.Errorf("missing ODD orchestrator for %s", runtime)
		}
	}
	if len(seen) != 12 {
		t.Errorf("ODD orchestrator inventory changed: got %d runtimes, want 12", len(seen))
	}
}

func TestPlannedPathGateDistinguishesUnqualifiedRejection(t *testing.T) {
	for _, tt := range []struct {
		name, content string
		rejects       bool
	}{
		{"unqualified gate", "A path that does not resolve FAILS the gate.", true},
		{"planned path exception", "A path that does not resolve FAILS the gate. " + plannedPathCarveOut, false},
		{"no path-existence gate", "Track planned implementation paths as tasks.", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := rejectsPlannedPaths(tt.content); got != tt.rejects {
				t.Errorf("rejectsPlannedPaths() = %v, want %v", got, tt.rejects)
			}
		})
	}
}
