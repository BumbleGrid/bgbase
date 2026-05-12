// Package parser defines decoding of BGSpec documents from on-disk or wire formats.
package parser

import "github.com/BumbleGrid/bgbase/graph"

// Parser decodes a BGSpec payload into a graph.BGSpecDocument.
type Parser interface {
	Parse(src []byte) (*graph.BGSpecDocument, error)
}
