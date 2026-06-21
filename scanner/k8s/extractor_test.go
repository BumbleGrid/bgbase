package k8s

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/BumbleGrid/bgbase/edge"
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
		node.BgKindGateway,
		node.BgKindWorkload,
		node.BgKindJobRunner,
		node.BgKindServiceDiscovery,
		node.BgKindLoadBalancer,
		node.BgKindExternalService,
	}
}

func k8sBgKindsNotEmittedByNarrowedMapper() []node.BgKind {
	return []node.BgKind{
		node.BgKindStorage,
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
	}
}

func fullClusterObjects(namespace string) []runtime.Object {
	return namespacedK8sObjects(namespace)
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
	for _, absent := range k8sBgKindsNotEmittedByNarrowedMapper() {
		if _, ok := seen[absent]; ok {
			t.Errorf("unexpected bgKind %q (narrowed Floor 0 mapper should not emit it)", absent)
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
	objs := append(namespacedK8sObjects("alpha"), namespacedK8sObjects("beta")...)
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
	wantNodes := 11
	if len(content.Nodes) != wantNodes {
		t.Fatalf("len(nodes) = %d, want %d (cluster + namespace + 9 namespaced resources across 8 kinds)", len(content.Nodes), wantNodes)
	}
}

type trackingLister struct {
	inner   K8sLister
	forbidden map[string]int
}

func newTrackingLister(inner K8sLister) *trackingLister {
	return &trackingLister{
		inner:     inner,
		forbidden: make(map[string]int),
	}
}

func (tracker *trackingLister) recordForbidden(method string) error {
	tracker.forbidden[method]++
	return fmt.Errorf("unexpected %s call", method)
}

func (tracker *trackingLister) ListNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	return tracker.inner.ListNamespaces(ctx)
}

func (tracker *trackingLister) ListPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	return nil, tracker.recordForbidden("ListPersistentVolumes")
}

func (tracker *trackingLister) ListIngressClasses(ctx context.Context) ([]networkingv1.IngressClass, error) {
	return nil, tracker.recordForbidden("ListIngressClasses")
}

func (tracker *trackingLister) ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	return tracker.inner.ListDeployments(ctx, namespace)
}

func (tracker *trackingLister) ListStatefulSets(ctx context.Context, namespace string) ([]appsv1.StatefulSet, error) {
	return tracker.inner.ListStatefulSets(ctx, namespace)
}

func (tracker *trackingLister) ListDaemonSets(ctx context.Context, namespace string) ([]appsv1.DaemonSet, error) {
	return tracker.inner.ListDaemonSets(ctx, namespace)
}

func (tracker *trackingLister) ListReplicaSets(ctx context.Context, namespace string) ([]appsv1.ReplicaSet, error) {
	return nil, tracker.recordForbidden("ListReplicaSets")
}

func (tracker *trackingLister) ListCronJobs(ctx context.Context, namespace string) ([]batchv1.CronJob, error) {
	return tracker.inner.ListCronJobs(ctx, namespace)
}

func (tracker *trackingLister) ListJobs(ctx context.Context, namespace string) ([]batchv1.Job, error) {
	return tracker.inner.ListJobs(ctx, namespace)
}

func (tracker *trackingLister) ListServices(ctx context.Context, namespace string) ([]corev1.Service, error) {
	return tracker.inner.ListServices(ctx, namespace)
}

func (tracker *trackingLister) ListIngresses(ctx context.Context, namespace string) ([]networkingv1.Ingress, error) {
	return tracker.inner.ListIngresses(ctx, namespace)
}

func (tracker *trackingLister) ListConfigMaps(ctx context.Context, namespace string) ([]corev1.ConfigMap, error) {
	return nil, tracker.recordForbidden("ListConfigMaps")
}

func (tracker *trackingLister) ListSecrets(ctx context.Context, namespace string) ([]corev1.Secret, error) {
	return nil, tracker.recordForbidden("ListSecrets")
}

func (tracker *trackingLister) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	return nil, tracker.recordForbidden("ListPersistentVolumeClaims")
}

func (tracker *trackingLister) ListNetworkPolicies(ctx context.Context, namespace string) ([]networkingv1.NetworkPolicy, error) {
	return nil, tracker.recordForbidden("ListNetworkPolicies")
}

func (tracker *trackingLister) ListHorizontalPodAutoscalersV2(ctx context.Context, namespace string) ([]autoscalingv2.HorizontalPodAutoscaler, error) {
	return nil, tracker.recordForbidden("ListHorizontalPodAutoscalersV2")
}

func TestFloor0Extractor_doesNotListRemovedResourceTypes(t *testing.T) {
	ctx := context.Background()
	nsName := "demo"
	cs := fake.NewSimpleClientset(fullClusterObjects(nsName)...)
	reader := NewReaderWithClientset(cs)
	lister := newTrackingLister(reader)
	trans := NewNodeTranslator()
	res := NewEdgeResolver()

	_, err := Floor0Extractor(ctx, lister, trans, res, testTranslateContext())
	if err != nil {
		t.Fatalf("Floor0Extractor: %v", err)
	}
	if len(lister.forbidden) != 0 {
		t.Fatalf("removed resource types were listed: %v", lister.forbidden)
	}
}

func TestFloor0Extractor_resolvesCoreWorkloadEdges(t *testing.T) {
	ctx := context.Background()
	nsName := "demo"
	cs := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: nsName}},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: nsName},
			Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, ClusterIP: "10.96.0.1"},
		},
		&networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: nsName}},
	)
	reader := NewReaderWithClientset(cs)
	trans := NewNodeTranslator()
	res := NewEdgeResolver()

	content, err := Floor0Extractor(ctx, reader, trans, res, testTranslateContext())
	if err != nil {
		t.Fatalf("Floor0Extractor: %v", err)
	}

	svcID := "cluster/main/k8s/namespaces/demo/services/web"
	depID := "cluster/main/k8s/namespaces/demo/deployments/web"
	ingID := "cluster/main/k8s/namespaces/demo/ingresses/web"

	var exposes, routes bool
	for idx := range content.Edges {
		edgeData := content.Edges[idx].Data
		if edgeData.Source == svcID && edgeData.Target == depID && edgeData.BGRelation == edge.BgRelationExposes {
			exposes = true
		}
		if edgeData.Source == ingID && edgeData.Target == svcID && edgeData.BGRelation == edge.BgRelationRoutes {
			routes = true
		}
	}
	if !exposes {
		t.Fatalf("missing Service->Deployment Exposes edge among %+v", content.Edges)
	}
	if !routes {
		t.Fatalf("missing Ingress->Service Routes edge among %+v", content.Edges)
	}
}
