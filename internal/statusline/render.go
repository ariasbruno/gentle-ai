package statusline

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
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
	ansiOrange        = "\x1b[38;5;214m"
	ansiRose          = "\x1b[38;2;255;118;195m"
	ansiEffort        = "\x1b[38;2;137;220;235m"
)

const (
	gaugeCells         = 8
	compactGaugeCells  = 6
	warningThreshold   = 80
	errorThreshold     = 95
	contextWarnPercent = 90
	defaultWidth       = 80
)

// petalGlyph returns the static rose glyph in Gentle Shell rose pink always:
// consistent color, no animation, identical in all agent states.
func petalGlyph(p *Payload) string {
	return ansiRose + ansiBold + "✿" + ansiReset
}

// colorMode colors the cycle/execution mode:
// auto = Gentle Shell rose pink, accept-edits = yellow, plan = orange.
func colorMode(mode string) string {
	switch strings.ToLower(mode) {
	case "auto":
		return ansiRose + mode + ansiReset
	case "accept-edits":
		return ansiWarning + mode + ansiReset
	case "plan", "planning":
		return ansiOrange + mode + ansiReset
	default:
		return mode
	}
}

// formatModelDisplay formats the model name and effort matching native Antigravity CLI:
// "Gemini 3.8 Flash · high"
func formatModelDisplay(m Model, fallbackEffort ...string) string {
	name := m.DisplayName
	if name == "" {
		name = m.ID
	}
	effort := m.Effort
	if effort == "" && len(fallbackEffort) > 0 {
		effort = fallbackEffort[0]
	}
	if effort != "" {
		effortPattern := regexp.MustCompile(`(?i)\s*(?:\(` + regexp.QuoteMeta(effort) + `\)|[·-]\s*` + regexp.QuoteMeta(effort) + `)$`)
		stripped := effortPattern.ReplaceAllString(name, "")
		dot := " " + ansiDim + "·" + ansiReset + " "
		effortStyled := ansiEffort + strings.ToLower(effort) + ansiReset
		if stripped != "" {
			return stripped + dot + effortStyled
		}
		return name + dot + effortStyled
	}
	return name
}

// colorRDD styles the RDD status indicator: on (accent green), off (dim),
// review in progress / fixing (warning yellow), approved (accent green).
func colorRDD(mode string) string {
	switch mode {
	case "on":
		return ansiAccent + "on" + ansiReset
	case "off":
		return ansiDim + "off" + ansiReset
	case "Review unavailable":
		return ansiDim + "Review unavailable" + ansiReset
	default:
		lower := strings.ToLower(mode)
		if strings.Contains(lower, "approved") {
			return ansiAccent + mode + ansiReset
		}
		if strings.Contains(lower, "review") || strings.Contains(lower, "fixing") {
			return ansiWarning + mode + ansiReset
		}
		return mode
	}
}

var changesPattern = regexp.MustCompile(`^(±\S+)(?:\s+\+(\d+)/(?:−|-)(\d+))?$`)

// colorChanges styles git changes: ±N (warning yellow), +A (accent green), −D (error red).
func colorChanges(s string) string {
	m := changesPattern.FindStringSubmatch(s)
	if m == nil {
		return s
	}
	countPart := ansiWarning + m[1] + ansiReset
	if len(m) > 2 && m[2] != "" && m[3] != "" {
		addedPart := ansiAccent + "+" + m[2] + ansiReset
		slashPart := ansiDim + "/" + ansiReset
		deletedPart := ansiError + "−" + m[3] + ansiReset
		return countPart + " " + addedPart + slashPart + deletedPart
	}
	return countPart
}

// FallbackLine is the petal-only line rendered for absent, empty, or
// unparsable payloads — graceful, never an error (S6, L19).
const FallbackLine = "✿"

// Segment tags drive the degradation cascade (S3): the survival priority is
// petal+mode·model > tasklist > rdd > engram > git > jobs > subagents > ctx >
// cost > quota, and dropOrder walks the exact reverse.
const (
	tagPetal     = "petal"
	tagCtx       = "ctx"
	tagCost      = "cost"
	tagQuota     = "quota"
	tagJobs      = "jobs"
	tagSubagents = "subagents"
	tagMem       = "mem"
	tagRDD       = "rdd"
	tagTasklist  = "tasklist"
	tagChanges   = "changes"
)

var dropOrder = []string{tagQuota, tagCost, tagCtx, tagSubagents, tagJobs, tagChanges, tagMem, tagRDD, tagTasklist}

type segment struct {
	tag string
	// text is the full rendering of the segment.
	text string
	// compact is the shortened form the cascade tries before dropping the
	// segment; empty means the segment has no compact form.
	compact string
}

// Extras carries the values the CLI layer resolves out of band: the
// in-process RDD review display (S15), the optional git changes summary
// (S5/S18), and the Engram observation count (S13; zero hides the segment).
type Extras struct {
	// RDDMode is the review display: "on", "off", or a state label — empty
	// hides the segment.
	RDDMode string
	// Changes is a preformatted "±N +A/−D" string — empty hides the segment.
	Changes string
	// Observations is the Engram observation count of the payload cwd.
	Observations int
	// EngramProject is the Engram project name of the payload cwd (empty
	// drops the name part of the 🧠 segment).
	EngramProject string
	// Tasklist is the parsed ODD feature-doc task state (S17); Total == 0
	// hides the segment.
	Tasklist TasklistStats
}

// TasklistStats carries the parsed ## Tasks state of the most recent ODD
// feature document (S17).
type TasklistStats struct {
	Done  int
	Total int
	// CurrentID/CurrentStatus name the first in_progress task (the first
	// pending one when nothing runs); empty means no active task.
	CurrentID     string
	CurrentStatus string
	Blocked       int
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

// renderGauge builds the bar with the given cell count (8 comfortable,
// 6 in compact density).
func renderGauge(pct, cells int) string {
	filled := pct * cells / 100
	if filled > cells {
		filled = cells
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("▰", filled) + strings.Repeat("▱", cells-filled)
}

// paintGauge builds the visual gauge with filled cells in the status color
// and empty cells in ansiDim, providing crisp contrast matching Gentle Shell.
func paintGauge(pct, cells int) string {
	filled := pct * cells / 100
	if filled > cells {
		filled = cells
	}
	if filled < 0 {
		filled = 0
	}
	empty := cells - filled

	var b strings.Builder
	if filled > 0 {
		b.WriteString(gaugeColor(pct))
		b.WriteString(strings.Repeat("▰", filled))
		b.WriteString(ansiReset)
	}
	if empty > 0 {
		b.WriteString(ansiDim)
		b.WriteString(strings.Repeat("▱", empty))
		b.WriteString(ansiReset)
	}
	return b.String()
}

// roundPercent rounds half up, matching the Gentle Shell Math.round.
func roundPercent(v float64) int {
	return int(v + 0.5)
}

// formatCost mirrors the Gentle Shell formatCost: two decimals at or above
// one dollar, three below — plain, with no suffix (L13).
func formatCost(cost float64) string {
	if cost < 1 {
		return fmt.Sprintf("$%.3f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}

// quotaCountdown formats the bucket's reset countdown:
// ≥24h → "Nd" (e.g. 5d), ≥1h → "Nh" (e.g. 1h), <1h → "Nm" (e.g. 40m).
// Never compound formats like 4d16h or 1h4m.
func quotaCountdown(seconds float64) string {
	if seconds <= 0 {
		return ""
	}
	total := int(seconds)
	if days := total / 86400; days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	if hours := total / 3600; hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", total/60)
}

// quotaLabelFromKey derives the window label from the bucket key — the S16
// fallback when a bucket carries no usable reset countdown. The active model
// family prefix and separators are stripped: "gemini-weekly" → "weekly".
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

// buildSegments assembles the ordered, visible segments (S2, L18/L19). Each
// segment carries its compact form for the cascade (S3). Density picks the
// starting forms: comfortable = full, compact = compact with 6-cell gauges,
// minimal = only the petal block, tasklist, rdd, and engram (compact).
func buildSegments(p *Payload, config Config, extras Extras) []segment {
	density := config.Density
	cells := gaugeCells
	useCompact := false
	minimal := false
	switch density {
	case DensityCompact:
		cells = compactGaugeCells
		useCompact = true
	case DensityMinimal:
		useCompact = true
		minimal = true
	}
	_ = minimal

	var segs []segment
	add := func(tag, text, compact string) {
		if text == "" {
			return
		}
		if useCompact && compact != "" {
			text = compact
		}
		segs = append(segs, segment{tag: tag, text: text, compact: compact})
	}
	inMinimal := func(tag string) bool {
		switch tag {
		case tagPetal, tagTasklist, tagRDD, tagMem:
			return true
		}
		return false
	}
	segmentOf := func(tag, text, compact string) {
		if minimal && !inMinimal(tag) {
			return
		}
		add(tag, text, compact)
	}

	// Block 1 (never drops): bare petal + `mode · model` (L18/L19/L25).
	// The mode renders verbatim when agy reports it — the live cycle_mode
	// ("plan", "accept-edits", "auto" where the account has it), the
	// canonical execution_mode as fallback — and the default mode renders
	// nothing (L25): absent means unset, never an invented label.
	mode := p.CycleMode
	if mode == "" {
		mode = p.ExecutionMode
	}
	petalText := petalGlyph(p)
	if config.Visibility.ModelDetails {
		modelText := formatModelDisplay(p.Model, p.Effort)
		if modelText == "" {
			modelText = colorMode(mode)
		} else if mode != "" {
			modelText = colorMode(mode) + " " + ansiDim + "·" + ansiReset + " " + modelText
		}
		if modelText != "" {
			petalText += " " + modelText
		}
	}
	segmentOf(tagPetal, petalText, "")

	if config.Visibility.UsageCost {
		pct := roundPercent(p.ContextWindow.UsedPercentage)
		marker := ""
		if config.Visibility.ContextWarning && pct >= contextWarnPercent {
			marker = " !"
		}
		segmentOf(tagCtx,
			fmt.Sprintf("%sctx%s %s %d%%%s", ansiDim, ansiReset, paintGauge(pct, cells), pct, marker),
			fmt.Sprintf("%sctx%s %s%d%%%s", ansiDim, ansiReset, gaugeColor(pct), pct, ansiReset))

		cost := 0.0
		if p.Cost != nil {
			cost = *p.Cost
		}
		costStr := formatCost(cost)
		if cost == 0 {
			costStr = ansiDim + costStr + ansiReset
		}
		segmentOf(tagCost, costStr, "")

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
			label := quotaCountdown(float64(bucket.ResetInSeconds))
			if label == "" {
				label = quotaLabelFromKey(key, family)
			}
			segmentOf(tagQuota,
				fmt.Sprintf("%s%s%s %s %d%%", ansiDim, label, ansiReset, paintGauge(used, cells), used),
				fmt.Sprintf("%s%s%s %s%d%%%s", ansiDim, label, ansiReset, gaugeColor(used), used, ansiReset))
		}
	}

	if config.Visibility.Agents {
		if p.TaskCount > 0 {
			jobsWord := "jobs"
			if p.TaskCount == 1 {
				jobsWord = "job"
			}
			segmentOf(tagJobs,
				fmt.Sprintf("%s⧗%s %d %s", ansiEffort, ansiReset, p.TaskCount, jobsWord),
				fmt.Sprintf("%s⧗%s %d", ansiEffort, ansiReset, p.TaskCount))
		}
		if count := p.SubagentCount(); count > 0 {
			agentsWord := "subagents"
			if count == 1 {
				agentsWord = "subagent"
			}
			segmentOf(tagSubagents,
				fmt.Sprintf("%s✦%s %d %s", ansiEffort, ansiReset, count, agentsWord),
				fmt.Sprintf("%s✦%s %d", ansiEffort, ansiReset, count))
		}
	}

	if config.Visibility.Engram && extras.Observations > 0 {
		full := fmt.Sprintf("🧠 %d", extras.Observations)
		if extras.EngramProject != "" {
			full = fmt.Sprintf("🧠 %s %s·%s %d", extras.EngramProject, ansiDim, ansiReset, extras.Observations)
		}
		segmentOf(tagMem, full, fmt.Sprintf("🧠 %d", extras.Observations))
	}

	if config.Visibility.RDD && extras.RDDMode != "" {
		segmentOf(tagRDD, "🌹 "+colorRDD(extras.RDDMode), "")
	}

	if config.Visibility.Tasklist && extras.Tasklist.Total > 0 {
		tasklist := extras.Tasklist
		var full, compact string
		switch {
		case tasklist.Done == tasklist.Total:
			full = ansiAccent + fmt.Sprintf("⋗ %d/%d ✓", tasklist.Done, tasklist.Total) + ansiReset
			compact = full
		case tasklist.CurrentID != "":
			full = ansiWarning + fmt.Sprintf("⋗ %d/%d · %s %s", tasklist.Done, tasklist.Total, tasklist.CurrentID, tasklist.CurrentStatus) + ansiReset
			if tasklist.Blocked > 0 {
				full += ansiWarning + fmt.Sprintf(" ⚠%d", tasklist.Blocked) + ansiReset
			}
			compact = ansiWarning + fmt.Sprintf("⋗ %d/%d", tasklist.Done, tasklist.Total) + ansiReset
		default:
			full = fmt.Sprintf("⋗ %d/%d", tasklist.Done, tasklist.Total)
			compact = full
		}
		segmentOf(tagTasklist, full, compact)
	}

	if config.Visibility.Changes && extras.Changes != "" {
		segmentOf(tagChanges, colorChanges(extras.Changes), "")
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

// Render builds the final status line for one payload (S3): the density-
// adapted segments first, then the width-driven cascade — each class in
// dropOrder first switches to its compact form, then is dropped — and a
// final ellipsis truncation. The petal block never drops.
func Render(payload *Payload, config Config, extras Extras) string {
	if payload == nil {
		return FallbackLine
	}
	width := payload.TerminalWidth
	if width <= 0 {
		width = defaultWidth
	}
	segs := buildSegments(payload, config, extras)

	line := joinSegments(segs)
	if visibleWidth(line) <= width {
		return line
	}

	for _, drop := range dropOrder {
		// Compact step: switch every segment of the class to its compact
		// form when one exists.
		compacted := false
		for i := range segs {
			if segs[i].tag == drop && segs[i].compact != "" && segs[i].text != segs[i].compact {
				segs[i].text = segs[i].compact
				compacted = true
			}
		}
		if compacted {
			line = joinSegments(segs)
			if visibleWidth(line) <= width {
				return line
			}
		}
		// Drop step: remove every segment of the class.
		kept := segs[:0]
		for _, seg := range segs {
			if seg.tag != drop {
				kept = append(kept, seg)
			}
		}
		segs = kept
		line = joinSegments(segs)
		if visibleWidth(line) <= width {
			return line
		}
	}

	// Final truncation of whatever survives (the petal block at minimum).
	if plain := stripANSI(line); utf8.RuneCountInString(plain) > width {
		return string([]rune(plain)[:width-1]) + "…"
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
