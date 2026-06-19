package k8s

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/BumbleGrid/bgbase/edge"
	"github.com/BumbleGrid/bgbase/node"
)

type fakeIstioLister struct {
	present          bool
	virtualServices  map[string][]IstioVirtualService
	destinationRules map[string][]IstioDestinationRule
	serviceEntries   map[string][]IstioServiceEntry
}

func (fake *fakeIstioLister) Present(ctx context.Context) bool {
	if fake == nil {
		return false
	}
	return fake.present
}

func (fake *fakeIstioLister) ListVirtualServices(ctx context.Context, namespace string) ([]IstioVirtualService, error) {
	if fake == nil || fake.virtualServices == nil {
		return nil, nil
	}
	return fake.virtualServices[namespace], nil
}

func (fake *fakeIstioLister) ListDestinationRules(ctx context.Context, namespace string) ([]IstioDestinationRule, error) {
	if fake == nil || fake.destinationRules == nil {
		return nil, nil
	}
	return fake.destinationRules[namespace], nil
}

func (fake *fakeIstioLister) ListServiceEntries(ctx context.Context, namespace string) ([]IstioServiceEntry, error) {
	if fake == nil || fake.serviceEntries == nil {
		return nil, nil
	}
	return fake.serviceEntries[namespace], nil
}

func testServiceNode(cluster, namespace, name string) node.Data {
	id := cluster + "/k8s/namespaces/" + namespace + "/services/" + name
	return testEdgeNode(id, node.BgKindServiceDiscovery, name, nil)
}

func testWorkloadNode(cluster, namespace, name string) node.Data {
	id := cluster + "/k8s/namespaces/" + namespace + "/deployments/" + name
	return testEdgeNode(id, node.BgKindWorkload, name, nil)
}

func testExternalServiceNode(cluster, namespace, host string) node.Data {
	id := cluster + "/k8s/namespaces/" + namespace + "/services/" + host
	return testEdgeNode(id, node.BgKindExternalService, host, nil)
}

func TestIsIstioPresent(t *testing.T) {
	if IsIstioPresent(context.Background(), nil) {
		t.Fatal("nil discovery client should return false")
	}
}

func TestResolveIstioEdgesAbsent(t *testing.T) {
	lister := &fakeIstioLister{present: false}
	res := NewEdgeResolver(EdgeResolverWithIstioLister(lister))
	nodes := []node.Data{
		testServiceNode("cluster/main", "apps", "payments"),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	for idx := range out {
		if out[idx].ExtractionSource == edge.ExtractionSourceIstioManifest {
			t.Fatalf("unexpected istio edge: %+v", out[idx])
		}
	}
}

func TestResolveIstioEdgesVirtualServiceSingleDestination(t *testing.T) {
	cluster := "cluster/main"
	namespace := "apps"
	payments := testServiceNode(cluster, namespace, "payments")
	identity := testServiceNode(cluster, namespace, "identity")
	lister := &fakeIstioLister{
		present: true,
		virtualServices: map[string][]IstioVirtualService{
			namespace: {{
				Namespace: namespace,
				Name:      "payments-route",
				Hosts:     []string{"payments"},
				HTTP: []IstioHTTPRoute{{
					MatchPrefix:  strPtr("/api"),
					Destinations: []string{"identity"},
				}},
			}},
		},
	}
	res := NewEdgeResolver(EdgeResolverWithIstioLister(lister))
	out, err := res.ResolveEdges(context.Background(), []node.Data{payments, identity})
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	var istioEdges []edge.Data
	for idx := range out {
		if out[idx].ExtractionSource == edge.ExtractionSourceIstioManifest {
			istioEdges = append(istioEdges, out[idx])
		}
	}
	if len(istioEdges) != 1 {
		t.Fatalf("istio edge count = %d, want 1; all edges: %+v", len(istioEdges), out)
	}
	got := istioEdges[0]
	if got.Source != payments.ID || got.Target != identity.ID {
		t.Fatalf("source/target: %+v", got)
	}
	if got.BGRelation != edge.BgRelationRoutes {
		t.Fatalf("relation %q", got.BGRelation)
	}
	if got.Inferred {
		t.Fatal("want inferred false")
	}
	if got.Label == nil || *got.Label != "/api" {
		t.Fatalf("label = %v, want /api", got.Label)
	}
	if !strings.Contains(got.Meta.Description, "Istio VirtualService: payments-route") {
		t.Fatalf("description %q", got.Meta.Description)
	}
	wantID := payments.ID + "--routes--" + identity.ID
	if got.ID != wantID {
		t.Fatalf("id %q, want %q", got.ID, wantID)
	}
}

func TestResolveIstioEdgesVirtualServiceWeightedRoutes(t *testing.T) {
	cluster := "cluster/main"
	namespace := "apps"
	catalog := testServiceNode(cluster, namespace, "catalog")
	v1 := testServiceNode(cluster, namespace, "catalog-v1")
	v2 := testServiceNode(cluster, namespace, "catalog-v2")
	lister := &fakeIstioLister{
		present: true,
		virtualServices: map[string][]IstioVirtualService{
			namespace: {{
				Namespace: namespace,
				Name:      "catalog-canary",
				Hosts:     []string{"catalog"},
				HTTP: []IstioHTTPRoute{{
					Destinations: []string{"catalog-v1", "catalog-v2"},
				}},
			}},
		},
	}
	res := NewEdgeResolver(EdgeResolverWithIstioLister(lister))
	out, err := res.ResolveEdges(context.Background(), []node.Data{catalog, v1, v2})
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	routeTargets := map[string]struct{}{}
	for idx := range out {
		item := out[idx]
		if item.ExtractionSource != edge.ExtractionSourceIstioManifest || item.BGRelation != edge.BgRelationRoutes {
			continue
		}
		if item.Source != catalog.ID {
			t.Fatalf("unexpected source %q", item.Source)
		}
		routeTargets[item.Target] = struct{}{}
	}
	if len(routeTargets) != 2 {
		t.Fatalf("route target count = %d, want 2", len(routeTargets))
	}
	if _, ok := routeTargets[v1.ID]; !ok {
		t.Fatalf("missing route to v1")
	}
	if _, ok := routeTargets[v2.ID]; !ok {
		t.Fatalf("missing route to v2")
	}
}

func TestResolveIstioEdgesServiceEntryNoMatchingNode(t *testing.T) {
	cluster := "cluster/main"
	namespace := "apps"
	payments := testServiceNode(cluster, namespace, "payments")
	identity := testServiceNode(cluster, namespace, "identity")
	var warnings []string
	lister := &fakeIstioLister{
		present: true,
		virtualServices: map[string][]IstioVirtualService{
			namespace: {{
				Namespace: namespace,
				Name:      "payments-route",
				Hosts:     []string{"payments"},
				HTTP: []IstioHTTPRoute{{
					Destinations: []string{"api.external.com"},
				}},
			}},
		},
		serviceEntries: map[string][]IstioServiceEntry{
			namespace: {{
				Namespace: namespace,
				Name:      "external-api",
				Hosts:     []string{"api.external.com"},
			}},
		},
	}
	res := NewEdgeResolver(
		EdgeResolverWithIstioLister(lister),
		EdgeResolverWithLogger(nil, func(format string, args ...any) {
			warnings = append(warnings, fmt.Sprintf(format, args...))
		}),
	)
	out, err := res.ResolveEdges(context.Background(), []node.Data{payments, identity})
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	for idx := range out {
		if out[idx].BGRelation == edge.BgRelationCalls && out[idx].ExtractionSource == edge.ExtractionSourceIstioManifest {
			t.Fatalf("unexpected calls edge: %+v", out[idx])
		}
	}
	if len(warnings) == 0 {
		t.Fatal("expected warning log")
	}
}

func TestResolveIstioEdgesSkipsDuplicateK8sRoute(t *testing.T) {
	cluster := "cluster/main"
	namespace := "apps"
	payments := testServiceNode(cluster, namespace, "payments")
	identity := testServiceNode(cluster, namespace, "identity")
	nodes := []node.Data{
		testEdgeNode(payments.ID, node.BgKindServiceDiscovery, "payments", map[string]string{
			"k8s.edge.routes": identity.ID,
		}),
		identity,
	}
	lister := &fakeIstioLister{
		present: true,
		virtualServices: map[string][]IstioVirtualService{
			namespace: {{
				Namespace: namespace,
				Name:      "payments-route",
				Hosts:     []string{"payments"},
				HTTP: []IstioHTTPRoute{{
					Destinations: []string{"identity"},
				}},
			}},
		},
	}
	res := NewEdgeResolver(EdgeResolverWithIstioLister(lister))
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	routeCount := 0
	for idx := range out {
		if out[idx].BGRelation != edge.BgRelationRoutes {
			continue
		}
		if out[idx].Source != payments.ID || out[idx].Target != identity.ID {
			continue
		}
		routeCount++
	}
	if routeCount != 1 {
		t.Fatalf("route edge count = %d, want 1 (duplicate istio route skipped)", routeCount)
	}
	if out[0].ExtractionSource != edge.ExtractionSourceK8sManifest {
		t.Fatalf("expected k8s-manifest edge, got %q", out[0].ExtractionSource)
	}
}

func strPtr(value string) *string {
	return &value
}
