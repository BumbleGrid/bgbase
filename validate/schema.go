package validate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/BumbleGrid/bgbase/graph"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

const (
	rootSchemaID       = "https://bumblegrid.tech/schemas/bgspec/v0.1/bgspec.schema.json"
	floor0NodeSchemaID = "https://bumblegrid.tech/schemas/bgspec/v0.1/floor-0-node.json"
	floor0EdgeSchemaID = "https://bumblegrid.tech/schemas/bgspec/v0.1/floor-0-edge.json"
	floorUpperNodeID   = "https://bumblegrid.tech/schemas/bgspec/v0.1/floor-upper-node.json"
	floorUpperEdgeID   = "https://bumblegrid.tech/schemas/bgspec/v0.1/floor-upper-edge.json"
	cytoscapeVisualID  = "https://bumblegrid.tech/schemas/bgspec/v0.1/cytoscape-visual.json"

	bgspecSchemaFilename     = "bgspec.schema.json"
	floor0NodeSchemaFilename = "floor-0-node.json"
	floor0EdgeSchemaFilename = "floor-0-edge.json"
	floorUpperNodeFilename   = "floor-upper-node.json"
	floorUpperEdgeFilename   = "floor-upper-edge.json"
	cytoscapeVisualFilename  = "cytoscape-visual.json"
)

// SchemaValidator is the JSON-Schema-backed implementation of Validator.
// Construct via NewSchemaValidator; the zero value is not usable.
//
// Safe for concurrent use once constructed.
type SchemaValidator struct {
	compiled *jsonschema.Schema
}

// NewSchemaValidator compiles the embedded BGSpec JSON-Schema and returns a ready validator.
// The compiled schema is held internally; callers should construct one per process at startup
// and reuse it across requests.
func NewSchemaValidator() (*SchemaValidator, error) {
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft7
	if err := addEmbeddedResource(compiler, floor0NodeSchemaID, floor0NodeSchemaFilename); err != nil {
		return nil, err
	}
	if err := addEmbeddedResource(compiler, floor0EdgeSchemaID, floor0EdgeSchemaFilename); err != nil {
		return nil, err
	}
	if err := addEmbeddedResource(compiler, floorUpperNodeID, floorUpperNodeFilename); err != nil {
		return nil, err
	}
	if err := addEmbeddedResource(compiler, floorUpperEdgeID, floorUpperEdgeFilename); err != nil {
		return nil, err
	}
	if err := addEmbeddedResource(compiler, cytoscapeVisualID, cytoscapeVisualFilename); err != nil {
		return nil, err
	}
	if err := addEmbeddedResource(compiler, rootSchemaID, bgspecSchemaFilename); err != nil {
		return nil, err
	}
	compiled, err := compiler.Compile(rootSchemaID)
	if err != nil {
		return nil, fmt.Errorf("bgbase/validate: compile bgspec schema: %w", err)
	}
	return &SchemaValidator{compiled: compiled}, nil
}

func addEmbeddedResource(compiler *jsonschema.Compiler, id, filename string) error {
	body, err := schemaFS.ReadFile("testdata/specs/" + filename)
	if err != nil {
		return fmt.Errorf("bgbase/validate: load embedded schema %q: %w", filename, err)
	}
	if err := compiler.AddResource(id, bytes.NewReader(body)); err != nil {
		return fmt.Errorf("bgbase/validate: register schema %q: %w", id, err)
	}
	return nil
}

// Validate runs doc through the BGSpec JSON-Schema. Returns nil on success; on failure
// returns a *MultiError holding every violation. Returns an error if doc is nil.
//
// Validation round-trips through graph.MarshalBGSpecJSON so nil slices become empty
// arrays. Unknown JSON fields present only in raw wire bytes are dropped if the caller
// unmarshals into graph.BGSpecDocument first; use ValidateJSON for authoritative
// wire-format checks.
func (validator *SchemaValidator) Validate(doc *graph.BGSpecDocument) error {
	if doc == nil {
		return errors.New("bgbase/validate: cannot validate nil document")
	}
	body, err := graph.MarshalBGSpecJSON(*doc)
	if err != nil {
		return fmt.Errorf("bgbase/validate: marshal doc: %w", err)
	}
	return validator.ValidateJSON(body)
}

// ValidateJSON validates raw BGSpec JSON bytes against the embedded schema.
func (validator *SchemaValidator) ValidateJSON(body []byte) error {
	var raw any
	if err := decodeJSONNumber(bytes.NewReader(body), &raw); err != nil {
		return fmt.Errorf("bgbase/validate: decode json: %w", err)
	}
	if err := validator.compiled.Validate(raw); err != nil {
		var validationErr *jsonschema.ValidationError
		if errors.As(err, &validationErr) {
			return collectViolations(validationErr)
		}
		return fmt.Errorf("bgbase/validate: %w", err)
	}
	return nil
}

func decodeJSONNumber(reader io.Reader, out *any) error {
	dec := json.NewDecoder(reader)
	dec.UseNumber()
	return dec.Decode(out)
}
