package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gentleman-programming/gentle-ai/v4/internal/reviewtransaction"
	"github.com/gentleman-programming/gentle-ai/v4/internal/skillregistry"
)

// AntigravityCLIToolCall represents a tool invocation requested by the model.
type AntigravityCLIToolCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// AntigravityCLIHookPayload defines the JSON payload sent by Antigravity CLI to hooks.
type AntigravityCLIHookPayload struct {
	ConversationID        string                  `json:"conversationId"`
	WorkspacePaths        []string                `json:"workspacePaths"`
	InvocationNum         int                     `json:"invocationNum"`
	InitialNumSteps       int                     `json:"initialNumSteps"`
	ExecutionNum          int                     `json:"executionNum"`
	Error                 string                  `json:"error,omitempty"`
	TranscriptPath        string                  `json:"transcriptPath,omitempty"`
	ArtifactDirectoryPath string                  `json:"artifactDirectoryPath,omitempty"`
	ModelName             string                  `json:"modelName,omitempty"`
	StepIdx               int                     `json:"stepIdx"`
	TerminationReason     string                  `json:"terminationReason"`
	FullyIdle             bool                    `json:"fullyIdle"`
	ToolCall              *AntigravityCLIToolCall `json:"toolCall,omitempty"`
}

// AntigravityCLIHookOutput represents the response envelope returned to Antigravity CLI.
type AntigravityCLIHookOutput struct {
	Decision            string `json:"decision,omitempty"`
	Reason              string `json:"reason,omitempty"`
	InjectSteps         []any  `json:"injectSteps,omitempty"`
	TerminationBehavior string `json:"terminationBehavior,omitempty"`
}

// RunHook is the CLI entry point for `gentle-ai hook [args...]`.
func RunHook(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("hook subcommand required (e.g. gentle-ai hook run)")
	}

	switch args[0] {
	case "run":
		return runHookCommand(args[1:], os.Stdin, stdout)
	default:
		// refusal:by-design operator-knowledge: the caller must use a supported hook subcommand; only "run" is defined
		return fmt.Errorf("unknown hook subcommand %q", args[0])
	}
}

const antigravityCLIHookEventHelp = "hook event name (PreInvocation, PreToolUse, PostToolUse, PostInvocation, Stop)"

func runHookCommand(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("hook run", flag.ContinueOnError)
	agent := fs.String("agent", "", "target agent (required: antigravity-cli)")
	event := fs.String("event", "", antigravityCLIHookEventHelp)
	cwd := fs.String("cwd", "", "optional workspace cwd override")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *agent != "antigravity-cli" {
		// refusal:by-design operator-knowledge: the caller must pass --agent with the correct target agent for this hook
		return fmt.Errorf("hook run requires agent %q; got %q", "antigravity-cli", *agent)
	}

	return RunAntigravityCLIHook(*event, *agent, stdin, stdout, *cwd)
}

// RunAntigravityCLIHook handles the official Antigravity CLI lifecycle events.
// The caller states which runtime is executing; the printed reminders never
// spell out the identity themselves (issue #2440).
func RunAntigravityCLIHook(event, agent string, stdin io.Reader, stdout io.Writer, cwdOverride string) error {
	raw, err := io.ReadAll(io.LimitReader(stdin, maxReviewStopHookPayloadBytes))
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "{}")
		return nil
	}

	var payload AntigravityCLIHookPayload
	if event == "PreToolUse" && !isAntigravityPreToolUseObject(raw) {
		return writeAntigravityPreToolUseDecision(stdout, "deny", "malformed Antigravity PreToolUse payload.")
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil && event == "PreToolUse" {
			// A malformed PreToolUse payload is never an allow. If it still
			// identifies a native invoke_subagent call, retain the specific
			// reason; otherwise fail closed on the malformed transport itself.
			if decision, reason, applies := evaluateAntigravityCLIPreToolUse(raw); applies {
				return writeAntigravityPreToolUseDecision(stdout, decision, reason)
			}
			return writeAntigravityPreToolUseDecision(stdout, "deny", "malformed Antigravity PreToolUse payload.")
		}
	}

	workspace := cwdOverride
	if workspace == "" && len(payload.WorkspacePaths) > 0 {
		workspace = payload.WorkspacePaths[0]
	}
	if workspace == "" {
		workspace, _ = os.Getwd()
	}

	switch event {
	case "PreInvocation":
		return handleAntigravityPreInvocation(payload, workspace, stdout)
	case "PreToolUse":
		return handleAntigravityPreToolUse(payload, workspace, stdout)
	case "PostToolUse":
		return handleAntigravityPostToolUse(payload, workspace, stdout)
	case "PostInvocation":
		return handleAntigravityPostInvocation(payload, workspace, stdout)
	case "Stop":
		return handleAntigravityStop(payload, workspace, agent, stdout)
	default:
		_, _ = fmt.Fprintln(stdout, "{}")
		return nil
	}
}

// antigravityCLIMutationRecord is one observed file write in the per-conversation
// mutation ledger. The ledger mirrors what Gentle Shell's tool_result handler
// records: every write_to_file/replace_file_content/multi_replace_file_content
// that completed without error feeds the review evidence even when the
// status pipeline itself is unavailable.
type antigravityCLIMutationRecord struct {
	ToolName  string `json:"tool_name"`
	Path      string `json:"path"`
	Timestamp int64  `json:"timestamp"`
}

// antigravityCLIMutationLedgerPath returns the per-conversation mutation
// ledger file, co-located with the circuit-breaker records.
func antigravityCLIMutationLedgerPath(home, conversationID string) string {
	sum := sha256.Sum256([]byte(conversationID))
	return filepath.Join(home, ".gentle-ai", "antigravity-mutations", "v1", hex.EncodeToString(sum[:16])+".jsonl")
}

// recordAntigravityCLIMutation appends one write observation to the ledger.
// The append is best-effort: a ledger failure never blocks the tool result.
func recordAntigravityCLIMutation(home, conversationID, toolName, path string) {
	if conversationID == "" || path == "" {
		return
	}
	record := antigravityCLIMutationRecord{ToolName: toolName, Path: path, Timestamp: time.Now().Unix()}
	encoded, err := json.Marshal(record)
	if err != nil {
		return
	}
	ledgerPath := antigravityCLIMutationLedgerPath(home, conversationID)
	if err := os.MkdirAll(filepath.Dir(ledgerPath), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(ledgerPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.Write(append(encoded, '\n'))
}

// antigravityStopCandidateKind names what the candidate behind a Stop really
// is, so the reminder tells the model what it is looking at instead of
// claiming unreviewed changes when the worktree has none.
type antigravityStopCandidateKind int

const (
	// antigravityStopCandidateChanges is a candidate with pending changes to
	// review.
	antigravityStopCandidateChanges antigravityStopCandidateKind = iota
	// antigravityStopCandidateCommittedRange is the committed range: the
	// worktree has no pending changes and the review needs a base ref.
	antigravityStopCandidateCommittedRange
	// antigravityStopCandidateUntrackedSelection is a candidate whose review
	// is waiting for the untracked selection to be declared.
	antigravityStopCandidateUntrackedSelection
)

// antigravityStopCandidateOf classifies the candidate from the STATUS
// transition the hook already resolved.
func antigravityStopCandidateOf(status ReviewTargetStatusResult) antigravityStopCandidateKind {
	if status.NextTransition == nil {
		return antigravityStopCandidateChanges
	}
	switch status.NextTransition.ReasonCode {
	case "empty_candidate_base_ref_required":
		return antigravityStopCandidateCommittedRange
	case "intended_untracked_selection_required":
		return antigravityStopCandidateUntrackedSelection
	default:
		return antigravityStopCandidateChanges
	}
}

// antigravityCLIMutationLedgerState returns the state of the conversation's
// mutation ledger for one repository root: how many entries the ledger holds
// in total, how many of them were written under root, and how many of those
// are still pending after the consumed watermark. A malformed line (a partial
// append from a concurrent write) is skipped instead of failing the read.
//
// The watermark counts ledger entries, not writes under root, so consuming
// settles the prefix recorded so far and a write that lands during the
// review stays pending: the same consumption semantics Gentle Shell applies
// to its receipt ledger.
func antigravityCLIMutationLedgerState(home, conversationID, root string, consumed int) (records int, underRoot int, pending int) {
	if conversationID == "" {
		return 0, 0, 0
	}
	data, err := os.ReadFile(antigravityCLIMutationLedgerPath(home, conversationID))
	if err != nil {
		return 0, 0, 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var record antigravityCLIMutationRecord
		if json.Unmarshal([]byte(line), &record) != nil {
			continue
		}
		records++
		if !antigravityCLIMutationUnderRoot(record.Path, root) {
			continue
		}
		underRoot++
		if records > consumed {
			pending++
		}
	}
	return records, underRoot, pending
}

// antigravityCLIMutationUnderRoot reports whether a recorded mutation path
// belongs to the reviewed repository. A relative path cannot be attributed
// and counts as outside every root, so the gate keeps its strict behavior
// instead of settling a write it cannot place.
func antigravityCLIMutationUnderRoot(path, root string) bool {
	if path == "" || root == "" {
		return false
	}
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(root), clean)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// PostToolUse records completed file writes in the per-conversation mutation
// ledger (the Gentle Shell tool_result equivalent) and returns the
// contract-mandated {} — PostToolUse has no decision vocabulary.
func handleAntigravityPostToolUse(payload AntigravityCLIHookPayload, _ string, stdout io.Writer) error {
	if payload.ToolCall != nil && payload.Error == "" {
		switch payload.ToolCall.Name {
		case "write_to_file", "replace_file_content", "multi_replace_file_content":
			if path, ok := antigravityCLIRequiredString(payload.ToolCall.Args, "TargetFile"); ok {
				home, _ := os.UserHomeDir()
				recordAntigravityCLIMutation(home, payload.ConversationID, payload.ToolCall.Name, path)
			}
		case "write_file", "edit_file":
			if path, ok := antigravityCLIRequiredString(payload.ToolCall.Args, "Path"); ok {
				home, _ := os.UserHomeDir()
				recordAntigravityCLIMutation(home, payload.ConversationID, payload.ToolCall.Name, path)
			}
		}
	}
	_, _ = fmt.Fprintln(stdout, "{}")
	return nil
}

func handleAntigravityPostInvocation(_ AntigravityCLIHookPayload, _ string, stdout io.Writer) error {
	_, _ = fmt.Fprintln(stdout, "{}")
	return nil
}

func handleAntigravityPreInvocation(payload AntigravityCLIHookPayload, workspace string, stdout io.Writer) error {
	if payload.InvocationNum == 0 {
		// Silent refresh of skill registry
		home, _ := os.UserHomeDir()
		if home != "" && workspace != "" {
			if reason := skillregistry.RefreshSkip(workspace, home); reason == skillregistry.SkipNone {
				_ = skillregistry.EnsureATLIgnored(workspace)
				_, _ = skillregistry.Regenerate(workspace, home, false)
			}
		}

		// RDD baseline recording
		if payload.ConversationID != "" && workspace != "" {
			ctx := context.Background()
			_, targetIdentity, _, ok, _ := antigravityResolveTargetIdentity(ctx, workspace)
			if ok {
				_ = recordReviewStopHookBaseline(payload.ConversationID, targetIdentity)
			}
		}
	}

	_, _ = fmt.Fprintln(stdout, "{}")
	return nil
}

func antigravityResolveTargetIdentity(ctx context.Context, repo string) (root, targetIdentity string, status ReviewTargetStatusResult, ok bool, err error) {
	if repo == "" {
		return "", "", ReviewTargetStatusResult{}, false, nil
	}
	root, rootErr := (reviewtransaction.SnapshotBuilder{Repo: repo}).ResolveRepositoryRoot(ctx)
	if rootErr != nil {
		return "", "", ReviewTargetStatusResult{}, false, nil
	}

	global, globalErr := readGlobalRDDMode()
	if globalErr != nil {
		return "", "", ReviewTargetStatusResult{}, false, fmt.Errorf("antigravity hook: read global review mode: %w", globalErr)
	}
	rddStatus, rddErr := reviewtransaction.ResolveRDDMode(ctx, root, global)
	if rddErr != nil {
		return "", "", ReviewTargetStatusResult{}, false, fmt.Errorf("antigravity hook: resolve review mode: %w", rddErr)
	}
	if !rddStatus.Enabled() {
		return "", "", ReviewTargetStatusResult{}, false, nil
	}

	var statusOutput bytes.Buffer
	statusErr := runReviewStatus(ctx, []string{
		"--cwd", root, "--contract", ReviewIntegrationContractV2,
		"--next-transition",
	}, &statusOutput)
	if statusErr != nil {
		return "", "", ReviewTargetStatusResult{}, false, fmt.Errorf("antigravity hook: review status: %w", statusErr)
	}
	var result ReviewTargetStatusResult
	if err := json.Unmarshal(statusOutput.Bytes(), &result); err != nil {
		return "", "", ReviewTargetStatusResult{}, false, fmt.Errorf("antigravity hook: decode review status: %w", err)
	}
	return root, result.TargetIdentity, result, true, nil
}

func handleAntigravityStop(payload AntigravityCLIHookPayload, workspace, agent string, stdout io.Writer) error {
	if (payload.TerminationReason != "model_stop" && payload.TerminationReason != "NO_TOOL_CALL") || !payload.FullyIdle {
		_, _ = fmt.Fprintln(stdout, "{}")
		return nil
	}

	ctx := context.Background()
	root, targetIdentity, status, ok, err := antigravityResolveTargetIdentity(ctx, workspace)
	// The pipeline failure stays visible: the model sees why the preflight
	// could not resolve and can surface the problem instead of closing
	// silently over potentially unreviewed changes. The !ok && err == nil
	// cases (no workspace, no repository, RDD off) remain silent by design.
	if err != nil {
		out := AntigravityCLIHookOutput{
			Decision: "continue",
			Reason:   "Receipt-Driven Development: the review preflight could not resolve the review state. Run gentle-ai review status to see the problem before completing.",
		}
		enc, _ := json.Marshal(out)
		_, _ = fmt.Fprintln(stdout, string(enc))
		return nil
	}
	if !ok {
		_, _ = fmt.Fprintln(stdout, "{}")
		return nil
	}

	// A candidate is guardable when STATUS reports Action=start, meaning there
	// are unreviewed tracked changes. We do not gate on NextTransition here
	// because the workspace may contain untracked files (the operator's own,
	// or the model's) that cause STATUS to return
	// collect/intended_untracked_selection_required even when there is a
	// valid candidate — the Action field reflects the candidate state
	// independently of untracked-file resolution.
	if targetIdentity == "" || status.Action != reviewtransaction.TargetStatusActionStart {
		_, _ = fmt.Fprintln(stdout, "{}")
		return nil
	}

	// The reminder record de-silences only when the candidate predates the
	// session (baseline match) or the operator explicitly refused consent
	// (stale marker). The same-target reminder record no longer silences:
	// a candidate that still reports Action=start has not been reviewed, so
	// the Stop keeps blocking until the review is consumed and the status
	// moves past start. This is the consumed-check: the model cannot close
	// over an unreviewed candidate by ignoring the first reminder.
	if payload.ConversationID != "" && reviewStopHookSessionIDPattern.MatchString(payload.ConversationID) {
		home, herr := os.UserHomeDir()
		if herr != nil {
			_, _ = fmt.Fprintln(stdout, "{}")
			return nil
		}
		record, hadRecord, rerr := readReviewStopHookRecord(reviewStopHookRecordPath(home, payload.ConversationID))
		if rerr != nil {
			_, _ = fmt.Fprintln(stdout, "{}")
			return nil
		}
		if hadRecord && record.BaselineTargetIdentity != "" && record.BaselineTargetIdentity == targetIdentity {
			_, _ = fmt.Fprintln(stdout, "{}")
			return nil
		}
		if marker, mok, merr := readReviewConsentStaleMarker(root); merr == nil && mok &&
			time.Since(marker.LastRefusedAt) < reviewConsentStaleMarkerWindow {
			if record.LastSeenTargetIdentity != targetIdentity {
				_ = recordReviewStopHookLastSeen(payload.ConversationID, targetIdentity)
				_, _ = fmt.Fprintln(stdout, "{}")
				return nil
			}
		}
	}

	// The session mutation ledger turns the reminder into a receipt: a write
	// this session recorded earns one reminder, and the next write earns the
	// next one. That is the consumption contract Gentle Shell applies to its
	// receipt ledger. A candidate the ledger cannot attribute -- the session
	// recorded no write under this repository, or wrote through a tool the
	// ledger does not observe -- keeps the strict consumed-check: the Stop
	// keeps blocking until the review is consumed.
	//
	// The receipt settles the candidate's content, not only the session's own
	// writes. A subagent runs in its own conversation and its writes never
	// enter this ledger, so a ledger-only consumption would settle over work
	// nobody reviewed. The content fingerprint (the workspace snapshot tree)
	// is stable across a commit and changes whenever any file's content
	// changes, so unattributed work re-arms the reminder.
	ledgerRecords, ledgerUnderRoot, ledgerPending := 0, 0, 0
	ledgerTracked := false
	contentTree := status.Projection.CurrentCandidateTree
	if payload.ConversationID != "" && reviewStopHookSessionIDPattern.MatchString(payload.ConversationID) {
		home, herr := os.UserHomeDir()
		if herr != nil {
			_, _ = fmt.Fprintln(stdout, "{}")
			return nil
		}
		record, _, rerr := readReviewStopHookRecord(reviewStopHookRecordPath(home, payload.ConversationID))
		if rerr != nil {
			_, _ = fmt.Fprintln(stdout, "{}")
			return nil
		}
		ledgerTracked = true
		ledgerRecords, ledgerUnderRoot, ledgerPending = antigravityCLIMutationLedgerState(home, payload.ConversationID, root, record.ConsumedMutationCount)
		if ledgerUnderRoot > 0 && ledgerPending == 0 &&
			contentTree != "" && record.ConsumedContentTree == contentTree {
			// Every write this session recorded under this repository was
			// already reminded, and the candidate's content has not moved
			// since: this generation is settled.
			_, _ = fmt.Fprintln(stdout, "{}")
			return nil
		}
	}

	// Only a review_due candidate earns a reminder, by the same in-process
	// assessment `review assess` publishes: a passive or medium candidate
	// closes with a neutral {} even while STATUS still reports an unreviewed
	// candidate. An assessment that cannot answer fails closed to due, so the
	// reminder is never lost to an assessment error.
	if !reviewStopHookReviewDue(root, agent) {
		_, _ = fmt.Fprintln(stdout, "{}")
		return nil
	}

	out := AntigravityCLIHookOutput{
		Decision: "continue",
	}
	// The reminder carries the exact commands the model needs: the STATUS
	// preflight and, when available, the provider-issued START. This mirrors
	// the Claude stop hook's reason text so AGY models get the same actionable
	// detail instead of a generic "run preflight" that they may not act on.
	statusCommand := fmt.Sprintf("gentle-ai review status --cwd %s --contract %s --agent %s --next-transition", root, ReviewIntegrationContractV2, agent)
	reason := "Receipt-Driven Development: "
	switch antigravityStopCandidateOf(status) {
	case antigravityStopCandidateCommittedRange:
		reason += "unreviewed committed work detected since session baseline (target " + targetIdentity + "). The worktree has no pending changes: the candidate is the committed range, and the review needs a base ref to compare against."
	case antigravityStopCandidateUntrackedSelection:
		reason += "unreviewed candidate changes detected since session baseline (target " + targetIdentity + "). Some files are not trackable yet: the review needs the untracked selection to be declared."
	default:
		reason += "unreviewed candidate changes detected since session baseline (target " + targetIdentity + ")."
	}
	reason += " Run the STATUS preflight: " + statusCommand
	if status.NextTransition != nil && status.NextTransition.Execute != nil && status.NextTransition.Execute.Command != "" {
		reason += ". Then run the provider-issued START unchanged: " + status.NextTransition.Execute.Command
	}
	if ledgerTracked && ledgerUnderRoot > 0 {
		reason += ". This Stop reminds once per content generation: it stays quiet while the candidate's content does not change."
	} else {
		reason += ". This Stop keeps blocking until the review is consumed."
	}
	out.Reason = reason
	enc, _ := json.Marshal(out)
	_, _ = fmt.Fprintln(stdout, string(enc))

	if payload.ConversationID != "" {
		_, _ = recordReviewStopHookReminder(payload.ConversationID, targetIdentity)
	}
	// Consumption happens after the reminder is delivered, and only for the
	// writes this session recorded: the receipt settles the prefix the
	// reminder covered, so a later write earns its own reminder, and it
	// settles the content the reminder covered, so work this session cannot
	// attribute earns one too.
	if ledgerTracked && ledgerUnderRoot > 0 {
		_ = recordReviewStopHookConsumption(payload.ConversationID, ledgerRecords, contentTree)
	}

	return nil
}

const (
	antigravityCLIInvokeSubagentTool            = "invoke_subagent"
	antigravityCLISubagentsKey                  = "Subagents"
	antigravityCLISubagentTypeKey               = "TypeName"
	antigravityCLISubagentRoleKey               = "Role"
	antigravityCLISubagentPromptKey             = "Prompt"
	antigravityCLISubagentSpaceKey              = "Workspace"
	antigravityCLIInvokeSubagentMalformedReason = "Antigravity invoke_subagent arguments are malformed; expected native Subagents entries with TypeName, Role, and Prompt."
	maxConsecutiveIdenticalToolCalls            = 3
)

type circuitBreakerRecord struct {
	LastToolName     string `json:"last_tool_name"`
	LastArgsHash     string `json:"last_args_hash"`
	ConsecutiveCount int    `json:"consecutive_count"`
	UpdatedAt        int64  `json:"updated_at"`
}

func circuitBreakerRecordPath(home, conversationID string) string {
	h := sha256.Sum256([]byte(conversationID))
	safeID := hex.EncodeToString(h[:])
	if home == "" {
		return filepath.Join(os.TempDir(), "gentle-ai-circuit-breaker", safeID+".json")
	}
	if info, err := os.Stat(home); err != nil || !info.IsDir() {
		return filepath.Join(os.TempDir(), "gentle-ai-circuit-breaker", safeID+".json")
	}
	return filepath.Join(home, ".gentle-ai", "circuit-breaker", "v1", safeID+".json")
}

func allowPreToolUse(stdout io.Writer) error {
	return writeAntigravityPreToolUseDecision(stdout, "allow", "")
}

func writeAntigravityPreToolUseDecision(stdout io.Writer, decision, reason string) error {
	out := AntigravityCLIHookOutput{Decision: decision, Reason: reason}
	enc, err := json.Marshal(out)
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "{}")
		return nil
	}
	_, _ = fmt.Fprintln(stdout, string(enc))
	return nil
}

func isAntigravityPreToolUseObject(raw []byte) bool {
	if len(bytes.TrimSpace(raw)) == 0 {
		return false
	}
	var object map[string]json.RawMessage
	return json.Unmarshal(raw, &object) == nil && object != nil
}

func rawContainsAntigravityInvokeSubagent(raw []byte) bool {
	return bytes.Contains(raw, []byte(`"invoke_subagent"`))
}

func evaluateAntigravityCLIPreToolUse(raw []byte) (decision, reason string, applies bool) {
	var payload AntigravityCLIHookPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		if rawContainsAntigravityInvokeSubagent(raw) {
			return "deny", antigravityCLIInvokeSubagentMalformedReason, true
		}
		return "", "", false
	}
	return evaluateAntigravityCLIToolCall(payload.ToolCall)
}

func evaluateAntigravityCLIToolCall(toolCall *AntigravityCLIToolCall) (decision, reason string, applies bool) {
	if toolCall == nil || toolCall.Name != antigravityCLIInvokeSubagentTool {
		return "", "", false
	}
	if validationReason := validateAntigravityCLIInvokeSubagentArgs(toolCall.Args); validationReason != "" {
		return "deny", validationReason, true
	}
	return "allow", "", true
}

func validateAntigravityCLIInvokeSubagentArgs(args map[string]any) string {
	if args == nil {
		return antigravityCLIInvokeSubagentMalformedReason
	}
	rawSubagents, hasSubagents := args[antigravityCLISubagentsKey]
	if hasSubagents {
		subagents, ok := rawSubagents.([]any)
		if !ok || len(subagents) == 0 {
			return antigravityCLIInvokeSubagentMalformedReason
		}
		for index, rawSubagent := range subagents {
			subagent, ok := rawSubagent.(map[string]any)
			if !ok {
				return fmt.Sprintf("Antigravity invoke_subagent Subagents[%d] is not an object.", index)
			}
			if reason := validateAntigravityCLISingleSubagentSpec(subagent, fmt.Sprintf("Subagents[%d]", index)); reason != "" {
				return reason
			}
		}
		return ""
	}

	// Flat native AGY format: invoke_subagent(TypeName, Role, Prompt, Workspace?, Model?)
	if _, ok := args[antigravityCLISubagentTypeKey]; ok {
		return validateAntigravityCLISingleSubagentSpec(args, "args")
	}
	if _, ok := args[antigravityCLISubagentRoleKey]; ok {
		return validateAntigravityCLISingleSubagentSpec(args, "args")
	}
	if _, ok := args[antigravityCLISubagentPromptKey]; ok {
		return validateAntigravityCLISingleSubagentSpec(args, "args")
	}

	return antigravityCLIInvokeSubagentMalformedReason
}

func validateAntigravityCLISingleSubagentSpec(subagent map[string]any, label string) string {
	if _, ok := antigravityCLIRequiredString(subagent, antigravityCLISubagentTypeKey); !ok {
		return fmt.Sprintf("Antigravity invoke_subagent %s has no valid TypeName.", label)
	}
	if _, ok := antigravityCLIRequiredString(subagent, antigravityCLISubagentRoleKey); !ok {
		return fmt.Sprintf("Antigravity invoke_subagent %s has no valid Role.", label)
	}
	if _, ok := antigravityCLIRequiredString(subagent, antigravityCLISubagentPromptKey); !ok {
		return fmt.Sprintf("Antigravity invoke_subagent %s has no valid Prompt.", label)
	}
	if _, exists := subagent[antigravityCLISubagentSpaceKey]; exists {
		workspaceName, ok := antigravityCLIRequiredString(subagent, antigravityCLISubagentSpaceKey)
		if !ok || !antigravityCLIValidWorkspace(workspaceName) {
			return fmt.Sprintf("Antigravity invoke_subagent %s has an invalid Workspace.", label)
		}
	}
	return ""
}

func antigravityCLIRequiredString(values map[string]any, key string) (string, bool) {
	value, ok := values[key].(string)
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	return value, value != ""
}

func antigravityCLIValidWorkspace(workspace string) bool {
	switch workspace {
	case "inherit", "branch", "share":
		return true
	default:
		return false
	}
}

// antigravityCLISensitivePathPattern is the sensitive-path deny set the
// settings schema cannot express: the vendor file targets support no globs,
// so these Claude-parity patterns are evaluated on the concrete tool-call
// path at any depth by the PreToolUse guard.
var antigravityCLISensitivePathPattern = regexp.MustCompile(`(?:^|/)\.env(?:\.[^/]+)?$|(?:^|/)\.ssh(?:/|$)|(?:^|/)\.credentials(?:/|$)|(?:^|/)\.aws/credentials$|\.(?:pem|key)$|(?:^|/)\.config/gh/hosts\.yml$|(?:^|/)Library/Keychains(?:/|$)|(?:^|/)secrets(?:/|$)`)

// antigravityCLISensitiveToolPath extracts the concrete path targeted by a
// file or search tool call and reports whether it matches the sensitive
// deny set. run_command is handled by the ask layer below; the permission
// engine and the settings command rules gate the rest of the commands.
func antigravityCLISensitiveToolPath(call *AntigravityCLIToolCall) (string, bool) {
	if call == nil {
		return "", false
	}
	argument := ""
	switch call.Name {
	case "view_file":
		argument = "AbsolutePath"
	case "read_file":
		argument = "Path"
	case "write_to_file", "replace_file_content", "multi_replace_file_content":
		argument = "TargetFile"
	case "write_file", "edit_file":
		argument = "Path"
	case "grep_search":
		argument = "SearchPath"
	default:
		return "", false
	}
	path, _ := antigravityCLIRequiredString(call.Args, argument)
	if path == "" {
		return "", false
	}
	if antigravityCLISensitivePathPattern.MatchString(filepath.ToSlash(path)) {
		return path, true
	}
	return "", false
}

// antigravityCLISensitiveCommandPattern is the run_command variant of the
// sensitive set: command lines are tokenized on whitespace and each token
// is matched like a path, with .aws broadened from the credentials file to
// the whole directory. Shell command lines referencing sensitive paths are
// denied directly to prevent leaking secrets via commands like `cat .env`.
var antigravityCLISensitiveCommandPattern = regexp.MustCompile(`(?:^|/)\.env(?:\.[^/]+)?$|(?:^|/)\.ssh(?:/|$)|(?:^|/)\.credentials(?:/|$)|(?:^|/)\.aws(?:/|$)|\.(?:pem|key)$|(?:^|/)\.config/gh/hosts\.yml$|(?:^|/)Library/Keychains(?:/|$)|(?:^|/)secrets(?:/|$)`)

// antigravityCLIDestructiveCommandPattern identifies destructive shell
// commands that can cause irreversible data loss or unrecoverable git states.
// In Antigravity CLI, these must return "force_ask" so the CLI always prompts
// the operator interactively, even if the base binary is on the allow list.
var antigravityCLIDestructiveCommandPattern = regexp.MustCompile(`(?i)(?:^|[;&|]\s*)\s*(?:(?:sudo\s+)?rm\s+-[a-z0-9]*[rf]|git\s+reset\s+--hard|git\s+push\s+(?:[^\s;&|]+\s+)*--force\b|git\s+push\s+(?:[^\s;&|]+\s+)*-f\b|git\s+clean\s+-[a-z0-9]*f|git\s+branch\s+-[a-z0-9]*D\b|chmod\s+-[a-z0-9]*R\s+777|chown\s+-[a-z0-9]*R|mkfs\b|dd\s+if=)`)

// antigravityCLISensitiveCommandLine reports whether a run_command line
// references a sensitive path. The line is split on whitespace and each
// token is matched like a file path; a match surfaces a deny decision.
func antigravityCLISensitiveCommandLine(call *AntigravityCLIToolCall) (string, bool) {
	if call == nil || call.Name != "run_command" {
		return "", false
	}
	line, _ := antigravityCLIRequiredString(call.Args, "CommandLine")
	if line == "" {
		return "", false
	}
	for _, token := range strings.Fields(line) {
		if antigravityCLISensitiveCommandPattern.MatchString(filepath.ToSlash(token)) {
			return token, true
		}
	}
	return "", false
}

// antigravityCLIDestructiveCommandLine reports whether a run_command line
// matches known destructive patterns (e.g. rm -rf, git reset --hard,
// git push --force).
func antigravityCLIDestructiveCommandLine(call *AntigravityCLIToolCall) (string, bool) {
	if call == nil || call.Name != "run_command" {
		return "", false
	}
	line, _ := antigravityCLIRequiredString(call.Args, "CommandLine")
	if line == "" {
		return "", false
	}
	match := antigravityCLIDestructiveCommandPattern.FindString(line)
	if match != "" {
		return strings.TrimSpace(match), true
	}
	return "", false
}

func handleAntigravityPreToolUse(payload AntigravityCLIHookPayload, _ string, stdout io.Writer) error {
	// Statusline phase (S12): best-effort activity record; it never changes
	// the decision below.
	recordAntigravityStatusLinePhase(payload)
	if decision, reason, applies := evaluateAntigravityCLIToolCall(payload.ToolCall); applies && decision != "allow" {
		return writeAntigravityPreToolUseDecision(stdout, decision, reason)
	}
	// The settings deny lists cannot express these patterns: the vendor file
	// targets support no globs. Like the invoke_subagent denials, the guard
	// returns before the circuit breaker, so denials never write breaker state.
	if path, blocked := antigravityCLISensitiveToolPath(payload.ToolCall); blocked {
		return writeAntigravityPreToolUseDecision(stdout, "deny", "Sensitive file path is blocked by the Gentle AI permissions guard: "+path)
	}
	// Command lines referencing sensitive paths (secrets) are denied directly to
	// prevent bypassing file guards with commands like `cat .env`.
	if token, sensitive := antigravityCLISensitiveCommandLine(payload.ToolCall); sensitive {
		return writeAntigravityPreToolUseDecision(stdout, "deny", "Command line references sensitive path blocked by the Gentle AI permissions guard: "+token)
	}
	// Destructive commands (rm -rf, git reset --hard, etc.) require interactive
	// user confirmation via force_ask, which bypasses cached allow rules.
	if cmd, destructive := antigravityCLIDestructiveCommandLine(payload.ToolCall); destructive {
		return writeAntigravityPreToolUseDecision(stdout, "force_ask", "Destructive command detected ("+cmd+"). Requires explicit confirmation.")
	}
	if payload.ConversationID == "" || payload.ToolCall == nil || payload.ToolCall.Name == "" {
		if payload.ToolCall != nil && payload.ToolCall.Name == "run_command" {
			return writeAntigravityPreToolUseDecision(stdout, "ask", "")
		}
		return allowPreToolUse(stdout)
	}

	canonicalArgs, _ := json.Marshal(payload.ToolCall.Args)
	h := sha256.Sum256(canonicalArgs)
	argsHash := hex.EncodeToString(h[:])

	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}

	recPath := circuitBreakerRecordPath(home, payload.ConversationID)

	var record circuitBreakerRecord
	data, readErr := os.ReadFile(recPath)
	if readErr != nil {
		fallbackPath := filepath.Join(os.TempDir(), "gentle-ai-circuit-breaker", filepath.Base(recPath))
		if fallbackData, ferr := os.ReadFile(fallbackPath); ferr == nil {
			data = fallbackData
			readErr = nil
			recPath = fallbackPath
		}
	}
	if readErr == nil {
		_ = json.Unmarshal(data, &record)
	}

	if record.LastToolName == payload.ToolCall.Name && record.LastArgsHash == argsHash {
		record.ConsecutiveCount++
	} else {
		record.ConsecutiveCount = 1
		record.LastToolName = payload.ToolCall.Name
		record.LastArgsHash = argsHash
	}
	record.UpdatedAt = time.Now().Unix()

	if dirErr := os.MkdirAll(filepath.Dir(recPath), 0o755); dirErr != nil {
		recPath = filepath.Join(os.TempDir(), "gentle-ai-circuit-breaker", filepath.Base(recPath))
		_ = os.MkdirAll(filepath.Dir(recPath), 0o755)
	}
	if enc, err := json.Marshal(record); err == nil {
		if writeErr := os.WriteFile(recPath, enc, 0o644); writeErr != nil {
			fallbackPath := filepath.Join(os.TempDir(), "gentle-ai-circuit-breaker", filepath.Base(recPath))
			if recPath != fallbackPath {
				_ = os.MkdirAll(filepath.Dir(fallbackPath), 0o755)
				_ = os.WriteFile(fallbackPath, enc, 0o644)
			}
		}
	}

	if record.ConsecutiveCount >= maxConsecutiveIdenticalToolCalls {
		out := AntigravityCLIHookOutput{
			Decision: "deny",
			Reason: fmt.Sprintf(
				"Loop Circuit Breaker: Tool %q with identical arguments was called %d times consecutively without intervening actions. Further identical calls are blocked to prevent infinite loops. Use the retrieved information to take action, edit files, or respond to the user.",
				payload.ToolCall.Name,
				record.ConsecutiveCount,
			),
		}
		res, _ := json.Marshal(out)
		_, _ = fmt.Fprintln(stdout, string(res))
		return nil
	}

	if payload.ToolCall.Name == "run_command" {
		return writeAntigravityPreToolUseDecision(stdout, "ask", "")
	}
	return allowPreToolUse(stdout)
}
