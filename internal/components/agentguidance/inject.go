package agentguidance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gentleman-programming/gentle-ai/v4/internal/agents"
	"github.com/gentleman-programming/gentle-ai/v4/internal/components/filemerge"
	"github.com/gentleman-programming/gentle-ai/v4/internal/components/opencodedefault"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
)

// RoutingSectionID is the managed marker section that owns routing guidance.
// It is deliberately independent of the SDD sections so that installing or
// removing optional SDD assets can never add or drop routing guidance.
const RoutingSectionID = "agent-routing"

// routingModuleFile is the include module written for adapters whose system
// prompt is a Jinja router. It is derived from the section ID so the template
// include and the managed marker can never drift apart.
const routingModuleFile = RoutingSectionID + ".md"

// ErrInvalidTarget rejects an unusable installation root before any write, so
// a caller that lost its resolved home directory fails loudly instead of
// writing guidance into an unexpected location.
var ErrInvalidTarget = errors.New("invalid routing guidance target directory")

// ErrUnreadableSettings fails closed when an adapter's settings document cannot
// be decoded. Routing guidance is worth less than a user's configuration, so a
// document we cannot understand is never rewritten from an empty base.
var ErrUnreadableSettings = errors.New("unreadable agent settings document")

// ErrUnloadableGuidance fails closed when guidance would be written to a place
// the agent does not actually load. Silently installing unread guidance is the
// exact failure this component exists to prevent.
var ErrUnloadableGuidance = errors.New("routing guidance target is not loaded by the agent")

// Result mirrors the shape returned by the SDD injector so both installers can
// be aggregated by the same caller.
type Result struct {
	Changed bool
	Files   []string
}

// templateBootstrapper is the optional adapter capability that restores a Jinja
// router template from its embedded asset. Adapters that rewrite their entry
// template on every install must expose it, because guidance can only survive
// outside that template.
type templateBootstrapper interface {
	BootstrapTemplate(homeDir string) error
}

// RoutingOptions supplies a caller-resolved settings path for adapters whose
// guidance is delivered through a managed orchestrator definition. The adapter
// retains its normal targetDir-derived path when SettingsPath is empty.
//
// ReviewContract renders the native review execution contract embedded in the
// orchestrator of a receipt-driven development runtime. It takes precedence
// over the package-level fallback (SetReviewContractSource); installers must
// always set it.
type RoutingOptions struct {
	// OrchestratorCapability selects only generic instruction text. "small"
	// explicitly opts into the small variant; empty or unknown uses capable.
	// Runtime-specific assets ignore this hint. It does not configure models.
	OrchestratorCapability      string
	SettingsPath                string
	ReviewContract              ReviewContractSource
	CodexPhaseModelAssignments  map[string]string
	CodexModelAssignments       map[string]model.CodexEffort
	CodexCarrilModelAssignments map[string]string
}

// InjectRoutingWithOptions installs the organic routing guidance for one
// supported agent under targetDir, which is the installation root the adapter
// resolves its configuration paths from, using a caller-resolved settings path
// when the adapter's effective configuration authority differs from its
// ordinary targetDir-derived path.
//
// Delivery is strategy-aware: writing one markdown file for every adapter would
// land the guidance in a scope the agent never loads, or inside a template its
// own installer rewrites from an embedded asset on the next sync.
//
// Every non-Pi runtime also receives its orchestrator instructions as a second
// managed section placed ahead of routing (see RenderOrchestrator); a v3.7.0
// sdd-orchestrator block is converted in place so exactly one remains.
//
// Only the marked sections are owned by Gentle AI: everything a user wrote
// around them is preserved verbatim, and a second identical injection is a no-op.
func InjectRoutingWithOptions(targetDir string, agent model.AgentID, options RoutingOptions) (Result, error) {
	// Conductor is detection/catalog-only: its workspaces inherit Claude Code
	// configuration, so there is no standalone guidance target to write. Skip
	// it cleanly instead of failing closed like an unknown delivery (see
	// isCatalogOnlyGuidanceTarget).
	if isCatalogOnlyGuidanceTarget(agent) {
		return Result{}, nil
	}

	// Render before resolving the delivery so an unsupported agent is rejected
	// without having touched the filesystem.
	rendered, err := RenderRouting(agent)
	if err != nil {
		return Result{}, err
	}
	if agent == model.AgentCodex {
		rendered += "\n\n### Codex ODD worker assignments\n\nUse the exact model and reasoning_effort for the selected worker class when calling `spawn_agent`; set `fork_turns: \"none\"` for overrides. These assignments apply to ODD delegation, not native RDD review.\n\n" +
			model.RenderCodexODDAssignments(options.CodexPhaseModelAssignments, options.CodexModelAssignments, options.CodexCarrilModelAssignments)
	}

	delivery, err := resolveRoutingDelivery(targetDir, agent, options)
	if err != nil {
		return Result{}, err
	}

	// Keep the boundary in the unconditional guidance carrier, independent of
	// persona and SDD selection. Nesting inside the owned routing section also
	// preserves it when other component writers retain that section.
	rendered = InjectRemoteAuthorization(rendered)

	// The orchestrator travels with routing into the same always-loaded scope
	// and the same write, so the two can never land in different files or be
	// half-applied. Pi is the only runtime without one: Gentle Shell owns it.
	var orchestrator string
	if agent != model.AgentPi && agent != model.AgentAntigravityCLI {
		orchestrator, err = RenderOrchestratorWithSource(agent, options.ReviewContract, options.OrchestratorCapability)
		if err != nil {
			return Result{}, err
		}
	}
	merge := func(existing string) string {
		if orchestrator != "" {
			existing = injectOrchestratorSection(existing, orchestrator)
		}
		return filemerge.InjectMarkdownSection(existing, RoutingSectionID, rendered)
	}

	switch delivery.kind {
	case deliveryOrchestratorPrompt:
		return injectOrchestratorPrompt(delivery, agent, merge)
	case deliveryJinjaModule:
		return injectJinjaModule(targetDir, delivery, agent, merge)
	case deliveryAntigravityCLISplit:
		return injectAntigravityCLISplit(targetDir, delivery, rendered)
	default:
		return injectPromptSection(delivery, merge)
	}
}

// RoutingPaths reports the exact filesystem paths InjectRouting would write for
// one supported agent under targetDir, without creating or touching anything.
//
// Install and sync must snapshot every file they are about to rewrite. Routing
// guidance is delivered outside the component loop, so a selection whose
// components do not happen to cover the same file would otherwise be rewritten
// without a backup and could never be rolled back. Both answers come from the
// same delivery resolution, so the backup contract cannot drift away from what
// the injector actually writes.
func RoutingPaths(targetDir string, agent model.AgentID) ([]string, error) {
	return RoutingPathsWithOptions(targetDir, agent, RoutingOptions{})
}

// RoutingPathsWithOptions reports the same paths InjectRoutingWithOptions would
// write, including any caller-resolved effective settings path.
func RoutingPathsWithOptions(targetDir string, agent model.AgentID, options RoutingOptions) ([]string, error) {
	if isCatalogOnlyGuidanceTarget(agent) {
		// Same catalog-only skip as InjectRoutingWithOptions: no guidance
		// target exists, so the backup snapshot must declare no path.
		return nil, nil
	}

	delivery, err := resolveRoutingDelivery(targetDir, agent, options)
	if err != nil {
		return nil, err
	}
	return delivery.paths, nil
}

// isCatalogOnlyGuidanceTarget reports whether an agent is detection and
// catalog only, with no standalone guidance target for install/sync to write.
// Conductor inherits Claude Code configuration for the workspaces it manages,
// and its capability manifest claims no managed system prompt; the skip must
// track that canonical contract (guarded by
// TestConductorIsTheOnlyCatalogOnlyGuidanceTarget).
func isCatalogOnlyGuidanceTarget(agent model.AgentID) bool {
	return agent == model.AgentConductor
}

// routingDeliveryKind names the three scopes an agent actually loads guidance
// from. Writing one markdown file for every adapter would land the guidance
// where the agent never reads it, or inside a template its own installer
// rewrites on the next sync.
type routingDeliveryKind int

const (
	deliveryPromptSection routingDeliveryKind = iota
	deliveryJinjaModule
	deliveryOrchestratorPrompt
	deliveryAntigravityCLISplit
)

// routingDelivery is the single resolution of how and where guidance reaches
// one agent. It is intentionally pure so the path reporter and the injector can
// share it without the reporter gaining write side effects.
type routingDelivery struct {
	kind    routingDeliveryKind
	adapter agents.Adapter
	// bootstrapper is set only for deliveryJinjaModule, where the router
	// template must be restored before the module is worth writing.
	bootstrapper templateBootstrapper
	paths        []string
}

// resolveRoutingDelivery selects the delivery strategy and its target paths.
//
// It fails closed on anything that would make the guidance unreachable, because
// an unreachable target is exactly the failure this component exists to
// prevent — and reporting a path the injector cannot write is the same defect
// seen from the backup side.
func resolveRoutingDelivery(targetDir string, agent model.AgentID, options RoutingOptions) (routingDelivery, error) {
	if strings.TrimSpace(targetDir) == "" {
		return routingDelivery{}, fmt.Errorf("%w: %q", ErrInvalidTarget, targetDir)
	}

	adapter, err := agents.NewAdapter(agent)
	if err != nil {
		return routingDelivery{}, fmt.Errorf("resolve routing guidance adapter for %q: %w", agent, err)
	}

	switch {
	case DeliversThroughOrchestratorPrompt(agent):
		settingsPath := options.SettingsPath
		if strings.TrimSpace(settingsPath) == "" {
			settingsPath = adapter.SettingsPath(targetDir)
		}
		if strings.TrimSpace(settingsPath) == "" {
			return routingDelivery{}, fmt.Errorf("%w: adapter %q exposes no settings path", ErrInvalidTarget, agent)
		}
		return routingDelivery{kind: deliveryOrchestratorPrompt, adapter: adapter, paths: []string{settingsPath}}, nil

	case adapter.SystemPromptStrategy() == model.StrategyJinjaModules:
		bootstrapper, ok := adapter.(templateBootstrapper)
		if !ok {
			return routingDelivery{}, fmt.Errorf("%w: adapter %q renders Jinja modules but cannot bootstrap its router template", ErrUnloadableGuidance, agent)
		}
		configDir := adapter.GlobalConfigDir(targetDir)
		if strings.TrimSpace(configDir) == "" {
			return routingDelivery{}, fmt.Errorf("%w: adapter %q exposes no global config dir", ErrInvalidTarget, agent)
		}
		return routingDelivery{
			kind:         deliveryJinjaModule,
			adapter:      adapter,
			bootstrapper: bootstrapper,
			paths:        []string{filepath.Join(configDir, routingModuleFile), adapter.SystemPromptFile(targetDir)},
		}, nil

	case agent == model.AgentAntigravityCLI:
		rulesDir := adapter.SystemPromptDir(targetDir)
		if strings.TrimSpace(rulesDir) == "" {
			return routingDelivery{}, fmt.Errorf("%w: adapter %q exposes no system prompt dir", ErrInvalidTarget, agent)
		}
		// Return the canonical first part path; the split writer will generate
		// the full part set from this base.
		return routingDelivery{
			kind:    deliveryAntigravityCLISplit,
			adapter: adapter,
			paths:   []string{filepath.Join(rulesDir, "gentle-ai-routing.md")},
		}, nil

	default:
		promptPath := adapter.SystemPromptFile(targetDir)
		if strings.TrimSpace(promptPath) == "" {
			return routingDelivery{}, fmt.Errorf("%w: adapter %q exposes no system prompt file", ErrInvalidTarget, agent)
		}
		return routingDelivery{kind: deliveryPromptSection, adapter: adapter, paths: []string{promptPath}}, nil
	}
}

// DeliversThroughOrchestratorPrompt reports whether an agent reads its
// always-on instructions from the managed orchestrator agent definition inside
// its settings document rather than from a global prompt file. For these
// adapters the global prompt file is a separate, non-always-loaded scope, so
// guidance placed there would simply never reach the model. It is exported so
// installers can resolve the delivery root for these agents without keeping a
// second copy of the agent list that could drift from this dispatch.
func DeliversThroughOrchestratorPrompt(agent model.AgentID) bool {
	return agent == model.AgentOpenCode || agent == model.AgentKilocode
}

// guidanceMerge folds every managed guidance section into existing content.
type guidanceMerge func(existing string) string

// antigravityCLIRuleCap is the per-file byte limit enforced by agy (1.2.16).
// agy truncates any rule file at 24,000 bytes on line boundaries (builtin docs
// rules.md:29-37). Files must be strictly less than this limit.
const antigravityCLIRuleCap = 24000

// antigravityCLIRuleFrontmatter is the YAML frontmatter agy requires on every
// plugin rule file other than AGENTS.md: without a valid trigger, agy drops
// the file on every turn as an unspecified trigger. AGENTS.md itself must
// stay plain markdown because agy always loads it unconditionally.
const antigravityCLIRuleFrontmatter = `---
trigger: always_on
description: Gentle AI ODD workflow, delegation routing, test-first policy, and RDD review boundaries
---`

// antigravityCLIRoutingMarker is the ownership marker that identifies
// Gentle AI routing rule files for stale-part cleanup.
const antigravityCLIRoutingMarker = "<!-- gentle-ai:agent-routing -->"

// injectAntigravityCLISplit writes the routing render split into N parts at
// H3 (### ) boundaries, each carrying the same frontmatter and ownership
// marker. Part 1 is gentle-ai-routing.md, parts 2..N are gentle-ai-routing-2.md,
// etc. After writing the current part set, stale marked parts are removed.
func injectAntigravityCLISplit(targetDir string, delivery routingDelivery, rendered string) (Result, error) {
	rulesDir := delivery.adapter.SystemPromptDir(targetDir)
	if strings.TrimSpace(rulesDir) == "" {
		return Result{}, fmt.Errorf("%w: adapter %q exposes no system prompt dir", ErrInvalidTarget, delivery.adapter.Agent())
	}

	// Split the rendered content into parts
	parts, err := splitRoutingForAntigravityCLI(rendered)
	if err != nil {
		return Result{}, err
	}

	writtenPaths := make([]string, 0, len(parts))
	anyChanged := false
	for i, part := range parts {
		var partPath string
		if i == 0 {
			partPath = filepath.Join(rulesDir, "gentle-ai-routing.md")
		} else {
			partPath = filepath.Join(rulesDir, fmt.Sprintf("gentle-ai-routing-%d.md", i+1))
		}

		// Wrap part with marker and prepend frontmatter
		wrapped := filemerge.InjectMarkdownSection("", RoutingSectionID, part)
		withFrontmatter := filemerge.PrependYAMLFrontmatter(wrapped, antigravityCLIRuleFrontmatter)

		writeResult, err := filemerge.WriteFileAtomic(partPath, []byte(withFrontmatter), filemerge.ExistingFileMode(partPath, 0o644))
		if err != nil {
			return Result{}, err
		}
		if writeResult.Changed {
			anyChanged = true
		}
		writtenPaths = append(writtenPaths, partPath)
	}

	// Stale-part cleanup: remove gentle-ai-routing*.md files that have our
	// ownership marker but are not in the written set.
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return Result{}, err
	}
	writtenSet := make(map[string]struct{}, len(writtenPaths))
	for _, p := range writtenPaths {
		writtenSet[filepath.Base(p)] = struct{}{}
	}

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "gentle-ai-routing") || !strings.HasSuffix(name, ".md") {
			continue
		}
		if _, ok := writtenSet[name]; ok {
			continue
		}
		// Check if file contains our ownership marker
		fullPath := filepath.Join(rulesDir, name)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue // Best effort
		}
		if strings.Contains(string(data), antigravityCLIRoutingMarker) {
			_ = os.Remove(fullPath) // Best effort cleanup
		}
	}

	return Result{Changed: anyChanged, Files: writtenPaths}, nil
}

// splitRoutingForAntigravityCLI splits the rendered routing at H3 (### )
// boundaries with greedy in-order packing. Each part body (without frontmatter
// and marker) must be < antigravityCLIRuleCap minus overhead.
// Returns parts in order, each as a string (the body content for one part).
func splitRoutingForAntigravityCLI(rendered string) ([]string, error) {
	// Split at H3 boundaries (\n### ). The preamble (before first H3) is the first block.
	blocks := strings.Split(rendered, "\n### ")
	if len(blocks) == 0 {
		return []string{rendered}, nil
	}

	// Restore the "### " prefix for blocks 1..N
	for i := 1; i < len(blocks); i++ {
		blocks[i] = "### " + blocks[i]
	}

	// Calculate overhead per part: frontmatter + marker open/close + newlines
	frontmatterBytes := len(antigravityCLIRuleFrontmatter) + 1 // +1 for trailing \n after frontmatter
	markerOpen := "<!-- gentle-ai:agent-routing -->\n"
	markerClose := "\n<!-- /gentle-ai:agent-routing -->\n"
	overhead := frontmatterBytes + len(markerOpen) + len(markerClose)
	bodyBudget := antigravityCLIRuleCap - overhead - 1 // -1 for safety margin

	if bodyBudget <= 0 {
		return nil, fmt.Errorf("antigravityCLIRuleCap %d too small for frontmatter+marker overhead %d", antigravityCLIRuleCap, overhead)
	}

	var parts []string
	var currentPart strings.Builder
	currentSize := 0

	for _, block := range blocks {
		blockSize := len(block)
		// If adding this block would exceed budget, finalize current part
		if currentSize > 0 && currentSize+1+blockSize > bodyBudget { // +1 for \n between blocks
			parts = append(parts, strings.TrimRight(currentPart.String(), "\n"))
			currentPart.Reset()
			currentSize = 0
		}

		// If single block exceeds budget, split it further
		if blockSize > bodyBudget {
			if currentSize > 0 {
				parts = append(parts, strings.TrimRight(currentPart.String(), "\n"))
				currentPart.Reset()
				currentSize = 0
			}
			subParts, err := splitLargeBlock(block, bodyBudget)
			if err != nil {
				return nil, err
			}
			parts = append(parts, subParts...)
			continue
		}

		if currentSize > 0 {
			currentPart.WriteString("\n")
			currentSize++
		}
		currentPart.WriteString(block)
		currentSize += blockSize
	}

	if currentSize > 0 {
		parts = append(parts, strings.TrimRight(currentPart.String(), "\n"))
	}

	if len(parts) == 0 {
		parts = []string{""}
	}

	return parts, nil
}

// splitLargeBlock splits a single block that exceeds the body budget.
// First tries paragraph (\n\n) boundaries, then list-item boundaries.
// If a single line still exceeds budget, returns an error.
func splitLargeBlock(block string, budget int) ([]string, error) {
	// Try paragraph split first
	paragraphs := strings.Split(block, "\n\n")
	if len(paragraphs) > 1 {
		var parts []string
		var current strings.Builder
		currentSize := 0
		for _, para := range paragraphs {
			paraSize := len(para)
			if currentSize > 0 && currentSize+2+paraSize > budget { // +2 for \n\n
				parts = append(parts, strings.TrimRight(current.String(), "\n"))
				current.Reset()
				currentSize = 0
			}
			if paraSize > budget {
				// Paragraph still too large, try list-item split
				if currentSize > 0 {
					parts = append(parts, strings.TrimRight(current.String(), "\n"))
					current.Reset()
					currentSize = 0
				}
				subParts, err := splitByListItems(para, budget)
				if err != nil {
					return nil, err
				}
				parts = append(parts, subParts...)
				continue
			}
			if currentSize > 0 {
				current.WriteString("\n\n")
				currentSize += 2
			}
			current.WriteString(para)
			currentSize += paraSize
		}
		if currentSize > 0 {
			parts = append(parts, strings.TrimRight(current.String(), "\n"))
		}
		return parts, nil
	}

	// Try list-item split (lines starting with - , * , or digits)
	return splitByListItems(block, budget)
}

// splitByListItems splits at top-level list item boundaries.
func splitByListItems(block string, budget int) ([]string, error) {
	lines := strings.Split(block, "\n")
	var parts []string
	var current strings.Builder
	currentSize := 0

	for _, line := range lines {
		lineSize := len(line) + 1 // +1 for \n
		isListItem := strings.HasPrefix(strings.TrimSpace(line), "- ") ||
			strings.HasPrefix(strings.TrimSpace(line), "* ") ||
			(len(line) > 0 && line[0] >= '1' && line[0] <= '9' && strings.HasPrefix(strings.TrimSpace(line)[1:], ". "))

		if currentSize > 0 && isListItem && currentSize+lineSize > budget {
			parts = append(parts, strings.TrimRight(current.String(), "\n"))
			current.Reset()
			currentSize = 0
		}

		if lineSize > budget {
			return nil, fmt.Errorf("single line exceeds body budget %d bytes: %d bytes (line: %.80q)", budget, lineSize, line)
		}

		if currentSize > 0 {
			current.WriteString("\n")
			currentSize++
		}
		current.WriteString(line)
		currentSize += len(line)
	}

	if currentSize > 0 {
		parts = append(parts, strings.TrimRight(current.String(), "\n"))
	}
	return parts, nil
}

// injectPromptSection is the default delivery: managed marker sections inside
// the adapter's own system prompt file.
func injectPromptSection(delivery routingDelivery, merge guidanceMerge) (Result, error) {
	promptPath := delivery.paths[0]

	existing, err := readFileOrEmpty(promptPath)
	if err != nil {
		return Result{}, err
	}

	updated := merge(existing)

	// Antigravity CLI delivers routing through a dedicated plugin rule file,
	// not AGENTS.md, so the file needs always_on frontmatter to load at all.
	if delivery.adapter.Agent() == model.AgentAntigravityCLI {
		updated = filemerge.PrependYAMLFrontmatter(updated, antigravityCLIRuleFrontmatter)
	}

	writeResult, err := filemerge.WriteFileAtomic(promptPath, []byte(updated), filemerge.ExistingFileMode(promptPath, 0o644))
	if err != nil {
		return Result{}, err
	}

	return Result{Changed: writeResult.Changed, Files: []string{promptPath}}, nil
}

// injectJinjaModule delivers guidance as a standalone include module.
//
// A Jinja adapter's system prompt file is a router that its installer rewrites
// verbatim from an embedded asset on every run, so anything injected into that
// file is destroyed by the next sync. The module survives because the router
// only references it.
func injectJinjaModule(targetDir string, delivery routingDelivery, agent model.AgentID, merge guidanceMerge) (Result, error) {
	hubPath := delivery.adapter.SystemPromptFile(targetDir)
	hubBefore, err := readBytesOrEmpty(hubPath)
	if err != nil {
		return Result{}, err
	}

	// Bootstrap first: the module is only ever read through the router template,
	// so the template must exist before the module is worth writing.
	if err := delivery.bootstrapper.BootstrapTemplate(targetDir); err != nil {
		return Result{}, fmt.Errorf("bootstrap routing guidance template for %q: %w", agent, err)
	}

	if err := requireModuleIsIncluded(delivery.adapter, agent, targetDir); err != nil {
		return Result{}, err
	}

	modulePath := delivery.paths[0]
	existing, err := readFileOrEmpty(modulePath)
	if err != nil {
		return Result{}, err
	}

	updated := merge(existing)

	writeResult, err := filemerge.WriteFileAtomic(modulePath, []byte(updated), filemerge.ExistingFileMode(modulePath, 0o644))
	if err != nil {
		return Result{}, err
	}

	// Some Jinja-module adapters (current kimi-code) read a plain AGENTS.md hub
	// rather than evaluating {% include %} at runtime. Refresh the hub after the
	// module write so those adapters load the newly written module content while
	// legacy Jinja runtimes retain their router template unchanged.
	if err := delivery.bootstrapper.BootstrapTemplate(targetDir); err != nil {
		return Result{}, fmt.Errorf("refresh routing guidance template for %q: %w", agent, err)
	}
	hubAfter, err := readBytesOrEmpty(hubPath)
	if err != nil {
		return Result{}, err
	}

	return Result{Changed: writeResult.Changed || !bytes.Equal(hubBefore, hubAfter), Files: delivery.paths}, nil
}

// requireModuleIsIncluded verifies the freshly bootstrapped router template
// references the guidance module. Without the reference the module is inert, so
// failing here is preferable to reporting a successful install of guidance no
// agent will ever read.
func requireModuleIsIncluded(adapter agents.Adapter, agent model.AgentID, targetDir string) error {
	templatePath := adapter.SystemPromptFile(targetDir)
	if strings.TrimSpace(templatePath) == "" {
		return fmt.Errorf("%w: adapter %q exposes no router template", ErrUnloadableGuidance, agent)
	}

	template, err := readFileOrEmpty(templatePath)
	if err != nil {
		return err
	}
	if !strings.Contains(template, routingModuleFile) {
		return fmt.Errorf("%w: router template %q does not include %q", ErrUnloadableGuidance, templatePath, routingModuleFile)
	}
	return nil
}

// injectOrchestratorPrompt delivers guidance inside the managed orchestrator
// agent definition of the adapter's settings document — the always-loaded scope
// for the OpenCode family. Every other key in that document is preserved.
func injectOrchestratorPrompt(delivery routingDelivery, agent model.AgentID, merge guidanceMerge) (Result, error) {
	settingsPath := delivery.paths[0]

	raw, err := readBytesOrEmpty(settingsPath)
	if err != nil {
		return Result{}, err
	}

	// Decode before merging: a document we cannot parse must abort the install
	// instead of being silently rebuilt from an empty base, which would discard
	// the user's entire agent configuration.
	settings, err := filemerge.UnmarshalJSONObject(raw)
	if err != nil {
		return Result{}, fmt.Errorf("%w %q: %w", ErrUnreadableSettings, settingsPath, err)
	}

	existingPrompt, err := managedOrchestratorPrompt(settings, settingsPath)
	if err != nil {
		return Result{}, err
	}

	updatedPrompt := merge(existingPrompt)

	overlay, err := json.Marshal(map[string]any{
		"agent": map[string]any{
			opencodedefault.ManagedAgent: map[string]any{"prompt": updatedPrompt},
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("encode routing guidance overlay for %q: %w", agent, err)
	}

	merged, err := filemerge.MergeJSONObjects(raw, overlay)
	if err != nil {
		return Result{}, fmt.Errorf("merge routing guidance into %q: %w", settingsPath, err)
	}

	writeResult, err := filemerge.WriteFileAtomic(settingsPath, merged, filemerge.ExistingFileMode(settingsPath, 0o644))
	if err != nil {
		return Result{}, err
	}

	return Result{Changed: writeResult.Changed, Files: []string{settingsPath}}, nil
}

// managedOrchestratorPrompt reads the current prompt of the managed
// orchestrator agent. A missing agent map, agent, or prompt is a legitimate
// first install and yields empty content; a value of an unexpected type is not
// something this component may overwrite, so it fails closed.
func managedOrchestratorPrompt(settings map[string]any, settingsPath string) (string, error) {
	agentsRaw, ok := settings["agent"]
	if !ok || agentsRaw == nil {
		return "", nil
	}
	agentsMap, ok := agentsRaw.(map[string]any)
	if !ok {
		return "", fmt.Errorf("%w %q: %q is not an object", ErrUnreadableSettings, settingsPath, "agent")
	}

	orchestratorRaw, ok := agentsMap[opencodedefault.ManagedAgent]
	if !ok || orchestratorRaw == nil {
		return "", nil
	}
	orchestratorMap, ok := orchestratorRaw.(map[string]any)
	if !ok {
		return "", fmt.Errorf("%w %q: agent %q is not an object", ErrUnreadableSettings, settingsPath, opencodedefault.ManagedAgent)
	}

	promptRaw, ok := orchestratorMap["prompt"]
	if !ok || promptRaw == nil {
		return "", nil
	}
	prompt, ok := promptRaw.(string)
	if !ok {
		return "", fmt.Errorf("%w %q: agent %q has a non-string prompt", ErrUnreadableSettings, settingsPath, opencodedefault.ManagedAgent)
	}
	return prompt, nil
}

// readFileOrEmpty treats a missing prompt file as empty content: the first
// install legitimately has nothing to merge into.
func readFileOrEmpty(path string) (string, error) {
	data, err := readBytesOrEmpty(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func readBytesOrEmpty(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	return data, nil
}
