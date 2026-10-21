package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// antigravityReviewLensAgents names the read-only review lenses the parent
// dispatches through invoke_subagent. The names match the installed agent
// directories, and they arrive as structured tool-call arguments, so coverage
// never has to parse a prompt.
var antigravityReviewLensAgents = map[string]bool{
	"review-risk":        true,
	"review-resilience":  true,
	"review-readability": true,
	"review-reliability": true,
}

const (
	// antigravityReviewCoverageMaxRebounds bounds how many turns the gate may
	// add. A safety net that can loop is worse than no safety net.
	antigravityReviewCoverageMaxRebounds = 2
	// antigravityReviewCoverageEnv disables the gate for one session when set
	// to "off"; every other value, including unset, keeps it enabled.
	antigravityReviewCoverageEnv = "GENTLE_AI_AGY_REVIEW_COVERAGE"
	// antigravityTranscriptMaxLineBytes bounds one transcript line. The runtime
	// truncates message content, so a line stays far below this; the limit only
	// protects the scanner from failing on an unexpected record.
	antigravityTranscriptMaxLineBytes = 1 << 20
)

// antigravityReviewCoverage counts what one transcript shows about the review
// lenses: how many the parent dispatched and how many artifacts came back.
type antigravityReviewCoverage struct {
	Dispatched int
	Returned   int
}

type antigravityTranscriptRecord struct {
	Type      string                          `json:"type"`
	Content   string                          `json:"content"`
	ToolCalls []antigravityTranscriptToolCall `json:"tool_calls"`
}

type antigravityTranscriptToolCall struct {
	Name string                     `json:"name"`
	Args map[string]json.RawMessage `json:"args"`
}

// antigravityReviewCoverageFromTranscript derives coverage from one transcript
// file. The format is private to Antigravity CLI and is not a contract: every
// unrecognized record is skipped, and the caller fails open on any error.
func antigravityReviewCoverageFromTranscript(path string) (antigravityReviewCoverage, error) {
	file, err := os.Open(path)
	if err != nil {
		return antigravityReviewCoverage{}, err
	}
	defer func() { _ = file.Close() }()

	var coverage antigravityReviewCoverage
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), antigravityTranscriptMaxLineBytes)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var record antigravityTranscriptRecord
		if err := json.Unmarshal(line, &record); err != nil {
			continue
		}
		switch record.Type {
		case "PLANNER_RESPONSE":
			for _, call := range record.ToolCalls {
				if call.Name != "invoke_subagent" {
					continue
				}
				coverage.Dispatched += antigravityReviewLensesInCall(call.Args)
			}
		case "SYSTEM_MESSAGE":
			if isAntigravityReviewArtifact(record.Content) {
				coverage.Returned++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return coverage, err
	}
	return coverage, nil
}

// antigravityReviewLensesInCall counts the review lenses among the subagents of
// one invoke_subagent call. The Subagents argument is a JSON document encoded as
// a string, which is how the runtime accepts it.
func antigravityReviewLensesInCall(args map[string]json.RawMessage) int {
	raw, ok := args["Subagents"]
	if !ok {
		return 0
	}
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return 0
	}
	var subagents []struct {
		TypeName string `json:"TypeName"`
	}
	if err := json.Unmarshal([]byte(encoded), &subagents); err != nil {
		return 0
	}
	count := 0
	for _, subagent := range subagents {
		if antigravityReviewLensAgents[subagent.TypeName] {
			count++
		}
	}
	return count
}

// isAntigravityReviewArtifact reports whether a subagent message carries a
// reviewer result. Only the visible prefix is inspected: the runtime truncates
// long message content, so a complete parse is not available from a transcript
// and a partial artifact must not be mistaken for prose.
func isAntigravityReviewArtifact(content string) bool {
	const (
		senderMarker  = "sender="
		contentMarker = "content="
	)
	if !strings.Contains(content, senderMarker) {
		return false
	}
	index := strings.Index(content, contentMarker)
	if index < 0 {
		return false
	}
	artifact := strings.TrimSpace(content[index+len(contentMarker):])
	if !strings.HasPrefix(artifact, "{") {
		return false
	}
	// The lens contract requires these top-level fields, so their presence in
	// the visible prefix separates an artifact from prose that opens with a
	// brace. Fields beyond the truncation boundary cannot be checked here.
	return strings.Contains(artifact, `"subject_hash"`) && strings.Contains(artifact, `"findings"`)
}

func antigravityReviewCoveragePath(home, conversationID string) string {
	return filepath.Join(home, ".gentle-ai", "antigravity-review-coverage", "v1", conversationID+".json")
}

func readAntigravityReviewCoverageRebounds(path string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var record struct {
		Rebounds int `json:"rebounds"`
	}
	if err := json.Unmarshal(raw, &record); err != nil {
		return 0
	}
	return record.Rebounds
}

func recordAntigravityReviewCoverageRebound(path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	record := struct {
		Rebounds int `json:"rebounds"`
	}{Rebounds: readAntigravityReviewCoverageRebounds(path) + 1}
	encoded, err := json.Marshal(record)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, encoded, 0o644)
}

// antigravityReviewCoverageDecision returns the Stop response that keeps the
// session open when dispatched review lenses did not report. It returns nil when
// coverage holds, the gate is disabled, or anything is unreadable: this is a
// safety net, never the reason a session cannot close.
func antigravityReviewCoverageDecision(payload AntigravityCLIHookPayload, home string) *AntigravityCLIHookOutput {
	if strings.EqualFold(strings.TrimSpace(os.Getenv(antigravityReviewCoverageEnv)), "off") {
		return nil
	}
	if payload.TranscriptPath == "" || payload.ConversationID == "" {
		return nil
	}
	coverage, err := antigravityReviewCoverageFromTranscript(payload.TranscriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gentle-ai antigravity hook: review coverage unavailable: %v\n", err)
		return nil
	}
	if coverage.Dispatched == 0 || coverage.Returned >= coverage.Dispatched {
		return nil
	}
	path := antigravityReviewCoveragePath(home, payload.ConversationID)
	if rebounds := readAntigravityReviewCoverageRebounds(path); rebounds >= antigravityReviewCoverageMaxRebounds {
		fmt.Fprintf(os.Stderr, "gentle-ai antigravity hook: review coverage still short after %d rebounds (%d of %d artifacts)\n",
			rebounds, coverage.Returned, coverage.Dispatched)
		return nil
	}
	recordAntigravityReviewCoverageRebound(path)
	return &AntigravityCLIHookOutput{
		Decision: "continue",
		Reason: fmt.Sprintf(
			"Review coverage: %d review lens(es) were dispatched and %d artifact(s) came back. Re-run every missing lens with invoke_subagent; each lens must return exactly one JSON object carrying subject_hash, inspection, findings, and evidence.",
			coverage.Dispatched, coverage.Returned),
	}
}
