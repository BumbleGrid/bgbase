// Package merge deep-merges an extractor-produced floor.Content (Floor 0) into a
// previously-persisted graph.BGSpecDocument. The algorithm is intentionally narrow:
// Floor 0 nodes/edges/meta are replaced from the extractor output; Floors 1..N and
// styleMap are passthrough; document.updatedAt is bumped.
//
// See merge_test.go for the canonical behaviour spec.
package merge

import (
	"time"

	"github.com/BumbleGrid/bgbase/graph"
)

// MergeOptions tunes a Merge call. Zero value is valid (AutoArrange = false,
// PreserveFloor0Meta = false, Now = time.Now().UTC()).
type MergeOptions struct {
	Now                time.Time
	AutoArrange        bool
	PreserveFloor0Meta bool
}

// MergeResult is the output of one Merge call.
type MergeResult struct {
	Document       graph.BGSpecDocument
	Diff           MergeDiff
	AutoArrangedAt *time.Time
}

// MergeDiff summarises the changes Merge applied to Floor 0.
type MergeDiff struct {
	Floor0          FloorDiff `json:"floor0"`
	StyleMapTouched bool      `json:"styleMapTouched"`
}

// FloorDiff is the per-floor structured change set used by MergeDiff.
type FloorDiff struct {
	NodesAdded   []string `json:"nodesAdded"`
	NodesRemoved []string `json:"nodesRemoved"`
	NodesChanged []string `json:"nodesChanged"`
	EdgesAdded   []string `json:"edgesAdded"`
	EdgesRemoved []string `json:"edgesRemoved"`
	EdgesChanged []string `json:"edgesChanged"`
}

// HasChanges reports whether the floor changed in any way.
func (diff FloorDiff) HasChanges() bool {
	return len(diff.NodesAdded)+len(diff.NodesRemoved)+len(diff.NodesChanged) > 0 ||
		len(diff.EdgesAdded)+len(diff.EdgesRemoved)+len(diff.EdgesChanged) > 0
}
