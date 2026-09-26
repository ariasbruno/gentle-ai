package reviewerprovider

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestResolveAntigravityCLIReviewRouting(t *testing.T) {
	for _, tc := range []struct {
		name, config, model string
		invalid             bool
		assignmentKeyMissing bool
	}{
		{"string", `{"review-refuter":"provider/model"}`, "provider/model", false, false},
		{"model", `{"review-refuter":{"model":" provider/model "}}`, "provider/model", false, false},
		{"inherit string", `{"review-refuter":"inherit"}`, "", false, false},
		{"inherit object", `{"review-refuter":{"model":"inherit"}}`, "", false, false},
		{"inherit uppercase", `{"review-refuter":{"model":"INHERIT"}}`, "", false, false},
		{"independent", `{"review-refuter":{"model":"a","effort":"high"},"review-validator":{"model":"b"}}`, "a", false, false},
		{"empty", `{"review-refuter":{}}`, "", false, false},
		{"missing", `{"unrelated":false}`, "", false, true},
		{"syntax", `{`, "", true, false},
		{"array", `[]`, "", true, false},
		{"null", `null`, "", true, false},
		{"null entry", `{"review-refuter":null}`, "", true, false},
		{"bad selected", `{"review-refuter":false}`, "", true, false},
		{"bad model", `{"review-refuter":{"model":"bad model"}}`, "", true, false},
		{"bad effort with no model", `{"review-refuter":{"effort":"huge"}}`, "", true, false},
		{"unknown field", `{"review-refuter":{"typo":"a"}}`, "", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			path := antigravityCLIReviewModelsPath(home)
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.config), 0o600); err != nil {
				t.Fatal(err)
			}
			got, assignmentKeyMissing, err := ResolveAntigravityCLIReviewRouting(ReviewRoutingKeyRefuter)
			if (err != nil) != tc.invalid {
				t.Fatalf("routing error = %v", err)
			}
			if assignmentKeyMissing != tc.assignmentKeyMissing {
				t.Fatalf("assignmentKeyMissing = %v, want %v", assignmentKeyMissing, tc.assignmentKeyMissing)
			}
			if !tc.invalid && got.Model != tc.model {
				t.Fatalf("routing = %+v, want model %q", got, tc.model)
			}
		})
	}
}

func TestResolveAntigravityCLIReviewRoutingCoversRolesAndLenses(t *testing.T) {
	config := `{
		"review-refuter":"refuter/model",
		"review-validator":"validator/model",
		"review-risk":"risk/model",
		"review-resilience":{"model":"resilience/model"},
		"review-readability":{"model":"readability/model","effort":"high"},
		"review-reliability":"reliability/model"
	}`
	want := map[string]AntigravityCLIReviewRouting{
		ReviewRoutingKeyRefuter:     {Model: "refuter/model"},
		ReviewRoutingKeyValidator:   {Model: "validator/model"},
		ReviewRoutingKeyRisk:        {Model: "risk/model"},
		ReviewRoutingKeyResilience:  {Model: "resilience/model"},
		ReviewRoutingKeyReadability: {Model: "readability/model"},
		ReviewRoutingKeyReliability: {Model: "reliability/model"},
	}
	keys := []string{
		ReviewRoutingKeyRefuter, ReviewRoutingKeyValidator, ReviewRoutingKeyRisk,
		ReviewRoutingKeyResilience, ReviewRoutingKeyReadability, ReviewRoutingKeyReliability,
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	path := antigravityCLIReviewModelsPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		got, assignmentKeyMissing, err := ResolveAntigravityCLIReviewRouting(key)
		if err != nil {
			t.Fatalf("routing %q error = %v", key, err)
		}
		if assignmentKeyMissing {
			t.Fatalf("routing %q assignmentKeyMissing = true, want false", key)
		}
		if got != want[key] {
			t.Fatalf("routing %q = %+v, want %+v", key, got, want[key])
		}
	}
}

func TestResolveAntigravityCLIReviewRoutingLegacyEffortTolerated(t *testing.T) {
	config := `{
		"review-risk":{"model":"a","effort":"low"},
		"review-resilience":{"model":"b","effort":"medium"},
		"review-readability":{"model":"c","effort":"high"}
	}`
	want := []AntigravityCLIReviewRouting{
		{Model: "a"},
		{Model: "b"},
		{Model: "c"},
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	path := antigravityCLIReviewModelsPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	for index, key := range []string{ReviewRoutingKeyRisk, ReviewRoutingKeyResilience, ReviewRoutingKeyReadability} {
		got, assignmentKeyMissing, err := ResolveAntigravityCLIReviewRouting(key)
		if err != nil {
			t.Fatalf("routing %q error = %v", key, err)
		}
		if assignmentKeyMissing {
			t.Fatalf("routing %q assignmentKeyMissing = true, want false", key)
		}
		if got != want[index] {
			t.Fatalf("routing %q = %+v, want %+v", key, got, want[index])
		}
	}
}

func TestResolveAntigravityCLIReviewRoutingAbsence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	got, assignmentKeyMissing, err := ResolveAntigravityCLIReviewRouting(ReviewRoutingKeyRisk)
	if err != nil {
		t.Fatalf("missing assignment file error = %v, want nil", err)
	}
	if assignmentKeyMissing {
		t.Fatalf("missing assignment file assignmentKeyMissing = true, want false (file-absent stays silent)")
	}
	if got != (AntigravityCLIReviewRouting{}) {
		t.Fatalf("routing without assignment file = %+v, want zero value", got)
	}
}

func TestInvalidReviewRoutingErrorIsTyped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	path := antigravityCLIReviewModelsPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"review-risk":{"model":"bad model"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := ResolveAntigravityCLIReviewRouting(ReviewRoutingKeyRisk)
	if err == nil {
		t.Fatal("routing error = nil, want typed InvalidReviewRoutingError")
	}
	var typed *InvalidReviewRoutingError
	if !errors.As(err, &typed) {
		t.Fatalf("routing error = %T (%v), want errors.As to match *InvalidReviewRoutingError", err, err)
	}
	if typed.Role != ReviewRoutingKeyRisk {
		t.Fatalf("typed.Role = %q, want %q", typed.Role, ReviewRoutingKeyRisk)
	}
	if typed.Path != path {
		t.Fatalf("typed.Path = %q, want %q", typed.Path, path)
	}
}

func TestReviewRoutingKeyForLens(t *testing.T) {
	for _, lens := range []string{ReviewRoutingKeyRisk, ReviewRoutingKeyResilience, ReviewRoutingKeyReadability, ReviewRoutingKeyReliability} {
		key, err := ReviewRoutingKeyForLens(lens)
		if err != nil {
			t.Fatalf("ReviewRoutingKeyForLens(%q) error = %v", lens, err)
		}
		if key != lens {
			t.Fatalf("ReviewRoutingKeyForLens(%q) = %q, want itself", lens, key)
		}
	}
	if key, err := ReviewRoutingKeyForLens("review-bogus"); err == nil {
		t.Fatalf("ReviewRoutingKeyForLens(review-bogus) = %q, want error", key)
	}
}

func antigravityCLIReviewModelsPath(home string) string {
	return filepath.Join(home, ".gemini", "antigravity-cli", "gentle-ai", "review-models.json")
}

func TestSaveAndLoadAllAntigravityCLIReviewRouting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	// Loading when file does not exist returns empty map without error
	loaded, err := LoadAllAntigravityCLIReviewRouting()
	if err != nil {
		t.Fatalf("LoadAllAntigravityCLIReviewRouting() unexpected error: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("LoadAllAntigravityCLIReviewRouting() on missing file len = %d, want 0", len(loaded))
	}

	assignments := map[string]AntigravityCLIReviewRouting{
		ReviewRoutingKeyRisk:        {Model: "claude-opus-4-6-thinking"},
		ReviewRoutingKeyReadability: {Model: "gemini-3.8-flash-medium"},
		ReviewRoutingKeyValidator:   {Model: "gemini-3.1-pro-high"},
	}

	if err := SaveAntigravityCLIReviewRouting(assignments); err != nil {
		t.Fatalf("SaveAntigravityCLIReviewRouting() error: %v", err)
	}

	loaded, err = LoadAllAntigravityCLIReviewRouting()
	if err != nil {
		t.Fatalf("LoadAllAntigravityCLIReviewRouting() error: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("LoadAllAntigravityCLIReviewRouting() len = %d, want 3", len(loaded))
	}

	for k, want := range assignments {
		got, ok := loaded[k]
		if !ok {
			t.Errorf("missing key %q in loaded routing", k)
			continue
		}
		if got.Model != want.Model {
			t.Errorf("key %q = %+v, want %+v", k, got, want)
		}
	}
}

func TestAntigravityReviewPresets(t *testing.T) {
	roles := CanonicalReviewRoles()
	if len(roles) != 6 {
		t.Fatalf("CanonicalReviewRoles() len = %d, want 6", len(roles))
	}

	presets := []struct {
		name        AntigravityReviewPreset
		fn          func() map[string]AntigravityCLIReviewRouting
		expectEmpty bool
	}{
		{AntigravityPresetTypeDefault, AntigravityPresetDefault, true},
		{AntigravityPresetTypeRecommended, AntigravityPresetRecommended, false},
		{AntigravityPresetTypePerformance, AntigravityPresetPerformance, false},
		{AntigravityPresetTypeEconomy, AntigravityPresetEconomy, false},
	}

	for _, tc := range presets {
		t.Run(string(tc.name), func(t *testing.T) {
			got := tc.fn()
			if len(got) != 6 {
				t.Errorf("preset %s returned %d assignments, want 6", tc.name, len(got))
			}
			for _, role := range roles {
				route, ok := got[role]
				if !ok {
					t.Errorf("preset %s missing role %s", tc.name, role)
					continue
				}
				if tc.expectEmpty && route.Model != "" {
					t.Errorf("preset %s role %s has model %q, want empty", tc.name, role, route.Model)
				}
				if !tc.expectEmpty && route.Model == "" {
					t.Errorf("preset %s role %s has empty model", tc.name, role)
				}
			}
		})
	}
}

func TestSaveAntigravityCLIReviewRoutingSerializesInherit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	assignments := map[string]AntigravityCLIReviewRouting{
		ReviewRoutingKeyRisk:      {Model: ""},
		ReviewRoutingKeyValidator: {Model: "inherit"},
	}

	if err := SaveAntigravityCLIReviewRouting(assignments); err != nil {
		t.Fatalf("SaveAntigravityCLIReviewRouting() error: %v", err)
	}

	raw, err := os.ReadFile(antigravityCLIReviewModelsPath(home))
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, `"review-risk": "inherit"`) {
		t.Errorf("expected review-risk to be saved as inherit, got: %s", content)
	}
	if !strings.Contains(content, `"review-validator": "inherit"`) {
		t.Errorf("expected review-validator to be saved as inherit, got: %s", content)
	}

	loaded, err := LoadAllAntigravityCLIReviewRouting()
	if err != nil {
		t.Fatalf("LoadAllAntigravityCLIReviewRouting() error: %v", err)
	}
	if loaded[ReviewRoutingKeyRisk].Model != "" {
		t.Errorf("loaded review-risk model = %q, want empty", loaded[ReviewRoutingKeyRisk].Model)
	}
	if loaded[ReviewRoutingKeyValidator].Model != "" {
		t.Errorf("loaded review-validator model = %q, want empty", loaded[ReviewRoutingKeyValidator].Model)
	}
}

func TestAntigravityReviewRoutingsEqual(t *testing.T) {
	a := AntigravityPresetRecommended()
	b := AntigravityPresetRecommended()
	if !AntigravityReviewRoutingsEqual(a, b) {
		t.Error("identical presets reported not equal")
	}

	b[ReviewRoutingKeyRisk] = AntigravityCLIReviewRouting{Model: "different"}
	if AntigravityReviewRoutingsEqual(a, b) {
		t.Error("modified preset reported equal")
	}

	if AntigravityReviewRoutingsEqual(a, nil) {
		t.Error("non-nil compared to nil reported equal")
	}
}

func TestParseAntigravityModelsOutput(t *testing.T) {
	sample := `⠋ Fetching available models...⠙ Fetching available models...
gemini-3.8-flash-high     Gemini 3.8 Flash (High)
gemini-3.8-flash-medium   Gemini 3.8 Flash (Medium)
claude-opus-4-6-thinking  Claude Opus 4.6 (Thinking)
`
	got := ParseAntigravityModelsOutput(sample)
	want := []string{
		"gemini-3.8-flash-high",
		"gemini-3.8-flash-medium",
		"claude-opus-4-6-thinking",
	}
	if len(got) != len(want) {
		t.Fatalf("ParseAntigravityModelsOutput() returned %d models, want %d", len(got), len(want))
	}
	for i, m := range want {
		if got[i] != m {
			t.Errorf("model[%d] = %q, want %q", i, got[i], m)
		}
	}
}

func TestDiscoverAntigravityModels(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		output      string
		timeout     time.Duration
		want        []string
		wantTimeout bool
	}{
		{
			name:    "success",
			mode:    "output",
			output:  "gemini-3.8-flash-high     Gemini 3.8 Flash (High)\nclaude-opus-4-6-thinking  Claude Opus 4.6 (Thinking)\n",
			timeout: time.Second,
			want:    []string{"gemini-3.8-flash-high", "claude-opus-4-6-thinking"},
		},
		{
			name:    "command failure",
			mode:    "failure",
			timeout: time.Second,
			want:    DefaultAntigravityReviewModels,
		},
		{
			name:        "timeout",
			mode:        "timeout",
			timeout:     20 * time.Millisecond,
			want:        DefaultAntigravityReviewModels,
			wantTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalCommand := antigravityModelCommand
			originalTimeout := antigravityModelDiscoveryTimeout
			t.Cleanup(func() {
				antigravityModelCommand = originalCommand
				antigravityModelDiscoveryTimeout = originalTimeout
			})

			var (
				called      bool
				gotContext  context.Context
				gotName     string
				gotArgs     []string
				contextDone chan error
			)
			antigravityModelDiscoveryTimeout = tt.timeout
			antigravityModelCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
				called = true
				gotContext = ctx
				gotName = name
				gotArgs = append([]string(nil), args...)
				if tt.wantTimeout {
					contextDone = make(chan error, 1)
					go func() {
						<-ctx.Done()
						contextDone <- ctx.Err()
					}()
				}
				return antigravityModelsHelperCommand(ctx, tt.mode, tt.output)
			}

			got := DiscoverAntigravityModels()
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("DiscoverAntigravityModels() = %v, want %v", got, tt.want)
			}
			if !called {
				t.Fatal("Antigravity model command was not called")
			}
			if gotContext == nil {
				t.Fatal("Antigravity model command received a nil context")
			}
			if _, ok := gotContext.Deadline(); !ok {
				t.Fatal("Antigravity model command context has no deadline")
			}
			if gotName != "agy" {
				t.Errorf("command name = %q, want %q", gotName, "agy")
			}
			if want := []string{"models"}; !reflect.DeepEqual(gotArgs, want) {
				t.Errorf("command args = %v, want %v", gotArgs, want)
			}
			if tt.wantTimeout {
				select {
				case err := <-contextDone:
					if !errors.Is(err, context.DeadlineExceeded) {
						t.Fatalf("command context error = %v, want deadline exceeded", err)
					}
				case <-time.After(time.Second):
					t.Fatal("command context was not canceled by the timeout")
				}
			}
		})
	}
}

func antigravityModelsHelperCommand(ctx context.Context, mode, output string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAntigravityModelsHelperProcess$", "--")
	cmd.Env = append(os.Environ(),
		"GO_WANT_ANTIGRAVITY_MODELS_HELPER=1",
		"GO_ANTIGRAVITY_MODELS_HELPER_MODE="+mode,
		"GO_ANTIGRAVITY_MODELS_HELPER_OUTPUT="+output,
	)
	return cmd
}

func TestAntigravityModelsHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_ANTIGRAVITY_MODELS_HELPER") != "1" {
		return
	}

	switch os.Getenv("GO_ANTIGRAVITY_MODELS_HELPER_MODE") {
	case "failure":
		os.Exit(1)
	case "timeout":
		select {}
	default:
		_, _ = os.Stdout.WriteString(os.Getenv("GO_ANTIGRAVITY_MODELS_HELPER_OUTPUT"))
	}
	os.Exit(0)
}

func TestIsKnownAntigravityModel(t *testing.T) {
	if !IsKnownAntigravityModel("claude-opus-4-6-thinking", nil) {
		t.Error("claude-opus-4-6-thinking should be known in default catalog")
	}
	if IsKnownAntigravityModel("deprecated-gemini-1.0-ultra", nil) {
		t.Error("deprecated-gemini-1.0-ultra should NOT be known in default catalog")
	}
}
