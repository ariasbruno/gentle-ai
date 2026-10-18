package filemerge

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// These cases pin the merge behavior around permission lists beyond the two
// defects the scope fix repaired. They exist because an independent verifier
// exercised exactly these inputs against the pre-fix commit: each row records
// what the merge does now, and the shape the fix was introduced for.
func TestMergeJSONObjectsPermissionListEdgeCases(t *testing.T) {
	cases := []struct {
		name    string
		base    string
		overlay string
		path    []string
		want    []any
	}{
		{
			name:    "bare deny drops a duplicated base value",
			base:    `{"deny":["a","a"]}`,
			overlay: `{"deny":["a"]}`,
			path:    []string{"deny"},
			want:    []any{"a"},
		},
		{
			name:    "nested non-permission deny keeps its siblings",
			base:    `{"tools":{"deny":["ours"],"keep":1}}`,
			overlay: `{"tools":{"deny":["replacement"]}}`,
			path:    []string{"tools", "deny"},
			want:    []any{"replacement"},
		},
		{
			name:    "permission allow unions when deny is absent",
			base:    `{"permissions":{"allow":["user rule"]}}`,
			overlay: `{"permissions":{"allow":["ours"]}}`,
			path:    []string{"permissions", "allow"},
			want:    []any{"user rule", "ours"},
		},
		{
			name:    "scalar deny is replaced by the overlay list",
			base:    `{"deny":"all"}`,
			overlay: `{"deny":["replacement"]}`,
			path:    []string{"deny"},
			want:    []any{"replacement"},
		},
		{
			name:    "object deny is replaced by the overlay list",
			base:    `{"deny":{"nested":true}}`,
			overlay: `{"deny":["replacement"]}`,
			path:    []string{"deny"},
			want:    []any{"replacement"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			merged, err := MergeJSONObjects([]byte(tc.base), []byte(tc.overlay))
			if err != nil {
				t.Fatalf("MergeJSONObjects() error = %v", err)
			}
			if got := listAtPath(t, merged, tc.path); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("merged %v = %#v, want %#v", tc.path, got, tc.want)
			}
		})
	}
}

// A scalar restriction in the base still wins over an overlay object. The
// protect rule predates the scope fix and keeps a generated overlay from
// replacing an ordered rule object with something looser.
func TestMergeJSONObjectsKeepsScalarPermissionRestriction(t *testing.T) {
	merged, err := MergeJSONObjects([]byte(`{"permissions":"deny"}`), []byte(`{"permissions":{"deny":["ours"]}}`))
	if err != nil {
		t.Fatalf("MergeJSONObjects() error = %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(merged, &root); err != nil {
		t.Fatalf("unmarshal merged json: %v\n%s", err, merged)
	}
	if got := root["permissions"]; !reflect.DeepEqual(got, "deny") {
		t.Fatalf(`permissions = %#v, want the base scalar "deny"`, got)
	}
}

// A malformed overlay must fail loudly instead of degrading into a silent
// partial merge: a caller that cannot be parsed is a bug in the caller.
func TestMergeJSONObjectsRejectsMalformedOverlay(t *testing.T) {
	_, err := MergeJSONObjects([]byte(`{"deny":["a"]}`), []byte(`{"deny":[`))
	if err == nil {
		t.Fatal("MergeJSONObjects() accepted a malformed overlay")
	}
	if !strings.Contains(err.Error(), "unmarshal overlay json") {
		t.Fatalf("MergeJSONObjects() error = %v, want an overlay unmarshal error", err)
	}
}

// A malformed base is still merged as an overlay-only document rather than
// failing: that is the long-standing behavior for a file the user has broken by
// hand, and the scope fix did not change it.
func TestMergeJSONObjectsOverlaysMalformedBase(t *testing.T) {
	merged, err := MergeJSONObjects([]byte(`{`), []byte(`{"deny":["ours"]}`))
	if err != nil {
		t.Fatalf("MergeJSONObjects() error = %v", err)
	}
	if got := listAtPath(t, merged, []string{"deny"}); !reflect.DeepEqual(got, []any{"ours"}) {
		t.Fatalf("bare deny = %#v, want the overlay list", got)
	}
}
