package floor

import (
	"testing"

	"github.com/BumbleGrid/bgbase/node"
)

func TestBuildK8sNodeTrees_implicitCluster(t *testing.T) {
	clusterID := "cluster/main"
	content := Content{
		Nodes: []node.Wrapper{
			{Data: node.Data{ID: "cluster/main/k8s/namespaces/prod", Label: "prod", BgKind: node.BgKindNamespace, Parent: strPtr(clusterID)}},
			{Data: node.Data{ID: "cluster/main/k8s/namespaces/prod/deployments/web", Label: "web", BgKind: node.BgKindWorkload, Parent: strPtr("cluster/main/k8s/namespaces/prod")}},
			{Data: node.Data{ID: "cluster/main/k8s/persistentvolumes/pv1", Label: "pv1", BgKind: node.BgKindStorage, Parent: strPtr(clusterID)}},
		},
	}

	trees := buildK8sNodeTrees(content)
	if len(trees) != 2 {
		t.Fatalf("got %d root trees, want 2 (namespace + pv)", len(trees))
	}

	nsTree, pvTree := trees[0], trees[1]
	if nsTree.Node.BgKind != node.BgKindNamespace {
		nsTree, pvTree = pvTree, nsTree
	}
	if nsTree.Node.ID != "cluster/main/k8s/namespaces/prod" {
		t.Fatalf("namespace root id = %q", nsTree.Node.ID)
	}
	if len(nsTree.Children) != 1 || nsTree.Children[0].Node.ID != "cluster/main/k8s/namespaces/prod/deployments/web" {
		t.Fatalf("namespace children = %#v", nsTree.Children)
	}
	if pvTree.Node.ID != "cluster/main/k8s/persistentvolumes/pv1" {
		t.Fatalf("pv root id = %q", pvTree.Node.ID)
	}
	if len(pvTree.Children) != 0 {
		t.Fatalf("pv should have no children, got %#v", pvTree.Children)
	}
}

func TestBuildK8sNodeTrees_explicitCluster(t *testing.T) {
	clusterID := "cluster-a"
	content := Content{
		Nodes: []node.Wrapper{
			{Data: node.Data{ID: clusterID, Label: "A", BgKind: node.BgKindCluster}},
			{Data: node.Data{ID: "workload-1", Label: "w1", BgKind: node.BgKindWorkload, Parent: strPtr(clusterID)}},
			{Data: node.Data{ID: "workload-2", Label: "w2", BgKind: node.BgKindWorkload, Parent: strPtr(clusterID)}},
		},
	}

	trees := buildK8sNodeTrees(content)
	if len(trees) != 1 {
		t.Fatalf("got %d root trees, want 1 cluster", len(trees))
	}
	cluster := trees[0]
	if cluster.Node.ID != clusterID {
		t.Fatalf("root id = %q", cluster.Node.ID)
	}
	if len(cluster.Children) != 2 {
		t.Fatalf("cluster children = %d, want 2", len(cluster.Children))
	}
}

func TestBuildK8sNodeTrees_invalidParentsBecomeRoots(t *testing.T) {
	content := Content{
		Nodes: []node.Wrapper{
			{Data: node.Data{ID: "cluster/x", BgKind: node.BgKindCluster, Parent: strPtr("bogus")}},
			{Data: node.Data{ID: "ns/y", BgKind: node.BgKindNamespace, Parent: strPtr("not-a-cluster")}},
			{Data: node.Data{ID: "leaf", BgKind: node.BgKindWorkload, Parent: strPtr("missing")}},
		},
	}

	trees := buildK8sNodeTrees(content)
	if len(trees) != 3 {
		t.Fatalf("got %d roots, want 3", len(trees))
	}
}
