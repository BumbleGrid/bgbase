package k8s

import (
	"context"
	"testing"

	"github.com/BumbleGrid/bgbase/node"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testK8sTranslateContext() K8sTranslateContext {
	return K8sTranslateContext{
		Floor:                 0,
		Meta:                  node.Meta{ExtractorVersion: "0.9.0"},
		ClusterNodeID:         "cluster/main",
		NamespaceName:         "prod",
		NamespaceParentNodeID: "cluster/main/k8s/namespaces/prod",
	}
}

func TestTranslateNamespaces(t *testing.T) {
	trans := NewNodeTranslator()
	tctx := testK8sTranslateContext()
	wantParent := tctx.ClusterNodeID
	nodes, err := trans.TranslateNamespaces(context.Background(), tctx, []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "prod"}},
	})
	if err != nil {
		t.Fatalf("TranslateNamespaces: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("len(nodes) = %d, want 1", len(nodes))
	}
	got := nodes[0]
	if got.ID != "cluster/main/k8s/namespaces/prod" {
		t.Errorf("ID = %q", got.ID)
	}
	if got.Label != "prod" {
		t.Errorf("Label = %q", got.Label)
	}
	if got.BgKind != node.BgKindNamespace {
		t.Errorf("BgKind = %q", got.BgKind)
	}
	if got.Parent == nil || *got.Parent != wantParent {
		t.Errorf("Parent = %v, want %q", got.Parent, wantParent)
	}
	if got.Floor != 0 {
		t.Errorf("Floor = %d", got.Floor)
	}
	if got.InfraProvider != node.InfraProviderKubernetes {
		t.Errorf("InfraProvider = %q", got.InfraProvider)
	}
	if got.Meta == nil || got.Meta.ExtractorVersion != "0.9.0" {
		t.Errorf("Meta = %+v", got.Meta)
	}
}

func TestTranslateDeploymentsUsesNamespaceParent(t *testing.T) {
	trans := NewNodeTranslator()
	tctx := testK8sTranslateContext()
	nodes, err := trans.TranslateDeployments(context.Background(), tctx, []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "web"},
		},
	})
	if err != nil {
		t.Fatalf("TranslateDeployments: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("len(nodes) = %d", len(nodes))
	}
	got := nodes[0]
	if got.ID != "cluster/main/k8s/namespaces/prod/deployments/web" {
		t.Errorf("ID = %q", got.ID)
	}
	if got.Parent == nil || *got.Parent != tctx.NamespaceParentNodeID {
		t.Errorf("Parent = %v", got.Parent)
	}
	if got.BgKind != node.BgKindWorkload {
		t.Errorf("BgKind = %q", got.BgKind)
	}
}

func TestTranslatePersistentVolumesClusterParent(t *testing.T) {
	trans := NewNodeTranslator()
	tctx := testK8sTranslateContext()
	nodes, err := trans.TranslatePersistentVolumes(context.Background(), tctx, []corev1.PersistentVolume{
		{ObjectMeta: metav1.ObjectMeta{Name: "pv1"}},
	})
	if err != nil {
		t.Fatalf("TranslatePersistentVolumes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("len(nodes) = %d", len(nodes))
	}
	got := nodes[0]
	if got.ID != "cluster/main/k8s/persistentvolumes/pv1" {
		t.Errorf("ID = %q", got.ID)
	}
	if got.Parent == nil || *got.Parent != tctx.ClusterNodeID {
		t.Errorf("Parent = %v", got.Parent)
	}
	if got.BgKind != node.BgKindStorage {
		t.Errorf("BgKind = %q", got.BgKind)
	}
}

func TestTranslateServicesBgKindByType(t *testing.T) {
	trans := NewNodeTranslator()
	tctx := testK8sTranslateContext()
	baseMeta := metav1.ObjectMeta{Namespace: "prod", Name: "api"}
	tests := []struct {
		name     string
		svc      corev1.Service
		wantKind node.BgKind
	}{
		{
			name: "ClusterIP",
			svc: corev1.Service{
				ObjectMeta: baseMeta,
				Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP},
			},
			wantKind: node.BgKindServiceDiscovery,
		},
		{
			name: "LoadBalancer",
			svc: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "lb"},
				Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer},
			},
			wantKind: node.BgKindLoadBalancer,
		},
		{
			name: "ExternalName",
			svc: corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "ext"},
				Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeExternalName, ExternalName: "example.com"},
			},
			wantKind: node.BgKindExternalService,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := trans.TranslateServices(context.Background(), tctx, []corev1.Service{tt.svc})
			if err != nil {
				t.Fatalf("TranslateServices: %v", err)
			}
			if len(nodes) != 1 {
				t.Fatalf("len(nodes) = %d", len(nodes))
			}
			if nodes[0].BgKind != tt.wantKind {
				t.Errorf("BgKind = %q, want %q", nodes[0].BgKind, tt.wantKind)
			}
		})
	}
}

func TestTranslateEmptyInput(t *testing.T) {
	trans := NewNodeTranslator()
	tctx := testK8sTranslateContext()
	nodes, err := trans.TranslateConfigMaps(context.Background(), tctx, nil)
	if err != nil {
		t.Fatalf("TranslateConfigMaps: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("len = %d", len(nodes))
	}
}

func TestTranslateCancelledContext(t *testing.T) {
	trans := NewNodeTranslator()
	tctx := testK8sTranslateContext()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := trans.TranslateSecrets(ctx, tctx, []corev1.Secret{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "db"}},
	})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestTranslateKindsCoverage(t *testing.T) {
	trans := NewNodeTranslator()
	tctx := testK8sTranslateContext()
	ctx := context.Background()

	check := func(name string, got []node.Data, err error, wantID string, wantKind node.BgKind) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(got) != 1 {
			t.Fatalf("%s: len = %d", name, len(got))
		}
		if got[0].ID != wantID {
			t.Errorf("%s: ID = %q, want %q", name, got[0].ID, wantID)
		}
		if got[0].BgKind != wantKind {
			t.Errorf("%s: BgKind = %q, want %q", name, got[0].BgKind, wantKind)
		}
	}

	got, err := trans.TranslateStatefulSets(ctx, tctx, []appsv1.StatefulSet{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "db"}},
	})
	check("StatefulSet", got, err, "cluster/main/k8s/namespaces/prod/statefulsets/db", node.BgKindWorkload)

	got, err = trans.TranslateDaemonSets(ctx, tctx, []appsv1.DaemonSet{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "agent"}},
	})
	check("DaemonSet", got, err, "cluster/main/k8s/namespaces/prod/daemonsets/agent", node.BgKindWorkload)

	got, err = trans.TranslateReplicaSets(ctx, tctx, []appsv1.ReplicaSet{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "rs-1"}},
	})
	check("ReplicaSet", got, err, "cluster/main/k8s/namespaces/prod/replicasets/rs-1", node.BgKindWorkload)

	got, err = trans.TranslateCronJobs(ctx, tctx, []batchv1.CronJob{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "tick"}},
	})
	check("CronJob", got, err, "cluster/main/k8s/namespaces/prod/cronjobs/tick", node.BgKindJobRunner)

	got, err = trans.TranslateJobs(ctx, tctx, []batchv1.Job{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "once"}},
	})
	check("Job", got, err, "cluster/main/k8s/namespaces/prod/jobs/once", node.BgKindJobRunner)

	got, err = trans.TranslateIngresses(ctx, tctx, []networkingv1.Ingress{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "public"}},
	})
	check("Ingress", got, err, "cluster/main/k8s/namespaces/prod/ingresses/public", node.BgKindGateway)

	got, err = trans.TranslateIngressClasses(ctx, tctx, []networkingv1.IngressClass{
		{ObjectMeta: metav1.ObjectMeta{Name: "nginx"}},
	})
	check("IngressClass", got, err, "cluster/main/k8s/ingressclasses/nginx", node.BgKindGateway)

	got, err = trans.TranslateConfigMaps(ctx, tctx, []corev1.ConfigMap{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "app"}},
	})
	check("ConfigMap", got, err, "cluster/main/k8s/namespaces/prod/configmaps/app", node.BgKindConfigSource)

	got, err = trans.TranslatePersistentVolumeClaims(ctx, tctx, []corev1.PersistentVolumeClaim{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "data"}},
	})
	check("PVC", got, err, "cluster/main/k8s/namespaces/prod/persistentvolumeclaims/data", node.BgKindStorage)

	got, err = trans.TranslateNetworkPolicies(ctx, tctx, []networkingv1.NetworkPolicy{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "deny"}},
	})
	check("NetworkPolicy", got, err, "cluster/main/k8s/namespaces/prod/networkpolicies/deny", node.BgKindNetworkPolicy)

	got, err = trans.TranslateHorizontalPodAutoscalersV2(ctx, tctx, []autoscalingv2.HorizontalPodAutoscaler{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "web"}},
	})
	check("HPA", got, err, "cluster/main/k8s/namespaces/prod/horizontalpodautoscalers/web", node.BgKindWorkload)
}

func TestK8sRESTID(t *testing.T) {
	if got := k8sRESTID("c", "namespaces/x"); got != "c/k8s/namespaces/x" {
		t.Errorf("got %q", got)
	}
}

func TestTranslateCronJobObjectLabelsBecomeMetaTags(t *testing.T) {
	trans := NewNodeTranslator()
	tctx := testK8sTranslateContext()
	nodes, err := trans.TranslateCronJobs(context.Background(), tctx, []batchv1.CronJob{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "prod",
				Name:      "tick",
				Labels: map[string]string{
					"env":      "test",
					"domain":   "billing",
					"scenario": "complex",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("TranslateCronJobs: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("len(nodes) = %d", len(nodes))
	}
	got := nodes[0]
	if got.Meta == nil {
		t.Fatal("Meta = nil")
	}
	if got.Meta.Tags["env"] != "test" || got.Meta.Tags["domain"] != "billing" {
		t.Errorf("Tags = %#v", got.Meta.Tags)
	}
	if got.Meta.ExtractorVersion != "0.9.0" {
		t.Errorf("ExtractorVersion = %q", got.Meta.ExtractorVersion)
	}
}

func TestTranslateMergesContextMetaTagsWithObjectLabels(t *testing.T) {
	trans := NewNodeTranslator()
	tctx := testK8sTranslateContext()
	tctx.Meta.Tags = map[string]string{
		"from-context": "keep",
		"overlap":      "ctx",
	}
	nodes, err := trans.TranslateDeployments(context.Background(), tctx, []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "prod",
				Name:      "web",
				Labels: map[string]string{
					"overlap": "from-object",
					"app":     "web",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("TranslateDeployments: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("len(nodes) = %d", len(nodes))
	}
	tags := nodes[0].Meta.Tags
	if tags["from-context"] != "keep" {
		t.Errorf("from-context = %q", tags["from-context"])
	}
	if tags["overlap"] != "from-object" {
		t.Errorf("overlap = %q, want object label to win", tags["overlap"])
	}
	if tags["app"] != "web" {
		t.Errorf("app = %q", tags["app"])
	}
}
