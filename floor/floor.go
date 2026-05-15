// Package floor defines floor index constants and one BGSpec floors[] entry
// (nodes + edges at a given abstraction level).
package floor

import (
	"github.com/BumbleGrid/bgbase/edge"
	"github.com/BumbleGrid/bgbase/node"
)

// Content is one floor entry in the root floors array (nodes + edges at that abstraction level).
type Content struct {
	Floor       int            `json:"floor"`
	Label       string         `json:"label"`
	Description string         `json:"description"`
	Nodes       []node.Wrapper `json:"nodes"`
	Edges       []edge.Wrapper `json:"edges"`
	Meta        *BlockMeta     `json:"meta,omitempty"`
}

// BlockMeta is optional floor-level metadata on the root floors[] items.
type BlockMeta struct {
	ExtractedAt      string `json:"extractedAt,omitempty"`
	ExtractorVersion string `json:"extractorVersion,omitempty"`
	Notes            string `json:"notes,omitempty"`
}
