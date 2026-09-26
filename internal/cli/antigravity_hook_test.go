package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAntigravityHookPreInvocationBaselineAndSilent(t *testing.T) {
	home := reviewModeHome(t)
	repo := initReviewCLIRepo(t)
	stageReviewStopHookCandidate(t, repo)
	enableReviewStopHookRDD(t, repo)

	convID := "conv-pre-0"
	payload := fmtAntigravityHookPayload(convID, []string{repo}, 0, "")

	var stdout bytes.Buffer
	err := RunAntigravityCLIHook("PreInvocation", strings.NewReader(payload), &stdout, repo)
	if err != nil {
		t.Fatalf("RunAntigravityCLIHook error: %v", err)
	}

	if stdout.String() != "{}\n" && stdout.String() != "{}" {
		t.Fatalf("expected stdout to be {}, got %q", stdout.String())
	}

	// Verify baseline file created under ~/.gentle-ai/review-stop-hook/v1/<convID>.json
	recordPath := filepath.Join(home, ".gentle-ai", "review-stop-hook", "v1", convID+".json")
	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error: %v", recordPath, err)
	}
	var record struct {
		BaselineTargetIdentity string `json:"baseline_target_identity"`
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if record.BaselineTargetIdentity == "" {
		t.Errorf("expected BaselineTargetIdentity to be non-empty")
	}

	// Subsequent invocation (invocationNum > 0)
	stdout.Reset()
	payloadSubsequent := fmtAntigravityHookPayload(convID, []string{repo}, 1, "")
	err = RunAntigravityCLIHook("PreInvocation", strings.NewReader(payloadSubsequent), &stdout, repo)
	if err != nil {
		t.Fatalf("subsequent PreInvocation error: %v", err)
	}
	if stdout.String() != "{}\n" && stdout.String() != "{}" {
		t.Fatalf("expected stdout {}, got %q", stdout.String())
	}
}

func TestAntigravityHookStopOutcomes(t *testing.T) {
	tests := []struct {
		name              string
		terminationReason string
		fullyIdle         bool
		wantDecision      string
		wantReminder      bool
	}{
		{name: "fully idle model stop continues", terminationReason: "model_stop", fullyIdle: true, wantDecision: "continue", wantReminder: true},
		{name: "non-idle model stop stays neutral", terminationReason: "model_stop", fullyIdle: false},
		{name: "non-model stop stays neutral", terminationReason: "max_steps_exceeded", fullyIdle: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := reviewModeHome(t)
			repo := initReviewCLIRepo(t)
			enableReviewStopHookRDD(t, repo)
			convID := "conv-stop-" + strings.ReplaceAll(tt.name, " ", "-")

			var stdout bytes.Buffer
			prePayload := fmtAntigravityHookPayload(convID, []string{repo}, 0, "")
			if err := RunAntigravityCLIHook("PreInvocation", strings.NewReader(prePayload), &stdout, repo); err != nil {
				t.Fatalf("PreInvocation error: %v", err)
			}
			recordPath := filepath.Join(home, ".gentle-ai", "review-stop-hook", "v1", convID+".json")
			before, err := os.ReadFile(recordPath)
			if err != nil {
				t.Fatalf("read baseline record: %v", err)
			}

			stageReviewStopHookCandidate(t, repo)
			stopPayload := fmtAntigravityStopPayload(convID, []string{repo}, 5, tt.terminationReason, tt.fullyIdle)
			stdout.Reset()
			if err := RunAntigravityCLIHook("Stop", strings.NewReader(stopPayload), &stdout, repo); err != nil {
				t.Fatalf("Stop error: %v", err)
			}

			raw := strings.TrimSpace(stdout.String())
			if tt.wantDecision == "" {
				if raw != "{}" {
					t.Fatalf("Stop output = %q, want neutral {}", raw)
				}
			} else {
				var out AntigravityCLIHookOutput
				if err := json.Unmarshal([]byte(raw), &out); err != nil {
					t.Fatalf("decode Stop output %q: %v", raw, err)
				}
				if out.Decision != tt.wantDecision || !strings.Contains(out.Reason, "Receipt-Driven Development: unreviewed candidate changes detected") {
					t.Fatalf("Stop output = %+v, want continuation reminder", out)
				}
			}

			after, err := os.ReadFile(recordPath)
			if err != nil {
				t.Fatalf("read record after Stop: %v", err)
			}
			if tt.wantReminder {
				var record struct {
					TargetIdentity string `json:"target_identity"`
				}
				if err := json.Unmarshal(after, &record); err != nil || record.TargetIdentity == "" {
					t.Fatalf("reminder record = %q, error = %v", after, err)
				}
				stdout.Reset()
				if err := RunAntigravityCLIHook("Stop", strings.NewReader(stopPayload), &stdout, repo); err != nil {
					t.Fatalf("second Stop error: %v", err)
				}
				if raw := strings.TrimSpace(stdout.String()); raw != "{}" {
					t.Fatalf("second Stop output = %q, want neutral {}", raw)
				}
			} else if !bytes.Equal(after, before) {
				t.Fatalf("neutral Stop changed reminder state:\nbefore=%s\nafter=%s", before, after)
			}
		})
	}
}

func fmtAntigravityHookPayload(convID string, wsPaths []string, invNum int, termReason string) string {
	m := map[string]any{
		"conversationId": convID,
		"workspacePaths": wsPaths,
		"invocationNum":  invNum,
	}
	if termReason != "" {
		m["terminationReason"] = termReason
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func fmtAntigravityStopPayload(convID string, wsPaths []string, invNum int, termReason string, fullyIdle bool) string {
	m := map[string]any{
		"conversationId":    convID,
		"workspacePaths":    wsPaths,
		"invocationNum":     invNum,
		"terminationReason": termReason,
		"fullyIdle":         fullyIdle,
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func fmtAntigravityToolCallPayload(convID string, toolName string, args map[string]any, stepIdx int) string {
	m := map[string]any{
		"conversationId": convID,
		"stepIdx":        stepIdx,
		"toolCall": map[string]any{
			"name": toolName,
			"args": args,
		},
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func TestAntigravityHookPreToolUseCircuitBreaker(t *testing.T) {
	home := reviewModeHome(t)
	convID := "conv-cb-1"

	callHook := func(toolName string, args map[string]any, stepIdx int) (AntigravityCLIHookOutput, string) {
		payload := fmtAntigravityToolCallPayload(convID, toolName, args, stepIdx)
		var stdout bytes.Buffer
		err := RunAntigravityCLIHook("PreToolUse", strings.NewReader(payload), &stdout, "")
		if err != nil {
			t.Fatalf("RunAntigravityCLIHook returned error: %v", err)
		}
		raw := strings.TrimSpace(stdout.String())
		var out AntigravityCLIHookOutput
		if raw != "{}" {
			if err := json.Unmarshal([]byte(raw), &out); err != nil {
				t.Fatalf("Unmarshal error for raw %q: %v", raw, err)
			}
		}
		return out, raw
	}

	readRecord := func() circuitBreakerRecord {
		recPath := circuitBreakerRecordPath(home, convID)
		data, err := os.ReadFile(recPath)
		if err != nil {
			t.Fatalf("failed to read circuit breaker record from %s: %v", recPath, err)
		}
		var rec circuitBreakerRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			t.Fatalf("failed to unmarshal record: %v", err)
		}
		return rec
	}

	// 1st call: allowed (decision: allow)
	out, raw := callHook("view_file", map[string]any{"path": "internal/foo.go"}, 1)
	if out.Decision != "allow" {
		t.Fatalf("call 1 expected decision: allow, got %+v (raw: %q)", out, raw)
	}
	rec := readRecord()
	if rec.ConsecutiveCount != 1 {
		t.Errorf("call 1 expected ConsecutiveCount=1, got %d", rec.ConsecutiveCount)
	}
	if rec.LastToolName != "view_file" {
		t.Errorf("call 1 expected LastToolName=view_file, got %q", rec.LastToolName)
	}

	// 2nd consecutive identical call: allowed (decision: allow)
	out, raw = callHook("view_file", map[string]any{"path": "internal/foo.go"}, 2)
	if out.Decision != "allow" {
		t.Fatalf("call 2 expected decision: allow, got %+v (raw: %q)", out, raw)
	}
	rec = readRecord()
	if rec.ConsecutiveCount != 2 {
		t.Errorf("call 2 expected ConsecutiveCount=2, got %d", rec.ConsecutiveCount)
	}

	// 3rd consecutive identical call: blocked (decision: deny)
	out, raw = callHook("view_file", map[string]any{"path": "internal/foo.go"}, 3)
	if out.Decision != "deny" {
		t.Fatalf("call 3 expected decision: deny, got %+v (raw: %q)", out, raw)
	}
	if !strings.Contains(out.Reason, "Loop Circuit Breaker") {
		t.Errorf("call 3 reason should mention 'Loop Circuit Breaker', got %q", out.Reason)
	}
	if !strings.Contains(out.Reason, "3 times consecutively") {
		t.Errorf("call 3 reason should mention count 3, got %q", out.Reason)
	}
	rec = readRecord()
	if rec.ConsecutiveCount != 3 {
		t.Errorf("call 3 expected ConsecutiveCount=3, got %d", rec.ConsecutiveCount)
	}

	// Calling a different tool: resets the count and is allowed
	out, raw = callHook("edit_file", map[string]any{"path": "internal/foo.go"}, 4)
	if out.Decision != "allow" {
		t.Fatalf("different tool expected decision: allow, got %+v (raw: %q)", out, raw)
	}
	rec = readRecord()
	if rec.ConsecutiveCount != 1 {
		t.Errorf("different tool expected count reset to 1, got %d", rec.ConsecutiveCount)
	}
	if rec.LastToolName != "edit_file" {
		t.Errorf("expected LastToolName=edit_file, got %q", rec.LastToolName)
	}

	// 2nd call to edit_file: allowed
	out, raw = callHook("edit_file", map[string]any{"path": "internal/foo.go"}, 5)
	if out.Decision != "allow" {
		t.Fatalf("2nd edit_file expected decision: allow, got %+v (raw: %q)", out, raw)
	}
	rec = readRecord()
	if rec.ConsecutiveCount != 2 {
		t.Errorf("2nd edit_file expected count=2, got %d", rec.ConsecutiveCount)
	}

	// Calling the same tool with different args: resets the count and is allowed
	out, raw = callHook("edit_file", map[string]any{"path": "internal/bar.go"}, 6)
	if out.Decision != "allow" {
		t.Fatalf("same tool different args expected decision: allow, got %+v (raw: %q)", out, raw)
	}
	rec = readRecord()
	if rec.ConsecutiveCount != 1 {
		t.Errorf("same tool different args expected count reset to 1, got %d", rec.ConsecutiveCount)
	}

	// Calling edit_file with bar.go 2nd time: allowed
	out, raw = callHook("edit_file", map[string]any{"path": "internal/bar.go"}, 7)
	if out.Decision != "allow" {
		t.Fatalf("2nd call with bar.go expected decision: allow, got %+v (raw: %q)", out, raw)
	}

	// Calling edit_file with bar.go 3rd time: denied
	out, _ = callHook("edit_file", map[string]any{"path": "internal/bar.go"}, 8)
	if out.Decision != "deny" {
		t.Fatalf("3rd call with bar.go expected decision: deny, got %+v", out)
	}
	if !strings.Contains(out.Reason, "Loop Circuit Breaker") {
		t.Errorf("reason should mention 'Loop Circuit Breaker', got %q", out.Reason)
	}
}

func TestAntigravityHookPreToolUseKeyOrdering(t *testing.T) {
	_ = reviewModeHome(t)
	convID := "conv-cb-ordering"

	// Call 1 with args {"a": 1, "b": 2}
	payload1 := `{"conversationId":"` + convID + `","stepIdx":1,"toolCall":{"name":"test_tool","args":{"a":1,"b":2}}}`
	var stdout bytes.Buffer
	if err := RunAntigravityCLIHook("PreToolUse", strings.NewReader(payload1), &stdout, ""); err != nil {
		t.Fatalf("call 1 error: %v", err)
	}
	var out1 AntigravityCLIHookOutput
	if err := json.Unmarshal(stdout.Bytes(), &out1); err != nil || out1.Decision != "allow" {
		t.Fatalf("call 1 expected decision: allow, got %+v (raw: %q)", out1, stdout.String())
	}

	// Call 2 with args {"b": 2, "a": 1} - should be recognized as identical arguments
	stdout.Reset()
	payload2 := `{"conversationId":"` + convID + `","stepIdx":2,"toolCall":{"name":"test_tool","args":{"b":2,"a":1}}}`
	if err := RunAntigravityCLIHook("PreToolUse", strings.NewReader(payload2), &stdout, ""); err != nil {
		t.Fatalf("call 2 error: %v", err)
	}
	var out2 AntigravityCLIHookOutput
	if err := json.Unmarshal(stdout.Bytes(), &out2); err != nil || out2.Decision != "allow" {
		t.Fatalf("call 2 expected decision: allow, got %+v (raw: %q)", out2, stdout.String())
	}

	// Call 3 with args {"a": 1, "b": 2} - 3rd consecutive identical call -> deny
	stdout.Reset()
	if err := RunAntigravityCLIHook("PreToolUse", strings.NewReader(payload1), &stdout, ""); err != nil {
		t.Fatalf("call 3 error: %v", err)
	}
	var out AntigravityCLIHookOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if out.Decision != "deny" {
		t.Fatalf("call 3 expected decision: deny, got %+v", out)
	}
}

func TestAntigravityHookPreToolUseEdgeCases(t *testing.T) {
	_ = reviewModeHome(t)

	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{name: "empty conversation ID", payload: `{"conversationId":"","toolCall":{"name":"view_file","args":{}}}`, want: "allow"},
		{name: "nil tool call", payload: `{"conversationId":"conv-nil","stepIdx":1}`, want: "allow"},
		{name: "empty tool name", payload: `{"conversationId":"conv-noname","toolCall":{"name":"","args":{}}}`, want: "allow"},
		{name: "malformed JSON", payload: "not json", want: "deny"},
		{name: "empty payload", payload: "", want: "deny"},
		{name: "null payload", payload: "null", want: "deny"},
		{name: "array payload", payload: "[]", want: "deny"},
		{name: "malformed invoke payload", payload: `{"toolCall":{"name":"invoke_subagent","args":`, want: "deny"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			if err := RunAntigravityCLIHook("PreToolUse", strings.NewReader(test.payload), &stdout, ""); err != nil {
				t.Fatalf("RunAntigravityCLIHook() error = %v", err)
			}
			var out AntigravityCLIHookOutput
			if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
				t.Fatalf("decode output %q: %v", stdout.String(), err)
			}
			if out.Decision != test.want {
				t.Errorf("decision = %q, want %q (reason %q)", out.Decision, test.want, out.Reason)
			}
		})
	}
}

func TestAntigravityHookPreToolUseFallbackPath(t *testing.T) {
	// Inaccessible home directory should fallback to os.TempDir()
	convID := "conv-fallback-test"
	fallbackPath := circuitBreakerRecordPath("", convID)
	expectedPrefix := filepath.Join(os.TempDir(), "gentle-ai-circuit-breaker")
	if !strings.HasPrefix(fallbackPath, expectedPrefix) {
		t.Errorf("expected fallback path to start with %s, got %s", expectedPrefix, fallbackPath)
	}

	// Non-existent home should also fallback
	nonExistentHomePath := circuitBreakerRecordPath("/path/to/nonexistent/directory/12345", convID)
	if !strings.HasPrefix(nonExistentHomePath, expectedPrefix) {
		t.Errorf("expected non-existent home path to fallback to tempdir, got %s", nonExistentHomePath)
	}
}

func TestAntigravityHookRunHookCLIPreToolUse(t *testing.T) {
	_ = reviewModeHome(t)
	convID := "conv-cli-tool"
	payload := fmtAntigravityToolCallPayload(convID, "read_code", map[string]any{"symbol": "Foo"}, 1)

	var stdout bytes.Buffer
	cmd := []string{"run", "--agent", "antigravity-cli", "--event", "PreToolUse"}
	err := runHookCommand(cmd[1:], strings.NewReader(payload), &stdout)
	if err != nil {
		t.Fatalf("runHookCommand error: %v", err)
	}
	var out AntigravityCLIHookOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil || out.Decision != "allow" {
		t.Errorf("expected decision allow, got %+v (raw: %q)", out, stdout.String())
	}
}

func TestAntigravityHookPostEvents(t *testing.T) {
	tests := []struct {
		name      string
		event     string
		payload   string
		malformed bool
	}{
		{
			name:    "PostToolUse accepts official fields and unknown fields",
			event:   "PostToolUse",
			payload: `{"conversationId":"conv-post-tool","workspacePaths":["/tmp/workspace"],"invocationNum":2,"initialNumSteps":3,"executionNum":4,"error":"tool completed after a warning","transcriptPath":"/tmp/transcript.json","artifactDirectoryPath":"/tmp/artifacts","modelName":"test-model","stepIdx":5,"toolCall":{"name":"read_code","args":{"symbol":"Foo"}},"futureField":{"nested":true}}`,
		},
		{
			name:    "PostInvocation accepts official fields and unknown fields",
			event:   "PostInvocation",
			payload: `{"conversationId":"conv-post-invocation","workspacePaths":["/tmp/workspace"],"invocationNum":6,"initialNumSteps":3,"executionNum":7,"error":"invocation completed with a warning","transcriptPath":"/tmp/transcript.json","artifactDirectoryPath":"/tmp/artifacts","modelName":"test-model","futureField":["ignored"]}`,
		},
		{
			name:      "PostToolUse accepts malformed payload safely",
			event:     "PostToolUse",
			payload:   `{"toolCall":`,
			malformed: true,
		},
		{
			name:      "PostInvocation accepts malformed payload safely",
			event:     "PostInvocation",
			payload:   `not json`,
			malformed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			if err := RunAntigravityCLIHook(tt.event, strings.NewReader(tt.payload), &stdout, t.TempDir()); err != nil {
				t.Fatalf("RunAntigravityCLIHook(%q) error: %v", tt.event, err)
			}
			if got := stdout.String(); got != "{}\n" {
				t.Fatalf("output = %q, want exact neutral object %q", got, "{}\\n")
			}

			if tt.malformed {
				return
			}
			var payload AntigravityCLIHookPayload
			if err := json.Unmarshal([]byte(tt.payload), &payload); err != nil {
				t.Fatalf("valid payload was rejected: %v", err)
			}
			if payload.ConversationID == "" || len(payload.WorkspacePaths) == 0 ||
				payload.TranscriptPath == "" || payload.ArtifactDirectoryPath == "" || payload.ModelName == "" {
				t.Fatalf("official payload fields were not decoded: %+v", payload)
			}
			if tt.event == "PostToolUse" && (payload.StepIdx != 5 || payload.ToolCall == nil || payload.ToolCall.Name != "read_code") {
				t.Fatalf("PostToolUse fields were not decoded: %+v", payload)
			}
		})
	}
}

func TestAntigravityHookRunHookCLIPostEvents(t *testing.T) {
	for _, event := range []string{"PostToolUse", "PostInvocation"} {
		t.Run(event, func(t *testing.T) {
			var stdout bytes.Buffer
			args := []string{"--agent", "antigravity-cli", "--event", event}
			if err := runHookCommand(args, strings.NewReader(`{"unknownField":true}`), &stdout); err != nil {
				t.Fatalf("runHookCommand(%s) error: %v", event, err)
			}
			if got := stdout.String(); got != "{}\n" {
				t.Fatalf("output = %q, want exact neutral object %q", got, "{}\\n")
			}
		})
	}
}

func TestAntigravityHookHelpNamesOfficialEvents(t *testing.T) {
	for _, event := range []string{"PreInvocation", "PreToolUse", "PostToolUse", "PostInvocation", "Stop"} {
		if !strings.Contains(antigravityCLIHookEventHelp, event) {
			t.Errorf("hook help omits %s: %q", event, antigravityCLIHookEventHelp)
		}
	}
}

func TestAntigravityHookPreToolUseMalformedInvokePayloadFailsClosed(t *testing.T) {
	payload := `{"conversationId":"conv-agy-malformed","toolCall":{"name":"invoke_subagent","args":"not-an-object"}}`
	var stdout bytes.Buffer
	if err := RunAntigravityCLIHook("PreToolUse", strings.NewReader(payload), &stdout, ""); err != nil {
		t.Fatalf("RunAntigravityCLIHook() error = %v", err)
	}
	var output AntigravityCLIHookOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode hook output %q: %v", stdout.String(), err)
	}
	if output.Decision != "deny" || !strings.Contains(output.Reason, "arguments are malformed") {
		t.Fatalf("malformed invoke_subagent output = %+v, want fail-closed denial", output)
	}
}

func TestAntigravityCLIHookOutputPostInvocationShape(t *testing.T) {
	empty, err := json.Marshal(AntigravityCLIHookOutput{})
	if err != nil {
		t.Fatalf("marshal empty output: %v", err)
	}
	if string(empty) != "{}" {
		t.Fatalf("empty output = %q, want %q", empty, "{}")
	}

	output := AntigravityCLIHookOutput{
		InjectSteps:         []any{map[string]any{"ephemeralMessage": "context"}},
		TerminationBehavior: "force_continue",
	}
	raw, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal PostInvocation output: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode PostInvocation output: %v", err)
	}
	if _, ok := decoded["injectSteps"]; !ok {
		t.Fatalf("PostInvocation output omitted injectSteps: %s", raw)
	}
	if decoded["terminationBehavior"] != "force_continue" {
		t.Fatalf("terminationBehavior = %v, want force_continue: %s", decoded["terminationBehavior"], raw)
	}
}

func TestAntigravityHookPreToolUseInvokeSubagentValidation(t *testing.T) {
	_ = reviewModeHome(t)

	tests := []struct {
		name       string
		args       map[string]any
		want       string
		wantReason string
	}{
		{
			name: "valid subagent allowed",
			args: map[string]any{"Subagents": []any{map[string]any{
				"TypeName": "worker",
				"Role":     "Implementation",
				"Prompt":   "Write code.",
			}}},
			want: "allow",
		},
		{
			name: "valid subagent with valid workspace",
			args: map[string]any{"Subagents": []any{map[string]any{
				"TypeName":  "worker",
				"Role":      "Implementation",
				"Prompt":    "Write code.",
				"Workspace": "branch",
			}}},
			want: "allow",
		},
		{
			name: "subagent with invalid workspace",
			args: map[string]any{"Subagents": []any{map[string]any{
				"TypeName":  "worker",
				"Role":      "Implementation",
				"Prompt":    "Write code.",
				"Workspace": "invalid-mode",
			}}},
			want:       "deny",
			wantReason: "invalid Workspace",
		},
		{
			name: "missing TypeName",
			args: map[string]any{"Subagents": []any{map[string]any{
				"Role":   "Implementation",
				"Prompt": "Write code.",
			}}},
			want:       "deny",
			wantReason: "no valid TypeName",
		},
		{
			name: "missing Role",
			args: map[string]any{"Subagents": []any{map[string]any{
				"TypeName": "worker",
				"Prompt":   "Write code.",
			}}},
			want:       "deny",
			wantReason: "no valid Role",
		},
		{
			name: "missing Prompt",
			args: map[string]any{"Subagents": []any{map[string]any{
				"TypeName": "worker",
				"Role":     "Implementation",
			}}},
			want:       "deny",
			wantReason: "no valid Prompt",
		},
		{
			name:       "Subagents not an array",
			args:       map[string]any{"Subagents": "not-an-array"},
			want:       "deny",
			wantReason: "arguments are malformed",
		},
		{
			name:       "empty Subagents array",
			args:       map[string]any{"Subagents": []any{}},
			want:       "deny",
			wantReason: "arguments are malformed",
		},
		{
			name: "Subagent item not an object",
			args: map[string]any{"Subagents": []any{"not-an-object"}},
			want:       "deny",
			wantReason: "is not an object",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := fmtAntigravityToolCallPayload("conv-agy-validate-"+strings.ReplaceAll(test.name, " ", "-"), "invoke_subagent", test.args, 1)
			var stdout bytes.Buffer
			if err := RunAntigravityCLIHook("PreToolUse", strings.NewReader(payload), &stdout, ""); err != nil {
				t.Fatalf("RunAntigravityCLIHook() error = %v", err)
			}
			var output AntigravityCLIHookOutput
			if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
				t.Fatalf("decode hook output %q: %v", stdout.String(), err)
			}
			if output.Decision != test.want {
				t.Fatalf("decision = %q, want %q (reason %q)", output.Decision, test.want, output.Reason)
			}
			if test.wantReason != "" && !strings.Contains(output.Reason, test.wantReason) {
				t.Fatalf("reason = %q, want substring %q", output.Reason, test.wantReason)
			}
		})
	}
}
