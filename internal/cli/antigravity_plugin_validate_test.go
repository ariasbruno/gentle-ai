package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	antigravitycliagent "github.com/gentleman-programming/gentle-ai/v4/internal/agents/antigravitycli"
)

const (
	// antigravityRuleByteCap is the per-file rule cap agy enforces when it loads
	// a plugin rule. The validator in the binary reports it verbatim as:
	//
	//	the content is cut by N bytes to fit the 24000-byte limit
	antigravityRuleByteCap = 24000
	// antigravityValidatedVersion is the agy release this integration declares
	// and the oldest one whose validator both supports --json and enforces the
	// rule byte cap. agy 1.3.1 ignores --json, prints a human report, and
	// returns "[ok]" for an oversized rule, so validating with it would make the
	// cap assertion below vacuous.
	antigravityValidatedVersion = "1.4.3"
	// antigravityRequireAGYEnv turns an unavailable or too old validator into a
	// failure instead of a skip, so a job that claims to validate can prove the
	// validator actually ran.
	antigravityRequireAGYEnv = "GENTLE_AI_REQUIRE_AGY"
)

type agyValidationFinding struct {
	Severity string `json:"severity"`
	File     string `json:"file"`
	Message  string `json:"message"`
}

// agyUnavailable reports a validator the tests cannot trust. It skips by
// default, because a developer without agy must still be able to run the suite,
// and fails when GENTLE_AI_REQUIRE_AGY=1 asks for proof that the validator ran.
func agyUnavailable(t *testing.T, reason string) {
	t.Helper()
	if strings.TrimSpace(os.Getenv(antigravityRequireAGYEnv)) == "1" {
		t.Fatalf("%s (%s=1 requires the official plugin validator)", reason, antigravityRequireAGYEnv)
	}
	t.Skipf("%s (set %s=1 to require it)", reason, antigravityRequireAGYEnv)
}

// parseAGYVersion reads a dotted numeric version. Anything else is reported as
// unreadable so callers can skip instead of asserting on an unknown validator.
func parseAGYVersion(version string) ([3]int, bool) {
	var parsed [3]int
	fields := strings.Split(strings.TrimPrefix(strings.TrimSpace(version), "v"), ".")
	if len(fields) != len(parsed) {
		return parsed, false
	}
	for i, field := range fields {
		number, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil || number < 0 {
			return parsed, false
		}
		parsed[i] = number
	}
	return parsed, true
}

// agyVersionAtLeast compares dotted numeric versions component by component, so
// 1.10.0 is newer than 1.4.3 instead of sorting below it as a string.
func agyVersionAtLeast(actual, minimum string) bool {
	got, ok := parseAGYVersion(actual)
	if !ok {
		return false
	}
	want, ok := parseAGYVersion(minimum)
	if !ok {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return got[i] > want[i]
		}
	}
	return true
}

// requireAGYBinary returns the agy binary the validator tests must invoke. The
// suite runs from whatever PATH gives it, and an installed agy can be older than
// the declared release, so the version is checked before anything is asserted:
// a validator that cannot see the cap would make the tests pass for the wrong
// reason.
func requireAGYBinary(t *testing.T) string {
	t.Helper()
	agy, err := exec.LookPath("agy")
	if err != nil {
		agyUnavailable(t, "agy is not installed; the official plugin validator is unavailable")
		return ""
	}
	output, err := exec.Command(agy, "--version").Output()
	if err != nil {
		agyUnavailable(t, fmt.Sprintf("%s --version failed: %v", agy, err))
		return ""
	}
	version := strings.TrimSpace(string(output))
	if !agyVersionAtLeast(version, antigravityValidatedVersion) {
		agyUnavailable(t, fmt.Sprintf(
			"agy reports version %q, which is older than the validated %s or unreadable; an older validator ignores --json and does not enforce the %d-byte rule cap",
			version, antigravityValidatedVersion, antigravityRuleByteCap))
		return ""
	}
	return agy
}

// validatePluginWithAGY runs the validator shipped with agy over dir and returns
// its findings. Counting bytes in Go only approximates the cap: agy expands
// @[label](path) includes before measuring and truncates on line boundaries, so
// the binary is the only authority for what it accepts. The validator also
// reports rules whose frontmatter has no valid trigger, which byte counts miss.
func validatePluginWithAGY(t *testing.T, agy, home, dir string) []agyValidationFinding {
	t.Helper()
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
	agy := requireAGYBinary(t)
	home, pluginDir := installGlobalAntigravityPlugin(t)
	for _, finding := range validatePluginWithAGY(t, agy, home, pluginDir) {
		t.Errorf("agy finding: [%s] %s: %s", finding.Severity, finding.File, finding.Message)
	}
}

// TestAGYStrictValidationDetectsOversizedRule is the negative control for the
// test above: without it, a validator invocation that silently reported nothing
// would make the zero-findings assertion vacuous.
func TestAGYStrictValidationDetectsOversizedRule(t *testing.T) {
	agy := requireAGYBinary(t)
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

	findings := validatePluginWithAGY(t, agy, home, pluginDir)
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

// TestAGYVersionAtLeast pins the comparison the version gate depends on. The
// 1.10.0 case is the one a string comparison gets wrong, and the unreadable
// cases must fall to the skip path instead of asserting on an unknown validator.
func TestAGYVersionAtLeast(t *testing.T) {
	cases := []struct {
		actual  string
		minimum string
		want    bool
	}{
		{actual: "1.4.3", minimum: "1.4.3", want: true},
		{actual: "1.4.3\n", minimum: "1.4.3", want: true},
		{actual: "v1.4.3", minimum: "1.4.3", want: true},
		{actual: "1.4.2", minimum: "1.4.3", want: false},
		{actual: "1.3.1", minimum: "1.4.3", want: false},
		{actual: "1.10.0", minimum: "1.4.3", want: true},
		{actual: "2.0.0", minimum: "1.4.3", want: true},
		{actual: "1.4", minimum: "1.4.3", want: false},
		{actual: "", minimum: "1.4.3", want: false},
		{actual: "garbage", minimum: "1.4.3", want: false},
	}
	for _, tc := range cases {
		if got := agyVersionAtLeast(tc.actual, tc.minimum); got != tc.want {
			t.Errorf("agyVersionAtLeast(%q, %q) = %v, want %v", tc.actual, tc.minimum, got, tc.want)
		}
	}
}
