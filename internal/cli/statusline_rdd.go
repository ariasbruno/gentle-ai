package cli

// Statusline RDD review states (S15): beyond the review switch, the segment
// shows the state of the repo's active review authority with the Gentle
// Shell label set (Review in progress / ✓ Approved / Fixing findings… / …).
// Resolution reuses the stop hook's in-process path and caches the label per
// repository root for thirty seconds; the switch itself reads through the
// same read-only resolver the review command uses.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/gentleman-programming/gentle-ai/v4/internal/reviewtransaction"
)

const (
	rddStateCacheTTL = 5 * time.Second
	rddStateDeadline = 2 * time.Second
)

type rddStateCacheRecord struct {
	Cwd   string `json:"cwd"`
	TS    int64  `json:"ts"`
	Label string `json:"label"`
}

func rddStateCachePath(home string) string {
	return filepath.Join(home, ".gentle-ai", "statusline-rdd-cache.json")
}

// rddStateLabel maps one review authority state to the Gentle Shell display
// label. Unknown states must not look like a successful switch-only read.
func rddStateLabel(state reviewtransaction.State) string {
	switch state {
	case reviewtransaction.StateUnreviewed:
		return "Not reviewed yet"
	case reviewtransaction.StateReviewing,
		reviewtransaction.StateValidating,
		reviewtransaction.StateFindingsFrozen,
		reviewtransaction.StateEvidenceClassified:
		return "Review in progress"
	case reviewtransaction.StateFixRequired,
		reviewtransaction.StateCorrectionRequired,
		reviewtransaction.StateFixing,
		reviewtransaction.StateFixValidating:
		return "Fixing findings…"
	case reviewtransaction.StateReadyFinalVerification,
		reviewtransaction.StateFinalVerifying:
		return "Verifying…"
	case reviewtransaction.StateApproved:
		return "✓ Approved"
	case reviewtransaction.StateEscalated:
		return "Needs your decision"
	case reviewtransaction.StateInvalidated:
		return "Outdated · code changed"
	default:
		return "Review unavailable"
	}
}

// resolveStatusLineRDD builds the RDD segment display: `off` when the switch
// is off, the active authority's state label when one exists (cached per
// root), and the plain `on` otherwise. Errors show `Review unavailable` and never
// blocks the line.
func resolveStatusLineRDD(home, cwd string) string {
	ctx, cancel := context.WithTimeout(context.Background(), rddStateDeadline)
	defer cancel()

	mode, err := ReviewModeStatus(ctx, cwd)
	if err != nil || mode.Effective != reviewtransaction.RDDModeOn {
		if err != nil {
			return "Review unavailable"
		}
		return "off"
	}

	candidateRoots := antigravityCandidateRoots(ctx, cwd, home, "")
	if len(candidateRoots) == 0 {
		candidateRoots = []string{cwd}
	}
	if label, ok := readRDDStateCache(home, candidateRoots[0]); ok {
		return label
	}

	label := "on"
	hasValidRoot := false

	for _, root := range candidateRoots {
		_, targetIdentity, status, ok, resolveErr := antigravityResolveTargetIdentity(ctx, root)
		if resolveErr != nil || ctx.Err() != nil {
			continue
		}
		hasValidRoot = true

		if ok && targetIdentity != "" {
			// 1. If this exact candidate was already reviewed and acknowledged, show Approved.
			if consumed, err := reviewtransaction.CompactTargetConsumed(ctx, root, targetIdentity); err == nil && consumed {
				label = "✓ Approved"
				writeRDDStateCache(home, candidateRoots[0], label)
				return label
			}

			// 2. Inspect active compact authority leaves matching this target or active lineage.
			leaves, leavesErr := reviewtransaction.CompactAuthorityLeaves(ctx, root)
			if leavesErr == nil && len(leaves) > 0 {
				var activeRecord *reviewtransaction.CompactRecord
				for _, store := range leaves {
					record, loadErr := store.LoadContext(ctx)
					if loadErr != nil {
						continue
					}
					matchesTarget := record.State.InitialSnapshot.Identity == targetIdentity ||
						record.State.CurrentSnapshot.Identity == targetIdentity
					matchesLineage := status.NextTransition.Execute != nil &&
						status.NextTransition.Execute.Binding.LineageID != "" &&
						status.NextTransition.Execute.Binding.LineageID == record.State.LineageID

					if matchesTarget || matchesLineage {
						if activeRecord == nil || record.State.Generation > activeRecord.State.Generation {
							recCopy := record
							activeRecord = &recCopy
						}
					}
				}
				if activeRecord != nil {
					label = rddStateLabel(activeRecord.State.State)
					writeRDDStateCache(home, candidateRoots[0], label)
					return label
				}
			}
		}
	}

	if !hasValidRoot && ctx.Err() != nil {
		return "Review unavailable"
	}
	writeRDDStateCache(home, candidateRoots[0], label)
	return label
}

func readRDDStateCache(home, cwd string) (string, bool) {
	raw, err := os.ReadFile(rddStateCachePath(home))
	if err != nil {
		return "", false
	}
	var record rddStateCacheRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return "", false
	}
	if record.Cwd == "" || record.Cwd != cwd || time.Since(time.Unix(record.TS, 0)) > rddStateCacheTTL {
		return "", false
	}
	return record.Label, true
}

func writeRDDStateCache(home, cwd, label string) {
	if home == "" || cwd == "" {
		return
	}
	record := rddStateCacheRecord{Cwd: cwd, TS: time.Now().Unix(), Label: label}
	encoded, err := json.Marshal(record)
	if err != nil {
		return
	}
	path := rddStateCachePath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, encoded, 0o644); err != nil {
		return
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
	}
}
