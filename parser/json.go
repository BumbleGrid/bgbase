package parser

import (
	"encoding/json"

	"github.com/BumbleGrid/bgbase/graph"
)

// JSON implements Parser using encoding/json.
type JSON struct{}

// Parse unmarshals src into a BGSpecDocument.
func (JSON) Parse(src []byte) (*graph.BGSpecDocument, error) {
	var doc graph.BGSpecDocument
	if err := json.Unmarshal(src, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}
