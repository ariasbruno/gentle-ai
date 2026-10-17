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
	return Extras{RDDMode: "on"}
}

// TestRenderFullLine pins the comfortable full line: the bare petal with the
// mode·model block first, then ctx, cost, quota, jobs, subagents, engram,
// rdd, and changes — and nothing that L19 removed.
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
	plain := stripANSI(line)
	for _, want := range []string{
		"planning · Gemini 3.5 Pro (High)", "ctx ", "43%", "$0.312",
		"1d0h", "30%", "⧗ 2 jobs", "✦ 2 subagents", "🌹 on",
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("missing segment %q in: %s", want, line)
		}
	}
	for _, banned := range []string{"gentle ai", "feat/login", "~/", "⊘ sandbox", "vim.NORMAL", "fix login", "…"} {
		if strings.Contains(plain, banned) {
			t.Errorf("removed segment %q still rendered: %s", banned, line)
		}
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
	for _, want := range []string{"7d0h", "1d0h", "1h0m", "30%", "70%", "50%"} {
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
	if !strings.Contains(line, "7d0h") {
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

// TestRenderCascade pins the L18 cascade: quota compacts then drops first,
// then cost, then ctx; the petal block always survives and the line always
// fits the terminal width.
func TestRenderCascade(t *testing.T) {
	payload := mustPayload(t, `{
		"agent_state": "working",
		"model": {"display_name": "Gemini 3.5 Pro (High)"},
		"context_window": {"used_percentage": 42.5},
		"cost": 0.312,
		"quota": {"gemini-weekly": {"remaining_fraction": 0.7, "reset_in_seconds": 604800}},
		"task_count": 2,
		"subagents": [{"name": "a"}, {"name": "b"}],
		"terminal_width": 200,
		"execution_mode": "planning"
	}`)
	extras := defaultExtras()
	extras.Observations = 918
	extras.EngramProject = "gentle-ai"

	full := Render(payload, DefaultConfig(), extras)
	plainFull := stripANSI(full)
	for _, want := range []string{"7d0h", "$0.312", "ctx ", "⧗ 2 jobs", "✦ 2 subagents", "🧠 gentle-ai · 918", "🌹 on"} {
		if !strings.Contains(plainFull, want) {
			t.Fatalf("wide line missing %q: %s", want, full)
		}
	}

	// Narrow enough that the quota bars overflow: quota drops, everything
	// above it in priority survives, and the line fits.
	narrow := *payload
	narrow.TerminalWidth = 60
	line := Render(&narrow, DefaultConfig(), extras)
	plain := stripANSI(line)
	if strings.Contains(plain, "7d0h") || strings.Contains(plain, "▰") {
		t.Errorf("quota survived the cascade at 60 cols: %s", line)
	}
	if strings.Contains(plain, "$0.312") || strings.Contains(plain, "ctx ") {
		t.Errorf("cost/ctx should drop after quota: %s", line)
	}
	if !strings.Contains(plain, "🌹 on") || !strings.Contains(plain, "🧠 918") {
		t.Errorf("rdd/engram dropped before usage segments: %s", line)
	}
	if !strings.Contains(plain, "planning · Gemini 3.5 Pro (High)") {
		t.Errorf("petal block did not survive: %s", line)
	}
	if visibleWidth(line) > 60 {
		t.Errorf("cascaded line overflows: %d cols: %s", visibleWidth(line), line)
	}

	// Extremely narrow: only the petal block survives, truncated.
	tiny := narrow
	tiny.TerminalWidth = 20
	line = Render(&tiny, DefaultConfig(), extras)
	if visibleWidth(line) > 20 {
		t.Errorf("tiny line overflows: %d cols: %s", visibleWidth(line), line)
	}
	tinyPlain := stripANSI(line)
	petalSeen := false
	for _, frame := range petalFrames {
		if strings.Contains(tinyPlain, frame) {
			petalSeen = true
		}
	}
	if !petalSeen {
		t.Errorf("tiny line lost the petal: %s", line)
	}
}

// TestRenderMinimal pins the minimal density: only the petal block, tasklist,
// rdd, and engram survive; usage segments and agents never render.
func TestRenderMinimal(t *testing.T) {
	payload := mustPayload(t, `{
		"agent_state": "working",
		"model": {"display_name": "Gemini 3.5 Pro"},
		"context_window": {"used_percentage": 10},
		"cost": 0.312,
		"quota": {"gemini-weekly": {"remaining_fraction": 0.7, "reset_in_seconds": 86400}},
		"task_count": 2,
		"terminal_width": 200
	}`)
	extras := defaultExtras()
	extras.Observations = 918
	config := DefaultConfig()
	config.Density = DensityMinimal
	line := Render(payload, config, extras)
	plain := stripANSI(line)
	if !strings.Contains(plain, "Gemini 3.5 Pro") || !strings.Contains(plain, "🌹 on") || !strings.Contains(plain, "🧠 918") {
		t.Errorf("minimal lost the priority segments: %s", line)
	}
	for _, banned := range []string{"ctx", "$0.312", "1d0h", "⧗ 2 jobs", "gentle ai"} {
		if strings.Contains(plain, banned) {
			t.Errorf("minimal shows %q: %s", banned, line)
		}
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
		{"usageCost", func(c *Config) { c.Visibility.UsageCost = false }, []string{"ctx", "$0.312", "1d0h"}},
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

// TestRenderPetalState pins the bare petal (L19): idle is the static flower
// alone, active is the animated frame with no label, no brand, and no
// scanner; modelDetails only hides the mode·model text.
func TestRenderPetalState(t *testing.T) {
	idle := mustPayload(t, `{"cwd": "/home/user/project", "terminal_width": 200, "agent_state": "idle"}`)
	line := Render(idle, DefaultConfig(), defaultExtras())
	plain := stripANSI(line)
	if !strings.HasPrefix(plain, "✿") {
		t.Errorf("idle lost the static petal: %s", line)
	}
	if strings.Contains(plain, "gentle ai") || strings.Contains(plain, "…") {
		t.Errorf("idle shows brand or label: %s", line)
	}

	active := mustPayload(t, `{"cwd": "/home/user/project", "terminal_width": 200, "agent_state": "thinking", "model": {"display_name": "M"}}`)
	line = Render(active, DefaultConfig(), defaultExtras())
	plain = stripANSI(line)
	if strings.Contains(plain, "…") || strings.Contains(plain, "thinking") || strings.Contains(plain, "gentle ai") {
		t.Errorf("active petal carries a label or brand: %s", line)
	}
	petalSeen := false
	for _, frame := range petalFrames {
		if strings.Contains(plain, frame) {
			petalSeen = true
		}
	}
	if !petalSeen {
		t.Errorf("active state missing animated petal: %s", line)
	}

	noModel := DefaultConfig()
	noModel.Visibility.ModelDetails = false
	noModel.Visibility.UsageCost = false
	noModel.Visibility.RDD = false
	line = Render(active, noModel, defaultExtras())
	plain = stripANSI(line)
	if strings.Contains(plain, "M") {
		t.Errorf("modelDetails:false still shows the model: %s", line)
	}
	petalSeen = false
	for _, frame := range petalFrames {
		if strings.Contains(plain, frame) {
			petalSeen = true
		}
	}
	if !petalSeen {
		t.Errorf("modelDetails:false lost the petal: %s", line)
	}
}

// TestRenderFallback pins the brand-only line for nil payloads.
func TestRenderFallback(t *testing.T) {
	if got := Render(nil, DefaultConfig(), defaultExtras()); got != FallbackLine {
		t.Errorf("Render(nil) = %q, want %q", got, FallbackLine)
	}
	if FallbackLine != "✿" {
		t.Errorf("FallbackLine = %q", FallbackLine)
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
	line := Render(payload, config, Extras{Changes: "±3 +10/−2"})
	if !strings.Contains(line, "±3 +10/−2") {
		t.Errorf("changes segment missing: %s", line)
	}

	config.Visibility.Changes = false
	line = Render(payload, config, Extras{Changes: "±3 +10/−2"})
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
