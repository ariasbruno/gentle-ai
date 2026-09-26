package screens

import (
	"testing"

	"github.com/gentleman-programming/gentle-ai/v3/internal/model"
)

func TestAgentOptionsIncludesAntigravityAndAntigravityCLI(t *testing.T) {
	options := AgentOptions()

	seenAntigravity := false
	seenAntigravityCLI := false

	for _, option := range options {
		if option == model.AgentAntigravityCLI {
			seenAntigravityCLI = true
		}
		if option == model.AgentAntigravity {
			seenAntigravity = true
		}
	}

	if !seenAntigravity {
		t.Fatal("AgentOptions() missing Antigravity option")
	}
	if !seenAntigravityCLI {
		t.Fatal("AgentOptions() missing Antigravity CLI option")
	}
}
