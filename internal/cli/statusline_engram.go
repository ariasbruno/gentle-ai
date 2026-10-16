package cli

// Statusline Engram segment (S13): `gentle-ai statusline` shells out to
// `engram stats` in the payload cwd under a 500ms deadline and renders the
// current project's observation count as `mem N`. The result is cached per
// cwd at ~/.gentle-ai/statusline-engram-cache.json for thirty seconds; every
// failure (missing binary, timeout, unparsable output) hides the segment.

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

const (
	engramCacheTTL     = 30 * time.Second
	engramCommandNamer = "engram"
)

var engramObservationsPattern = regexp.MustCompile(`Observations:\s+(\d+)`)

type engramCacheRecord struct {
	Cwd          string `json:"cwd"`
	TS           int64  `json:"ts"`
	Observations int    `json:"observations"`
}

func engramCachePath(home string) string {
	return filepath.Join(home, ".gentle-ai", "statusline-engram-cache.json")
}

// statusLineEngramObservations returns the Engram observation count for the
// cwd, served from the cache when fresh; zero on any failure.
func statusLineEngramObservations(home, cwd string) int {
	if home == "" || cwd == "" {
		return 0
	}
	if cached := readEngramCache(home, cwd); cached >= 0 {
		return cached
	}
	observations := runEngramStats(cwd)
	writeEngramCache(home, cwd, observations)
	return observations
}

// readEngramCache returns the cached count for the exact cwd, or -1 when the
// cache is absent, stale, or for another directory.
func readEngramCache(home, cwd string) int {
	raw, err := os.ReadFile(engramCachePath(home))
	if err != nil {
		return -1
	}
	var record engramCacheRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return -1
	}
	if record.Cwd != cwd || time.Since(time.Unix(record.TS, 0)) > engramCacheTTL {
		return -1
	}
	return record.Observations
}

// writeEngramCache stores the count atomically; failures are ignored — a
// lost cache only costs one extra engram run.
func writeEngramCache(home, cwd string, observations int) {
	record := engramCacheRecord{Cwd: cwd, TS: time.Now().Unix(), Observations: observations}
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
// and extracts the Observations count; any failure returns zero.
func runEngramStats(cwd string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	command := exec.CommandContext(ctx, "engram", "stats")
	command.Dir = cwd
	raw, err := command.Output()
	if err != nil {
		return 0
	}
	match := engramObservationsPattern.FindStringSubmatch(string(raw))
	if match == nil {
		return 0
	}
	observations, convErr := strconv.Atoi(match[1])
	if convErr != nil {
		return 0
	}
	return observations
}
