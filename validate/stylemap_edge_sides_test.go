package validate

import (
	"testing"
)

func TestSchemaValidator_styleMapEdgeSourceAndTargetSide(t *testing.T) {
	validator, err := NewSchemaValidator()
	if err != nil {
		t.Fatalf("NewSchemaValidator: %v", err)
	}

	body := []byte(`{
		"bgspec":"0.1",
		"document":{"title":"t","company":"c","updatedAt":"2026-01-01T00:00:00Z"},
		"styleMap":{
			"byId":{
				"edge-ab":{
					"edge":{"sourceSide":"left","targetSide":"right","color":"#112233"}
				}
			}
		},
		"floors":[
			{"floor":0,"label":"F0","description":"","nodes":[],"edges":[]},
			{"floor":1,"label":"F1","description":"","nodes":[],"edges":[]}
		]
	}`)

	if err := validator.ValidateJSON(body); err != nil {
		t.Fatalf("ValidateJSON: %v", err)
	}
}
