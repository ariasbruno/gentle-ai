package cli

import (
	"bytes"
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

// TestRunStatusLinePhaseRecordedNotRendered pins the L19 decision: the
// PreToolUse hook still records phase files and readStatusLinePhase reads
// them back (S12 infra), but the rendered line carries no phase label.
func TestRunStatusLinePhaseRecordedNotRendered(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	payload := `{
		"conversationId": "conv-789",
		"agent_state": "working",
		"cwd": "` + home + `/project",
		"model": {"display_name": "Gemini 3.5 Pro"},
		"terminal_width": 200,
		"toolCall": {"name": "read_file", "args": {"path": "x"}}
	}`
	var hookPayload AntigravityCLIHookPayload
	if err := json.Unmarshal([]byte(payload), &hookPayload); err != nil {
		t.Fatal(err)
	}
	var hookOut bytes.Buffer
	if err := handleAntigravityPreToolUse(hookPayload, home, &hookOut); err != nil {
		t.Fatal(err)
	}
	if phase := readStatusLinePhase(home, "conv-789"); phase != "exploring" {
		t.Fatalf("phase record readback = %q, want exploring", phase)
	}
	line := runStatusLineTest(t, nil, payload, home)
	if plain := stripStatusLineANSI(line); strings.Contains(plain, "…") {
		t.Errorf("statusline renders a phase label: %s", line)
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
