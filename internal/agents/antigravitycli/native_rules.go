package antigravitycli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gentleman-programming/gentle-ai/v4/internal/assets"
)

// Keep sources byte-stable: their exact embedded bytes prove ownership of the
// previous root/support copies. Native metadata belongs to deployment only.
var nativeRuleDescriptions = map[string]string{
	"orchestrator-delegation.md":           "Use when the parent routes work or delegates scoped exploration, implementation, or verification to subagents.",
	"orchestrator-delegation-reference.md": "Use when the delegation card names the reference for the routing ladder, workflows, background, fallback, or language detail.",
	"orchestrator-memory.md":               "Use when the parent records organic progress, maintains large-task continuity, or recovers an interrupted task with Engram.",
	"orchestrator-skills.md":               "Use when the parent resolves skill paths or discovers applicable skills before acting or delegating.",
	"orchestrator-prompts.md":              "Use when relaying a blocking prompt or menu, or handing off a Gentle AI provider defect to the user.",
	"orchestrator-tracking.md":             "Use when a task fails the core resume test or the user requests tracking, to maintain authorized ODD progress and evidence.",
	"orchestrator-verification.md":         "Use when selecting risk-based independent verification or preparing exact authorized writer verification commands.",
	"orchestrator-writer.md":               "Use when the Writer rule fires or an explicitly activated Judgment Day fix batch needs bounded edit surfaces and dispatch.",
	"strict-tdd.md":                        "Use for implementation with applicable runnable deterministic behavior tests, to capture observed RED, GREEN, triangulation, and refactor evidence.",
	"strict-tdd-verify.md":                 "Use when verifying applicable test-first evidence or a justified exception, with proportionate checks and truthful reporting.",
}

func renderNativeRule(source, content string) string {
	if _, ok := nativeRuleDescriptions[filepath.Base(source)]; !ok && source != "antigravitycli/orchestrator.md" {
		return content
	}
	// Absolute installed references must be rewritten before bare relative names.
	content = strings.ReplaceAll(content, "gentle-ai/orchestrator.md", "gentle-ai/rules/gentle-ai-orchestrator.md")
	content = strings.ReplaceAll(content, "`orchestrator.md`", "`gentle-ai-orchestrator.md`")
	for name := range nativeRuleDescriptions {
		old := name
		if strings.HasPrefix(name, "strict-tdd") {
			old = "support/" + name
		}
		content = strings.ReplaceAll(content, "gentle-ai/"+old, "gentle-ai/rules/"+name)
		content = strings.ReplaceAll(content, "`"+old+"`", "`"+name+"`")
	}
	if description, ok := nativeRuleDescriptions[filepath.Base(source)]; ok {
		content = "---\ntrigger: model_decision\ndescription: " + strconv.Quote(description) + "\n---\n\n" + content
	}
	return content
}

const nativeOwnershipFile = ".native-rule-ownership.json"

// Exact SHA-256 of the 8,616-byte legacy core at the pre-modular HEAD.
const legacyCoreSHA256 = "f1e7b59ea21f496fe1b811a6acbf9fb7c2a8a490945c966c93f54e58d2d6e47c"

type nativeOwnership struct {
	Version int               `json:"version"`
	Rules   map[string]string `json:"rules"`
}

func ruleDigest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func inspectNativePath(pluginDir, relative string) error {
	path, current := filepath.Join(pluginDir, relative), pluginDir
	for _, part := range append([]string{""}, strings.Split(filepath.ToSlash(relative), "/")...) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || (current == path && !info.Mode().IsRegular()) || (current != path && !info.IsDir()) {
			return fmt.Errorf("native rule migration conflict at %q: preserve non-regular file or symlink", current)
		}
	}
	return nil
}

func loadNativeOwnership(pluginDir string) (nativeOwnership, error) {
	var record nativeOwnership
	if err := inspectNativePath(pluginDir, nativeOwnershipFile); err != nil {
		return record, err
	}
	data, err := os.ReadFile(filepath.Join(pluginDir, nativeOwnershipFile))
	if os.IsNotExist(err) {
		return nativeOwnership{Version: 1, Rules: make(map[string]string)}, nil
	}
	if err != nil {
		return record, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return record, fmt.Errorf("native rule ownership conflict: %w", err)
	}
	if decoder.Decode(new(any)) != io.EOF || record.Version != 1 || record.Rules == nil {
		return record, fmt.Errorf("native rule ownership conflict: invalid version or record")
	}
	for path, digest := range record.Rules {
		name := strings.TrimPrefix(path, "rules/")
		_, known := nativeRuleDescriptions[name]
		decoded, err := hex.DecodeString(digest)
		if !known || path != "rules/"+name || err != nil || len(decoded) != sha256.Size || digest != strings.ToLower(digest) {
			return record, fmt.Errorf("native rule ownership conflict: invalid path or digest %q", path)
		}
	}
	return record, nil
}

// Preflight all native destinations before writes. Only fixed modular paths
// accept prior deployment digests; the runtime-filled core needs type checks.
func (a *Adapter) preflightNativeRules(pluginDir string) ([]string, error) {
	prior, err := loadNativeOwnership(pluginDir)
	if err != nil {
		return nil, err
	}
	var obsolete []string
	for _, pair := range a.BundleAssets() {
		_, modular := nativeRuleDescriptions[filepath.Base(pair[0])]
		if !modular && pair[0] != "antigravitycli/orchestrator.md" {
			continue
		}
		original, err := assets.Read(pair[0])
		if err != nil {
			return nil, err
		}
		legacy := strings.TrimPrefix(pair[0], "antigravitycli/")
		paths := []string{legacy, pair[1]}
		for _, relative := range paths {
			path := filepath.Join(pluginDir, relative)
			if err := inspectNativePath(pluginDir, relative); err != nil {
				return nil, err
			}
			if !modular && relative == pair[1] {
				continue // Filled RDD bytes may redeploy.
			}
			data, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, err
			}
			want := original
			if relative == pair[1] {
				want = renderNativeRule(pair[0], original)
			}
			owned := string(data) == want
			if modular && relative == pair[1] {
				if digest, recorded := prior.Rules[filepath.ToSlash(relative)]; recorded {
					owned = owned || ruleDigest(data) == digest
				}
			}
			if !modular && relative == legacy && ruleDigest(data) == legacyCoreSHA256 {
				owned = true
			}
			if !owned {
				return nil, fmt.Errorf("native rule migration conflict at %q: preserve user content; reconcile explicitly before retrying", path)
			}
			if relative == legacy {
				obsolete = append(obsolete, path)
			}
		}
	}
	return obsolete, nil
}
