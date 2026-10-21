// Package localexclude adds a rule to a repository's local exclude file,
// the per-clone ignore list git keeps outside the worktree.
//
// It is the mechanism the product uses to keep its own runtime state out of
// the worktree without editing the user's .gitignore. A new .gitignore is a
// change to the user's work, and an untracked file makes every review
// candidate unassessable ("untracked files require an explicit
// declaration"), which turns a review reminder into a permanent block.
package localexclude

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// EnsureInWorkTree adds the entry to the worktree's local exclude file when
// it is not there yet.
//
// The root must be the work tree root: an enclosing repository's exclude
// file is never changed, because the caller asked about a specific worktree
// and the change would outlive it.
func EnsureInWorkTree(root, entry, header string) error {
	return ensure(root, entry, header, true)
}

// EnsureForDirectory adds the entry to the local exclude file of the
// repository that contains the root. It is the variant for a rule scoped to
// a directory inside a repository.
func EnsureForDirectory(root, entry, header string) error {
	return ensure(root, entry, header, false)
}

func ensure(root, entry, header string, requireWorkTreeRoot bool) error {
	exclude, err := resolveExcludePath(root, requireWorkTreeRoot)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(exclude)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if EntryPresent(string(content), entry) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
		return err
	}
	// The rule is appended, so a rule another process writes at the same
	// time is kept.
	file, err := os.OpenFile(exclude, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	text := ""
	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		text = "\n"
	}
	if header != "" {
		text += header + "\n"
	}
	text += entry + "\n"
	if _, err := file.WriteString(text); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

// EntryPresent reports whether an ignore file already carries the rule,
// tolerating the spellings git accepts ("x", "x/", "/x" and "/x/").
func EntryPresent(content, entry string) bool {
	wanted := strings.Trim(strings.TrimSpace(entry), "/")
	for _, line := range strings.Split(content, "\n") {
		candidate := strings.Trim(strings.TrimSpace(line), "/")
		if candidate == wanted {
			return true
		}
	}
	return false
}

// resolveExcludePath resolves the local exclude file of the repository that
// contains the root.
func resolveExcludePath(root string, requireWorkTreeRoot bool) (string, error) {
	output, err := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree", "--show-toplevel", "--git-path", "info/exclude").Output()
	if err != nil {
		return "", fmt.Errorf("resolve git exclude file: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 3 || lines[0] != "true" {
		// refusal:by-design world-action: only a warning; the caller continues and there is no worktree whose exclude file could change.
		return "", fmt.Errorf("%q is not a git work tree", root)
	}
	if requireWorkTreeRoot {
		topLevel, err := filepath.EvalSymlinks(lines[1])
		if err != nil || topLevel != root {
			// refusal:by-design world-action: only a warning; the caller continues and an enclosing repository's exclude file must not change.
			return "", fmt.Errorf("%q is not the git work tree root", root)
		}
	}
	exclude := lines[2]
	if !filepath.IsAbs(exclude) {
		exclude = filepath.Join(root, exclude)
	}
	return exclude, nil
}
