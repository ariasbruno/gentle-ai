package reviewerprovider

import (
	"testing"

	"github.com/gentleman-programming/gentle-ai/v3/internal/model"
)

// TestAntigravityCLIReviewRuntimeBoundaryIsFrozen pins the antigravity-cli
// review runtime boundary at the compiled state the antigravity-cli
// integration shipped: AGY is an admitted RDD review runtime whose sandboxed
// agy adapter is compiled in-process — RegisteredRuntime must be true and
// CapturesInProcess must be true — and since the slice-4 flip its manifest
// advertisement also holds, so review invocations admit it.
//
// This is the regression lock for the review transport design recorded in
// docs/antigravity-cli-integration/antigravity-cli-architecture.md section 11:
// the advertisement flip lands in a later slice deliberately, never silently.
// Any accidental change to these facts fails here before it can reach the
// capability manifest or the provider contract bundle.
func TestAntigravityCLIReviewRuntimeBoundaryIsFrozen(t *testing.T) {
	t.Parallel()

	if !RegisteredRuntime(model.AgentAntigravityCLI) {
		t.Fatal("RegisteredRuntime(antigravity-cli) = false, want true: AGY must enter the closed review runtime list once its adapter is compiled")
	}
	if !CapturesInProcess(model.AgentAntigravityCLI) {
		t.Fatal("CapturesInProcess(antigravity-cli) = false, want true: AGY's sandboxed agy adapter is compiled in-process")
	}

	// The five runtimes with real transports must stay admitted so this pin
	// cannot pass vacuously if the closed list ever shrinks.
	for _, agent := range []model.AgentID{model.AgentClaudeCode, model.AgentCodex, model.AgentOpenCode, model.AgentPi, model.AgentAntigravityCLI} {
		if !RegisteredRuntime(agent) {
			t.Fatalf("RegisteredRuntime(%s) = false, want true: closed runtime list lost an admitted runtime", agent)
		}
	}
}
