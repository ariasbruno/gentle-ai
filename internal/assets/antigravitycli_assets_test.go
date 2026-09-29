package assets_test

import (
	"encoding/json"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v3/internal/assets"
)

func TestAntigravityCLI_PluginManifest(t *testing.T) {
	content, err := assets.Read("antigravitycli/plugin.json")
	if err != nil {
		t.Fatalf("Read(antigravitycli/plugin.json) error: %v", err)
	}
	var manifest struct {
		Schema      string `json:"$schema"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(content), &manifest); err != nil {
		t.Fatalf("parse plugin manifest: %v", err)
	}
	if manifest.Schema != "https://antigravity.google/schemas/v1/plugin.json" {
		t.Errorf("$schema = %q, want official Antigravity v1 schema", manifest.Schema)
	}
	if manifest.Name != "gentle-ai" {
		t.Errorf("name = %q, want gentle-ai", manifest.Name)
	}
	if strings.TrimSpace(manifest.Description) == "" {
		t.Error("description is empty")
	}
}

var expectedSubagentNames = []string{
	"gentle-ai-explore",
	"gentle-ai-verify",
	"gentle-ai-worker",
	"jd-fix-agent",
	"jd-judge-a",
	"jd-judge-b",
	"review-readability",
	"review-reliability",
	"review-resilience",
	"review-risk",
}

var nativeReviewAgents = map[string]bool{
	"review-readability": true,
	"review-reliability": true,
	"review-resilience":  true,
	"review-risk":        true,
}

var forbiddenPiTools = []string{
	"read", "grep", "find", "edit", "write", "bash", "ask_user_choice", "ask_user_question", "subagent_run",
}

// TestAntigravityCLI_Invariant1_Subagents asserts that exactly 10 subagents
// are embedded under antigravitycli/agents/<name>/agent.md.
func TestAntigravityCLI_Invariant1_Subagents(t *testing.T) {
	entries, err := fs.ReadDir(assets.FS, "antigravitycli/agents")
	if err != nil {
		t.Fatalf("ReadDir(antigravitycli/agents) error: %v", err)
	}

	foundAgents := make(map[string]bool)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		agentMDPath := filepath.Join("antigravitycli/agents", name, "agent.md")
		data, err := fs.ReadFile(assets.FS, agentMDPath)
		if err != nil {
			t.Errorf("agent %q missing agent.md: %v", name, err)
			continue
		}
		if len(data) < 20 {
			t.Errorf("agent %q agent.md too small (%d bytes)", name, len(data))
		}
		foundAgents[name] = true
	}

	if len(foundAgents) != 10 {
		t.Errorf("found %d subagents, want exactly 10", len(foundAgents))
	}

	for _, expected := range expectedSubagentNames {
		if !foundAgents[expected] {
			t.Errorf("missing expected subagent %q", expected)
		}
	}
}

// TestAntigravityCLI_Invariant2_Frontmatter asserts that each agent.md has valid
// YAML frontmatter with native Antigravity CLI tools and no forbidden Pi tools.
func TestAntigravityCLI_Invariant2_Frontmatter(t *testing.T) {
	for _, name := range expectedSubagentNames {
		path := filepath.Join("antigravitycli/agents", name, "agent.md")
		content, err := assets.Read(path)
		if err != nil {
			t.Errorf("Read(%s) error: %v", path, err)
			continue
		}

		if !strings.HasPrefix(content, "---\n") {
			t.Errorf("%s: missing leading frontmatter fence", path)
			continue
		}
		endIdx := strings.Index(content[4:], "\n---\n")
		if endIdx < 0 {
			t.Errorf("%s: missing closing frontmatter fence", path)
			continue
		}
		fm := content[4 : 4+endIdx]
		for _, tool := range forbiddenPiTools {
			if strings.Contains(fm, "- "+tool+"\n") || strings.Contains(fm, " "+tool+",") || strings.Contains(fm, "["+tool+"]") || strings.Contains(fm, "["+tool+",") || strings.Contains(fm, ", "+tool+"]") || strings.Contains(fm, ", "+tool+",") {
				t.Errorf("%s: frontmatter contains forbidden Pi tool %q", path, tool)
			}
		}
		lower := strings.ToLower(content)
		for _, residue := range []string{"subagent_run", "gentle-pi", "host-relay"} {
			if strings.Contains(lower, residue) {
				t.Errorf("%s: contains forbidden Pi runtime residue %q", path, residue)
			}
		}
		if nativeReviewAgents[name] {
			if !strings.Contains(content, "invoke_subagent") {
				t.Errorf("%s: review delegation does not use the Antigravity CLI-native invoke_subagent route", path)
			}
			if strings.Contains(content, "send_message") {
				t.Errorf("%s: reviewer claims send_message without granting that tool", path)
			}
			if end := strings.Index(content, "\n---\n"); end >= 0 && strings.Contains(content[:end], "send_message") {
				t.Errorf("%s: reviewer frontmatter grants send_message without a matching body contract", path)
			}
		}
	}
}

// TestAntigravityCLI_Invariant3_Satellites asserts exactly 3 satellites exist.
func TestAntigravityCLI_Invariant3_Satellites(t *testing.T) {
	satellites := []string{
		"orchestrator-delegation.md",
		"orchestrator-memory.md",
		"orchestrator-skills.md",
	}

	for _, sat := range satellites {
		path := filepath.Join("antigravitycli", sat)
		content, err := assets.Read(path)
		if err != nil {
			t.Errorf("Read(%s) error: %v", path, err)
			continue
		}
		if len(content) < 50 {
			t.Errorf("%s: content unexpectedly short (%d bytes)", path, len(content))
		}
		if strings.Contains(content, "parent Pi session") {
			t.Errorf("%s: still contains un-transpiled 'parent Pi session'", path)
		}
	}
}

func TestAntigravityCLI_ParentSatelliteSurfaceHasNoExecutablePiResidue(t *testing.T) {
	paths := []string{
		"antigravitycli/orchestrator.md",
		"antigravitycli/orchestrator-delegation.md",
		"antigravitycli/orchestrator-memory.md",
		"antigravitycli/orchestrator-skills.md",
	}
	forbidden := []string{
		"ask_user_choice", "ask_user_question", "subagent_run", "gentle_review",
		".pi/", "~/.pi/agent", "gentle-pi", "Pi Runtime Overlays", "Pi Subagent Model Routing",
		"## Model Assignments", "{{GENTLE_PI_BACKGROUND_POLICY}}",
		"/gentle-sdd-init", "/gentle-sdd-new", "/gentle-sdd-ff", "/gentle-sdd-continue", "/gentle-sdd-status", "/gentle:sdd-preflight",
		"`/sdd-init`", "`/sdd-new`", "`/sdd-ff`", "`/sdd-status`", "`/sdd-continue`", "`/sdd-*`",
	}
	for _, path := range paths {
		content, err := assets.Read(path)
		if err != nil {
			t.Errorf("Read(%s) error: %v", path, err)
			continue
		}
		lower := strings.ToLower(content)
		for _, residue := range forbidden {
			if strings.Contains(lower, strings.ToLower(residue)) {
				t.Errorf("%s contains executable Pi residue %q", path, residue)
			}
		}
	}
	parent, err := assets.Read("antigravitycli/orchestrator.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ask_question", "invoke_subagent", "#### Review Execution Contract"} {
		if !strings.Contains(parent, want) {
			t.Errorf("AGY parent asset missing native contract %q", want)
		}
	}
	delegation, err := assets.Read("antigravitycli/orchestrator-delegation.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(delegation, "send_message") {
		t.Error("AGY delegation satellite does not use native send_message")
	}
}

func TestAntigravityCLI_AssetsHaveNoUnsupportedSlashCommands(t *testing.T) {
	forbidden := []string{
		"/gentle-sdd-init", "/gentle-sdd-new", "/gentle-sdd-ff", "/gentle-sdd-continue", "/gentle-sdd-status", "/gentle:sdd-preflight",
		"`/sdd-init`", "`/sdd-new`", "`/sdd-ff`", "`/sdd-status`", "`/sdd-continue`", "`/sdd-*`",
	}
	var paths []string
	err := fs.WalkDir(assets.FS, "antigravitycli", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".md" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk AGY assets: %v", err)
	}
	for _, path := range paths {
		content, err := assets.Read(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, token := range forbidden {
			if strings.Contains(content, token) {
				t.Errorf("%s contains unsupported Antigravity slash command %q", path, token)
			}
		}
	}
}

// TestAntigravityCLI_AssetsUseInstalledRuntimePaths keeps package source paths
// out of shipped assets. `assets/...` is the repository layout; the installed
// plugin lives under ~/.gemini/config/plugins/gentle-ai, so a reference to the
// source tree is runtime-broken text.
func TestAntigravityCLI_AssetsUseInstalledRuntimePaths(t *testing.T) {
	paths, err := antigravityCLIAssetMarkdownPaths()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		content, err := assets.Read(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(content, "assets/") {
			t.Errorf("%s references package source paths instead of installed paths", path)
		}
	}
}

func antigravityCLIAssetMarkdownPaths() ([]string, error) {
	var paths []string
	err := fs.WalkDir(assets.FS, "antigravitycli", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".md" {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

// TestAntigravityCLI_DelegationFallbackIsNotSelfContradictory guards the
// bounded-writer sentence: rewriting `subagent_*` to the Antigravity tool names
// used to produce "use `invoke_subagent` even when `invoke_subagent` ...".
func TestAntigravityCLI_DelegationFallbackIsNotSelfContradictory(t *testing.T) {
	delegation, err := assets.Read("antigravitycli/orchestrator-delegation.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"even when `invoke_subagent` or `send_message` tools are available",
		"even when invoke_subagent or send_message tools are available",
		"native `Agent`",
	} {
		if strings.Contains(delegation, forbidden) {
			t.Errorf("AGY delegation satellite retains contradictory fallback text %q", forbidden)
		}
	}
	if !strings.Contains(delegation, "If neither worker definition exists, delegate the bounded write with Antigravity's native `invoke_subagent`.") {
		t.Error("AGY delegation satellite lost the truthful bounded-writer fallback")
	}
}

// TestAntigravityCLI_Invariant4_SupportContracts asserts 2 support contracts exist.
func TestAntigravityCLI_Invariant4_SupportContracts(t *testing.T) {
	contracts := []string{
		"strict-tdd.md",
		"strict-tdd-verify.md",
	}

	for _, c := range contracts {
		path := filepath.Join("antigravitycli/support", c)
		content, err := assets.Read(path)
		if err != nil {
			t.Errorf("Read(%s) error: %v", path, err)
			continue
		}
		if len(content) < 50 {
			t.Errorf("%s: content unexpectedly short (%d bytes)", path, len(content))
		}
	}
}

// TestAntigravityCLI_Invariant5_Chains asserts 1 execution chain exists.
func TestAntigravityCLI_Invariant5_Chains(t *testing.T) {
	chains := []string{
		"4r-review.chain.md",
	}

	for _, ch := range chains {
		path := filepath.Join("antigravitycli/chains", ch)
		content, err := assets.Read(path)
		if err != nil {
			t.Errorf("Read(%s) error: %v", path, err)
			continue
		}
		if len(content) < 50 {
			t.Errorf("%s: content unexpectedly short (%d bytes)", path, len(content))
		}
	}
}

// TestAntigravityCLI_RuleSizeLimit asserts that the rule file orchestrator.md
// strictly stays within the Antigravity 24,000-byte cap.
func TestAntigravityCLI_RuleSizeLimit(t *testing.T) {
	const maxRuleBytes = 24000
	content, err := assets.Read("antigravitycli/orchestrator.md")
	if err != nil {
		t.Fatalf("Read(antigravitycli/orchestrator.md) error: %v", err)
	}
	if len(content) >= maxRuleBytes {
		t.Errorf("antigravitycli/orchestrator.md size (%d bytes) exceeds rule limit of %d bytes", len(content), maxRuleBytes)
	}
}

// TestAntigravityCLI_ExploreAgentUsesMCPBridgeForCodeGraph pins the explore
// agent's body to the runtime reality: agy has no cwd-scoped `codegraph` tool
// with init/query/explore operations (that is the Pi interface), and MCP tools
// are reached through the injected call_mcp_tool bridge. Phantom instructions
// would make the agent fail into its grep fallback on every structural task.
func TestAntigravityCLI_ExploreAgentUsesMCPBridgeForCodeGraph(t *testing.T) {
	data, err := assets.Read("antigravitycli/agents/gentle-ai-explore/agent.md")
	if err != nil {
		t.Fatalf("read gentle-ai-explore agent: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "call_mcp_tool") || !strings.Contains(content, "gentle-ai_codegraph") {
		t.Errorf("explore body must route CodeGraph through the MCP bridge (call_mcp_tool, server gentle-ai_codegraph)")
	}
	if strings.Contains(content, "cwd-scoped") || strings.Contains(content, `operation: "init"`) {
		t.Errorf("explore body still describes the Pi-only cwd-scoped codegraph tool interface")
	}
	if strings.Contains(content, "`read`, `grep`, and `find`") {
		t.Errorf("explore fallback names Pi tools; agy names them view_file, grep_search, find_by_name")
	}
}
