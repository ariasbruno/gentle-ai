package cli

// Statusline tasklist segment (S17): reads the repository's odd/tasks/*.md,
// picks the most recently modified feature document, and parses its ## Tasks
// section into the rendered fraction. Resolution failures (no repository, no
// odd directory, unparsable document) return a zero Total, which hides the
// segment — the statusline never errors on it.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gentleman-programming/gentle-ai/v4/internal/reviewtransaction"
	"github.com/gentleman-programming/gentle-ai/v4/internal/statusline"
)

const tasklistDeadline = 500 * time.Millisecond

// statusLineTasklist resolves the repository root from the payload cwd and
// returns the task stats of the newest ODD feature document.
func statusLineTasklist(cwd string) statusline.TasklistStats {
	ctx, cancel := context.WithTimeout(context.Background(), tasklistDeadline)
	defer cancel()
	root, err := (reviewtransaction.SnapshotBuilder{Repo: cwd}).ResolveRepositoryRoot(ctx)
	if err != nil || root == "" {
		return statusline.TasklistStats{}
	}
	tasksDir := filepath.Join(root, "odd", "tasks")
	entries, readErr := os.ReadDir(tasksDir)
	if readErr != nil {
		return statusline.TasklistStats{}
	}
	var newestPath string
	var newestMod time.Time
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		if newestPath == "" || info.ModTime().After(newestMod) {
			newestPath = filepath.Join(tasksDir, entry.Name())
			newestMod = info.ModTime()
		}
	}
	if newestPath == "" {
		return statusline.TasklistStats{}
	}
	raw, readErr := os.ReadFile(newestPath)
	if readErr != nil {
		return statusline.TasklistStats{}
	}
	return parseODDTasklist(string(raw))
}

// parseODDTasklist counts the ## Tasks section lines
// (`- T# | specs | route | status | evidence`): total, done, blocked, and the
// first in_progress task — falling back to the first pending one when
// nothing runs. Lines outside the section never count.
func parseODDTasklist(doc string) statusline.TasklistStats {
	type taskLine struct{ id, status string }
	var tasks []taskLine
	inSection := false
	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(line, "## ") {
			inSection = strings.TrimSpace(line) == "## Tasks"
			continue
		}
		if !inSection {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		fields := strings.Split(trimmed, "|")
		if len(fields) < 4 {
			continue
		}
		id := strings.TrimSpace(strings.TrimPrefix(fields[0], "- "))
		status := strings.ToLower(strings.TrimSpace(fields[3]))
		if id == "" || status == "" {
			continue
		}
		tasks = append(tasks, taskLine{id: id, status: status})
	}

	stats := statusline.TasklistStats{}
	for _, task := range tasks {
		stats.Total++
		switch task.status {
		case "done":
			stats.Done++
		case "blocked":
			stats.Blocked++
		}
	}
	for _, task := range tasks {
		if task.status == "in_progress" {
			stats.CurrentID, stats.CurrentStatus = task.id, "in_progress"
			return stats
		}
	}
	for _, task := range tasks {
		if task.status == "pending" {
			stats.CurrentID, stats.CurrentStatus = task.id, "pending"
			break
		}
	}
	return stats
}
