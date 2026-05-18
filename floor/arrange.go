package floor

import (
	"math"
	"sort"

	"github.com/BumbleGrid/bgbase/node"
	"github.com/BumbleGrid/bgbase/stylemap"
)

const (
	arrangeCellWidth  = 178
	arrangeCellHeight = 104
	arrangeCellGap    = 48
)

var (
	arrangeCellPitchX = float64(arrangeCellWidth + arrangeCellGap)
	arrangeCellPitchY = float64(arrangeCellHeight + arrangeCellGap)
)

type nodeIndex struct {
	byID     map[string]node.Data
	children map[string][]string
}

func buildNodeIndex(nodes []node.Wrapper) nodeIndex {
	idx := nodeIndex{
		byID:     make(map[string]node.Data, len(nodes)),
		children: make(map[string][]string),
	}
	for _, wrapped := range nodes {
		data := wrapped.Data
		idx.byID[data.ID] = data
		if data.Parent == nil {
			continue
		}
		parentID := *data.Parent
		idx.children[parentID] = append(idx.children[parentID], data.ID)
	}
	for parentID := range idx.children {
		sort.Strings(idx.children[parentID])
	}
	return idx
}

func findClusterRoots(idx nodeIndex) []string {
	seen := make(map[string]struct{})
	var roots []string

	for id, data := range idx.byID {
		if data.BgKind == node.BgKindCluster {
			roots = append(roots, id)
			seen[id] = struct{}{}
		}
	}

	referencedParents := make(map[string]struct{})
	for _, data := range idx.byID {
		if data.Parent == nil {
			continue
		}
		referencedParents[*data.Parent] = struct{}{}
	}

	for parentID := range referencedParents {
		if _, ok := seen[parentID]; ok {
			continue
		}
		data, exists := idx.byID[parentID]
		if exists {
			if data.BgKind == node.BgKindCluster {
				roots = append(roots, parentID)
				seen[parentID] = struct{}{}
			}
			continue
		}
		roots = append(roots, parentID)
		seen[parentID] = struct{}{}
	}

	sort.Strings(roots)
	return roots
}

func gridDimensions(count int) (cols, rows int) {
	if count <= 0 {
		return 0, 0
	}
	cols = int(math.Ceil(math.Sqrt(float64(count))))
	rows = int(math.Ceil(float64(count) / float64(cols)))
	return cols, rows
}

func childLayoutWeight(idx nodeIndex, childID string, data node.Data) int {
	if data.BgKind != node.BgKindNamespace {
		return 1
	}
	return len(idx.children[childID]) + 1
}

type layoutBounds struct {
	maxX float64
	maxY float64
}

func (bounds *layoutBounds) include(x, y float64) {
	if x > bounds.maxX {
		bounds.maxX = x
	}
	if y > bounds.maxY {
		bounds.maxY = y
	}
}

func layoutContainer(
	idx nodeIndex,
	containerID string,
	originX, originY float64,
	positions map[string]stylemap.CanvasPosition,
) layoutBounds {
	childIDs := idx.children[containerID]
	if len(childIDs) == 0 {
		return layoutBounds{}
	}

	type layoutUnit struct {
		id          string
		weight      int
		isNamespace bool
	}

	units := make([]layoutUnit, 0, len(childIDs))
	totalWeight := 0
	for _, childID := range childIDs {
		data := idx.byID[childID]
		weight := childLayoutWeight(idx, childID, data)
		totalWeight += weight
		units = append(units, layoutUnit{
			id:          childID,
			weight:      weight,
			isNamespace: data.BgKind == node.BgKindNamespace,
		})
	}

	gridCols, _ := gridDimensions(totalWeight)
	var bounds layoutBounds
	unitCol := 0
	unitRow := 0
	rowSpan := 0

	for _, unit := range units {
		blockCols, blockRows := gridDimensions(unit.weight)
		posX := originX + float64(unitCol)*arrangeCellPitchX
		posY := originY + float64(unitRow)*arrangeCellPitchY
		positions[unit.id] = stylemap.CanvasPosition{X: posX, Y: posY}
		bounds.include(posX, posY)

		if unit.isNamespace {
			innerOriginX := posX + arrangeCellPitchX
			innerOriginY := posY + arrangeCellPitchY
			inner := layoutContainer(idx, unit.id, innerOriginX, innerOriginY, positions)
			bounds.include(inner.maxX, inner.maxY)
		}

		unitCol += blockCols
		if blockRows > rowSpan {
			rowSpan = blockRows
		}
		if unitCol >= gridCols {
			unitCol = 0
			unitRow += rowSpan
			rowSpan = 0
		}
	}

	return bounds
}

// AutoArrangeStyleMap assigns canvas positions for nodes on a floor in styleMap.byId.
// Cluster roots (explicit Cluster nodes or implicit parent IDs such as cluster/main) are
// laid out from (0,0). Direct children are placed in a near-square grid; a Namespace child
// reserves len(children)+1 cells at the parent level and recurses with the same rules inside.
func AutoArrangeStyleMap(content Content) stylemap.StyleMap {
	idx := buildNodeIndex(content.Nodes)
	clusterRoots := findClusterRoots(idx)

	positions := make(map[string]stylemap.CanvasPosition)
	var clusterOffsetX float64

	for _, clusterID := range clusterRoots {
		clusterStartX := clusterOffsetX
		bounds := layoutContainer(idx, clusterID, clusterStartX, 0, positions)
		span := bounds.maxX - clusterStartX
		if span < 0 {
			span = 0
		}
		clusterOffsetX = clusterStartX + span + arrangeCellPitchX
	}

	byID := make(map[string]stylemap.StyleRules, len(positions))
	for nodeID, pos := range positions {
		position := pos
		byID[nodeID] = stylemap.StyleRules{
			Node: &stylemap.NodeStyleRules{
				Position: &position,
			},
		}
	}

	return stylemap.StyleMap{ByID: byID}
}
