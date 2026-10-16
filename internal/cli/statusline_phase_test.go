package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestInferStatusLinePhase pins the S12 tool-call → phase mapping.
func TestInferStatusLinePhase(t *testing.T) {
	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{"read_file", nil, "exploring"},
		{"list_directory", nil, "exploring"},
		{"search_codebase", nil, "exploring"},
		{"grep_matches", nil, "exploring"},
		{"write_file", nil, "implementing"},
		{"edit_file", nil, "implementing"},
		{"create_directory", nil, "implementing"},
		{"run_command", map[string]any{"command": "go test ./internal/statusline/"}, "checking"},
		{"run_command", map[string]any{"command": "pytest -x"}, "checking"},
		{"run_command", map[string]any{"command": "make build"}, "implementing"},
		{"invoke_subagent", nil, "delegating"},
		{"manage_subagents", nil, "delegating"},
		{"something_else", nil, "working"},
	}
	for _, testCase := range cases {
		call := &AntigravityCLIToolCall{Name: testCase.name, Args: testCase.args}
		if got := inferStatusLinePhase(call); got != testCase.want {
			t.Errorf("inferStatusLinePhase(%q) = %q, want %q", testCase.name, got, testCase.want)
		}
	}
}

// TestRecordAndReadStatusLinePhase pins the record/read roundtrip, the fresh
// TTL, and the malformed/absent fallbacks.
func TestRecordAndReadStatusLinePhase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	recordAntigravityStatusLinePhase(AntigravityCLIHookPayload{
		ConversationID: "conv-1",
		ToolCall:       &AntigravityCLIToolCall{Name: "read_file"},
	})
	if got := readStatusLinePhase(home, "conv-1"); got != "exploring" {
		t.Errorf("readStatusLinePhase = %q, want exploring", got)
	}

	// Another conversation is isolated.
	if got := readStatusLinePhase(home, "conv-2"); got != "" {
		t.Errorf("unrelated conversation returned phase %q", got)
	}

	// A stale record reads as empty.
	path := statusLinePhasePath(home, "conv-stale")
	stale, _ := json.Marshal(statusLinePhaseRecord{Phase: "checking", TS: time.Now().Add(-time.Hour).Unix()})
	if err := os.WriteFile(path, stale, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readStatusLinePhase(home, "conv-stale"); got != "" {
		t.Errorf("stale record returned phase %q", got)
	}

	// A malformed record reads as empty.
	path = statusLinePhasePath(home, "conv-broken")
	if err := os.WriteFile(path, []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readStatusLinePhase(home, "conv-broken"); got != "" {
		t.Errorf("malformed record returned phase %q", got)
	}
}

// TestRecordStatusLinePhasePrunesStaleSiblings pins the one-hour prune.
func TestRecordStatusLinePhasePrunesStaleSiblings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Two records: one ancient, one fresh. The ancient file's mtime is set
	// back so the prune removes it.
	ancient := statusLinePhasePath(home, "conv-ancient")
	fresh := statusLinePhasePath(home, "conv-fresh")
	dir := filepath.Dir(ancient)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	record, _ := json.Marshal(statusLinePhaseRecord{Phase: "exploring", TS: time.Now().Unix()})
	if err := os.WriteFile(ancient, record, 0o644); err != nil {
		t.Fatal(err)
	}
	staleTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(ancient, staleTime, staleTime); err != nil {
		t.Fatal(err)
	}

	recordAntigravityStatusLinePhase(AntigravityCLIHookPayload{
		ConversationID: "conv-fresh",
		ToolCall:       &AntigravityCLIToolCall{Name: "write_file"},
	})

	if _, err := os.Stat(ancient); !os.IsNotExist(err) {
		t.Error("ancient phase record survived the prune")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("fresh record missing: %v", err)
	}
}

// TestRunStatusLineUsesPhaseRecord pins the end-to-end wiring: a fresh phase
// record drives the petal label while the agent is active.
func TestRunStatusLineUsesPhaseRecord(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	recordAntigravityStatusLinePhase(AntigravityCLIHookPayload{
		ConversationID: "conv-demo",
		ToolCall:       &AntigravityCLIToolCall{Name: "read_file"},
	})
	payload := `{
		"conversation_id": "conv-demo",
		"agent_state": "working",
		"cwd": "` + home + `/project",
		"model": {"display_name": "Gemini 3.5 Pro"},
		"context_window": {"used_percentage": 10},
		"terminal_width": 200
	}`
	line := runStatusLineTest(t, nil, payload, home)
	if !strings.Contains(line, "exploring…") {
		t.Errorf("phase label missing: %s", line)
	}

	// Without a record the agent_state fallback applies.
	payload = `{
		"conversation_id": "conv-other",
		"agent_state": "working",
		"cwd": "` + home + `/project",
		"model": {"display_name": "Gemini 3.5 Pro"},
		"context_window": {"used_percentage": 10},
		"terminal_width": 200
	}`
	line = runStatusLineTest(t, nil, payload, home)
	if !strings.Contains(line, "working…") {
		t.Errorf("fallback label missing: %s", line)
	}

	// While idle the phase record does not surface any label.
	recordAntigravityStatusLinePhase(AntigravityCLIHookPayload{
		ConversationID: "conv-idle",
		ToolCall:       &AntigravityCLIToolCall{Name: "read_file"},
	})
	payload = `{
		"conversation_id": "conv-idle",
		"agent_state": "idle",
		"cwd": "` + home + `/project",
		"model": {"display_name": "Gemini 3.5 Pro"},
		"context_window": {"used_percentage": 10},
		"terminal_width": 200
	}`
	line = runStatusLineTest(t, nil, payload, home)
	if strings.Contains(line, "…") {
		t.Errorf("idle line shows an activity label: %s", line)
	}
	if !strings.Contains(line, "✿ gentle ai") {
		t.Errorf("idle line lost the static brand: %s", line)
	}
}

// TestAntigravityHookPreToolUseWritesStatusLinePhase pins the hook wiring:
// a PreToolUse event leaves the phase record behind without changing the
// decision.
func TestAntigravityHookPreToolUseWritesStatusLinePhase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	payload := `{
		"conversationId": "conv-hook",
		"workspacePaths": ["` + home + `/project"],
		"toolCall": {"name": "edit_file", "args": {"path": "a.go"}}
	}`
	var stdout strings.Builder
	if err := RunAntigravityCLIHook("PreToolUse", "antigravity-cli", strings.NewReader(payload), &stdout, home+"/project"); err != nil {
		t.Fatalf("RunAntigravityCLIHook error: %v", err)
	}
	if got := readStatusLinePhase(home, "conv-hook"); got != "implementing" {
		t.Errorf("phase record after hook = %q, want implementing", got)
	}
	if !strings.Contains(stdout.String(), "\"decision\"") {
		t.Errorf("hook decision missing from output: %s", stdout.String())
	}
}
