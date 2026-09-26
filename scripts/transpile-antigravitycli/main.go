package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultRemoteRepo = "https://github.com/Gentleman-Programming/gentle-shell"
	defaultOutputDir  = "internal/assets/antigravitycli"

	antigravityCLIReviewContractHeading = "#### Review Execution Contract"
	antigravityCLIReviewContractNext    = "Cost and Context Balance"
	antigravityCLIReviewContractMarker  = "<!-- antigravity-review-execution-contract:insert -->"
)

// UnmappedToolWarning records a Pi tool that lacked a native Antigravity CLI translation.
type UnmappedToolWarning struct {
	Agent string
	Tool  string
}

var unmappedToolWarnings []UnmappedToolWarning

// piToAntigravityToolMap maps primitives from Gentle Pi frontmatter to native Antigravity CLI tools.
var piToAntigravityToolMap = map[string][]string{
	"read":                               {"view_file", "list_dir"},
	"grep":                               {"grep_search"},
	"find":                               {"find_by_name"},
	"edit":                               {"replace_file_content"},
	"write":                              {"write_to_file"},
	"bash":                               {"run_command"},
	"web_search":                         {"search_web", "read_url_content"},
	"web_fetch":                          {"read_url_content"},
	"fetch_content":                      {"read_url_content"},
	"get_search_content":                 {"read_url_content"},
	"source_check":                       {"read_url_content"},
	"ask_user_question":                  {"ask_question"},
	"@juicesharp/rpiv-ask-user-question": {"ask_question"},
	"ask_user_choice":                    {"ask_question"},
	"subagent_run":                       {"invoke_subagent", "send_message"},
}

// canonicalToolOrder guarantees deterministic output order in the generated frontmatter.
var canonicalToolOrder = []string{
	"view_file",
	"replace_file_content",
	"write_to_file",
	"list_dir",
	"grep_search",
	"find_by_name",
	"read_url_content",
	"search_web",
	"ask_question",
	"run_command",
	"invoke_subagent",
	"send_message",
}

var isolatedAgents = map[string]bool{
	"jd-judge-a":         true,
	"jd-judge-b":         true,
	"review-readability": true,
	"review-reliability": true,
	"review-resilience":  true,
	"review-risk":        true,
}

// subagentAliases declares intentional output names generated from a source agent.
// In ODD, all legacy SDD aliases (such as sdd-proposal -> sdd-propose) are dropped.
var subagentAliases = map[string][]string{}

func main() {
	sourceDir := flag.String("source-dir", "", "Path to local gentle-pi repository clone (if empty, clones remote)")
	sourceRepo := flag.String("source-repo", defaultRemoteRepo, "Remote git URL for gentle-pi")
	outputDir := flag.String("output-dir", defaultOutputDir, "Destination directory for generated assets")
	strict := flag.Bool("strict", false, "Fail with exit code 1 if any unmapped tools or frontmatter discrepancies are detected")
	flag.Parse()

	piRoot, cleanup, err := resolveSourceRoot(*sourceDir, *sourceRepo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving source repository: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	if err := transpileAll(piRoot, *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Transpilation failed: %v\n", err)
		os.Exit(1)
	}

	if len(unmappedToolWarnings) > 0 {
		fmt.Fprintf(os.Stderr, "\n⚠️  [WARN] Encountered %d unmapped tool(s) without Antigravity CLI translations:\n", len(unmappedToolWarnings))
		for _, w := range unmappedToolWarnings {
			fmt.Fprintf(os.Stderr, "   - Agent %q: Pi tool %q has no translation and was omitted\n", w.Agent, w.Tool)
		}
		fmt.Fprintf(os.Stderr, "\nMaintainer note: To resolve this, add the mapping to piToAntigravityToolMap or declare it in isKnownIgnoredPiTool if it's an internal directive.\n\n")
		if *strict {
			fmt.Fprintf(os.Stderr, "Strict mode: exiting with code 1 due to unmapped tools.\n")
			os.Exit(1)
		}
	}

	fmt.Println("Successfully transpiled Antigravity CLI assets from Gentle Pi.")
}

func resolveSourceRoot(localDir, remoteRepo string) (string, func(), error) {
	if localDir != "" {
		if info, err := os.Stat(localDir); err == nil && info.IsDir() {
			return localDir, func() {}, nil
		}
		return "", func() {}, fmt.Errorf("specified --source-dir %q does not exist or is not a directory", localDir)
	}

	// Check environment variables
	for _, envKey := range []string{"GENTLE_SHELL_DIR", "GENTLE_PI_DIR"} {
		if val := os.Getenv(envKey); val != "" {
			if info, err := os.Stat(val); err == nil && info.IsDir() {
				return val, func() {}, nil
			}
		}
	}

	// Check common sibling dev directories relative to cwd
	candidates := []string{
		filepath.Join("..", "gentle-shell"),
		filepath.Join("..", "gentle-pi"),
		filepath.Join("..", "..", "gentle-shell"),
		filepath.Join("..", "..", "gentle-pi"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, func() {}, nil
		}
	}

	// Clone from remote into a temporary directory
	tempDir, err := os.MkdirTemp("", "gentle-pi-source-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("create temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }

	cmd := exec.Command("git", "clone", "--depth", "1", remoteRepo, tempDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("git clone %s: %w", remoteRepo, err)
	}

	return tempDir, cleanup, nil
}

func transpileAll(piRoot, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// 1. Output plugin.json
	if err := writePluginManifest(outputDir); err != nil {
		return fmt.Errorf("write plugin manifest: %w", err)
	}

	// 2. Transpile Subagents (10 ODD agents)
	if err := transpileSubagents(piRoot, filepath.Join(outputDir, "agents")); err != nil {
		return fmt.Errorf("transpile subagents: %w", err)
	}

	// 3. Transpile Satellites (orchestrator-delegation.md, orchestrator-memory.md, orchestrator-skills.md)
	if err := transpileSatellites(piRoot, outputDir); err != nil {
		return fmt.Errorf("transpile satellites: %w", err)
	}

	// 4. Transpile Support Contracts (strict-tdd.md, strict-tdd-verify.md)
	if err := transpileSupportContracts(piRoot, filepath.Join(outputDir, "support")); err != nil {
		return fmt.Errorf("transpile support contracts: %w", err)
	}

	// 5. Transpile Chains (4r-review.chain.md)
	if err := transpileChains(piRoot, filepath.Join(outputDir, "chains")); err != nil {
		return fmt.Errorf("transpile chains: %w", err)
	}

	// 6. Generate top-level orchestrator asset (orchestrator.md)
	if err := transpileTopLevelOrchestrator(piRoot, outputDir); err != nil {
		return fmt.Errorf("transpile top-level orchestrator: %w", err)
	}

	return nil
}

func writePluginManifest(outputDir string) error {
	manifest := map[string]string{
		"$schema":     "https://antigravity.google/schemas/v1/plugin.json",
		"name":        "gentle-ai",
		"description": "Gentle-AI ecosystem, framework, and workflow guidance for AI coding agents.",
	}
	bytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plugin.json: %w", err)
	}
	return writeDeterministicFile(filepath.Join(outputDir, "plugin.json"), string(bytes))
}

type plannedSubagent struct {
	outputName string
	sourcePath string
}

func planSubagents(piAgentsDir string) ([]plannedSubagent, error) {
	entries, err := os.ReadDir(piAgentsDir)
	if err != nil {
		return nil, fmt.Errorf("read pi agents dir: %w", err)
	}

	planned := make([]plannedSubagent, 0, len(entries))
	sources := make(map[string]string, len(entries))
	addOutput := func(outputName, sourcePath string) error {
		if previous, exists := sources[outputName]; exists {
			return fmt.Errorf("subagent output %q is claimed by both %s and %s", outputName, previous, sourcePath)
		}
		sources[outputName] = sourcePath
		planned = append(planned, plannedSubagent{outputName: outputName, sourcePath: sourcePath})
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		baseName := strings.TrimSuffix(entry.Name(), ".md")
		sourcePath := filepath.Join(piAgentsDir, entry.Name())
		if err := addOutput(baseName, sourcePath); err != nil {
			return nil, err
		}
		for _, alias := range subagentAliases[baseName] {
			if err := addOutput(alias, sourcePath); err != nil {
				return nil, err
			}
		}
	}

	if len(planned) == 0 {
		return nil, fmt.Errorf("no agent source files found in %s", piAgentsDir)
	}
	return planned, nil
}

func transpileSubagents(piRoot, agentsDestDir string) error {
	piAgentsDir := filepath.Join(piRoot, "assets", "agents")
	planned, err := planSubagents(piAgentsDir)
	if err != nil {
		return err
	}

	parentDir := filepath.Dir(agentsDestDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("create agent destination parent: %w", err)
	}
	stagingDir, err := os.MkdirTemp(parentDir, "."+filepath.Base(agentsDestDir)+".staging-*")
	if err != nil {
		return fmt.Errorf("create agent staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	written := make(map[string]bool, len(planned))
	for _, agent := range planned {
		rawBytes, err := os.ReadFile(agent.sourcePath)
		if err != nil {
			return fmt.Errorf("read %s: %w", agent.sourcePath, err)
		}
		transpiled, err := transpileSubagentMarkdown(agent.outputName, string(rawBytes))
		if err != nil {
			return fmt.Errorf("transpile subagent %s: %w", agent.outputName, err)
		}
		targetPath := filepath.Join(stagingDir, agent.outputName, "agent.md")
		if err := writeDeterministicFile(targetPath, transpiled); err != nil {
			return err
		}
		written[agent.outputName] = true
	}

	if len(written) != len(planned) {
		return fmt.Errorf("staged subagent set is incomplete: wrote %d, planned %d", len(written), len(planned))
	}
	return replaceStagedAgentDirectory(stagingDir, agentsDestDir)
}

func replaceStagedAgentDirectory(stagingDir, destination string) error {
	backupDir := stagingDir + ".previous"
	hadDestination := true
	if _, err := os.Lstat(destination); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("inspect agent destination %s: %w", destination, err)
		}
		hadDestination = false
	}

	if hadDestination {
		if err := os.Rename(destination, backupDir); err != nil {
			return fmt.Errorf("stage previous agent destination: %w", err)
		}
	}
	if err := os.Rename(stagingDir, destination); err != nil {
		if hadDestination {
			if rollbackErr := os.Rename(backupDir, destination); rollbackErr != nil {
				return fmt.Errorf("commit staged agents: %v; restore previous agents: %w", err, rollbackErr)
			}
		}
		return fmt.Errorf("commit staged agents: %w", err)
	}
	if hadDestination {
		if err := os.RemoveAll(backupDir); err != nil {
			return fmt.Errorf("remove previous agent tree %s after committed replacement: %w", backupDir, err)
		}
	}
	return nil
}

func transpileSubagentMarkdown(agentName, raw string) (string, error) {
	raw = stripDynamicInjections(raw)

	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "---") {
		return "", fmt.Errorf("missing opening frontmatter fence (---)")
	}
	endIdx := strings.Index(raw[4:], "\n---\n")
	if endIdx < 0 {
		return "", fmt.Errorf("missing closing frontmatter fence (---)")
	}

	description, err := extractDescription(raw)
	if err != nil {
		return "", err
	}

	toolsStr, err := translatePiTools(agentName, raw)
	if err != nil {
		return "", err
	}

	extraFrontmatter := "mainAgent: false\n"
	if isolatedAgents[agentName] {
		extraFrontmatter += "excludeDefaultComponents: true\n"
	}
	frontmatter := fmt.Sprintf("---\nname: %s\ndescription: %s\ntools: %s\n%s---\n", agentName, description, toolsStr, extraFrontmatter)

	body := strings.TrimLeft(raw[4+endIdx+5:], "\n")
	body = stripSection(body, "## Parent Preflight Transport")

	body = mutateAgentRuntimeInstructions(body)
	body = applyCommonTextMutations(body)
	body = applyAntigravityCLIParentMutations(body)
	body, err = alignAntigravityCLIReviewerContract(agentName, body)
	if err != nil {
		return "", err
	}
	trimmedBody := strings.TrimSpace(body)
	if trimmedBody == "" {
		return "", fmt.Errorf("empty body content after frontmatter")
	}

	return frontmatter + "\n" + trimmedBody + "\n", nil
}

// transpileTopLevelOrchestrator generates internal/assets/antigravitycli/orchestrator.md
// from the Pi orchestrator source.
func transpileTopLevelOrchestrator(piRoot, outputDir string) error {
	orchestratorSrc := filepath.Join(piRoot, "assets", "orchestrator.md")
	orchBytes, err := os.ReadFile(orchestratorSrc)
	if err != nil {
		return fmt.Errorf("read orchestrator source %s: %w", orchestratorSrc, err)
	}

	raw := stripDynamicInjections(string(orchBytes))
	body := raw
	if idx := strings.Index(raw, "\n---\n"); idx >= 0 {
		body = strings.TrimLeft(raw[idx+5:], "\n")
	}

	body = applyCommonTextMutations(body)
	body = applyAntigravityCLIParentMutations(body)
	body, err = ensureAntigravityCLIReviewContract(body)
	if err != nil {
		return fmt.Errorf("prepare Antigravity review contract section: %w", err)
	}

	header := `# Agent Teams Lite — Orchestrator Instructions (Antigravity)

Bind this to the dedicated ODD orchestrator Antigravity context only. Do NOT apply it to delegated execution workers.

## Agent Teams Orchestrator (Unified Adapter)

You are the **Google Antigravity agent** running inside **Mission Control**. Antigravity supports native static subagents deployed under ` + "`~/.gemini/config/plugins/gentle-ai/agents/<name>/agent.md`" + ` (or workspace ` + "`.agents/agents/<name>/agent.md`" + `). When delegating bounded ODD work, prioritize delegating directly via ` + "`invoke_subagent`" + `. If static subagents are not detected in the runtime environment, fall back to defining the subagent dynamically with ` + "`define_subagent`" + ` (` + "`enable_mcp_tools: true`" + `) before invoking it.

Your role is to maintain a thin working thread, delegate bounded work, and synthesize results.
`

	var rest string
	if idx := strings.Index(body, "### Lossless Blocking Prompts"); idx >= 0 {
		rest = body[idx:]
	} else if idx := strings.Index(body, "## Core Role"); idx >= 0 {
		rest = body[idx:]
	} else {
		rest = body
	}

	destPath := filepath.Join(outputDir, "orchestrator.md")
	return writeDeterministicFile(destPath, header+"\n"+strings.TrimSpace(rest)+"\n")
}

func ensureAntigravityCLIReviewContract(content string) (string, error) {
	count := strings.Count(content, antigravityCLIReviewContractHeading)
	if count > 1 {
		return "", fmt.Errorf("source contains %d review-contract headings, want exactly one", count)
	}
	if count == 1 {
		if !strings.Contains(content, antigravityCLIReviewContractMarker) {
			return "", fmt.Errorf("review-contract heading is missing deterministic marker %q", antigravityCLIReviewContractMarker)
		}
		return content, nil
	}
	anchor := "## Gentle AI RDD ownership"
	anchorIndex := strings.Index(content, anchor)
	section := antigravityCLIReviewContractHeading + "\n\n" + antigravityCLIReviewContractMarker + "\n\n#### " + antigravityCLIReviewContractNext + "\n\n"
	if anchorIndex < 0 {
		return strings.TrimRight(content, "\n") + "\n\n" + section, nil
	}
	return content[:anchorIndex] + section + content[anchorIndex:], nil
}

func transpileSatellites(piRoot, outputDir string) error {
	satellites := []string{
		"orchestrator-delegation.md",
		"orchestrator-memory.md",
		"orchestrator-skills.md",
	}

	for _, name := range satellites {
		srcPath := filepath.Join(piRoot, "assets", name)
		rawBytes, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read satellite %s: %w", srcPath, err)
		}

		content := string(rawBytes)
		content = stripDynamicInjections(content)
		content = applyCommonTextMutations(content)
		content = applyAntigravityCLIParentMutations(content)

		if name == "orchestrator-skills.md" {
			content = strings.ReplaceAll(content, ".pi/skills", "~/.gemini/config/skills")
		}

		destPath := filepath.Join(outputDir, name)
		if err := writeDeterministicFile(destPath, content); err != nil {
			return err
		}
	}

	return nil
}

func transpileSupportContracts(piRoot, supportDestDir string) error {
	if err := os.MkdirAll(supportDestDir, 0o755); err != nil {
		return err
	}

	contracts := []string{
		"strict-tdd.md",
		"strict-tdd-verify.md",
	}

	for _, name := range contracts {
		srcPath := filepath.Join(piRoot, "assets", "support", name)
		rawBytes, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read support contract %s: %w", srcPath, err)
		}

		content := string(rawBytes)
		content = stripDynamicInjections(content)
		content = applyCommonTextMutations(content)
		content = applyAntigravityCLIParentMutations(content)

		destPath := filepath.Join(supportDestDir, name)
		if err := writeDeterministicFile(destPath, content); err != nil {
			return err
		}
	}

	return nil
}

func transpileChains(piRoot, chainsDestDir string) error {
	if err := os.MkdirAll(chainsDestDir, 0o755); err != nil {
		return err
	}

	chains := []string{
		"4r-review.chain.md",
	}

	for _, name := range chains {
		srcPath := filepath.Join(piRoot, "assets", "chains", name)
		rawBytes, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read chain %s: %w", srcPath, err)
		}

		content := string(rawBytes)
		content = stripDynamicInjections(content)
		content = stripSection(content, "## Parent preflight transport guard")
		content = applyCommonTextMutations(content)
		content = applyAntigravityCLIParentMutations(content)

		destPath := filepath.Join(chainsDestDir, name)
		if err := writeDeterministicFile(destPath, content); err != nil {
			return err
		}
	}

	return nil
}

func isKnownIgnoredPiTool(tool string) bool {
	t := strings.Trim(tool, `"'`)
	if strings.HasPrefix(t, "*") || strings.HasPrefix(t, `*":`) || strings.Contains(t, "false") || strings.Contains(t, "true") {
		return true
	}
	if strings.HasPrefix(tool, "mem_") {
		// Handled via gentle-ai_engram MCP server
		return true
	}
	if tool == "codegraph" || strings.HasPrefix(tool, "codegraph_") {
		// Handled via gentle-ai_codegraph MCP server
		return true
	}
	if strings.HasPrefix(tool, "gentle_") {
		// Internal Gentle Pi runtime extension
		return true
	}
	return false
}

func extractPiTools(raw string) ([]string, bool) {
	var tools []string
	inFrontmatter := false
	inTools := false
	hasToolsBlock := false

	lines := strings.Split(raw, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				break
			}
		}
		if !inFrontmatter {
			continue
		}

		if strings.HasPrefix(trimmed, "tools:") {
			hasToolsBlock = true
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "tools:"))
			if rest != "" {
				for _, tok := range strings.Split(rest, ",") {
					tok = strings.Trim(strings.TrimSpace(tok), `"'`)
					if tok != "" {
						tools = append(tools, tok)
					}
				}
				inTools = false
			} else {
				inTools = true
			}
			continue
		}

		if inTools {
			if strings.HasPrefix(trimmed, "- ") {
				toolItem := strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")), `"'`)
				if toolItem != "" {
					tools = append(tools, toolItem)
				}
			} else if !strings.HasPrefix(trimmed, "#") && trimmed != "" {
				inTools = false
			}
		}
	}
	return tools, hasToolsBlock
}

func translatePiTools(agentName, raw string) (string, error) {
	piTools, hasToolsBlock := extractPiTools(raw)
	if !hasToolsBlock {
		return "", fmt.Errorf("missing 'tools:' block in frontmatter")
	}
	toolSet := make(map[string]bool)

	for _, pt := range piTools {
		if mapped, ok := piToAntigravityToolMap[pt]; ok {
			for _, m := range mapped {
				toolSet[m] = true
			}
		} else if isKnownIgnoredPiTool(pt) {
			continue
		} else {
			unmappedToolWarnings = append(unmappedToolWarnings, UnmappedToolWarning{
				Agent: agentName,
				Tool:  pt,
			})
		}
	}

	var orderedTools []string
	for _, ct := range canonicalToolOrder {
		if toolSet[ct] {
			orderedTools = append(orderedTools, ct)
			delete(toolSet, ct)
		}
	}

	var extraTools []string
	for t := range toolSet {
		extraTools = append(extraTools, t)
	}
	sort.Strings(extraTools)
	orderedTools = append(orderedTools, extraTools...)

	if len(orderedTools) == 0 {
		return "[]", nil
	}
	return "[" + strings.Join(orderedTools, ", ") + "]", nil
}

func extractDescription(raw string) (string, error) {
	lines := strings.Split(raw, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "description:") {
			desc := strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
			if desc != "" {
				return desc, nil
			}
			return "", fmt.Errorf("description field is empty")
		}
	}
	return "", fmt.Errorf("missing 'description:' in frontmatter")
}

func stripDynamicInjections(content string) string {
	if start := strings.Index(content, "<!-- gentle-ai:codegraph-guidance -->"); start >= 0 {
		if end := strings.Index(content[start:], "<!-- /gentle-ai:codegraph-guidance -->"); end >= 0 {
			content = content[:start] + content[start+end+len("<!-- /gentle-ai:codegraph-guidance -->"):]
		}
	}
	if start := strings.Index(content, "<!-- gentle-ai:agent-language-contract -->"); start >= 0 {
		if end := strings.Index(content[start:], "<!-- /gentle-ai:agent-language-contract -->"); end >= 0 {
			content = content[:start] + content[start+end+len("<!-- /gentle-ai:agent-language-contract -->"):]
		}
	}
	if start := strings.Index(content, "<!-- gentle-ai:remote-authorization -->"); start >= 0 {
		if end := strings.Index(content[start:], "<!-- /gentle-ai:remote-authorization -->"); end >= 0 {
			content = content[:start] + content[start+end+len("<!-- /gentle-ai:remote-authorization -->"):]
		}
	}
	lines := strings.Split(content, "\n")
	var filtered []string
	for _, line := range lines {
		if strings.Contains(line, "CRITICAL FIRST ACTION — Ensure these Engram MCP tools") {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
}

func stripSection(content, sectionHeader string) string {
	idx := strings.Index(content, sectionHeader)
	if idx < 0 {
		return content
	}

	rest := content[idx+len(sectionHeader):]
	nextIdx := strings.Index(rest, "\n## ")
	if nextIdx >= 0 {
		return content[:idx] + rest[nextIdx+1:]
	}
	return content[:idx]
}

const antigravityCLIReviewerOutputContract = `## Output contract

Return one provider-bound reviewer result and no prose. Required top-level fields are ` + "`subject_hash`" + `, ` + "`inspection`" + `, ` + "`findings`" + `, and ` + "`evidence`" + `; the optional ` + "`lens`" + ` field may name this selected lens. Emit no other top-level fields.

Run this selected lens exactly once against the supplied provider-bound immutable candidate. Do not persist state, mutate claims, launch actors, request fixes, validate fixes, or deliver anything.

Inspect only that candidate. Set ` + "`inspection.status`" + ` to ` + "`completed`" + ` only when every changed path in the supplied manifest was inspected, and set ` + "`paths`" + ` to the complete unique unordered set. If inspection cannot be completed, use ` + "`inspection.status`" + ` ` + "`unavailable`" + `, an empty ` + "`paths`" + ` array, and a non-empty ` + "`inspection.reason`" + `; never claim completion.

Every candidate finding must include exact location, severity, claim, ` + "`evidence_class`" + ` (` + "`deterministic | inferential | insufficient`" + `), ` + "`causal_disposition`" + ` (` + "`introduced | behavior-activated | worsened | pre-existing | base-only | unknown`" + `), and ` + "`proof_refs`" + `. Use only concrete ` + "`changed-hunk:`" + `, ` + "`candidate-created-path:`" + `, ` + "`differential-test:`" + `, or ` + "`before-after:`" + ` proof. Use ` + "`BLOCKER | CRITICAL | WARNING | SUGGESTION`" + `. BLOCKER/CRITICAL findings need changed-hunk, candidate-created-path, differential-test, or before/after proof of introduced, behavior-activated, or worsened behavior. Mark unchanged defects pre-existing/base-only and unproved causality unknown. Style or suspicion is not a finding.

Return exactly one JSON object with this shape:

` + "```json" + `
{
  "subject_hash": "<artifact_subject.subject_hash>",
  "inspection": {
    "status": "completed",
    "paths": ["<complete unique unordered set>"]
  },
  "findings": [
    {
      "location": "path:line or path:start-end",
      "severity": "CRITICAL",
      "claim": "observable incorrect behavior",
      "evidence_class": "deterministic",
      "causal_disposition": "introduced",
      "proof_refs": ["concrete proof"]
    }
  ],
  "evidence": ["what was inspected"]
}
` + "```" + `

Copy ` + "`subject_hash`" + ` from ` + "`artifact_subject.subject_hash`" + `; never compute or invent it. If clean and inspection is completed, return ` + "`\"findings\": []`" + ` and one concrete ` + "`\"evidence\"`" + ` entry. If inspection is unavailable, return ` + "`\"findings\": []`" + `, the concrete ` + "`inspection.reason`" + `, and non-empty ` + "`evidence`" + `. Do not emit ` + "`summary`" + `, ` + "`skill_resolution`" + `, prose, or orchestration metadata.

`

func alignAntigravityCLIReviewerContract(agentName, body string) (string, error) {
	if !strings.HasPrefix(agentName, "review-") {
		return body, nil
	}
	start := strings.Index(body, "## Output contract")
	if start < 0 {
		return "", fmt.Errorf("reviewer %s is missing its output contract", agentName)
	}
	remainder := body[start:]
	tail := strings.Index(remainder, "Only candidate-caused")
	if tail < 0 {
		return "", fmt.Errorf("reviewer %s output contract has no terminal candidate-caused clause", agentName)
	}
	return body[:start] + antigravityCLIReviewerOutputContract + remainder[tail:], nil
}

func mutateAgentRuntimeInstructions(content string) string {
	replacements := []struct {
		oldVal string
		newVal string
	}{
		{
			"> Manual/compat-lane only: the provider host-relay capture path never loads this agent definition; native lens capture materializes the Go-issued opaque prompt through the gentle-pi host relay.",
			"> The parent delegates this read-only review lens with `invoke_subagent`; the native invocation returns the subagent's scoped result to the parent.",
		},
		{
			"Use `subagent_run` to delegate work and return the result.",
			"Use `invoke_subagent` to delegate work and `send_message` to return the result.",
		},
		{
			"The `subagent_run` background runtime returns immediately.",
			"Antigravity CLI keeps delegated work active until the subagent returns a result.",
		},
		{
			"Never delegate or invoke `subagent_*` tools.",
			"Never delegate with `invoke_subagent` or `send_message`.",
		},
		{
			"or any `subagent_*` tool.",
			"or any `invoke_subagent` or `send_message` call.",
		},
		{
			"(gentle-pi#661, RDD-aware pilot)",
			"(Gentle AI RDD-aware workflow)",
		},
		{
			"intentionally unsupported in gentle-pi until `lib/openspec-deltas.ts` implements executable rename semantics",
			"intentionally unsupported by the current Gentle AI workflow",
		},
		{
			"Do not use `assets/support/...` as a runtime path; that is only the package source path before installation.",
			"Use an installed support path at runtime; never use the package source tree as a runtime path.",
		},
		{
			"~/.pi/agent/gentle-ai/",
			"~/.gemini/config/plugins/gentle-ai/",
		},
		{
			".pi/gentle-ai/",
			".gemini/gentle-ai/",
		},
	}

	for _, replacement := range replacements {
		content = strings.ReplaceAll(content, replacement.oldVal, replacement.newVal)
	}

	return content
}

func applyAntigravityCLIParentMutations(content string) string {
	content = strings.ReplaceAll(content,
		"If neither worker definition exists, fall back to the native `Agent` even when `subagent_*` tools are available.",
		"If neither worker definition exists, delegate the bounded write with Antigravity's native `invoke_subagent`.")

	content = stripSection(content, "## Pi Runtime Overlays")
	content = stripAntigravityCLIMarkedSection(content, "<!-- gentle-pi:background-subagents -->", "<!-- /gentle-pi:background-subagents -->")
	content = stripAntigravityCLISectionUntil(content, "#### Pi Trigger Runtime Bindings", "### Work Routing Ladder")
	content = stripAntigravityCLISectionUntil(content, "#### Pi Subagent Model Routing", "Default balanced pattern for bounded implementation:")
	content = stripAntigravityCLISectionUntil(content, "## Pi Delegation Bindings", "### Canonical Lightweight Workflows")
	content = stripAntigravityCLISectionUntil(content, "## Model Assignments", "## Judgment Day fix routing")

	content = replaceAntigravityCLILineContaining(content, "- Native route: For every strictly closed single-select envelope", "- Native route: Use `ask_question` when the complete choice envelope is representable in one native call. Preserve every provider-owned label, description, effect, and answer token exactly; never synthesize or append a continuation. If the native question UI is unavailable or the envelope is unrepresentable, use the complete plain-chat fallback below and stop.")
	content = replaceAntigravityCLILineContaining(content, "Mandatory Delegation Triggers — once fired", "Mandatory Delegation Triggers — once fired, delegate through Antigravity's native `invoke_subagent` and use `send_message` for follow-up guidance:")
	content = replaceAntigravityCLILineContaining(content, "This package injects the mirrored provider-bundle", "This package injects the canonical native review execution contract into the Antigravity parent prompt. Follow that contract exactly; this package owns no separate review lifecycle route.")
	content = replaceAntigravityCLILineContaining(content, "- An eligible interactive Pi host", "- For a provider-owned consent envelope, use `ask_question` only when the complete envelope is representable; otherwise relay it losslessly and stop. Never infer a continuation.")
	content = replaceAntigravityCLILineContaining(content, "Use the configured subagent runtime when available", "Use Antigravity's native subagent runtime. Prefer `invoke_subagent` for a fresh delegated context and `send_message` for follow-up guidance; do not invent a background or model-routing contract.")
	content = replaceAntigravityCLILineContaining(content, "Delegate generic non-SDD verification", "Delegate generic verification with `invoke_subagent` to `gentle-ai-verify`; keep only a small read-only check inline. The parent owns the native review lifecycle and all delivery decisions.")
	content = replaceAntigravityCLILineContaining(content, "For delegation other than bounded multi-file writes", "For delegation other than bounded multi-file writes, use the generic fallback: if `invoke_subagent` or `send_message` is unavailable, stop and explain the blocker. The delegation trigger remains mandatory; do not silently continue inline.")
	content = replaceAntigravityCLILineContaining(content, "Never search for, request, or invoke review tools", "The parent owns candidate review disposition and lifecycle. Never search for, request, or invoke review tools.")
	content = replaceAntigravityCLILineContaining(content, "When RDD is enabled, first use native candidate risk assessment", "When RDD is enabled, follow the projected `#### Review Execution Contract` exactly: use only the native review operations returned by that contract, preserve provider-owned bindings, and never infer authority or delivery from model judgment or task size. When RDD is disabled, ordinary checks remain.")
	content = replaceAntigravityCLILineContaining(content, "5. **Verification rule**", "5. **Verification rule**: executing or delegating verification commands routes to `gentle-ai-verify`; only a small read-only check stays inline.")
	content = replaceAntigravityCLILineContaining(content, "{{GENTLE_PI_BACKGROUND_POLICY}}", "Use Antigravity's native `invoke_subagent` and keep delegated work foreground until the result returns")
	content = replaceAntigravityCLILineContaining(content, "| Native `nextRecommended` | Pi executor |", "| Native `nextRecommended` | Antigravity executor |")
	content = strings.ReplaceAll(content, "`fetch_content`", "`read_url_content`")
	content = strings.ReplaceAll(content, "`web_search`", "`search_web`")
	content = strings.ReplaceAll(content, "`source_check`", "`read_url_content`")
	content = strings.ReplaceAll(content, "`get_search_content`", "`read_url_content`")
	content = strings.ReplaceAll(content, "Do not use `~/.gemini/config/plugins/gentle-ai/support/...` as a runtime path; that is only the package source path before installation.", "Do not use a package source path as a runtime path.")

	content = strings.ReplaceAll(content, "parent Pi session", "parent Antigravity session")
	content = strings.ReplaceAll(content, "Pi session", "Antigravity session")
	content = strings.ReplaceAll(content, "Within one Pi runner", "Within one Antigravity session")
	content = strings.ReplaceAll(content, "not a Pi schema", "not a local schema")
	content = strings.ReplaceAll(content, "Pi subagents", "Antigravity subagents")
	content = strings.ReplaceAll(content, "Pi runtime", "Antigravity runtime")
	content = strings.ReplaceAll(content, "Pi's concrete runtime", "Antigravity's concrete runtime")
	content = strings.ReplaceAll(content, "On Pi", "On Antigravity")
	content = strings.ReplaceAll(content, "on Pi", "on Antigravity")
	content = strings.ReplaceAll(content, "a Antigravity session", "an Antigravity session")
	content = strings.ReplaceAll(content, "`subagent_run`", "`invoke_subagent`")
	content = strings.ReplaceAll(content, "subagent_run", "invoke_subagent")
	content = strings.ReplaceAll(content, "`subagent_*`", "`invoke_subagent` or `send_message`")
	content = strings.ReplaceAll(content, "subagent_*", "invoke_subagent or send_message")
	content = strings.ReplaceAll(content, " even when `invoke_subagent` or `send_message` tools are available", "")
	content = strings.ReplaceAll(content, " even when invoke_subagent or send_message tools are available", "")
	content = strings.ReplaceAll(content, "`ask_user_choice`", "`ask_question`")
	content = strings.ReplaceAll(content, "ask_user_choice", "ask_question")
	content = strings.ReplaceAll(content, "`ask_user_question`", "`ask_question`")
	content = strings.ReplaceAll(content, "ask_user_question", "ask_question")
	content = strings.ReplaceAll(content, "`gentle_review`", "the native review lifecycle")
	content = strings.ReplaceAll(content, "gentle_review", "the native review lifecycle")
	content = strings.ReplaceAll(content, "{{GENTLE_PI_BACKGROUND_POLICY}}", "Use Antigravity's native `invoke_subagent` and keep delegated work foreground until the result returns")
	content = strings.ReplaceAll(content, ".pi/skills", "~/.gemini/config/skills")
	content = strings.ReplaceAll(content, "~/.pi/agent/gentle-ai/", "~/.gemini/config/plugins/gentle-ai/")
	content = strings.ReplaceAll(content, "~/.pi/agent/", "~/.gemini/config/plugins/gentle-ai/")
	content = strings.ReplaceAll(content, ".pi/gentle-ai/", ".gemini/gentle-ai/")
	content = strings.ReplaceAll(content, ".pi/", "~/.gemini/")
	content = strings.ReplaceAll(content, "fall back to the native `Agent`", "use `invoke_subagent`")
	content = strings.ReplaceAll(content, "fall back to Pi's native `Agent`", "use `invoke_subagent`")
	content = strings.ReplaceAll(content, "fall back to Pi's native `Agent` tool", "use `invoke_subagent`")
	content = strings.ReplaceAll(content, "native `Agent` fallback", "`invoke_subagent` fallback")
	content = strings.ReplaceAll(content, "native `Agent` tool", "`invoke_subagent`")
	content = strings.ReplaceAll(content, "Pi's native `Agent`", "Antigravity's `invoke_subagent`")
	content = strings.ReplaceAll(content, "Pi native `Agent`", "Antigravity `invoke_subagent`")
	content = strings.ReplaceAll(content, "native `Agent`", "`invoke_subagent`")
	content = strings.ReplaceAll(content, "Pi's native", "Antigravity's native")
	content = strings.ReplaceAll(content, "background or model-routing", "an alternate delegation or model-routing")
	content = strings.ReplaceAll(content, "a an alternate", "an alternate")
	content = strings.ReplaceAll(content, "gentle-pi", "Gentle AI")
	content = rewriteAntigravityCLIUnsupportedSlashCommands(content)
	return strings.TrimSpace(content)
}

func rewriteAntigravityCLIUnsupportedSlashCommands(content string) string {
	for _, replacement := range []struct{ old, new string }{
		{"Before handling any `/sdd-*` command", "Before handling an SDD request"},
		{"Read this file before handling `/sdd-*`,", "Read this file before handling an SDD request,"},
		{"slash SDD flows and `/gentle-sdd-init` run preflight automatically", "explicit SDD requests and the installed `sdd-init` skill run preflight automatically"},
		{"run `/gentle-sdd-init` if available", "run the installed `sdd-init` skill if available"},
		{"`/gentle-sdd-init` never writes that file", "the installed `sdd-init` skill never writes that file"},
		{"Never re-trigger `/gentle-sdd-init`", "Never re-trigger the installed `sdd-init` skill"},
		{"runs or reuses `/gentle:sdd-preflight`", "runs or reuses the canonical session SDD preflight"},
		{"`/gentle:sdd-preflight` is the explicit preflight command", "The canonical session SDD preflight is the explicit preflight step"},
		{"invokes `/gentle-sdd-new`, `/gentle-sdd-ff`, or `/gentle-sdd-continue`", "starts the native SDD flow or invokes `gentle-ai sdd-continue`"},
	} {
		content = strings.ReplaceAll(content, replacement.old, replacement.new)
	}

	for _, replacement := range []struct{ old, new string }{
		{"`/gentle-sdd-continue`", "`gentle-ai sdd-continue`"},
		{"`/gentle-sdd-status`", "`gentle-ai sdd-status`"},
		{"`/gentle-sdd-init`", "the installed `sdd-init` skill"},
		{"`/gentle-sdd-new`", "the native SDD start flow"},
		{"`/gentle-sdd-ff`", "the native SDD fast-forward flow"},
		{"`/gentle:sdd-preflight`", "the canonical session SDD preflight"},
		{"`/gentle-sdd-status [change]`", "`gentle-ai sdd-status [change]`"},
		{"`/sdd-status [change]`", "`gentle-ai sdd-status [change]`"},
		{"`/sdd-status`", "`gentle-ai sdd-status`"},
		{"`/sdd-continue`", "`gentle-ai sdd-continue`"},
		{"`/sdd-init`", "the installed `sdd-init` skill"},
		{"`/sdd-new`", "the native SDD start flow"},
		{"`/sdd-ff`", "the native SDD fast-forward flow"},
		{"`/sdd-*`", "an SDD request"},
	} {
		content = strings.ReplaceAll(content, replacement.old, replacement.new)
	}
	return content
}

func replaceAntigravityCLILineContaining(content, needle, replacement string) string {
	lines := strings.Split(content, "\n")
	for index, line := range lines {
		if strings.Contains(line, needle) {
			lines[index] = replacement
		}
	}
	return strings.Join(lines, "\n")
}

func stripAntigravityCLIMarkedSection(content, startMarker, endMarker string) string {
	start := strings.Index(content, startMarker)
	if start < 0 {
		return content
	}
	end := strings.Index(content[start+len(startMarker):], endMarker)
	if end < 0 {
		return content
	}
	end += start + len(startMarker) + len(endMarker)
	return strings.TrimRight(content[:start], "\n") + "\n\n" + strings.TrimLeft(content[end:], "\n")
}

func stripAntigravityCLISectionUntil(content, heading, nextHeading string) string {
	start := strings.Index(content, heading)
	if start < 0 {
		return content
	}
	relative := strings.Index(content[start+len(heading):], nextHeading)
	if relative < 0 {
		return content
	}
	next := start + len(heading) + relative
	return strings.TrimRight(content[:start], "\n") + "\n\n" + strings.TrimLeft(content[next:], "\n")
}

func applyCommonTextMutations(s string) string {
	replacements := []struct {
		oldVal string
		newVal string
	}{
		{"parent Pi session", "parent Antigravity session"},
		{"Pi session", "Antigravity session"},
		{"Pi runtime", "Antigravity runtime"},
		{"Gentle Pi", "Gentle AI"},
		{"{{GENTLE_PI_ASSETS_ROOT}}", "~/.gemini/config/plugins/gentle-ai"},
		{"assets/support/", "~/.gemini/config/plugins/gentle-ai/support/"},
		{"assets/agents/*.md", "~/.gemini/config/plugins/gentle-ai/agents/*/agent.md"},
		{"assets/orchestrator-delegation.md", "~/.gemini/config/plugins/gentle-ai/orchestrator-delegation.md"},
		{"assets/orchestrator-memory.md", "~/.gemini/config/plugins/gentle-ai/orchestrator-memory.md"},
		{"assets/orchestrator-skills.md", "~/.gemini/config/plugins/gentle-ai/orchestrator-skills.md"},
		{"assets/orchestrator.md", "~/.gemini/config/plugins/gentle-ai/orchestrator.md"},
		{"assets/chains/", "~/.gemini/config/plugins/gentle-ai/chains/"},
	}

	for _, r := range replacements {
		s = strings.ReplaceAll(s, r.oldVal, r.newVal)
	}

	return s
}

func writeDeterministicFile(path, content string) error {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return fmt.Errorf("refusing to write empty content to %s", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %s: %w", path, err)
	}

	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.TrimSpace(normalized) + "\n"

	if err := os.WriteFile(path, []byte(normalized), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
