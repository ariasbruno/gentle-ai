#!/usr/bin/env bash
set -euo pipefail

# Gentle AI statusline for Antigravity CLI (agy)
# Reads agy's status-line JSON payload from stdin and renders a Gentle Shell-style line.

# Check for jq
if ! command -v jq >/dev/null 2>&1; then
    echo "✿ gentle ai ⟡ install jq for the live status line"
    exit 0
fi

# Read all stdin into a variable (single jq pass pattern)
payload=$(cat)

# Parse config file (second jq pass)
config_file="${HOME}/.gentle-ai/statusline.json"
density="comfortable"
visibility_changes="false"
visibility_rdd="true"
visibility_agents="true"
visibility_usageCost="true"
visibility_modelDetails="true"
visibility_contextWarning="true"

if [[ -f "$config_file" ]]; then
    # Use jq to parse config with defaults, ignore malformed files
    config_density=$(jq -r '.density // "comfortable"' "$config_file" 2>/dev/null || echo "comfortable")
    if [[ "$config_density" =~ ^(comfortable|compact|minimal)$ ]]; then
        density="$config_density"
    fi

    # jq's // treats false as falsy (false // true == true), so explicit
    # false flags are compared instead of coalesced. Malformed files fall
    # back to the defaults below (S4).
    visibility_changes=$(jq -r 'if .visibility.changes == true then "true" else "false" end' "$config_file" 2>/dev/null || echo "false")
    visibility_rdd=$(jq -r 'if .visibility.rdd == false then "false" else "true" end' "$config_file" 2>/dev/null || echo "true")
    visibility_agents=$(jq -r 'if .visibility.agents == false then "false" else "true" end' "$config_file" 2>/dev/null || echo "true")
    visibility_usageCost=$(jq -r 'if .visibility.usageCost == false then "false" else "true" end' "$config_file" 2>/dev/null || echo "true")
    visibility_modelDetails=$(jq -r 'if .visibility.modelDetails == false then "false" else "true" end' "$config_file" 2>/dev/null || echo "true")
    visibility_contextWarning=$(jq -r 'if .visibility.contextWarning == false then "false" else "true" end' "$config_file" 2>/dev/null || echo "true")
fi

# Handle --title mode
if [[ "${1:-}" == "--title" ]]; then
    branch=$(echo "$payload" | jq -r '.vcs.branch // ""')
    model_name=$(echo "$payload" | jq -r '.model.display_name // ""')
    if [[ -n "$branch" && -n "$model_name" ]]; then
        echo "✿ $branch · $model_name"
    elif [[ -n "$branch" ]]; then
        echo "✿ $branch"
    elif [[ -n "$model_name" ]]; then
        echo "✿ $model_name"
    else
        echo "✿"
    fi
    exit 0
fi

# Validate the payload parses as a JSON object before extraction (S6 fallback)
if ! printf '%s' "$payload" | jq -e 'type == "object"' >/dev/null 2>&1; then
    echo "✿ gentle ai ⟡ install jq for the live status line"
    exit 0
fi

# Single jq pass to extract all needed fields from payload (newline-separated values)
# Using // empty for defensive defaults
# Order: AGENT_STATE, CWD, VCS_BRANCH, VCS_DIRTY, MODEL_NAME, EXECUTION_MODE,
#        USED_PERCENTAGE, COST, PLAN_TIER, TERMINAL_WIDTH, TASK_COUNT,
#        SUBAGENTS_COUNT, SANDBOX_ENABLED, SANDBOX_ALLOW_NETWORK, VIM_MODE,
#        CONVERSATION_TITLE, QUOTA_JSON (as compact JSON string)
{
read -r AGENT_STATE
read -r CWD
read -r VCS_BRANCH
read -r VCS_DIRTY
read -r MODEL_NAME
read -r EXECUTION_MODE
read -r USED_PERCENTAGE
read -r COST
read -r PLAN_TIER
read -r TERMINAL_WIDTH
read -r TASK_COUNT
read -r SUBAGENTS_COUNT
read -r SANDBOX_ENABLED
read -r SANDBOX_ALLOW_NETWORK
read -r VIM_MODE
read -r CONVERSATION_TITLE
read -r QUOTA_JSON
} <<< "$(echo "$payload" | jq -r '
    .agent_state // "idle",
    .cwd // "",
    .vcs.branch // "",
    (.vcs.dirty // false | tostring),
    .model.display_name // "",
    .execution_mode // "",
    (.context_window.used_percentage // 0 | tostring),
    (.cost // 0 | tostring),
    .plan_tier // "",
    (.terminal_width // 80 | tostring),
    (.task_count // 0 | tostring),
    (.subagents | if type == "array" then length else 0 end | tostring),
    (.sandbox.enabled // false | tostring),
    (.sandbox.allow_network // false | tostring),
    .vim.mode // "",
    .conversation_title // "",
    (.quota // {} | to_entries | map({key, value}) | @json)
')"

# Default empty strings to safe values
: "${AGENT_STATE:=idle}"
: "${CWD:=$HOME}"
: "${VCS_BRANCH:=}"
: "${VCS_DIRTY:=false}"
: "${MODEL_NAME:=}"
: "${EXECUTION_MODE:=}"
: "${USED_PERCENTAGE:=0}"
# Normalize to a rounded integer percent: bash arithmetic and test operators are
# integer-only ( Gentle Shell renders Math.round ).
if ! [[ "$USED_PERCENTAGE" =~ ^[0-9]+$ ]]; then
    USED_PERCENTAGE=$(awk -v v="$USED_PERCENTAGE" 'BEGIN { printf "%d", int(v + 0.5) }' 2>/dev/null || echo "${USED_PERCENTAGE%%.*}")
    USED_PERCENTAGE="${USED_PERCENTAGE%%.*}"
    [[ "$USED_PERCENTAGE" =~ ^[0-9]+$ ]] || USED_PERCENTAGE=0
fi
: "${COST:=0}"
: "${PLAN_TIER:=}"
: "${TERMINAL_WIDTH:=80}"
: "${TASK_COUNT:=0}"
: "${SUBAGENTS_COUNT:=0}"
: "${SANDBOX_ENABLED:=false}"
: "${SANDBOX_ALLOW_NETWORK:=false}"
: "${VIM_MODE:=}"
: "${CONVERSATION_TITLE:=}"
: "${QUOTA_JSON:={}}"

# Helper: ANSI color codes (16-color palette only)
color_reset=$'\033[0m'
color_accent=$'\033[32m'
color_warning=$'\033[33m'
color_error=$'\033[31m'
color_dim=$'\033[2m'
color_bold=$'\033[1m'

# Helper: get color for percentage (context/quota gauges); accepts decimals, uses integer part
get_gauge_color() {
    local pct="${1%%.*}"
    pct="${pct:-0}"
    if (( pct >= 95 )); then
        echo "$color_error"
    elif (( pct >= 80 )); then
        echo "$color_warning"
    else
        echo "$color_accent"
    fi
}

# Helper: render 8-cell gauge; accepts decimals, uses integer part.
# Pure bash repetition: tr(1) is byte-oriented and corrupts multibyte cells.
render_gauge() {
    local pct="${1%%.*}"
    pct="${pct:-0}"
    local filled=$(( pct * 8 / 100 ))
    if (( filled > 8 )); then filled=8; fi
    local empty=$(( 8 - filled ))
    local cells=""
    local i
    for ((i = 0; i < filled; i++)); do cells+="▰"; done
    for ((i = 0; i < empty; i++)); do cells+="▱"; done
    printf "%s" "$cells"
}

# Helper: shorten cwd with ~
shorten_cwd() {
    local cwd="$1"
    local home="$2"
    if [[ "$cwd" == "$home"* ]]; then
        cwd="~${cwd#$home}"
    fi
    echo "$cwd"
}

# Helper: format cost (Gentle Shell formatCost: 2 decimals >=1, 3 decimals <1)
format_cost() {
    local cost="$1"
    # Use awk for floating point formatting
    awk -v c="$cost" 'BEGIN {
        if (c >= 1) printf "$%.2f", c;
        else printf "$%.3f", c;
    }'
}

# Helper: format quota label from reset_in_seconds
format_quota_label() {
    local reset_sec=$1
    if (( reset_sec >= 604800 )); then # week (7*24*3600)
        echo "week"
    elif (( reset_sec >= 86400 )); then # days
        echo "$(( reset_sec / 86400 ))d"
    elif (( reset_sec >= 3600 )); then # hours
        echo "$(( reset_sec / 3600 ))h"
    else # minutes
        echo "$(( reset_sec / 60 ))m"
    fi
}

# Helper: get RDD status with caching
get_rdd_status() {
    local cwd="$1"
    local cache_file="${HOME}/.gentle-ai/statusline-rdd-cache.json"
    local now=$(date +%s)
    local cached_mode=""
    local cached_ts=0

    # Read cache if exists and valid
    if [[ -f "$cache_file" ]]; then
        cached_mode=$(jq -r '.mode // ""' "$cache_file" 2>/dev/null || echo "")
        cached_ts=$(jq -r '.ts // 0' "$cache_file" 2>/dev/null || echo 0)
        if (( now - cached_ts < 30 )) && [[ -n "$cached_mode" ]]; then
            echo "$cached_mode"
            return
        fi
    fi

    # Check if timeout command exists
    if ! command -v timeout >/dev/null 2>&1; then
        echo "?"
        return
    fi

    # Run gentle-ai review mode status with 0.5s timeout
    local result
    result=$(timeout 0.5 gentle-ai review mode status --json --cwd "$cwd" 2>/dev/null || echo "")

    local mode="?"
    if [[ -n "$result" ]]; then
        mode=$(echo "$result" | jq -r '.status.effective // "?"' 2>/dev/null || echo "?")
        # Pin exact enum literals from RDDMode constants in internal/reviewtransaction/rdd_mode.go:
        # RDDModeOn = "on", RDDModeOff = "off", RDDModeUnset = ""
        # Effective is "on" or "off" (unset defaults to on in resolve logic)
        if [[ "$mode" != "on" && "$mode" != "off" ]]; then
            mode="?"
        fi
    fi

    # Write cache atomically
    local tmp_cache="${cache_file}.tmp"
    mkdir -p "$(dirname "$cache_file")"
    printf '{"ts":%d,"mode":"%s"}\n' "$now" "$mode" > "$tmp_cache" 2>/dev/null || true
    mv -f "$tmp_cache" "$cache_file" 2>/dev/null || true

    echo "$mode"
}

# Helper: get git changes segment
get_changes_segment() {
    local cwd="$1"
    if ! command -v timeout >/dev/null 2>&1; then
        echo ""
        return
    fi
    if ! git -C "$cwd" rev-parse --git-dir >/dev/null 2>&1; then
        echo ""
        return
    fi

    local porcelain_count=0
    local added=0
    local deleted=0

    porcelain_count=$(timeout 0.5 git -C "$cwd" status --porcelain 2>/dev/null | wc -l | tr -d ' ' || echo 0)
    local numstat
    numstat=$(timeout 0.5 git -C "$cwd" diff --numstat 2>/dev/null || echo "")
    if [[ -n "$numstat" ]]; then
        added=$(echo "$numstat" | awk '{sum+=$1} END {print sum+0}')
        deleted=$(echo "$numstat" | awk '{sum+=$2} END {print sum+0}')
    fi

    if (( porcelain_count > 0 || added > 0 || deleted > 0 )); then
        printf "±%d +%d/−%d" "$porcelain_count" "$added" "$deleted"
    fi
}

# Build segments with tags for degradation
declare -a seg_tags
declare -a seg_values

separator=" ${color_dim}⟡${color_reset} "

# 1. Brand (hidden in minimal density)
if [[ "$density" != "minimal" ]]; then
    seg_tags+=("brand")
    seg_values+=("${color_bold}✿ gentle ai${color_reset}")
else
    seg_tags+=("brand")
    seg_values+=("")
fi

# 2. State badge (agent_state enum: idle, thinking, working, tool_use, initializing)
state_badge=""
case "$AGENT_STATE" in
    "idle") state_badge="● ready" ;;
    "thinking") state_badge="◆ thinking" ;;
    "working") state_badge="⚙ working" ;;
    "tool_use") state_badge="🔧 tool" ;;
    "initializing") state_badge="⏳ initializing" ;;
    *) state_badge="○ $AGENT_STATE" ;;
esac
seg_tags+=("state")
seg_values+=("$state_badge")

# 3. Location (cwd with ~, branch, dirty *)
location=$(shorten_cwd "$CWD" "$HOME")
if [[ -n "$VCS_BRANCH" ]]; then
    location="$location $VCS_BRANCH"
fi
if [[ "$VCS_DIRTY" == "true" ]]; then
    location="$location*"
fi
seg_tags+=("location")
seg_values+=("$location")

# 4. Model + execution mode
if [[ "$visibility_modelDetails" == "true" && -n "$MODEL_NAME" ]]; then
    model_seg="$MODEL_NAME"
    if [[ -n "$EXECUTION_MODE" ]]; then
        model_seg="$model_seg · $EXECUTION_MODE"
    fi
    seg_tags+=("model")
    seg_values+=("$model_seg")
else
    seg_tags+=("model")
    seg_values+=("")
fi

# 5. Context gauge
if [[ "$visibility_usageCost" == "true" ]]; then
    ctx_color=$(get_gauge_color "$USED_PERCENTAGE")
    gauge=$(render_gauge "$USED_PERCENTAGE")
    ctx_marker=""
    if [[ "$visibility_contextWarning" == "true" && "$USED_PERCENTAGE" -ge 90 ]]; then
        ctx_marker=" !"
    fi
    ctx_seg="ctx ${ctx_color}${gauge}${color_reset} ${USED_PERCENTAGE}%${ctx_marker}"
    seg_tags+=("ctx")
    seg_values+=("$ctx_seg")
else
    seg_tags+=("ctx")
    seg_values+=("")
fi

# 6. Cost
if [[ "$visibility_usageCost" == "true" ]]; then
    cost_str=$(format_cost "$COST")
    if [[ -n "$PLAN_TIER" ]]; then
        cost_str="$cost_str sub"
    fi
    seg_tags+=("cost")
    seg_values+=("$cost_str")
else
    seg_tags+=("cost")
    seg_values+=("")
fi

# 7. Quota windows (quota is object keyed by bucket id, each value has remaining_fraction, reset_in_seconds)
quota_segments=()
while IFS= read -r entry; do
    [[ -z "$entry" ]] && continue
    remaining_fraction=$(echo "$entry" | jq -r '.value.remaining_fraction // 1')
    reset_in_seconds=$(echo "$entry" | jq -r '.value.reset_in_seconds // 604800')
    used_pct=$(awk -v r="$remaining_fraction" 'BEGIN { u=(1-r)*100; if(u<0) u=0; if(u>100) u=100; printf "%d", int(u + 0.5) }')
    label=$(format_quota_label "$reset_in_seconds")
    q_color=$(get_gauge_color "$used_pct")
    q_gauge=$(render_gauge "$used_pct")
    quota_segments+=("${label} ${q_color}${q_gauge}${color_reset} ${used_pct}%")
done < <(echo "$QUOTA_JSON" | jq -c '.[]' 2>/dev/null)

for qs in "${quota_segments[@]}"; do
    seg_tags+=("quota")
    seg_values+=("$qs")
done
# usageCost:false hides ctx+cost+quota together (S4)
if [[ "$visibility_usageCost" != "true" ]]; then
    for i in "${!seg_tags[@]}"; do
        if [[ "${seg_tags[i]}" == "quota" ]]; then
            seg_values[i]=""
        fi
    done
fi

# 8. Jobs
if [[ "$visibility_agents" == "true" && "$TASK_COUNT" -gt 0 ]]; then
    seg_tags+=("jobs")
    seg_values+=("⧗ $TASK_COUNT jobs")
else
    seg_tags+=("jobs")
    seg_values+=("")
fi

# 9. Subagents
if [[ "$visibility_agents" == "true" && "$SUBAGENTS_COUNT" -gt 0 ]]; then
    seg_tags+=("subagents")
    seg_values+=("✦ $SUBAGENTS_COUNT subagents")
else
    seg_tags+=("subagents")
    seg_values+=("")
fi

# 10. Sandbox
if [[ "$SANDBOX_ENABLED" == "true" ]]; then
    sandbox_seg="⊘ sandbox"
    if [[ "$SANDBOX_ALLOW_NETWORK" == "true" ]]; then
        sandbox_seg="$sandbox_seg·net"
    fi
    seg_tags+=("sandbox")
    seg_values+=("$sandbox_seg")
else
    seg_tags+=("sandbox")
    seg_values+=("")
fi

# 11. Vim mode
if [[ -n "$VIM_MODE" ]]; then
    seg_tags+=("vim")
    seg_values+=("vim.$VIM_MODE")
else
    seg_tags+=("vim")
    seg_values+=("")
fi

# 12. RDD segment
if [[ "$visibility_rdd" == "true" ]]; then
    rdd_mode=$(get_rdd_status "$CWD")
    case "$rdd_mode" in
        "on") rdd_seg="🌹 on" ;;
        "off") rdd_seg="🌹 off" ;;
        *) rdd_seg="🌹 ?" ;;
    esac
    seg_tags+=("rdd")
    seg_values+=("$rdd_seg")
else
    seg_tags+=("rdd")
    seg_values+=("")
fi

# 13. Changes segment
if [[ "$visibility_changes" == "true" ]]; then
    changes_seg=$(get_changes_segment "$CWD")
    if [[ -n "$changes_seg" ]]; then
        seg_tags+=("changes")
        seg_values+=("$changes_seg")
    else
        seg_tags+=("changes")
        seg_values+=("")
    fi
else
    seg_tags+=("changes")
    seg_values+=("")
fi

# Function to measure visible width (strip ANSI)
visible_width() {
    local str="$1"
    local stripped=$(printf "%s" "$str" | sed 's/\x1b\[[0-9;]*m//g')
    printf "%s" "$stripped" | wc -m
}

# Function to build line from current segments (respecting empty values)
build_line() {
    local -n tags_ref=$1
    local -n vals_ref=$2
    local out_segments=()
    for i in "${!tags_ref[@]}"; do
        if [[ -n "${vals_ref[i]}" ]]; then
            out_segments+=("${vals_ref[i]}")
        fi
    done
    # ${arr[*]} joins with only the FIRST character of IFS, so the painted
    # separator string is applied explicitly instead.
    local joined=""
    local first=1
    for seg in "${out_segments[@]}"; do
        if (( first )); then
            first=0
            joined="$seg"
        else
            joined+="$separator$seg"
        fi
    done
    printf '%s\n' "$joined"
}

# Title handling
title="$CONVERSATION_TITLE"

# Try full line with right-aligned title
try_output_with_title() {
    local line="$1"
    if [[ -n "$title" ]]; then
        local line_width=$(visible_width "$line")
        local title_width=$(visible_width "$title")
        if (( line_width + 1 + title_width <= TERMINAL_WIDTH )); then
            local padding=$(( TERMINAL_WIDTH - line_width - title_width ))
            printf "%s%*s%s\n" "$line" "$padding" "" "$title"
            return 0
        fi
    else
        local line_width=$(visible_width "$line")
        if (( line_width <= TERMINAL_WIDTH )); then
            echo "$line"
            return 0
        fi
    fi
    return 1
}

# Initial attempt
line=$(build_line seg_tags seg_values)
if try_output_with_title "$line"; then
    exit 0
fi

# Degradation step 1: compact location (cwd -> last segment, branch clipped to 15)
compact_cwd=$(basename "$CWD")
if [[ "$CWD" == "$HOME"* ]]; then
    compact_cwd="~/${CWD#$HOME/}"
    compact_cwd=$(basename "$compact_cwd")
fi
if [[ "$compact_cwd" == "$HOME" ]]; then
    compact_cwd="~"
fi

compact_branch="$VCS_BRANCH"
if [[ -n "$compact_branch" && ${#compact_branch} -gt 15 ]]; then
    compact_branch="${compact_branch:0:14}…"
fi

compact_location="$compact_cwd"
if [[ -n "$compact_branch" ]]; then
    compact_location="$compact_location $compact_branch"
fi
if [[ "$VCS_DIRTY" == "true" ]]; then
    compact_location="$compact_location*"
fi

# Update location segment
for i in "${!seg_tags[@]}"; do
    if [[ "${seg_tags[i]}" == "location" ]]; then
        seg_values[i]="$compact_location"
        break
    fi
done

line=$(build_line seg_tags seg_values)
if try_output_with_title "$line"; then
    exit 0
fi

# Degradation step 2: drop trailing segments in order
# Order: changes, rdd, vim, sandbox, subagents, jobs, quota, cost, ctx, model, state
# (auxiliary subprocess segments die first; brand and location survive longest)
drop_order=("changes" "rdd" "vim" "sandbox" "subagents" "jobs" "quota" "cost" "ctx" "model" "state")

for drop_tag in "${drop_order[@]}"; do
    if [[ "$drop_tag" == "quota" ]]; then
        # Drop all quota segments
        for i in "${!seg_tags[@]}"; do
            if [[ "${seg_tags[i]}" == "quota" ]]; then
                seg_values[i]=""
            fi
        done
    else
        # Drop the last segment with this tag
        for (( i=${#seg_tags[@]}-1; i>=0; i-- )); do
            if [[ "${seg_tags[i]}" == "$drop_tag" ]]; then
                seg_values[i]=""
                break
            fi
        done
    fi

    line=$(build_line seg_tags seg_values)
    if try_output_with_title "$line"; then
        exit 0
    fi
done

# Final truncation with …
line=$(build_line seg_tags seg_values)
line_width=$(visible_width "$line")
if (( line_width > TERMINAL_WIDTH )); then
    stripped=$(printf "%s" "$line" | sed 's/\x1b\[[0-9;]*m//g')
    if [[ -n "$title" ]]; then
        title_width=$(visible_width "$title")
        avail=$(( TERMINAL_WIDTH - title_width - 1 ))
        if (( avail > 1 )); then
            if (( ${#stripped} > avail )); then
                truncated="${stripped:0:avail-1}…"
                padding=$(( TERMINAL_WIDTH - avail - title_width ))
                printf "%s%*s%s\n" "$truncated" "$padding" "" "$title"
                exit 0
            fi
        fi
    else
        if (( ${#stripped} > TERMINAL_WIDTH )); then
            echo "${stripped:0:TERMINAL_WIDTH-1}…"
            exit 0
        fi
    fi
fi

# Output final line (with title if fits, otherwise without)
try_output_with_title "$line" || echo "$line"