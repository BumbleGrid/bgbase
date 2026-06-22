package merge

import (
	"fmt"
	"slices"
	"time"

	"github.com/BumbleGrid/bgbase/edge"
	"github.com/BumbleGrid/bgbase/floor"
	"github.com/BumbleGrid/bgbase/graph"
	"github.com/BumbleGrid/bgbase/node"
)

// Merge applies the extractor-produced Floor 0 content to previous and returns the
// merged document plus a structured diff.
func Merge(previous graph.BGSpecDocument, extracted floor.Content, options MergeOptions) (MergeResult, error) {
	if extracted.Floor != 0 {
		return MergeResult{}, fmt.Errorf("bgbase/merge: extracted floor must be 0, got %d", extracted.Floor)
	}
	if len(previous.Floors) == 0 {
		return MergeResult{}, fmt.Errorf("bgbase/merge: previous document has no floors; need at least the floor 0 slot")
	}
	if previous.Floors[0].Floor != 0 {
		return MergeResult{}, fmt.Errorf("bgbase/merge: previous floors[0].floor must be 0, got %d", previous.Floors[0].Floor)
	}

	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	merged := previous
	merged.Floors = slices.Clone(previous.Floors)

	previousFloor0 := previous.Floors[0]
	nextFloor0 := mergeFloor0(previousFloor0, extracted, options, now)
	merged.Floors[0] = nextFloor0
	merged.Document.UpdatedAt = now.UTC().Format(time.RFC3339)

	diff := MergeDiff{
		Floor0: compareFloors(previousFloor0, nextFloor0),
	}

	result := MergeResult{Document: merged, Diff: diff}

	if options.AutoArrange && previous.StyleMap == nil {
		arranged := floor.AutoArrangeStyleMap(nextFloor0)
		merged.StyleMap = &arranged
		result.Document = merged
		stamp := now
		result.AutoArrangedAt = &stamp
		result.Diff.StyleMapTouched = true
	}

	return result, nil
}

func mergeFloor0(previousFloor0, extracted floor.Content, options MergeOptions, now time.Time) floor.Content {
	return floor.Content{
		Floor:       0,
		Label:       chooseString(previousFloor0.Label, extracted.Label),
		Description: chooseString(previousFloor0.Description, extracted.Description),
		Nodes:       cloneNodes(extracted.Nodes),
		Edges:       cloneEdges(extracted.Edges),
		Meta:        mergeFloor0Meta(previousFloor0.Meta, extracted.Meta, options, now),
	}
}

func chooseString(previous, extracted string) string {
	if previous == "" {
		return extracted
	}
	return previous
}

func mergeFloor0Meta(previous, extracted *floor.BlockMeta, options MergeOptions, now time.Time) *floor.BlockMeta {
	if !options.PreserveFloor0Meta {
		if extracted == nil {
			return &floor.BlockMeta{ExtractedAt: now.UTC().Format(time.RFC3339)}
		}
		stamped := *extracted
		if stamped.ExtractedAt == "" {
			stamped.ExtractedAt = now.UTC().Format(time.RFC3339)
		}
		return &stamped
	}
	merged := floor.BlockMeta{}
	if previous != nil {
		merged = *previous
	}
	if extracted != nil {
		if extracted.ExtractedAt != "" {
			merged.ExtractedAt = extracted.ExtractedAt
		}
		if extracted.ExtractorVersion != "" {
			merged.ExtractorVersion = extracted.ExtractorVersion
		}
		if extracted.Notes != "" {
			merged.Notes = extracted.Notes
		}
	}
	if merged.ExtractedAt == "" {
		merged.ExtractedAt = now.UTC().Format(time.RFC3339)
	}
	return &merged
}

func cloneNodes(in []node.Wrapper) []node.Wrapper {
	if len(in) == 0 {
		return []node.Wrapper{}
	}
	out := make([]node.Wrapper, len(in))
	copy(out, in)
	return out
}

func cloneEdges(in []edge.Wrapper) []edge.Wrapper {
	if len(in) == 0 {
		return []edge.Wrapper{}
	}
	out := make([]edge.Wrapper, len(in))
	copy(out, in)
	return out
}
