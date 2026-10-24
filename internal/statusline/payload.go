// Package statusline renders the Gentle AI status line for the Antigravity
// CLI (agy). agy pipes a state JSON payload to the registered statusLine
// command on every agent state change and renders stdout with full ANSI
// support. Rendering happens in-process inside the gentle-ai binary — the
// same binary the plugin hooks already depend on — so there is no external
// parser (no jq) and no shell dependency (Windows-portable).
package statusline

import "encoding/json"

// Model carries the active model identity.
type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Effort      string `json:"effort,omitempty"`
}

// Workspace carries the session directories.
type Workspace struct {
	CurrentDir string `json:"current_dir"`
	ProjectDir string `json:"project_dir"`
}

// ContextWindow carries the token usage snapshot.
type ContextWindow struct {
	TotalInputTokens    int     `json:"total_input_tokens"`
	TotalOutputTokens   int     `json:"total_output_tokens"`
	ContextWindowSize   int     `json:"context_window_size"`
	UsedPercentage      float64 `json:"used_percentage"`
	RemainingPercentage float64 `json:"remaining_percentage"`
}

// Quota is one named usage window bucket.
type Quota struct {
	RemainingFraction float64 `json:"remaining_fraction"`
	ResetTime         string  `json:"reset_time"`
	ResetInSeconds    int64   `json:"reset_in_seconds"`
}

// VCS carries version-control state; Type reports git/jj/hg.
type VCS struct {
	Type   string `json:"type"`
	Branch string `json:"branch"`
	Client string `json:"client"`
	Dirty  bool   `json:"dirty"`
}

// Sandbox carries the terminal sandbox state.
type Sandbox struct {
	Enabled      bool `json:"enabled"`
	AllowNetwork bool `json:"allow_network"`
}

// Vim carries the vim editor mode when enabled.
type Vim struct {
	Mode string `json:"mode"`
}

// Payload mirrors the agy status-line JSON contract. Every field is optional;
// unknown fields are ignored. The subagents entry shape is not documented
// upstream, so only the array length is consumed.
type Payload struct {
	Cwd               string           `json:"cwd"`
	ConversationID    string           `json:"conversation_id"`
	ConversationTitle string           `json:"conversation_title"`
	Model             Model            `json:"model"`
	Workspace         Workspace        `json:"workspace"`
	ContextWindow     ContextWindow    `json:"context_window"`
	Cost              *float64         `json:"cost"`
	Quota             map[string]Quota `json:"quota"`
	AgentState        string           `json:"agent_state"`
	VCS               VCS              `json:"vcs"`
	Sandbox           Sandbox          `json:"sandbox"`
	ArtifactCount     int              `json:"artifact_count"`
	PlanTier          string           `json:"plan_tier"`
	PendingInputCount int              `json:"pending_input_count"`
	TaskCount         int              `json:"task_count"`
	Subagents         json.RawMessage  `json:"subagents"`
	TerminalWidth     int              `json:"terminal_width"`
	ExecutionMode     string           `json:"execution_mode"`
	// CycleMode is agy 1.4.1's live execution-mode field (undocumented
	// upstream): "plan"/"accept-edits" when set, absent in the default
	// (auto) mode — verified by payload captures in L23.
	CycleMode string `json:"cycle_mode"`
	Effort    string `json:"effort,omitempty"`
	Vim       Vim    `json:"vim"`
}

// SubagentCount reports the number of running subagents; a missing or
// non-array value counts as zero.
func (p *Payload) SubagentCount() int {
	if len(p.Subagents) == 0 {
		return 0
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(p.Subagents, &entries); err != nil {
		return 0
	}
	return len(entries)
}

// ParsePayload decodes one agy status-line payload.
func ParsePayload(raw []byte) (*Payload, error) {
	var payload Payload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}
