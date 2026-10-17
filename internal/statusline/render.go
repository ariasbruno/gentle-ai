package statusline

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// ANSI escapes — the standard 16-color palette, matching the upstream agy
// example's portability guidance.
const (
	ansiReset         = "\x1b[0m"
	ansiAccent        = "\x1b[32m"
	ansiWarning       = "\x1b[33m"
	ansiError         = "\x1b[31m"
	ansiDim           = "\x1b[2m"
	ansiBold          = "\x1b[1m"
	ansiMagenta       = "\x1b[35m"
	ansiBrightMagenta = "\x1b[95m"
	ansiCyan          = "\x1b[36m"
	ansiYellow        = "\x1b[33m"
	ansiBrightWhite   = "\x1b[97m"
	ansiBrightCyan    = "\x1b[96m"
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

// The animated petal, ported from Gentle Shell's prompt (lib/shell-prompt.ts):
// idle keeps a static flower; working spins four frames while the tone walks
// the GS pulse deep → soft → rose → bright → rose → soft → deep.
var petalFrames = [4]string{"✿", "❀", "❁", "✾"}
var petalTones = [8]string{
	ansiMagenta, ansiCyan, ansiYellow, ansiBrightMagenta,
	ansiYellow, ansiCyan, ansiMagenta, ansiMagenta,
}

// The reflected-light wave over the working label, ported from Gentle Shell's
// scanWorkingText (lib/shell-prompt.ts): a bright head sweeps the text and
// the neighboring characters settle through accent and soft into the muted
// base, one step per render tick.
const (
	scannerSteps       = 15
	scannerEntryOffset = 3
)

func scanWorkingLabel(text string, tick int64) string {
	phase := ((tick % scannerSteps) + scannerSteps) % scannerSteps
	head := int(phase) - scannerEntryOffset
	var builder strings.Builder
	for index, char := range []rune(text) {
		distance := head - index
		if distance < 0 {
			distance = -distance
		}
		var tone string
		switch distance {
		case 0:
			tone = ansiBrightWhite
		case 1:
			tone = ansiYellow
		case 2:
			tone = ansiBrightCyan
		default:
			tone = ansiDim
		}
		builder.WriteString(tone)
		builder.WriteRune(char)
		builder.WriteString(ansiReset)
	}
	return builder.String()
}

// petalState builds the merged brand/state segment: idle → the static
// brand (no label, hidden entirely in minimal density); active → the
// animated petal plus the phase label (S12 phase when provided, otherwise
// the agent_state fallback), like the Gentle Shell prompt frame.
func petalState(p *Payload, config Config, extras Extras) string {
	if p.AgentState == "" || p.AgentState == "idle" {
		if config.Density == DensityMinimal {
			return ""
		}
		return ansiBold + brandText + ansiReset
	}
	tick := time.Now().Unix()
	glyph := petalFrames[tick%int64(len(petalFrames))]
	tone := petalTones[tick%int64(len(petalTones))]
	label := extras.Phase
	if label == "" {
		label = stateBadge(p.AgentState)
	}
	return tone + ansiBold + glyph + ansiReset + " " + scanWorkingLabel(label+"…", tick)
}

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
	tagMem       = "mem"
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
// directory for ~ shortening, the in-process RDD review switch mode, the
// optional git changes summary, the ODD phase inferred by the PreToolUse
// hook (S12; empty falls back to the agent_state label), and the Engram
// observation count (S13; zero hides the segment).
type Extras struct {
	Home string
	// RDDMode is "on", "off", or "?" — empty hides the segment.
	RDDMode string
	// Changes is a preformatted "±N +A/−D" string — empty hides the segment.
	Changes string
	// Phase is the hook-inferred ODD phase label ("exploring", …) — empty
	// falls back to the agent_state label while the agent is active.
	Phase string
	// Observations is the Engram observation count of the payload cwd.
	Observations int
	// EngramProject is the Engram project name of the payload cwd (empty
	// drops the name part of the 🧠 segment).
	EngramProject string
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

// stateBadge maps the agy agent_state enum to the fallback label shown while
// the agent is active (idle shows no label — the static petal brand is the
// whole state, Gentle Shell prompt parity).
func stateBadge(state string) string {
	switch state {
	case "thinking":
		return "thinking"
	case "working", "tool_use":
		return "working"
	case "initializing":
		return "starting"
	default:
		return "working"
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
// one dollar, three below — plain, with no suffix (L13: the `sub` marker
// confused more than it informed).
func formatCost(cost float64) string {
	if cost < 1 {
		return fmt.Sprintf("$%.3f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}

// quotaLabelFromKey derives the window label from the bucket key — never
// from reset_in_seconds, which is the countdown to the reset, not the window
// length (L13). The active model family prefix and separators are stripped:
// "gemini-weekly" → "weekly".
func quotaLabelFromKey(key, family string) string {
	label := key
	if family != "" && strings.HasPrefix(strings.ToLower(key), strings.ToLower(family)+"-") {
		label = key[len(family)+1:]
	}
	label = strings.Trim(label, "-_ ")
	if label == "" {
		return key
	}
	return strings.ToLower(label)
}

// modelFamily extracts the provider family from the active model identity —
// the first word of the model id, lowercased, falling back to the display
// name when the id is absent ("Gemini 3.5 Pro (High)" → "gemini").
func modelFamily(modelID, displayName string) string {
	for _, candidate := range []string{modelID, displayName} {
		for _, field := range strings.Fields(candidate) {
			var letters strings.Builder
			for _, r := range field {
				if !unicode.IsLetter(r) {
					break
				}
				letters.WriteRune(unicode.ToLower(r))
			}
			if letters.Len() > 0 {
				return letters.String()
			}
		}
	}
	return ""
}

// activeQuota returns the quota buckets of the active model's family; when
// no bucket matches the family the full set is kept (defensive: the payload
// shape is only confirmed by live use, L13).
func activeQuota(p *Payload) map[string]Quota {
	family := modelFamily(p.Model.ID, p.Model.DisplayName)
	if family == "" || len(p.Quota) == 0 {
		return p.Quota
	}
	filtered := make(map[string]Quota, len(p.Quota))
	for key, bucket := range p.Quota {
		if strings.Contains(strings.ToLower(key), family) {
			filtered[key] = bucket
		}
	}
	if len(filtered) == 0 {
		return p.Quota
	}
	return filtered
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

	// Brand/state petal (S2): idle → static brand, no label; active → the
	// animated petal plus the phase label, Gentle Shell prompt parity.
	add(tagBrand, petalState(p, config, extras))
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
		add(tagCost, formatCost(cost))

		quota := activeQuota(p)
		family := modelFamily(p.Model.ID, p.Model.DisplayName)
		keys := make([]string, 0, len(quota))
		for key := range quota {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			bucket := quota[key]
			used := roundPercent((1 - bucket.RemainingFraction) * 100)
			if used < 0 {
				used = 0
			}
			if used > 100 {
				used = 100
			}
			add(tagQuota, fmt.Sprintf("%s %s%s%s %d%%", quotaLabelFromKey(key, family), gaugeColor(used), renderGauge(used), ansiReset, used))
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

	if config.Visibility.Engram && extras.Observations > 0 {
		if extras.EngramProject != "" {
			add(tagMem, fmt.Sprintf("🧠 %s · %d", extras.EngramProject, extras.Observations))
		} else {
			add(tagMem, fmt.Sprintf("🧠 %d", extras.Observations))
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

	if config.Visibility.RDD && extras.RDDMode != "" {
		add(tagRDD, "🌹 "+extras.RDDMode)
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
