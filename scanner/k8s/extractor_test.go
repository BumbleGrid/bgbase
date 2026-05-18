package k8s

import (
	"context"
	"strings"
	"testing"

	"github.com/BumbleGrid/bgbase/node"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func int32Ptr(value int32) *int32 {
	return &value
}

func testTranslateContext() K8sTranslateContext {
	return K8sTranslateContext{
		Floor:         0,
		Meta:          node.Meta{ExtractorVersion: "test"},
		ClusterNodeID: "cluster/main",
	}
}

func k8sBgKindsEmittedByStockMapper() []node.BgKind {
	return []node.BgKind{
		node.BgKindCluster,
		node.BgKindNamespace,
		node.BgKindStorage,
		node.BgKindGateway,
		node.BgKindWorkload,
		node.BgKindJobRunner,
		node.BgKindServiceDiscovery,
		node.BgKindLoadBalancer,
		node.BgKindExternalService,
		node.BgKindConfigSource,
		node.BgKindSecretSource,
		node.BgKindNetworkPolicy,
	}
}

func k8sBgKindsNotEmittedByStockMapper() []node.BgKind {
	return []node.BgKind{
		node.BgKindDatabase,
		node.BgKindCache,
		node.BgKindMessageBroker,
	}
}

func collectBgKinds(nodes []node.Wrapper) map[node.BgKind]struct{} {
	out := make(map[node.BgKind]struct{}, len(nodes))
	for idx := range nodes {
		out[nodes[idx].Data.BgKind] = struct{}{}
	}
	return out
}

func clusterScopedK8sObjects() []runtime.Object {
	return []runtime.Object{
		&corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: "pv-shared"}},
		&networkingv1.IngressClass{ObjectMeta: metav1.ObjectMeta{Name: "public"}},
	}
}

func namespacedK8sObjects(namespace string) []runtime.Object {
	return []runtime.Object{
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: namespace},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: namespace},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: namespace},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Name: "web-rs", Namespace: namespace},
		},
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{Name: "tick", Namespace: namespace},
		},
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "once", Namespace: namespace},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "clusterip", Namespace: namespace},
			Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, ClusterIP: "10.96.0.1"},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "elb", Namespace: namespace},
			Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "extdns", Namespace: namespace},
			Spec: corev1.ServiceSpec{
				Type:         corev1.ServiceTypeExternalName,
				ExternalName: "upstream.example.com",
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: namespace},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "cfg", Namespace: namespace},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "tok", Namespace: namespace},
		},
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: namespace},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "default-deny", Namespace: namespace},
		},
		&autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: namespace},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "web",
				},
				MinReplicas: int32Ptr(1),
				MaxReplicas: 3,
			},
		},
	}
}

func fullClusterObjects(namespace string) []runtime.Object {
	out := append([]runtime.Object{}, clusterScopedK8sObjects()...)
	out = append(out, namespacedK8sObjects(namespace)...)
	return out
}

func TestFloor0Extractor_coversAllBgKindsFromStockK8sMapper(t *testing.T) {
	ctx := context.Background()
	nsName := "demo"
	cs := fake.NewSimpleClientset(fullClusterObjects(nsName)...)
	reader := NewReaderWithClientset(cs)
	trans := NewNodeTranslator()
	res := NewEdgeResolver()

	content, err := Floor0Extractor(ctx, reader, trans, res, testTranslateContext())
	if err != nil {
		t.Fatalf("Floor0Extractor: %v", err)
	}
	seen := collectBgKinds(content.Nodes)
	for _, wantKind := range k8sBgKindsEmittedByStockMapper() {
		if _, ok := seen[wantKind]; !ok {
			t.Errorf("missing bgKind %q among %d nodes", wantKind, len(content.Nodes))
		}
	}
	for _, absent := range k8sBgKindsNotEmittedByStockMapper() {
		if _, ok := seen[absent]; ok {
			t.Errorf("unexpected bgKind %q (stock K8s mapper should not emit it)", absent)
		}
	}
	if content.Floor != 0 {
		t.Fatalf("Floor = %d", content.Floor)
	}
	for idx := range content.Nodes {
		if content.Nodes[idx].Data.Floor != 0 {
			t.Fatalf("node %q Floor = %d", content.Nodes[idx].Data.ID, content.Nodes[idx].Data.Floor)
		}
	}
}

func TestFloor0Extractor_emptyCluster(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewSimpleClientset()
	reader := NewReaderWithClientset(cs)
	trans := NewNodeTranslator()
	res := NewEdgeResolver()

	content, err := Floor0Extractor(ctx, reader, trans, res, testTranslateContext())
	if err != nil {
		t.Fatalf("Floor0Extractor: %v", err)
	}
	if len(content.Nodes) != 1 {
		t.Fatalf("expected 1 cluster node, got %d", len(content.Nodes))
	}
	if content.Nodes[0].Data.BgKind != node.BgKindCluster {
		t.Fatalf("sole node bgKind = %q, want Cluster", content.Nodes[0].Data.BgKind)
	}
	if content.Nodes[0].Data.ID != "cluster/main" {
		t.Fatalf("cluster node id = %q", content.Nodes[0].Data.ID)
	}
	if content.Edges == nil {
		t.Fatal("Edges slice is nil, want non-nil (possibly empty)")
	}
	if len(content.Edges) != 0 {
		t.Fatalf("expected 0 edges, got %d", len(content.Edges))
	}
}

func TestFloor0Extractor_unconfiguredLister(t *testing.T) {
	ctx := context.Background()
	reader := NewReader(nil)
	trans := NewNodeTranslator()
	res := NewEdgeResolver()

	_, err := Floor0Extractor(ctx, reader, trans, res, testTranslateContext())
	if err == nil {
		t.Fatal("expected error when kubernetes clientset is not configured")
	}
	if !strings.Contains(err.Error(), "list namespaces") {
		t.Fatalf("error should mention list namespaces, got: %v", err)
	}
}

func TestFloor0Extractor_floorFromTranslateContext(t *testing.T) {
	ctx := context.Background()
	nsName := "prod"
	cs := fake.NewSimpleClientset(fullClusterObjects(nsName)...)
	reader := NewReaderWithClientset(cs)
	trans := NewNodeTranslator()
	res := NewEdgeResolver()

	tctx := testTranslateContext()
	tctx.Floor = 2

	content, err := Floor0Extractor(ctx, reader, trans, res, tctx)
	if err != nil {
		t.Fatalf("Floor0Extractor: %v", err)
	}
	if content.Floor != 2 {
		t.Fatalf("content.Floor = %d", content.Floor)
	}
	for idx := range content.Nodes {
		if content.Nodes[idx].Data.Floor != 2 {
			t.Fatalf("node %q Floor = %d", content.Nodes[idx].Data.ID, content.Nodes[idx].Data.Floor)
		}
	}
}

func TestFloor0Extractor_multipleNamespaces(t *testing.T) {
	ctx := context.Background()
	objs := append(clusterScopedK8sObjects(), namespacedK8sObjects("alpha")...)
	objs = append(objs, namespacedK8sObjects("beta")...)
	cs := fake.NewSimpleClientset(objs...)
	reader := NewReaderWithClientset(cs)
	trans := NewNodeTranslator()
	res := NewEdgeResolver()

	content, err := Floor0Extractor(ctx, reader, trans, res, testTranslateContext())
	if err != nil {
		t.Fatalf("Floor0Extractor: %v", err)
	}
	byID := make(map[string]struct{}, len(content.Nodes))
	for idx := range content.Nodes {
		byID[content.Nodes[idx].Data.ID] = struct{}{}
	}
	for _, ns := range []string{"alpha", "beta"} {
		want := "cluster/main/k8s/namespaces/" + ns + "/deployments/web"
		if _, ok := byID[want]; !ok {
			t.Errorf("missing deployment node id %q", want)
		}
	}
	nsSeen := 0
	for idx := range content.Nodes {
		if content.Nodes[idx].Data.BgKind == node.BgKindNamespace {
			nsSeen++
		}
	}
	if nsSeen != 2 {
		t.Fatalf("namespace nodes = %d, want 2", nsSeen)
	}
}

func TestFloor0Extractor_cancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cs := fake.NewSimpleClientset(fullClusterObjects("demo")...)
	reader := NewReaderWithClientset(cs)
	trans := NewNodeTranslator()
	res := NewEdgeResolver()

	_, err := Floor0Extractor(ctx, reader, trans, res, testTranslateContext())
	if err == nil {
		t.Fatal("expected error when context is cancelled before list")
	}
}

func TestFloor0Extractor_nodesOnlyUseDefinedBgKinds(t *testing.T) {
	ctx := context.Background()
	cs := fake.NewSimpleClientset(fullClusterObjects("demo")...)
	reader := NewReaderWithClientset(cs)
	trans := NewNodeTranslator()
	res := NewEdgeResolver()

	content, err := Floor0Extractor(ctx, reader, trans, res, testTranslateContext())
	if err != nil {
		t.Fatalf("Floor0Extractor: %v", err)
	}
	valid := make(map[node.BgKind]struct{})
	for _, kind := range k8sBgKindsEmittedByStockMapper() {
		valid[kind] = struct{}{}
	}
	for _, kind := range k8sBgKindsNotEmittedByStockMapper() {
		valid[kind] = struct{}{}
	}
	for idx := range content.Nodes {
		kind := content.Nodes[idx].Data.BgKind
		if _, ok := valid[kind]; !ok {
			t.Errorf("node %q has bgKind %q not in taxonomy union", content.Nodes[idx].Data.ID, kind)
		}
	}
}

func TestFloor0Extractor_expectedNodeCountSingleNamespace(t *testing.T) {
	ctx := context.Background()
	nsName := "demo"
	cs := fake.NewSimpleClientset(fullClusterObjects(nsName)...)
	reader := NewReaderWithClientset(cs)
	trans := NewNodeTranslator()
	res := NewEdgeResolver()

	content, err := Floor0Extractor(ctx, reader, trans, res, testTranslateContext())
	if err != nil {
		t.Fatalf("Floor0Extractor: %v", err)
	}
	wantNodes := 1 + 2 + 16
	if len(content.Nodes) != wantNodes {
		t.Fatalf("len(nodes) = %d, want %d (cluster + cluster-scoped PV+IngressClass + namespace + 15 namespaced kinds)", len(content.Nodes), wantNodes)
	}
}
