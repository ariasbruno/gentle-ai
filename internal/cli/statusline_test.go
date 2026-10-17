package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// runStatusLineTest invokes the command with a fresh HOME so the in-process
// RDD read resolves against empty user state (defaults on) and returns the
// rendered stdout.

func stripStatusLineANSI(line string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(line, "")
}

func runStatusLineTest(t *testing.T, args []string, stdinPayload, homeDir string) string {
	t.Helper()
	t.Setenv("HOME", homeDir)
	var stdout bytes.Buffer
	if err := runStatusLine(args, strings.NewReader(stdinPayload), &stdout, homeDir); err != nil {
		t.Fatalf("runStatusLine error: %v", err)
	}
	return strings.TrimSuffix(stdout.String(), "\n")
}

// TestRunStatusLineRendersPayload pins the end-to-end command: the payload
// piped on stdin renders with every visible segment, including the
// in-process RDD read resolving to the default-on switch under a fresh HOME.
func TestRunStatusLineRendersPayload(t *testing.T) {
	home := t.TempDir()
	payload := `{
		"agent_state": "working",
		"cwd": "` + home + `/project",
		"conversation_title": "fix login",
		"model": {"display_name": "Gemini 3.5 Pro (High)"},
		"context_window": {"used_percentage": 42.5},
		"cost": 0.312,
		"plan_tier": "Pro",
		"vcs": {"branch": "feat/login", "dirty": true},
		"task_count": 2,
		"terminal_width": 220,
		"execution_mode": "planning"
	}`
	line := runStatusLineTest(t, nil, payload, home)
	for _, want := range []string{
		"⟡", "planning · Gemini 3.5 Pro (High)",
		"ctx", "43%", "$0.312", "⧗ 2 jobs", "🌹 on",
	} {
		if !strings.Contains(stripStatusLineANSI(line), want) {
			t.Errorf("missing segment %q in: %s", want, line)
		}
	}
	if plain := stripStatusLineANSI(line); strings.Contains(plain, "feat/login") || strings.Contains(plain, "fix login") || strings.Contains(plain, "…") {
		t.Errorf("removed segments rendered: %s", line)
	}
}

// TestRunStatusLineTitleMode pins the --title flag end-to-end.
func TestRunStatusLineTitleMode(t *testing.T) {
	home := t.TempDir()
	payload := `{"vcs": {"branch": "feat/x"}, "model": {"display_name": "Gemini 3.5 Pro"}}`
	got := runStatusLineTest(t, []string{"--title"}, payload, home)
	if got != "✿ feat/x · Gemini 3.5 Pro" {
		t.Errorf("title mode output = %q", got)
	}
}

// TestRunStatusLineFallback pins the graceful degradation for absent, empty,
// and unparsable payloads: brand-only line, never an error.
func TestRunStatusLineFallback(t *testing.T) {
	home := t.TempDir()
	for name, payload := range map[string]string{
		"garbage":    "{ not json",
		"empty":      "",
		"not-object": `"just a string"`,
	} {
		got := runStatusLineTest(t, nil, payload, home)
		if got != "✿" {
			t.Errorf("%s payload: output = %q, want the petal-only fallback", name, got)
		}
	}
}

// TestRunStatusLineTitleFallback pins the title mode degradation.
func TestRunStatusLineTitleFallback(t *testing.T) {
	home := t.TempDir()
	if got := runStatusLineTest(t, []string{"--title"}, "{ not json", home); got != "✿" {
		t.Errorf("title fallback = %q, want ✿", got)
	}
}

// TestRunStatusLineConfigHidesRDD pins the user-owned config file driving
// segment visibility through the full command path.
func TestRunStatusLineConfigHidesRDD(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".gentle-ai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := `{"schema": "gentle-ai.statusline/v1", "visibility": {"rdd": false}}`
	if err := os.WriteFile(filepath.Join(configDir, "statusline.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	payload := `{"cwd": "` + home + `", "model": {"display_name": "M"}, "context_window": {"used_percentage": 10}, "terminal_width": 200}`
	line := runStatusLineTest(t, nil, payload, home)
	if strings.Contains(line, "🌹") {
		t.Errorf("rdd segment shown while disabled by config: %s", line)
	}
}

// TestRunStatusLineChangesSegment pins the opt-in git changes segment against
// a real repository with staged changes.
func TestRunStatusLineChangesSegment(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".gentle-ai")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := `{"schema": "gentle-ai.statusline/v1", "visibility": {"changes": true}}`
	if err := os.WriteFile(filepath.Join(configDir, "statusline.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := t.TempDir()
	if out, err := runGitCommand(repo, "init"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(repo, "file.txt"), []byte("first line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runGitCommand(repo, "add", "file.txt"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := runGitCommand(repo, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-m", "init"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(repo, "file.txt"), []byte("first line\nsecond line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	payload := `{"cwd": "` + repo + `", "model": {"display_name": "M"}, "context_window": {"used_percentage": 10}, "terminal_width": 220}`
	line := runStatusLineTest(t, nil, payload, home)
	if !strings.Contains(line, "±") {
		t.Errorf("changes segment missing for a dirty repo: %s", line)
	}
	if !strings.Contains(line, "+1/") {
		t.Errorf("unstaged line addition not counted: %s", line)
	}

	// A clean repository hides the segment.
	cleanRepo := t.TempDir()
	if out, err := runGitCommand(cleanRepo, "init"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	payload = `{"cwd": "` + cleanRepo + `", "model": {"display_name": "M"}, "context_window": {"used_percentage": 10}, "terminal_width": 220}`
	line = runStatusLineTest(t, nil, payload, home)
	if strings.Contains(line, "±") {
		t.Errorf("changes segment shown for a clean repo: %s", line)
	}
}

func runGitCommand(dir string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = dir
	command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := command.CombinedOutput()
	return string(out), err
}

// TestRunStatusLineDumpsPayload pins the L23 instrumentation: the raw live
// payload lands at ~/.gentle-ai/statusline-last-payload.json (0600) without
// affecting the rendered line.
func TestRunStatusLineDumpsPayload(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	payload := `{"cwd": "/home/user/project", "terminal_width": 200}`
	line := runStatusLineTest(t, nil, payload, home)
	if !strings.Contains(line, "✿") {
		t.Fatalf("dump broke the render: %s", line)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".gentle-ai", "statusline-last-payload.json"))
	if err != nil {
		t.Fatalf("payload dump missing: %v", err)
	}
	if string(raw) != payload {
		t.Errorf("dump = %q, want %q", raw, payload)
	}
	info, err := os.Stat(filepath.Join(home, ".gentle-ai", "statusline-last-payload.json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("dump permissions wrong: %v %v", info.Mode().Perm(), err)
	}
}
