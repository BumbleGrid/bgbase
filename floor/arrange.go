package floor

import (
	"math"

	"github.com/BumbleGrid/bgbase/node"
	"github.com/BumbleGrid/bgbase/stylemap"
)

const (
	arrangeDefaultNodeWidth  = 178
	arrangeDefaultNodeHeight = 104
	arrangeDefaultOffsetX    = 0
	arrangeChildGap          = 48
	arrangeCompoundScale     = 1

	arrangeCellWidth  = arrangeDefaultNodeWidth
	arrangeCellHeight = arrangeDefaultNodeHeight
	arrangeCellGap    = arrangeChildGap

	arrangeCompoundTopInset = 28
)

var (
	arrangeCellPitchX = float64(arrangeCellWidth + arrangeCellGap)
	arrangeCellPitchY = float64(arrangeCellHeight + arrangeCellGap)
)

type NodeTree struct {
	Node     node.Data
	Children []NodeTree
}

type compoundSize struct {
	width  float64
	height float64
}

type arrangeState struct {
	sizes map[string]compoundSize
}

func buildK8sNodeTrees(content Content) []NodeTree {
	clusterByID := make(map[string]node.Data)
	namespaceByID := make(map[string]node.Data)

	for _, wrapper := range content.Nodes {
		data := wrapper.Data
		switch data.BgKind {
		case node.BgKindCluster:
			clusterByID[data.ID] = data
		case node.BgKindNamespace:
			namespaceByID[data.ID] = data
		}
	}

	childrenByParent := make(map[string][]node.Data)
	var roots []node.Data

	for _, wrapper := range content.Nodes {
		data := wrapper.Data
		parent := k8sTreeParent(data, clusterByID, namespaceByID)
		if parent == nil {
			roots = append(roots, data)
			continue
		}
		childrenByParent[*parent] = append(childrenByParent[*parent], data)
	}

	trees := make([]NodeTree, 0, len(roots))
	for _, root := range roots {
		trees = append(trees, buildNodeTree(root, childrenByParent))
	}
	return trees
}

func k8sTreeParent(data node.Data, clusterByID, namespaceByID map[string]node.Data) *string {
	switch data.BgKind {
	case node.BgKindCluster:
		return nil
	case node.BgKindNamespace:
		if data.Parent == nil {
			return nil
		}
		if _, ok := clusterByID[*data.Parent]; ok {
			return data.Parent
		}
		return nil
	default:
		if data.Parent == nil {
			return nil
		}
		parentID := *data.Parent
		if _, ok := namespaceByID[parentID]; ok {
			return &parentID
		}
		if _, ok := clusterByID[parentID]; ok {
			return &parentID
		}
		return nil
	}
}

func buildNodeTree(data node.Data, childrenByParent map[string][]node.Data) NodeTree {
	tree := NodeTree{Node: data}
	for _, child := range childrenByParent[data.ID] {
		tree.Children = append(tree.Children, buildNodeTree(child, childrenByParent))
	}
	return tree
}

func compoundChildOrigin(parentKind node.BgKind, origin stylemap.CanvasPosition) stylemap.CanvasPosition {
	if parentKind == node.BgKindNamespace {
		return stylemap.CanvasPosition{
			X: origin.X + arrangeCellPitchX,
			Y: origin.Y + arrangeCompoundTopInset,
		}
	}
	return origin
}

func setNodePosition(byID map[string]stylemap.StyleRules, nodeID string, position stylemap.CanvasPosition) {
	rules := byID[nodeID]
	if rules.Node == nil {
		rules.Node = &stylemap.NodeStyleRules{}
	}
	pos := position
	rules.Node.Position = &pos
	byID[nodeID] = rules
}

func arrangeChildrenRow(tree NodeTree, childOrigin stylemap.CanvasPosition, byID map[string]stylemap.StyleRules) compoundSize {
	placeX := childOrigin.X
	placeY := childOrigin.Y
	content := compoundSize{}

	for idx, child := range tree.Children {
		childRules := arrangeNodeTree(child, byID)
		setNodePosition(byID, child.Node.ID, stylemap.CanvasPosition{X: placeX, Y: placeY})

		if idx == 0 {
			content.width = childRules.Width
			content.height = childRules.Height
		} else {
			content.width += arrangeChildGap + childRules.Width
			if childRules.Height > content.height {
				content.height = childRules.Height
			}
		}
		placeX += childRules.Width + arrangeChildGap
	}
	return content
}

func arrangeChildrenGrid(tree NodeTree, childOrigin stylemap.CanvasPosition, byID map[string]stylemap.StyleRules) compoundSize {
	gridCols, _ := gridDimensions(len(tree.Children))
	rowStartX := childOrigin.X
	placeX := rowStartX
	placeY := childOrigin.Y
	unitCol := 0
	rowMaxHeight := 0.0
	var rowWidth float64
	content := compoundSize{}
	firstRow := true

	for idx, child := range tree.Children {
		childRules := arrangeNodeTree(child, byID)
		setNodePosition(byID, child.Node.ID, stylemap.CanvasPosition{X: placeX, Y: placeY})

		if unitCol == 0 {
			rowWidth = childRules.Width
		} else {
			rowWidth += arrangeChildGap + childRules.Width
		}
		if childRules.Height > rowMaxHeight {
			rowMaxHeight = childRules.Height
		}

		unitCol++
		lastChild := idx == len(tree.Children)-1
		if unitCol >= gridCols || lastChild {
			if firstRow {
				content.width = rowWidth
				content.height = rowMaxHeight
				firstRow = false
			} else {
				if rowWidth > content.width {
					content.width = rowWidth
				}
				content.height += arrangeChildGap + rowMaxHeight
			}
			unitCol = 0
			if !lastChild {
				placeY += rowMaxHeight + arrangeChildGap
				placeX = rowStartX
				rowMaxHeight = 0
			}
		} else {
			placeX += childRules.Width + arrangeChildGap
		}
	}
	return content
}

func measurePlacedChildrenBounds(childOrigin stylemap.CanvasPosition, children []NodeTree, byID map[string]stylemap.StyleRules) compoundSize {
	bounds := compoundSize{}
	for _, child := range children {
		rules := byID[child.Node.ID].Node
		if rules == nil || rules.Position == nil {
			continue
		}
		endX := rules.Position.X + rules.Width
		endY := rules.Position.Y + rules.Height
		contentW := endX - childOrigin.X
		contentH := endY - childOrigin.Y
		if contentW > bounds.width {
			bounds.width = contentW
		}
		if contentH > bounds.height {
			bounds.height = contentH
		}
	}
	return bounds
}

func mergeCompoundContent(gridContent, measured compoundSize) compoundSize {
	out := gridContent
	if measured.width > out.width {
		out.width = measured.width
	}
	if measured.height > out.height {
		out.height = measured.height
	}
	return out
}

func subtreeCanvasBottom(tree NodeTree, canvasX, canvasY float64, byID map[string]stylemap.StyleRules) float64 {
	rules := byID[tree.Node.ID].Node
	nodeX, nodeY := canvasX, canvasY
	if rules != nil && rules.Position != nil {
		nodeX += rules.Position.X
		nodeY += rules.Position.Y
	}
	bottom := nodeY
	if rules != nil {
		bottom += rules.Height
	}
	for _, child := range tree.Children {
		childBottom := subtreeCanvasBottom(child, nodeX, nodeY, byID)
		if childBottom > bottom {
			bottom = childBottom
		}
	}
	return bottom
}

func arrangeNodeTree(tree NodeTree, byID map[string]stylemap.StyleRules) stylemap.NodeStyleRules {
	if len(tree.Children) == 0 {
		rules := stylemap.NodeStyleRules{
			Width:  arrangeDefaultNodeWidth,
			Height: arrangeDefaultNodeHeight,
		}
		byID[tree.Node.ID] = stylemap.StyleRules{Node: &rules}
		return rules
	}

	childOrigin := compoundChildOrigin(tree.Node.BgKind, stylemap.CanvasPosition{})
	var content compoundSize
	switch tree.Node.BgKind {
	case node.BgKindCluster, node.BgKindNamespace:
		content = arrangeChildrenGrid(tree, childOrigin, byID)
	default:
		content = arrangeChildrenRow(tree, childOrigin, byID)
	}

	measured := measurePlacedChildrenBounds(childOrigin, tree.Children, byID)
	content = mergeCompoundContent(content, measured)

	finalSize := finalizeCompoundSize(tree.Node.BgKind, len(tree.Children), content)
	parentRules := stylemap.NodeStyleRules{
		Width:  finalSize.width,
		Height: finalSize.height,
	}
	byID[tree.Node.ID] = stylemap.StyleRules{Node: &parentRules}
	return parentRules
}

func gridDimensions(count int) (cols, rows int) {
	if count <= 0 {
		return 0, 0
	}
	cols = int(math.Ceil(math.Sqrt(float64(count))))
	rows = int(math.Ceil(float64(count) / float64(cols)))
	return cols, rows
}

func compoundNodeSize(childCount int) (width, height float64) {
	if childCount <= 0 {
		return 0, 0
	}
	scale := float64(childCount) * arrangeCompoundScale
	return float64(arrangeCellWidth) * scale, float64(arrangeCellHeight) * scale
}

func finalizeCompoundSize(parentKind node.BgKind, childCount int, content compoundSize) compoundSize {
	formulaWidth, formulaHeight := compoundNodeSize(childCount)
	padded := compoundSize{}
	switch parentKind {
	case node.BgKindNamespace:
		padded.width = arrangeCellPitchX + content.width + arrangeChildGap
		padded.height = arrangeCompoundTopInset + content.height + arrangeChildGap
	default:
		padded.width = content.width + arrangeCellPitchX
		padded.height = content.height + arrangeCellPitchY
	}
	width := padded.width
	if formulaWidth > width {
		width = formulaWidth
	}
	height := padded.height
	if parentKind != node.BgKindNamespace && formulaHeight > height {
		height = formulaHeight
	}
	return compoundSize{width: width, height: height}
}

func (state *arrangeState) finalizeCompoundSize(parentKind node.BgKind, childCount int, content compoundSize) compoundSize {
	return finalizeCompoundSize(parentKind, childCount, content)
}

func AutoArrangeStyleMap(content Content) stylemap.StyleMap {
	trees := buildK8sNodeTrees(content)
	if len(trees) == 0 {
		return stylemap.StyleMap{}
	}

	byID := make(map[string]stylemap.StyleRules)
	var cursorX float64
	var cursorY float64
	for _, tree := range trees {
		rootRules := arrangeNodeTree(tree, byID)
		if tree.Node.BgKind == node.BgKindCluster {
			clusterCanvasY := cursorY
			setNodePosition(byID, tree.Node.ID, stylemap.CanvasPosition{X: 0, Y: clusterCanvasY})
			canvasBottom := subtreeCanvasBottom(tree, 0, clusterCanvasY, byID)
			declaredBottom := clusterCanvasY + rootRules.Height
			if canvasBottom > declaredBottom {
				rules := byID[tree.Node.ID]
				if rules.Node == nil {
					rules.Node = &stylemap.NodeStyleRules{}
				}
				rules.Node.Height = canvasBottom - clusterCanvasY
				byID[tree.Node.ID] = rules
			}
			cursorY = canvasBottom + arrangeChildGap
		} else {
			setNodePosition(byID, tree.Node.ID, stylemap.CanvasPosition{X: cursorX, Y: 0})
			cursorX += rootRules.Width + arrangeChildGap
		}
	}

	return stylemap.StyleMap{ByID: byID}
}
