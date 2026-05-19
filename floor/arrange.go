package floor

import (
	"math"

	"github.com/BumbleGrid/bgbase/node"
	"github.com/BumbleGrid/bgbase/stylemap"
)

const (
	arrangeDefaultNodeWidth  = 160
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
	for _, tree := range trees {
		rootRules := arrangeNodeTree(tree, byID)
		if tree.Node.BgKind == node.BgKindCluster {
			if cursorX > 0 {
				setNodePosition(byID, tree.Node.ID, stylemap.CanvasPosition{X: cursorX, Y: 0})
			} else {
				rules := byID[tree.Node.ID]
				rules.Node.Position = nil
				byID[tree.Node.ID] = rules
			}
		} else {
			setNodePosition(byID, tree.Node.ID, stylemap.CanvasPosition{X: cursorX, Y: 0})
		}
		cursorX += rootRules.Width + arrangeChildGap
	}

	return stylemap.StyleMap{ByID: byID}
}
