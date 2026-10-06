package cli

import (
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
	"github.com/gentleman-programming/gentle-ai/v4/internal/pipeline"
	"github.com/gentleman-programming/gentle-ai/v4/internal/planner"
)

// The Antigravity CLI integration is wired into shared upstream files through
// one-line hooks inside installRuntime.stagePlan (run.go) and
// syncRuntime.stagePlan (sync.go). Losing those hooks during an upstream sync
// does not break compilation or any other test: `agy plugin install` silently
// stops deploying the plugin tree, native subagents, and hooks. That exact loss
// shipped twice. These tests are the tripwire: red here after a sync means the
// loss was caught, not that the guard is wrong.

func TestStagePlansKeepAntigravityCLIPluginImportStep(t *testing.T) {
	const (
		installStepID = "install:antigravity-cli:plugin-import"
		syncStepID    = "sync:antigravity-cli:plugin-import"
	)
	agySelection := model.Selection{Agents: []model.AgentID{model.AgentAntigravityCLI}}
	nonAgySelection := model.Selection{Agents: []model.AgentID{model.AgentClaudeCode}}

	tests := []struct {
		name    string
		plan    pipeline.StagePlan
		wantID  string
		wantHit bool
	}{
		{
			name:    "install with antigravity-cli selected imports the plugin",
			plan:    (&installRuntime{selection: agySelection, resolved: planner.ResolvedPlan{Agents: agySelection.Agents}, state: &runtimeState{}}).stagePlan(),
			wantID:  installStepID,
			wantHit: true,
		},
		{
			name:    "install with antigravity-cli selected under workspace scope imports the plugin",
			plan:    (&installRuntime{selection: agySelection, resolved: planner.ResolvedPlan{Agents: agySelection.Agents}, scope: ScopeWorkspace, workspaceDir: "/tmp/ws", state: &runtimeState{}}).stagePlan(),
			wantID:  installStepID,
			wantHit: true,
		},
		{
			name:    "sync with antigravity-cli selected imports the plugin",
			plan:    (&syncRuntime{selection: agySelection, agentIDs: agySelection.Agents, scope: ScopeGlobal, state: &runtimeState{}}).stagePlan(),
			wantID:  syncStepID,
			wantHit: true,
		},
		{
			name:    "sync with antigravity-cli selected under workspace scope imports the plugin",
			plan:    (&syncRuntime{selection: agySelection, agentIDs: agySelection.Agents, scope: ScopeWorkspace, workspaceDir: "/tmp/ws", state: &runtimeState{}}).stagePlan(),
			wantID:  syncStepID,
			wantHit: true,
		},
		{
			name:    "install without antigravity-cli plans no plugin import",
			plan:    (&installRuntime{selection: nonAgySelection, resolved: planner.ResolvedPlan{Agents: nonAgySelection.Agents}, state: &runtimeState{}}).stagePlan(),
			wantID:  installStepID,
			wantHit: false,
		},
		{
			name:    "sync without antigravity-cli plans no plugin import",
			plan:    (&syncRuntime{selection: nonAgySelection, agentIDs: nonAgySelection.Agents, scope: ScopeGlobal, state: &runtimeState{}}).stagePlan(),
			wantID:  syncStepID,
			wantHit: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for _, step := range tt.plan.Apply {
				if step.ID() == tt.wantID {
					found = true
					break
				}
			}
			if found != tt.wantHit {
				t.Errorf("stagePlan plugin-import step %q: found=%v, want=%v — the antigravity-cli hook inside the shared plan file was lost or duplicated", tt.wantID, found, tt.wantHit)
			}
		})
	}
}
