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
		"⟡", "working…", "~/project/src/feature feat/login*",
		"Gemini 3.5 Pro (High) · planning", "ctx ", "43%", "$0.312",
		"weekly", "30%", "⧗ 2 jobs", "✦ 2 subagents", "⊘ sandbox·net",
		"vim.NORMAL", "🌹 on", "fix login",
	} {
		if !strings.Contains(stripANSI(line), want) {
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
		"model": {"id": "gemini-3.5-pro", "display_name": "Gemini 3.5 Pro"},
		"quota": {
			"gemini-weekly": {"remaining_fraction": 0.7, "reset_in_seconds": 604800},
			"gemini-daily": {"remaining_fraction": 0.3, "reset_in_seconds": 86400},
			"gemini-hourly": {"remaining_fraction": 0.5, "reset_in_seconds": 3600}
		},
		"terminal_width": 220,
		"context_window": {"used_percentage": 10}
	}`)
	line := Render(payload, DefaultConfig(), defaultExtras())
	for _, want := range []string{"weekly", "daily", "hourly", "30%", "70%", "50%"} {
		if !strings.Contains(line, want) {
			t.Errorf("missing quota marker %q in: %s", want, line)
		}
	}
}

// TestRenderQuotaFiltersInactiveFamily pins the L13 fix: only the active
// model's provider buckets render, never both providers at once.
func TestRenderQuotaFiltersInactiveFamily(t *testing.T) {
	payload := mustPayload(t, `{
		"cwd": "/home/user/project",
		"model": {"id": "gemini-3.5-pro", "display_name": "Gemini 3.5 Pro"},
		"quota": {
			"gemini-weekly": {"remaining_fraction": 0.7, "reset_in_seconds": 604800},
			"claude-5h": {"remaining_fraction": 0.2, "reset_in_seconds": 18000}
		},
		"terminal_width": 220,
		"context_window": {"used_percentage": 10}
	}`)
	line := Render(payload, DefaultConfig(), defaultExtras())
	if !strings.Contains(line, "weekly") {
		t.Errorf("active family bucket missing: %s", line)
	}
	if strings.Contains(line, "claude") || strings.Contains(line, "5h") {
		t.Errorf("inactive family bucket shown: %s", line)
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
	if !strings.Contains(line, "Gemini 3.5 Pro") {
		t.Errorf("minimal density lost the content segments: %s", line)
	}
	if strings.Contains(stripANSI(line), "…") {
		t.Errorf("minimal idle shows an activity label: %s", line)
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
		{"usageCost", func(c *Config) { c.Visibility.UsageCost = false }, []string{"ctx", "$0.312", "weekly"}},
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

// TestRenderPetalState pins the merged brand/state petal: idle keeps the
// static brand with no label; active spins a petal frame with the fallback
// label; a hook-provided phase wins over the fallback.
func TestRenderPetalState(t *testing.T) {
	idle := mustPayload(t, `{"cwd": "/home/user/project", "terminal_width": 200, "agent_state": "idle"}`)
	line := Render(idle, DefaultConfig(), defaultExtras())
	if !strings.Contains(line, brandText) {
		t.Errorf("idle lost the static brand: %s", line)
	}
	if strings.Contains(stripANSI(line), "…") {
		t.Errorf("idle shows an activity label: %s", line)
	}

	active := mustPayload(t, `{"cwd": "/home/user/project", "terminal_width": 200, "agent_state": "thinking"}`)
	line = Render(active, DefaultConfig(), defaultExtras())
	if !strings.Contains(stripANSI(line), "thinking…") {
		t.Errorf("active state missing fallback label: %s", line)
	}
	petalSeen := false
	for _, frame := range petalFrames {
		if strings.Contains(line, frame) {
			petalSeen = true
		}
	}
	if !petalSeen {
		t.Errorf("active state missing animated petal: %s", line)
	}

	phased := Extras{Home: "/home/user", Phase: "exploring"}
	line = Render(active, DefaultConfig(), phased)
	if !strings.Contains(stripANSI(line), "exploring…") {
		t.Errorf("hook phase label missing: %s", line)
	}
	if strings.Contains(stripANSI(line), "gentle ai") {
		t.Errorf("active state shows the idle brand: %s", line)
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

// TestQuotaLabelFromKey pins the key-derived window labels (never from
// reset_in_seconds, which is a countdown).
func TestQuotaLabelFromKey(t *testing.T) {
	cases := []struct{ key, family, want string }{
		{"gemini-weekly", "gemini", "weekly"},
		{"gemini-daily", "gemini", "daily"},
		{"claude-5h", "claude", "5h"},
		{"claude-5h", "gemini", "claude-5h"},
		{"unknown-shape", "", "unknown-shape"},
		{"-weird-", "", "weird"},
	}
	for _, testCase := range cases {
		if got := quotaLabelFromKey(testCase.key, testCase.family); got != testCase.want {
			t.Errorf("quotaLabelFromKey(%q, %q) = %q, want %q", testCase.key, testCase.family, got, testCase.want)
		}
	}
}

// TestModelFamily pins the provider-family extraction with the display-name
// fallback.
func TestModelFamily(t *testing.T) {
	cases := []struct{ id, display, want string }{
		{"gemini-3.5-pro", "Gemini 3.5 Pro", "gemini"},
		{"", "Gemini 3.5 Pro (High)", "gemini"},
		{"claude-opus-4", "Claude Opus", "claude"},
		{"", "", ""},
	}
	for _, testCase := range cases {
		if got := modelFamily(testCase.id, testCase.display); got != testCase.want {
			t.Errorf("modelFamily(%q, %q) = %q, want %q", testCase.id, testCase.display, got, testCase.want)
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

// TestFormatCost pins the plain cost formatting (no suffix, L13).
func TestFormatCost(t *testing.T) {
	if got := formatCost(0.312); got != "$0.312" {
		t.Errorf("formatCost sub-dollar = %q", got)
	}
	if got := formatCost(1.5); got != "$1.50" {
		t.Errorf("formatCost dollar = %q", got)
	}
}

// TestRenderMemSegment pins the Engram segment: 🧠 project · count, the
// count-only fallback, and the visibility/zero gates.
func TestRenderMemSegment(t *testing.T) {
	payload := mustPayload(t, `{
		"cwd": "/home/user/project",
		"model": {"display_name": "M"},
		"context_window": {"used_percentage": 10},
		"terminal_width": 200
	}`)
	extras := defaultExtras()
	extras.Observations = 918
	extras.EngramProject = "gentle-ai"
	line := Render(payload, DefaultConfig(), extras)
	if !strings.Contains(line, "🧠 gentle-ai · 918") {
		t.Errorf("engram segment missing: %s", line)
	}

	extras.EngramProject = ""
	line = Render(payload, DefaultConfig(), extras)
	if !strings.Contains(line, "🧠 918") {
		t.Errorf("count-only engram segment missing: %s", line)
	}

	noEngram := DefaultConfig()
	noEngram.Visibility.Engram = false
	extras.EngramProject = "gentle-ai"
	line = Render(payload, noEngram, extras)
	if strings.Contains(line, "🧠") {
		t.Errorf("engram segment shown while disabled: %s", line)
	}

	extras.Observations = 0
	line = Render(payload, DefaultConfig(), extras)
	if strings.Contains(line, "🧠") {
		t.Errorf("engram segment shown with zero observations: %s", line)
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

// TestScannerWave pins the exact per-character roles at a known render tick.
func TestScannerWave(t *testing.T) {
	got := scanWorkingLabel("abcde", 5) // phase 5, head at index 2
	want := ansiBrightCyan + "a" + ansiReset + ansiYellow + "b" + ansiReset + ansiBrightWhite + "c" + ansiReset + ansiYellow + "d" + ansiReset + ansiBrightCyan + "e" + ansiReset
	if got != want {
		t.Fatalf("scanner = %q, want %q", got, want)
	}
	if scanWorkingLabel("abcde", 20) != got {
		t.Fatal("scanner did not repeat at 15 steps")
	}
	if stripANSI(scanWorkingLabel("exploring…", 0)) != "exploring…" {
		t.Fatal("scanner altered visible label")
	}
}
