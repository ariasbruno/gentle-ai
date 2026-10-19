package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	antigravitycliagent "github.com/gentleman-programming/gentle-ai/v4/internal/agents/antigravitycli"
)

const (
	defaultRemoteRepo = "https://github.com/Gentleman-Programming/gentle-shell"
	defaultOutputDir  = "internal/assets/antigravitycli"

	antigravityCLIReviewContractHeading = "#### Review Execution Contract"
	antigravityCLIReviewContractNext    = "Cost and Context Balance"

	provenanceFileName = "PROVENANCE.json"
)

// Provenance records the exact source reference used for transpilation.
type Provenance struct {
	SourceRepo   string `json:"source_repo"`
	SourceCommit string `json:"source_commit"`
}

// UnmappedToolWarning records a Pi tool that lacked a native Antigravity CLI translation.
type UnmappedToolWarning struct {
	Agent string
	Tool  string
}

// The native core keeps mandatory policy and phase loading always on while
// the full portable procedures remain in the pinned Shell satellites.
//
//go:embed orchestrator-core.md
var antigravityCLIOrchestratorCore string

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
	sourceDir := flag.String("source-dir", "", "Path to a clean local source clone (including untracked/ignored files; if empty, clones remote)")
	sourceRepo := flag.String("source-repo", defaultRemoteRepo, "Remote git URL for gentle-pi")
	outputDir := flag.String("output-dir", defaultOutputDir, "Destination directory for generated assets")
	strict := flag.Bool("strict", false, "Reject unmapped tools during generation or --check")
	check := flag.Bool("check", false, "Verify the committed bundle matches a fresh transpilation from the pinned provenance (drift gate)")
	flag.Parse()

	if *check {
		if err := runCheckGate(*outputDir, *sourceDir, *sourceRepo, *strict); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Println("Drift check passed: committed bundle matches provenance-pinned transpilation.")
		return
	}

	piRoot, cleanup, provenance, err := resolveSourceRootWithProvenance(*sourceDir, *sourceRepo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving source repository: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	if err := transpileAll(piRoot, *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Transpilation failed: %v\n", err)
		os.Exit(1)
	}

	if err := writeProvenance(*outputDir, provenance); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write provenance: %v\n", err)
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

func resolveSourceRootWithProvenance(localDir, remoteRepo string) (string, func(), Provenance, error) {
	var provenance Provenance
	provenance.SourceRepo = remoteRepo

	if localDir != "" {
		if info, err := os.Stat(localDir); err == nil && info.IsDir() {
			sha, err := getSourceHEAD(localDir)
			if err != nil {
				return "", func() {}, provenance, fmt.Errorf("get git HEAD for %s: %w", localDir, err)
			}
			provenance.SourceCommit = sha
			return localDir, func() {}, provenance, nil
		}
		return "", func() {}, provenance, fmt.Errorf("specified --source-dir %q does not exist or is not a directory", localDir)
	}

	// Check environment variables
	for _, envKey := range []string{"GENTLE_SHELL_DIR", "GENTLE_PI_DIR"} {
		if val := os.Getenv(envKey); val != "" {
			if info, err := os.Stat(val); err == nil && info.IsDir() {
				sha, err := getSourceHEAD(val)
				if err != nil {
					return "", func() {}, provenance, fmt.Errorf("get git HEAD for %s: %w", val, err)
				}
				provenance.SourceCommit = sha
				return val, func() {}, provenance, nil
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
			sha, err := getSourceHEAD(candidate)
			if err != nil {
				return "", func() {}, provenance, fmt.Errorf("get git HEAD for %s: %w", candidate, err)
			}
			provenance.SourceCommit = sha
			return candidate, func() {}, provenance, nil
		}
	}

	// Clone from remote into a temporary directory
	tempDir, err := os.MkdirTemp("", "gentle-pi-source-*")
	if err != nil {
		return "", func() {}, provenance, fmt.Errorf("create temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }

	cmd := exec.Command("git", "clone", "--depth", "1", remoteRepo, tempDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		cleanup()
		return "", func() {}, provenance, fmt.Errorf("git clone %s: %w", remoteRepo, err)
	}

	sha, err := getGitHEAD(tempDir)
	if err != nil {
		cleanup()
		return "", func() {}, provenance, fmt.Errorf("get git HEAD for cloned repo: %w", err)
	}
	provenance.SourceCommit = sha

	return tempDir, cleanup, provenance, nil
}

func getSourceHEAD(dir string) (string, error) {
	cmd := exec.Command("git", "status", "--porcelain=v1", "--untracked-files=all", "--ignored")
	cmd.Dir = dir
	status, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("check source worktree: %w", err)
	}
	if len(status) != 0 {
		return "", errors.New("source checkout must be clean (including untracked and ignored files)")
	}
	return getGitHEAD(dir)
}

func getGitHEAD(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
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

	// 7. Transpile Skills (all gentle-shell skills + _shared + gentle-ai harness)
	if err := transpileSkills(piRoot, filepath.Join(outputDir, "skills")); err != nil {
		return fmt.Errorf("transpile skills: %w", err)
	}

	// 8. Make every emitted skill's identity its folder, and carry the same
	// rename through every reference to it in the generated tree.
	if err := normalizeEmittedSkillNames(outputDir); err != nil {
		return fmt.Errorf("normalize emitted skill names: %w", err)
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

func writeProvenance(outputDir string, provenance Provenance) error {
	bytes, err := json.MarshalIndent(provenance, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal provenance: %w", err)
	}
	return writeDeterministicFile(filepath.Join(outputDir, provenanceFileName), string(bytes))
}

func readProvenance(outputDir string) (Provenance, error) {
	var prov Provenance
	path := filepath.Join(outputDir, provenanceFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return prov, fmt.Errorf("read provenance: %w", err)
	}
	if err := json.Unmarshal(data, &prov); err != nil {
		return prov, fmt.Errorf("parse provenance: %w", err)
	}
	return prov, nil
}

// getCommittedTree resolves the tree the drift gate must compare against:
// the version of outputDir COMMITTED at git HEAD, extracted via git archive.
// A dirty worktree must not make the local --check pass trivially — in CI
// the checkout is clean so HEAD and disk coincide. When outputDir is not
// inside a usable git repository (or extraction fails), it falls back to the
// on-disk tree with a warning so non-repo invocations keep working.
func getCommittedTree(outputDir string) (string, Provenance, func(), error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = outputDir
	output, err := cmd.Output()
	if err != nil {
		prov, provErr := readProvenance(outputDir)
		if provErr != nil {
			return "", Provenance{}, nil, provErr
		}
		fmt.Fprintf(os.Stderr, "[WARN] --check: output dir not in a git repo; comparing against on-disk tree\n")
		return outputDir, prov, func() {}, nil
	}
	gitRoot := strings.TrimSpace(string(output))

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		prov, _ := readProvenance(outputDir)
		fmt.Fprintf(os.Stderr, "[WARN] --check: cannot resolve absolute path (%v); comparing against on-disk tree\n", err)
		return outputDir, prov, func() {}, nil
	}
	absGitRoot, err := filepath.Abs(gitRoot)
	if err != nil {
		prov, _ := readProvenance(outputDir)
		fmt.Fprintf(os.Stderr, "[WARN] --check: cannot resolve git root (%v); comparing against on-disk tree\n", err)
		return outputDir, prov, func() {}, nil
	}
	relPath, err := filepath.Rel(absGitRoot, absOutputDir)
	if err != nil || strings.HasPrefix(relPath, "..") {
		prov, _ := readProvenance(outputDir)
		fmt.Fprintf(os.Stderr, "[WARN] --check: output dir not inside git repo (%v); comparing against on-disk tree\n", err)
		return outputDir, prov, func() {}, nil
	}

	tempDir, err := os.MkdirTemp("", "agy-committed-*")
	if err != nil {
		prov, _ := readProvenance(outputDir)
		fmt.Fprintf(os.Stderr, "[WARN] --check: cannot create temp dir for git archive (%v); comparing against on-disk tree\n", err)
		return outputDir, prov, func() {}, nil
	}

	// git archive HEAD -- <relPath> | tar -x -C <tempDir>
	archiveCmd := exec.Command("git", "archive", "HEAD", "--", relPath)
	archiveCmd.Dir = gitRoot
	extractCmd := exec.Command("tar", "-x", "-C", tempDir)
	extractCmd.Stdin, _ = archiveCmd.StdoutPipe()
	extractCmd.Stdout = os.Stdout
	extractCmd.Stderr = os.Stderr
	if err := extractCmd.Start(); err != nil {
		os.RemoveAll(tempDir)
		prov, _ := readProvenance(outputDir)
		fmt.Fprintf(os.Stderr, "[WARN] --check: git archive pipe failed (%v); comparing against on-disk tree\n", err)
		return outputDir, prov, func() {}, nil
	}
	if err := archiveCmd.Run(); err != nil {
		extractCmd.Wait()
		os.RemoveAll(tempDir)
		prov, _ := readProvenance(outputDir)
		fmt.Fprintf(os.Stderr, "[WARN] --check: git archive failed for path %q (%v); comparing against on-disk tree\n", relPath, err)
		return outputDir, prov, func() {}, nil
	}
	if err := extractCmd.Wait(); err != nil {
		os.RemoveAll(tempDir)
		prov, _ := readProvenance(outputDir)
		fmt.Fprintf(os.Stderr, "[WARN] --check: tar extract failed (%v); comparing against on-disk tree\n", err)
		return outputDir, prov, func() {}, nil
	}

	// The extracted tree nests under the repo-relative path.
	committedExtractedDir := filepath.Join(tempDir, relPath)
	if _, err := os.Stat(committedExtractedDir); err != nil {
		os.RemoveAll(tempDir)
		prov, _ := readProvenance(outputDir)
		fmt.Fprintf(os.Stderr, "[WARN] --check: extracted path %q not found (%v); comparing against on-disk tree\n", relPath, err)
		return outputDir, prov, func() {}, nil
	}

	// The pin being verified is HEAD's, so provenance comes from the
	// committed tree, not from the worktree.
	prov, err := readProvenance(committedExtractedDir)
	if err != nil {
		os.RemoveAll(tempDir)
		prov, provErr := readProvenance(outputDir)
		if provErr != nil {
			return "", Provenance{}, nil, provErr
		}
		fmt.Fprintf(os.Stderr, "[WARN] --check: cannot read provenance from committed tree (%v); comparing against on-disk tree\n", err)
		return outputDir, prov, func() {}, nil
	}

	cleanup := func() { _ = os.RemoveAll(tempDir) }
	return committedExtractedDir, prov, cleanup, nil
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
	// Agents whose Pi source includes MCP-backed tools (codegraph, mem_save)
	// need MCP access in agy: subagents inherit MCP servers only with
	// enable_mcp_tools: true (agy 1.2.3 fix). In agy these tools are exposed
	// via call_mcp_tool with the server prefix (gentle-ai_codegraph,
	// gentle-ai_engram).
	if agentName == "gentle-ai-worker" {
		// The bounded writer consumes parent-selected context, not persistent
		// memory or arbitrary MCP surfaces inherited from the plugin.
		extraFrontmatter += "enable_mcp_tools: false\n"
	} else if strings.Contains(raw, "codegraph") || strings.Contains(raw, "mem_save") {
		extraFrontmatter += "enable_mcp_tools: true\n"
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

	return frontmatter + "\n" + nativeAgentPhaseGuidance(agentName) + trimmedBody + "\n", nil
}

func nativeAgentPhaseGuidance(agentName string) string {
	var action, module string
	switch agentName {
	case "gentle-ai-worker":
		action, module = "Before the first implementation edit", "strict-tdd.md"
	case "gentle-ai-verify":
		action, module = "Before verification", "strict-tdd-verify.md"
	default:
		return ""
	}
	return "## Native phase context\n\n" + action + ", read `~/.gemini/config/plugins/gentle-ai/rules/" + module + "` (global) or `<workspace>/.agents/plugins/gentle-ai/rules/" + module + "` (workspace), using the parent's effective plugin root. Do not assume access to parent history, persona, rules or authorization. Use only exact authorized scope and checks. Applicable deterministic behavior changes require observed RED before implementation and observed GREEN afterward; justify non-applicability and run proportionate checks otherwise.\n\n"
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

	// Validate the source contract shape above, but do not concatenate the
	// monolithic Pi prompt with Go's growing provider contract. AGY's native
	// core routes to the fully transpiled phase rules and retains mandatory
	// safety, ODD and verification obligations within its per-rule byte cap.
	core, err := ensureAntigravityCLIReviewContract(antigravityCLIOrchestratorCore)
	if err != nil {
		return fmt.Errorf("prepare native Antigravity core: %w", err)
	}
	return writeDeterministicFile(filepath.Join(outputDir, "orchestrator.md"), core)
}

func ensureAntigravityCLIReviewContract(content string) (string, error) {
	count := strings.Count(content, antigravityCLIReviewContractHeading)
	if count > 1 {
		return "", fmt.Errorf("source contains %d review-contract headings, want exactly one", count)
	}
	if count == 1 {
		if !strings.Contains(content, antigravitycliagent.ReviewContractInsertMarker) {
			return "", fmt.Errorf("review-contract heading is missing deterministic marker %q", antigravitycliagent.ReviewContractInsertMarker)
		}
		return content, nil
	}
	anchor := "## Gentle AI RDD ownership"
	anchorIndex := strings.Index(content, anchor)
	section := antigravityCLIReviewContractHeading + "\n\n" + antigravitycliagent.ReviewContractInsertMarker + "\n\n#### " + antigravityCLIReviewContractNext + "\n\n"
	if anchorIndex < 0 {
		return strings.TrimRight(content, "\n") + "\n\n" + section, nil
	}
	return content[:anchorIndex] + section + content[anchorIndex:], nil
}

// transpileSkills consumes ALL of gentle-shell/skills (every skill dir + _shared fragments + gentle-ai harness)
// and emits them to <output>/skills/<name>/... preserving each skill's file layout (SKILL.md plus any extra files).
// Runs the SAME transformation pipeline already used for the other assets.
func transpileSkills(piRoot, skillsDestDir string) error {
	piSkillsDir := filepath.Join(piRoot, "skills")
	if _, err := os.Stat(piSkillsDir); err != nil {
		return fmt.Errorf("source skills directory %s does not exist: %w", piSkillsDir, err)
	}

	entries, err := os.ReadDir(piSkillsDir)
	if err != nil {
		return fmt.Errorf("read pi skills dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillName := entry.Name()

		// The review-ledger-contract.md shared fragment describes Pi's runtime
		// review transport (host relay, SessionManager, globalThis[Symbol.for]
		// registry, print-mode pi subprocess) — concepts that do not exist in
		// Antigravity CLI. agy receives its review protocol from the
		// orchestrator rule's review contract fill (the internal catalog's
		// review-ledger-contract.md, adapted per transport), so shipping this
		// Pi-specific fragment in the plugin is actively misleading.
		if skillName == "_shared" {
			// Still transpile _shared, but skip the Pi-runtime review ledger.
			if err := transpileSharedDirExcludingReviewLedger(filepath.Join(piSkillsDir, skillName), filepath.Join(skillsDestDir, skillName)); err != nil {
				return fmt.Errorf("transpile skill %s: %w", skillName, err)
			}
			continue
		}

		srcSkillDir := filepath.Join(piSkillsDir, skillName)
		destSkillDir := filepath.Join(skillsDestDir, skillName)

		if err := copyAndTransformSkillDir(srcSkillDir, destSkillDir, skillName); err != nil {
			return fmt.Errorf("transpile skill %s: %w", skillName, err)
		}
	}

	return nil
}

// transpileSharedDirExcludingReviewLedger copies the _shared directory minus
// the Pi-runtime review-ledger-contract.md, which describes Pi's extension
// system (host relay, SessionManager, globalThis[Symbol.for]) and has no
// meaning for Antigravity CLI.
func transpileSharedDirExcludingReviewLedger(srcDir, destDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Skip the Pi-runtime review ledger — its content is Pi's extension
		// protocol, not agy's. agy gets its review protocol from the
		// orchestrator rule fill (internal catalog).
		if d.Name() == "review-ledger-contract.md" {
			return nil
		}
		return copyAndTransformFile(path, filepath.Join(destDir, relPath))
	})
}

// copyAndTransformFile copies one file, applying the skill transformation
// pipeline to markdown files and copying everything else verbatim.
func copyAndTransformFile(srcPath, destPath string) error {
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", srcPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	if strings.HasSuffix(strings.ToLower(filepath.Base(srcPath)), ".md") {
		transformed := transformSkillContent(string(content), "")
		return writeDeterministicFile(destPath, transformed)
	}
	return os.WriteFile(destPath, content, 0o644)
}

// copyAndTransformSkillDir recursively copies a skill directory, applying transformations to markdown files.
func copyAndTransformSkillDir(srcDir, destDir, skillName string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		// Apply transformations to markdown files (SKILL.md and any .md files in references/)
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			transformed := transformSkillContent(string(content), skillName)
			return writeDeterministicFile(destPath, transformed)
		}

		// Copy non-markdown files as-is
		return os.WriteFile(destPath, content, 0o644)
	})
}

// transformSkillContent applies the standard transpilation pipeline to skill markdown content.
func transformSkillContent(content, skillName string) string {
	content = stripDynamicInjections(content)
	content = applyCommonTextMutations(content)
	content = applyAntigravityCLIParentMutations(content)
	// Shell 4.1.0 still ships these historical workflow references. The Go
	// runtime has retired that workflow; preserve the portable commit/registry
	// policy without inventing replacement CLI commands.
	content = strings.ReplaceAll(content, "`sdd-tasks`", "`odd/tasks/<feature-name>.md`")
	content = strings.ReplaceAll(content, "`sdd-*`, ", "")
	content = strings.ReplaceAll(content, "regardless of SDD persistence mode", "when indexing installed skills")
	content = strings.ReplaceAll(content, "SDD", "ODD")
	return content
}

// normalizeEmittedSkillNames makes every emitted skill's identity its folder
// and rewrites the references to it.
//
// Gentle Shell ships its skills with a namespaced frontmatter name
// (`gentle-ai-<folder>`) inside a `<folder>` directory, because Pi's registry
// shares one namespace with the user's own skills. Antigravity CLI's plugin
// loader requires a skill's name to match its folder, and rejects the plugin
// under `--strict` when it does not, so the emitted plugin carries the folder
// as the skill name. Every reference to a renamed skill is rewritten with it,
// so the plugin never names a skill it does not ship under that name.
func normalizeEmittedSkillNames(outputDir string) error {
	skillsDir := filepath.Join(outputDir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return fmt.Errorf("read emitted skills dir: %w", err)
	}

	renames := map[string]string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(skillsDir, entry.Name(), "SKILL.md"))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read emitted skill %s: %w", entry.Name(), err)
		}
		name := frontmatterName(string(raw))
		if name == "" || name == entry.Name() {
			continue
		}
		renames[name] = entry.Name()
	}
	if len(renames) == 0 {
		return nil
	}

	oldNames := make([]string, 0, len(renames))
	for old := range renames {
		oldNames = append(oldNames, old)
	}
	sort.Slice(oldNames, func(i, j int) bool {
		if len(oldNames[i]) != len(oldNames[j]) {
			return len(oldNames[i]) > len(oldNames[j])
		}
		return oldNames[i] < oldNames[j]
	})

	return filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !isEmittedTextAsset(path) {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		content := string(raw)
		updated := content
		for _, old := range oldNames {
			updated = replaceSkillName(updated, old, renames[old])
		}
		if updated == content {
			return nil
		}
		return writeDeterministicFile(path, updated)
	})
}

// frontmatterName returns the `name` declared in a document's leading
// frontmatter block, or an empty string when it declares none.
func frontmatterName(content string) string {
	rest, ok := strings.CutPrefix(content, "---\n")
	if !ok {
		return ""
	}
	block, _, found := strings.Cut(rest, "\n---")
	if !found {
		return ""
	}
	for _, line := range strings.Split(block, "\n") {
		value, ok := strings.CutPrefix(line, "name:")
		if !ok {
			continue
		}
		return strings.Trim(strings.TrimSpace(value), "\"'")
	}
	return ""
}

// replaceSkillName replaces every standalone occurrence of old with new. A
// match that continues with a name character is left alone, so a skill whose
// name is a prefix of another's cannot corrupt it.
func replaceSkillName(content, old, new string) string {
	if old == "" {
		return content
	}
	var out strings.Builder
	for {
		index := strings.Index(content, old)
		if index < 0 {
			break
		}
		end := index + len(old)
		if end < len(content) && isSkillNameChar(content[end]) {
			out.WriteString(content[:end])
			content = content[end:]
			continue
		}
		out.WriteString(content[:index])
		out.WriteString(new)
		content = content[end:]
	}
	out.WriteString(content)
	return out.String()
}

func isSkillNameChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case c == '-', c == '_':
		return true
	}
	return false
}

// isEmittedTextAsset reports whether an emitted file carries text the skill
// renames can reach.
func isEmittedTextAsset(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".json":
		return true
	}
	return false
}

func transpileSatellites(piRoot, outputDir string) error {
	satellites := []string{
		"orchestrator-delegation.md",
		"orchestrator-delegation-reference.md",
		"orchestrator-memory.md",
		"orchestrator-skills.md",
		"orchestrator-prompts.md",
		"orchestrator-tracking.md",
		"orchestrator-verification.md",
		"orchestrator-writer.md",
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
			// The fallback skill-dir list in Pi includes `.pi/skills` as a
			// project-level path. agy's project-level equivalent is
			// `.agents/skills/` (already in the list). The correct agy global
			// skill root is `~/.gemini/config/skills/`, listed alongside the
			// other agents' global roots — not in the project-level position.
			content = replaceAntigravityCLILineContaining(content,
				"search common project/user skill dirs such as",
				"3. If the expected skill is absent from the registry but the request clearly names a known workflow, search common project/user skill dirs such as `./skills`, `.agents/skills`, `~/.gemini/config/skills`, `~/.config/opencode/skills`, `~/.claude/skills`, and other configured skill roots.")
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
	if tool == "web_enable" {
		// Pi activation is unnecessary: AGY exposes native web tools directly.
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
			"use the cwd-scoped `codegraph` tool before broad filesystem searches. Initialize the workspace index with `operation: \"init\"` when it is absent, then use `query` or `explore`; never ask it to target another path.",
			"use the `gentle-ai_codegraph` MCP tool `codegraph_explore` before broad filesystem searches. Pass a natural-language question or the relevant symbol and file names; never ask it to target another path.",
		},
		{
			"`codegraph` may create or update only the current workspace `.codegraph/` index.",
			"CodeGraph may create or update only the current workspace `.codegraph/` index (the server manages its own index maintenance).",
		},
		{
			"use `read`, `grep`, and `find` as the fallback",
			"use `view_file`, `grep_search`, and `find_by_name` as the fallback",
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

	// Antigravity CLI injects no `<available_skills>` system-prompt block (the
	// same runtime fact Hermes states in its persona assets). The inherited Pi
	// caution against over-trusting that block therefore reads as if one were
	// injected here. Keep the caution, name the runtime fact.
	content = strings.ReplaceAll(content,
		"For skill-shaped requests, do not treat injected `<available_skills>` as complete.",
		"For skill-shaped requests, do not assume an injected skill list is complete: Antigravity CLI injects no `<available_skills>` block.")

	// AGY has native subagent messaging, not the Pi cross-session directory.
	content = stripAntigravityCLISectionUntil(content, "## Session subject and display identity", "## Pi Runtime Overlays")
	content = replaceAntigravityCLILineContaining(content, "Before work beyond small direct tasks, publish own", "")
	content = replaceAntigravityCLILineContaining(content, "Before delegating, declare a subject via", "")
	content = replaceAntigravityCLILineContaining(content, "Progress is published automatically from the todo list", "")
	content = replaceAntigravityCLILineContaining(content, "For delegation other than a large task's bounded writes", "For delegation other than a large task's bounded writes, use the generic fallback: if the native subagent mechanism is unavailable, stop the complex work and explain the blocker instead of silently continuing inline. The delegation trigger remains mandatory; the fallback changes the runtime, not the requirement to delegate.")
	// Shell 4.3.0 groups the generic delegation rules and routing ladder under
	// this Pi-only heading; drop the heading and its intro, never the generic
	// subsections beneath it. Only the delegation reference carries it.
	content = strings.ReplaceAll(content, "## Pi Runtime Overlays\n\nThese sections bind generic delegation to Pi's runtime, adding routing without changing ODD ownership.\n\n", "")
	content = stripAntigravityCLIMarkedSection(content, "<!-- gentle-pi:background-subagents -->", "<!-- /gentle-pi:background-subagents -->")
	content = stripAntigravityCLISectionUntil(content, "#### Pi Trigger Runtime Bindings", "### Work Routing Ladder")
	content = stripAntigravityCLISectionUntil(content, "#### Pi Subagent Model Routing", "## Pi Delegation Bindings")
	content = stripAntigravityCLISectionUntil(content, "## Pi Delegation Bindings", "### Canonical Lightweight Workflows")
	content = stripAntigravityCLISectionUntil(content, "## Model Assignments", "## Judgment Day fix routing")

	content = replaceAntigravityCLILineContaining(content, "- Native route: For every strictly closed single-select envelope", "- Native route: Use `ask_question` when the complete choice envelope is representable in one native call. Preserve every provider-owned label, description, effect, and answer token exactly; never synthesize or append a continuation. If the native question UI is unavailable or the envelope is unrepresentable, use the complete plain-chat fallback below and stop.")
	content = replaceAntigravityCLILineContaining(content, "Mandatory Delegation Triggers — once fired", "Mandatory Delegation Triggers — once fired, delegate through Antigravity's native `invoke_subagent` and use `send_message` for follow-up guidance:")
	content = replaceAntigravityCLILineContaining(content, "This package injects the mirrored provider-bundle", "This package injects the canonical native review execution contract into the Antigravity parent prompt. Follow that contract exactly; this package owns no separate review lifecycle route.")
	content = replaceAntigravityCLILineContaining(content, "- An eligible interactive Pi host", "- For a provider-owned consent envelope, use `ask_question` only when the complete envelope is representable; otherwise relay it losslessly and stop. Never infer a continuation.")
	content = replaceAntigravityCLILineContaining(content, "A child must be able to do the task it is given.", "A child must be able to do the task it is given. Read its installed agent definition and declared tools before launching; never assume the parent's tools transfer. With no qualified role, report the capability gap; a fired trigger is never replaced by inline work.")
	content = replaceAntigravityCLILineContaining(content, "Use the configured subagent runtime when available", "Use Antigravity's native subagent runtime. Prefer `invoke_subagent` for a fresh delegated context and `send_message` for follow-up guidance; do not invent a background or model-routing contract.")
	content = replaceAntigravityCLILineContaining(content, "Delegate generic non-SDD verification", "Delegate generic verification with `invoke_subagent` to `gentle-ai-verify`; keep only a small read-only check inline. The parent owns the native review lifecycle and all delivery decisions.")
	content = replaceAntigravityCLILineContaining(content, "For delegation other than bounded multi-file writes", "For delegation other than bounded multi-file writes, use the generic fallback: if `invoke_subagent` or `send_message` is unavailable, stop and explain the blocker. The delegation trigger remains mandatory; do not silently continue inline.")
	content = replaceAntigravityCLILineContaining(content, "Never search for, request, or invoke review tools", "The parent owns candidate review disposition and lifecycle. Never search for, request, or invoke review tools.")
	content = replaceAntigravityCLILineContaining(content, "When RDD is enabled, first use native candidate risk assessment", "When RDD is enabled, follow the projected `#### Review Execution Contract` exactly: use only the native review operations returned by that contract, preserve provider-owned bindings, and never infer authority or delivery from model judgment or task size. When RDD is disabled, ordinary checks remain.")
	content = replaceAntigravityCLILineContaining(content, "5. **Verification rule**", "5. **Verification rule**: the bounded writer always self-verifies with exact authorized checks. Follow the native risk tier and the table below in any RDD state. High risk always requires independent `gentle-ai-verify` before native review; medium adds it for a small-model writer, and failed assessment is treated as high. Native review never replaces or skips verification and never authorizes delivery.")
	content = replaceAntigravityCLILineContaining(content, "{{GENTLE_PI_BACKGROUND_POLICY}}", "Use Antigravity's native `invoke_subagent` and keep delegated work foreground until the result returns")
	content = replaceAntigravityCLILineContaining(content, "| Native `nextRecommended` | Pi executor |", "| Native `nextRecommended` | Antigravity executor |")
	content = replaceAntigravityCLILineContaining(content, "the provider host-relay capture path never loads this chain", "> Manual/compat-lane only: this chain exists solely for explicit manual 4R invocation; native review lifecycle captures do not load it.")
	content = strings.ReplaceAll(content, "Use `web_enable` if activation is needed; headless sessions may activate the web tools at startup.", "Use native `search_web` and `read_url_content` directly; no Pi activation step is needed.")
	content = replaceAntigravityCLILineContaining(content, "Waiting on CI, a build, or a server is not delegation:", "Waiting on CI, a build, or a server is not delegation: use native `run_command` and its documented completion mechanism. Do not invent Pi background tools, poll, sleep, or delegate an idle waiter; report an unavailable completion mechanism.")
	content = strings.ReplaceAll(content, "`fetch_content`", "`read_url_content`")
	content = strings.ReplaceAll(content, "`web_search`", "`search_web`")
	content = strings.ReplaceAll(content, "`source_check`", "`read_url_content`")
	content = strings.ReplaceAll(content, "`get_search_content`", "`read_url_content`")
	content = strings.ReplaceAll(content, "Do not use `~/.gemini/config/plugins/gentle-ai/support/...` as a runtime path; that is only the package source path before installation.", "Do not use a package source path as a runtime path.")

	content = stripAntigravityCLISectionUntil(content, "#### Signaling the ODD phase to the Gentle prompt", "When RDD is enabled")
	content = replaceAntigravityCLILineContaining(content, "stall watchdog uses", "- A long foreground command is live work, not silence: a verification command announced under `## Verification` is not an idle timeout to be killed mid-run; it is live work whose observed result the task needs.")
	content = strings.ReplaceAll(content, "Pi-specific coding-agent harness", "coding-agent harness")
	content = strings.ReplaceAll(content, "Pi never mints delivery authority", "The runtime never mints delivery authority")
	content = strings.ReplaceAll(content, "no Pi delivery authority", "no runtime delivery authority")
	content = strings.ReplaceAll(content, "for Pi or another agent runtime", "for your agent runtime")
	content = strings.ReplaceAll(content, "for Pi work", "for agent work")
	content = strings.ReplaceAll(content, "When a code-execution tool such as `codemode` is available", "When a shell command runner is available")
	content = strings.ReplaceAll(content, "follow the Evidence-budget rule under Pi Trigger Runtime Bindings", "follow the Evidence-budget rule under Mandatory Delegation Triggers")
	content = replaceAntigravityCLILineContaining(content, "launch every disjoint unit in the same turn", "- **Parallelism**: with the Background subagent policy on, launch every disjoint unit via `invoke_subagent`, receiving each result before launching the next; if units cannot be launched together, work inline.")
	content = replaceAntigravityCLILineContaining(content, "pass `\"escalate\": {\"item\":", "When the change touches a Task Size high-risk item and the native tier is passive or medium, the native review assess operation raises the tier to high automatically; the raise binds the verification plan only, not the native review tier. The model never calls assess directly.")
	content = replaceAntigravityCLILineContaining(content, "Let `pi-subagents` resolve", "For generic Antigravity subagents (`delegate`, `worker`, `scout`, `context-builder`, `oracle`, `planner`, `researcher`, or other general agents), do not pass the `model` parameter by default. Antigravity resolves model and thinking from the subagent's inherited session defaults and runtime configuration.")
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
	content = strings.ReplaceAll(content, "`gentle_review_capture_group`", "the native review lifecycle capture group")
	gentleReviewRe := regexp.MustCompile(`\bgentle_review\b`)
	content = gentleReviewRe.ReplaceAllString(content, "the native review lifecycle")
	content = strings.ReplaceAll(content, "{{GENTLE_PI_BACKGROUND_POLICY}}", "Use Antigravity's native `invoke_subagent` and keep delegated work foreground until the result returns")
	content = strings.ReplaceAll(content, "~/.pi/agent/gentle-ai/", "~/.gemini/config/plugins/gentle-ai/")
	content = replaceAntigravityCLILineContaining(content, "Run R1 Risk exactly once against the supplied `initial_review_tree`. Return candidate rows for security", "Run R1 Risk exactly once against the supplied provider-bound immutable candidate. Return findings for security, privilege boundaries, data exposure, dependencies, and merge-blocking vulnerabilities. If clean, return an empty findings array with completed inspection.")
	content = replaceAntigravityCLILineContaining(content, "Run R4 Resilience exactly once against the supplied `initial_review_tree`. Return candidate rows for fallbacks", "Run R4 Resilience exactly once against the supplied provider-bound immutable candidate. Return findings for fallbacks, retry/backoff, graceful degradation, observability, load, rollback, and SLO risks. If clean, return an empty findings array with completed inspection.")
	content = replaceAntigravityCLILineContaining(content, "Run R2 Readability exactly once against the supplied `initial_review_tree`. Return candidate rows for naming", "Run R2 Readability exactly once against the supplied provider-bound immutable candidate. Return findings for naming, complexity, intention, maintainability, review size, and context clarity. If clean, return an empty findings array with completed inspection.")
	content = replaceAntigravityCLILineContaining(content, "Run R3 Reliability exactly once against the supplied `initial_review_tree`. Return candidate rows for behavior-first", "Run R3 Reliability exactly once against the supplied provider-bound immutable candidate. Return findings for behavior-first test coverage, edge cases, determinism, contracts, and regressions. If clean, return an empty findings array with completed inspection.")
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
		{"assets/orchestrator-delegation-reference.md", "~/.gemini/config/plugins/gentle-ai/orchestrator-delegation-reference.md"},
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

func runCheckGate(outputDir, sourceDir, sourceRepo string, strict ...bool) error {
	// The drift gate compares against the tree COMMITTED at HEAD, not the
	// on-disk output: a dirty worktree must not make the local gate pass
	// trivially. In CI the checkout is clean, so HEAD and disk coincide.
	committedDir, committedProv, cleanupCommitted, err := getCommittedTree(outputDir)
	if err != nil {
		return fmt.Errorf("get committed tree: %w", err)
	}
	defer cleanupCommitted()

	// Resolve source at the pinned commit
	var sourceRoot string
	var cleanup func()

	if sourceDir != "" {
		// Check if local source HEAD matches the pinned commit
		localSHA, err := getSourceHEAD(sourceDir)
		if err != nil {
			return fmt.Errorf("get local source HEAD: %w", err)
		}
		if localSHA == committedProv.SourceCommit {
			sourceRoot = sourceDir
			cleanup = func() {}
		} else {
			// Local source doesn't match pin; fall through to clone
			sourceDir = ""
		}
	}

	if sourceRoot == "" {
		// Clone remote and checkout pinned commit
		tempDir, err := os.MkdirTemp("", "gentle-pi-check-*")
		if err != nil {
			return fmt.Errorf("create temp dir for check: %w", err)
		}
		cleanup = func() { _ = os.RemoveAll(tempDir) }

		// Clone with full history to allow checkout of any commit
		cmd := exec.Command("git", "clone", sourceRepo, tempDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			cleanup()
			return fmt.Errorf("git clone %s: %w", sourceRepo, err)
		}

		// Checkout the pinned commit
		cmd = exec.Command("git", "checkout", committedProv.SourceCommit)
		cmd.Dir = tempDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			cleanup()
			return fmt.Errorf("git checkout %s: %w", committedProv.SourceCommit, err)
		}
		sourceRoot = tempDir
	}
	defer cleanup()

	// Transpile to a temp directory
	tempOutDir, err := os.MkdirTemp("", "agy-check-*")
	if err != nil {
		return fmt.Errorf("create temp output dir: %w", err)
	}
	defer os.RemoveAll(tempOutDir)

	if err := transpileAll(sourceRoot, tempOutDir); err != nil {
		return fmt.Errorf("transpile for check: %w", err)
	}

	if len(strict) > 0 && strict[0] && len(unmappedToolWarnings) > 0 {
		return errors.New("Strict mode: unmapped tools detected during drift check.")
	}

	// Write provenance to temp output (deterministic, so it should match)
	if err := writeProvenance(tempOutDir, committedProv); err != nil {
		return fmt.Errorf("write provenance for check: %w", err)
	}

	// Compare the regenerated tree against the COMMITTED tree
	diffs, err := compareTrees(committedDir, tempOutDir)
	if err != nil {
		return fmt.Errorf("compare trees: %w", err)
	}

	if len(diffs) > 0 {
		var msg strings.Builder
		msg.WriteString("Drift detected:\n")
		for _, d := range diffs {
			msg.WriteString("  " + d + "\n")
		}
		return errors.New(msg.String())
	}

	return nil
}

func compareTrees(committedDir, tempDir string) ([]string, error) {
	var diffs []string

	// Walk committed dir
	committedFiles := make(map[string]bool)
	err := filepath.WalkDir(committedDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(committedDir, path)
		if err != nil {
			return err
		}
		committedFiles[rel] = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Walk temp dir
	tempFiles := make(map[string]bool)
	err = filepath.WalkDir(tempDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}
		tempFiles[rel] = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Check for missing files (in committed but not in temp)
	for rel := range committedFiles {
		if !tempFiles[rel] {
			diffs = append(diffs, fmt.Sprintf("missing in regenerated: %s", rel))
		}
	}

	// Check for extra files (in temp but not in committed)
	for rel := range tempFiles {
		if !committedFiles[rel] {
			diffs = append(diffs, fmt.Sprintf("extra in regenerated: %s", rel))
		}
	}

	// Compare content for common files
	for rel := range committedFiles {
		if tempFiles[rel] {
			committedPath := filepath.Join(committedDir, rel)
			tempPath := filepath.Join(tempDir, rel)
			committedContent, err := os.ReadFile(committedPath)
			if err != nil {
				return nil, fmt.Errorf("read committed %s: %w", rel, err)
			}
			tempContent, err := os.ReadFile(tempPath)
			if err != nil {
				return nil, fmt.Errorf("read temp %s: %w", rel, err)
			}
			if !bytesEqual(committedContent, tempContent) {
				diffs = append(diffs, fmt.Sprintf("content differs: %s", rel))
			}
		}
	}

	sort.Strings(diffs)
	return diffs, nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
