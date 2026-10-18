package persona

import (
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/assets"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
)

// The Antigravity CLI harness does not inject an `<available_skills>` block, so
// agy cannot receive the shared generic persona wording: it would tell the model
// to consult a block that never appears. Verified on the real agy binary, the
// runtime-agnostic wording makes the model discover skills through its own
// mechanism and read SKILL.md before answering.
//
// agy therefore owns both persona variants. These tests pin that split in both
// directions: agy never falls back to the generic asset, and no other agent is
// moved onto agy's asset.
func TestPersonaContentAntigravityCLIUsesItsOwnRuntimeAgnosticAssets(t *testing.T) {
	genericGentleman := assets.MustRead("generic/persona-gentleman.md")
	genericNeutral := assets.MustRead("generic/persona-neutral.md")
	agyGentleman := personaContent(model.AgentAntigravityCLI, model.PersonaGentleman, false)
	agyNeutral := personaContent(model.AgentAntigravityCLI, model.PersonaNeutral, false)

	if agyGentleman == "" || agyNeutral == "" {
		t.Fatal("personaContent(antigravity-cli) returned an empty persona")
	}
	if agyGentleman == genericGentleman {
		t.Fatal("antigravity-cli gentleman persona is still the generic asset")
	}
	if agyNeutral == genericNeutral {
		t.Fatal("antigravity-cli neutral persona is still the generic asset")
	}

	for name, content := range map[string]string{
		"gentleman": agyGentleman,
		"neutral":   agyNeutral,
	} {
		if strings.Contains(content, "<available_skills>") {
			t.Errorf("antigravity-cli %s persona still tells the model to read an <available_skills> block, which the agy harness never injects", name)
		}
		if !strings.Contains(content, "your runtime's skill mechanism") {
			t.Errorf("antigravity-cli %s persona lost the runtime-agnostic skill discovery wording", name)
		}
		if !strings.Contains(content, "Contextual Skill Loading") {
			t.Errorf("antigravity-cli %s persona lost the Contextual Skill Loading section", name)
		}
	}

	// agy is delivered by StrategyAppendToFile and has no residual channel, so
	// its gentleman persona must stay a full persona rather than a slim variant.
	if !strings.Contains(agyGentleman, "Senior Architect") {
		t.Error("antigravity-cli gentleman persona is not a full persona: missing the tone content every non-residual agent keeps")
	}

	// The generic assets are shared and must stay untouched: other agents still
	// receive them byte for byte. This is what keeps the agy carve-out from
	// silently rewording the persona of every other runtime.
	for _, agent := range []model.AgentID{model.AgentCodex, model.AgentGeminiCLI, model.AgentWindsurf} {
		if got := personaContent(agent, model.PersonaGentleman, false); got != genericGentleman {
			t.Errorf("personaContent(%q, gentleman) is no longer the generic asset", agent)
		}
		if got := personaContent(agent, model.PersonaNeutral, false); got != genericNeutral {
			t.Errorf("personaContent(%q, neutral) is no longer the generic asset", agent)
		}
	}
}
