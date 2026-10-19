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

func TestAntigravityCLISkillLoadingWordingIsTruthful(t *testing.T) {
	input := "For skill-shaped requests, do not treat injected `<available_skills>` as complete. " +
		"Use the registry and filesystem only as a discovery aid; do not let a trigger table override the user's concrete request.\n"
	got := applyAntigravityCLIParentMutations(applyCommonTextMutations(input))
	if !strings.Contains(got, "Antigravity CLI injects no `<available_skills>` block") {
		t.Fatalf("skill-loading wording still presumes an injected block:\n%s", got)
	}
	if strings.Contains(got, "do not treat injected `<available_skills>` as complete") {
		t.Errorf("generated text retains the Pi-only caution:\n%s", got)
	}
	if !strings.Contains(got, "do not let a trigger table override the user's concrete request") {
		t.Errorf("generated text dropped the trigger-table caution:\n%s", got)
	}
}

// writeNamespacedSkill writes a skill whose frontmatter name does not match
// its folder, the way Gentle Shell ships them.
func writeNamespacedSkill(t *testing.T, sourceRoot, folder, name, body string) {
	t.Helper()
	skillDir := filepath.Join(sourceRoot, "skills", folder)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := fmt.Sprintf("---\nname: %s\ndescription: Namespaced skill fixture.\n---\n\n%s", name, body)
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Gentle Shell ships its skills with a namespaced frontmatter name
// (`gentle-ai-<folder>`) inside a `<folder>` directory. Antigravity CLI's
// plugin loader requires the two to match, so the emitted plugin must carry the
// folder as the skill name, and every reference to that skill must follow it.
func TestEmittedSkillNamesMatchTheirFolders(t *testing.T) {
	sourceRoot := t.TempDir()
	writeMinimalSourceTree(t, sourceRoot)

	writeNamespacedSkill(t, sourceRoot, "chained-pr", "gentle-ai-chained-pr",
		"Use `gentle-ai-skill-creator` when authoring a reusable pattern.\n")
	writeNamespacedSkill(t, sourceRoot, "skill-creator", "gentle-ai-skill-creator",
		"Create reusable patterns.\n")

	// The skills satellite routes intents to skills by name.
	satellite := filepath.Join(sourceRoot, "assets", "orchestrator-skills.md")
	if err := os.WriteFile(satellite, []byte("# Skills\n\n| Split/stack/large PR | `gentle-ai-chained-pr` |\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	if err := transpileAll(sourceRoot, outputDir); err != nil {
		t.Fatal(err)
	}

	for _, folder := range []string{"chained-pr", "skill-creator"} {
		raw, err := os.ReadFile(filepath.Join(outputDir, "skills", folder, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "name: "+folder+"\n") {
			t.Errorf("emitted skill %s does not carry its folder as its name:\n%s", folder, raw)
		}
	}

	// No emitted file may name a skill the plugin does not ship under that name.
	for old, folder := range map[string]string{
		"gentle-ai-chained-pr":    "chained-pr",
		"gentle-ai-skill-creator": "skill-creator",
	} {
		err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(raw), old) {
				t.Errorf("emitted %s still names %q; the plugin ships it as %q", path, old, folder)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestEmittedSkillNamesSurvivePrefixCollisions pins the guard that keeps a
// skill whose old name is a prefix of another skill's old name from being
// corrupted by the rename: the shorter name must not be replaced inside the
// longer one, and both must end up carrying their own folder.
func TestEmittedSkillNamesSurvivePrefixCollisions(t *testing.T) {
	sourceRoot := t.TempDir()
	writeMinimalSourceTree(t, sourceRoot)

	// "gentle-ai-skill" is a strict prefix of "gentle-ai-skill-creator".
	writeNamespacedSkill(t, sourceRoot, "skill", "gentle-ai-skill",
		"Use `gentle-ai-skill-creator` for patterns, and `gentle-ai-skill` for itself.\n")
	writeNamespacedSkill(t, sourceRoot, "skill-creator", "gentle-ai-skill-creator",
		"Related: `gentle-ai-skill` and `gentle-ai-skill-creator`.\n")

	satellite := filepath.Join(sourceRoot, "assets", "orchestrator-skills.md")
	if err := os.WriteFile(satellite, []byte("# Skills\n\n| A | `gentle-ai-skill` |\n| B | `gentle-ai-skill-creator` |\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	if err := transpileAll(sourceRoot, outputDir); err != nil {
		t.Fatal(err)
	}

	read := func(rel string) string {
		raw, err := os.ReadFile(filepath.Join(outputDir, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		return string(raw)
	}

	for folder, want := range map[string]string{
		"skills/skill/SKILL.md":         "name: skill\n",
		"skills/skill-creator/SKILL.md": "name: skill-creator\n",
	} {
		if !strings.Contains(read(folder), want) {
			t.Errorf("%s must carry %q, got:\n%s", folder, want, read(folder))
		}
	}

	// The prefix case must not corrupt the longer name nor leave a hybrid behind.
	if body := read("skills/skill/SKILL.md"); !strings.Contains(body, "Use `skill-creator` for patterns, and `skill` for itself.") {
		t.Errorf("prefix body corrupted:\n%s", body)
	}
	if body := read("skills/skill-creator/SKILL.md"); !strings.Contains(body, "Related: `skill` and `skill-creator`.") {
		t.Errorf("prefix body corrupted:\n%s", body)
	}

	// No old name may survive anywhere in the emitted tree.
	for _, old := range []string{"gentle-ai-skill", "gentle-ai-skill-creator"} {
		err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if strings.Contains(string(raw), old) {
				t.Errorf("emitted %s still names %q", path, old)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestCommittedBundleSkillNamesMatchTheirFolders pins the property the AGY
// plugin loader enforces on the COMMITTED bundle, not only on a fixture: every
// emitted skill's frontmatter name equals its folder, and no file in the
// committed tree names a skill the plugin does not ship under that name.
func TestCommittedBundleSkillNamesMatchTheirFolders(t *testing.T) {
	bundleDir := filepath.Join("..", "..", "internal", "assets", "antigravitycli")
	skillsDir := filepath.Join(bundleDir, "skills")

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("read committed skills dir: %v", err)
	}

	shipped := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(skillsDir, entry.Name(), "SKILL.md"))
		if err != nil {
			t.Fatalf("read committed skill %s: %v", entry.Name(), err)
		}
		name := frontmatterName(string(raw))
		if name != entry.Name() {
			t.Errorf("committed skill %s declares name %q; the AGY plugin loader requires the name to equal the folder", entry.Name(), name)
		}
		shipped[name] = true
	}
	if len(shipped) == 0 {
		t.Fatal("committed bundle ships no skills")
	}

	// Every namespaced name the tree might still carry must be gone.
	err = filepath.Walk(bundleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, stale := range []string{
			"gentle-ai-branch-pr", "gentle-ai-chained-pr", "gentle-ai-cognitive-doc-design",
			"gentle-ai-comment-writer", "gentle-ai-issue-creation", "gentle-ai-judgment-day",
			"gentle-ai-rdd-defect-workflow", "gentle-ai-skill-creator", "gentle-ai-skill-improver",
			"gentle-ai-skill-registry", "gentle-ai-work-unit-commits",
		} {
			if strings.Contains(string(raw), stale) {
				t.Errorf("committed %s still names the skill %q; the plugin ships it under its folder", path, stale)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
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
		"orchestrator-delegation.md":           "# Delegation\n\nContent.\n",
		"orchestrator-delegation-reference.md": "# Delegation Reference\n\nContent.\n",
		"orchestrator-memory.md":               "# Memory\n\nContent.\n",
		"orchestrator-skills.md":               "# Skills\n\nContent.\n",
		"orchestrator-prompts.md":              "# Prompts\n\nContent.\n",
		"orchestrator-tracking.md":             "# Tracking\n\nContent.\n",
		"orchestrator-verification.md":         "# Verification\n\nContent.\n",
		"orchestrator-writer.md":               "# Writer\n\nContent.\n",
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
	// skills (minimal fixture for transpiler)
	skillsDir := filepath.Join(sourceRoot, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a minimal test skill
	testSkillDir := filepath.Join(skillsDir, "test-skill")
	if err := os.MkdirAll(testSkillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	testSkillContent := "---\nname: test-skill\ndescription: Test skill fixture.\n---\n\nTest skill content.\n"
	if err := os.WriteFile(filepath.Join(testSkillDir, "SKILL.md"), []byte(testSkillContent), 0o644); err != nil {
		t.Fatal(err)
	}
	// _shared fragment
	sharedDir := filepath.Join(skillsDir, "_shared")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sharedContent := "# Shared Fragment\n\nShared content.\n"
	if err := os.WriteFile(filepath.Join(sharedDir, "test-fragment.md"), []byte(sharedContent), 0o644); err != nil {
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

// TestCheckGateComparesAgainstCommittedTree pins the git-path contract: the
// drift gate compares against the tree COMMITTED at HEAD, not the on-disk
// output. A dirty worktree must not make the gate pass or fail differently
// (the regression this exists for), while a COMMITTED hand edit must fail.
func TestCheckGateComparesAgainstCommittedTree(t *testing.T) {
	sourceRoot := t.TempDir()
	writeMinimalSourceTree(t, sourceRoot)
	if err := initGitRepo(t, sourceRoot); err != nil {
		t.Fatalf("init source git: %v", err)
	}
	sha, err := getGitHEAD(sourceRoot)
	if err != nil {
		t.Fatalf("get source HEAD: %v", err)
	}

	// A git repository whose COMMITTED tree holds the bundle under a nested path.
	repoRoot := t.TempDir()
	outputDir := filepath.Join(repoRoot, "bundle")
	if err := transpileAll(sourceRoot, outputDir); err != nil {
		t.Fatalf("transpile bundle: %v", err)
	}
	if err := writeProvenance(outputDir, Provenance{SourceRepo: defaultRemoteRepo, SourceCommit: sha}); err != nil {
		t.Fatalf("write provenance: %v", err)
	}
	if err := initGitRepo(t, repoRoot); err != nil {
		t.Fatalf("init bundle repo: %v", err)
	}

	// 1. Clean worktree: HEAD comparison passes.
	if err := runCheckGate(outputDir, sourceRoot, defaultRemoteRepo); err != nil {
		t.Fatalf("check gate failed on clean committed tree: %v", err)
	}

	// 2. Dirty worktree: the gate still passes because it reads HEAD, not disk.
	dirtyPath := filepath.Join(outputDir, "plugin.json")
	original, err := os.ReadFile(dirtyPath)
	if err != nil {
		t.Fatalf("read plugin.json: %v", err)
	}
	if err := os.WriteFile(dirtyPath, append(original, []byte("<!-- dirty worktree -->")...), 0o644); err != nil {
		t.Fatalf("dirty plugin.json: %v", err)
	}
	if err := runCheckGate(outputDir, sourceRoot, defaultRemoteRepo); err != nil {
		t.Fatalf("check gate must ignore a dirty worktree (it compares HEAD): %v", err)
	}
	if err := os.WriteFile(dirtyPath, original, 0o644); err != nil {
		t.Fatalf("restore plugin.json: %v", err)
	}

	// 3. Committed drift: a hand edit that is committed must fail the gate.
	if err := os.WriteFile(dirtyPath, append(original, []byte("<!-- committed drift -->")...), 0o644); err != nil {
		t.Fatalf("drift plugin.json: %v", err)
	}
	cmd := exec.Command("git", "commit", "-am", "deliberate drift")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit drift: %v: %s", err, out)
	}
	err = runCheckGate(outputDir, sourceRoot, defaultRemoteRepo)
	if err == nil {
		t.Fatal("check gate must fail on committed drift")
	}
	if !strings.Contains(err.Error(), "Drift detected") {
		t.Errorf("drift error = %q, want it to name the drift", err.Error())
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
