package reviewerprovider

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

const agyAdapterHelperModeArgument = "--agy-adapter-helper-mode="
const agyAdapterHelperOutputPathArgument = "--agy-adapter-helper-output-path="

func TestAntigravityCLIAdapterPassesOnlyRuntimeEnvironment(t *testing.T) {
	t.Setenv("PATH", "/runtime/path")
	t.Setenv("HOME", "/runtime/home")
	t.Setenv("OPENAI_API_KEY", "sentinel-api-key")
	t.Setenv("UNRELATED_ENVIRONMENT", "sentinel-unrelated")
	t.Setenv("GENTLE_ARBITRARY_ENVIRONMENT", "sentinel-gentle")

	environmentPath := filepath.Join(t.TempDir(), "environment")
	var command *exec.Cmd
	adapter := &AntigravityCLIAdapter{
		LookPath: func(string) (string, error) { return "agy", nil },
		commandContext: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			command = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAntigravityCLIAdapterHelperProcess$", "--",
				agyAdapterHelperModeArgument+"environment",
				agyAdapterHelperOutputPathArgument+environmentPath)
			return command
		},
	}
	if _, err := adapter.Review(context.Background(), NewInvocation([]byte("prompt"))); err != nil {
		t.Fatal(err)
	}
	if command.Env == nil {
		t.Fatal("agy command Env is nil; want an explicit allowlist")
	}
	explicit := environmentMap([]byte(strings.Join(command.Env, "\x00")))
	if explicit["PATH"] != "/runtime/path" || explicit["HOME"] != "/runtime/home" {
		t.Errorf("explicit environment = %q, want runtime PATH and HOME", command.Env)
	}
	for _, name := range []string{"OPENAI_API_KEY", "UNRELATED_ENVIRONMENT", "GENTLE_ARBITRARY_ENVIRONMENT"} {
		if _, found := explicit[name]; found {
			t.Errorf("explicit environment leaks %s=%q", name, explicit[name])
		}
	}

	environment, err := os.ReadFile(environmentPath)
	if err != nil {
		t.Fatal(err)
	}
	got := environmentMap(environment)
	for _, name := range []string{"PATH", "HOME"} {
		if got[name] == "" {
			t.Errorf("child environment lacks required %s", name)
		}
	}
	for _, name := range []string{"OPENAI_API_KEY", "UNRELATED_ENVIRONMENT", "GENTLE_ARBITRARY_ENVIRONMENT"} {
		if _, found := got[name]; found {
			t.Errorf("child environment leaks %s=%q", name, got[name])
		}
	}
	for name := range got {
		if name == "PATH" || name == "HOME" || (runtime.GOOS == "windows" && (name == "SYSTEMROOT" || name == "USERPROFILE" || name == "HOMEDRIVE" || name == "HOMEPATH")) {
			continue
		}
		t.Errorf("child environment includes non-runtime variable %s", name)
	}
}

func TestAntigravityCLIAdapterReturnsNoBytesWhenUnavailable(t *testing.T) {
	adapter := &AntigravityCLIAdapter{LookPath: func(string) (string, error) { return "", errors.New("not found") }}
	raw, err := adapter.Review(context.Background(), NewInvocation([]byte("provider prompt")))
	if err == nil || !strings.Contains(err.Error(), "antigravity-cli reviewer transport unavailable") || raw != nil {
		t.Fatalf("Review() = %q, %v; want unavailable transport error and no bytes", raw, err)
	}
}

func TestAntigravityCLIAdapterUsesStdinLockedDownArgumentsAndReturnsUntouchedRawOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the helper process uses POSIX argument handling")
	}
	promptPath := filepath.Join(t.TempDir(), "prompt")
	var commandArguments []string
	var command *exec.Cmd
	adapter := &AntigravityCLIAdapter{
		LookPath: func(string) (string, error) { return "agy", nil },
		commandContext: func(ctx context.Context, _ string, arguments ...string) *exec.Cmd {
			commandArguments = append([]string(nil), arguments...)
			helperArguments := append([]string{"-test.run=^TestAntigravityCLIAdapterHelperProcess$", "--", agyAdapterHelperModeArgument + "success", agyAdapterHelperOutputPathArgument + promptPath}, arguments...)
			command = exec.CommandContext(ctx, os.Args[0], helperArguments...)
			return command
		},
	}
	prompt := []byte("provider prompt\nwith bytes")
	raw, err := adapter.Review(context.Background(), NewInvocation(prompt))
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte("raw\x00agy\xffoutput"); !bytes.Equal(raw, want) {
		t.Fatalf("Review() = %q, want untouched raw bytes %q", raw, want)
	}
	if got, err := os.ReadFile(promptPath); err != nil || !bytes.Equal(got, prompt) {
		t.Fatalf("reviewer stdin = %q, %v; want %q", got, err, prompt)
	}
	if want := []string{"--sandbox", "--output-format", "text"}; !slices.Equal(commandArguments, want) {
		t.Fatalf("agy arguments = %q, want %q", commandArguments, want)
	}
	if command.WaitDelay != agyReviewerWaitDelay {
		t.Fatalf("agy WaitDelay = %v, want %v so a held pipe cannot outlive a context kill", command.WaitDelay, agyReviewerWaitDelay)
	}
}

func TestAntigravityCLIAdapterFailsClosedOnProcessFailureAndEmptyOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the helper process uses POSIX argument handling")
	}
	for name, want := range map[string]string{"fail": "antigravity-cli reviewer transport failed", "empty": "produced no final message"} {
		adapter := &AntigravityCLIAdapter{
			LookPath: func(string) (string, error) { return "agy", nil },
			commandContext: func(ctx context.Context, _ string, arguments ...string) *exec.Cmd {
				helperArguments := append([]string{"-test.run=^TestAntigravityCLIAdapterHelperProcess$", "--", agyAdapterHelperModeArgument + name}, arguments...)
				return exec.CommandContext(ctx, os.Args[0], helperArguments...)
			},
		}
		raw, err := adapter.Review(context.Background(), NewInvocation([]byte("prompt")))
		if err == nil || !strings.Contains(err.Error(), want) || raw != nil {
			t.Fatalf("Review(%s) = %q, %v; want %q", name, raw, err, want)
		}
	}
}

// TestAntigravityCLIAdapterHelperProcess is the fake agy binary. It exits
// explicitly so a helper run never prints Go test PASS noise into the
// captured raw stream.
func TestAntigravityCLIAdapterHelperProcess(t *testing.T) {
	mode := agyAdapterHelperOption(agyAdapterHelperModeArgument)
	if mode == "" {
		return
	}
	if outputPath := agyAdapterHelperOption(agyAdapterHelperOutputPathArgument); outputPath != "" {
		var output []byte
		if mode == "arguments" {
			output = []byte(strings.Join(os.Args, "\x00"))
		} else if mode == "environment" {
			output = []byte(strings.Join(os.Environ(), "\x00"))
		} else {
			var err error
			output, err = io.ReadAll(os.Stdin)
			if err != nil {
				os.Exit(1)
			}
		}
		if err := os.WriteFile(outputPath, output, 0o600); err != nil {
			os.Exit(1)
		}
	}
	switch mode {
	case "fail":
		os.Exit(3)
	case "empty":
		os.Exit(0)
	case "stdout-failure":
		_, _ = os.Stdout.WriteString("Not logged in\nsecond line\n")
		os.Exit(1)
	}
	if _, err := os.Stdout.WriteString("raw\x00agy\xffoutput"); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestAntigravityCLIAdapterRoutingArguments(t *testing.T) {
	for _, route := range []struct{ model string }{{""}, {"provider/model"}, {"other/model"}} {
		t.Run(route.model, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "argv")
			adapter := &AntigravityCLIAdapter{Model: route.model,
				LookPath: func(string) (string, error) { return "agy", nil },
				commandContext: func(ctx context.Context, _ string, args ...string) *exec.Cmd {
					return exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestAntigravityCLIAdapterHelperProcess$", "--", agyAdapterHelperModeArgument + "arguments", agyAdapterHelperOutputPathArgument + path}, args...)...)
				},
			}
			if _, err := adapter.Review(t.Context(), NewInvocation([]byte("opaque"))); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got := strings.Split(string(raw), "\x00")[5:]
			want := []string{"--sandbox", "--output-format", "text"}
			if route.model != "" {
				want = append(want, "--model", route.model)
			}
			if !slices.Equal(got, want) {
				t.Fatalf("child argv = %q, want %q", got, want)
			}
		})
	}
}

func agyAdapterHelperOption(prefix string) string {
	for _, argument := range os.Args {
		if strings.HasPrefix(argument, prefix) {
			return strings.TrimPrefix(argument, prefix)
		}
	}
	return ""
}

// A agy child that prints its reason to stdout and exits non-zero must
// surface that reason instead of an empty tail after the exit status.
func TestAntigravityCLIAdapterFailureNamesStdoutReasonWhenStderrIsEmpty(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the helper process uses POSIX argument handling")
	}
	adapter := &AntigravityCLIAdapter{
		LookPath: func(string) (string, error) { return "agy", nil },
		commandContext: func(ctx context.Context, _ string, arguments ...string) *exec.Cmd {
			helperArguments := append([]string{"-test.run=^TestAntigravityCLIAdapterHelperProcess$", "--", agyAdapterHelperModeArgument + "stdout-failure"}, arguments...)
			return exec.CommandContext(ctx, os.Args[0], helperArguments...)
		},
	}
	raw, err := adapter.Review(context.Background(), NewInvocation([]byte("prompt")))
	if raw != nil || err == nil || !strings.Contains(err.Error(), "antigravity-cli reviewer transport failed") ||
		!strings.Contains(err.Error(), "Not logged in") || strings.Contains(err.Error(), "\n") {
		t.Fatalf("Review() = %q, %v; want a single-line transport failure naming the stdout reason", raw, err)
	}
}

func environmentMap(environment []byte) map[string]string {
	values := make(map[string]string)
	for _, entry := range strings.Split(string(environment), "\x00") {
		name, value, found := strings.Cut(entry, "=")
		if found {
			values[name] = value
		}
	}
	return values
}
