package reviewerprovider

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// prepareAntigravityCLIProfile separates runtime authentication from user
// instructions, settings, plugins and MCP. Provider material stays on stdin.
func prepareAntigravityCLIProfile(root string) (home, workspace string, err error) {
	sourceHome, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	sourceDir := sourceHome
	for _, name := range []string{".gemini", "antigravity-cli"} {
		sourceDir = filepath.Join(sourceDir, name)
		info, err := os.Lstat(sourceDir)
		if err != nil {
			return "", "", fmt.Errorf("inspect authentication directory: %w", err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", "", fmt.Errorf("authentication directory must be a regular directory, not a symlink")
		}
	}
	home, workspace = filepath.Join(root, "profile"), filepath.Join(root, "workspace")
	authDir := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(authDir, 0700); err != nil {
		return "", "", err
	}
	if err := os.Mkdir(workspace, 0700); err != nil {
		return "", "", err
	}
	if err := copyAntigravityCLIAuthFile(sourceDir, authDir, "antigravity-oauth-token", false); err != nil {
		return "", "", err
	}
	if err := copyAntigravityCLIAuthFile(sourceDir, authDir, "installation_id", true); err != nil {
		return "", "", err
	}
	return home, workspace, nil
}

func copyAntigravityCLIAuthFile(sourceDir, destinationDir, name string, optional bool) error {
	path := filepath.Join(sourceDir, name)
	info, err := os.Lstat(path)
	if optional && os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect authentication file %s: %w", name, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("authentication file %s must be regular, not a symlink", name)
	}
	if !optional && info.Size() == 0 {
		return fmt.Errorf("authentication file %s is empty", name)
	}
	source, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open authentication file %s: %w", name, err)
	}
	defer source.Close()
	opened, err := source.Stat()
	if err != nil {
		return err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return fmt.Errorf("authentication file %s changed during preparation", name)
	}
	destination, err := os.OpenFile(filepath.Join(destinationDir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil {
		return fmt.Errorf("copy authentication file %s: %w", name, copyErr)
	}
	return closeErr
}
