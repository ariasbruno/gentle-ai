package app_test

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// This guard keeps the repository's own documentation and task records from
// leaking machine-specific or account-specific identifiers. It covers docs/ and
// odd/, which are prose written by whoever ran the work and are the easiest
// place to paste a local path or an opaque provider identifier by accident.
//
// Rule 1 — no UUID. A relay function id reached a task record that way. UUIDs
// have no legitimate place in prose: when a record needs to cite an opaque
// identifier, cite a short prefix instead.
//
// Rule 2 — absolute home paths are a ratchet, not a ban. Upstream already
// records them in older audit and task documents that this guard does not own,
// so a flat ban would fail on pre-existing content. The count may not grow.
var (
	repoDocsHomePathRe = regexp.MustCompile(`/(?:home|Users)/[A-Za-z0-9._-]+/`)
	repoDocsUUIDRe     = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
)

// repoDocsHomePathBaseline is the number of absolute home path occurrences in
// docs/ and odd/ at the time this guard was added. Lower it whenever the count
// drops; never raise it to accommodate new prose.
const repoDocsHomePathBaseline = 32

var repoDocsScannedRoots = []string{"docs", "odd"}

func repoRootFromSourceFile(t *testing.T) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
}

func TestRepositoryDocsDoNotLeakMachineOrAccountIdentifiers(t *testing.T) {
	root := repoRootFromSourceFile(t)

	var (
		homePathHits []string
		uuidHits     []string
		scanned      int
	)

	for _, scannedRoot := range repoDocsScannedRoots {
		start := filepath.Join(root, scannedRoot)
		err := filepath.WalkDir(start, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if bytes.IndexByte(content, 0) >= 0 {
				// Binary asset under a scanned tree; not prose.
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			rel = filepath.ToSlash(rel)
			scanned++

			for lineNumber, line := range strings.Split(string(content), "\n") {
				location := rel + ":" + strconv.Itoa(lineNumber+1)
				for range repoDocsHomePathRe.FindAllString(line, -1) {
					homePathHits = append(homePathHits, location)
				}
				if repoDocsUUIDRe.MatchString(line) {
					uuidHits = append(uuidHits, location)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", start, err)
		}
	}

	if scanned == 0 {
		t.Fatal("no prose files scanned; the guard scanned nothing and proves nothing")
	}

	if len(uuidHits) != 0 {
		t.Errorf("UUIDs found in repository prose (redact them to a short prefix): %s", strings.Join(uuidHits, ", "))
	}

	if len(homePathHits) > repoDocsHomePathBaseline {
		t.Errorf("absolute home path occurrences grew: got %d, baseline %d\nnew occurrences:\n%s",
			len(homePathHits), repoDocsHomePathBaseline, strings.Join(homePathHits[repoDocsHomePathBaseline:], "\n"))
	}
}
