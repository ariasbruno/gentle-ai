package reviewerprovider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// agyReviewerWaitDelay forcibly releases Wait shortly after a context kill so
// a grandchild holding the inherited stdout pipe cannot outlive the deadline.
const agyReviewerWaitDelay = 5 * time.Second

// AntigravityCLIAdapter invokes a brand-new agy process with an opaque provider
// invocation and returns its raw final bytes without interpreting them.
type AntigravityCLIAdapter struct {
	Model          string
	LookPath       func(string) (string, error)
	commandContext func(context.Context, string, ...string) *exec.Cmd
	Env            []string // extra environment entries appended to the minimal runtime environment (used by tests to inject fake-binary expectations)
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
func (adapter *AntigravityCLIAdapter) Review(ctx context.Context, invocation Invocation) ([]byte, error) {
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
	defer os.RemoveAll(scratch)
	commandContext := adapter.commandContext
	if commandContext == nil {
		commandContext = exec.CommandContext
	}
	arguments := []string{"--sandbox", "--output-format", "text"}
	if adapter.Model != "" {
		arguments = append(arguments, "--model", adapter.Model)
	}
	command := commandContext(ctx, binary, arguments...)
	command.Dir = scratch
	command.Env = append(antigravityCLIRuntimeEnvironment(), adapter.Env...)
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

// antigravityCLIRuntimeEnvironment passes only the process locators agy needs
// to launch and resolve its local configuration. Credentials remain in
// ~/.gemini/antigravity-cli, outside every allowlisted variable.
func antigravityCLIRuntimeEnvironment() []string {
	environment := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
	}
	if runtime.GOOS == "windows" {
		environment = appendAntigravityCLIRuntimeEnvironmentValue(environment, "SYSTEMROOT")
		environment = appendAntigravityCLIRuntimeEnvironmentValue(environment, "USERPROFILE")
		environment = appendAntigravityCLIRuntimeEnvironmentValue(environment, "HOMEDRIVE")
		environment = appendAntigravityCLIRuntimeEnvironmentValue(environment, "HOMEPATH")
	}
	return environment
}

func appendAntigravityCLIRuntimeEnvironmentValue(environment []string, name string) []string {
	if value := os.Getenv(name); value != "" {
		return append(environment, name+"="+value)
	}
	return environment
}
