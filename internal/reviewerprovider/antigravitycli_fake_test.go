package reviewerprovider

// Deterministic CI transport proof for the antigravity-cli reviewer adapter
// (advisory finding R3 from review-1ca9a96860c75583). Unlike the organic
// proof in antigravitycli_organic_test.go (behind the "organic" build tag),
// this file runs in plain CI with no real agy: a fake POSIX sh executable
// stands in for the binary and self-checks the adapter's sandboxed argv
// contract and non-empty sealed stdin. Expected routing values arrive as the
// FAKE_AGY_EXPECTED_MODEL and FAKE_AGY_EXPECTED_EFFORT environment variables
// rather than being interpolated into the script; the fake then echoes the
// prompt-kernel subject_hash so the round-trip is asserted end-to-end without
// any external runtime.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeAntigravityCLIScript is the POSIX sh body of the fake agy executable. It
// runs under the adapter's minimal PATH+HOME environment, so it relies only on
// POSIX utilities (`cat`, `sed`, `printf`) and builtins. It self-checks that
// the invocation carries --sandbox and --output-format text, requires a
// non-empty sealed prompt on stdin, and answers with exactly one lens-shaped
// JSON object echoing the binding's subject_hash.
const fakeAntigravityCLIScript = `#!/bin/sh
expected_model="$FAKE_AGY_EXPECTED_MODEL"

sandbox_seen=0
output_format_seen=0
model_seen=0
model_value=""
previous=""

for argument in "$@"; do
  if [ "$argument" = "--sandbox" ]; then
    sandbox_seen=1
  fi
  if [ "$previous" = "--output-format" ] && [ "$argument" = "text" ]; then
    output_format_seen=1
  fi
  if [ "$previous" = "--model" ]; then
    model_seen=1
    model_value="$argument"
  fi
  previous="$argument"
done

if [ "$sandbox_seen" -ne 1 ] || [ "$output_format_seen" -ne 1 ]; then
  echo "fake agy: invocation must carry --sandbox and --output-format text" >&2
  exit 3
fi
if [ -n "$expected_model" ]; then
  if [ "$model_seen" -ne 1 ] || [ "$model_value" != "$expected_model" ]; then
    echo "fake agy: expected --model $expected_model in argv" >&2
    exit 4
  fi
elif [ "$model_seen" -eq 1 ]; then
  echo "fake agy: unexpected --model in argv" >&2
  exit 5
fi

stdin=$(cat)
if [ -z "$stdin" ]; then
  echo "fake agy: empty sealed prompt on stdin" >&2
  exit 8
fi
subject_hash=$(printf '%s\n' "$stdin" | sed -n '1s/.*"subject_hash": *"\([^"]*\)".*/\1/p')
if [ -z "$subject_hash" ]; then
  echo "fake agy: cannot extract subject_hash from the prompt binding" >&2
  exit 9
fi

printf '{"subject_hash":"%s","inspection":{"status":"completed","paths":["probe.go"]},"findings":[],"evidence":["reviewed the complete probe scope"]}\n' "$subject_hash"
exit 0
`

// fakeAntigravityCLITargetHash is distinct from the organic proof's all-zero
// hash so the echo assertion proves extraction, not a constant.
const fakeAntigravityCLITargetHash = "sha256:1111111111111111111111111111111111111111111111111111111111111111"

func TestAntigravityCLIAdapterFakeRoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake agy executable is a POSIX sh script")
	}
	for _, test := range []struct {
		name  string
		model string
	}{
		{name: "sandboxed invocation without routing", model: ""},
		{name: "model routing", model: "fake-model"},
	} {
		t.Run(test.name, func(t *testing.T) {
			adapter := &AntigravityCLIAdapter{
				Model:    test.model,
				Env:      []string{"FAKE_AGY_EXPECTED_MODEL=" + test.model},
				LookPath: func(string) (string, error) { return writeFakeAntigravityCLI(t), nil },
			}
			raw, err := adapter.Review(context.Background(), NewInvocation(fakeAntigravityCLILensPrompt(fakeAntigravityCLITargetHash)))
			if err != nil {
				t.Fatalf("Review() with fake agy failed: %v", err)
			}
			decoded := assertFakeAntigravityCLISingleJSONObject(t, raw)
			if got, _ := decoded["subject_hash"].(string); got != fakeAntigravityCLITargetHash {
				t.Errorf("subject_hash = %q, want echoed input %q", got, fakeAntigravityCLITargetHash)
			}
			inspection, _ := decoded["inspection"].(map[string]any)
			if inspection == nil || inspection["status"] != "completed" {
				t.Errorf("inspection = %v, want object with status \"completed\"", decoded["inspection"])
			}
			paths, _ := inspection["paths"].([]any)
			if len(paths) != 1 || paths[0] != "probe.go" {
				t.Errorf("inspection.paths = %v, want [probe.go]", inspection["paths"])
			}
			if _, okay := decoded["findings"].([]any); !okay {
				t.Errorf("findings = %T, want array root", decoded["findings"])
			}
			if _, okay := decoded["evidence"].([]any); !okay {
				t.Errorf("evidence = %T, want array root", decoded["evidence"])
			}
		})
	}
}

func TestAntigravityCLIAdapterFakeFailsClosedOnMissingBinary(t *testing.T) {
	adapter := &AntigravityCLIAdapter{LookPath: func(string) (string, error) { return "", errors.New("agy not installed") }}
	raw, err := adapter.Review(context.Background(), NewInvocation([]byte("sealed prompt")))
	if err == nil || raw != nil || !strings.Contains(err.Error(), "antigravity-cli reviewer transport unavailable") {
		t.Fatalf("Review() = %q, %v; want typed transport-unavailable error and no bytes", raw, err)
	}
}

// writeFakeAntigravityCLI writes the POSIX sh fake agy executable into a fresh
// temp dir with mode 0700 and returns its path. Each caller supplies its own
// t so subtests keep isolated binaries; expected routing values arrive via the
// FAKE_AGY_EXPECTED_MODEL and FAKE_AGY_EXPECTED_EFFORT environment variables
// so the script self-checks the exact argv contract.
func writeFakeAntigravityCLI(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agy")
	if err := os.WriteFile(path, []byte(fakeAntigravityCLIScript), 0o700); err != nil {
		t.Fatalf("write fake agy executable: %v", err)
	}
	return path
}

// assertFakeAntigravityCLISingleJSONObject decodes raw as exactly one JSON
// object and returns it, failing on non-object JSON or trailing content.
func assertFakeAntigravityCLISingleJSONObject(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	if len(bytes.TrimSpace(raw)) == 0 {
		t.Fatal("fake agy returned empty bytes")
	}
	decoded := map[string]any{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&decoded); err != nil {
		t.Fatalf("fake agy output is not one JSON object: %v\nraw: %s", err, raw)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("fake agy output has trailing content after the JSON object: %v\nraw: %s", err, raw)
	}
	return decoded
}

// fakeAntigravityCLILensPrompt mirrors organicAntigravityCLILensPrompt with a
// caller-chosen subject hash: binding prefix, frozen review result shape, and
// the expectation of exactly one JSON object.
func fakeAntigravityCLILensPrompt(subjectHash string) []byte {
	return []byte(fmt.Sprintf(`GENTLE_AI_REVIEW_BINDING {"lineage":"fake-slice-4-round-trip","target":"sha256:0000000000000000000000000000000000000000000000000000000000000000","lens":"review-risk","order":1,"revision":"sha256:0000000000000000000000000000000000000000000000000000000000000000","repository_context":"","subject_hash":"%s"}

You are the read-only review-risk lens. Inspect the provided evidence and return exactly one JSON object with no prose, matching this schema:

{
  "subject_hash": "sha256:<the echoed subject hash>",
  "inspection": {"status": "completed", "paths": ["probe.go"]},
  "findings": [],
  "evidence": ["reviewed the complete probe scope"]
}

Native Go alone decides findings, receipts, and delivery gates.`, subjectHash))
}
