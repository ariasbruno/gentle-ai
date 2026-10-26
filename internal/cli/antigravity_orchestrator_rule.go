package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	antigravitycliagent "github.com/gentleman-programming/gentle-ai/v4/internal/agents/antigravitycli"
	"github.com/gentleman-programming/gentle-ai/v4/internal/components/filemerge"
	"github.com/gentleman-programming/gentle-ai/v4/internal/components/reviewassets"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
)

// antigravityOrchestratorRuleFrontmatter is the YAML frontmatter agy requires on
// every plugin rule file other than AGENTS.md: without a valid trigger, agy
// silently drops the file on every turn.
const antigravityOrchestratorRuleFrontmatter = `---
trigger: always_on
description: Gentle AI orchestrator instructions for Antigravity - coordinator role, delegation ladder, memory and skill registry
---`

// antigravityReviewRuleFrontmatter is the YAML frontmatter for the detached
// Review Execution Contract rule in AGY.
const antigravityReviewRuleFrontmatter = `---
trigger: model_decision
description: "Use when executing or coordinating Receipt-Driven Development (RDD) review lifecycle, review status, reviewer capture, or acknowledgement."
---`

const antigravityReviewContractPointer = `Execute the native compact review lifecycle defined in rules/gentle-ai-review.md. Native Go owns frozen lenses, provider context and admission, refutation, bounded correction, repository evidence, targeted validation, and approved closure.`

// ensureAntigravityOrchestratorRule turns the deployed orchestrator bundle asset
// into a loadable agy rule: it replaces the deterministic review-contract insert
// marker with a pointer to the detached review rule, prepends always_on
// frontmatter, and deploys the detached review execution contract rule. It runs
// on every install and sync right after DeployPluginTree, and is idempotent so
// repeated syncs produce byte-identical output.
// LEGACY: Uses heuristic workspace detection. Call sites with known scope MUST
// use ensureAntigravityOrchestratorRuleForScope instead.
func ensureAntigravityOrchestratorRule(homeDir string, agy *antigravitycliagent.Adapter) error {
	pluginDir := agy.PluginDir(homeDir)
	rulePath := filepath.Join(pluginDir, "rules", "gentle-ai-orchestrator.md")
	reviewPath := filepath.Join(pluginDir, "rules", "gentle-ai-review.md")
	if err := ensureAntigravityOrchestratorRuleAtPath(rulePath, agy); err != nil {
		return err
	}
	return ensureAntigravityReviewRuleAtPath(reviewPath, agy)
}

// ensureAntigravityOrchestratorRuleForScope finalizes the orchestrator rule and
// review execution contract rule at the correct plugin directory for the given scope.
// Global scope uses GlobalPluginDir(homeDir); workspace scope uses WorkspacePluginDir(workspaceDir).
func ensureAntigravityOrchestratorRuleForScope(scope InstallScope, homeDir, workspaceDir string, agy *antigravitycliagent.Adapter) error {
	var pluginDir string
	if scope == ScopeWorkspace && strings.TrimSpace(workspaceDir) != "" {
		pluginDir = agy.WorkspacePluginDir(workspaceDir)
	} else {
		pluginDir = agy.GlobalPluginDir(homeDir)
	}
	rulePath := filepath.Join(pluginDir, "rules", "gentle-ai-orchestrator.md")
	reviewPath := filepath.Join(pluginDir, "rules", "gentle-ai-review.md")
	if err := ensureAntigravityOrchestratorRuleAtPath(rulePath, agy); err != nil {
		return err
	}
	return ensureAntigravityReviewRuleAtPath(reviewPath, agy)
}

// ensureAntigravityOrchestratorRuleAtPath is the internal implementation that
// finalizes the orchestrator rule at a specific path.
func ensureAntigravityOrchestratorRuleAtPath(rulePath string, agy *antigravitycliagent.Adapter) error {
	raw, err := os.ReadFile(rulePath)
	if err != nil {
		return fmt.Errorf("read Antigravity CLI orchestrator rule %q: %w", rulePath, err)
	}
	content := string(raw)

	if marker := antigravitycliagent.ReviewContractInsertMarker; strings.Contains(content, marker) {
		content = strings.Replace(content, marker, antigravityReviewContractPointer, 1)
	} else if !strings.Contains(content, antigravityReviewContractPointer) && !strings.Contains(content, "rules/gentle-ai-review.md") {
		return fmt.Errorf("orchestrator rule %q carries neither the deterministic insert marker nor a review pointer; redeploy the plugin tree with `gentle-ai sync --agents antigravity-cli` and retry", rulePath)
	}

	content = filemerge.PrependYAMLFrontmatter(content, antigravityOrchestratorRuleFrontmatter)

	if _, err := filemerge.WriteFileAtomic(rulePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write Antigravity CLI orchestrator rule %q: %w", rulePath, err)
	}
	return nil
}

// ensureAntigravityReviewRuleAtPath is the internal implementation that
// generates the detached Review Execution Contract rule at a specific path.
func ensureAntigravityReviewRuleAtPath(reviewPath string, agy *antigravitycliagent.Adapter) error {
	contract, err := reviewassets.ReviewExecutionContractFor(model.AgentAntigravityCLI)
	if err != nil {
		return fmt.Errorf("render review execution contract for %s: %w", model.AgentAntigravityCLI, err)
	}

	content := "# el Gentleman Native Review Execution Contract (Antigravity CLI)\n\n" + contract
	content = filemerge.PrependYAMLFrontmatter(content, antigravityReviewRuleFrontmatter)

	if err := os.MkdirAll(filepath.Dir(reviewPath), 0o755); err != nil {
		return fmt.Errorf("create review rule dir %q: %w", filepath.Dir(reviewPath), err)
	}
	if _, err := filemerge.WriteFileAtomic(reviewPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write Antigravity CLI review rule %q: %w", reviewPath, err)
	}
	return nil
}
