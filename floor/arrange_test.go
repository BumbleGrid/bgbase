package floor

import (
	"testing"

	"github.com/BumbleGrid/bgbase/node"
	"github.com/BumbleGrid/bgbase/stylemap"
)

func strPtr(value string) *string {
	return &value
}

func TestAutoArrangeStyleMap_implicitCluster(t *testing.T) {
	clusterID := "cluster/main"
	content := Content{
		Nodes: []node.Wrapper{
			{Data: node.Data{ID: "cluster/main/k8s/namespaces/prod", Label: "prod", BgKind: node.BgKindNamespace, Parent: strPtr(clusterID)}},
			{Data: node.Data{ID: "cluster/main/k8s/namespaces/prod/deployments/web", Label: "web", BgKind: node.BgKindWorkload, Parent: strPtr("cluster/main/k8s/namespaces/prod")}},
			{Data: node.Data{ID: "cluster/main/k8s/namespaces/prod/services/api", Label: "api", BgKind: node.BgKindServiceDiscovery, Parent: strPtr("cluster/main/k8s/namespaces/prod")}},
			{Data: node.Data{ID: "cluster/main/k8s/persistentvolumes/pv1", Label: "pv1", BgKind: node.BgKindStorage, Parent: strPtr(clusterID)}},
		},
	}

	styleMap := AutoArrangeStyleMap(content)
	if styleMap.ByID == nil {
		t.Fatal("expected byId entries")
	}

	nsPos := styleMap.ByID["cluster/main/k8s/namespaces/prod"].Node.Position
	pvPos := styleMap.ByID["cluster/main/k8s/persistentvolumes/pv1"].Node.Position
	if nsPos == nil || pvPos == nil {
		t.Fatal("expected cluster children to receive positions")
	}
	if nsPos.X != 0 || nsPos.Y != 0 {
		t.Fatalf("first cluster child position = (%v,%v), want (0,0)", nsPos.X, nsPos.Y)
	}
	nsWidth := styleMap.ByID["cluster/main/k8s/namespaces/prod"].Node.Width
	wantPVX := nsWidth + arrangeCellGap
	if pvPos.X != wantPVX || pvPos.Y != 0 {
		t.Fatalf("pv position = (%v,%v), want (%v,0)", pvPos.X, pvPos.Y, wantPVX)
	}

	webPos := styleMap.ByID["cluster/main/k8s/namespaces/prod/deployments/web"].Node.Position
	svcPos := styleMap.ByID["cluster/main/k8s/namespaces/prod/services/api"].Node.Position
	if webPos == nil || svcPos == nil {
		t.Fatal("expected namespace children to receive positions")
	}
	if webPos.X != arrangeCellPitchX || webPos.Y != arrangeCompoundTopInset {
		t.Fatalf("web position = (%v,%v), want parent-relative (%v,%v)", webPos.X, webPos.Y, arrangeCellPitchX, arrangeCompoundTopInset)
	}
	if svcPos.X == webPos.X && svcPos.Y == webPos.Y {
		t.Fatal("namespace children should not share the same position")
	}

	nsRules := styleMap.ByID["cluster/main/k8s/namespaces/prod"].Node
	wantNsMinWidth := float64(arrangeCellPitchX + 2*arrangeDefaultNodeWidth + arrangeChildGap + arrangeChildGap)
	wantNsMinHeight := float64(arrangeCompoundTopInset + arrangeDefaultNodeHeight + arrangeChildGap)
	if nsRules.Width < wantNsMinWidth || nsRules.Height < wantNsMinHeight {
		t.Fatalf("namespace size = (%v,%v), want at least (%v,%v)", nsRules.Width, nsRules.Height, wantNsMinWidth, wantNsMinHeight)
	}
}

func TestAutoArrangeStyleMap_namespaceChildrenGrid(t *testing.T) {
	clusterID := "cluster/main"
	nsID := "cluster/main/k8s/namespaces/prod"
	content := Content{
		Nodes: []node.Wrapper{
			{Data: node.Data{ID: clusterID, Label: "main", BgKind: node.BgKindCluster}},
			{Data: node.Data{ID: nsID, Label: "prod", BgKind: node.BgKindNamespace, Parent: strPtr(clusterID)}},
			{Data: node.Data{ID: nsID + "/deployments/a", Label: "a", BgKind: node.BgKindWorkload, Parent: strPtr(nsID)}},
			{Data: node.Data{ID: nsID + "/deployments/b", Label: "b", BgKind: node.BgKindWorkload, Parent: strPtr(nsID)}},
			{Data: node.Data{ID: nsID + "/deployments/c", Label: "c", BgKind: node.BgKindWorkload, Parent: strPtr(nsID)}},
			{Data: node.Data{ID: nsID + "/deployments/d", Label: "d", BgKind: node.BgKindWorkload, Parent: strPtr(nsID)}},
			{Data: node.Data{ID: nsID + "/deployments/e", Label: "e", BgKind: node.BgKindWorkload, Parent: strPtr(nsID)}},
		},
	}

	styleMap := AutoArrangeStyleMap(content)
	firstRow := []string{
		nsID + "/deployments/a",
		nsID + "/deployments/b",
		nsID + "/deployments/c",
	}
	secondRow := []string{
		nsID + "/deployments/d",
		nsID + "/deployments/e",
	}

	rowStartX := styleMap.ByID[firstRow[0]].Node.Position.X
	if rowStartX != arrangeCellPitchX {
		t.Fatalf("first child x = %v, want %v", rowStartX, arrangeCellPitchX)
	}
	firstRowY := styleMap.ByID[firstRow[0]].Node.Position.Y
	if firstRowY != arrangeCompoundTopInset {
		t.Fatalf("first child y = %v, want %v", firstRowY, arrangeCompoundTopInset)
	}
	for _, id := range firstRow[1:] {
		pos := styleMap.ByID[id].Node.Position
		if pos.Y != firstRowY {
			t.Fatalf("%s y = %v, want first row y %v", id, pos.Y, firstRowY)
		}
	}

	secondRowStart := styleMap.ByID[secondRow[0]].Node.Position
	if secondRowStart.X != rowStartX {
		t.Fatalf("second row should start at x=%v, got %v", rowStartX, secondRowStart.X)
	}
	secondRowY := secondRowStart.Y
	if secondRowY <= firstRowY {
		t.Fatalf("second row y = %v, want below first row y %v", secondRowY, firstRowY)
	}
}

func TestAutoArrangeStyleMap_clusterNamespaceGrid(t *testing.T) {
	clusterID := "cluster/main"
	content := Content{
		Nodes: []node.Wrapper{
			{Data: node.Data{ID: clusterID, Label: "main", BgKind: node.BgKindCluster}},
			{Data: node.Data{ID: "cluster/main/k8s/namespaces/a", Label: "a", BgKind: node.BgKindNamespace, Parent: strPtr(clusterID)}},
			{Data: node.Data{ID: "cluster/main/k8s/namespaces/b", Label: "b", BgKind: node.BgKindNamespace, Parent: strPtr(clusterID)}},
			{Data: node.Data{ID: "cluster/main/k8s/namespaces/c", Label: "c", BgKind: node.BgKindNamespace, Parent: strPtr(clusterID)}},
			{Data: node.Data{ID: "cluster/main/k8s/namespaces/d", Label: "d", BgKind: node.BgKindNamespace, Parent: strPtr(clusterID)}},
			{Data: node.Data{ID: "cluster/main/k8s/namespaces/e", Label: "e", BgKind: node.BgKindNamespace, Parent: strPtr(clusterID)}},
		},
	}

	styleMap := AutoArrangeStyleMap(content)
	gridCols, gridRows := gridDimensions(5)
	if gridCols != 3 || gridRows != 2 {
		t.Fatalf("gridDimensions(5) = (%d,%d), want (3,2)", gridCols, gridRows)
	}

	firstRow := []string{
		"cluster/main/k8s/namespaces/a",
		"cluster/main/k8s/namespaces/b",
		"cluster/main/k8s/namespaces/c",
	}
	secondRow := []string{
		"cluster/main/k8s/namespaces/d",
		"cluster/main/k8s/namespaces/e",
	}

	rowStartX := styleMap.ByID[firstRow[0]].Node.Position.X
	firstRowY := styleMap.ByID[firstRow[0]].Node.Position.Y
	for _, id := range firstRow[1:] {
		pos := styleMap.ByID[id].Node.Position
		if pos.Y != firstRowY {
			t.Fatalf("%s y = %v, want first row y %v", id, pos.Y, firstRowY)
		}
	}
	secondRowStart := styleMap.ByID[secondRow[0]].Node.Position
	if secondRowStart.X != rowStartX {
		t.Fatalf("second row should start at x=%v, got %v", rowStartX, secondRowStart.X)
	}
	secondRowY := secondRowStart.Y
	if secondRowY <= firstRowY {
		t.Fatalf("second row y = %v, want below first row y %v", secondRowY, firstRowY)
	}
	for _, id := range secondRow[1:] {
		pos := styleMap.ByID[id].Node.Position
		if pos.Y != secondRowY {
			t.Fatalf("%s y = %v, want second row y %v", id, pos.Y, secondRowY)
		}
		if pos.X <= secondRowStart.X {
			t.Fatalf("%s should be to the right of %s on the same row", id, secondRow[0])
		}
	}
}

func TestAutoArrangeStyleMap_multipleNamespacesInCluster(t *testing.T) {
	clusterID := "cluster/main"
	nsAlpha := "cluster/main/k8s/namespaces/alpha"
	nsBeta := "cluster/main/k8s/namespaces/beta"
	content := Content{
		Nodes: []node.Wrapper{
			{Data: node.Data{ID: clusterID, Label: "main", BgKind: node.BgKindCluster}},
			{Data: node.Data{ID: nsAlpha, Label: "alpha", BgKind: node.BgKindNamespace, Parent: strPtr(clusterID)}},
			{Data: node.Data{ID: nsBeta, Label: "beta", BgKind: node.BgKindNamespace, Parent: strPtr(clusterID)}},
			{Data: node.Data{ID: nsAlpha + "/deployments/web", Label: "web", BgKind: node.BgKindWorkload, Parent: strPtr(nsAlpha)}},
			{Data: node.Data{ID: nsBeta + "/deployments/api", Label: "api", BgKind: node.BgKindWorkload, Parent: strPtr(nsBeta)}},
		},
	}

	styleMap := AutoArrangeStyleMap(content)
	alphaWeb := styleMap.ByID[nsAlpha+"/deployments/web"].Node.Position
	betaAPI := styleMap.ByID[nsBeta+"/deployments/api"].Node.Position
	if alphaWeb == nil || betaAPI == nil {
		t.Fatal("expected workload positions inside namespaces")
	}
	if alphaWeb.X != arrangeCellPitchX || alphaWeb.Y != arrangeCompoundTopInset {
		t.Fatalf("alpha web position = (%v,%v), want (%v,%v)", alphaWeb.X, alphaWeb.Y, arrangeCellPitchX, arrangeCompoundTopInset)
	}
	if betaAPI.X != arrangeCellPitchX || betaAPI.Y != arrangeCompoundTopInset {
		t.Fatalf("beta api position = (%v,%v), want (%v,%v)", betaAPI.X, betaAPI.Y, arrangeCellPitchX, arrangeCompoundTopInset)
	}
}

func TestCompoundNodeSize(t *testing.T) {
	width, height := compoundNodeSize(3)
	wantWidth := float64(arrangeCellWidth * 3)
	wantHeight := float64(arrangeCellHeight * 3)
	if width != wantWidth || height != wantHeight {
		t.Fatalf("compoundNodeSize(3) = (%v,%v), want (%v,%v)", width, height, wantWidth, wantHeight)
	}
}

func TestFinalizeCompoundSize_usesPackedLayout(t *testing.T) {
	state := &arrangeState{sizes: make(map[string]compoundSize)}
	packed := compoundSize{width: 400, height: 200}
	got := state.finalizeCompoundSize(node.BgKindCluster, 2, packed)
	if got.width <= packed.width || got.height <= packed.height {
		t.Fatalf("finalizeCompoundSize = (%v,%v), want larger than packed (%v,%v)", got.width, got.height, packed.width, packed.height)
	}
}

func TestAutoArrangeStyleMap_explicitCluster(t *testing.T) {
	clusterID := "cluster-a"
	content := Content{
		Nodes: []node.Wrapper{
			{Data: node.Data{ID: clusterID, Label: "A", BgKind: node.BgKindCluster}},
			{Data: node.Data{ID: "workload-1", Label: "w1", BgKind: node.BgKindWorkload, Parent: strPtr(clusterID)}},
			{Data: node.Data{ID: "workload-2", Label: "w2", BgKind: node.BgKindWorkload, Parent: strPtr(clusterID)}},
		},
	}

	styleMap := AutoArrangeStyleMap(content)
	w1 := styleMap.ByID["workload-1"].Node.Position
	w2 := styleMap.ByID["workload-2"].Node.Position
	if w1 == nil || w2 == nil {
		t.Fatal("expected workload positions")
	}
	if w1.X != 0 || w1.Y != 0 {
		t.Fatalf("first workload at (%v,%v), want (0,0)", w1.X, w1.Y)
	}
	if w2.X != arrangeCellPitchX || w2.Y != 0 {
		t.Fatalf("second workload at (%v,%v), want (%v,0)", w2.X, w2.Y, arrangeCellPitchX)
	}
	clusterRules := styleMap.ByID[clusterID].Node
	if clusterRules == nil {
		t.Fatal("expected cluster style rules")
	}
	if clusterRules.Position == nil || clusterRules.Position.X != 0 || clusterRules.Position.Y != 0 {
		t.Fatalf("single cluster position = %v, want (0,0)", clusterRules.Position)
	}
	if clusterRules.Width < arrangeCellPitchX*2 || clusterRules.Height < arrangeCellPitchY {
		t.Fatalf("cluster size = (%v,%v), expected at least packed layout minimum", clusterRules.Width, clusterRules.Height)
	}
	if clusterRules.Width <= float64(arrangeCellWidth*2) {
		t.Fatal("cluster width should exceed formula-only size when children are packed side by side")
	}
}

func TestAutoArrangeStyleMap_multipleClustersOffset(t *testing.T) {
	content := Content{
		Nodes: []node.Wrapper{
			{Data: node.Data{ID: "cluster/a", Label: "A", BgKind: node.BgKindCluster}},
			{Data: node.Data{ID: "cluster/b", Label: "B", BgKind: node.BgKindCluster}},
			{Data: node.Data{ID: "a-child", Label: "a1", BgKind: node.BgKindWorkload, Parent: strPtr("cluster/a")}},
			{Data: node.Data{ID: "b-child", Label: "b1", BgKind: node.BgKindWorkload, Parent: strPtr("cluster/b")}},
		},
	}

	styleMap := AutoArrangeStyleMap(content)
	aPos := styleMap.ByID["a-child"].Node.Position
	bPos := styleMap.ByID["b-child"].Node.Position
	if aPos == nil || bPos == nil {
		t.Fatal("expected child positions")
	}
	if aPos.X != 0 {
		t.Fatalf("first cluster child x = %v, want 0", aPos.X)
	}
	if bPos.X != 0 {
		t.Fatalf("cluster child position should be parent-relative, got b at x=%v", bPos.X)
	}
	clusterARules := styleMap.ByID["cluster/a"].Node
	clusterBPos := styleMap.ByID["cluster/b"].Node.Position
	if clusterBPos == nil {
		t.Fatal("expected second cluster position")
	}
	wantClusterBY := clusterARules.Height + arrangeChildGap
	if clusterBPos.X != 0 || clusterBPos.Y != wantClusterBY {
		t.Fatalf("second cluster position = (%v,%v), want (0,%v)", clusterBPos.X, clusterBPos.Y, wantClusterBY)
	}
}

func TestAutoArrangeStyleMap_emptyFloor(t *testing.T) {
	styleMap := AutoArrangeStyleMap(Content{})
	if styleMap.ByID != nil && len(styleMap.ByID) != 0 {
		t.Fatalf("expected empty byId, got %d entries", len(styleMap.ByID))
	}
}

func TestGridDimensions(t *testing.T) {
	cases := []struct {
		count    int
		wantCols int
		wantRows int
	}{
		{0, 0, 0},
		{1, 1, 1},
		{4, 2, 2},
		{5, 3, 2},
		{9, 3, 3},
	}
	for _, tc := range cases {
		cols, rows := gridDimensions(tc.count)
		if cols != tc.wantCols || rows != tc.wantRows {
			t.Errorf("gridDimensions(%d) = (%d,%d), want (%d,%d)", tc.count, cols, rows, tc.wantCols, tc.wantRows)
		}
	}
}

func TestAutoArrangeStyleMap_returnsStyleMapShape(t *testing.T) {
	styleMap := AutoArrangeStyleMap(Content{Nodes: []node.Wrapper{
		{Data: node.Data{ID: "cluster/x", BgKind: node.BgKindCluster}},
		{Data: node.Data{ID: "leaf", BgKind: node.BgKindWorkload, Parent: strPtr("cluster/x")}},
	}})
	var _ stylemap.StyleMap = styleMap
}
