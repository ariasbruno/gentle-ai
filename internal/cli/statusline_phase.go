package cli

// Statusline ODD-phase inference (S12). The PreToolUse hook mirrors Gentle
// Shell's tool-activity inference — reads → exploring, edits → implementing,
// test runs → checking — and records the phase for the status line in
// <HOME>/.gentle-ai/statusline-phase/<sha16(conversationID)>.json. The write
// is best-effort: it never changes a hook decision, never fails the hook, and
// prunes sibling records older than an hour. `gentle-ai statusline` reads the
// record when it is under ten minutes old and the agent is active; otherwise
// the agent_state fallback label applies.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	statusLinePhaseDir        = "statusline-phase"
	statusLinePhaseFreshTTL   = 10 * time.Minute
	statusLinePhasePruneAfter = time.Hour
)

type statusLinePhaseRecord struct {
	Phase string `json:"phase"`
	TS    int64  `json:"ts"`
}

func statusLinePhasePath(home, conversationID string) string {
	sum := sha256.Sum256([]byte(conversationID))
	return filepath.Join(home, ".gentle-ai", statusLinePhaseDir, hex.EncodeToString(sum[:16])+".json")
}

var statusLineTestCommandPattern = regexp.MustCompile(`\b(pytest|jest|vitest|test)\b|go test|cargo test|npm test`)

// inferStatusLinePhase maps one tool call to the ODD activity label. The
// order matters: the subagent check comes first, then command runs (test
// runs are checking, other commands implementing), then reads and writes;
// anything unrecognized keeps the generic working label.
func inferStatusLinePhase(call *AntigravityCLIToolCall) string {
	name := strings.ToLower(call.Name)
	switch {
	case strings.Contains(name, "subagent"):
		return "delegating"
	case strings.Contains(name, "run") || strings.Contains(name, "command") ||
		strings.Contains(name, "bash") || strings.Contains(name, "exec"):
		if raw, err := json.Marshal(call.Args); err == nil && statusLineTestCommandPattern.Match(raw) {
			return "checking"
		}
		return "implementing"
	case strings.Contains(name, "read") || strings.Contains(name, "list") ||
		strings.Contains(name, "search") || strings.Contains(name, "grep") || strings.Contains(name, "find"):
		return "exploring"
	case strings.Contains(name, "write") || strings.Contains(name, "edit") ||
		strings.Contains(name, "create") || strings.Contains(name, "move"):
		return "implementing"
	default:
		return "working"
	}
}

// recordAntigravityStatusLinePhase writes the phase record for the payload's
// conversation, best-effort: missing home, conversation id, or tool call is a
// no-op, and so is every write error — the phase never blocks the hook.
func recordAntigravityStatusLinePhase(payload AntigravityCLIHookPayload) {
	if payload.ConversationID == "" || payload.ToolCall == nil || payload.ToolCall.Name == "" {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return
	}
	record := statusLinePhaseRecord{
		Phase: inferStatusLinePhase(payload.ToolCall),
		TS:    time.Now().Unix(),
	}
	path := statusLinePhasePath(home, payload.ConversationID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, encoded, 0o644); err != nil {
		return
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return
	}
	pruneStatusLinePhaseRecords(filepath.Dir(path))
}

// pruneStatusLinePhaseRecords removes sibling phase records older than an
// hour; failures are ignored (the next write retries).
func pruneStatusLinePhaseRecords(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-statusLinePhasePruneAfter)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, statErr := entry.Info()
		if statErr != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}

// readStatusLinePhase returns the fresh phase label for a conversation, or ""
// when the record is absent, unreadable, or older than the fresh TTL.
func readStatusLinePhase(home, conversationID string) string {
	if home == "" || conversationID == "" {
		return ""
	}
	raw, err := os.ReadFile(statusLinePhasePath(home, conversationID))
	if err != nil {
		return ""
	}
	var record statusLinePhaseRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return ""
	}
	if record.Phase == "" || time.Since(time.Unix(record.TS, 0)) > statusLinePhaseFreshTTL {
		return ""
	}
	return record.Phase
}
