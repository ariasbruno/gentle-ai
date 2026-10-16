package antigravitycli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestStatuslineScript_JQMissing verifies the fallback when jq is not in PATH.
func TestStatuslineScript_JQMissing(t *testing.T) {
	home := t.TempDir()
	// PATH pointing at a directory with no executables, so jq is genuinely
	// absent even when jq lives in /usr/bin on the host.
	env := []string{"PATH=" + home, "HOME=" + home}

	payload := `{"agent_state":"working","cwd":"/home/user/project","vcs":{"branch":"main","dirty":true},"model":{"display_name":"Gemini 2.5 Pro"},"context_window":{"used_percentage":42},"cost":0.312,"plan_tier":"Pro","terminal_width":120,"task_count":0,"subagents":[],"sandbox":{"enabled":false},"vim":{"mode":"NORMAL"},"conversation_title":"test"}`
	cmd := NewStatuslineCmd(env, home, payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v", err)
	}
	if strings.TrimSpace(string(output)) != "✿ gentle ai ⟡ install jq for the live status line" {
		t.Errorf("unexpected output: %q", strings.TrimSpace(string(output)))
	}
}

// TestStatuslineScript_FullLine verifies the comfortable density full line snapshot.
func TestStatuslineScript_FullLine(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	payload := `{
		"agent_state": "working",
		"cwd": "/home/user/project/src/feature",
		"vcs": {"branch": "feat/login", "dirty": true},
		"model": {"display_name": "Gemini 2.5 Pro"},
		"context_window": {"used_percentage": 42.5},
		"cost": 0.312,
		"plan_tier": "Pro",
		"terminal_width": 200,
		"task_count": 2,
		"subagents": [{"name": "agent1"}, {"name": "agent2"}],
		"sandbox": {"enabled": true, "allow_network": true},
		"vim": {"mode": "NORMAL"},
		"conversation_title": "fix login"
	}`

	cmd := NewStatuslineCmd(env, home, payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	t.Logf("output: %s", line)

	// Verify key segments exist
	checks := []string{
		"✿ gentle ai",
		"⟡",
		"⚙ working",
		"feat/login",
		"*", // dirty marker
		"Gemini 2.5 Pro",
		"ctx",
		"▰", // gauge fill
		"$0.312 sub",
		"⧗ 2 jobs",
		"✦ 2 subagents",
		"⊘ sandbox·net",
		"vim.NORMAL",
		"fix login", // right-aligned title
	}
	for _, check := range checks {
		if !strings.Contains(line, check) {
			t.Errorf("missing segment %q in output: %s", check, line)
		}
	}
}

// TestStatuslineScript_NarrowWidth verifies degradation at narrow terminal widths.
func TestStatuslineScript_NarrowWidth(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	payload := `{
		"agent_state": "working",
		"cwd": "/home/user/project/src/feature",
		"vcs": {"branch": "feat/login", "dirty": true},
		"model": {"display_name": "Gemini 2.5 Pro"},
		"context_window": {"used_percentage": 42.5},
		"cost": 0.312,
		"plan_tier": "Pro",
		"terminal_width": 80,
		"task_count": 2,
		"subagents": [{"name": "agent1"}],
		"sandbox": {"enabled": true, "allow_network": false},
		"vim": {"mode": "NORMAL"},
		"conversation_title": "fix login"
	}`

	cmd := NewStatuslineCmd(env, home, payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	t.Logf("narrow output: %s", line)

	// At narrow width, location should be compacted and some segments dropped
	// Brand and location should survive
	if !strings.Contains(line, "✿ gentle ai") {
		t.Errorf("brand missing at narrow width: %s", line)
	}
	if !strings.Contains(line, "feat/login") && !strings.Contains(line, "feature") {
		t.Errorf("location missing at narrow width: %s", line)
	}
}

// TestStatuslineScript_TitleMode verifies --title output.
func TestStatuslineScript_TitleMode(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	payload := `{
		"vcs": {"branch": "feat/login"},
		"model": {"display_name": "Gemini 2.5 Pro"}
	}`

	cmd := NewStatuslineCmd(env, home, payload, "--title")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	expected := "✿ feat/login · Gemini 2.5 Pro"
	if line != expected {
		t.Errorf("--title output = %q, want %q", line, expected)
	}
}

// TestStatuslineScript_TitleModePartial verifies --title with only branch or only model.
func TestStatuslineScript_TitleModePartial(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	tests := []struct {
		name     string
		payload  string
		expected string
	}{
		{
			name:     "only branch",
			payload:  `{"vcs":{"branch":"main"},"model":{"display_name":""}}`,
			expected: "✿ main",
		},
		{
			name:     "only model",
			payload:  `{"vcs":{"branch":""},"model":{"display_name":"Claude 3.5"}}`,
			expected: "✿ Claude 3.5",
		},
		{
			name:     "neither",
			payload:  `{"vcs":{"branch":""},"model":{"display_name":""}}`,
			expected: "✿",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewStatuslineCmd(env, home, tt.payload, "--title")
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("script exited with error: %v, output: %s", err, output)
			}
			line := strings.TrimSpace(string(output))
			if line != tt.expected {
				t.Errorf("output = %q, want %q", line, tt.expected)
			}
		})
	}
}

// TestStatuslineScript_ConfigHiding verifies config-driven segment hiding.
func TestStatuslineScript_ConfigHiding(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	// Write config with visibility.rdd=false and density=minimal
	configDir := filepath.Join(home, ".gentle-ai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := map[string]any{
		"schema":  "gentle-ai.statusline/v1",
		"density": "minimal",
		"visibility": map[string]bool{
			"changes":        false,
			"rdd":            false,
			"agents":         true,
			"usageCost":      true,
			"modelDetails":   true,
			"contextWarning": true,
		},
	}
	configData, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(filepath.Join(configDir, "statusline.json"), configData, 0o644); err != nil {
		t.Fatal(err)
	}

	payload := `{
		"agent_state": "working",
		"cwd": "/home/user/project",
		"vcs": {"branch": "main", "dirty": false},
		"model": {"display_name": "Gemini 2.5 Pro"},
		"context_window": {"used_percentage": 42},
		"cost": 0.312,
		"plan_tier": "Pro",
		"terminal_width": 200,
		"task_count": 1,
		"subagents": [{"name": "agent1"}],
		"sandbox": {"enabled": true, "allow_network": false},
		"vim": {"mode": "NORMAL"},
		"conversation_title": "test"
	}`

	cmd := NewStatuslineCmd(env, home, payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	t.Logf("config hiding output: %s", line)

	// Minimal density hides brand
	if strings.Contains(line, "✿ gentle ai") {
		t.Errorf("brand should be hidden in minimal density: %s", line)
	}
	// RDD should be hidden
	if strings.Contains(line, "🌹") {
		t.Errorf("RDD should be hidden via visibility.rdd=false: %s", line)
	}
	// Other segments should still show
	for _, check := range []string{"⚙ working", "main", "Gemini 2.5 Pro", "ctx", "⧗ 1 jobs", "✦ 1 subagents", "⊘ sandbox"} {
		if !strings.Contains(line, check) {
			t.Errorf("missing expected segment %q: %s", check, line)
		}
	}
}

// TestStatuslineScript_ConfigDefaults verifies defaults when config is absent/malformed.
func TestStatuslineScript_ConfigDefaults(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	// No config file - defaults should apply
	payload := `{
		"agent_state": "idle",
		"cwd": "/home/user/project",
		"vcs": {"branch": "main", "dirty": false},
		"model": {"display_name": "Gemini 2.5 Pro"},
		"context_window": {"used_percentage": 10},
		"cost": 0.01,
		"terminal_width": 120,
		"task_count": 0,
		"subagents": [],
		"sandbox": {"enabled": false},
		"vim": {},
		"conversation_title": "test"
	}`

	cmd := NewStatuslineCmd(env, home, payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	t.Logf("defaults output: %s", line)

	// Defaults: density comfortable (brand shown), rdd on, agents on, usageCost on, modelDetails on, contextWarning on, changes off
	for _, check := range []string{"✿ gentle ai", "● ready", "main", "Gemini 2.5 Pro", "ctx", "$0.010"} {
		if !strings.Contains(line, check) {
			t.Errorf("missing default segment %q: %s", check, line)
		}
	}
	// RDD should show (default on)
	if !strings.Contains(line, "🌹") {
		t.Errorf("RDD should show by default: %s", line)
	}
	// Changes should NOT show (default off)
	if strings.Contains(line, "±") {
		t.Errorf("changes should be hidden by default: %s", line)
	}
}

// TestStatuslineScript_ConfigMalformed verifies malformed config falls back to defaults.
func TestStatuslineScript_ConfigMalformed(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	// Write malformed JSON
	configDir := filepath.Join(home, ".gentle-ai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "statusline.json"), []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	payload := `{
		"agent_state": "idle",
		"cwd": "/home/user/project",
		"vcs": {"branch": "main", "dirty": false},
		"model": {"display_name": "Gemini 2.5 Pro"},
		"context_window": {"used_percentage": 10},
		"cost": 0.01,
		"terminal_width": 120,
		"task_count": 0,
		"subagents": [],
		"sandbox": {"enabled": false},
		"vim": {},
		"conversation_title": "test"
	}`

	cmd := NewStatuslineCmd(env, home, payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	// Should fall back to defaults (same as ConfigDefaults)
	for _, check := range []string{"✿ gentle ai", "● ready", "main", "Gemini 2.5 Pro", "ctx"} {
		if !strings.Contains(line, check) {
			t.Errorf("missing default segment with malformed config %q: %s", check, line)
		}
	}
}

// TestStatuslineScript_QuotaRendering verifies quota window rendering.
func TestStatuslineScript_QuotaRendering(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	payload := `{
		"agent_state": "idle",
		"cwd": "/home/user/project",
		"vcs": {"branch": "main", "dirty": false},
		"model": {"display_name": "Gemini 2.5 Pro"},
		"context_window": {"used_percentage": 10},
		"cost": 0.01,
		"terminal_width": 150,
		"task_count": 0,
		"subagents": [],
		"sandbox": {"enabled": false},
		"vim": {},
		"conversation_title": "test",
		"quota": {
			"gemini-weekly": {"remaining_fraction": 0.7, "reset_in_seconds": 604800},
			"gemini-daily": {"remaining_fraction": 0.3, "reset_in_seconds": 86400},
			"gemini-hourly": {"remaining_fraction": 0.5, "reset_in_seconds": 3600}
		}
	}`

	cmd := NewStatuslineCmd(env, home, payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	t.Logf("quota output: %s", line)

	// Check quota labels and gauges
	for _, check := range []string{"week", "1d", "1h", "30%", "70%", "50%"} {
		if !strings.Contains(line, check) {
			t.Errorf("missing quota segment %q: %s", check, line)
		}
	}
}

// TestStatuslineScript_HighContextWarning verifies context warning marker at >=90%.
func TestStatuslineScript_HighContextWarning(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	payload := `{
		"agent_state": "idle",
		"cwd": "/home/user/project",
		"vcs": {"branch": "main", "dirty": false},
		"model": {"display_name": "Gemini 2.5 Pro"},
		"context_window": {"used_percentage": 95},
		"cost": 0.01,
		"terminal_width": 120,
		"task_count": 0,
		"subagents": [],
		"sandbox": {"enabled": false},
		"vim": {},
		"conversation_title": "test"
	}`

	cmd := NewStatuslineCmd(env, home, payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	t.Logf("high context output: %s", line)

	// Should have ! marker at >=90%
	if !strings.Contains(line, "!") {
		t.Errorf("missing context warning marker at 95%%: %s", line)
	}
	// Gauge should be red (error color) at >=95%
	if !strings.Contains(line, "\033[31m") {
		t.Errorf("gauge should be red at 95%%: %s", line)
	}
}

// TestStatuslineScript_RDDHiddenViaConfig verifies RDD segment hidden via visibility.rdd:false.
func TestStatuslineScript_RDDHiddenViaConfig(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	// Write config with visibility.rdd=false
	configDir := filepath.Join(home, ".gentle-ai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := map[string]any{
		"schema":  "gentle-ai.statusline/v1",
		"density": "comfortable",
		"visibility": map[string]bool{
			"rdd": false,
		},
	}
	configData, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(filepath.Join(configDir, "statusline.json"), configData, 0o644); err != nil {
		t.Fatal(err)
	}

	payload := `{
		"agent_state": "idle",
		"cwd": "/home/user/project",
		"vcs": {"branch": "main", "dirty": false},
		"model": {"display_name": "Gemini 2.5 Pro"},
		"context_window": {"used_percentage": 10},
		"cost": 0.01,
		"terminal_width": 120,
		"task_count": 0,
		"subagents": [],
		"sandbox": {"enabled": false},
		"vim": {},
		"conversation_title": "test"
	}`

	cmd := NewStatuslineCmd(env, home, payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	if strings.Contains(line, "🌹") {
		t.Errorf("RDD should be hidden via visibility.rdd=false: %s", line)
	}
}

// TestStatuslineScript_AgentStateMapping verifies all agent_state enum mappings.
func TestStatuslineScript_AgentStateMapping(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	tests := []struct {
		state    string
		expected string
	}{
		{"idle", "● ready"},
		{"thinking", "◆ thinking"},
		{"working", "⚙ working"},
		{"tool_use", "🔧 tool"},
		{"initializing", "⏳ initializing"},
		{"unknown_state", "○ unknown_state"},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			payload := map[string]any{
				"agent_state":        tt.state,
				"cwd":                "/home/user/project",
				"vcs":                map[string]any{"branch": "main", "dirty": false},
				"model":              map[string]string{"display_name": "Gemini 2.5 Pro"},
				"context_window":     map[string]float64{"used_percentage": 10},
				"cost":               0.01,
				"terminal_width":     120,
				"task_count":         0,
				"subagents":          []any{},
				"sandbox":            map[string]bool{"enabled": false},
				"vim":                map[string]string{},
				"conversation_title": "test",
			}
			payloadBytes, _ := json.Marshal(payload)
			cmd := NewStatuslineCmd(env, home, string(payloadBytes))
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("script exited with error: %v, output: %s", err, output)
			}
			line := strings.TrimSpace(string(output))
			if !strings.Contains(line, tt.expected) {
				t.Errorf("state %q: expected badge %q not found in %s", tt.state, tt.expected, line)
			}
		})
	}
}

// TestStatuslineScript_ChangesSegment verifies changes segment when visibility.changes=true.
func TestStatuslineScript_ChangesSegment(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	// Initialize a git repo in temp dir
	cwd := filepath.Join(home, "repo")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	// Init git repo
	runGit(t, cwd, "init")
	runGit(t, cwd, "config", "user.email", "test@test.com")
	runGit(t, cwd, "config", "user.name", "Test User")
	// Create a file and commit
	if err := os.WriteFile(filepath.Join(cwd, "file.txt"), []byte("initial"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, cwd, "add", "file.txt")
	runGit(t, cwd, "commit", "-m", "initial")
	// Modify file (staged)
	if err := os.WriteFile(filepath.Join(cwd, "file.txt"), []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, cwd, "add", "file.txt")
	// Create untracked file
	if err := os.WriteFile(filepath.Join(cwd, "untracked.txt"), []byte("untracked"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write config with visibility.changes=true
	configDir := filepath.Join(home, ".gentle-ai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := map[string]any{
		"schema":  "gentle-ai.statusline/v1",
		"density": "comfortable",
		"visibility": map[string]bool{
			"changes": true,
		},
	}
	configData, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(filepath.Join(configDir, "statusline.json"), configData, 0o644); err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{
		"agent_state":        "idle",
		"cwd":                cwd,
		"vcs":                map[string]any{"branch": "main", "dirty": true},
		"model":              map[string]string{"display_name": "Gemini 2.5 Pro"},
		"context_window":     map[string]float64{"used_percentage": 10},
		"cost":               0.01,
		"terminal_width":     150,
		"task_count":         0,
		"subagents":          []any{},
		"sandbox":            map[string]bool{"enabled": false},
		"vim":                map[string]string{},
		"conversation_title": "test",
	}
	payloadBytes, _ := json.Marshal(payload)

	cmd := NewStatuslineCmd(env, home, string(payloadBytes))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	t.Logf("changes output: %s", line)

	// Should show changes segment: ±N +A/−D
	if !strings.Contains(line, "±") {
		t.Errorf("changes segment missing: %s", line)
	}
	if !strings.Contains(line, "+") || !strings.Contains(line, "−") {
		t.Errorf("changes segment missing +/- counts: %s", line)
	}
}

// TestStatuslineScript_NoGitRepo verifies changes segment hidden in non-git dir.
func TestStatuslineScript_NoGitRepo(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	cwd := filepath.Join(home, "not-a-repo")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}

	configDir := filepath.Join(home, ".gentle-ai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := map[string]any{
		"schema":  "gentle-ai.statusline/v1",
		"density": "comfortable",
		"visibility": map[string]bool{
			"changes": true,
		},
	}
	configData, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(filepath.Join(configDir, "statusline.json"), configData, 0o644); err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{
		"agent_state":        "idle",
		"cwd":                cwd,
		"vcs":                map[string]any{"branch": "main", "dirty": true},
		"model":              map[string]string{"display_name": "Gemini 2.5 Pro"},
		"context_window":     map[string]float64{"used_percentage": 10},
		"cost":               0.01,
		"terminal_width":     150,
		"task_count":         0,
		"subagents":          []any{},
		"sandbox":            map[string]bool{"enabled": false},
		"vim":                map[string]string{},
		"conversation_title": "test",
	}
	payloadBytes, _ := json.Marshal(payload)

	cmd := NewStatuslineCmd(env, home, string(payloadBytes))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	// Should NOT show changes segment in non-git dir
	if strings.Contains(line, "±") {
		t.Errorf("changes segment should be hidden in non-git dir: %s", line)
	}
}

// TestStatuslineScript_UsageCostFalse verifies usageCost:false hides ctx+cost+quota.
func TestStatuslineScript_UsageCostFalse(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	configDir := filepath.Join(home, ".gentle-ai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := map[string]any{
		"schema":  "gentle-ai.statusline/v1",
		"density": "comfortable",
		"visibility": map[string]bool{
			"usageCost": false,
		},
	}
	configData, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(filepath.Join(configDir, "statusline.json"), configData, 0o644); err != nil {
		t.Fatal(err)
	}

	payload := `{
		"agent_state": "idle",
		"cwd": "/home/user/project",
		"vcs": {"branch": "main", "dirty": false},
		"model": {"display_name": "Gemini 2.5 Pro"},
		"context_window": {"used_percentage": 50},
		"cost": 1.50,
		"plan_tier": "Pro",
		"terminal_width": 120,
		"task_count": 0,
		"subagents": [],
		"sandbox": {"enabled": false},
		"vim": {},
		"conversation_title": "test",
		"quota": {"gemini-weekly": {"remaining_fraction": 0.5, "reset_in_seconds": 604800}}
	}`

	cmd := NewStatuslineCmd(env, home, payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	// ctx, cost, quota should all be hidden
	for _, check := range []string{"ctx", "$1.50", "week"} {
		if strings.Contains(line, check) {
			t.Errorf("segment %q should be hidden with usageCost:false: %s", check, line)
		}
	}
}

// TestStatuslineScript_ModelDetailsFalse verifies modelDetails:false hides model segment.
func TestStatuslineScript_ModelDetailsFalse(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	configDir := filepath.Join(home, ".gentle-ai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := map[string]any{
		"schema":  "gentle-ai.statusline/v1",
		"density": "comfortable",
		"visibility": map[string]bool{
			"modelDetails": false,
		},
	}
	configData, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(filepath.Join(configDir, "statusline.json"), configData, 0o644); err != nil {
		t.Fatal(err)
	}

	payload := `{
		"agent_state": "idle",
		"cwd": "/home/user/project",
		"vcs": {"branch": "main", "dirty": false},
		"model": {"display_name": "Gemini 2.5 Pro"},
		"context_window": {"used_percentage": 10},
		"cost": 0.01,
		"terminal_width": 120,
		"task_count": 0,
		"subagents": [],
		"sandbox": {"enabled": false},
		"vim": {},
		"conversation_title": "test"
	}`

	cmd := NewStatuslineCmd(env, home, payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	if strings.Contains(line, "Gemini 2.5 Pro") {
		t.Errorf("model segment should be hidden with modelDetails:false: %s", line)
	}
}

// TestStatuslineScript_AgentsFalse verifies agents:false hides jobs+subagents.
func TestStatuslineScript_AgentsFalse(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	configDir := filepath.Join(home, ".gentle-ai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := map[string]any{
		"schema":  "gentle-ai.statusline/v1",
		"density": "comfortable",
		"visibility": map[string]bool{
			"agents": false,
		},
	}
	configData, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(filepath.Join(configDir, "statusline.json"), configData, 0o644); err != nil {
		t.Fatal(err)
	}

	payload := `{
		"agent_state": "idle",
		"cwd": "/home/user/project",
		"vcs": {"branch": "main", "dirty": false},
		"model": {"display_name": "Gemini 2.5 Pro"},
		"context_window": {"used_percentage": 10},
		"cost": 0.01,
		"terminal_width": 120,
		"task_count": 3,
		"subagents": [{"name": "a1"}, {"name": "a2"}],
		"sandbox": {"enabled": false},
		"vim": {},
		"conversation_title": "test"
	}`

	cmd := NewStatuslineCmd(env, home, payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	for _, check := range []string{"⧗ 3 jobs", "✦ 2 subagents"} {
		if strings.Contains(line, check) {
			t.Errorf("segment %q should be hidden with agents:false: %s", check, line)
		}
	}
}

// TestStatuslineScript_ContextWarningFalse verifies contextWarning:false hides ! marker.
func TestStatuslineScript_ContextWarningFalse(t *testing.T) {
	home := t.TempDir()
	env := scrubEnv(home)

	configDir := filepath.Join(home, ".gentle-ai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := map[string]any{
		"schema":  "gentle-ai.statusline/v1",
		"density": "comfortable",
		"visibility": map[string]bool{
			"contextWarning": false,
		},
	}
	configData, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(filepath.Join(configDir, "statusline.json"), configData, 0o644); err != nil {
		t.Fatal(err)
	}

	payload := `{
		"agent_state": "idle",
		"cwd": "/home/user/project",
		"vcs": {"branch": "main", "dirty": false},
		"model": {"display_name": "Gemini 2.5 Pro"},
		"context_window": {"used_percentage": 95},
		"cost": 0.01,
		"terminal_width": 120,
		"task_count": 0,
		"subagents": [],
		"sandbox": {"enabled": false},
		"vim": {},
		"conversation_title": "test"
	}`

	cmd := NewStatuslineCmd(env, home, payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script exited with error: %v, output: %s", err, output)
	}

	line := strings.TrimSpace(string(output))
	if strings.Contains(line, "!") {
		t.Errorf("! marker should be hidden with contextWarning:false: %s", line)
	}
}

// NewStatuslineCmd creates a command to run the statusline script with given env and payload.
func NewStatuslineCmd(env []string, home, payload string, args ...string) *exec.Cmd {
	// Resolve the script against the repository root derived from this test
	// file's location: go test runs with the package directory as cwd.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("statusline test: cannot locate source file")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	scriptPath := filepath.Join(repoRoot, "internal", "assets", "antigravitycli", "statusline.sh")
	cmdArgs := append([]string{"bash", scriptPath}, args...)
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Env = env
	cmd.Dir = repoRoot
	// Write payload to stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	go func() {
		defer stdin.Close()
		stdin.Write([]byte(payload))
	}()
	return cmd
}

func scrubEnv(home string) []string {
	env := os.Environ()
	var out []string
	for _, e := range env {
		if strings.HasPrefix(e, "PI_CODING_AGENT_DIR=") || strings.HasPrefix(e, "GENTLE_PI_AGENT_HOME=") {
			continue
		}
		out = append(out, e)
	}
	out = append(out, "HOME="+home)
	return out
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
