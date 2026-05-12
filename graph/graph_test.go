package graph

import "testing"

func TestBGSpecDocumentZero(t *testing.T) {
	var doc BGSpecDocument
	if doc.BGSpec != "" {
		t.Fatal("expected zero value")
	}
}
