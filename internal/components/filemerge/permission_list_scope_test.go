package filemerge

import (
	"encoding/json"
	"reflect"
	"testing"
)

// unionedPermissionLists are the list keys whose merge is a union rather than a
// replacement: an overlay adds restrictions on top of whatever the user already
// had, so merging must never erase an existing deny. That is only true while the
// list belongs to a permission object. A key named "deny", "ask" or "allow"
// anywhere else is ordinary data, and merging it as a union silently kept values
// the overlay had deliberately dropped.
func TestMergeJSONObjectsOnlyUnionsPermissionScopedLists(t *testing.T) {
	cases := []struct {
		name    string
		base    string
		overlay string
		path    []string
		want    []any
	}{
		{
			name:    "bare top-level deny is replaced",
			base:    `{"deny":["ours"]}`,
			overlay: `{"deny":["replacement"]}`,
			path:    []string{"deny"},
			want:    []any{"replacement"},
		},
		{
			name:    "non-permission nested deny is replaced",
			base:    `{"tools":{"deny":["ours"]}}`,
			overlay: `{"tools":{"deny":["replacement"]}}`,
			path:    []string{"tools", "deny"},
			want:    []any{"replacement"},
		},
		{
			name:    "plural permissions deny unions",
			base:    `{"permissions":{"deny":["user rule"]}}`,
			overlay: `{"permissions":{"deny":["ours"]}}`,
			path:    []string{"permissions", "deny"},
			want:    []any{"user rule", "ours"},
		},
		{
			name:    "singular permission deny unions",
			base:    `{"permission":{"deny":["user rule"]}}`,
			overlay: `{"permission":{"deny":["ours"]}}`,
			path:    []string{"permission", "deny"},
			want:    []any{"user rule", "ours"},
		},
		{
			name:    "permission allow unions",
			base:    `{"permissions":{"allow":["user rule"]}}`,
			overlay: `{"permissions":{"allow":["ours"]}}`,
			path:    []string{"permissions", "allow"},
			want:    []any{"user rule", "ours"},
		},
		{
			name:    "permission ask unions",
			base:    `{"permissions":{"ask":["user rule"]}}`,
			overlay: `{"permissions":{"ask":["ours"]}}`,
			path:    []string{"permissions", "ask"},
			want:    []any{"user rule", "ours"},
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

func listAtPath(t *testing.T, raw []byte, path []string) []any {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("unmarshal merged json: %v\n%s", err, raw)
	}
	current := any(root)
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("path %v: %q is not an object in %s", path, key, raw)
		}
		current, ok = object[key]
		if !ok {
			t.Fatalf("path %v: key %q missing in %s", path, key, raw)
		}
	}
	list, ok := current.([]any)
	if !ok {
		t.Fatalf("path %v: value %#v is not a list in %s", path, current, raw)
	}
	return list
}
