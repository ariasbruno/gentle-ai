package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	dumpStatusLinePayload(homeDir, raw)
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
		extras.Changes = statusLineGitChangesCached(homeDir, cwd)
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

// The changes cache (S18): the git subprocess runs at most once per cwd per
// five seconds; a lost cache only costs one extra git run.

const changesCacheTTL = 5 * time.Second

type changesCacheRecord struct {
	Cwd     string `json:"cwd"`
	TS      int64  `json:"ts"`
	Changes string `json:"changes"`
}

func changesCachePath(home string) string {
	return filepath.Join(home, ".gentle-ai", "statusline-changes-cache.json")
}

// statusLineGitChangesCached serves the git summary from the cache when
// fresh; otherwise it recomputes and stores it (the clean-empty result
// caches too).
func statusLineGitChangesCached(home, cwd string) string {
	if home == "" || cwd == "" {
		return ""
	}
	raw, err := os.ReadFile(changesCachePath(home))
	if err == nil {
		var record changesCacheRecord
		if json.Unmarshal(raw, &record) == nil &&
			record.Cwd == cwd && time.Since(time.Unix(record.TS, 0)) <= changesCacheTTL {
			return record.Changes
		}
	}
	changes := statusLineGitChanges(cwd)
	record := changesCacheRecord{Cwd: cwd, TS: time.Now().Unix(), Changes: changes}
	if encoded, err := json.Marshal(record); err == nil {
		path := changesCachePath(home)
		if os.MkdirAll(filepath.Dir(path), 0o755) == nil {
			temporary := path + ".tmp"
			if os.WriteFile(temporary, encoded, 0o644) == nil {
				_ = os.Rename(temporary, path)
			}
		}
	}
	return changes
}

// dumpStatusLinePayload keeps the last live payload (best-effort): agy's
// real schema is the ground truth for display decisions — e.g. whether the
// current version populates execution_mode — and this lets the exact bytes
// agy sends be inspected after the fact. Failures are ignored; the dump
// never affects the rendered line. 0600: the payload can carry the user's
// email.
func dumpStatusLinePayload(home string, raw []byte) {
	if home == "" || len(raw) == 0 {
		return
	}
	path := filepath.Join(home, ".gentle-ai", "statusline-last-payload.json")
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	_ = os.WriteFile(path, raw, 0o600)
}
