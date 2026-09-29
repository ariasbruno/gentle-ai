package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	antigravitycliagent "github.com/gentleman-programming/gentle-ai/v3/internal/agents/antigravitycli"
	"github.com/gentleman-programming/gentle-ai/v3/internal/components/filemerge"
	"github.com/gentleman-programming/gentle-ai/v3/internal/components/reviewassets"
	"github.com/gentleman-programming/gentle-ai/v3/internal/model"
)

// antigravityOrchestratorRuleFrontmatter is the YAML frontmatter agy requires on
// every plugin rule file other than AGENTS.md: without a valid trigger, agy
// silently drops the file on every turn.
const antigravityOrchestratorRuleFrontmatter = `---
trigger: always_on
description: Gentle AI orchestrator instructions for Antigravity - coordinator role, delegation ladder, memory and skill registry, review execution contract
---`

// ensureAntigravityOrchestratorRule turns the deployed orchestrator bundle asset
// into a loadable agy rule: it replaces the deterministic review-contract insert
// marker with the runtime-bound rendered contract and prepends always_on
// frontmatter. It runs on every install and sync right after DeployPluginTree,
// and is idempotent so repeated syncs produce byte-identical output.
func ensureAntigravityOrchestratorRule(homeDir string, agy *antigravitycliagent.Adapter) error {
	rulePath := filepath.Join(agy.PluginDir(homeDir), "rules", "gentle-ai-orchestrator.md")
	raw, err := os.ReadFile(rulePath)
	if err != nil {
		return fmt.Errorf("read Antigravity CLI orchestrator rule %q: %w", rulePath, err)
	}
	content := string(raw)

	contract, err := reviewassets.ReviewExecutionContractFor(model.AgentAntigravityCLI)
	if err != nil {
		return fmt.Errorf("render review execution contract for %s: %w", model.AgentAntigravityCLI, err)
	}

	if marker := antigravitycliagent.ReviewContractInsertMarker; strings.Contains(content, marker) {
		content = strings.Replace(content, marker, contract, 1)
	} else if !strings.Contains(content, contract) {
		return fmt.Errorf("orchestrator rule %q carries neither the deterministic insert marker nor an already-filled contract; redeploy the plugin tree with `gentle-ai sync --agents antigravity-cli` and retry", rulePath)
	}

	content = filemerge.PrependYAMLFrontmatter(content, antigravityOrchestratorRuleFrontmatter)

	if _, err := filemerge.WriteFileAtomic(rulePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write Antigravity CLI orchestrator rule %q: %w", rulePath, err)
	}
	return nil
}
