package filemerge

import (
	"testing"
)

func TestPrependYAMLFrontmatter_EmptyFrontmatterReturnsContent(t *testing.T) {
	content := "# My Document\n\nSome content here.\n"
	result := PrependYAMLFrontmatter(content, "")
	if result != content {
		t.Fatalf("empty frontmatter should return content unchanged:\ngot:  %q\nwant: %q", result, content)
	}
}

func TestPrependYAMLFrontmatter_PrependsToCleanContent(t *testing.T) {
	content := "# My Document\n\nSome content here.\n"
	frontmatter := "---\ntrigger: always_on\n---\n"
	result := PrependYAMLFrontmatter(content, frontmatter)

	expected := frontmatter + content
	if result != expected {
		t.Fatalf("prepend on clean content:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestPrependYAMLFrontmatter_Idempotent(t *testing.T) {
	content := "# My Document\n\nSome content here.\n"
	frontmatter := "---\ntrigger: always_on\n---\n"

	first := PrependYAMLFrontmatter(content, frontmatter)
	second := PrependYAMLFrontmatter(first, frontmatter)

	if first != second {
		t.Fatalf("idempotent: second call changed result:\nfirst:  %q\nsecond: %q", first, second)
	}
}

func TestPrependYAMLFrontmatter_ReplacesStaleFrontmatter(t *testing.T) {
	staleFrontmatter := "---\ntrigger: something_else\n---\n"
	content := staleFrontmatter + "# My Document\n\nSome content here.\n"
	newFrontmatter := "---\ntrigger: always_on\n---\n"

	result := PrependYAMLFrontmatter(content, newFrontmatter)

	expected := newFrontmatter + "# My Document\n\nSome content here.\n"
	if result != expected {
		t.Fatalf("replaces stale frontmatter:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestPrependYAMLFrontmatter_StaleFrontmatterWithDescription(t *testing.T) {
	staleFrontmatter := "---\ntrigger: always_on\ndescription: old description\n---\n"
	content := staleFrontmatter + "# My Document\n\nSome content here.\n"
	newFrontmatter := "---\ntrigger: always_on\ndescription: new description\n---\n"

	result := PrependYAMLFrontmatter(content, newFrontmatter)

	expected := newFrontmatter + "# My Document\n\nSome content here.\n"
	if result != expected {
		t.Fatalf("replaces stale frontmatter with description:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestPrependYAMLFrontmatter_PreservesContentAfterFrontmatter(t *testing.T) {
	frontmatter := "---\ntrigger: always_on\n---\n"
	content := frontmatter + "# My Document\n\nSome content here.\nMore content.\n"
	newFrontmatter := "---\ntrigger: always_on\ndescription: new\n---\n"

	result := PrependYAMLFrontmatter(content, newFrontmatter)

	expected := newFrontmatter + "# My Document\n\nSome content here.\nMore content.\n"
	if result != expected {
		t.Fatalf("preserves content after frontmatter:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestPrependYAMLFrontmatter_NoLeadingFrontmatterWhenContentStartsWithDashes(t *testing.T) {
	// Content starts with --- but not a valid frontmatter block (no closing ---)
	content := "--- not a frontmatter block\n# My Document\n\nContent.\n"
	frontmatter := "---\ntrigger: always_on\n---\n"

	result := PrependYAMLFrontmatter(content, frontmatter)

	// Should prepend since there's no valid closing ---
	expected := frontmatter + content
	if result != expected {
		t.Fatalf("content starting with --- but no closing ---:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestPrependYAMLFrontmatter_FrontmatterWithOnlyOpeningDash(t *testing.T) {
	// Content has opening --- but no closing ---
	content := "---\ntrigger: something\n# My Document\n\nContent.\n"
	frontmatter := "---\ntrigger: always_on\n---\n"

	result := PrependYAMLFrontmatter(content, frontmatter)

	// Should NOT strip because no closing ---
	expected := frontmatter + content
	if result != expected {
		t.Fatalf("content with opening --- but no closing ---:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestPrependYAMLFrontmatter_FrontmatterWithMultipleLines(t *testing.T) {
	content := "# My Document\n\nContent.\n"
	frontmatter := "---\ntrigger: always_on\ndescription: Gentle AI ODD workflow\nrules:\n  - rule1\n  - rule2\n---\n"

	result := PrependYAMLFrontmatter(content, frontmatter)

	expected := frontmatter + content
	if result != expected {
		t.Fatalf("multi-line frontmatter:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestPrependYAMLFrontmatter_StaleFrontmatterMultipleLines(t *testing.T) {
	stale := "---\ntrigger: something_else\ndescription: old\ncustom_field: value\n---\n"
	content := stale + "# My Document\n\nContent.\n"
	newFrontmatter := "---\ntrigger: always_on\n---\n"

	result := PrependYAMLFrontmatter(content, newFrontmatter)

	expected := newFrontmatter + "# My Document\n\nContent.\n"
	if result != expected {
		t.Fatalf("replaces multi-line stale frontmatter:\ngot:  %q\nwant: %q", result, expected)
	}
}