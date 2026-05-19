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
			Y: origin.Y + arrangeCellPitchY,
		}
	}
	return origin
}

func arrangeNodeTree(tree NodeTree, origin stylemap.CanvasPosition, byID map[string]stylemap.StyleRules) stylemap.NodeStyleRules {
	if len(tree.Children) == 0 {
		position := stylemap.CanvasPosition{
			X: origin.X + arrangeDefaultOffsetX,
			Y: origin.Y,
		}
		rules := stylemap.NodeStyleRules{
			Position: &position,
			Width:    arrangeDefaultNodeWidth,
			Height:   arrangeDefaultNodeHeight,
		}
		byID[tree.Node.ID] = stylemap.StyleRules{Node: &rules}
		return rules
	}

	childOrigin := compoundChildOrigin(tree.Node.BgKind, origin)
	content := compoundSize{}
	maxNodesPerRow := int(math.Sqrt(float64(len(tree.Children))))
	if maxNodesPerRow < 5 {
		maxNodesPerRow = 5
	} else if tree.Node.BgKind == node.BgKindCluster {
		maxNodesPerRow = 999
	}

	parentSide := compoundSize{}

	for idx, child := range tree.Children {
		childRules := arrangeNodeTree(child, childOrigin, byID)
		if idx == 0 {
			content.width = childRules.Width
			content.height = childRules.Height
		} else {
			content.width += arrangeChildGap + childRules.Width
			if childRules.Height > content.height {
				content.height = childRules.Height
			}
		}
		if childRules.Position != nil {
			childOrigin = stylemap.CanvasPosition{
				X: childRules.Position.X + childRules.Width + arrangeChildGap,
				Y: compoundChildOrigin(tree.Node.BgKind, childOrigin).Y,
			}

			if idx%maxNodesPerRow == 0 {
				childOrigin.Y += childRules.Position.Y + childRules.Height + arrangeCellPitchY
				childOrigin.X = origin.X + arrangeDefaultOffsetX
			}

			if idx <= maxNodesPerRow {
				parentSide.width += childRules.Width + arrangeChildGap
			}
			parentSide.height = childOrigin.Y + childRules.Height + arrangeCellPitchY - origin.Y

		}
	}

	parentPosition := origin
	parentRules := stylemap.NodeStyleRules{
		Position: &parentPosition,
		Width:    parentSide.width,
		Height:   parentSide.height,
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

func finalizeCompoundSize(childCount int, content compoundSize) compoundSize {
	formulaWidth, formulaHeight := compoundNodeSize(childCount)
	padded := compoundSize{
		width:  content.width + arrangeCellPitchX,
		height: content.height + arrangeCellPitchY,
	}
	width := padded.width
	if formulaWidth > width {
		width = formulaWidth
	}
	height := padded.height
	if formulaHeight > height {
		height = formulaHeight
	}
	return compoundSize{width: width, height: height}
}

func (state *arrangeState) finalizeCompoundSize(childCount int, content compoundSize) compoundSize {
	return finalizeCompoundSize(childCount, content)
}

func AutoArrangeStyleMap(content Content) stylemap.StyleMap {
	trees := buildK8sNodeTrees(content)
	if len(trees) == 0 {
		return stylemap.StyleMap{}
	}

	byID := make(map[string]stylemap.StyleRules)
	var cursorX float64
	var cursorY float64
	var lastClusterSize compoundSize
	for _, tree := range trees {
		if tree.Node.BgKind == node.BgKindCluster {
			rootRules := arrangeNodeTree(tree, stylemap.CanvasPosition{X: cursorX, Y: cursorY}, byID)
			if cursorX > 0 {
				position := stylemap.CanvasPosition{X: cursorX, Y: cursorY}
				rules := byID[tree.Node.ID]
				rules.Node.Position = &position
				byID[tree.Node.ID] = rules
			} else {
				rules := byID[tree.Node.ID]
				rules.Node.Position = nil
				byID[tree.Node.ID] = rules
			}
			cursorX += lastClusterSize.width + arrangeChildGap
			cursorY += lastClusterSize.height + arrangeChildGap
			lastClusterSize = compoundSize{width: rootRules.Width, height: rootRules.Height}
			continue
		}

		origin := stylemap.CanvasPosition{X: cursorX, Y: cursorY}
		rootRules := arrangeNodeTree(tree, origin, byID)
		cursorX += rootRules.Width + arrangeChildGap
		cursorY += rootRules.Height + arrangeChildGap
	}

	return stylemap.StyleMap{ByID: byID}
}
