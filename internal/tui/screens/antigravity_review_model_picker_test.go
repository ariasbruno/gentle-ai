package screens

import (
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v3/internal/reviewerprovider"
)

func TestNewAntigravityReviewModelPickerState(t *testing.T) {
	state := NewAntigravityReviewModelPickerState()
	if state.Preset != reviewerprovider.AntigravityPresetTypeDefault {
		t.Errorf("Preset = %v, want %v", state.Preset, reviewerprovider.AntigravityPresetTypeDefault)
	}
	if state.CustomMode != AntigravityReviewCustomModeNone {
		t.Errorf("CustomMode = %v, want None", state.CustomMode)
	}
	if len(state.CustomAssignments) != 6 {
		t.Errorf("CustomAssignments len = %d, want 6", len(state.CustomAssignments))
	}
}

func TestNewAntigravityReviewModelPickerStateFromAssignments(t *testing.T) {
	// Empty -> Default (Inherit)
	s1 := NewAntigravityReviewModelPickerStateFromAssignments(nil)
	if s1.Preset != reviewerprovider.AntigravityPresetTypeDefault {
		t.Errorf("empty assignments preset = %v, want Default", s1.Preset)
	}

	// Recommended preset
	rec := reviewerprovider.AntigravityPresetRecommended()
	sRec := NewAntigravityReviewModelPickerStateFromAssignments(rec)
	if sRec.Preset != reviewerprovider.AntigravityPresetTypeRecommended {
		t.Errorf("recommended assignments preset = %v, want Recommended", sRec.Preset)
	}

	// Performance preset
	perf := reviewerprovider.AntigravityPresetPerformance()
	s2 := NewAntigravityReviewModelPickerStateFromAssignments(perf)
	if s2.Preset != reviewerprovider.AntigravityPresetTypePerformance {
		t.Errorf("performance assignments preset = %v, want Performance", s2.Preset)
	}

	// Custom assignments
	custom := map[string]reviewerprovider.AntigravityCLIReviewRouting{
		reviewerprovider.ReviewRoutingKeyRisk: {Model: "custom-model"},
	}
	s3 := NewAntigravityReviewModelPickerStateFromAssignments(custom)
	if s3.Preset != reviewerprovider.AntigravityPresetTypeCustom {
		t.Errorf("custom assignments preset = %v, want Custom", s3.Preset)
	}
}

func TestAntigravityReviewModelPickerOptionCount(t *testing.T) {
	state := NewAntigravityReviewModelPickerState()
	// Main preset: 5 presets + 1 Back = 6
	if count := AntigravityReviewModelPickerOptionCount(state); count != 6 {
		t.Errorf("main option count = %d, want 6", count)
	}

	state.CustomMode = AntigravityReviewCustomModeRoleList
	// Role list: 6 roles + Confirm + Back = 8
	if count := AntigravityReviewModelPickerOptionCount(state); count != 8 {
		t.Errorf("role list option count = %d, want 8", count)
	}
}

func TestHandleAntigravityPresetNav(t *testing.T) {
	state := NewAntigravityReviewModelPickerState()

	// Enter on Default (cursor 0)
	handled, assignments := HandleAntigravityReviewModelPickerNav("enter", &state, 0)
	if !handled || assignments == nil {
		t.Fatal("expected enter on Default to be handled with assignments")
	}
	if !reviewerprovider.AntigravityReviewRoutingsEqual(assignments, reviewerprovider.AntigravityPresetDefault()) {
		t.Error("assignments did not match Default preset")
	}

	// Enter on Recommended (cursor 1)
	handled, assignments = HandleAntigravityReviewModelPickerNav("enter", &state, 1)
	if !handled || assignments == nil {
		t.Fatal("expected enter on Recommended to be handled with assignments")
	}
	if !reviewerprovider.AntigravityReviewRoutingsEqual(assignments, reviewerprovider.AntigravityPresetRecommended()) {
		t.Error("assignments did not match Recommended preset")
	}

	// Enter on Performance (cursor 2)
	handled, assignments = HandleAntigravityReviewModelPickerNav("enter", &state, 2)
	if !handled || assignments == nil {
		t.Fatal("expected enter on Performance to be handled with assignments")
	}
	if !reviewerprovider.AntigravityReviewRoutingsEqual(assignments, reviewerprovider.AntigravityPresetPerformance()) {
		t.Error("assignments did not match Performance preset")
	}

	// Enter on Custom (cursor 4) -> enters custom mode
	handled, assignments = HandleAntigravityReviewModelPickerNav("enter", &state, 4)
	if !handled || assignments != nil {
		t.Fatal("expected enter on Custom to return handled=true and nil assignments")
	}
	if state.CustomMode != AntigravityReviewCustomModeRoleList {
		t.Errorf("CustomMode = %v, want RoleList", state.CustomMode)
	}

	// Back row (cursor 5) -> unhandled so parent confirmSelection handles Back
	state.CustomMode = AntigravityReviewCustomModeNone
	handled, assignments = HandleAntigravityReviewModelPickerNav("enter", &state, 5)
	if handled || assignments != nil {
		t.Fatal("expected enter on Back row to be unhandled")
	}
}

func TestHandleAntigravityCustomNavFlow(t *testing.T) {
	state := NewAntigravityReviewModelPickerState()
	// Enter Custom mode (cursor 4)
	HandleAntigravityReviewModelPickerNav("enter", &state, 4)

	// In RoleList: select first role (review-risk, cursor 0)
	handled, _ := HandleAntigravityReviewModelPickerNav("enter", &state, 0)
	if !handled || state.CustomMode != AntigravityReviewCustomModeModelSelect {
		t.Fatalf("expected transition to ModelSelect, got %v", state.CustomMode)
	}

	// In ModelSelect: search for "gemini-3.8"
	HandleAntigravityReviewModelPickerNav("g", &state, 0)
	HandleAntigravityReviewModelPickerNav("e", &state, 0)
	HandleAntigravityReviewModelPickerNav("m", &state, 0)
	if state.CustomModelSearch != "gem" {
		t.Errorf("CustomModelSearch = %q, want 'gem'", state.CustomModelSearch)
	}

	// Clear search with ctrl+u
	HandleAntigravityReviewModelPickerNav("ctrl+u", &state, 0)
	if state.CustomModelSearch != "" {
		t.Errorf("CustomModelSearch after ctrl+u = %q, want empty", state.CustomModelSearch)
	}

	// Select first option in list (which is "(default / inherit)") -> directly assigns empty model and returns to RoleList
	HandleAntigravityReviewModelPickerNav("enter", &state, 0)
	if state.CustomMode != AntigravityReviewCustomModeRoleList {
		t.Fatalf("expected return to RoleList, got %v", state.CustomMode)
	}
	assigned := state.CustomAssignments[reviewerprovider.ReviewRoutingKeyRisk]
	if assigned.Model != "" {
		t.Errorf("assigned model = %q, want empty for inherit", assigned.Model)
	}

	// Enter ModelSelect again for first role and pick cursor 1 (a concrete model)
	HandleAntigravityReviewModelPickerNav("enter", &state, 0)
	state.CustomModelCursor = 1
	HandleAntigravityReviewModelPickerNav("enter", &state, 0)
	assigned = state.CustomAssignments[reviewerprovider.ReviewRoutingKeyRisk]
	if assigned.Model == "" {
		t.Errorf("assigned model is empty, want concrete model")
	}

	// In RoleList: select Confirm (cursor 6)
	handled, finalAssignments := HandleAntigravityReviewModelPickerNav("enter", &state, 6)
	if !handled || finalAssignments == nil {
		t.Fatal("expected Confirm to return handled=true with assignments")
	}
	if !state.CustomConfirmed {
		t.Error("CustomConfirmed should be true")
	}
	if finalAssignments[reviewerprovider.ReviewRoutingKeyRisk].Model != assigned.Model {
		t.Errorf("final assignment model = %q, want %q", finalAssignments[reviewerprovider.ReviewRoutingKeyRisk].Model, assigned.Model)
	}
}

func TestRenderAntigravityReviewModelPicker(t *testing.T) {
	state := NewAntigravityReviewModelPickerState()

	// Main preset render
	out := RenderAntigravityReviewModelPicker(state, 0)
	if !strings.Contains(out, "Antigravity CLI Review Model Assignments") {
		t.Errorf("rendered main output missing title: %s", out)
	}

	// Custom role list render
	state.CustomMode = AntigravityReviewCustomModeRoleList
	out = RenderAntigravityReviewModelPicker(state, 0)
	if !strings.Contains(out, "Custom Antigravity CLI Review Models") {
		t.Errorf("rendered custom role list missing title: %s", out)
	}

	// Deprecated model warning
	state.CustomAssignments[reviewerprovider.ReviewRoutingKeyRisk] = reviewerprovider.AntigravityCLIReviewRouting{
		Model: "deprecated-model-xyz",
	}
	out = RenderAntigravityReviewModelPicker(state, 0)
	if !strings.Contains(out, "DEPRECATED / UNKNOWN") {
		t.Errorf("rendered output missing deprecated warning badge: %s", out)
	}

	// Model select render
	state.CustomMode = AntigravityReviewCustomModeModelSelect
	out = RenderAntigravityReviewModelPicker(state, 0)
	if !strings.Contains(out, "Select model") {
		t.Errorf("rendered model select missing title: %s", out)
	}
}
