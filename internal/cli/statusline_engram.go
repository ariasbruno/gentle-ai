package cli

// Statusline Engram segment (S13): `gentle-ai statusline` shells out to
// `engram stats` in the payload cwd under a 500ms deadline and renders
// `🧠 <project> · <count>` with the current project's observation count. The
// result is cached per cwd at ~/.gentle-ai/statusline-engram-cache.json for
// thirty seconds; every failure (missing binary, timeout, unparsable output)
// hides the segment. The GS widget's transient `✓ saved #N` needs the
// gentle-engram MCP event channel, which a statusline process cannot
// observe (L15).

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

const engramCacheTTL = 30 * time.Second

var (
	engramObservationsPattern = regexp.MustCompile(`Observations:\s+(\d+)`)
	engramProjectPattern      = regexp.MustCompile(`Projects:\s+(\S+)`)
)

type engramCacheRecord struct {
	Cwd          string `json:"cwd"`
	TS           int64  `json:"ts"`
	Project      string `json:"project,omitempty"`
	Observations int    `json:"observations"`
}

type engramProjectState struct {
	Status         string `json:"status"` // e.g. "ready", "✓ saved #3450"
	ConversationID string `json:"conversation_id,omitempty"`
	TS             int64  `json:"ts"`
}

type engramStateFile struct {
	Projects map[string]engramProjectState `json:"projects"`
}

func engramStatePath(home string) string {
	return filepath.Join(home, ".gentle-ai", "statusline-engram-state.json")
}

// statusLineEngramStatus returns the project name and dynamic memory status
// ("ready", "✓ saved #3450", etc.) for the current project and conversation.
// If no state has been saved for this conversation yet, status defaults to "ready".
// An empty project string indicates that engram is not active for the workspace.
func statusLineEngramStatus(home, cwd, conversationID string) (project, status string) {
	if home == "" || cwd == "" {
		return "", ""
	}
	p, obs := statusLineEngramStats(home, cwd)
	if p == "" && obs == 0 {
		return "", ""
	}
	project = p

	state := readEngramProjectStatus(home, project)
	if state.Status != "" {
		if conversationID == "" || state.ConversationID == "" || state.ConversationID == conversationID {
			return project, state.Status
		}
	}
	return project, "ready"
}

func readEngramProjectStatus(home, project string) engramProjectState {
	var file engramStateFile
	raw, err := os.ReadFile(engramStatePath(home))
	if err != nil {
		return engramProjectState{}
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return engramProjectState{}
	}
	if file.Projects == nil {
		return engramProjectState{}
	}
	return file.Projects[project]
}

func writeEngramProjectStatus(home, project, status, conversationID string) {
	if home == "" || project == "" || status == "" {
		return
	}
	var file engramStateFile
	raw, err := os.ReadFile(engramStatePath(home))
	if err == nil {
		_ = json.Unmarshal(raw, &file)
	}
	if file.Projects == nil {
		file.Projects = make(map[string]engramProjectState)
	}
	file.Projects[project] = engramProjectState{
		Status:         status,
		ConversationID: conversationID,
		TS:             time.Now().Unix(),
	}
	encoded, err := json.Marshal(file)
	if err != nil {
		return
	}
	path := engramStatePath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, encoded, 0o644); err != nil {
		return
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
	}
}

func engramCachePath(home string) string {
	return filepath.Join(home, ".gentle-ai", "statusline-engram-cache.json")
}

// statusLineEngramStats returns the Engram project name and observation count
// for the cwd, served from the cache when fresh; empty project and zero
// observations on any failure.
func statusLineEngramStats(home, cwd string) (project string, observations int) {
	if home == "" || cwd == "" {
		return "", 0
	}
	if cached, ok := readEngramCache(home, cwd); ok {
		return cached.Project, cached.Observations
	}
	project, observations = runEngramStats(cwd)
	writeEngramCache(home, cwd, project, observations)
	return project, observations
}

// readEngramCache returns the cached record for the exact cwd, or ok=false
// when the cache is absent, stale, or for another directory.
func readEngramCache(home, cwd string) (engramCacheRecord, bool) {
	var cached engramCacheRecord
	raw, err := os.ReadFile(engramCachePath(home))
	if err != nil {
		return cached, false
	}
	if err := json.Unmarshal(raw, &cached); err != nil {
		return cached, false
	}
	if cached.Cwd != cwd || time.Since(time.Unix(cached.TS, 0)) > engramCacheTTL {
		return cached, false
	}
	return cached, true
}

// writeEngramCache stores the record atomically; failures are ignored — a
// lost cache only costs one extra engram run.
func writeEngramCache(home, cwd, project string, observations int) {
	record := engramCacheRecord{Cwd: cwd, TS: time.Now().Unix(), Project: project, Observations: observations}
	encoded, err := json.Marshal(record)
	if err != nil {
		return
	}
	path := engramCachePath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, encoded, 0o644); err != nil {
		return
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
	}
}

// runEngramStats executes `engram stats` in the cwd under a 500ms deadline
// and extracts the project name and Observations count; any failure returns
// zero values.
func runEngramStats(cwd string) (project string, observations int) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	command := exec.CommandContext(ctx, "engram", "stats")
	command.Dir = cwd
	raw, err := command.Output()
	if err != nil {
		return "", 0
	}
	output := string(raw)
	if match := engramProjectPattern.FindStringSubmatch(output); match != nil {
		project = match[1]
	}
	if match := engramObservationsPattern.FindStringSubmatch(output); match != nil {
		if parsed, convErr := strconv.Atoi(match[1]); convErr == nil {
			observations = parsed
		}
	}
	return project, observations
}
