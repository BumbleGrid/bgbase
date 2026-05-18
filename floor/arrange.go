package floor

import (
	"math"
	"sort"

	"github.com/BumbleGrid/bgbase/node"
	"github.com/BumbleGrid/bgbase/stylemap"
)

const (
	arrangeCellWidth     = 160
	arrangeCellHeight    = 104
	arrangeCellGap       = 48
	arrangeCompoundScale = 1
)

var (
	arrangeCellPitchX = float64(arrangeCellWidth + arrangeCellGap)
	arrangeCellPitchY = float64(arrangeCellHeight + arrangeCellGap)
)

type nodeIndex struct {
	byID     map[string]node.Data
	children map[string][]string
}

type compoundSize struct {
	width  float64
	height float64
}

type layoutUnit struct {
	id          string
	weight      int
	isNamespace bool
}

type arrangeState struct {
	idx       nodeIndex
	positions map[string]stylemap.CanvasPosition
	sizes     map[string]compoundSize
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

func compoundNodeSize(childCount int) (width, height float64) {
	if childCount <= 0 {
		return 0, 0
	}
	scale := float64(childCount) * arrangeCompoundScale
	return float64(arrangeCellWidth) * scale, float64(arrangeCellHeight) * scale
}

func isCompoundBgKind(kind node.BgKind) bool {
	return kind == node.BgKindCluster || kind == node.BgKindNamespace
}

func partitionClusterChildren(idx nodeIndex, clusterID string) (namespaces, others []string) {
	for _, childID := range idx.children[clusterID] {
		if idx.byID[childID].BgKind == node.BgKindNamespace {
			namespaces = append(namespaces, childID)
		} else {
			others = append(others, childID)
		}
	}
	return namespaces, others
}

func buildLayoutUnits(idx nodeIndex, childIDs []string) ([]layoutUnit, int) {
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
	return units, totalWeight
}

func (state *arrangeState) layoutLeafGrid(originX, originY float64, childIDs []string) {
	if len(childIDs) == 0 {
		return
	}
	gridCols, _ := gridDimensions(len(childIDs))
	unitCol := 0
	unitRow := 0
	rowSpan := 0

	for _, childID := range childIDs {
		posX := originX + float64(unitCol)*arrangeCellPitchX
		posY := originY + float64(unitRow)*arrangeCellPitchY
		state.positions[childID] = stylemap.CanvasPosition{X: posX, Y: posY}

		unitCol++
		if unitCol >= gridCols {
			unitCol = 0
			unitRow += rowSpan
			if rowSpan == 0 {
				rowSpan = 1
			}
			rowSpan = 0
		}
	}
}

func (state *arrangeState) prepareNamespace(nsID string) {
	childIDs := state.idx.children[nsID]
	width, height := compoundNodeSize(len(childIDs))
	state.sizes[nsID] = compoundSize{width: width, height: height}
	state.layoutLeafGrid(arrangeCellPitchX, arrangeCellPitchY, childIDs)
}

func (state *arrangeState) unitFootprint(unit layoutUnit) compoundSize {
	if unit.isNamespace {
		size := state.sizes[unit.id]
		return compoundSize{
			width:  size.width + arrangeCellGap,
			height: size.height + arrangeCellGap,
		}
	}
	return compoundSize{width: arrangeCellPitchX, height: arrangeCellPitchY}
}

func (state *arrangeState) packUnits(originX, originY float64, units []layoutUnit) compoundSize {
	if len(units) == 0 {
		return compoundSize{}
	}

	gridCols, _ := gridDimensions(len(units))
	unitCol := 0
	rowMaxHeight := 0.0
	pixelX := originX
	pixelY := originY
	maxX := originX
	maxY := originY

	for _, unit := range units {
		footprint := state.unitFootprint(unit)
		if unitCol > 0 && unitCol >= gridCols {
			pixelX = originX
			pixelY += rowMaxHeight
			unitCol = 0
			rowMaxHeight = 0
		}

		state.positions[unit.id] = stylemap.CanvasPosition{X: pixelX, Y: pixelY}

		endX := pixelX + footprint.width
		endY := pixelY + footprint.height
		if endX > maxX {
			maxX = endX
		}
		if endY > maxY {
			maxY = endY
		}

		pixelX += footprint.width
		if footprint.height > rowMaxHeight {
			rowMaxHeight = footprint.height
		}
		unitCol++
	}

	return compoundSize{
		width:  maxX - originX,
		height: maxY - originY,
	}
}

func (state *arrangeState) layoutCluster(clusterID string, clusterStartX float64) compoundSize {
	namespaces, others := partitionClusterChildren(state.idx, clusterID)
	childIDs := make([]string, 0, len(namespaces)+len(others))
	childIDs = append(childIDs, namespaces...)
	childIDs = append(childIDs, others...)

	for _, nsID := range namespaces {
		state.prepareNamespace(nsID)
	}

	units, _ := buildLayoutUnits(state.idx, childIDs)
	originX := clusterStartX
	if _, exists := state.idx.byID[clusterID]; exists {
		if clusterStartX > 0 {
			state.positions[clusterID] = stylemap.CanvasPosition{X: clusterStartX, Y: 0}
		}
		originX = 0
	}
	_ = state.packUnits(originX, 0, units)

	childCount := len(childIDs)
	width, height := compoundNodeSize(childCount)
	clusterSize := compoundSize{width: width, height: height}
	if _, exists := state.idx.byID[clusterID]; exists {
		state.sizes[clusterID] = clusterSize
	}
	return clusterSize
}

func (state *arrangeState) toStyleMap() stylemap.StyleMap {
	byID := make(map[string]stylemap.StyleRules, len(state.positions)+len(state.sizes))
	for nodeID, pos := range state.positions {
		position := pos
		byID[nodeID] = stylemap.StyleRules{
			Node: &stylemap.NodeStyleRules{
				Position: &position,
			},
		}
	}
	for nodeID, size := range state.sizes {
		rules := byID[nodeID]
		if rules.Node == nil {
			rules.Node = &stylemap.NodeStyleRules{}
		}
		rules.Node.Width = size.width
		rules.Node.Height = size.height
		byID[nodeID] = rules
	}
	return stylemap.StyleMap{ByID: byID}
}

// AutoArrangeStyleMap assigns canvas positions and compound sizes for nodes on a floor
// in styleMap.byId. Each cluster is processed in order: namespace interiors and sizes
// first, then cluster child positions using those footprints, then cluster size.
func AutoArrangeStyleMap(content Content) stylemap.StyleMap {
	idx := buildNodeIndex(content.Nodes)
	state := &arrangeState{
		idx:       idx,
		positions: make(map[string]stylemap.CanvasPosition),
		sizes:     make(map[string]compoundSize),
	}

	var clusterOffsetX float64
	for _, clusterID := range findClusterRoots(idx) {
		clusterSize := state.layoutCluster(clusterID, clusterOffsetX)
		clusterOffsetX += clusterSize.width + arrangeCellGap
	}

	return state.toStyleMap()
}
