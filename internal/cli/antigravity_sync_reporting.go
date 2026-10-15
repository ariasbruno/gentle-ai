package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gentleman-programming/gentle-ai/v4/internal/agents/antigravitycli"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
)

// antigravitySyncReportPaths discovers regular plugin files for the final-byte
// report, independently of component selection and rollback ownership. Calling
// it before and after deployment covers deleted and newly created resources.
// Scope-relative ancestors are checked before walking; descendant links are
// skipped. The caller-provided home/workspace directory is the trusted anchor.
func antigravitySyncReportPaths(homeDir, workspaceDir string, scope InstallScope, agents []model.AgentID) ([]string, error) {
	if !containsAgent(agents, model.AgentAntigravityCLI) {
		return nil, nil
	}
	adapter := antigravitycli.NewAdapter()
	base, root := homeDir, adapter.GlobalPluginDir(homeDir)
	if scope == ScopeWorkspace {
		base, root = workspaceDir, adapter.WorkspacePluginDir(workspaceDir)
	}
	relative, err := filepath.Rel(base, root)
	if err != nil {
		return nil, err
	}
	current := base
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("refuse linked Antigravity plugin path: %s", current) // refusal:by-design world-action: acting through a linked path could write or delete outside the trusted plugin anchor; only the operator can remove the link and restore the real directory
		}
	}
	var paths []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if path == root && os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}
