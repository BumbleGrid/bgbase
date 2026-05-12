// Package node defines BGSpec node types and taxonomy (bgKind).
// Kubernetes JSON (`k8s`) and resolver-only hints use types from the root document package.
package node

// Wrapper is one Cytoscape-style node object { "data": { ... } }.
type Wrapper struct {
	Data Data `json:"data"`
}

// Data is the Floor 0 node data block inside data (BGSpec Floor 0 node schema).
type Data struct {
	// Graph identity
	ID      string  `json:"id"`
	Label   string  `json:"label"`
	Floor   int     `json:"floor"`
	BgKind  BgKind  `json:"bgKind"`
	Parent  *string `json:"parent,omitempty"`
	DrillTo *string `json:"drillTo,omitempty"`

	InfraProvider InfraProvider `json:"infraProvider"`

	Style *Style `json:"style,omitempty"`
	Meta  *Meta  `json:"meta,omitempty"`

	// TODO maybe we will need to add ProviderMetadata here
}

// BgKind is the BGSpec node taxonomy (Workload, Database, etc.).
type BgKind string

const (
	BgKindWorkload         BgKind = "Workload"
	BgKindDatabase         BgKind = "Database"
	BgKindCache            BgKind = "Cache"
	BgKindMessageBroker    BgKind = "MessageBroker"
	BgKindStorage          BgKind = "Storage"
	BgKindGateway          BgKind = "Gateway"
	BgKindLoadBalancer     BgKind = "LoadBalancer"
	BgKindExternalService  BgKind = "ExternalService"
	BgKindConfigSource     BgKind = "ConfigSource"
	BgKindSecretSource     BgKind = "SecretSource"
	BgKindServiceDiscovery BgKind = "ServiceDiscovery"
	BgKindNetworkPolicy    BgKind = "NetworkPolicy"
	BgKindJobRunner        BgKind = "JobRunner"
	BgKindNamespace        BgKind = "Namespace"
	BgKindCluster          BgKind = "Cluster"
)

// InfraProvider identifies the IaC source for a Floor 0 node (BGSpec infraProvider).
type InfraProvider string

const (
	InfraProviderKubernetes     InfraProvider = "kubernetes"
	InfraProviderTerraform      InfraProvider = "terraform"
	InfraProviderPulumi         InfraProvider = "pulumi"
	InfraProviderCloudFormation InfraProvider = "cloudformation"
	InfraProviderManual         InfraProvider = "manual"
)

// Style holds renderer hints for Floor 0 nodes (BGSpec style block).
type Style struct {
	Color       string    `json:"color,omitempty"`
	TextColor   string    `json:"textColor,omitempty"`
	BorderColor string    `json:"borderColor,omitempty"`
	BorderWidth float64   `json:"borderWidth,omitempty"`
	Shape       string    `json:"shape,omitempty"`
	Position    *Position `json:"position,omitempty"`
}

// Position is a fixed canvas position under style.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Meta holds operational metadata for a Floor 0 node (BGSpec meta block).
type Meta struct {
	Description      string   `json:"description,omitempty"`
	Team             string   `json:"team,omitempty"`
	Repo             string   `json:"repo,omitempty"`
	Docs             string   `json:"docs,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	SLA              string   `json:"sla,omitempty"`
	ExtractedAt      string   `json:"extractedAt,omitempty"`
	ExtractorVersion string   `json:"extractorVersion,omitempty"`
}
