// Package validate defines validation for BGSpec documents (shared + per-floor rules).
package validate

import "github.com/BumbleGrid/bgbase/graph"

// Validator runs validation rules against a parsed document.
type Validator interface {
	Validate(doc *graph.BGSpecDocument) error
}
