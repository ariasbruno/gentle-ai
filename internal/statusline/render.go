package statusline

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// ANSI escapes — the standard 16-color palette, matching the upstream agy
// example's portability guidance.
const (
	ansiReset   = "\x1b[0m"
	ansiAccent  = "\x1b[32m"
	ansiWarning = "\x1b[33m"
	ansiError   = "\x1b[31m"
	ansiDim     = "\x1b[2m"
	ansiBold    = "\x1b[1m"
)

const (
	brandText          = "✿ gentle ai"
	gaugeCells         = 8
	warningThreshold   = 80
	errorThreshold     = 95
	contextWarnPercent = 90
	compactBranchWidth = 15
	defaultWidth       = 80
)

// FallbackLine is the brand-only line rendered for absent, empty, or
// unparsable payloads — graceful, never an error (S6).
const FallbackLine = brandText

// Segment tags drive the degradation cascade: auxiliary segments die first;
// brand and location survive longest.
const (
	tagBrand     = "brand"
	tagState     = "state"
	tagLocation  = "location"
	tagModel     = "model"
	tagCtx       = "ctx"
	tagCost      = "cost"
	tagQuota     = "quota"
	tagJobs      = "jobs"
	tagSubagents = "subagents"
	tagSandbox   = "sandbox"
	tagVim       = "vim"
	tagRDD       = "rdd"
	tagChanges   = "changes"
)

// dropOrder is the degradation step-2 order (S3): changes and rdd first
// (auxiliary, subprocess-derived), then chrome, then the counters, keeping
// brand and location alive longest.
var dropOrder = []string{tagChanges, tagRDD, tagVim, tagSandbox, tagSubagents, tagJobs, tagQuota, tagCost, tagCtx, tagModel, tagState}

type segment struct {
	tag  string
	text string
}

// Extras carries the values the CLI layer resolves out of band: the home
// directory for ~ shortening, the in-process RDD review switch mode, and the
// optional git changes summary.
type Extras struct {
	Home string
	// RDDMode is "on", "off", or "?" — empty hides the segment.
	RDDMode string
	// Changes is a preformatted "±N +A/−D" string — empty hides the segment.
	Changes string
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

// visibleWidth counts display characters with ANSI stripped, matching the
// upstream example's wc -m approximation.
func visibleWidth(s string) int {
	return utf8.RuneCountInString(stripANSI(s))
}

// gaugeColor picks the gauge tone by percentage thresholds (S2).
func gaugeColor(pct int) string {
	switch {
	case pct >= errorThreshold:
		return ansiError
	case pct >= warningThreshold:
		return ansiWarning
	default:
		return ansiAccent
	}
}

// renderGauge builds the 8-cell bar.
func renderGauge(pct int) string {
	filled := pct * gaugeCells / 100
	if filled > gaugeCells {
		filled = gaugeCells
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("▰", filled) + strings.Repeat("▱", gaugeCells-filled)
}

// roundPercent rounds half up, matching the Gentle Shell Math.round.
func roundPercent(v float64) int {
	return int(v + 0.5)
}

// stateBadge maps the agy agent_state enum to the Gentle badge (S2).
func stateBadge(state string) string {
	switch state {
	case "", "idle":
		return "● ready"
	case "thinking":
		return "◆ thinking"
	case "working":
		return "⚙ working"
	case "tool_use":
		return "🔧 tool"
	case "initializing":
		return "⏳ initializing"
	default:
		return "○ " + state
	}
}

// shortenCwd replaces the home prefix with ~.
func shortenCwd(cwd, home string) string {
	if home != "" && strings.HasPrefix(cwd, home) {
		return "~" + strings.TrimPrefix(cwd, home)
	}
	return cwd
}

// compactLocation shrinks the location for degradation step 2: cwd becomes
// its last path segment (~ for the home itself) and a long branch is clipped
// to 15 visible characters with an ellipsis (S3).
func compactLocation(cwd, home, branch string, dirty bool) string {
	compact := cwd
	if home != "" && cwd == home {
		compact = "~"
	} else if trimmed := strings.TrimSuffix(cwd, "/"); trimmed != "" {
		if index := strings.LastIndex(trimmed, "/"); index >= 0 {
			compact = trimmed[index+1:]
		}
	}
	if branch != "" {
		if runes := []rune(branch); len(runes) > compactBranchWidth {
			branch = string(runes[:compactBranchWidth-1]) + "…"
		}
		compact += " " + branch
	}
	if dirty {
		compact += "*"
	}
	return compact
}

// formatCost mirrors the Gentle Shell formatCost: two decimals at or above
// one dollar, three below, with the subscription suffix (S2).
func formatCost(cost float64, subscription bool) string {
	amount := fmt.Sprintf("%.2f", cost)
	if cost < 1 {
		amount = fmt.Sprintf("%.3f", cost)
	}
	if subscription {
		return "$" + amount + " sub"
	}
	return "$" + amount
}

// quotaLabel derives the window label from reset_in_seconds: week, whole
// days, whole hours, or rounded minutes (S2).
func quotaLabel(resetInSeconds int64) string {
	const (
		minute int64 = 60
		hour         = 60 * minute
		day          = 24 * hour
		week         = 7 * day
	)
	switch {
	case resetInSeconds == week:
		return "week"
	case resetInSeconds >= day && resetInSeconds%day == 0:
		return fmt.Sprintf("%dd", resetInSeconds/day)
	case resetInSeconds >= hour && resetInSeconds%hour == 0:
		return fmt.Sprintf("%dh", resetInSeconds/hour)
	default:
		return fmt.Sprintf("%dm", int(float64(resetInSeconds)/float64(minute)+0.5))
	}
}

// buildSegments assembles the ordered, visible segments (S2). Empty segments
// are omitted; visibility flags and density gate their groups.
func buildSegments(p *Payload, config Config, extras Extras) []segment {
	var segs []segment
	add := func(tag, text string) {
		if text != "" {
			segs = append(segs, segment{tag: tag, text: text})
		}
	}

	if config.Density != DensityMinimal {
		add(tagBrand, ansiBold+brandText+ansiReset)
	}
	add(tagState, stateBadge(p.AgentState))
	location := shortenCwd(p.Cwd, extras.Home)
	if p.VCS.Branch != "" {
		location += " " + p.VCS.Branch
	}
	if p.VCS.Dirty {
		location += "*"
	}
	add(tagLocation, location)

	if config.Visibility.ModelDetails {
		model := p.Model.DisplayName
		if p.ExecutionMode != "" {
			model += " · " + p.ExecutionMode
		}
		add(tagModel, model)
	}

	if config.Visibility.UsageCost {
		pct := roundPercent(p.ContextWindow.UsedPercentage)
		marker := ""
		if config.Visibility.ContextWarning && pct >= contextWarnPercent {
			marker = " !"
		}
		add(tagCtx, fmt.Sprintf("ctx %s%s%s %d%%%s", gaugeColor(pct), renderGauge(pct), ansiReset, pct, marker))

		cost := 0.0
		if p.Cost != nil {
			cost = *p.Cost
		}
		add(tagCost, formatCost(cost, p.PlanTier != ""))

		keys := make([]string, 0, len(p.Quota))
		for key := range p.Quota {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			bucket := p.Quota[key]
			used := roundPercent((1 - bucket.RemainingFraction) * 100)
			if used < 0 {
				used = 0
			}
			if used > 100 {
				used = 100
			}
			add(tagQuota, fmt.Sprintf("%s %s%s%s %d%%", quotaLabel(bucket.ResetInSeconds), gaugeColor(used), renderGauge(used), ansiReset, used))
		}
	}

	if config.Visibility.Agents {
		if p.TaskCount > 0 {
			add(tagJobs, fmt.Sprintf("⧗ %d jobs", p.TaskCount))
		}
		if count := p.SubagentCount(); count > 0 {
			add(tagSubagents, fmt.Sprintf("✦ %d subagents", count))
		}
	}

	if p.Sandbox.Enabled {
		badge := "⊘ sandbox"
		if p.Sandbox.AllowNetwork {
			badge += "·net"
		}
		add(tagSandbox, badge)
	}

	if p.Vim.Mode != "" {
		add(tagVim, "vim."+p.Vim.Mode)
	}

	if config.Visibility.RDD {
		switch extras.RDDMode {
		case "on":
			add(tagRDD, "🌹 on")
		case "off":
			add(tagRDD, "🌹 off")
		case "?":
			add(tagRDD, "🌹 ?")
		}
	}

	if config.Visibility.Changes {
		add(tagChanges, extras.Changes)
	}

	return segs
}

// joinSegments paints the dim separator between the surviving segments.
func joinSegments(segs []segment) string {
	separator := " " + ansiDim + "⟡" + ansiReset + " "
	parts := make([]string, 0, len(segs))
	for _, seg := range segs {
		parts = append(parts, seg.text)
	}
	return strings.Join(parts, separator)
}

// fitWithTitle renders line plus right-aligned title when both fit the
// width; the title keeps at least one leading space.
func fitWithTitle(line, title string, width int) (string, bool) {
	lineWidth := visibleWidth(line)
	if title == "" {
		if lineWidth <= width {
			return line, true
		}
		return "", false
	}
	titleWidth := visibleWidth(title)
	if lineWidth+1+titleWidth <= width {
		padding := width - lineWidth - titleWidth
		if padding < 1 {
			padding = 1
		}
		return line + strings.Repeat(" ", padding) + title, true
	}
	return "", false
}

// Render builds the final status line for one payload: the full line first,
// then the width-driven degradation cascade (compact location → dropped
// segments → ellipsis truncation), with the conversation title right-aligned
// whenever it fits (S3).
func Render(payload *Payload, config Config, extras Extras) string {
	if payload == nil {
		return FallbackLine
	}
	width := payload.TerminalWidth
	if width <= 0 {
		width = defaultWidth
	}
	segs := buildSegments(payload, config, extras)
	title := payload.ConversationTitle

	if line, ok := fitWithTitle(joinSegments(segs), title, width); ok {
		return line
	}

	for i := range segs {
		if segs[i].tag == tagLocation {
			segs[i].text = compactLocation(payload.Cwd, extras.Home, payload.VCS.Branch, payload.VCS.Dirty)
			break
		}
	}
	if line, ok := fitWithTitle(joinSegments(segs), title, width); ok {
		return line
	}

	for _, drop := range dropOrder {
		for i := len(segs) - 1; i >= 0; i-- {
			if segs[i].tag == drop {
				segs = append(segs[:i], segs[i+1:]...)
				if drop != tagQuota {
					break
				}
			}
		}
		if line, ok := fitWithTitle(joinSegments(segs), title, width); ok {
			return line
		}
	}

	line := joinSegments(segs)
	if visibleWidth(line) > width {
		plain := stripANSI(line)
		if title != "" {
			titleWidth := visibleWidth(title)
			avail := width - titleWidth - 1
			if avail > 1 && utf8.RuneCountInString(plain) > avail {
				truncated := string([]rune(plain)[:avail-1]) + "…"
				padding := width - avail - titleWidth
				if padding < 0 {
					padding = 0
				}
				return truncated + strings.Repeat(" ", padding) + title
			}
		} else if utf8.RuneCountInString(plain) > width {
			return string([]rune(plain)[:width-1]) + "…"
		}
	}
	if fitted, ok := fitWithTitle(line, title, width); ok {
		return fitted
	}
	return line
}

// RenderTitle renders the --title mode: the terminal window title, plain
// (agy strips ANSI escapes there anyway), with partial fallbacks (S7).
func RenderTitle(payload *Payload) string {
	if payload == nil {
		return "✿"
	}
	switch {
	case payload.VCS.Branch != "" && payload.Model.DisplayName != "":
		return fmt.Sprintf("✿ %s · %s", payload.VCS.Branch, payload.Model.DisplayName)
	case payload.VCS.Branch != "":
		return "✿ " + payload.VCS.Branch
	case payload.Model.DisplayName != "":
		return "✿ " + payload.Model.DisplayName
	default:
		return "✿"
	}
}
