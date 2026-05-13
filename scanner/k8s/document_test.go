package k8s

import (
	"testing"

	"github.com/BumbleGrid/bgbase/floor"
	"github.com/BumbleGrid/bgbase/node"
)

func TestNewBGSpecDocument_floorsAndPlaceholders(t *testing.T) {
	f0 := floor.Content{
		Floor:       0,
		Label:       "Live cluster",
		Description: "Floor 0 from scan",
		Nodes:       []node.Wrapper{{Data: node.Data{ID: "n1", Label: "x", Floor: 0}}},
		Edges:       nil,
	}
	doc := NewBGSpecDocument(f0)
	if doc.BGSpec != defaultBGSpecVersion {
		t.Fatalf("BGSpec = %q", doc.BGSpec)
	}
	if doc.Document.Title == "" || doc.Document.UpdatedAt == "" {
		t.Fatalf("document defaults: %#v", doc.Document)
	}
	if len(doc.Floors) != 4 {
		t.Fatalf("len(Floors) = %d, want 4", len(doc.Floors))
	}
	if doc.Floors[0].Floor != 0 || len(doc.Floors[0].Nodes) != 1 {
		t.Fatalf("floor0: %#v", doc.Floors[0])
	}
	for idx := 1; idx <= 3; idx++ {
		fl := doc.Floors[idx]
		if fl.Floor != idx {
			t.Fatalf("floors[%d].Floor = %d", idx, fl.Floor)
		}
		if len(fl.Nodes) != 0 || len(fl.Edges) != 0 {
			t.Fatalf("floors[%d] expected empty nodes/edges: nodes=%d edges=%d", idx, len(fl.Nodes), len(fl.Edges))
		}
		if fl.Meta == nil {
			t.Fatalf("floors[%d].Meta is nil", idx)
		}
	}
}
