package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	antigravitycliagent "github.com/gentleman-programming/gentle-ai/v4/internal/agents/antigravitycli"
)

func TestTranspileSubagentsCount(t *testing.T) {
	sourceRoot := t.TempDir()
	destination := filepath.Join(t.TempDir(), "agents")

	for i := 0; i < 10; i++ {
		writeAgentFixture(t, sourceRoot, fmt.Sprintf("agent-%02d", i), "Representative agent body.\n")
	}

	staleMarker := filepath.Join(destination, "stale-agent", "marker.txt")
	if err := os.MkdirAll(filepath.Dir(staleMarker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staleMarker, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := transpileSubagents(sourceRoot, destination); err != nil {
		t.Fatalf("transpileSubagents() error = %v", err)
	}

	entries, err := os.ReadDir(destination)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 10 {
		t.Fatalf("generated %d agent directories, want 10 source agents", len(entries))
	}
	if _, err := os.Stat(staleMarker); !os.IsNotExist(err) {
		t.Fatalf("stale agent survived successful replacement; stat error = %v", err)
	}
}

func TestTranspileSubagentsFailureLeavesDestinationUnchanged(t *testing.T) {
	sourceRoot := t.TempDir()
	destination := filepath.Join(t.TempDir(), "agents")
	writeAgentFixture(t, sourceRoot, "a-valid", "This valid source is processed first.\n")
	if err := os.WriteFile(filepath.Join(sourceRoot, "assets", "agents", "z-invalid.md"), []byte("not an agent\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	existingPath := filepath.Join(destination, "a-valid", "agent.md")
	staleMarker := filepath.Join(destination, "stale-agent", "marker.txt")
	if err := os.MkdirAll(filepath.Dir(existingPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(staleMarker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existingPath, []byte("previous generation\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staleMarker, []byte("keep on failure\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := transpileSubagents(sourceRoot, destination)
	if err == nil || !strings.Contains(err.Error(), "transpile subagent z-invalid") {
		t.Fatalf("transpileSubagents() error = %v, want invalid trailing agent error", err)
	}
	existing, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(existing) != "previous generation\n" {
		t.Fatalf("existing agent was partially refreshed: %q", existing)
	}
	marker, err := os.ReadFile(staleMarker)
	if err != nil || string(marker) != "keep on failure\n" {
		t.Fatalf("stale agent was partially pruned: marker=%q error=%v", marker, err)
	}
}

func TestTranspileSubagentMarkdownRemovesPiRuntimeInstructions(t *testing.T) {
	raw := `---
name: runtime-fixture
description: Representative runtime mutation fixture.
tools:
  - read
  - subagent_run
---

> Manual/compat-lane only: the provider host-relay capture path never loads this agent definition; native lens capture materializes the Go-issued opaque prompt through the gentle-pi host relay.

Use ` + "`subagent_run`" + ` to delegate work and return the result.
The ` + "`subagent_run`" + ` background runtime returns immediately.
Never delegate or invoke ` + "`subagent_*`" + ` tools.
Project override: ` + "`.pi/gentle-ai/support/strict-tdd.md`" + `; global: ` + "`~/.pi/agent/gentle-ai/support/strict-tdd.md`" + `.
Do not use ` + "`assets/support/...`" + ` as a runtime path; that is only the package source path before installation.
The verification contract is a (gentle-pi#661, RDD-aware pilot).
Historical note: Pi migration history remains unchanged.
`

	got, err := transpileSubagentMarkdown("runtime-fixture", raw)
	if err != nil {
		t.Fatalf("transpileSubagentMarkdown() error = %v", err)
	}
	for _, forbidden := range []string{"subagent_run", "subagent_*", "gentle-pi", "host-relay", "host relay", "background runtime", ".pi/"} {
		if strings.Contains(strings.ToLower(got), forbidden) {
			t.Errorf("generated agent contains forbidden runtime residue %q:\n%s", forbidden, got)
		}
	}
	for _, required := range []string{
		"`invoke_subagent` to delegate work and `send_message` to return the result",
		".gemini/gentle-ai/support/strict-tdd.md",
		"~/.gemini/config/plugins/gentle-ai/support/strict-tdd.md",
		"Historical note: Pi migration history remains unchanged.",
	} {
		if !strings.Contains(got, required) {
			t.Errorf("generated agent is missing %q:\n%s", required, got)
		}
	}
}

func TestTranspileTopLevelOrchestratorAddsDeterministicAGYContracts(t *testing.T) {
	sourceRoot := t.TempDir()
	assetsDir := filepath.Join(sourceRoot, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := `# Source Orchestrator

## Core Role
Delegate through subagent_run and ask_user_choice. Use .pi/gentle-ai and gentle_review.

## Pi Runtime Overlays
Remove this Pi-only section.

## Gentle AI RDD ownership
RDD ownership.

## Safety
Stay safe.
`
	if err := os.WriteFile(filepath.Join(assetsDir, "orchestrator.md"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	outputDir := t.TempDir()
	if err := transpileTopLevelOrchestrator(sourceRoot, outputDir); err != nil {
		t.Fatalf("transpileTopLevelOrchestrator() error = %v", err)
	}
	first, err := os.ReadFile(filepath.Join(outputDir, "orchestrator.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := transpileTopLevelOrchestrator(sourceRoot, outputDir); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(outputDir, "orchestrator.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("AGY orchestrator generation is not deterministic")
	}
	got := string(first)
	for _, want := range []string{
		antigravityCLIReviewContractHeading,
		antigravitycliagent.ReviewContractInsertMarker,
		"`invoke_subagent`",
		"ask_question",
		"~/.gemini/config/plugins/gentle-ai",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated AGY parent missing %q:\n%s", want, got)
		}
	}
	for _, forbidden := range []string{
		"ask_user_choice", "subagent_run", "gentle_review", ".pi/", "gentle-pi", "Pi Runtime Overlays", "background-subagents",
		"/gentle-sdd-init", "/gentle-sdd-new", "/gentle-sdd-ff", "/gentle-sdd-continue", "/gentle-sdd-status", "/gentle:sdd-preflight", "`/sdd-status`", "`/sdd-continue`", "`/sdd-*`",
	} {
		if strings.Contains(got, forbidden) {
			t.Errorf("generated AGY parent retains executable/residual Pi surface %q:\n%s", forbidden, got)
		}
	}
}

func TestTranspileReviewerSubagentPreservesProviderEnvelope(t *testing.T) {
	raw := `---
name: review-risk
description: Reviewer fixture.
tools:
  - read
---

> Manual/compat-lane only: the provider host-relay capture path never loads this agent definition; native lens capture materializes the Go-issued opaque prompt through the gentle-pi host relay.

## Output contract

Return only this compact-v2 native JSON envelope:

` + "```json" + `
{"review_result":{"lens_results":[]}}
` + "```" + `

Only candidate-caused BLOCKER or CRITICAL findings may require correction.
`
	got, err := transpileSubagentMarkdown("review-risk", raw)
	if err != nil {
		t.Fatalf("transpileSubagentMarkdown() error = %v", err)
	}
	for _, forbidden := range []string{"compact-v2", "review_result", "lens_results", "gentle-pi", "send_message"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("generated reviewer contains obsolete contract %q:\n%s", forbidden, got)
		}
	}
	for _, want := range []string{`"subject_hash"`, `"inspection"`, `"findings"`, `"evidence"`, "Emit no other top-level fields", "provider-bound immutable candidate"} {
		if !strings.Contains(got, want) {
			t.Errorf("generated reviewer missing provider contract %q:\n%s", want, got)
		}
	}
}

func TestAntigravityCLIMutationPreservesPortableSkillDiscovery(t *testing.T) {
	input := "Search ~/.config/opencode/skills and ~/.claude/skills before falling back.\n"
	got := applyAntigravityCLIParentMutations(input)
	if strings.TrimSpace(got) != strings.TrimSpace(input) {
		t.Fatalf("portable cross-runtime skill discovery changed:\n got %q\nwant %q", got, input)
	}
}

func TestAntigravityCLIInstalledAssetPaths(t *testing.T) {
	input := "Loaded on demand from `assets/orchestrator.md`'s pointers.\n" +
		"Generic delegations follow `assets/orchestrator-delegation.md`.\n" +
		"Read `assets/support/strict-tdd.md` for status.\n" +
		"```text\nassets/orchestrator-memory.md\nassets/orchestrator-skills.md\nassets/chains/4r-review.chain.md\n```\n"
	got := applyAntigravityCLIParentMutations(applyCommonTextMutations(input))
	for _, want := range []string{
		"`~/.gemini/config/plugins/gentle-ai/orchestrator.md`",
		"`~/.gemini/config/plugins/gentle-ai/orchestrator-delegation.md`",
		"`~/.gemini/config/plugins/gentle-ai/support/strict-tdd.md`",
		"~/.gemini/config/plugins/gentle-ai/orchestrator-memory.md",
		"~/.gemini/config/plugins/gentle-ai/orchestrator-skills.md",
		"~/.gemini/config/plugins/gentle-ai/chains/4r-review.chain.md",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated text missing installed path %q:\n%s", want, got)
		}
	}
	for _, forbidden := range []string{
		"assets/",
		"~/.pi/",
	} {
		if strings.Contains(got, forbidden) {
			t.Errorf("generated text still references a non-installed path %q:\n%s", forbidden, got)
		}
	}
}

func TestAntigravityCLIDelegationFallbackIsTruthful(t *testing.T) {
	input := "For bounded multi-file writes, prefer the installed package-owned `gentle-ai-worker`, then a user-configured `worker`. " +
		"If neither worker definition exists, fall back to the native `Agent` even when `subagent_*` tools are available. " +
		"If no delegation mechanism is available, stop and explain the blocker.\n"
	got := applyAntigravityCLIParentMutations(input)
	if !strings.Contains(got, "If neither worker definition exists, delegate the bounded write with Antigravity's native `invoke_subagent`.") {
		t.Fatalf("bounded-writer fallback was not made truthful:\n%s", got)
	}
	for _, forbidden := range []string{"even when", "native `Agent`", "subagent_"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("generated fallback retains %q:\n%s", forbidden, got)
		}
	}
}

func writeAgentFixture(t *testing.T, sourceRoot, name, body string) {
	t.Helper()
	agentsDir := filepath.Join(sourceRoot, "assets", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := fmt.Sprintf("---\nname: %s\ndescription: Representative agent fixture.\ntools:\n  - read\n---\n\n%s", name, body)
	if err := os.WriteFile(filepath.Join(agentsDir, name+".md"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeMinimalSourceTree(t *testing.T, sourceRoot string) {
	t.Helper()
	assetsDir := filepath.Join(sourceRoot, "assets")
	// agents dir with at least one agent
	agentsDir := filepath.Join(assetsDir, "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeAgentFixture(t, sourceRoot, "test-agent", "Test body.\n")
	// satellites
	satellites := map[string]string{
		"orchestrator-delegation.md": "# Delegation\n\nContent.\n",
		"orchestrator-memory.md":     "# Memory\n\nContent.\n",
		"orchestrator-skills.md":     "# Skills\n\nContent.\n",
	}
	for name, content := range satellites {
		if err := os.WriteFile(filepath.Join(assetsDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// support contracts
	supportDir := filepath.Join(assetsDir, "support")
	if err := os.MkdirAll(supportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	supportFiles := map[string]string{
		"strict-tdd.md":        "# Strict TDD\n\nContent.\n",
		"strict-tdd-verify.md": "# Strict TDD Verify\n\nContent.\n",
	}
	for name, content := range supportFiles {
		if err := os.WriteFile(filepath.Join(supportDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// chains
	chainsDir := filepath.Join(assetsDir, "chains")
	if err := os.MkdirAll(chainsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chainsDir, "4r-review.chain.md"), []byte("# Chain\n\nContent.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// top-level orchestrator
	if err := os.WriteFile(filepath.Join(assetsDir, "orchestrator.md"), []byte("# Orchestrator\n\nContent.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProvenanceEmissionDeterministic(t *testing.T) {
	sourceRoot := t.TempDir()
	writeMinimalSourceTree(t, sourceRoot)
	if err := initGitRepo(t, sourceRoot); err != nil {
		t.Fatalf("init git: %v", err)
	}

	// Get the commit SHA for provenance
	sha, err := getGitHEAD(sourceRoot)
	if err != nil {
		t.Fatalf("get HEAD: %v", err)
	}
	prov := Provenance{SourceRepo: defaultRemoteRepo, SourceCommit: sha}

	// First transpilation
	outputDir1 := t.TempDir()
	if err := transpileAll(sourceRoot, outputDir1); err != nil {
		t.Fatalf("first transpile: %v", err)
	}
	if err := writeProvenance(outputDir1, prov); err != nil {
		t.Fatalf("write provenance 1: %v", err)
	}

	// Second transpilation
	outputDir2 := t.TempDir()
	if err := transpileAll(sourceRoot, outputDir2); err != nil {
		t.Fatalf("second transpile: %v", err)
	}
	if err := writeProvenance(outputDir2, prov); err != nil {
		t.Fatalf("write provenance 2: %v", err)
	}

	// Read both provenance files
	prov1Path := filepath.Join(outputDir1, provenanceFileName)
	prov2Path := filepath.Join(outputDir2, provenanceFileName)
	prov1Data, err := os.ReadFile(prov1Path)
	if err != nil {
		t.Fatalf("read provenance 1: %v", err)
	}
	prov2Data, err := os.ReadFile(prov2Path)
	if err != nil {
		t.Fatalf("read provenance 2: %v", err)
	}

	// Compare byte-for-byte (deterministic)
	if string(prov1Data) != string(prov2Data) {
		t.Fatalf("provenance not deterministic:\nfirst: %s\nsecond: %s", prov1Data, prov2Data)
	}

	// Verify schema
	var provParsed Provenance
	if err := json.Unmarshal(prov1Data, &provParsed); err != nil {
		t.Fatalf("parse provenance: %v", err)
	}
	if provParsed.SourceRepo == "" {
		t.Fatal("source_repo is empty")
	}
	if provParsed.SourceCommit == "" {
		t.Fatal("source_commit is empty")
	}
	if len(provParsed.SourceCommit) != 40 {
		t.Fatalf("source_commit not 40-char SHA: %q", provParsed.SourceCommit)
	}
	// No timestamps or machine paths
	provStr := string(prov1Data)
	if strings.Contains(provStr, "timestamp") || strings.Contains(provStr, "machine") || strings.Contains(provStr, "hostname") {
		t.Fatalf("provenance contains non-deterministic fields: %s", provStr)
	}
}

func TestProvenanceContent(t *testing.T) {
	sourceRoot := t.TempDir()
	writeMinimalSourceTree(t, sourceRoot)
	if err := initGitRepo(t, sourceRoot); err != nil {
		t.Fatalf("init git: %v", err)
	}

	sha, err := getGitHEAD(sourceRoot)
	if err != nil {
		t.Fatalf("get HEAD: %v", err)
	}
	prov := Provenance{SourceRepo: defaultRemoteRepo, SourceCommit: sha}

	outputDir := t.TempDir()
	if err := transpileAll(sourceRoot, outputDir); err != nil {
		t.Fatalf("transpile: %v", err)
	}
	if err := writeProvenance(outputDir, prov); err != nil {
		t.Fatalf("write provenance: %v", err)
	}

	provPath := filepath.Join(outputDir, provenanceFileName)
	data, err := os.ReadFile(provPath)
	if err != nil {
		t.Fatalf("read provenance: %v", err)
	}

	var provParsed Provenance
	if err := json.Unmarshal(data, &provParsed); err != nil {
		t.Fatalf("parse provenance: %v", err)
	}

	// Verify exact expected schema
	expected := Provenance{
		SourceRepo:   defaultRemoteRepo,
		SourceCommit: "", // Will be filled by actual git HEAD
	}
	if provParsed.SourceRepo != expected.SourceRepo {
		t.Errorf("source_repo = %q, want %q", provParsed.SourceRepo, expected.SourceRepo)
	}
	if provParsed.SourceCommit == "" {
		t.Error("source_commit is empty")
	}
	if len(provParsed.SourceCommit) != 40 {
		t.Errorf("source_commit length = %d, want 40", len(provParsed.SourceCommit))
	}

	// Verify deterministic formatting
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, data, "", "  "); err != nil {
		t.Fatalf("re-indent: %v", err)
	}
	if string(data) != pretty.String() {
		t.Errorf("provenance not deterministically formatted:\ngot: %s\nwant: %s", data, pretty.String())
	}
}

func TestCheckGateIdenticalTreesPass(t *testing.T) {
	// Create a source repo with a known commit
	sourceRoot := t.TempDir()
	writeMinimalSourceTree(t, sourceRoot)

	// Initialize git repo
	if err := initGitRepo(t, sourceRoot); err != nil {
		t.Fatalf("init git: %v", err)
	}

	// Get the commit SHA
	sha, err := getGitHEAD(sourceRoot)
	if err != nil {
		t.Fatalf("get HEAD: %v", err)
	}

	// Transpile to committed output dir
	committedDir := t.TempDir()
	if err := transpileAll(sourceRoot, committedDir); err != nil {
		t.Fatalf("transpile committed: %v", err)
	}

	// Manually write provenance with the pinned commit (simulating committed state)
	prov := Provenance{SourceRepo: defaultRemoteRepo, SourceCommit: sha}
	if err := writeProvenance(committedDir, prov); err != nil {
		t.Fatalf("write provenance: %v", err)
	}

	// Run check gate with local source matching the pin
	if err := runCheckGate(committedDir, sourceRoot, defaultRemoteRepo); err != nil {
		t.Fatalf("check gate failed on identical trees: %v", err)
	}
}

func TestCheckGateContentDriftFails(t *testing.T) {
	sourceRoot := t.TempDir()
	writeMinimalSourceTree(t, sourceRoot)

	if err := initGitRepo(t, sourceRoot); err != nil {
		t.Fatalf("init git: %v", err)
	}
	sha, err := getGitHEAD(sourceRoot)
	if err != nil {
		t.Fatalf("get HEAD: %v", err)
	}

	committedDir := t.TempDir()
	if err := transpileAll(sourceRoot, committedDir); err != nil {
		t.Fatalf("transpile committed: %v", err)
	}
	prov := Provenance{SourceRepo: defaultRemoteRepo, SourceCommit: sha}
	if err := writeProvenance(committedDir, prov); err != nil {
		t.Fatalf("write provenance: %v", err)
	}

	// Corrupt a file in committed dir
	corruptPath := filepath.Join(committedDir, "plugin.json")
	if err := os.WriteFile(corruptPath, []byte("corrupted\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Check gate should fail
	err = runCheckGate(committedDir, sourceRoot, defaultRemoteRepo)
	if err == nil {
		t.Fatal("check gate should have failed on content drift")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "content differs: plugin.json") {
		t.Errorf("error message should mention content drift: %s", errStr)
	}
}

func TestCheckGateMissingFileFails(t *testing.T) {
	sourceRoot := t.TempDir()
	writeMinimalSourceTree(t, sourceRoot)

	if err := initGitRepo(t, sourceRoot); err != nil {
		t.Fatalf("init git: %v", err)
	}
	sha, err := getGitHEAD(sourceRoot)
	if err != nil {
		t.Fatalf("get HEAD: %v", err)
	}

	committedDir := t.TempDir()
	if err := transpileAll(sourceRoot, committedDir); err != nil {
		t.Fatalf("transpile committed: %v", err)
	}
	prov := Provenance{SourceRepo: defaultRemoteRepo, SourceCommit: sha}
	if err := writeProvenance(committedDir, prov); err != nil {
		t.Fatalf("write provenance: %v", err)
	}

	// Remove a file from committed dir (so regenerated has it but committed doesn't)
	removePath := filepath.Join(committedDir, "plugin.json")
	if err := os.Remove(removePath); err != nil {
		t.Fatal(err)
	}

	err = runCheckGate(committedDir, sourceRoot, defaultRemoteRepo)
	if err == nil {
		t.Fatal("check gate should have failed on missing file")
	}
	errStr := err.Error()
	// File exists in regenerated but not in committed -> "extra in regenerated"
	if !strings.Contains(errStr, "extra in regenerated: plugin.json") {
		t.Errorf("error message should mention extra in regenerated: %s", errStr)
	}
}

func TestCheckGateExtraFileFails(t *testing.T) {
	sourceRoot := t.TempDir()
	writeMinimalSourceTree(t, sourceRoot)

	if err := initGitRepo(t, sourceRoot); err != nil {
		t.Fatalf("init git: %v", err)
	}
	sha, err := getGitHEAD(sourceRoot)
	if err != nil {
		t.Fatalf("get HEAD: %v", err)
	}

	committedDir := t.TempDir()
	if err := transpileAll(sourceRoot, committedDir); err != nil {
		t.Fatalf("transpile committed: %v", err)
	}
	prov := Provenance{SourceRepo: defaultRemoteRepo, SourceCommit: sha}
	if err := writeProvenance(committedDir, prov); err != nil {
		t.Fatalf("write provenance: %v", err)
	}

	// Add an extra file to committed dir (so committed has it but regenerated doesn't)
	extraPath := filepath.Join(committedDir, "extra.txt")
	if err := os.WriteFile(extraPath, []byte("extra\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err = runCheckGate(committedDir, sourceRoot, defaultRemoteRepo)
	if err == nil {
		t.Fatal("check gate should have failed on extra file")
	}
	errStr := err.Error()
	// File exists in committed but not in regenerated -> "missing in regenerated"
	if !strings.Contains(errStr, "missing in regenerated: extra.txt") {
		t.Errorf("error message should mention missing in regenerated: %s", errStr)
	}
}

func initGitRepo(t *testing.T, dir string) error {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %w: %s", err, out)
	}
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config email: %w: %s", err, out)
	}
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config name: %w: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(""), 0o644); err != nil {
		return err
	}
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w: %s", err, out)
	}
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, out)
	}
	return nil
}
