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
	"strings"
	"time"

	"github.com/gentleman-programming/gentle-ai/v3/internal/reviewtransaction"
	"github.com/gentleman-programming/gentle-ai/v3/internal/skillregistry"
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

	return RunAntigravityCLIHook(*event, stdin, stdout, *cwd)
}

// RunAntigravityCLIHook handles the official Antigravity CLI lifecycle events.
func RunAntigravityCLIHook(event string, stdin io.Reader, stdout io.Writer, cwdOverride string) error {
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
		return handleAntigravityStop(payload, workspace, stdout)
	default:
		_, _ = fmt.Fprintln(stdout, "{}")
		return nil
	}
}

// PostToolUse and PostInvocation are transport-only in the portable integration.
func handleAntigravityPostToolUse(_ AntigravityCLIHookPayload, _ string, stdout io.Writer) error {
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

func handleAntigravityStop(payload AntigravityCLIHookPayload, workspace string, stdout io.Writer) error {
	if payload.TerminationReason != "model_stop" || !payload.FullyIdle {
		_, _ = fmt.Fprintln(stdout, "{}")
		return nil
	}

	ctx := context.Background()
	root, targetIdentity, status, ok, err := antigravityResolveTargetIdentity(ctx, workspace)
	if err != nil || !ok {
		_, _ = fmt.Fprintln(stdout, "{}")
		return nil
	}

	// A candidate is guardable when STATUS reports Action=start, meaning there
	// are unreviewed tracked changes. We do not gate on NextTransition here
	// because the workspace may contain untracked files (e.g. .gitignore
	// written by the PreInvocation skillregistry step) that cause STATUS to
	// return collect/intended_untracked_selection_required even when there is
	// a valid candidate — the Action field reflects the candidate state
	// independently of untracked-file resolution.
	if targetIdentity == "" || status.Action != reviewtransaction.TargetStatusActionStart {
		_, _ = fmt.Fprintln(stdout, "{}")
		return nil
	}

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
		if hadRecord && ((record.BaselineTargetIdentity != "" && record.BaselineTargetIdentity == targetIdentity) ||
			(record.TargetIdentity != "" && record.TargetIdentity == targetIdentity)) {
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

	out := AntigravityCLIHookOutput{
		Decision: "continue",
		Reason:   "Receipt-Driven Development: unreviewed candidate changes detected since session baseline. Please run gentle-ai review status preflight before completing.",
	}
	enc, _ := json.Marshal(out)
	_, _ = fmt.Fprintln(stdout, string(enc))

	if payload.ConversationID != "" {
		_, _ = recordReviewStopHookReminder(payload.ConversationID, targetIdentity)
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
	rawSubagents, ok := args[antigravityCLISubagentsKey]
	if !ok {
		return antigravityCLIInvokeSubagentMalformedReason
	}
	subagents, ok := rawSubagents.([]any)
	if !ok || len(subagents) == 0 {
		return antigravityCLIInvokeSubagentMalformedReason
	}
	for index, rawSubagent := range subagents {
		subagent, ok := rawSubagent.(map[string]any)
		if !ok {
			return fmt.Sprintf("Antigravity invoke_subagent Subagents[%d] is not an object.", index)
		}
		if _, ok := antigravityCLIRequiredString(subagent, antigravityCLISubagentTypeKey); !ok {
			return fmt.Sprintf("Antigravity invoke_subagent Subagents[%d] has no valid TypeName.", index)
		}
		if _, ok := antigravityCLIRequiredString(subagent, antigravityCLISubagentRoleKey); !ok {
			return fmt.Sprintf("Antigravity invoke_subagent Subagents[%d] has no valid Role.", index)
		}
		if _, ok := antigravityCLIRequiredString(subagent, antigravityCLISubagentPromptKey); !ok {
			return fmt.Sprintf("Antigravity invoke_subagent Subagents[%d] has no valid Prompt.", index)
		}
		if _, exists := subagent[antigravityCLISubagentSpaceKey]; exists {
			workspaceName, ok := antigravityCLIRequiredString(subagent, antigravityCLISubagentSpaceKey)
			if !ok || !antigravityCLIValidWorkspace(workspaceName) {
				return fmt.Sprintf("Antigravity invoke_subagent Subagents[%d] has an invalid Workspace.", index)
			}
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

func handleAntigravityPreToolUse(payload AntigravityCLIHookPayload, _ string, stdout io.Writer) error {
	if decision, reason, applies := evaluateAntigravityCLIToolCall(payload.ToolCall); applies && decision != "allow" {
		return writeAntigravityPreToolUseDecision(stdout, decision, reason)
	}
	if payload.ConversationID == "" || payload.ToolCall == nil || payload.ToolCall.Name == "" {
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

	return allowPreToolUse(stdout)
}
