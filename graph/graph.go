// Package graph defines the top-level BGSpec document shape (root JSON object).
package graph

import "github.com/BumbleGrid/bgbase/floor"

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
