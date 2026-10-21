package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeAntigravityTranscript(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	body := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func antigravityDispatchLine(t *testing.T, typeNames ...string) string {
	t.Helper()
	subagents := make([]string, 0, len(typeNames))
	for _, name := range typeNames {
		subagents = append(subagents, fmt.Sprintf(`{"Prompt":"review the candidate","Role":"lens","TypeName":%q}`, name))
	}
	return antigravityTranscriptLine(t, map[string]any{
		"type": "PLANNER_RESPONSE",
		"tool_calls": []any{map[string]any{
			"name": "invoke_subagent",
			"args": map[string]any{"Subagents": "[" + strings.Join(subagents, ",") + "]", "toolAction": "Review"},
		}},
	})
}

func antigravityMessageLine(t *testing.T, sender, content string) string {
	t.Helper()
	body := "The following is a <SYSTEM_MESSAGE> not actually sent by the user.\n\n<SYSTEM_MESSAGE>\n" +
		"[Message] timestamp=2026-10-20T00:00:00Z sender=" + sender + " priority=MESSAGE_PRIORITY_HIGH content=" + content
	return antigravityTranscriptLine(t, map[string]any{"type": "SYSTEM_MESSAGE", "content": body})
}

func antigravityTranscriptLine(t *testing.T, record map[string]any) string {
	t.Helper()
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func TestAntigravityReviewCoverageCountsDispatchedAndReturned(t *testing.T) {
	path := writeAntigravityTranscript(t,
		antigravityDispatchLine(t, "review-risk", "review-resilience"),
		antigravityMessageLine(t, "child-a", `{"subject_hash":"abc","inspection":{"status":"completed"},"findings":[]}`),
		antigravityDispatchLine(t, "gentle-ai-worker", "review-readability"),
		antigravityMessageLine(t, "child-b", `{"subject_hash":"abc","inspection":{"status":"completed"},"findings":[],"evidence":[]}`),
		antigravityMessageLine(t, "child-c", "I reviewed the candidate and everything looks fine."),
	)

	coverage, err := antigravityReviewCoverageFromTranscript(path)
	if err != nil {
		t.Fatal(err)
	}
	if coverage.Dispatched != 3 {
		t.Errorf("Dispatched = %d, want 3 (review lenses only)", coverage.Dispatched)
	}
	if coverage.Returned != 2 {
		t.Errorf("Returned = %d, want 2 (prose is not an artifact)", coverage.Returned)
	}
}

func TestIsAntigravityReviewArtifact(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"no sender marker", "content={}", false},
		{"prose", "sender=child content=I found three problems.", false},
		{"brace without lens fields", `sender=child content={"summary":"ok"}`, false},
		{"complete artifact", `sender=child content={"subject_hash":"abc","findings":[]}`, true},
		{"truncated artifact", `sender=child content={"subject_hash":"abc","inspection":{"status":"completed"},"findings":[{"severity":"feature`, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isAntigravityReviewArtifact(test.content); got != test.want {
				t.Fatalf("isAntigravityReviewArtifact(%q) = %v, want %v", test.content, got, test.want)
			}
		})
	}
}

func TestAntigravityReviewCoverageDecisionReboundsThenFailsOpen(t *testing.T) {
	home := t.TempDir()
	short := writeAntigravityTranscript(t,
		antigravityDispatchLine(t, "review-risk", "review-resilience"),
		antigravityMessageLine(t, "child-a", `{"subject_hash":"abc","findings":[]}`),
	)
	payload := AntigravityCLIHookPayload{ConversationID: "conv-short", TranscriptPath: short}

	for rebound := 1; rebound <= antigravityReviewCoverageMaxRebounds; rebound++ {
		out := antigravityReviewCoverageDecision(payload, home)
		if out == nil {
			t.Fatalf("rebound %d: decision = nil, want a continue response", rebound)
		}
		if out.Decision != "continue" {
			t.Fatalf("rebound %d: decision = %q, want continue", rebound, out.Decision)
		}
		if !strings.Contains(out.Reason, "2 review lens(es)") || !strings.Contains(out.Reason, "1 artifact(s)") {
			t.Fatalf("rebound %d: reason = %q, want the dispatched and returned counts", rebound, out.Reason)
		}
	}
	if out := antigravityReviewCoverageDecision(payload, home); out != nil {
		t.Fatalf("decision after %d rebounds = %+v, want nil so the session can close", antigravityReviewCoverageMaxRebounds, out)
	}
}

func TestAntigravityReviewCoverageDecisionStaysSilentWhenCoverageHolds(t *testing.T) {
	home := t.TempDir()
	full := writeAntigravityTranscript(t,
		antigravityDispatchLine(t, "review-risk"),
		antigravityMessageLine(t, "child-a", `{"subject_hash":"abc","findings":[]}`),
	)
	cases := map[string]struct {
		payload AntigravityCLIHookPayload
		env     string
	}{
		"coverage holds": {payload: AntigravityCLIHookPayload{ConversationID: "conv-full", TranscriptPath: full}},
		"no dispatch": {payload: AntigravityCLIHookPayload{
			ConversationID: "conv-none",
			TranscriptPath: writeAntigravityTranscript(t, antigravityDispatchLine(t, "gentle-ai-worker")),
		}},
		"missing transcript": {payload: AntigravityCLIHookPayload{
			ConversationID: "conv-missing",
			TranscriptPath: filepath.Join(home, "absent.jsonl"),
		}},
		"no conversation id": {payload: AntigravityCLIHookPayload{TranscriptPath: full}},
		"gate disabled": {
			payload: AntigravityCLIHookPayload{ConversationID: "conv-off", TranscriptPath: writeAntigravityTranscript(t,
				antigravityDispatchLine(t, "review-risk"),
				antigravityMessageLine(t, "child-a", "no artifact here"),
			)},
			env: "off",
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			if test.env != "" {
				t.Setenv(antigravityReviewCoverageEnv, test.env)
			}
			if out := antigravityReviewCoverageDecision(test.payload, home); out != nil {
				t.Fatalf("decision = %+v, want nil", out)
			}
		})
	}
}

func TestAntigravityStopHookReboundsOnMissingReviewArtifact(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	transcript := writeAntigravityTranscript(t,
		antigravityDispatchLine(t, "review-risk", "review-reliability"),
		antigravityMessageLine(t, "child-a", `{"subject_hash":"abc","findings":[]}`),
	)
	payload, err := json.Marshal(map[string]any{
		"conversationId":    "conv-stop",
		"transcriptPath":    transcript,
		"terminationReason": "model_stop",
		"fullyIdle":         true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// A workspace outside any repository keeps the pre-existing RDD Stop gate
	// silent, so this test observes only the coverage gate.
	workspace := t.TempDir()
	var first strings.Builder
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(string(payload)), &first, workspace); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(first.String(), `"decision":"continue"`) || !strings.Contains(first.String(), "Review coverage") {
		t.Fatalf("first stop = %s, want a coverage continue", first.String())
	}

	var second strings.Builder
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(string(payload)), &second, workspace); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(second.String(), "Review coverage") {
		t.Fatalf("second stop = %s, want a second coverage rebound", second.String())
	}
	var third strings.Builder
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(string(payload)), &third, workspace); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(third.String()) != "{}" {
		t.Fatalf("third stop = %s, want {} after the rebound cap", third.String())
	}
}

func TestAntigravityStopHookIgnoresNonIdleStop(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	transcript := writeAntigravityTranscript(t, antigravityDispatchLine(t, "review-risk"))
	payload, err := json.Marshal(map[string]any{
		"conversationId":    "conv-idle",
		"transcriptPath":    transcript,
		"terminationReason": "model_stop",
		"fullyIdle":         false,
	})
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := RunAntigravityCLIHook("Stop", "antigravity-cli", strings.NewReader(string(payload)), &out, ""); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "{}" {
		t.Fatalf("stop = %s, want {} for a non-idle stop", out.String())
	}
}
