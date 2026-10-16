package statusline

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Density selects how many chrome elements the line renders.
type Density string

const (
	DensityComfortable Density = "comfortable"
	DensityCompact     Density = "compact"
	DensityMinimal     Density = "minimal"
)

// Visibility toggles individual segment groups, mirroring the Gentle Shell
// visual customization policy.
type Visibility struct {
	Changes        bool
	RDD            bool
	Agents         bool
	UsageCost      bool
	ModelDetails   bool
	ContextWarning bool
}

// Config is the user-owned status line configuration loaded from
// ~/.gentle-ai/statusline.json. The file is never written by gentle-ai.
type Config struct {
	Density    Density
	Visibility Visibility
}

// DefaultConfig returns the shipped defaults: comfortable density, every
// segment visible except the opt-in changes segment.
func DefaultConfig() Config {
	return Config{
		Density: DensityComfortable,
		Visibility: Visibility{
			Changes:        false,
			RDD:            true,
			Agents:         true,
			UsageCost:      true,
			ModelDetails:   true,
			ContextWarning: true,
		},
	}
}

// configFile mirrors the on-disk schema. Unknown keys are ignored; a missing
// or malformed file falls back to the defaults and never fails the render.
type configFile struct {
	Schema     string `json:"schema"`
	Density    string `json:"density"`
	Visibility *struct {
		Changes        *bool `json:"changes"`
		RDD            *bool `json:"rdd"`
		Agents         *bool `json:"agents"`
		UsageCost      *bool `json:"usageCost"`
		ModelDetails   *bool `json:"modelDetails"`
		ContextWarning *bool `json:"contextWarning"`
	} `json:"visibility"`
}

// ConfigPath returns the user-owned config location for a home directory.
func ConfigPath(homeDir string) string {
	return filepath.Join(homeDir, ".gentle-ai", "statusline.json")
}

// LoadConfig reads the user config; every absent or malformed field keeps
// its default. It never returns an error: a broken config degrades, it never
// breaks the status line.
func LoadConfig(homeDir string) Config {
	config := DefaultConfig()
	raw, err := os.ReadFile(ConfigPath(homeDir))
	if err != nil {
		return config
	}
	var file configFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return config
	}
	switch file.Density {
	case string(DensityComfortable), string(DensityCompact), string(DensityMinimal):
		config.Density = Density(file.Density)
	}
	if file.Visibility != nil {
		if file.Visibility.Changes != nil {
			config.Visibility.Changes = *file.Visibility.Changes
		}
		if file.Visibility.RDD != nil {
			config.Visibility.RDD = *file.Visibility.RDD
		}
		if file.Visibility.Agents != nil {
			config.Visibility.Agents = *file.Visibility.Agents
		}
		if file.Visibility.UsageCost != nil {
			config.Visibility.UsageCost = *file.Visibility.UsageCost
		}
		if file.Visibility.ModelDetails != nil {
			config.Visibility.ModelDetails = *file.Visibility.ModelDetails
		}
		if file.Visibility.ContextWarning != nil {
			config.Visibility.ContextWarning = *file.Visibility.ContextWarning
		}
	}
	return config
}
