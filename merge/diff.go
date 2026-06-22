package merge

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/BumbleGrid/bgbase/edge"
	"github.com/BumbleGrid/bgbase/floor"
	"github.com/BumbleGrid/bgbase/node"
)

// CompareFloors returns a structured diff between two Floor 0 instances. Identity
// is by node.data.id and edge.data.id; "changed" means the JSON-serialised payload
// differs while the id matches.
func CompareFloors(previous, next floor.Content) FloorDiff {
	prevNodes := indexNodes(previous.Nodes)
	nextNodes := indexNodes(next.Nodes)
	prevEdges := indexEdges(previous.Edges)
	nextEdges := indexEdges(next.Edges)

	return FloorDiff{
		NodesAdded:   sortedKeysOnlyInB(prevNodes, nextNodes),
		NodesRemoved: sortedKeysOnlyInA(prevNodes, nextNodes),
		NodesChanged: sortedKeysWithDifferentPayload(prevNodes, nextNodes, equalNodes),
		EdgesAdded:   sortedKeysOnlyInB(prevEdges, nextEdges),
		EdgesRemoved: sortedKeysOnlyInA(prevEdges, nextEdges),
		EdgesChanged: sortedKeysWithDifferentPayload(prevEdges, nextEdges, equalEdges),
	}
}

func compareFloors(previous, next floor.Content) FloorDiff {
	return CompareFloors(previous, next)
}

func indexNodes(in []node.Wrapper) map[string]node.Wrapper {
	out := make(map[string]node.Wrapper, len(in))
	for _, wrapper := range in {
		out[wrapper.Data.ID] = wrapper
	}
	return out
}

func indexEdges(in []edge.Wrapper) map[string]edge.Wrapper {
	out := make(map[string]edge.Wrapper, len(in))
	for _, wrapper := range in {
		out[wrapper.Data.ID] = wrapper
	}
	return out
}

func sortedKeysOnlyInB[T any](a, b map[string]T) []string {
	out := make([]string, 0)
	for key := range b {
		if _, ok := a[key]; !ok {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func sortedKeysOnlyInA[T any](a, b map[string]T) []string {
	return sortedKeysOnlyInB(b, a)
}

func sortedKeysWithDifferentPayload[T any](a, b map[string]T, eq func(T, T) bool) []string {
	out := make([]string, 0)
	for key, left := range a {
		right, ok := b[key]
		if !ok {
			continue
		}
		if !eq(left, right) {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func equalNodes(a, b node.Wrapper) bool { return equalJSON(a, b) }
func equalEdges(a, b edge.Wrapper) bool { return equalJSON(a, b) }

func equalJSON[T any](a, b T) bool {
	left, leftErr := json.Marshal(a)
	right, rightErr := json.Marshal(b)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return bytes.Equal(left, right)
}
