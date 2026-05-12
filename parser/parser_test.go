package parser

import (
	"encoding/json"
	"testing"

	"github.com/BumbleGrid/bgbase"
	"github.com/BumbleGrid/bgbase/edge"
	"github.com/BumbleGrid/bgbase/graph"
	"github.com/BumbleGrid/bgbase/node"
)

func TestJSON_ParseMinimal(t *testing.T) {
	src := `{
		"bgspec": "0.1",
		"document": {"title": "t", "company": "c", "updatedAt": "2020-01-01T00:00:00Z"},
		"floors": [{
			"floor": 0,
			"label": "Infra",
			"description": "d",
			"nodes": [{"data": {
				"id": "a",
				"label": "A",
				"floor": 0,
				"bgKind": "Workload",
				"infraProvider": "kubernetes",
				"k8s": {"kind": "Deployment", "name": "a", "namespace": "ns"}
			}}],
			"edges": [{"data": {
				"id": "e1",
				"source": "a",
				"target": "b",
				"floor": 0,
				"bgRelation": "Routes",
				"extractionSource": "k8s-manifest",
				"inferred": false,
				"style": {},
				"meta": {"extractedAt": "2020-01-01T00:00:00Z", "extractorVersion": "0.1.0"}
			}}]
		}, {
			"floor": 1,
			"label": "Features",
			"description": "d2",
			"nodes": [],
			"edges": []
		}]
	}`
	var p JSON
	doc, err := p.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if doc.BGSpec != "0.1" || doc.Document.Title != "t" {
		t.Fatalf("document: %+v", doc.Document)
	}
	if len(doc.Floors) != 2 {
		t.Fatalf("floors: %d", len(doc.Floors))
	}
	if doc.Floors[0].Nodes[0].Data.BgKind != node.BgKindWorkload {
		t.Fatalf("node bgKind: %q", doc.Floors[0].Nodes[0].Data.BgKind)
	}
	if doc.Floors[0].Nodes[0].Data.InfraProvider != node.InfraProviderKubernetes {
		t.Fatalf("infraProvider: %q", doc.Floors[0].Nodes[0].Data.InfraProvider)
	}
	if doc.Floors[0].Edges[0].Data.BGRelation != edge.BgRelationRoutes {
		t.Fatalf("edge relation: %q", doc.Floors[0].Edges[0].Data.BGRelation)
	}
}

func TestJSON_ParseInvalid(t *testing.T) {
	var p JSON
	_, err := p.Parse([]byte(`{`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestJSON_RoundTripShape(t *testing.T) {
	ns := "ns"
	doc := &graph.BGSpecDocument{
		BGSpec: "0.1",
		Document: graph.DocumentMeta{
			Title: "t", Company: "c", UpdatedAt: "2020-01-01T00:00:00Z",
		},
		Floors: []graph.FloorContent{
			{
				Floor: 0, Label: "L0", Description: "d",
				Nodes: []node.Wrapper{{
					Data: node.Data{
						ID: "1", Label: "n", Floor: 0, BgKind: node.BgKindCluster,
						InfraProvider: node.InfraProviderKubernetes,
						K8s: &document.K8sNode{
							Kind: "Namespace", Name: "demo", Namespace: &ns,
						},
					},
				}},
				Edges: []edge.Wrapper{{
					Data: edge.Data{
						ID: "e", Source: "1", Target: "2", Floor: 0, BGRelation: edge.BgRelationCalls,
						ExtractionSource: edge.ExtractionSourceK8sManifest,
						Style:            edge.Style{},
						Meta: edge.Meta{
							ExtractedAt: "2020-01-01T00:00:00Z", ExtractorVersion: "0.1.0",
						},
					},
				}},
			},
			{Floor: 1, Label: "L1", Description: "d2", Nodes: nil, Edges: nil},
		},
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var p JSON
	out, err := p.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if out.Document.Title != doc.Document.Title {
		t.Fatalf("round trip: %+v", out.Document)
	}
}
