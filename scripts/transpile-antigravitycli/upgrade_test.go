package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the public generator against the source shapes required by Shell
// 4.2.0, including preserved native projections from the earlier update, without reading a sibling checkout or writing installed user assets.
func TestShell420PublicTranspiler(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "transpiler")
	if err := exec.Command("go", "build", "-o", binary, ".").Run(); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		prepare func(*testing.T, string)
		check   func(*testing.T, string)
	}{
		{"existing-local-tools", func(t *testing.T, source string) {}, func(t *testing.T, output string) {
			got := readUpgradeFixture(t, output, "agents/test-agent/agent.md")
			for _, want := range []string{"view_file", "list_dir", "Test body."} {
				if !strings.Contains(got, want) {
					t.Errorf("existing local behavior lost %q", want)
				}
			}
		}},
		{"non-markdown-resource-bytes", func(t *testing.T, source string) {
			resource := filepath.Join(source, "skills/test-skill/payload.bin")
			if err := os.WriteFile(resource, []byte{0xff, 0x00, ' ', 0x0d, 0x0a, ' '}, 0o644); err != nil {
				t.Fatal(err)
			}
		}, func(t *testing.T, output string) {
			got, err := os.ReadFile(filepath.Join(output, "skills/test-skill/payload.bin"))
			if err != nil {
				t.Fatal(err)
			}
			want := []byte{0xff, 0x00, ' ', 0x0d, 0x0a, ' '}
			if !bytes.Equal(got, want) {
				t.Errorf("resource bytes changed: got %x want %x", got, want)
			}
		}},
		{"explorer-web", func(t *testing.T, source string) {
			writeUpgradeFixture(t, filepath.Join(source, "assets/agents/test-agent.md"), "---\nname: test-agent\ndescription: Read-only local and web exploration.\ntools:\n  - read\n  - codegraph\n  - web_enable\n  - web_search\n  - source_check\n  - fetch_content\n  - get_search_content\n---\n\nUse `web_enable` if activation is needed; headless sessions may activate the web tools at startup.\nDo not edit, write, run commands, or mutate project state.\n")
		}, func(t *testing.T, output string) {
			got := readUpgradeFixture(t, output, "agents/test-agent/agent.md")
			for _, want := range []string{"view_file", "list_dir", "search_web", "read_url_content", "enable_mcp_tools: true", "Do not edit, write, run commands"} {
				if !strings.Contains(got, want) {
					t.Errorf("explorer lost %q", want)
				}
			}
			for _, forbidden := range []string{"web_enable", "source_check", "fetch_content", "get_search_content", "tools: [run_command"} {
				if strings.Contains(got, forbidden) {
					t.Errorf("explorer retains %q", forbidden)
				}
			}
		}},
		{"risk-based-verification", func(t *testing.T, source string) {
			writeUpgradeFixture(t, filepath.Join(source, "assets/orchestrator-verification.md"), "# Verification\n\n5. **Verification rule**: The bounded writer always self-verifies. Native risk decides independent verification in any RDD state; native review never replaces or skips verification.\n\n| Native risk tier | Verification (any RDD state) |\n| high | independent verifier always |\n")
		}, func(t *testing.T, output string) {
			got := readUpgradeFixture(t, output, "orchestrator-verification.md")
			for _, want := range []string{"writer", "risk", "any RDD state", "never replaces", "independent"} {
				if !strings.Contains(got, want) {
					t.Errorf("verification lost %q", want)
				}
			}
		}},
		{"native-command-waiting", func(t *testing.T, source string) {
			writeUpgradeFixture(t, filepath.Join(source, "assets/orchestrator-delegation.md"), "# Delegation\n\nWaiting on CI, a build, or a server is not delegation: use `bash_background` with the wait inside the command and end the turn. Never `sleep`, poll, or spend a subagent to wait.\n")
		}, func(t *testing.T, output string) {
			got := readUpgradeFixture(t, output, "orchestrator-delegation.md")
			if strings.Contains(got, "bash_background") || !strings.Contains(got, "run_command") || !strings.Contains(got, "not delegation") {
				t.Errorf("non-native waiting policy: %s", got)
			}
		}},
		{"worker-native-phase", func(t *testing.T, source string) {
			writeAgentFixture(t, source, "gentle-ai-worker", "The parent selects context. Never read persistent memory.\n")
		}, func(t *testing.T, output string) {
			got := readUpgradeFixture(t, output, "agents/gentle-ai-worker/agent.md")
			for _, want := range []string{"Before the first implementation edit", "rules/strict-tdd.md", "Do not assume access to parent history", "observed RED", "observed GREEN", "enable_mcp_tools: false"} {
				if !strings.Contains(got, want) {
					t.Errorf("worker lost %q", want)
				}
			}
		}},
		{"verifier-native-phase", func(t *testing.T, source string) {
			writeAgentFixture(t, source, "gentle-ai-verify", "Run exact authorized checks only.\n")
		}, func(t *testing.T, output string) {
			got := readUpgradeFixture(t, output, "agents/gentle-ai-verify/agent.md")
			for _, want := range []string{"Before verification", "rules/strict-tdd-verify.md", "Do not assume access to parent history"} {
				if !strings.Contains(got, want) {
					t.Errorf("verifier lost %q", want)
				}
			}
		}},
		{"no-pi-session-tools", func(t *testing.T, source string) {
			writeUpgradeFixture(t, filepath.Join(source, "assets/orchestrator-delegation.md"), "# Delegation\n\n## Session subject and display identity\nCall `orchestrator_session_id`; use `orchestrator_list.filter`.\n\n## Pi Runtime Overlays\nPi-specific setup.\n\n## Native delegation\nBefore work beyond small direct tasks, publish own state with subject via `orchestrator_session_id`. Refresh at task changes/completion.\nA child must be able to do the task it is given. Check `subagent_list_agents`.\n")
		}, func(t *testing.T, output string) {
			got := readUpgradeFixture(t, output, "orchestrator-delegation.md")
			for _, forbidden := range []string{"orchestrator_session_id", "orchestrator_list", "subagent_list_agents"} {
				if strings.Contains(got, forbidden) {
					t.Errorf("non-native session tool %q", forbidden)
				}
			}
			if !strings.Contains(got, "declared tools") {
				t.Error("child capability check disappeared")
			}
		}},
		{"native-modular-core", func(t *testing.T, source string) {}, func(t *testing.T, output string) {
			got := readUpgradeFixture(t, output, "orchestrator.md")
			for _, want := range []string{"Required Phase Loading", "Every task follows ODD", "rules/AGENTS.md", "orchestrator-verification.md", "asynchronous"} {
				if !strings.Contains(got, want) {
					t.Errorf("generated core lost %q", want)
				}
			}
		}},
		{"retired-workflow-skills", func(t *testing.T, source string) {
			writeUpgradeFixture(t, filepath.Join(source, "skills/test-skill/SKILL.md"), "---\nname: test-skill\ndescription: Work units.\n---\n\n## SDD Relationship\n\nWhen `sdd-tasks` produces a Review Workload Forecast:\n- High risk: follow SDD `delivery_strategy`.\nEach SDD work unit should map cleanly to a commit.\nThe same delivery-strategy vocabulary as SDD applies.\nAlways write `.atl/skill-registry.md` regardless of SDD persistence mode.\nSkip `sdd-*`, `_shared`, and `skill-registry`.\n")
		}, func(t *testing.T, output string) {
			got := readUpgradeFixture(t, output, "skills/test-skill/SKILL.md")
			if strings.Contains(strings.ToLower(got), "sdd") {
				t.Errorf("retired workflow survives: %s", got)
			}
			for _, want := range []string{"ODD", "delivery_strategy", ".atl/skill-registry.md", "_shared"} {
				if !strings.Contains(got, want) {
					t.Errorf("portable skill lost %q", want)
				}
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source, output := t.TempDir(), t.TempDir()
			writeMinimalSourceTree(t, source)
			tc.prepare(t, source)
			if err := initGitRepo(t, source); err != nil {
				t.Fatal(err)
			}
			command := exec.Command(binary, "--source-dir", source, "--output-dir", output, "--strict")
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			if err := command.Run(); err != nil {
				t.Fatalf("exit: %v; stdout=%q; stderr=%q", err, stdout.String(), stderr.String())
			}
			if stdout.String() != "Successfully transpiled Antigravity CLI assets from Gentle Pi.\n" || stderr.Len() != 0 {
				t.Fatalf("stdout=%q; stderr=%q", stdout.String(), stderr.String())
			}
			tc.check(t, output)
		})
	}
}

func writeUpgradeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readUpgradeFixture(t *testing.T, root, path string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestShell420PublicDirtySourceRejected(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "transpiler")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, out)
	}
	for _, kind := range []string{"tracked", "untracked", "ignored"} {
		t.Run(kind, func(t *testing.T) {
			source, output := t.TempDir(), t.TempDir()
			writeMinimalSourceTree(t, source)
			writeUpgradeFixture(t, filepath.Join(source, ".gitignore"), "ignored.txt\n")
			if err := initGitRepo(t, source); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(source, kind+".txt")
			if kind == "tracked" {
				path = filepath.Join(source, "assets/orchestrator.md")
			}
			writeUpgradeFixture(t, path, "dirty bytes\n")
			sentinel := filepath.Join(output, "sentinel")
			writeUpgradeFixture(t, sentinel, "preserve me\n")
			cmd := exec.Command(binary, "--source-dir", source, "--output-dir", output, "--strict")
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			exit, ok := err.(*exec.ExitError)
			want := "Error resolving source repository: get git HEAD for " + source + ": source checkout must be clean (including untracked and ignored files)\n"
			if !ok || exit.ExitCode() != 1 || stdout.Len() != 0 || stderr.String() != want {
				t.Errorf("exit=%v stdout=%q stderr=%q want=%q", err, stdout.String(), stderr.String(), want)
			}
			entries, readErr := os.ReadDir(output)
			if readErr != nil || len(entries) != 1 || readUpgradeFixture(t, output, "sentinel") != "preserve me\n" {
				t.Errorf("rejection changed output: entries=%v err=%v", entries, readErr)
			}
		})
	}
}

func TestShell420PublicStrictCheckRejectsUnknownTool(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "transpiler")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, out)
	}
	source, repo := t.TempDir(), t.TempDir()
	output := filepath.Join(repo, "bundle")
	writeMinimalSourceTree(t, source)
	writeUpgradeFixture(t, filepath.Join(source, "assets/agents/test-agent.md"), "---\nname: test-agent\ndescription: Unknown tool fixture.\ntools:\n  - unknown-fixture-tool\n---\n\nFixture body.\n")
	if err := initGitRepo(t, source); err != nil {
		t.Fatal(err)
	}
	generate := exec.Command(binary, "--source-dir", source, "--output-dir", output)
	if out, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("non-strict generation: %v: %s", err, out)
	}
	if err := initGitRepo(t, repo); err != nil {
		t.Fatal(err)
	}
	snapshot := func() string {
		cmd := exec.Command("git", "status", "--porcelain=v1", "--untracked-files=all")
		cmd.Dir = repo
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		return string(out)
	}
	before := snapshot()
	cmd := exec.Command(binary, "--source-dir", source, "--output-dir", output, "--strict", "--check")
	cmd.Dir = repo
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 1 || stdout.Len() != 0 || stderr.String() != "Strict mode: unmapped tools detected during drift check.\n" {
		t.Errorf("exit=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if after := snapshot(); after != before {
		t.Errorf("read-only rejection changed stored bundle: before=%q after=%q", before, after)
	}
}
