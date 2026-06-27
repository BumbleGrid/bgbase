package k8s

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestParseScanFilter_invalidWorkloadPattern(t *testing.T) {
	_, err := ParseScanFilter(nil, []string{"bad-pattern"})
	if err == nil {
		t.Fatal("expected error for invalid pattern")
	}
	_, err = ParseScanFilter(nil, []string{"demo/pods/web"})
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

func TestScanFilter_denyNamespace(t *testing.T) {
	ctx := context.Background()
	kubeSystem := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}}
	appsNS := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "apps"}}
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "apps"},
	}
	cs := fake.NewSimpleClientset(kubeSystem, appsNS, deploy)
	reader := NewReaderWithClientset(cs)

	filter, err := ParseScanFilter([]string{"kube-system"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	lister := NewListerWithScanFilter(reader, filter)

	namespaces, err := lister.ListNamespaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(namespaces) != 1 || namespaces[0].Name != "apps" {
		t.Fatalf("ListNamespaces = %#v", namespaces)
	}

	deployments, err := lister.ListDeployments(ctx, "apps")
	if err != nil {
		t.Fatal(err)
	}
	if len(deployments) != 1 || deployments[0].Name != "web" {
		t.Fatalf("ListDeployments = %#v", deployments)
	}
}

func TestScanFilter_denyWorkloadPatterns(t *testing.T) {
	ctx := context.Background()
	nsName := "demo"
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
	keep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: nsName}}
	dropGlob := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "web-front", Namespace: nsName}}
	dropExact := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "noise", Namespace: nsName}}
	svcKeep := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: nsName}}
	svcDrop := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "kube-dns", Namespace: nsName}}
	cs := fake.NewSimpleClientset(ns, keep, dropGlob, dropExact, svcKeep, svcDrop)
	reader := NewReaderWithClientset(cs)

	filter, err := ParseScanFilter(nil, []string{
		nsName + "/deployments/web*",
		"*/services/kube-dns",
	})
	if err != nil {
		t.Fatal(err)
	}
	lister := NewListerWithScanFilter(reader, filter)

	deployments, err := lister.ListDeployments(ctx, nsName)
	if err != nil {
		t.Fatal(err)
	}
	names := deploymentNames(deployments)
	if len(names) != 2 {
		t.Fatalf("deployments = %v, want 2 kept", names)
	}
	for _, want := range []string{"api", "noise"} {
		if !containsString(names, want) {
			t.Fatalf("deployments = %v, missing %q", names, want)
		}
	}
	if containsString(names, "web-front") {
		t.Fatalf("deployments = %v, web-front should be filtered", names)
	}

	services, err := lister.ListServices(ctx, nsName)
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 1 || services[0].Name != "api" {
		t.Fatalf("services = %#v", services)
	}
}

func TestScanFilter_workloadPatternAnyKind(t *testing.T) {
	ctx := context.Background()
	nsName := "payments"
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
	dropDeploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "eppo-main", Namespace: nsName}}
	keepDeploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "checkout", Namespace: nsName}}
	dropSvc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "eppo-api", Namespace: nsName}}
	cs := fake.NewSimpleClientset(ns, dropDeploy, keepDeploy, dropSvc)
	reader := NewReaderWithClientset(cs)

	filter, err := ParseScanFilter(nil, []string{nsName + "/*/eppo-*"})
	if err != nil {
		t.Fatal(err)
	}
	lister := NewListerWithScanFilter(reader, filter)

	deployments, err := lister.ListDeployments(ctx, nsName)
	if err != nil {
		t.Fatal(err)
	}
	if len(deployments) != 1 || deployments[0].Name != "checkout" {
		t.Fatalf("deployments = %#v", deployments)
	}

	services, err := lister.ListServices(ctx, nsName)
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 0 {
		t.Fatalf("services = %#v, want empty", services)
	}
}

func TestNewListerWithScanFilter_emptyFilterReturnsBase(t *testing.T) {
	reader := NewReaderWithClientset(fake.NewSimpleClientset())
	lister := NewListerWithScanFilter(reader, ScanFilter{})
	if lister != reader {
		t.Fatal("expected base lister unchanged for empty filter")
	}
}

func deploymentNames(items []appsv1.Deployment) []string {
	out := make([]string, 0, len(items))
	for idx := range items {
		out = append(out, items[idx].Name)
	}
	return out
}

func containsString(values []string, target string) bool {
	for idx := range values {
		if values[idx] == target {
			return true
		}
	}
	return false
}
