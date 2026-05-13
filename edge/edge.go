// Package edge defines BGSpec edge types and relation taxonomy (bgRelation).
package edge

// Wrapper is one Cytoscape-style edge object { "data": { ... } }.
type Wrapper struct {
	Data Data `json:"data"`
}

// Data is the Floor 0 edge data block inside data.
type Data struct {
	ID               string           `json:"id"`
	Source           string           `json:"source"`
	Target           string           `json:"target"`
	Floor            int              `json:"floor"`
	BGRelation       BgRelation       `json:"bgRelation"`
	ExtractionSource ExtractionSource `json:"extractionSource"`
	Inferred         bool             `json:"inferred"`
	Label            *string          `json:"label,omitempty"`
	Meta             Meta             `json:"meta"`
}

// BgRelation is the BGSpec edge relation kind.
type BgRelation string

const (
	BgRelationRoutes      BgRelation = "Routes"
	BgRelationExposes     BgRelation = "Exposes"
	BgRelationMounts      BgRelation = "Mounts"
	BgRelationScheduledBy BgRelation = "ScheduledBy"
	BgRelationCalls       BgRelation = "Calls"
)

// ExtractionSource declares the origin of a Floor 0 edge (BGSpec extractionSource).
type ExtractionSource string

const (
	ExtractionSourceK8sManifest   ExtractionSource = "k8s-manifest"
	ExtractionSourceIstioManifest ExtractionSource = "istio-manifest"
	ExtractionSourceManual        ExtractionSource = "manual"
)

// Meta holds extractor timestamps and version for an edge.
type Meta struct {
	Description      string `json:"description,omitempty"`
	ExtractedAt      string `json:"extractedAt,omitempty"`
	ExtractorVersion string `json:"extractorVersion,omitempty"`
}
