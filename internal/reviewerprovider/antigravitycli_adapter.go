package reviewerprovider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// agyReviewerWaitDelay forcibly releases Wait shortly after a context kill so
// a grandchild holding the inherited stdout pipe cannot outlive the deadline.
const agyReviewerWaitDelay = 5 * time.Second

// AntigravityCLIAdapter invokes a brand-new agy process with an opaque provider
// invocation and returns its raw final bytes without interpreting them.
type AntigravityCLIAdapter struct {
	Model          string
	Effort         string // optional --effort tier (low|medium|high|xhigh|max); empty or "inherit" omits the flag
	LookPath       func(string) (string, error)
	commandContext func(context.Context, string, ...string) *exec.Cmd
	Env            []string // test extras; protected runtime/profile locators take precedence
}

// NewAntigravityCLIAdapter returns an adapter using the agy binary resolved
// from PATH.
func NewAntigravityCLIAdapter() *AntigravityCLIAdapter {
	return &AntigravityCLIAdapter{LookPath: exec.LookPath, commandContext: exec.CommandContext}
}

// Review runs agy under its OS-level sandbox in an empty temporary directory.
// The prompt is delivered through stdin so command arguments never carry
// provider material, and agy is NOT given a text prompt flag: omitting
// -p/--print keeps headless agy from consuming the remaining flags as prompt
// text once it detects the piped input (--input-format defaults to text and
// reads prompts on stdin).
func (adapter *AntigravityCLIAdapter) Review(ctx context.Context, invocation Invocation) (raw []byte, resultErr error) {
	model := strings.TrimSpace(adapter.Model)
	lookPath := adapter.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	binary, err := lookPath("agy")
	if err != nil {
		return nil, fmt.Errorf("antigravity-cli reviewer transport unavailable: %w", err)
	}
	scratch, err := os.MkdirTemp("", "gentle-ai-antigravitycli-reviewer-*")
	if err != nil {
		return nil, fmt.Errorf("antigravity-cli reviewer transport unavailable: create scratch directory: %w", err)
	}
	defer func() {
		credentialsDestroyed := destroyAntigravityReviewerCredentials(scratch)
		if err := os.RemoveAll(scratch); err != nil {
			raw = nil
			warning := ""
			if !credentialsDestroyed {
				warning = fmt.Sprintf("; isolated credentials may remain on disk under %s", scratch)
			}
			resultErr = errors.Join(resultErr, fmt.Errorf("antigravity-cli reviewer transport failed: remove temporary profile: %w%s", err, warning))
		}
	}()
	home, workspace, err := prepareAntigravityCLIProfile(scratch)
	if err != nil {
		return nil, fmt.Errorf("antigravity-cli reviewer transport unavailable: prepare isolated authentication profile: %w", err)
	}
	commandContext := adapter.commandContext
	if commandContext == nil {
		commandContext = exec.CommandContext
	}
	// Inherit default: an empty or "inherit" model omits --model so the
	// isolated reviewer process resolves AGY's own default; the disposable
	// profile carries no model configuration from the habitual profile.
	arguments := []string{"--sandbox", "--output-format", "text"}
	if model != "" && !strings.EqualFold(model, "inherit") {
		arguments = append(arguments, "--model", model)
	}
	if effort := strings.TrimSpace(adapter.Effort); effort != "" && !strings.EqualFold(effort, "inherit") {
		arguments = append(arguments, "--effort", effort)
	}
	command := commandContext(ctx, binary, arguments...)
	command.Dir = workspace
	// Protected locators come last: test extras cannot restore the original HOME.
	command.Env = append(append([]string(nil), adapter.Env...), antigravityCLIRuntimeEnvironment(home)...)
	command.WaitDelay = agyReviewerWaitDelay
	command.Stdin = bytes.NewReader(invocation.Prompt())
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		// A deadline kill reads as the typed context error, not "signal: killed".
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		return nil, fmt.Errorf("antigravity-cli reviewer transport failed: %w: %s", err, reviewerTransportFailureDetail(stderr.String(), stdout.String()))
	}
	if len(bytes.TrimSpace(stdout.Bytes())) == 0 {
		return nil, errors.New("antigravity-cli reviewer transport produced no final message")
	}
	return stdout.Bytes(), nil
}

// antigravityCLIRuntimeEnvironment supplies only launch/profile locators.
// Every home locator points to the disposable auth-only profile.
func antigravityCLIRuntimeEnvironment(home string) []string {
	environment := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + home,
	}
	if runtime.GOOS == "windows" {
		environment = appendAntigravityCLIRuntimeEnvironmentValue(environment, "SYSTEMROOT")
		environment = append(environment, "USERPROFILE="+home, "HOMEDRIVE="+filepath.VolumeName(home), "HOMEPATH="+strings.TrimPrefix(home, filepath.VolumeName(home)))
	}
	return environment
}

func appendAntigravityCLIRuntimeEnvironmentValue(environment []string, name string) []string {
	if value := os.Getenv(name); value != "" {
		return append(environment, name+"="+value)
	}
	return environment
}

// destroyAntigravityReviewerCredentials removes the disposable credential
// copies best-effort before the scratch tree is removed, so a tree-wide
// removal failure cannot leave the OAuth token behind when its path is still
// reachable. It reports whether every credential file was confirmed removed;
// a false report is surfaced to the operator as a persistence warning. The
// two removals stay unrolled: the adapter minimality guard forbids loops.
func destroyAntigravityReviewerCredentials(scratch string) bool {
	credentials := filepath.Join(scratch, "profile", ".gemini", "antigravity-cli")
	tokenErr := os.Remove(filepath.Join(credentials, "antigravity-oauth-token"))
	installationErr := os.Remove(filepath.Join(credentials, "installation_id"))
	confirmed := func(err error) bool { return err == nil || os.IsNotExist(err) }
	return confirmed(tokenErr) && confirmed(installationErr)
}
