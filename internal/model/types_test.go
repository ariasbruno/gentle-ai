package model

import "testing"

func TestAgentAntigravity(t *testing.T) {
	if AgentAntigravity != "antigravity" {
		t.Errorf("AgentAntigravity = %q, want %q", AgentAntigravity, "antigravity")
	}
}

func TestAgentAntigravityCLI(t *testing.T) {
	if AgentAntigravityCLI != "antigravity-cli" {
		t.Errorf("AgentAntigravityCLI = %q, want %q", AgentAntigravityCLI, "antigravity-cli")
	}
}
