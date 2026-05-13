// Package graph defines the top-level BGSpec document shape (root JSON object).
package graph

import (
	"encoding/json"
	"slices"

	"github.com/BumbleGrid/bgbase/edge"
	"github.com/BumbleGrid/bgbase/floor"
	"github.com/BumbleGrid/bgbase/node"
)

// BGSpecDocument is the root BGSpec JSON payload (bgspec version, document metadata, floors).
type BGSpecDocument struct {
	BGSpec   string          `json:"bgspec"`
	Document DocumentMeta    `json:"document"`
	Floors   []floor.Content `json:"floors"`
}

// DocumentMeta is the required document block on the root BGSpec object.
type DocumentMeta struct {
	Title       string           `json:"title"`
	Company     string           `json:"company"`
	UpdatedAt   string           `json:"updatedAt"`
	Description *string          `json:"description,omitempty"`
	Authors     []DocumentAuthor `json:"authors,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
	Links       *DocumentLinks   `json:"links,omitempty"`
}

// DocumentAuthor names a person or system responsible for the document.
type DocumentAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	Role  string `json:"role,omitempty"`
}

// DocumentLinks holds optional well-known URIs for this document.
type DocumentLinks struct {
	Repo      string `json:"repo,omitempty"`
	Docs      string `json:"docs,omitempty"`
	Dashboard string `json:"dashboard,omitempty"`
}

// FloorContent is an alias for floor.Content for call-site readability.
type FloorContent = floor.Content

// FloorBlockMeta is an alias for floor.BlockMeta.
type FloorBlockMeta = floor.BlockMeta

// MarshalBGSpecJSON encodes doc as BGSpec root JSON (bgspec, document, floors) using
// json tags aligned with bgspec.schema.json (only those keys; additionalProperties false).
// Nil nodes or edges slices on a floor become empty JSON arrays because the schema
// requires arrays there, not null.
//
// Schema-level constraints (for example at least two floors, bgspec ^[0-9]+\.[0-9]+$, ISO 8601
// timestamps, Floor 0 node/edge payloads) are not validated here.
func MarshalBGSpecJSON(doc BGSpecDocument) ([]byte, error) {
	out := doc
	if out.Floors == nil {
		out.Floors = []floor.Content{}
	} else {
		out.Floors = slices.Clone(doc.Floors)
	}
	for idx := range out.Floors {
		floorEntry := &out.Floors[idx]
		if floorEntry.Nodes == nil {
			floorEntry.Nodes = []node.Wrapper{}
		}
		if floorEntry.Edges == nil {
			floorEntry.Edges = []edge.Wrapper{}
		}
	}
	return json.MarshalIndent(out, "", "  ")
}
