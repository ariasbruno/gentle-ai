package reviewerprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// AntigravityCLIReviewRouting carries the home-owned model assignment; zero
// values retain the antigravity-cli runtime's own defaults.
type AntigravityCLIReviewRouting struct{ Model string }

// InvalidReviewRoutingError reports a present-but-unusable home routing
// assignment: malformed JSON, unknown fields, blank or invalid model
// values, or a model outside the fixed identifier charset. Callers can detect
// the repair-required condition with errors.As instead of parsing prose.
type InvalidReviewRoutingError struct {
	Role string // routing key whose assignment is invalid
	Path string // assignment file path
}

func (e *InvalidReviewRoutingError) Error() string {
	return fmt.Sprintf("invalid antigravity-cli review routing for %s in %s; repair the assignment using the antigravity-cli review model configuration before retrying", e.Role, e.Path)
}

// Canonical routing assignment keys for the captured roles and the published
// review lenses. Runtime code and tests share these constants so a string
// literal can never drift from the resolver's accepted keys.
const (
	ReviewRoutingKeyRefuter     = "review-refuter"
	ReviewRoutingKeyValidator   = "review-validator"
	ReviewRoutingKeyRisk        = "review-risk"
	ReviewRoutingKeyResilience  = "review-resilience"
	ReviewRoutingKeyReadability = "review-readability"
	ReviewRoutingKeyReliability = "review-reliability"
)

var reviewModelIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9._~:@/+%-]+$`)

// ReviewRoutingKeyForLens returns the canonical routing assignment key for a
// published review lens, rejecting any value outside the published set so a
// drifted runtime lens string can never silently select the default route.
func ReviewRoutingKeyForLens(lens string) (string, error) {
	switch lens {
	case ReviewRoutingKeyRisk, ReviewRoutingKeyResilience, ReviewRoutingKeyReadability, ReviewRoutingKeyReliability:
		return lens, nil
	default:
		return "", fmt.Errorf("unknown review lens %q; routing assignment keys exist only for the published lenses %q, %q, %q, %q", lens, ReviewRoutingKeyRisk, ReviewRoutingKeyResilience, ReviewRoutingKeyReadability, ReviewRoutingKeyReliability)
	}
}

// ResolveAntigravityCLIReviewRouting reads gentle-ai's read-only model
// assignment for the antigravity-cli in-process reviewer runtime without
// modifying it. The assignment file grammar is: a top-level JSON object whose
// keys are routing roles or published lenses; each entry is either a bare
// model string, or an object with a model field.
// A JSON null entry is invalid, an empty object {} means runtime defaults,
// model whitespace is trimmed, and model names must match the fixed identifier
// charset [A-Za-z0-9._~:@/+%-]. Absence of the assignment file or of the named key
// resolves to the zero route, never an error and never written, with
// assignmentKeyMissing true only when the file exists but the key does not (the
// likely-typo case callers surface as a diagnostic); an invalid file fails
// closed with the typed InvalidReviewRoutingError. Unlike Pi, there is no
// repository-root fallback: the repository must not prescribe canonical
// models per role.
// AntigravityReviewModelsPath returns the canonical path to review-models.json.
func AntigravityReviewModelsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate antigravity-cli routing home: %w", err)
	}
	return filepath.Join(home, ".gemini", "antigravity-cli", "gentle-ai", "review-models.json"), nil
}

// DefaultAntigravityReviewModels contains the known model slugs available in Antigravity CLI.
var DefaultAntigravityReviewModels = []string{
	"gemini-3.8-flash-high",
	"gemini-3.8-flash-medium",
	"gemini-3.8-flash-low",
	"gemini-3.7-flash-high",
	"gemini-3.7-flash-medium",
	"gemini-3.7-flash-low",
	"gemini-3.6-flash-high",
	"gemini-3.6-flash-medium",
	"gemini-3.6-flash-low",
	"gemini-3.1-pro-high",
	"gemini-3.1-pro-low",
	"claude-sonnet-4-6",
	"claude-opus-4-6-thinking",
	"gpt-oss-120b-medium",
}

// CanonicalReviewRoles returns the 6 reviewer roles supported in review-models.json.
func CanonicalReviewRoles() []string {
	return []string{
		ReviewRoutingKeyRisk,
		ReviewRoutingKeyReadability,
		ReviewRoutingKeyReliability,
		ReviewRoutingKeyResilience,
		ReviewRoutingKeyRefuter,
		ReviewRoutingKeyValidator,
	}
}

// AntigravityReviewPreset represents a named preset for Antigravity review models.
type AntigravityReviewPreset string

const (
	AntigravityPresetTypeDefault     AntigravityReviewPreset = "default"
	AntigravityPresetTypeRecommended AntigravityReviewPreset = "recommended"
	AntigravityPresetTypePerformance AntigravityReviewPreset = "performance"
	AntigravityPresetTypeEconomy     AntigravityReviewPreset = "economy"
	AntigravityPresetTypeCustom      AntigravityReviewPreset = "custom"
)

// AntigravityPresetDescriptions provides descriptions for presets in the TUI.
var AntigravityPresetDescriptions = map[AntigravityReviewPreset]string{
	AntigravityPresetTypeDefault:     "Inherit CLI session default: omit --model flag so all reviewers use your Antigravity settings",
	AntigravityPresetTypeRecommended: "Claude Opus 4.6 for Risk & Refuter, Gemini Pro for Reliability & Validator, Flash for Resilience & Readability",
	AntigravityPresetTypePerformance: "Frontier reasoning models: Opus 4.6, Gemini 3.1 Pro, and Sonnet 4.6 across all review roles",
	AntigravityPresetTypeEconomy:     "Budget-conscious: Gemini 3.8 Flash high/medium/low tiers for fast review turns",
	AntigravityPresetTypeCustom:      "Assign a model to each of the 6 review roles individually",
}

// AntigravityPresetOrder specifies the order in which presets appear in the TUI picker.
var AntigravityPresetOrder = []AntigravityReviewPreset{
	AntigravityPresetTypeDefault,
	AntigravityPresetTypeRecommended,
	AntigravityPresetTypePerformance,
	AntigravityPresetTypeEconomy,
	AntigravityPresetTypeCustom,
}

// AntigravityPresetDefault returns empty model routing so all roles inherit CLI defaults.
func AntigravityPresetDefault() map[string]AntigravityCLIReviewRouting {
	m := make(map[string]AntigravityCLIReviewRouting, len(CanonicalReviewRoles()))
	for _, role := range CanonicalReviewRoles() {
		m[role] = AntigravityCLIReviewRouting{Model: ""}
	}
	return m
}

// AntigravityPresetRecommended returns the recommended reviewer model routing.
func AntigravityPresetRecommended() map[string]AntigravityCLIReviewRouting {
	return map[string]AntigravityCLIReviewRouting{
		ReviewRoutingKeyRisk:        {Model: "claude-opus-4-6-thinking"},
		ReviewRoutingKeyRefuter:     {Model: "claude-opus-4-6-thinking"},
		ReviewRoutingKeyReliability: {Model: "gemini-3.1-pro-high"},
		ReviewRoutingKeyValidator:   {Model: "gemini-3.1-pro-high"},
		ReviewRoutingKeyResilience:  {Model: "gemini-3.8-flash-medium"},
		ReviewRoutingKeyReadability: {Model: "gemini-3.8-flash-medium"},
	}
}

// AntigravityPresetPerformance returns frontier models for all reviewer roles.
func AntigravityPresetPerformance() map[string]AntigravityCLIReviewRouting {
	return map[string]AntigravityCLIReviewRouting{
		ReviewRoutingKeyRisk:        {Model: "claude-opus-4-6-thinking"},
		ReviewRoutingKeyRefuter:     {Model: "claude-opus-4-6-thinking"},
		ReviewRoutingKeyReliability: {Model: "claude-opus-4-6-thinking"},
		ReviewRoutingKeyValidator:   {Model: "gemini-3.1-pro-high"},
		ReviewRoutingKeyResilience:  {Model: "gemini-3.1-pro-high"},
		ReviewRoutingKeyReadability: {Model: "claude-sonnet-4-6"},
	}
}

// AntigravityPresetEconomy returns lightweight fast models for all reviewer roles.
func AntigravityPresetEconomy() map[string]AntigravityCLIReviewRouting {
	return map[string]AntigravityCLIReviewRouting{
		ReviewRoutingKeyRisk:        {Model: "gemini-3.8-flash-high"},
		ReviewRoutingKeyRefuter:     {Model: "gemini-3.8-flash-high"},
		ReviewRoutingKeyReliability: {Model: "gemini-3.8-flash-medium"},
		ReviewRoutingKeyValidator:   {Model: "gemini-3.8-flash-medium"},
		ReviewRoutingKeyResilience:  {Model: "gemini-3.8-flash-low"},
		ReviewRoutingKeyReadability: {Model: "gemini-3.8-flash-low"},
	}
}

// AntigravityPresetConstructors maps presets to their factory functions.
var AntigravityPresetConstructors = map[AntigravityReviewPreset]func() map[string]AntigravityCLIReviewRouting{
	AntigravityPresetTypeDefault:     AntigravityPresetDefault,
	AntigravityPresetTypeRecommended: AntigravityPresetRecommended,
	AntigravityPresetTypePerformance: AntigravityPresetPerformance,
	AntigravityPresetTypeEconomy:     AntigravityPresetEconomy,
}

// AntigravityReviewRoutingsEqual compares two routing configurations for exact equality.
func AntigravityReviewRoutingsEqual(a, b map[string]AntigravityCLIReviewRouting) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		modelA := strings.TrimSpace(va.Model)
		if strings.EqualFold(modelA, "inherit") {
			modelA = ""
		}
		modelB := strings.TrimSpace(vb.Model)
		if strings.EqualFold(modelB, "inherit") {
			modelB = ""
		}
		if !ok || modelA != modelB {
			return false
		}
	}
	return true
}

// ParseAntigravityModelsOutput extracts model slugs from "agy models" terminal output.
func ParseAntigravityModelsOutput(output string) []string {
	var models []string
	seen := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		clean := strings.TrimSpace(line)
		if idx := strings.LastIndex(clean, "..."); idx != -1 {
			clean = strings.TrimSpace(clean[idx+3:])
		}
		fields := strings.Fields(clean)
		if len(fields) == 0 {
			continue
		}
		candidate := fields[0]
		if reviewModelIdentifierPattern.MatchString(candidate) && !seen[candidate] {
			seen[candidate] = true
			models = append(models, candidate)
		}
	}
	return models
}

var (
	antigravityModelDiscoveryTimeout = 3 * time.Second
	antigravityModelCommand          = exec.CommandContext
)

// DiscoverAntigravityModels returns models from `agy models` if available, or DefaultAntigravityReviewModels.
func DiscoverAntigravityModels() []string {
	ctx, cancel := context.WithTimeout(context.Background(), antigravityModelDiscoveryTimeout)
	defer cancel()

	cmd := antigravityModelCommand(ctx, "agy", "models")
	out, err := cmd.Output()
	if err != nil {
		return DefaultAntigravityReviewModels
	}
	parsed := ParseAntigravityModelsOutput(string(out))
	if len(parsed) == 0 {
		return DefaultAntigravityReviewModels
	}
	return parsed
}

var (
	cachedAntigravityModels     []string
	cachedAntigravityModelsOnce sync.Once
)

// CachedAntigravityModels returns discovered models, caching the result after the first execution.
func CachedAntigravityModels() []string {
	cachedAntigravityModelsOnce.Do(func() {
		cachedAntigravityModels = DiscoverAntigravityModels()
	})
	return cachedAntigravityModels
}

// IsKnownAntigravityModel reports whether a model is in the provided or default active catalog.
func IsKnownAntigravityModel(model string, catalog []string) bool {
	if len(catalog) == 0 {
		catalog = DefaultAntigravityReviewModels
	}
	for _, m := range catalog {
		if m == model {
			return true
		}
	}
	return false
}

// LoadAllAntigravityCLIReviewRouting reads all configured reviewer routes from review-models.json.
func LoadAllAntigravityCLIReviewRouting() (map[string]AntigravityCLIReviewRouting, error) {
	path, err := AntigravityReviewModelsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return make(map[string]AntigravityCLIReviewRouting), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read antigravity-cli routing %s: %w", path, err)
	}
	var config map[string]json.RawMessage
	if json.Unmarshal(data, &config) != nil || config == nil {
		return nil, &InvalidReviewRoutingError{Role: "all", Path: path}
	}
	routes := make(map[string]AntigravityCLIReviewRouting)
	for key := range config {
		route, missing, err := ResolveAntigravityCLIReviewRouting(key)
		if err != nil {
			return nil, err
		}
		if !missing {
			routes[key] = route
		}
	}
	return routes, nil
}

// SaveAntigravityCLIReviewRouting writes reviewer routes to review-models.json.
func SaveAntigravityCLIReviewRouting(assignments map[string]AntigravityCLIReviewRouting) error {
	path, err := AntigravityReviewModelsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}
	config := make(map[string]any)
	for _, key := range CanonicalReviewRoles() {
		route, ok := assignments[key]
		if !ok {
			continue
		}
		model := strings.TrimSpace(route.Model)
		if model == "" || strings.EqualFold(model, "inherit") {
			config[key] = "inherit"
		} else {
			config[key] = model
		}
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal review routing: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// ResolveAntigravityCLIReviewRouting reads gentle-ai's read-only model
// assignment for the antigravity-cli in-process reviewer runtime without
// modifying it. The assignment file grammar is: a top-level JSON object whose
// keys are routing roles or published lenses; each entry is either a bare
// model string, or an object with a model field.
// A JSON null entry is invalid, an empty object {} means runtime defaults,
// model whitespace is trimmed, and model names must match the fixed identifier
// charset [A-Za-z0-9._~:@/+%-]. The sentinel value "inherit" resolves to the
// zero route (empty model, invoking agy without --model to inherit session defaults).
// Absence of the assignment file or of the named key resolves to the zero route,
// never an error and never written, with assignmentKeyMissing true only when the
// file exists but the key does not (the likely-typo case callers surface as a
// diagnostic); an invalid file fails closed with the typed InvalidReviewRoutingError.
// Unlike Pi, there is no repository-root fallback: the repository must not
// prescribe canonical models per role.
func ResolveAntigravityCLIReviewRouting(key string) (route AntigravityCLIReviewRouting, assignmentKeyMissing bool, err error) {
	path, err := AntigravityReviewModelsPath()
	if err != nil {
		return route, false, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return route, false, nil
	}
	if err != nil {
		return route, false, fmt.Errorf("read antigravity-cli routing %s: %w", path, err)
	}
	invalid := func() (AntigravityCLIReviewRouting, bool, error) {
		return AntigravityCLIReviewRouting{}, false, &InvalidReviewRoutingError{Role: key, Path: path}
	}
	var config map[string]json.RawMessage
	if json.Unmarshal(data, &config) != nil || config == nil {
		return invalid()
	}
	entry, exists := config[key]
	if !exists {
		return route, true, nil
	}
	var model string
	if json.Unmarshal(entry, &model) == nil {
		route.Model = strings.TrimSpace(model)
		if route.Model == "" {
			return invalid()
		}
	} else {
		var fields map[string]json.RawMessage
		if json.Unmarshal(entry, &fields) != nil || fields == nil {
			return invalid()
		}
		for field, value := range fields {
			switch field {
			case "model":
				if json.Unmarshal(value, &model) != nil {
					return invalid()
				}
				route.Model = strings.TrimSpace(model)
				if route.Model == "" {
					return invalid()
				}
			case "effort":
				// Effort is tolerated for backward compatibility with legacy configuration,
				// but is no longer validated or emitted.
			default:
				return invalid()
			}
		}
		if len(fields) > 0 && route.Model == "" {
			return invalid()
		}
	}
	if strings.EqualFold(route.Model, "inherit") {
		route.Model = ""
	}
	if route.Model != "" && !reviewModelIdentifierPattern.MatchString(route.Model) {
		return invalid()
	}
	return route, false, nil
}
