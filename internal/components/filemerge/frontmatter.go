package filemerge

import (
	"strings"
)

// PrependYAMLFrontmatter prepends a YAML frontmatter block to content.
// If frontmatter is empty, returns content unchanged.
// Idempotent: strips any existing leading YAML frontmatter block (a leading `---` line
// closed by a `---` line) before prepending, so repeated syncs never duplicate blocks.
// Preserves the rest of the content byte-for-byte after the (possibly removed) old block.
func PrependYAMLFrontmatter(content, frontmatter string) string {
	if strings.TrimSpace(frontmatter) == "" {
		return content
	}

	// Strip existing leading YAML frontmatter block if present
	content = stripLeadingYAMLFrontmatter(content)

	// Prepend the new frontmatter
	var sb strings.Builder
	sb.WriteString(frontmatter)
	if !strings.HasSuffix(frontmatter, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString(content)
	return sb.String()
}

// stripLeadingYAMLFrontmatter removes a leading YAML frontmatter block if present.
// A valid frontmatter block starts with a line containing only `---` and ends with
// a line containing only `---`. Returns the content after the block, or the original
// content if no valid frontmatter block is found at the start.
func stripLeadingYAMLFrontmatter(content string) string {
	// Check if content starts with `---` at the beginning of the file
	if !strings.HasPrefix(content, "---\n") && content != "---" {
		return content
	}

	// Find the closing `---` line (must be at the start of a line)
	searchStart := 4 // skip the opening `---\n` (or just `---` if no newline)
	if len(content) >= 4 && content[3] == '\n' {
		searchStart = 4
	} else if content == "---" {
		// Only opening --- with no content after
		return ""
	}

	// Look for closing `---` at line start
	for i := searchStart; i < len(content); i++ {
		if content[i] != '\n' {
			continue
		}
		if i+3 >= len(content) || content[i+1] != '-' || content[i+2] != '-' || content[i+3] != '-' {
			continue
		}
		endIdx := i + 4
		// The closing delimiter is exactly three dashes on their own line.
		// A longer dash run (for example a four-dash markdown horizontal rule)
		// is body content, not a frontmatter closer: keep scanning.
		if endIdx < len(content) && content[endIdx] != '\n' {
			continue
		}
		if endIdx < len(content) {
			endIdx++ // consume the newline after closing ---
		}
		return content[endIdx:]
	}

	// No valid closing --- found, return original content
	return content
}