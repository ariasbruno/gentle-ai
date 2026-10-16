package statusline

import (
	"strings"
	"testing"
)

func mustPayload(t *testing.T, raw string) *Payload {
	t.Helper()
	payload, err := ParsePayload([]byte(raw))
	if err != nil {
		t.Fatalf("parse payload: %v", err)
	}
	return payload
}

func defaultExtras() Extras {
	return Extras{Home: "/home/user", RDDMode: "on"}
}

// TestRenderFullLine pins the comfortable full line: every visible segment
// and the right-aligned conversation title.
func TestRenderFullLine(t *testing.T) {
	payload := mustPayload(t, `{
		"agent_state": "working",
		"cwd": "/home/user/project/src/feature",
		"conversation_title": "fix login",
		"model": {"display_name": "Gemini 3.5 Pro (High)"},
		"context_window": {"used_percentage": 42.5},
		"cost": 0.312,
		"quota": {"gemini-weekly": {"remaining_fraction": 0.7, "reset_in_seconds": 86400}},
		"vcs": {"branch": "feat/login", "dirty": true},
		"sandbox": {"enabled": true, "allow_network": true},
		"task_count": 2,
		"subagents": [{"name": "agent1"}, {"name": "agent2"}],
		"terminal_width": 220,
		"execution_mode": "planning",
		"plan_tier": "Pro",
		"vim": {"mode": "NORMAL"}
	}`)

	line := Render(payload, DefaultConfig(), defaultExtras())
	for _, want := range []string{
		brandText, "⟡", "⚙ working", "~/project/src/feature feat/login*",
		"Gemini 3.5 Pro (High) · planning", "ctx ", "43%", "$0.312 sub",
		"1d", "30%", "⧗ 2 jobs", "✦ 2 subagents", "⊘ sandbox·net",
		"vim.NORMAL", "🌹 on", "fix login",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("missing segment %q in: %s", want, line)
		}
	}
	if got := visibleWidth(line); got != 220 {
		t.Errorf("right-aligned line width = %d, want exactly the terminal width 220", got)
	}
}

// TestRenderQuotaWindows pins the window labels and used percentages for
// week/day/hour buckets.
func TestRenderQuotaWindows(t *testing.T) {
	payload := mustPayload(t, `{
		"cwd": "/home/user/project",
		"quota": {
			"gemini-weekly": {"remaining_fraction": 0.7, "reset_in_seconds": 604800},
			"gemini-daily": {"remaining_fraction": 0.3, "reset_in_seconds": 86400},
			"gemini-hourly": {"remaining_fraction": 0.5, "reset_in_seconds": 3600}
		},
		"terminal_width": 220,
		"context_window": {"used_percentage": 10},
		"model": {"display_name": "Gemini 3.5 Pro"}
	}`)
	line := Render(payload, DefaultConfig(), defaultExtras())
	for _, want := range []string{"week", "1d", "1h", "30%", "70%", "50%"} {
		if !strings.Contains(line, want) {
			t.Errorf("missing quota marker %q in: %s", want, line)
		}
	}
}

// TestRenderHighContextWarning pins the error-tone gauge and the >=90 marker.
func TestRenderHighContextWarning(t *testing.T) {
	payload := mustPayload(t, `{
		"cwd": "/home/user/project",
		"context_window": {"used_percentage": 95},
		"terminal_width": 200,
		"model": {"display_name": "Gemini 3.5 Pro"}
	}`)
	line := Render(payload, DefaultConfig(), defaultExtras())
	if !strings.Contains(line, ansiError) {
		t.Errorf("gauge not in error color at 95%%: %s", line)
	}
	if !strings.Contains(line, "95% !") {
		t.Errorf("context warning marker missing: %s", line)
	}

	noWarn := DefaultConfig()
	noWarn.Visibility.ContextWarning = false
	line = Render(payload, noWarn, defaultExtras())
	if strings.Contains(line, "!") {
		t.Errorf("context warning shown while disabled: %s", line)
	}
}

// TestRenderDegradationNarrow pins the width-driven cascade: compact location
// with a clipped branch, then dropped auxiliary segments.
func TestRenderDegradationNarrow(t *testing.T) {
	payload := mustPayload(t, `{
		"cwd": "/home/user/dev/very/deep/project/directory",
		"conversation_title": "fix login",
		"model": {"display_name": "Gemini 3.5 Pro (High)"},
		"context_window": {"used_percentage": 42.5},
		"cost": 0.312,
		"plan_tier": "Pro",
		"quota": {"gemini-weekly": {"remaining_fraction": 0.7, "reset_in_seconds": 604800}},
		"vcs": {"branch": "chore/agy-integration-ai-v4.1.1-shell-v4.2.0", "dirty": true},
		"task_count": 2,
		"terminal_width": 110,
		"vim": {"mode": "NORMAL"}
	}`)
	line := Render(payload, DefaultConfig(), defaultExtras())
	if !strings.Contains(line, "directory chore/agy-inte…*") {
		t.Errorf("location not compacted with clipped branch: %s", line)
	}
	if visibleWidth(line) > 110 {
		t.Errorf("degraded line still overflows: %d cols: %s", visibleWidth(line), line)
	}
}

// TestRenderMinimalHidesBrand pins the minimal density behavior.
func TestRenderMinimalHidesBrand(t *testing.T) {
	payload := mustPayload(t, `{
		"cwd": "/home/user/project",
		"model": {"display_name": "Gemini 3.5 Pro"},
		"context_window": {"used_percentage": 10},
		"terminal_width": 200
	}`)
	config := DefaultConfig()
	config.Density = DensityMinimal
	line := Render(payload, config, defaultExtras())
	if strings.Contains(line, brandText) {
		t.Errorf("minimal density still shows the brand: %s", line)
	}
	if !strings.Contains(line, "● ready") {
		t.Errorf("minimal density lost the state badge: %s", line)
	}
}

// TestRenderVisibilityHiding pins each visibility flag's segment group.
func TestRenderVisibilityHiding(t *testing.T) {
	payload := mustPayload(t, `{
		"cwd": "/home/user/project",
		"model": {"display_name": "Gemini 3.5 Pro"},
		"context_window": {"used_percentage": 42.5},
		"cost": 0.312,
		"plan_tier": "Pro",
		"quota": {"gemini-weekly": {"remaining_fraction": 0.7, "reset_in_seconds": 86400}},
		"task_count": 2,
		"terminal_width": 220
	}`)

	cases := []struct {
		name   string
		mutat  func(*Config)
		absent []string
	}{
		{"modelDetails", func(c *Config) { c.Visibility.ModelDetails = false }, []string{"Gemini 3.5 Pro"}},
		{"usageCost", func(c *Config) { c.Visibility.UsageCost = false }, []string{"ctx", "$0.312", "1d"}},
		{"agents", func(c *Config) { c.Visibility.Agents = false }, []string{"⧗ 2 jobs"}},
	}
	for _, testCase := range cases {
		config := DefaultConfig()
		testCase.mutat(&config)
		line := Render(payload, config, defaultExtras())
		for _, absent := range testCase.absent {
			if strings.Contains(line, absent) {
				t.Errorf("%s: segment %q still shown: %s", testCase.name, absent, line)
			}
		}
	}

	rddHidden := DefaultConfig()
	rddHidden.Visibility.RDD = false
	line := Render(payload, rddHidden, defaultExtras())
	if strings.Contains(line, "🌹") {
		t.Errorf("rdd: segment still shown: %s", line)
	}
}

// TestRenderStateBadges pins the agent_state mapping.
func TestRenderStateBadges(t *testing.T) {
	cases := map[string]string{
		"idle":         "● ready",
		"":             "● ready",
		"thinking":     "◆ thinking",
		"working":      "⚙ working",
		"tool_use":     "🔧 tool",
		"initializing": "⏳ initializing",
		"weird":        "○ weird",
	}
	for state, want := range cases {
		payload := &Payload{AgentState: state, TerminalWidth: 200}
		if line := Render(payload, DefaultConfig(), defaultExtras()); !strings.Contains(line, want) {
			t.Errorf("state %q: want badge %q in: %s", state, want, line)
		}
	}
}

// TestRenderFallback pins the brand-only line for nil payloads.
func TestRenderFallback(t *testing.T) {
	if got := Render(nil, DefaultConfig(), defaultExtras()); got != FallbackLine {
		t.Errorf("Render(nil) = %q, want %q", got, FallbackLine)
	}
	if FallbackLine != "✿ gentle ai" {
		t.Errorf("FallbackLine = %q", FallbackLine)
	}
}

// TestRenderTitleRightAligned pins the exact-width padding of the title.
func TestRenderTitleRightAligned(t *testing.T) {
	payload := mustPayload(t, `{
		"cwd": "/home/user/project",
		"model": {"display_name": "M"},
		"context_window": {"used_percentage": 10},
		"conversation_title": "fix login",
		"terminal_width": 120
	}`)
	line := Render(payload, DefaultConfig(), defaultExtras())
	plain := stripANSI(line)
	if !strings.HasSuffix(plain, "fix login") {
		t.Fatalf("title not right-aligned at the end: %q", plain)
	}
	if got := visibleWidth(line); got != 120 {
		t.Errorf("aligned width = %d, want 120", got)
	}
}

// TestRenderChangesSegment pins the opt-in changes segment rendering.
func TestRenderChangesSegment(t *testing.T) {
	payload := mustPayload(t, `{
		"cwd": "/home/user/project",
		"model": {"display_name": "M"},
		"context_window": {"used_percentage": 10},
		"terminal_width": 200
	}`)
	config := DefaultConfig()
	config.Visibility.Changes = true
	line := Render(payload, config, Extras{Home: "/home/user", Changes: "±3 +10/−2"})
	if !strings.Contains(line, "±3 +10/−2") {
		t.Errorf("changes segment missing: %s", line)
	}

	config.Visibility.Changes = false
	line = Render(payload, config, Extras{Home: "/home/user", Changes: "±3 +10/−2"})
	if strings.Contains(line, "±3") {
		t.Errorf("changes segment shown while disabled: %s", line)
	}
}

// TestRenderTitleMode pins the --title output with partial fallbacks.
func TestRenderTitleMode(t *testing.T) {
	full := mustPayload(t, `{"vcs": {"branch": "feat/x"}, "model": {"display_name": "Gemini 3.5 Pro"}}`)
	if got := RenderTitle(full); got != "✿ feat/x · Gemini 3.5 Pro" {
		t.Errorf("RenderTitle full = %q", got)
	}
	branchOnly := mustPayload(t, `{"vcs": {"branch": "feat/x"}}`)
	if got := RenderTitle(branchOnly); got != "✿ feat/x" {
		t.Errorf("RenderTitle branch-only = %q", got)
	}
	modelOnly := mustPayload(t, `{"model": {"display_name": "Gemini 3.5 Pro"}}`)
	if got := RenderTitle(modelOnly); got != "✿ Gemini 3.5 Pro" {
		t.Errorf("RenderTitle model-only = %q", got)
	}
	if got := RenderTitle(nil); got != "✿" {
		t.Errorf("RenderTitle nil = %q", got)
	}
}

// TestQuotaLabel pins the window label derivation.
func TestQuotaLabel(t *testing.T) {
	cases := map[int64]string{
		604800: "week",
		172800: "2d",
		18000:  "5h",
		90:     "2m",
		30:     "1m",
	}
	for seconds, want := range cases {
		if got := quotaLabel(seconds); got != want {
			t.Errorf("quotaLabel(%d) = %q, want %q", seconds, got, want)
		}
	}
}

// TestCompactLocation pins the degradation location compaction.
func TestCompactLocation(t *testing.T) {
	if got := compactLocation("/home/user", "/home/user", "", false); got != "~" {
		t.Errorf("home location = %q, want ~", got)
	}
	if got := compactLocation("/home/user/dev/proj", "/home/user", "feat/login", true); got != "proj feat/login*" {
		t.Errorf("compact location = %q", got)
	}
	long := "chore/agy-integration-ai-v4.1.1-shell-v4.2.0"
	got := compactLocation("/home/user/dev/proj", "/home/user", long, false)
	if !strings.Contains(got, "chore/agy-inte…") {
		t.Errorf("long branch not clipped to 15: %q", got)
	}
}

// TestFormatCost pins the cost formatting and subscription suffix.
func TestFormatCost(t *testing.T) {
	if got := formatCost(0.312, false); got != "$0.312" {
		t.Errorf("formatCost sub-dollar = %q", got)
	}
	if got := formatCost(1.5, false); got != "$1.50" {
		t.Errorf("formatCost dollar = %q", got)
	}
	if got := formatCost(0.312, true); got != "$0.312 sub" {
		t.Errorf("formatCost subscription = %q", got)
	}
}

// TestSubagentCount pins the defensive subagents handling.
func TestSubagentCount(t *testing.T) {
	cases := map[string]int{
		`[{"name":"a"},{"name":"b"}]`: 2,
		`[]`:                          0,
		`{}`:                          0,
		`"nope"`:                      0,
		``:                            0,
	}
	for raw, want := range cases {
		source := `{"subagents": ` + raw + `}`
		if raw == "" {
			source = `{"cwd": "/p"}`
		}
		payload := mustPayload(t, source)
		if got := payload.SubagentCount(); got != want {
			t.Errorf("SubagentCount(%q) = %d, want %d", raw, got, want)
		}
	}
}
