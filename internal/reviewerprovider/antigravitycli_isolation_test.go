package reviewerprovider

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func prepareAntigravityReviewerAuth(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"antigravity-oauth-token": "synthetic-auth-token", "installation_id": "synthetic-installation-id"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

func TestAntigravityCLIAdapterInheritsModelDefaultWithoutModelFlag(t *testing.T) {
	for _, model := range []string{"", "inherit", " InHeRiT ", "  "} {
		t.Run(model, func(t *testing.T) {
			home := prepareAntigravityReviewerAuth(t)
			path := filepath.Join(t.TempDir(), "argv")
			adapter := &AntigravityCLIAdapter{Model: model, LookPath: func(string) (string, error) { return "agy", nil }, commandContext: func(ctx context.Context, _ string, args ...string) *exec.Cmd {
				return exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestAntigravityCLIAdapterHelperProcess$", "--", agyAdapterHelperModeArgument + "arguments", agyAdapterHelperOutputPathArgument + path}, args...)...)
			}}
			raw, err := adapter.Review(t.Context(), NewInvocation([]byte("opaque provider input")))
			if err != nil || string(raw) != "raw\x00agy\xffoutput" {
				t.Fatalf("Review=%q,%v; want the inherit default to launch the reviewer", raw, err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got := strings.Split(string(data), "\x00")[5:]
			if slices.Contains(got, "--model") || !slices.Equal(got, []string{"--sandbox", "--output-format", "text"}) {
				t.Fatalf("child argv=%q; want no --model for the inherit default", got)
			}
			token, err := os.ReadFile(filepath.Join(home, ".gemini", "antigravity-cli", "antigravity-oauth-token"))
			if err != nil || string(token) != "synthetic-auth-token" {
				t.Fatal("inherit launch changed source authentication")
			}
		})
	}
}

func TestAntigravityCLIAdapterUsesAuthOnlyProfileAndCleansIt(t *testing.T) {
	for _, mode := range []string{"auth-profile", "fail"} {
		t.Run(mode, func(t *testing.T) {
			home := prepareAntigravityReviewerAuth(t)
			globals := []string{filepath.Join(home, ".gemini", "AGENTS.md"), filepath.Join(home, ".gemini", "antigravity-cli", "settings.json"), filepath.Join(home, ".gemini", "config", "plugins", "canary", "plugin.json")}
			for _, path := range globals {
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("UNRELATED_GLOBAL_CONTEXT"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			snapshotPath := filepath.Join(t.TempDir(), "snapshot.json")
			var command *exec.Cmd
			adapter := &AntigravityCLIAdapter{Model: "explicit-model", Env: []string{"HOME=" + home, "USERPROFILE=" + home}, LookPath: func(string) (string, error) { return "agy", nil }, commandContext: func(ctx context.Context, _ string, args ...string) *exec.Cmd {
				command = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAntigravityCLIIsolationHelper$", "--", agyAdapterHelperModeArgument+mode, agyAdapterHelperOutputPathArgument+snapshotPath)
				return command
			}}
			raw, err := adapter.Review(t.Context(), NewInvocation([]byte("opaque provider input")))
			if mode == "auth-profile" {
				if err != nil || string(raw) != "opaque-final-bytes" {
					t.Fatalf("Review=%q,%v", raw, err)
				}
			} else {
				if err == nil || raw != nil || !strings.Contains(err.Error(), "transport failed") {
					t.Fatalf("Review=%q,%v; want failed transport", raw, err)
				}
			}
			childHome := environmentMap([]byte(strings.Join(command.Env, "\x00")))["HOME"]
			if childHome == "" || childHome == home {
				t.Fatalf("child HOME=%q; want a disposable profile", childHome)
			}
			if _, err := os.Stat(childHome); !os.IsNotExist(err) {
				t.Fatalf("disposable HOME remains: %v", err)
			}
			if _, err := os.Stat(command.Dir); !os.IsNotExist(err) {
				t.Fatalf("scratch remains: %v", err)
			}
			for _, path := range globals {
				data, err := os.ReadFile(path)
				if err != nil || string(data) != "UNRELATED_GLOBAL_CONTEXT" {
					t.Fatalf("source global changed: %s", path)
				}
			}
			token, err := os.ReadFile(filepath.Join(home, ".gemini", "antigravity-cli", "antigravity-oauth-token"))
			if err != nil || string(token) != "synthetic-auth-token" {
				t.Fatal("source auth changed")
			}
			if mode == "auth-profile" {
				data, err := os.ReadFile(snapshotPath)
				if err != nil {
					t.Fatal(err)
				}
				var snapshot struct {
					Token string
					ID    string
					Files []string
					Mode  uint32
				}
				if err := json.Unmarshal(data, &snapshot); err != nil {
					t.Fatal(err)
				}
				if snapshot.Token != "synthetic-auth-token" || snapshot.ID != "synthetic-installation-id" || snapshot.Mode != 0600 || strings.Join(snapshot.Files, ",") != ".gemini/antigravity-cli/antigravity-oauth-token,.gemini/antigravity-cli/installation_id" {
					t.Fatalf("unexpected profile snapshot: %+v", snapshot)
				}
			}
		})
	}
}

func TestAntigravityCLIAdapterRefusesUnusableAuthWithoutLaunch(t *testing.T) {
	for _, kind := range []string{"missing", "empty", "directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			home := prepareAntigravityReviewerAuth(t)
			path := filepath.Join(home, ".gemini", "antigravity-cli", "antigravity-oauth-token")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "empty":
				if err := os.WriteFile(path, nil, 0600); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(filepath.Join(home, ".gemini", "antigravity-cli", "installation_id"), path); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			}
			called := false
			adapter := &AntigravityCLIAdapter{Model: "explicit-model", LookPath: func(string) (string, error) { return "agy", nil }, commandContext: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				called = true
				return exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAntigravityCLIAdapterHelperProcess$", "--", agyAdapterHelperModeArgument+"success")
			}}
			raw, err := adapter.Review(t.Context(), NewInvocation([]byte("opaque")))
			if raw != nil || err == nil || !strings.Contains(err.Error(), "transport unavailable: prepare isolated authentication profile") || called {
				t.Fatalf("Review=%q,%v; launched=%t", raw, err, called)
			}
			id, err := os.ReadFile(filepath.Join(home, ".gemini", "antigravity-cli", "installation_id"))
			if err != nil || string(id) != "synthetic-installation-id" {
				t.Fatal("refusal changed source data")
			}
		})
	}
}

func TestAntigravityCLIIsolationHelper(t *testing.T) {
	mode := agyAdapterHelperOption(agyAdapterHelperModeArgument)
	if mode != "auth-profile" && mode != "fail" && mode != "lock-home" && mode != "blind-scratch" {
		return
	}
	if mode == "fail" {
		os.Exit(3)
	}
	home := os.Getenv("HOME")
	if mode == "lock-home" {
		if err := os.Chmod(home, 0o000); err != nil {
			os.Exit(9)
		}
		os.Stdout.WriteString("opaque-final-bytes")
		os.Exit(0)
	}
	if mode == "blind-scratch" {
		if err := os.Chmod(filepath.Join(home, ".."), 0o100); err != nil {
			os.Exit(13)
		}
		os.Stdout.WriteString("opaque-final-bytes")
		os.Exit(0)
	}
	tokenPath := filepath.Join(home, ".gemini", "antigravity-cli", "antigravity-oauth-token")
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		os.Exit(4)
	}
	id, err := os.ReadFile(filepath.Join(home, ".gemini", "antigravity-cli", "installation_id"))
	if os.IsNotExist(err) {
		id = nil
	} else if err != nil {
		os.Exit(5)
	}
	info, err := os.Stat(tokenPath)
	if err != nil {
		os.Exit(6)
	}
	files := []string{}
	err = filepath.WalkDir(home, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			relative, _ := filepath.Rel(home, path)
			files = append(files, filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		os.Exit(7)
	}
	data, _ := json.Marshal(struct {
		Token string
		ID    string
		Files []string
		Mode  uint32
	}{string(token), string(id), files, uint32(info.Mode().Perm())})
	if err := os.WriteFile(agyAdapterHelperOption(agyAdapterHelperOutputPathArgument), data, 0600); err != nil {
		os.Exit(8)
	}
	os.Stdout.WriteString("opaque-final-bytes")
	os.Exit(0)
}

func TestAntigravityCLIAdapterRefusesSymlinkedAuthDirectoryWithoutLaunch(t *testing.T) {
	for _, name := range []string{"gemini", "antigravity-cli"} {
		t.Run(name, func(t *testing.T) {
			home := prepareAntigravityReviewerAuth(t)
			elsewhere := t.TempDir()
			authElsewhere := filepath.Join(elsewhere, "antigravity-cli")
			if err := os.MkdirAll(authElsewhere, 0700); err != nil {
				t.Fatal(err)
			}
			for _, file := range []string{"antigravity-oauth-token", "installation_id"} {
				if err := os.WriteFile(filepath.Join(authElsewhere, file), []byte("synthetic-auth-token"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			linked := filepath.Join(home, ".gemini")
			if name == "antigravity-cli" {
				linked = filepath.Join(home, ".gemini", "antigravity-cli")
			}
			if err := os.RemoveAll(linked); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(authElsewhere, linked); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			called := false
			adapter := &AntigravityCLIAdapter{Model: "explicit-model", LookPath: func(string) (string, error) { return "agy", nil }, commandContext: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				called = true
				return exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAntigravityCLIAdapterHelperProcess$", "--", agyAdapterHelperModeArgument+"success")
			}}
			raw, err := adapter.Review(t.Context(), NewInvocation([]byte("opaque")))
			if raw != nil || err == nil || !strings.Contains(err.Error(), "prepare isolated authentication profile") || !strings.Contains(err.Error(), "regular directory") || called {
				t.Fatalf("Review=%q,%v; launched=%t; want directory-symlink refusal", raw, err, called)
			}
			data, err := os.ReadFile(filepath.Join(authElsewhere, "antigravity-oauth-token"))
			if err != nil || string(data) != "synthetic-auth-token" {
				t.Fatal("refusal changed source data")
			}
		})
	}
}

func TestAntigravityCLIAdapterFailsClosedWhenCleanupFails(t *testing.T) {
	for _, mode := range []string{"lock-home", "blind-scratch"} {
		t.Run(mode, func(t *testing.T) {
			if runtime.GOOS == "windows" || os.Geteuid() == 0 {
				t.Skip("cleanup failure cannot be simulated on this platform or as root")
			}
			home := prepareAntigravityReviewerAuth(t)
			var command *exec.Cmd
			adapter := &AntigravityCLIAdapter{Model: "explicit-model", Env: []string{"HOME=" + home, "USERPROFILE=" + home}, LookPath: func(string) (string, error) { return "agy", nil }, commandContext: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
				command = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAntigravityCLIIsolationHelper$", "--", agyAdapterHelperModeArgument+mode)
				return command
			}}
			raw, err := adapter.Review(t.Context(), NewInvocation([]byte("opaque provider input")))
			if raw != nil || err == nil || !strings.Contains(err.Error(), "remove temporary profile") {
				t.Fatalf("Review=%q,%v; want nil bytes and cleanup transport failure", raw, err)
			}
			childHome := environmentMap([]byte(strings.Join(command.Env, "\x00")))["HOME"]
			if childHome == "" || childHome == home {
				t.Fatalf("child HOME=%q; want disposable profile", childHome)
			}
			tokenPath := filepath.Join(childHome, ".gemini", "antigravity-cli", "antigravity-oauth-token")
			if mode == "lock-home" {
				if !strings.Contains(err.Error(), "isolated credentials may remain") {
					t.Fatalf("error missing the credential persistence warning: %v", err)
				}
				if restoreErr := os.Chmod(childHome, 0o700); restoreErr != nil {
					t.Fatal(restoreErr)
				}
				if _, statErr := os.Stat(tokenPath); statErr != nil {
					t.Fatalf("credentials must persist when destruction is blocked, got: %v", statErr)
				}
			} else {
				if _, statErr := os.Stat(tokenPath); !os.IsNotExist(statErr) {
					t.Fatalf("credentials survived a reachable cleanup failure: %v", statErr)
				}
				if strings.Contains(err.Error(), "isolated credentials may remain") {
					t.Fatalf("warning must not appear when credentials were destroyed: %v", err)
				}
			}
			if restoreErr := os.Chmod(filepath.Dir(childHome), 0o700); restoreErr != nil {
				t.Fatal(restoreErr)
			}
			if err := os.RemoveAll(filepath.Dir(childHome)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAntigravityCLIAdapterToleratesMissingOptionalInstallationID(t *testing.T) {
	home := prepareAntigravityReviewerAuth(t)
	if err := os.Remove(filepath.Join(home, ".gemini", "antigravity-cli", "installation_id")); err != nil {
		t.Fatal(err)
	}
	snapshotPath := filepath.Join(t.TempDir(), "snapshot.json")
	var command *exec.Cmd
	adapter := &AntigravityCLIAdapter{Model: "explicit-model", Env: []string{"HOME=" + home, "USERPROFILE=" + home}, LookPath: func(string) (string, error) { return "agy", nil }, commandContext: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		command = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAntigravityCLIIsolationHelper$", "--", agyAdapterHelperModeArgument+"auth-profile", agyAdapterHelperOutputPathArgument+snapshotPath)
		return command
	}}
	raw, err := adapter.Review(t.Context(), NewInvocation([]byte("opaque provider input")))
	if err != nil || string(raw) != "opaque-final-bytes" {
		t.Fatalf("Review=%q,%v", raw, err)
	}
	childHome := environmentMap([]byte(strings.Join(command.Env, "\x00")))["HOME"]
	if _, err := os.Stat(childHome); !os.IsNotExist(err) {
		t.Fatalf("disposable HOME remains: %v", err)
	}
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		Token string
		ID    string
		Files []string
		Mode  uint32
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Token != "synthetic-auth-token" || snapshot.ID != "" || strings.Join(snapshot.Files, ",") != ".gemini/antigravity-cli/antigravity-oauth-token" {
		t.Fatalf("unexpected profile snapshot: %+v", snapshot)
	}
}
