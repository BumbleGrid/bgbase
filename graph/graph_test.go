package graph

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/BumbleGrid/bgbase/floor"
)

func TestBGSpecDocumentZero(t *testing.T) {
	var doc BGSpecDocument
	if doc.BGSpec != "" {
		t.Fatal("expected zero value")
	}
}

func TestMarshalBGSpecJSON_nilSlicesBecomeEmptyArrays(t *testing.T) {
	doc := BGSpecDocument{
		BGSpec: "0.1",
		Document: DocumentMeta{
			Title:     "Example",
			Company:   "Acme",
			UpdatedAt: "2026-05-12T12:00:00Z",
		},
		Floors: []floor.Content{
			{Floor: 0, Label: "Infrastructure", Description: "floor 0"},
			{Floor: 1, Label: "Features", Description: "floor 1"},
		},
	}
	payload, err := MarshalBGSpecJSON(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(payload), `"nodes":null`) || strings.Contains(string(payload), `"edges":null`) {
		t.Fatalf("expected empty arrays, got: %s", payload)
	}
	var round BGSpecDocument
	if err := json.Unmarshal(payload, &round); err != nil {
		t.Fatalf("unmarshal round-trip: %v", err)
	}
	if len(round.Floors) != 2 {
		t.Fatalf("floors len: got %d", len(round.Floors))
	}
	if round.Floors[0].Nodes == nil || round.Floors[0].Edges == nil {
		t.Fatal("expected non-nil slices after unmarshal")
	}
}

func TestMarshalFloorContentJSON_nilSlicesBecomeEmptyArrays(t *testing.T) {
	fl := floor.Content{
		Floor:       0,
		Label:       "Infrastructure",
		Description: "floor 0",
	}
	payload, err := MarshalFloorContentJSON(fl)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(payload), `"nodes":null`) || strings.Contains(string(payload), `"edges":null`) {
		t.Fatalf("expected empty arrays, got: %s", payload)
	}
}
