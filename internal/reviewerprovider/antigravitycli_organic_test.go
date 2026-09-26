//go:build organic

package reviewerprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Organic transport proof for slice 3 (antigravity-cli integration). This test
// is opt-in behind the "organic" build tag because it invokes the real agy
// binary with a real authenticated model; CI has neither, so it never runs
// there. It proves the compiled AntigravityCLIAdapter transport end-to-end:
//   - positive: a real lens-style sealed prompt reaches agy and the raw final
//     bytes are a JSON object with the review result roots.
//   - fail-closed: an unusable model selection surfaces a typed transport
//     error and no bytes, without hanging.
//
// Run with:
//
//	go test -tags organic -run TestAntigravityCLIAdapterOrganicTransport -v ./internal/reviewerprovider/
func TestAntigravityCLIAdapterOrganicTransport(t *testing.T) {
	if _, err := exec.LookPath("agy"); err != nil {
		t.Skipf("organic antigravity-cli transport proof requires real agy: %v", err)
	}

	t.Run("positive", func(t *testing.T) {
		adapter := NewAntigravityCLIAdapter()
		adapter.Model = "gemini-3.8-flash-low"
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		raw, err := adapter.Review(ctx, NewInvocation(organicAntigravityCLILensPrompt()))
		if err != nil {
			t.Fatalf("organic agy positive Review() failed: %v", err)
		}
		if len(bytes.TrimSpace(raw)) == 0 {
			t.Fatal("organic agy returned empty bytes")
		}
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("organic agy output is not one JSON object: %v\nraw: %s", err, raw)
		}
		subjectHash, _ := decoded["subject_hash"].(string)
		if !strings.HasPrefix(subjectHash, "sha256:") {
			t.Errorf("subject_hash = %q, want sha256: prefix", subjectHash)
		}
		inspection, _ := decoded["inspection"].(map[string]any)
		if inspection == nil {
			t.Errorf("inspection = %v, want object with status/paths", decoded["inspection"])
		}
		if _, okay := decoded["findings"].([]any); !okay {
			t.Errorf("findings = %T, want array", decoded["findings"])
		}
		if _, okay := decoded["evidence"].([]any); !okay {
			t.Errorf("evidence = %T, want array", decoded["evidence"])
		}
		t.Logf("organic positive output: %s", raw)
	})

	t.Run("fail_closed", func(t *testing.T) {
		adapter := NewAntigravityCLIAdapter()
		adapter.Model = "gemini-this-model-does-not-exist"
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		raw, err := adapter.Review(ctx, NewInvocation(organicAntigravityCLILensPrompt()))
		if err == nil {
			t.Fatalf("organic agy invalid model succeeded; want typed transport error, raw=%q", raw)
		}
		if raw != nil {
			t.Errorf("organic agy failure returned %d bytes; want none", len(raw))
		}
		if !strings.Contains(err.Error(), "antigravity-cli reviewer transport failed") &&
			!strings.Contains(err.Error(), "antigravity-cli reviewer transport") {
			t.Errorf("organic failure error = %v, want antigravity-cli reviewer transport failure", err)
		}
		t.Logf("organic fail-closed error: %v", err)
	})
}

// organicAntigravityCLILensPrompt is a sealed lens-style prompt: binding
// prefix, the frozen review result shape, and the expectation of one JSON
// object only. It mirrors the honest provider contract without embedding a
// real lineage.
func organicAntigravityCLILensPrompt() []byte {
	return []byte(`GENTLE_AI_REVIEW_BINDING {"lineage":"organic-slice-3-probe","target":"sha256:0000000000000000000000000000000000000000000000000000000000000000","lens":"review-risk","order":1,"revision":"sha256:0000000000000000000000000000000000000000000000000000000000000000","repository_context":"","subject_hash":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}

You are the read-only review-risk lens. Inspect the provided evidence and return exactly one JSON object with no prose, matching this schema:

{
  "subject_hash": "sha256:<the echoed subject hash>",
  "inspection": {"status": "completed", "paths": ["probe.go"]},
  "findings": [],
  "evidence": ["reviewed the complete probe scope"]
}

Native Go alone decides findings, receipts, and delivery gates.`)
}
