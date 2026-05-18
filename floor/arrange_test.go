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
	if pvPos.X != 0 || pvPos.Y != 2*arrangeCellPitchY {
		t.Fatalf("pv position = (%v,%v), want (0,%v)", pvPos.X, pvPos.Y, 2*arrangeCellPitchY)
	}

	webPos := styleMap.ByID["cluster/main/k8s/namespaces/prod/deployments/web"].Node.Position
	svcPos := styleMap.ByID["cluster/main/k8s/namespaces/prod/services/api"].Node.Position
	if webPos == nil || svcPos == nil {
		t.Fatal("expected namespace children to receive positions")
	}
	if webPos.X <= nsPos.X || webPos.Y <= nsPos.Y {
		t.Fatalf("web should be laid out inside namespace at offset from (%v,%v), got (%v,%v)", nsPos.X, nsPos.Y, webPos.X, webPos.Y)
	}
	if svcPos.X == webPos.X && svcPos.Y == webPos.Y {
		t.Fatal("namespace children should not share the same position")
	}

	nsRules := styleMap.ByID["cluster/main/k8s/namespaces/prod"].Node
	wantWidth, wantHeight := compoundNodeSize(2)
	if nsRules.Width != wantWidth || nsRules.Height != wantHeight {
		t.Fatalf("namespace size = (%v,%v), want (%v,%v)", nsRules.Width, nsRules.Height, wantWidth, wantHeight)
	}
}

func TestCompoundNodeSize(t *testing.T) {
	width, height := compoundNodeSize(3)
	wantWidth := 178 * 3 * 1.3
	wantHeight := 104 * 3 * 1.3
	if width != wantWidth || height != wantHeight {
		t.Fatalf("compoundNodeSize(3) = (%v,%v), want (%v,%v)", width, height, wantWidth, wantHeight)
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
	if clusterRules.Position != nil {
		t.Fatal("cluster container node should not receive a position entry")
	}
	wantWidth, wantHeight := compoundNodeSize(2)
	if clusterRules.Width != wantWidth || clusterRules.Height != wantHeight {
		t.Fatalf("cluster size = (%v,%v), want (%v,%v)", clusterRules.Width, clusterRules.Height, wantWidth, wantHeight)
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
	if bPos.X <= aPos.X {
		t.Fatalf("second cluster should be offset to the right, got b at x=%v", bPos.X)
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
