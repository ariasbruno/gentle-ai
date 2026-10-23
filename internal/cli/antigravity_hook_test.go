package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	err := RunAntigravityCLIHook("PreInvocation", "antigravity-cli", strings.NewReader(payload), &stdout, repo)
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
	err = RunAntigravityCLIHook("PreInvocation", "antigravity-cli", strings.NewReader(payloadSubsequent), &stdout, repo)
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
		{name: "fully idle NO_TOOL_CALL continues", terminationReason: "NO_TOOL_CALL", fullyIdle: true, wantDecision: "continue", wantReminder: true},
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
			if err := RunAntigravityCLIHook("PreInvocation", "antigravity-cli", strings.NewReader(prePayload), &stdout, repo); err != nil {
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
			if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
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
				if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
					t.Fatalf("second Stop error: %v", err)
				}
				// The consumed-check: the same unreviewed candidate keeps blocking
				// until the review is consumed, the baseline moves, or consent is
				// explicitly refused. The second Stop also emits a reminder.
				var secondOut AntigravityCLIHookOutput
				raw := strings.TrimSpace(stdout.String())
				if err := json.Unmarshal([]byte(raw), &secondOut); err != nil {
					t.Fatalf("decode second Stop output %q: %v", raw, err)
				}
				if secondOut.Decision != "continue" {
					t.Fatalf("second Stop output = %q, want continue (consumed-check keeps blocking the unreviewed candidate)", raw)
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

// TestAntigravityPostToolUseRecordsMutations pins the functional PostToolUse:
// completed write_to_file and write_file/edit_file calls record a mutation in the per-conversation
// ledger, mirroring Gentle Shell's tool_result handler.
func TestAntigravityPostToolUseRecordsMutations(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, tc := range []struct {
		name string
		args map[string]any
		wantPath string
	}{
		{name: "write_to_file", args: map[string]any{"TargetFile": "/project/src/main.go"}, wantPath: "/project/src/main.go"},
		{name: "write_file", args: map[string]any{"Path": "/project/src/new.go"}, wantPath: "/project/src/new.go"},
		{name: "edit_file", args: map[string]any{"Path": "/project/src/edit.go"}, wantPath: "/project/src/edit.go"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			convID := "conv-mut-" + tc.name
			payload := fmtAntigravityToolCallPayload(convID, tc.name, tc.args, 5)
			var stdout bytes.Buffer
			if err := RunAntigravityCLIHook("PostToolUse", "antigravity-cli", strings.NewReader(payload), &stdout, home); err != nil {
				t.Fatalf("RunAntigravityCLIHook: %v", err)
			}
			if strings.TrimSpace(stdout.String()) != "{}" {
				t.Fatalf("PostToolUse output = %q, want contract-mandated {}", stdout.String())
			}
			ledgerPath := antigravityCLIMutationLedgerPath(home, convID)
			data, err := os.ReadFile(ledgerPath)
			if err != nil {
				t.Fatalf("mutation ledger not created: %v", err)
			}
			if !strings.Contains(string(data), fmt.Sprintf(`"path":%q`, tc.wantPath)) {
				t.Fatalf("ledger entry missing path: %s", data)
			}
			if !strings.Contains(string(data), fmt.Sprintf(`"tool_name":%q`, tc.name)) {
				t.Fatalf("ledger entry missing tool name: %s", data)
			}
		})
	}
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
		err := RunAntigravityCLIHook("PreToolUse", "antigravity-cli", strings.NewReader(payload), &stdout, "")
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
	if err := RunAntigravityCLIHook("PreToolUse", "antigravity-cli", strings.NewReader(payload1), &stdout, ""); err != nil {
		t.Fatalf("call 1 error: %v", err)
	}
	var out1 AntigravityCLIHookOutput
	if err := json.Unmarshal(stdout.Bytes(), &out1); err != nil || out1.Decision != "allow" {
		t.Fatalf("call 1 expected decision: allow, got %+v (raw: %q)", out1, stdout.String())
	}

	// Call 2 with args {"b": 2, "a": 1} - should be recognized as identical arguments
	stdout.Reset()
	payload2 := `{"conversationId":"` + convID + `","stepIdx":2,"toolCall":{"name":"test_tool","args":{"b":2,"a":1}}}`
	if err := RunAntigravityCLIHook("PreToolUse", "antigravity-cli", strings.NewReader(payload2), &stdout, ""); err != nil {
		t.Fatalf("call 2 error: %v", err)
	}
	var out2 AntigravityCLIHookOutput
	if err := json.Unmarshal(stdout.Bytes(), &out2); err != nil || out2.Decision != "allow" {
		t.Fatalf("call 2 expected decision: allow, got %+v (raw: %q)", out2, stdout.String())
	}

	// Call 3 with args {"a": 1, "b": 2} - 3rd consecutive identical call -> deny
	stdout.Reset()
	if err := RunAntigravityCLIHook("PreToolUse", "antigravity-cli", strings.NewReader(payload1), &stdout, ""); err != nil {
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
			if err := RunAntigravityCLIHook("PreToolUse", "antigravity-cli", strings.NewReader(test.payload), &stdout, ""); err != nil {
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
			if err := RunAntigravityCLIHook(tt.event, "antigravity-cli", strings.NewReader(tt.payload), &stdout, t.TempDir()); err != nil {
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
	if err := RunAntigravityCLIHook("PreToolUse", "antigravity-cli", strings.NewReader(payload), &stdout, ""); err != nil {
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
			name:       "Subagent item not an object",
			args:       map[string]any{"Subagents": []any{"not-an-object"}},
			want:       "deny",
			wantReason: "is not an object",
		},
		{
			name: "native flat subagent allowed",
			args: map[string]any{
				"TypeName": "DeepCoder",
				"Role":     "Deep Coder",
				"Prompt":   "Write code.",
			},
			want: "allow",
		},
		{
			name: "native flat subagent with valid workspace",
			args: map[string]any{
				"TypeName":  "DeepCoder",
				"Role":      "Deep Coder",
				"Prompt":    "Write code.",
				"Workspace": "inherit",
			},
			want: "allow",
		},
		{
			name: "native flat subagent with invalid workspace",
			args: map[string]any{
				"TypeName":  "DeepCoder",
				"Role":      "Deep Coder",
				"Prompt":    "Write code.",
				"Workspace": "invalid-workspace",
			},
			want:       "deny",
			wantReason: "invalid Workspace",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := fmtAntigravityToolCallPayload("conv-agy-validate-"+strings.ReplaceAll(test.name, " ", "-"), "invoke_subagent", test.args, 1)
			var stdout bytes.Buffer
			if err := RunAntigravityCLIHook("PreToolUse", "antigravity-cli", strings.NewReader(payload), &stdout, ""); err != nil {
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

// TestAntigravityPreToolUseDeniesSensitiveFilePaths pins the dynamic guard
// for the sensitive-path deny set the settings schema cannot express: the
// vendor file targets support no globs, so the Claude-parity patterns are
// evaluated on the concrete tool-call path at any depth, for the file and
// search tools only. HOME is isolated so circuit-breaker records from the
// allow cases never leak to the real home.
func TestAntigravityPreToolUseDeniesSensitiveFilePaths(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, test := range []struct {
		name     string
		tool     string
		args     map[string]any
		wantDeny bool
	}{
		{"view env", "view_file", map[string]any{"AbsolutePath": "/project/.env"}, true},
		{"view nested env variant", "view_file", map[string]any{"AbsolutePath": "/project/backend/.env.production"}, true},
		{"view relative env", "view_file", map[string]any{"AbsolutePath": ".env"}, true},
		{"view ssh key", "view_file", map[string]any{"AbsolutePath": "/home/u/.ssh/id_ed25519"}, true},
		{"view credentials", "view_file", map[string]any{"AbsolutePath": "/home/u/.credentials/token"}, true},
		{"view aws credentials", "view_file", map[string]any{"AbsolutePath": "/home/u/.aws/credentials"}, true},
		{"view pem", "view_file", map[string]any{"AbsolutePath": "/project/certs/server.pem"}, true},
		{"view key", "view_file", map[string]any{"AbsolutePath": "/project/keys/tls.key"}, true},
		{"view gh hosts", "view_file", map[string]any{"AbsolutePath": "/home/u/.config/gh/hosts.yml"}, true},
		{"view keychain", "view_file", map[string]any{"AbsolutePath": "/Users/u/Library/Keychains/login.keychain-db"}, true},
		{"view secrets dir file", "view_file", map[string]any{"AbsolutePath": "/project/secrets/token"}, true},
		{"write env variant", "write_to_file", map[string]any{"TargetFile": "/project/.env.local"}, true},
		{"native write_file env variant", "write_file", map[string]any{"Path": "/project/.env.local"}, true},
		{"native edit_file env", "edit_file", map[string]any{"Path": "/project/app/.env"}, true},
		{"native read_file ssh key", "read_file", map[string]any{"Path": "/home/u/.ssh/id_ed25519"}, true},
		{"native write_file ordinary", "write_file", map[string]any{"Path": "/project/src/main.go"}, false},
		{"replace env", "replace_file_content", map[string]any{"TargetFile": "/project/app/.env"}, true},
		{"multi replace pem", "multi_replace_file_content", map[string]any{"TargetFile": "/project/cert.pem"}, true},
		{"grep ssh dir", "grep_search", map[string]any{"SearchPath": "/home/u/.ssh"}, true},
		{"view ordinary file", "view_file", map[string]any{"AbsolutePath": "/project/src/main.go"}, false},
		{"view envish name", "view_file", map[string]any{"AbsolutePath": "/project/ENV.txt"}, false},
		{"view pemx", "view_file", map[string]any{"AbsolutePath": "/project/server.pemx"}, false},
		{"view secretsmith", "view_file", map[string]any{"AbsolutePath": "/project/secretsmith.go"}, false},
		{"ordinary command not denied", "run_command", map[string]any{"CommandLine": "ls -la"}, false},
		{"file tool without path", "view_file", map[string]any{}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			payload := fmtAntigravityToolCallPayload("conv-sensitive-paths", test.tool, test.args, 1)
			var stdout bytes.Buffer
			if err := RunAntigravityCLIHook("PreToolUse", "antigravity-cli", strings.NewReader(payload), &stdout, ""); err != nil {
				t.Fatalf("RunAntigravityCLIHook() error = %v", err)
			}
			var output AntigravityCLIHookOutput
			if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
				t.Fatalf("decode hook output %q: %v", stdout.String(), err)
			}
			if test.wantDeny {
				if output.Decision != "deny" {
					t.Fatalf("decision = %q, want deny (output %q)", output.Decision, stdout.String())
				}
				if !strings.Contains(output.Reason, "Sensitive file path") {
					t.Fatalf("reason = %q, want the sensitive-path guard explanation", output.Reason)
				}
			} else if output.Decision == "deny" {
				t.Fatalf("decision = deny for an ordinary call (output %q)", stdout.String())
			}
		})
	}
}

// TestAntigravityStopStaysSilentForAMediumCandidate pins the same policy the
// Claude stop hook and Gentle Shell apply: only a review_due candidate earns a
// reminder, so a medium candidate closes with a neutral {} even with RDD
// enabled and START offered.
func TestAntigravityStopStaysSilentForAMediumCandidate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the review fixture uses POSIX git")
	}
	home := reviewModeHome(t)
	repo := initReviewCLIRepo(t)
	// A repository that already ignores .atl keeps PreInvocation from creating
	// an untracked .gitignore, so the candidate is assessable. An unassessable
	// candidate stays review_due by design and still blocks.
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(".atl\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runReviewCLIGit(t, repo, "add", ".gitignore")
	runReviewCLIGit(t, repo, "commit", "-qm", "ignore atl")
	enableReviewStopHookRDD(t, repo)
	convID := "conv-stop-medium"

	var stdout bytes.Buffer
	prePayload := fmtAntigravityHookPayload(convID, []string{repo}, 0, "")
	if err := RunAntigravityCLIHook("PreInvocation", "antigravity-cli", strings.NewReader(prePayload), &stdout, repo); err != nil {
		t.Fatalf("PreInvocation error: %v", err)
	}
	// Stage after the baseline: the candidate must be newer than the session.
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stopPayload := fmtAntigravityStopPayload(convID, []string{repo}, 5, "model_stop", true)
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	if raw := strings.TrimSpace(stdout.String()); raw != "{}" {
		t.Fatalf("Stop reminded about a medium candidate: %s", raw)
	}

	recordRaw, err := os.ReadFile(filepath.Join(home, ".gentle-ai", "review-stop-hook", "v1", convID+".json"))
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	var record struct {
		TargetIdentity string `json:"target_identity"`
	}
	if err := json.Unmarshal(recordRaw, &record); err != nil {
		t.Fatalf("decode record: %v", err)
	}
	if record.TargetIdentity != "" {
		t.Fatalf("Stop recorded a reminder for a medium candidate: %q", recordRaw)
	}
}

// TestAntigravityStopBlocksOnAnUnassessableCandidate pins the fail-closed
// edge of the risk gate: when the candidate cannot be assessed (here, an
// untracked file the operator has not declared), review assess reports
// review_due so the hook keeps reminding. The hook never chooses
// --untracked-scope on the operator's behalf.
func TestAntigravityStopBlocksOnAnUnassessableCandidate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the review fixture uses POSIX git")
	}
	reviewModeHome(t)
	repo := initReviewCLIRepo(t)
	enableReviewStopHookRDD(t, repo)
	convID := "conv-stop-unassessable"

	var stdout bytes.Buffer
	prePayload := fmtAntigravityHookPayload(convID, []string{repo}, 0, "")
	if err := RunAntigravityCLIHook("PreInvocation", "antigravity-cli", strings.NewReader(prePayload), &stdout, repo); err != nil {
		t.Fatalf("PreInvocation error: %v", err)
	}
	// The hook leaves no untracked file of its own, so the untracked
	// population here is the operator's own work.
	if err := os.WriteFile(filepath.Join(repo, "notes.md"), []byte("notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stopPayload := fmtAntigravityStopPayload(convID, []string{repo}, 5, "model_stop", true)
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"decision":"continue"`) {
		t.Fatalf("Stop closed on an unassessable candidate: %s", stdout.String())
	}
}

// TestAntigravityStopBlocksUntilConsumed pins the consumed-check: after the
// first reminder, the Stop keeps blocking the same unreviewed candidate
// instead of de-silencing on the second attempt. Only a target that
// predates the session (baseline match) or an explicit consent refusal
// (stale marker) may pass.
// TestAntigravityStopRemindsOncePerMutationGeneration pins the receipt
// semantics Gentle Shell applies: a write this session recorded earns one
// reminder, and the next write earns the next one. The same generation does
// not earn a second reminder while no new mutation is pending.
func TestAntigravityStopRemindsOncePerMutationGeneration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the review fixture uses POSIX git")
	}
	reviewModeHome(t)
	repo := initReviewCLIRepo(t)
	// A repository that already ignores .atl keeps PreInvocation from creating
	// an untracked .gitignore, so the candidate stays assessable.
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(".atl\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runReviewCLIGit(t, repo, "add", ".gitignore")
	runReviewCLIGit(t, repo, "commit", "-qm", "ignore atl")
	enableReviewStopHookRDD(t, repo)
	convID := "conv-stop-generation"

	var stdout bytes.Buffer
	prePayload := fmtAntigravityHookPayload(convID, []string{repo}, 0, "")
	if err := RunAntigravityCLIHook("PreInvocation", "antigravity-cli", strings.NewReader(prePayload), &stdout, repo); err != nil {
		t.Fatalf("PreInvocation error: %v", err)
	}

	// A session write the ledger records, plus the candidate it produced.
	recordAntigravitySessionWrite(t, convID, repo, filepath.Join(repo, "service-token.ts"))
	stageReviewStopHookCandidate(t, repo)

	stopPayload := fmtAntigravityStopPayload(convID, []string{repo}, 5, "model_stop", true)
	stdout.Reset()
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"decision":"continue"`) {
		t.Fatalf("Stop did not remind about a due candidate: %s", stdout.String())
	}

	// Same generation, no new mutation: quiet.
	stdout.Reset()
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	if raw := strings.TrimSpace(stdout.String()); raw != "{}" {
		t.Fatalf("Stop reminded twice for one mutation generation: %s", raw)
	}

	// A new mutation earns a new reminder.
	recordAntigravitySessionWrite(t, convID, repo, filepath.Join(repo, "api", "service-token.ts"))
	writeReviewStartCandidate(t, repo, "api/service-token.ts", "export const token = 1;\n", 0o644)
	stdout.Reset()
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"decision":"continue"`) {
		t.Fatalf("Stop did not remind about a new mutation generation: %s", stdout.String())
	}
}

// TestAntigravityStopRemindsWhenUnattributedWorkChangesTheCandidate pins the
// gap F2 left open: a subagent writes in its own conversation and its writes
// never enter the parent's mutation ledger, so a ledger-only consumption
// would settle the reminder over work nobody reviewed. The reminder settles
// the candidate's content, so work the session cannot attribute re-arms it.
func TestAntigravityStopRemindsWhenUnattributedWorkChangesTheCandidate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the review fixture uses POSIX git")
	}
	reviewModeHome(t)
	repo := initReviewCLIRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(".atl\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runReviewCLIGit(t, repo, "add", ".gitignore")
	runReviewCLIGit(t, repo, "commit", "-qm", "ignore atl")
	enableReviewStopHookRDD(t, repo)
	convID := "conv-stop-unattributed"

	var stdout bytes.Buffer
	prePayload := fmtAntigravityHookPayload(convID, []string{repo}, 0, "")
	if err := RunAntigravityCLIHook("PreInvocation", "antigravity-cli", strings.NewReader(prePayload), &stdout, repo); err != nil {
		t.Fatalf("PreInvocation error: %v", err)
	}

	recordAntigravitySessionWrite(t, convID, repo, filepath.Join(repo, "service-token.ts"))
	stageReviewStopHookCandidate(t, repo)

	stopPayload := fmtAntigravityStopPayload(convID, []string{repo}, 5, "model_stop", true)
	stdout.Reset()
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"decision":"continue"`) {
		t.Fatalf("Stop did not remind about a due candidate: %s", stdout.String())
	}

	// Same content, no new mutation: settled.
	stdout.Reset()
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	if raw := strings.TrimSpace(stdout.String()); raw != "{}" {
		t.Fatalf("Stop reminded twice for one content generation: %s", raw)
	}

	// Work the session cannot attribute: a subagent's write, with no entry in
	// the parent's ledger. The content changed, so the reminder re-arms.
	writeReviewStartCandidate(t, repo, "service-token.ts", "export const token = 2;\n", 0o644)
	stdout.Reset()
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"decision":"continue"`) {
		t.Fatalf("Stop settled over work it cannot attribute: %s", stdout.String())
	}

	// And that generation settles too.
	stdout.Reset()
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	if raw := strings.TrimSpace(stdout.String()); raw != "{}" {
		t.Fatalf("Stop reminded twice for one content generation: %s", raw)
	}
}

// TestAntigravityStopStaysQuietWhenTheCandidateIsOnlyCommitted pins the
// fingerprint the receipt settles: the workspace content tree, which is
// stable across a commit. Committing the same content is not new work, so
// the reminder does not re-arm and the session closes.
func TestAntigravityStopStaysQuietWhenTheCandidateIsOnlyCommitted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the review fixture uses POSIX git")
	}
	reviewModeHome(t)
	repo := initReviewCLIRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(".atl\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runReviewCLIGit(t, repo, "add", ".gitignore")
	runReviewCLIGit(t, repo, "commit", "-qm", "ignore atl")
	enableReviewStopHookRDD(t, repo)
	convID := "conv-stop-committed"

	var stdout bytes.Buffer
	prePayload := fmtAntigravityHookPayload(convID, []string{repo}, 0, "")
	if err := RunAntigravityCLIHook("PreInvocation", "antigravity-cli", strings.NewReader(prePayload), &stdout, repo); err != nil {
		t.Fatalf("PreInvocation error: %v", err)
	}

	recordAntigravitySessionWrite(t, convID, repo, filepath.Join(repo, "service-token.ts"))
	stageReviewStopHookCandidate(t, repo)

	stopPayload := fmtAntigravityStopPayload(convID, []string{repo}, 5, "model_stop", true)
	stdout.Reset()
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"decision":"continue"`) {
		t.Fatalf("Stop did not remind about a due candidate: %s", stdout.String())
	}

	// The same content, now committed: the fingerprint does not move.
	runReviewCLIGit(t, repo, "add", "-A")
	runReviewCLIGit(t, repo, "commit", "-qm", "candidate")
	stdout.Reset()
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	if raw := strings.TrimSpace(stdout.String()); raw != "{}" {
		t.Fatalf("Stop re-armed for a commit of the same content: %s", raw)
	}
}

// TestAntigravityStopNamesTheCommittedRangeCandidate pins the reason text of
// the committed-range case: the candidate is the committed range, the
// worktree has no pending changes, and the review needs a base ref. Telling
// the model that unreviewed changes were detected sends it after a worktree
// diff that does not exist.
func TestAntigravityStopNamesTheCommittedRangeCandidate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the review fixture uses POSIX git")
	}
	reviewModeHome(t)
	repo := initReviewCLIRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(".atl\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runReviewCLIGit(t, repo, "add", ".gitignore")
	runReviewCLIGit(t, repo, "commit", "-qm", "ignore atl")
	enableReviewStopHookRDD(t, repo)
	convID := "conv-stop-committed-range"

	var stdout bytes.Buffer
	prePayload := fmtAntigravityHookPayload(convID, []string{repo}, 0, "")
	if err := RunAntigravityCLIHook("PreInvocation", "antigravity-cli", strings.NewReader(prePayload), &stdout, repo); err != nil {
		t.Fatalf("PreInvocation error: %v", err)
	}
	// The model commits its work: the worktree is clean again and the
	// candidate is the committed range that grew.
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("committed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runReviewCLIGit(t, repo, "add", "-A")
	runReviewCLIGit(t, repo, "commit", "-qm", "work")
	stdout.Reset()
	stopPayload := fmtAntigravityStopPayload(convID, []string{repo}, 5, "model_stop", true)
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	var out AntigravityCLIHookOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &out); err != nil {
		t.Fatalf("decode Stop output %q: %v", stdout.String(), err)
	}
	if out.Decision != "continue" {
		t.Fatalf("Stop did not block on an unreviewed committed range: %+v", out)
	}
	if !strings.Contains(out.Reason, "committed") {
		t.Fatalf("Stop did not name the committed range: %s", out.Reason)
	}
	if strings.Contains(out.Reason, "unreviewed candidate changes detected") {
		t.Fatalf("Stop claimed unreviewed changes with a clean worktree: %s", out.Reason)
	}
}

// TestAntigravityStopReasonReflectsReceiptSettlement pins that the reason
// stops promising a block the receipt no longer delivers: once a session
// write is attributed to the ledger, the reminder settles with the content.
func TestAntigravityStopReasonReflectsReceiptSettlement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the review fixture uses POSIX git")
	}
	reviewModeHome(t)
	repo := initReviewCLIRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(".atl\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runReviewCLIGit(t, repo, "add", ".gitignore")
	runReviewCLIGit(t, repo, "commit", "-qm", "ignore atl")
	enableReviewStopHookRDD(t, repo)
	convID := "conv-stop-reason"

	var stdout bytes.Buffer
	prePayload := fmtAntigravityHookPayload(convID, []string{repo}, 0, "")
	if err := RunAntigravityCLIHook("PreInvocation", "antigravity-cli", strings.NewReader(prePayload), &stdout, repo); err != nil {
		t.Fatalf("PreInvocation error: %v", err)
	}

	recordAntigravitySessionWrite(t, convID, repo, filepath.Join(repo, "service-token.ts"))
	stageReviewStopHookCandidate(t, repo)

	stdout.Reset()
	stopPayload := fmtAntigravityStopPayload(convID, []string{repo}, 5, "model_stop", true)
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(stopPayload), &stdout, repo); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	var out AntigravityCLIHookOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &out); err != nil {
		t.Fatalf("decode Stop output %q: %v", stdout.String(), err)
	}
	if strings.Contains(out.Reason, "keeps blocking until the review is consumed") {
		t.Fatalf("Stop promised a block its receipt no longer delivers: %s", out.Reason)
	}
	if !strings.Contains(out.Reason, "content generation") {
		t.Fatalf("Stop did not describe the receipt settlement: %s", out.Reason)
	}
}

// recordAntigravitySessionWrite simulates a completed write_to_file call, the
// PostToolUse observation the per-conversation mutation ledger records.
func recordAntigravitySessionWrite(t *testing.T, convID, repo, path string) {
	t.Helper()
	payload := fmtAntigravityToolCallPayload(convID, "write_to_file", map[string]any{"TargetFile": path}, 1)
	var stdout bytes.Buffer
	if err := RunAntigravityCLIHook("PostToolUse", "antigravity-cli", strings.NewReader(payload), &stdout, repo); err != nil {
		t.Fatalf("PostToolUse error: %v", err)
	}
}

func TestAntigravityStopBlocksUntilConsumed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the review fixture uses POSIX git")
	}
	for _, tc := range []struct {
		name      string
		stopCount int
		wantBlock int // how many stops should produce decision continue
	}{
		{"first stop blocks", 1, 1},
		{"second stop blocks too", 2, 2},
		{"third stop still blocks", 3, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reviewModeHome(t)
			repo := initReviewCLIRepo(t)
			// A due candidate: only high risk blocks now, so the consumed-check
			// is exercised on the same shape the policy reminds about.
			stageReviewStopHookCandidate(t, repo)
			blocked := 0
			convID := fmt.Sprintf("conv-stop-consumed-%d", 0)
			for i := 0; i < tc.stopCount; i++ {
				payload := fmtAntigravityStopPayload(convID, []string{repo}, i, "model_stop", true)
				var stdout bytes.Buffer
				if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(payload), &stdout, repo); err != nil {
					t.Fatalf("stop %d: %v", i+1, err)
				}
				if strings.Contains(stdout.String(), "continue") {
					blocked++
				}
			}
			if blocked < tc.wantBlock {
				t.Fatalf("blocked %d stops, want at least %d (dedup de-silenced the unreviewed candidate)", blocked, tc.wantBlock)
			}
		})
	}
}

// TestAntigravityStopSurfacesPreflightFailure pins the silence fix: when the
// status pipeline itself fails, the Stop emits a continue naming the error
// instead of a mute {} that lets the model close over potentially unreviewed
// changes without any signal.
func TestAntigravityStopSurfacesPreflightFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the review fixture uses POSIX git")
	}
	reviewModeHome(t)
	repo := initReviewCLIRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "tracked.txt"), []byte("change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Break the review status pipeline by making the install state undecodable,
	// so readGlobalRDDMode fails and the preflight cannot resolve.
	statePath := filepath.Join(os.Getenv("HOME"), ".gentle-ai", "state.json")
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	payload := fmtAntigravityStopPayload("conv-stop-silence-test", []string{repo}, 0, "model_stop", true)
	var stdout bytes.Buffer
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(payload), &stdout, repo); err != nil {
		t.Fatalf("RunAntigravityCLIHook: %v", err)
	}
	var out AntigravityCLIHookOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &out); err != nil {
		t.Fatalf("decode Stop output %q: %v", stdout.String(), err)
	}
	if out.Decision != "continue" || !strings.Contains(out.Reason, "could not resolve") {
		t.Fatalf("Stop = %+v, want a continue naming the preflight failure", out)
	}
}

// TestAntigravityPreToolUseDeniesSensitiveCommandLines pins that shell command
// lines referencing sensitive paths are denied directly to prevent leaking
// secrets (e.g. cat .env) even if the base binary is allowed.
func TestAntigravityPreToolUseDeniesSensitiveCommandLines(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, test := range []struct {
		name     string
		command  string
		wantDeny bool
	}{
		{"cat env", "cat .env", true},
		{"grep ssh key", "grep token ~/.ssh/id_rsa", true},
		{"tar whole aws dir", "tar czf backup.tar.gz ~/.aws", true},
		{"pem path", "openssl x509 -in certs/server.pem -text", true},
		{"quoted env mention", "git commit -m \"fix .env example\"", true}, // accepted false positive
		{"hosts.yml", "cat ~/.config/gh/hosts.yml", true},
		{"keychain", "security dump-keychain ~/Library/Keychains/login.keychain-db", true},
		{"ordinary command", "ls -la", false},
		{"pemx not pem", "cat report.pemx", false},
		{"build command", "go build ./...", false},
		{"empty command", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			payload := fmtAntigravityToolCallPayload("conv-sensitive-commands", "run_command", map[string]any{"CommandLine": test.command}, 1)
			var stdout bytes.Buffer
			if err := RunAntigravityCLIHook("PreToolUse", "antigravity-cli", strings.NewReader(payload), &stdout, ""); err != nil {
				t.Fatalf("RunAntigravityCLIHook() error = %v", err)
			}
			var output AntigravityCLIHookOutput
			if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
				t.Fatalf("decode hook output %q: %v", stdout.String(), err)
			}
			if test.wantDeny {
				if output.Decision != "deny" {
					t.Fatalf("decision = %q, want deny (output %q)", output.Decision, stdout.String())
				}
				if !strings.Contains(output.Reason, "sensitive path") {
					t.Fatalf("reason = %q, want the sensitive-path guard explanation", output.Reason)
				}
			} else {
				if output.Decision != "ask" {
					t.Fatalf("decision = %q, want ask for an ordinary command (output %q)", output.Decision, stdout.String())
				}
			}
		})
	}
}

// TestAntigravityPreToolUseForcesAskForDestructiveCommandLines pins that
// destructive commands (rm -rf, git reset --hard, git push --force, etc.)
// return force_ask to prompt the user interactively, bypassing cached allow rules.
func TestAntigravityPreToolUseForcesAskForDestructiveCommandLines(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, test := range []struct {
		name         string
		command      string
		wantForceAsk bool
	}{
		{"rm rf simple", "rm -rf /tmp/foo", true},
		{"rm r", "rm -r mydir", true},
		{"rm f", "rm -f file.txt", true},
		{"rm fr", "rm -fr dir", true},
		{"sudo rm rf", "sudo rm -rf /var/log/app", true},
		{"git reset hard", "git reset --hard", true},
		{"git reset hard HEAD~1", "git reset --hard HEAD~1", true},
		{"git push force", "git push origin main --force", true},
		{"git push -f", "git push origin main -f", true},
		{"git clean -fd", "git clean -fd", true},
		{"git clean -f", "git clean -f", true},
		{"git branch -D", "git branch -D feature", true},
		{"chmod 777 recursive", "chmod -R 777 .", true},
		{"chown recursive", "chown -R user:group /app", true},
		{"chained command with rm rf", "npm test && rm -rf dist", true},
		{"ordinary git status", "git status", false},
		{"ordinary git push", "git push origin main", false},
		{"ordinary git clean dry-run", "git clean -n", false},
		{"ordinary rm without r or f", "rm test.txt", false},
		{"ordinary file edit", "echo 'hello' > test.txt", false},
		{"go test", "go test ./...", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			payload := fmtAntigravityToolCallPayload("conv-destructive-commands", "run_command", map[string]any{"CommandLine": test.command}, 1)
			var stdout bytes.Buffer
			if err := RunAntigravityCLIHook("PreToolUse", "antigravity-cli", strings.NewReader(payload), &stdout, ""); err != nil {
				t.Fatalf("RunAntigravityCLIHook() error = %v", err)
			}
			var output AntigravityCLIHookOutput
			if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
				t.Fatalf("decode hook output %q: %v", stdout.String(), err)
			}
			if test.wantForceAsk {
				if output.Decision != "force_ask" {
					t.Fatalf("decision = %q, want force_ask (output %q)", output.Decision, stdout.String())
				}
				if !strings.Contains(output.Reason, "Destructive command detected") {
					t.Fatalf("reason = %q, want destructive command explanation", output.Reason)
				}
			} else if output.Decision == "force_ask" || output.Decision == "deny" {
				t.Fatalf("decision = %q for non-destructive command (output %q)", output.Decision, stdout.String())
			}
		})
	}
}
