package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gentleman-programming/gentle-ai/v4/internal/statusline"
)

// RunStatusLine renders the agy (Antigravity CLI) status line from the
// payload agy pipes on stdin. Install/sync registers it in
// ~/.gemini/antigravity-cli/settings.json as `gentle-ai statusline` (and
// `gentle-ai statusline --title` for the terminal title). Everything
// resolves in-process: the payload parses with encoding/json — no jq, no
// bash — the RDD review switch reads through the same read-only resolver the
// review command uses, and the optional changes segment shells out to git
// under a 500ms deadline. A payload failure never errors: agy renders
// whatever this prints, so it degrades to the brand-only fallback line.
func RunStatusLine(args []string, stdout io.Writer) error {
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		home = ""
	}
	return runStatusLine(args, os.Stdin, stdout, home)
}

const maxStatuslinePayloadBytes = 1 << 20

func runStatusLine(args []string, stdin io.Reader, stdout io.Writer, homeDir string) error {
	title := false
	cwdOverride := ""
	fs := flag.NewFlagSet("statusline", flag.ContinueOnError)
	fs.BoolVar(&title, "title", false, "render the terminal window title instead of the status line")
	fs.StringVar(&cwdOverride, "cwd", "", "optional working directory override (defaults to the payload cwd)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	raw, _ := io.ReadAll(io.LimitReader(stdin, maxStatuslinePayloadBytes))
	payload, parseErr := statusline.ParsePayload(raw)

	if title {
		if parseErr != nil {
			_, _ = fmt.Fprintln(stdout, statusline.RenderTitle(nil))
			return nil
		}
		_, _ = fmt.Fprintln(stdout, statusline.RenderTitle(payload))
		return nil
	}
	if parseErr != nil {
		// Absent, empty, or unparsable payload: degrade, never error (S6).
		_, _ = fmt.Fprintln(stdout, statusline.FallbackLine)
		return nil
	}

	config := statusline.LoadConfig(homeDir)

	cwd := payload.Cwd
	if cwdOverride != "" {
		cwd = cwdOverride
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	// The PreToolUse hook still records phase files (S12 infra), but the
	// statusline no longer displays a label (L19: bare petal).
	extras := statusline.Extras{}
	if config.Visibility.RDD {
		extras.RDDMode = resolveStatusLineRDD(homeDir, cwd)
	}
	if config.Visibility.Changes {
		extras.Changes = statusLineGitChanges(cwd)
	}
	if config.Visibility.Engram {
		extras.EngramProject, extras.Observations = statusLineEngramStats(homeDir, cwd)
	}
	if config.Visibility.Tasklist {
		extras.Tasklist = statusLineTasklist(cwd)
	}

	_, _ = fmt.Fprintln(stdout, statusline.Render(payload, config, extras))
	return nil
}

// statusLineGitChanges renders the opt-in changes segment "±N +A/−D" from
// git status --porcelain and git diff --numstat under a shared 500ms
// deadline; any failure hides the segment (S5).
func statusLineGitChanges(cwd string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	porcelain, err := exec.CommandContext(ctx, "git", "-C", cwd, "status", "--porcelain").Output()
	if err != nil {
		return ""
	}
	count := 0
	if trimmed := strings.TrimSpace(string(porcelain)); trimmed != "" {
		count = len(strings.Split(trimmed, "\n"))
	}

	added, deleted := 0, 0
	if numstat, err := exec.CommandContext(ctx, "git", "-C", cwd, "diff", "--numstat").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(numstat)), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			if value, convErr := strconv.Atoi(fields[0]); convErr == nil && value > 0 {
				added += value
			}
			if value, convErr := strconv.Atoi(fields[1]); convErr == nil && value > 0 {
				deleted += value
			}
		}
	}
	if count == 0 && added == 0 && deleted == 0 {
		return ""
	}
	return fmt.Sprintf("±%d +%d/−%d", count, added, deleted)
}
