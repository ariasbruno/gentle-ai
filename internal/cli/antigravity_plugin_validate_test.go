package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	antigravitycliagent "github.com/gentleman-programming/gentle-ai/v4/internal/agents/antigravitycli"
)

// antigravityRuleByteCap is the per-file rule cap agy enforces when it loads a
// plugin rule. The validator in the binary reports it verbatim as:
//
//	the content is cut by N bytes to fit the 24000-byte limit
const antigravityRuleByteCap = 24000

type agyValidationFinding struct {
	Severity string `json:"severity"`
	File     string `json:"file"`
	Message  string `json:"message"`
}

func requireAGYBinary(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("agy"); err != nil {
		t.Skip("agy is not installed; the official plugin validator is unavailable")
	}
}

// validatePluginWithAGY runs the validator shipped with agy over dir and returns
// its findings. Counting bytes in Go only approximates the cap: agy expands
// @[label](path) includes before measuring and truncates on line boundaries, so
// the binary is the only authority for what it accepts. The validator also
// reports rules whose frontmatter has no valid trigger, which byte counts miss.
func validatePluginWithAGY(t *testing.T, home, dir string) []agyValidationFinding {
	t.Helper()
	agy, err := exec.LookPath("agy")
	if err != nil {
		t.Skip("agy is not installed; the official plugin validator is unavailable")
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(agy, "plugin", "validate", dir, "--strict", "--json")
	cmd.Env = append(os.Environ(), "HOME="+home)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// --strict exits non-zero to signal findings; the JSON report on stdout is
	// what the assertions read, so the exit status is deliberately ignored.
	_ = cmd.Run()

	var report struct {
		Findings []agyValidationFinding `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("agy plugin validate --json: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	return report.Findings
}

// installGlobalAntigravityPlugin materializes the plugin exactly as a global
// install leaves it for the user, so validation sees the finalized rules rather
// than the raw bundle.
func installGlobalAntigravityPlugin(t *testing.T) (home, pluginDir string) {
	t.Helper()
	home = t.TempDir()
	step := antigravityCLIPluginImportStep{id: "plugin-validate", homeDir: home, scope: ScopeGlobal}
	if err := step.Run(); err != nil {
		t.Fatalf("install Antigravity CLI plugin: %v", err)
	}
	return home, antigravitycliagent.NewAdapter().GlobalPluginDir(home)
}

// TestInstalledPluginPassesAGYStrictValidation pins the rule limits to agy's own
// behavior instead of byte counts kept in sync by hand: what the installer
// produces must survive `agy plugin validate --strict` with zero findings.
func TestInstalledPluginPassesAGYStrictValidation(t *testing.T) {
	requireAGYBinary(t)
	home, pluginDir := installGlobalAntigravityPlugin(t)
	for _, finding := range validatePluginWithAGY(t, home, pluginDir) {
		t.Errorf("agy finding: [%s] %s: %s", finding.Severity, finding.File, finding.Message)
	}
}

// TestAGYStrictValidationDetectsOversizedRule is the negative control for the
// test above: without it, a validator invocation that silently reported nothing
// would make the zero-findings assertion vacuous.
func TestAGYStrictValidationDetectsOversizedRule(t *testing.T) {
	requireAGYBinary(t)
	home, pluginDir := installGlobalAntigravityPlugin(t)
	corePath := filepath.Join(pluginDir, "rules", "gentle-ai-orchestrator.md")
	info, err := os.Stat(corePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() >= antigravityRuleByteCap {
		t.Fatalf("precondition: installed core is %d bytes, already at or over the %d-byte cap", info.Size(), antigravityRuleByteCap)
	}
	padding := antigravityRuleByteCap - info.Size() + 512
	file, err := os.OpenFile(corePath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(strings.Repeat("padding line\n", int(padding)/len("padding line\n")+1)); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	findings := validatePluginWithAGY(t, home, pluginDir)
	if len(findings) == 0 {
		t.Fatal("validator reported no finding for an oversized rule; the cap assertion in TestInstalledPluginPassesAGYStrictValidation would be vacuous")
	}
	for _, finding := range findings {
		if strings.Contains(finding.Message, "24000") {
			return
		}
	}
	t.Fatalf("no finding mentions the 24000-byte cap: %+v", findings)
}
