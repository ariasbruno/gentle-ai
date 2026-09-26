package screens

import (
	"fmt"
	"maps"
	"strings"
	"unicode"

	"github.com/gentleman-programming/gentle-ai/v3/internal/reviewerprovider"
	"github.com/gentleman-programming/gentle-ai/v3/internal/tui/styles"
)

// AntigravityReviewCustomMode represents the active sub-mode of the Custom reviewer picker flow.
type AntigravityReviewCustomMode int

const (
	AntigravityReviewCustomModeNone AntigravityReviewCustomMode = iota
	AntigravityReviewCustomModeRoleList
	AntigravityReviewCustomModeModelSelect
)

// AntigravityInheritModelOption is the display label for inheriting the CLI session model.
const AntigravityInheritModelOption = "(default / inherit)"

// AntigravityReviewRoleLabels maps canonical reviewer roles to descriptive UI labels.
var AntigravityReviewRoleLabels = map[string]string{
	reviewerprovider.ReviewRoutingKeyRisk:        "review-risk (Risk Lens)",
	reviewerprovider.ReviewRoutingKeyReadability: "review-readability (Readability Lens)",
	reviewerprovider.ReviewRoutingKeyReliability: "review-reliability (Reliability Lens)",
	reviewerprovider.ReviewRoutingKeyResilience:  "review-resilience (Resilience Lens)",
	reviewerprovider.ReviewRoutingKeyRefuter:     "review-refuter (Refuter Agent)",
	reviewerprovider.ReviewRoutingKeyValidator:   "review-validator (Validator Agent)",
}

// AntigravityReviewModelPickerState holds navigation state for the Antigravity review model picker.
type AntigravityReviewModelPickerState struct {
	Preset            reviewerprovider.AntigravityReviewPreset
	CustomAssignments map[string]reviewerprovider.AntigravityCLIReviewRouting
	CustomMode        AntigravityReviewCustomMode
	CustomRoleIdx     int
	CustomModelSearch string
	CustomModelCursor int
	AvailableModels   []string
	CustomConfirmed   bool
}

// NewAntigravityReviewModelPickerState returns the default state: Default (Inherit) preset.
func NewAntigravityReviewModelPickerState() AntigravityReviewModelPickerState {
	return AntigravityReviewModelPickerState{
		Preset:            reviewerprovider.AntigravityPresetTypeDefault,
		CustomAssignments: reviewerprovider.AntigravityPresetDefault(),
		AvailableModels:   reviewerprovider.CachedAntigravityModels(),
		CustomMode:        AntigravityReviewCustomModeNone,
	}
}

// NewAntigravityReviewModelPickerStateFromAssignments returns state initialized from existing assignments.
func NewAntigravityReviewModelPickerStateFromAssignments(assignments map[string]reviewerprovider.AntigravityCLIReviewRouting) AntigravityReviewModelPickerState {
	if len(assignments) == 0 {
		return NewAntigravityReviewModelPickerState()
	}
	for _, preset := range reviewerprovider.AntigravityPresetOrder {
		constructor, ok := reviewerprovider.AntigravityPresetConstructors[preset]
		if ok && reviewerprovider.AntigravityReviewRoutingsEqual(constructor(), assignments) {
			return AntigravityReviewModelPickerState{
				Preset:            preset,
				CustomAssignments: maps.Clone(assignments),
				AvailableModels:   reviewerprovider.CachedAntigravityModels(),
				CustomMode:        AntigravityReviewCustomModeNone,
			}
		}
	}
	return AntigravityReviewModelPickerState{
		Preset:            reviewerprovider.AntigravityPresetTypeCustom,
		CustomAssignments: maps.Clone(assignments),
		AvailableModels:   reviewerprovider.CachedAntigravityModels(),
		CustomMode:        AntigravityReviewCustomModeNone,
	}
}

func filteredAntigravityModels(state AntigravityReviewModelPickerState) []string {
	models := state.AvailableModels
	if len(models) == 0 {
		models = reviewerprovider.DefaultAntigravityReviewModels
	}
	query := strings.ToLower(strings.TrimSpace(state.CustomModelSearch))
	var filtered []string
	if query == "" || strings.Contains("(default / inherit)", query) || strings.Contains("inherit", query) || strings.Contains("default", query) {
		filtered = append(filtered, AntigravityInheritModelOption)
	}
	for _, m := range models {
		if query == "" || strings.Contains(strings.ToLower(m), query) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// AntigravityReviewModelPickerOptionCount returns selectable row count for the active mode.
func AntigravityReviewModelPickerOptionCount(state AntigravityReviewModelPickerState) int {
	switch state.CustomMode {
	case AntigravityReviewCustomModeRoleList:
		return len(reviewerprovider.CanonicalReviewRoles()) + 2 // roles + Confirm + Back
	case AntigravityReviewCustomModeModelSelect:
		models := filteredAntigravityModels(state)
		return len(models)
	}
	return len(reviewerprovider.AntigravityPresetOrder) + 1 // presets + Back
}

// HandleAntigravityReviewModelPickerNav processes a key event for the Antigravity review model picker.
func HandleAntigravityReviewModelPickerNav(
	key string,
	state *AntigravityReviewModelPickerState,
	cursor int,
) (handled bool, assignments map[string]reviewerprovider.AntigravityCLIReviewRouting) {
	if state.CustomMode != AntigravityReviewCustomModeNone {
		return handleAntigravityCustomNav(key, state, cursor)
	}

	if key != "enter" {
		return false, nil
	}

	// Back row: last option after presets.
	backIdx := len(reviewerprovider.AntigravityPresetOrder)
	if cursor >= backIdx {
		return false, nil
	}

	// Custom row: index 4.
	if cursor == len(reviewerprovider.AntigravityPresetOrder)-1 {
		state.CustomMode = AntigravityReviewCustomModeRoleList
		state.CustomRoleIdx = 0
		state.CustomModelSearch = ""
		state.CustomModelCursor = 0
		if state.CustomAssignments == nil {
			state.CustomAssignments = reviewerprovider.AntigravityPresetDefault()
		}
		return true, nil
	}

	// Preset rows.
	selected := reviewerprovider.AntigravityPresetOrder[cursor]
	state.Preset = selected
	state.CustomConfirmed = false
	constructor, ok := reviewerprovider.AntigravityPresetConstructors[selected]
	if ok {
		a := maps.Clone(constructor())
		return true, a
	}

	return false, nil
}

func handleAntigravityCustomNav(
	key string,
	state *AntigravityReviewModelPickerState,
	cursor int,
) (bool, map[string]reviewerprovider.AntigravityCLIReviewRouting) {
	switch state.CustomMode {
	case AntigravityReviewCustomModeRoleList:
		return handleAntigravityRoleListNav(key, state, cursor)
	case AntigravityReviewCustomModeModelSelect:
		return handleAntigravityModelSelectNav(key, state)
	}
	return false, nil
}

func handleAntigravityRoleListNav(
	key string,
	state *AntigravityReviewModelPickerState,
	cursor int,
) (bool, map[string]reviewerprovider.AntigravityCLIReviewRouting) {
	roles := reviewerprovider.CanonicalReviewRoles()
	roleCount := len(roles)

	switch key {
	case "esc":
		state.CustomMode = AntigravityReviewCustomModeNone
		return true, nil
	case "enter":
		// Confirm row: index roleCount
		if cursor == roleCount {
			base := reviewerprovider.AntigravityPresetDefault()
			for r, a := range state.CustomAssignments {
				base[r] = a
			}
			state.CustomConfirmed = true
			state.CustomMode = AntigravityReviewCustomModeNone
			return true, base
		}
		// Back row: index roleCount + 1
		if cursor == roleCount+1 {
			state.CustomMode = AntigravityReviewCustomModeNone
			return true, nil
		}
		// Select role -> enter model select
		if cursor < roleCount {
			state.CustomRoleIdx = cursor
			state.CustomMode = AntigravityReviewCustomModeModelSelect
			state.CustomModelSearch = ""
			state.CustomModelCursor = 0
			return true, nil
		}
	}
	return false, nil
}

func handleAntigravityModelSelectNav(
	key string,
	state *AntigravityReviewModelPickerState,
) (bool, map[string]reviewerprovider.AntigravityCLIReviewRouting) {
	models := filteredAntigravityModels(*state)
	if len(models) > 0 {
		state.CustomModelCursor = min(max(0, state.CustomModelCursor), len(models)-1)
	} else {
		state.CustomModelCursor = 0
	}

	switch key {
	case "up", "k":
		if state.CustomModelCursor > 0 {
			state.CustomModelCursor--
		}
		return true, nil
	case "down", "j":
		if state.CustomModelCursor < len(models)-1 {
			state.CustomModelCursor++
		}
		return true, nil
	case "enter":
		if len(models) == 0 {
			return true, nil
		}
		selected := models[state.CustomModelCursor]
		if selected == AntigravityInheritModelOption {
			selected = ""
		}
		roles := reviewerprovider.CanonicalReviewRoles()
		if state.CustomRoleIdx < len(roles) {
			role := roles[state.CustomRoleIdx]
			if state.CustomAssignments == nil {
				state.CustomAssignments = make(map[string]reviewerprovider.AntigravityCLIReviewRouting)
			}
			state.CustomAssignments[role] = reviewerprovider.AntigravityCLIReviewRouting{
				Model: selected,
			}
		}
		state.CustomMode = AntigravityReviewCustomModeRoleList
		state.CustomModelSearch = ""
		state.CustomModelCursor = 0
		return true, nil
	case "backspace":
		if state.CustomModelSearch != "" {
			runes := []rune(state.CustomModelSearch)
			state.CustomModelSearch = string(runes[:len(runes)-1])
			state.CustomModelCursor = 0
		}
		return true, nil
	case "ctrl+u":
		state.CustomModelSearch = ""
		state.CustomModelCursor = 0
		return true, nil
	case "esc":
		state.CustomMode = AntigravityReviewCustomModeRoleList
		state.CustomModelSearch = ""
		state.CustomModelCursor = 0
		return true, nil
	default:
		if isAntigravitySearchInput(key) {
			state.CustomModelSearch += key
			state.CustomModelCursor = 0
			return true, nil
		}
	}
	return false, nil
}

func isAntigravitySearchInput(key string) bool {
	runes := []rune(key)
	if len(runes) != 1 {
		return false
	}
	return unicode.IsPrint(runes[0]) && runes[0] != 'j' && runes[0] != 'k'
}

// RenderAntigravityReviewModelPicker renders the appropriate view based on the active sub-mode.
func RenderAntigravityReviewModelPicker(state AntigravityReviewModelPickerState, cursor int) string {
	switch state.CustomMode {
	case AntigravityReviewCustomModeRoleList:
		return renderAntigravityCustomRoleList(state, cursor)
	case AntigravityReviewCustomModeModelSelect:
		return renderAntigravityCustomModelSelect(state)
	}
	return renderAntigravityMainPicker(state, cursor)
}

func renderAntigravityMainPicker(state AntigravityReviewModelPickerState, cursor int) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Antigravity CLI Review Model Assignments"))
	b.WriteString("\n\n")
	b.WriteString(styles.SubtextStyle.Render("Choose how models are assigned to the 6 Antigravity CLI reviewer roles:"))
	b.WriteString("\n\n")

	for idx, preset := range reviewerprovider.AntigravityPresetOrder {
		isSelected := preset == state.Preset
		focused := idx == cursor
		label := string(preset)
		switch preset {
		case reviewerprovider.AntigravityPresetTypeDefault:
			label = "Default (Inherit) — use active CLI session model"
		case reviewerprovider.AntigravityPresetTypeRecommended:
			label = "Recommended"
		case reviewerprovider.AntigravityPresetTypePerformance:
			label = "Performance"
		case reviewerprovider.AntigravityPresetTypeEconomy:
			label = "Economy"
		case reviewerprovider.AntigravityPresetTypeCustom:
			label = "Custom — per-role model assignment"
		}
		b.WriteString(renderRadio(label, isSelected, focused))
		b.WriteString(styles.SubtextStyle.Render("    "+reviewerprovider.AntigravityPresetDescriptions[preset]) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(renderOptions([]string{"← Back"}, cursor-len(reviewerprovider.AntigravityPresetOrder)))
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("j/k: navigate • enter: select • esc: back"))

	return b.String()
}

func renderAntigravityCustomRoleList(state AntigravityReviewModelPickerState, cursor int) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Custom Antigravity CLI Review Models"))
	b.WriteString("\n\n")
	b.WriteString(styles.SubtextStyle.Render("Assign a model to each of the 6 reviewer roles. Deprecated or unknown models are flagged."))
	b.WriteString("\n\n")

	roles := reviewerprovider.CanonicalReviewRoles()
	catalog := state.AvailableModels
	if len(catalog) == 0 {
		catalog = reviewerprovider.DefaultAntigravityReviewModels
	}

	for idx, role := range roles {
		focused := idx == cursor
		assignment, ok := state.CustomAssignments[role]
		var modelInfo string
		if ok && assignment.Model != "" && !strings.EqualFold(assignment.Model, "inherit") {
			known := reviewerprovider.IsKnownAntigravityModel(assignment.Model, catalog)
			if !known {
				modelInfo = styles.WarningStyle.Render(fmt.Sprintf("%s [DEPRECATED / UNKNOWN]", assignment.Model))
			} else {
				modelInfo = styles.SuccessStyle.Render(assignment.Model)
			}
		} else {
			modelInfo = styles.SubtextStyle.Render("(default / inherit)")
		}

		label := fmt.Sprintf("%-36s %s", AntigravityReviewRoleLabels[role], modelInfo)
		if focused {
			b.WriteString(styles.SelectedStyle.Render(styles.Cursor+label) + "\n")
		} else {
			b.WriteString(styles.UnselectedStyle.Render("  "+label) + "\n")
		}
	}

	b.WriteString("\n")
	actionCursor := cursor - len(roles)
	b.WriteString(renderOptions([]string{"Confirm assignments", "← Back to presets"}, actionCursor))
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("j/k: navigate • enter: assign/confirm • esc: back to presets"))

	return b.String()
}

func renderAntigravityCustomModelSelect(state AntigravityReviewModelPickerState) string {
	var b strings.Builder

	roles := reviewerprovider.CanonicalReviewRoles()
	roleLabel := "reviewer"
	if state.CustomRoleIdx < len(roles) {
		roleLabel = AntigravityReviewRoleLabels[roles[state.CustomRoleIdx]]
	}

	b.WriteString(styles.TitleStyle.Render(fmt.Sprintf("Select model for %s:", roleLabel)))
	b.WriteString("\n\n")
	searchQuery := state.CustomModelSearch
	if searchQuery == "" {
		searchQuery = "_"
	} else {
		searchQuery = searchQuery + "_"
	}
	b.WriteString(styles.SubtextStyle.Render("Search: " + searchQuery))
	b.WriteString("\n\n")

	models := filteredAntigravityModels(state)
	cursor := state.CustomModelCursor
	if cursor >= len(models) && len(models) > 0 {
		cursor = len(models) - 1
	}

	if len(models) == 0 {
		b.WriteString(styles.WarningStyle.Render("  No models match your search."))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("type: search • backspace: delete • ctrl+u: clear • esc: back"))
		return b.String()
	}

	for i, m := range models {
		focused := i == cursor
		if focused {
			b.WriteString(styles.SelectedStyle.Render(styles.Cursor+m) + "\n")
		} else {
			b.WriteString(styles.UnselectedStyle.Render("  "+m) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("j/k: navigate • type: search • backspace: delete • ctrl+u: clear • enter: select • esc: back"))

	return b.String()
}
