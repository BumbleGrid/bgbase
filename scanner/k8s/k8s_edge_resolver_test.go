package k8s

import (
	"context"
	"strings"
	"testing"

	"github.com/BumbleGrid/bgbase/edge"
	"github.com/BumbleGrid/bgbase/node"
)

func testK8sStubForEdgeNodeID(id, label string) *node.K8sMetadata {
	const nsMarker = "/k8s/namespaces/"
	var k8sKind string
	if pos := strings.Index(id, nsMarker); pos >= 0 {
		tail := id[pos+len(nsMarker):]
		if !strings.Contains(tail, "/") {
			k8sKind = "Namespace"
		}
	}
	if k8sKind == "" {
		switch {
		case strings.Contains(id, "/deployments/"):
			k8sKind = "Deployment"
		case strings.Contains(id, "/statefulsets/"):
			k8sKind = "StatefulSet"
		case strings.Contains(id, "/daemonsets/"):
			k8sKind = "DaemonSet"
		case strings.Contains(id, "/replicasets/"):
			k8sKind = "ReplicaSet"
		case strings.Contains(id, "/cronjobs/"):
			k8sKind = "CronJob"
		case strings.Contains(id, "/jobs/"):
			k8sKind = "Job"
		case strings.Contains(id, "/ingresses/"):
			k8sKind = "Ingress"
		case strings.Contains(id, "/ingressclasses/"):
			k8sKind = "IngressClass"
		case strings.Contains(id, "/configmaps/"):
			k8sKind = "ConfigMap"
		case strings.Contains(id, "/secrets/"):
			k8sKind = "Secret"
		case strings.Contains(id, "/persistentvolumeclaims/"):
			k8sKind = "PersistentVolumeClaim"
		case strings.Contains(id, "/persistentvolumes/"):
			k8sKind = "PersistentVolume"
		case strings.Contains(id, "/networkpolicies/"):
			k8sKind = "NetworkPolicy"
		case strings.Contains(id, "/horizontalpodautoscalers/"):
			k8sKind = "HorizontalPodAutoscaler"
		default:
			k8sKind = "Service"
		}
	}
	km := &node.K8sMetadata{Kind: k8sKind, Name: label}
	if pos := strings.Index(id, nsMarker); pos >= 0 {
		after := id[pos+len(nsMarker):]
		slash := strings.IndexByte(after, '/')
		if slash > 0 {
			namespace := after[:slash]
			km.Namespace = &namespace
		}
	}
	if k8sKind == "PersistentVolume" || k8sKind == "IngressClass" || k8sKind == "Namespace" {
		km.Namespace = nil
	}
	return km
}

func testEdgeNode(id string, kind node.BgKind, label string, tags map[string]string) node.Data {
	meta := node.Meta{
		ExtractorVersion: "1.0",
	}
	if len(tags) > 0 {
		meta.Tags = tags
	}
	return node.Data{
		ID:            id,
		Label:         label,
		Floor:         0,
		BgKind:        kind,
		InfraProvider: node.InfraProviderKubernetes,
		K8s:           testK8sStubForEdgeNodeID(id, label),
		Meta:          &meta,
	}
}

func TestResolveEdgesEmptyInput(t *testing.T) {
	res := NewEdgeResolver()
	out, err := res.ResolveEdges(context.Background(), nil)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("want empty slice, got len=%d", len(out))
	}
	out, err = res.ResolveEdges(context.Background(), []node.Data{})
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("want empty slice, got len=%d", len(out))
	}
}

func TestResolveEdgesCancelledContext(t *testing.T) {
	res := NewEdgeResolver()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	nodes := []node.Data{
		testEdgeNode("c/k8s/namespaces/ns/services/a", node.BgKindServiceDiscovery, "a", nil),
	}
	_, err := res.ResolveEdges(ctx, nodes)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestResolveEdgesTagExplicitRoutes(t *testing.T) {
	res := NewEdgeResolver()
	svc := "c/k8s/namespaces/ns/services/api"
	ing := "c/k8s/namespaces/ns/ingresses/api"
	nodes := []node.Data{
		testEdgeNode(ing, node.BgKindGateway, "api", map[string]string{
			"k8s.edge.routes": svc,
		}),
		testEdgeNode(svc, node.BgKindServiceDiscovery, "api", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	got := out[0]
	if got.Source != ing || got.Target != svc {
		t.Fatalf("source/target: %+v", got)
	}
	if got.BGRelation != edge.BgRelationRoutes {
		t.Fatalf("relation %q", got.BGRelation)
	}
	if got.Inferred {
		t.Fatal("want explicit route (inferred false)")
	}
	if got.ExtractionSource != edge.ExtractionSourceK8sManifest {
		t.Fatalf("extractionSource %q", got.ExtractionSource)
	}
	if got.Meta.ExtractorVersion != "1.0" {
		t.Fatalf("meta %+v", got.Meta)
	}
}

func TestResolveEdgesTagInferredRoutes(t *testing.T) {
	res := NewEdgeResolver()
	svc := "c/k8s/namespaces/ns/services/x"
	ing := "c/k8s/namespaces/ns/ingresses/x"
	nodes := []node.Data{
		testEdgeNode(ing, node.BgKindGateway, "x", map[string]string{
			"k8s.edge.routes.inferred": svc,
		}),
		testEdgeNode(svc, node.BgKindServiceDiscovery, "x", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 1 || !out[0].Inferred {
		t.Fatalf("got %+v", out)
	}
}

func TestResolveEdgesTagMountsExplicitAndScheduledBy(t *testing.T) {
	res := NewEdgeResolver()
	pvc := "c/k8s/namespaces/ns/persistentvolumeclaims/data"
	pv := "c/k8s/persistentvolumes/pv1"
	job := "c/k8s/namespaces/ns/jobs/tick-abc"
	cron := "c/k8s/namespaces/ns/cronjobs/tick"
	nodes := []node.Data{
		testEdgeNode(pvc, node.BgKindStorage, "data", map[string]string{"k8s.edge.mounts": pv}),
		testEdgeNode(pv, node.BgKindStorage, "pv1", nil),
		testEdgeNode(job, node.BgKindJobRunner, "tick-abc", map[string]string{"k8s.edge.scheduled-by": cron}),
		testEdgeNode(cron, node.BgKindJobRunner, "tick", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d want 2: %+v", len(out), out)
	}
}

func TestResolveEdgesTagCallsInferred(t *testing.T) {
	res := NewEdgeResolver()
	a := "c/k8s/namespaces/ns/deployments/a"
	b := "c/k8s/namespaces/ns/deployments/b"
	nodes := []node.Data{
		testEdgeNode(a, node.BgKindWorkload, "a", map[string]string{"k8s.edge.calls.inferred": b}),
		testEdgeNode(b, node.BgKindWorkload, "b", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 1 || !out[0].Inferred || out[0].BGRelation != edge.BgRelationCalls {
		t.Fatalf("got %+v", out[0])
	}
}

func TestResolveEdgesTagSkipsUnknownPrefixAndUnknownRelation(t *testing.T) {
	res := NewEdgeResolver()
	svc := "c/k8s/namespaces/ns/services/s"
	nodes := []node.Data{
		testEdgeNode(svc, node.BgKindServiceDiscovery, "s", map[string]string{
			"other":            "value",
			"k8s.edge.unknown": svc,
			"k8s.edge.routes":  svc + "notfound",
		}),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("want no edges, got %+v", out)
	}
}

func TestResolveEdgesTagSkipsMissingTargetAndSelfLoop(t *testing.T) {
	res := NewEdgeResolver()
	self := "c/k8s/namespaces/ns/services/self"
	missing := "c/k8s/namespaces/ns/services/missing"
	nodes := []node.Data{
		testEdgeNode(self, node.BgKindServiceDiscovery, "self", map[string]string{
			"k8s.edge.routes":          self,
			"k8s.edge.routes.inferred": missing,
		}),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("got %+v", out)
	}
}

func TestResolveEdgesTagDedupesDuplicateTagLines(t *testing.T) {
	res := NewEdgeResolver()
	svc := "c/k8s/namespaces/ns/services/one"
	ing := "c/k8s/namespaces/ns/ingresses/one"
	nodes := []node.Data{
		testEdgeNode(ing, node.BgKindGateway, "one", map[string]string{"k8s.edge.routes": svc}),
		testEdgeNode(svc, node.BgKindServiceDiscovery, "one", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
}

func TestResolveEdgesHeuristicExposesSameNamespaceAndLabel(t *testing.T) {
	res := NewEdgeResolver()
	svc := "c/k8s/namespaces/ns/services/web"
	dep := "c/k8s/namespaces/ns/deployments/web"
	nodes := []node.Data{
		testEdgeNode(svc, node.BgKindServiceDiscovery, "web", nil),
		testEdgeNode(dep, node.BgKindWorkload, "web", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d %+v", len(out), out)
	}
	if out[0].BGRelation != edge.BgRelationExposes || !out[0].Inferred {
		t.Fatalf("got %+v", out[0])
	}
	if out[0].Source != svc || out[0].Target != dep {
		t.Fatalf("got %+v", out[0])
	}
}

func TestResolveEdgesHeuristicNoCrossNamespace(t *testing.T) {
	res := NewEdgeResolver()
	svc := "c/k8s/namespaces/a/services/web"
	dep := "c/k8s/namespaces/b/deployments/web"
	nodes := []node.Data{
		testEdgeNode(svc, node.BgKindServiceDiscovery, "web", nil),
		testEdgeNode(dep, node.BgKindWorkload, "web", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("want no edges, got %+v", out)
	}
}

func TestResolveEdgesHeuristicDifferentLabels(t *testing.T) {
	res := NewEdgeResolver()
	svc := "c/k8s/namespaces/ns/services/frontend"
	dep := "c/k8s/namespaces/ns/deployments/backend"
	nodes := []node.Data{
		testEdgeNode(svc, node.BgKindServiceDiscovery, "frontend", nil),
		testEdgeNode(dep, node.BgKindWorkload, "backend", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("got %+v", out)
	}
}

func TestResolveEdgesHeuristicIngressRoutesService(t *testing.T) {
	res := NewEdgeResolver()
	ing := "c/k8s/namespaces/ns/ingresses/api"
	svc := "c/k8s/namespaces/ns/services/api"
	nodes := []node.Data{
		testEdgeNode(ing, node.BgKindGateway, "api", nil),
		testEdgeNode(svc, node.BgKindLoadBalancer, "api", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 1 || out[0].BGRelation != edge.BgRelationRoutes {
		t.Fatalf("got %+v", out)
	}
	if out[0].Source != ing || out[0].Target != svc {
		t.Fatalf("got %+v", out)
	}
}

func TestResolveEdgesHeuristicJobScheduledByCronJob(t *testing.T) {
	res := NewEdgeResolver()
	job := "c/k8s/namespaces/ns/jobs/tick-7xq9"
	cron := "c/k8s/namespaces/ns/cronjobs/tick"
	nodes := []node.Data{
		testEdgeNode(job, node.BgKindJobRunner, "tick-7xq9", nil),
		testEdgeNode(cron, node.BgKindJobRunner, "tick", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d %+v", len(out), out)
	}
	if out[0].BGRelation != edge.BgRelationScheduledBy || !out[0].Inferred {
		t.Fatalf("got %+v", out[0])
	}
	if out[0].Source != job || out[0].Target != cron {
		t.Fatalf("want job->cron, got %+v", out[0])
	}
}

func TestResolveEdgesHeuristicJobNoCronPrefixMatch(t *testing.T) {
	res := NewEdgeResolver()
	job := "c/k8s/namespaces/ns/jobs/standalone"
	cron := "c/k8s/namespaces/ns/cronjobs/other"
	nodes := []node.Data{
		testEdgeNode(job, node.BgKindJobRunner, "standalone", nil),
		testEdgeNode(cron, node.BgKindJobRunner, "other", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("got %+v", out)
	}
}

func TestResolveEdgesHeuristicSkipsNonK8sPathIDs(t *testing.T) {
	res := NewEdgeResolver()
	nodes := []node.Data{
		{
			ID:            "legacy-service",
			Label:         "web",
			Floor:         0,
			BgKind:        node.BgKindServiceDiscovery,
			InfraProvider: node.InfraProviderKubernetes,
			Meta:          &node.Meta{},
		},
		{
			ID:            "legacy-dep",
			Label:         "web",
			Floor:         0,
			BgKind:        node.BgKindWorkload,
			InfraProvider: node.InfraProviderKubernetes,
			Meta:          &node.Meta{},
		},
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("want no heuristic edges %+v", out)
	}
}

func TestResolveEdgesHeuristicSkipsServiceWrongPath(t *testing.T) {
	res := NewEdgeResolver()
	svc := "c/k8s/namespaces/ns/configmaps/svc-named"
	dep := "c/k8s/namespaces/ns/deployments/svc-named"
	nodes := []node.Data{
		testEdgeNode(svc, node.BgKindServiceDiscovery, "svc-named", nil),
		testEdgeNode(dep, node.BgKindWorkload, "svc-named", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("service path is not /services/; got %+v", out)
	}
}

func TestResolveEdgesDeterministicSortByID(t *testing.T) {
	res := NewEdgeResolver()
	svcA := "c/k8s/namespaces/ns/services/a"
	svcB := "c/k8s/namespaces/ns/services/b"
	ingSecond := "c/k8s/namespaces/ns/ingresses/second"
	ingFirst := "c/k8s/namespaces/ns/ingresses/first"
	nodes := []node.Data{
		testEdgeNode(ingSecond, node.BgKindGateway, "second", map[string]string{"k8s.edge.routes": svcB}),
		testEdgeNode(ingFirst, node.BgKindGateway, "first", map[string]string{"k8s.edge.routes": svcA}),
		testEdgeNode(svcA, node.BgKindServiceDiscovery, "a", nil),
		testEdgeNode(svcB, node.BgKindServiceDiscovery, "b", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d %+v", len(out), out)
	}
	if out[0].ID > out[1].ID {
		t.Fatalf("not sorted: %q before %q", out[0].ID, out[1].ID)
	}
}

func TestResolveEdgesHeuristicAndTagSameEdgeDeduped(t *testing.T) {
	res := NewEdgeResolver()
	svc := "c/k8s/namespaces/ns/services/web"
	dep := "c/k8s/namespaces/ns/deployments/web"
	nodes := []node.Data{
		testEdgeNode(svc, node.BgKindServiceDiscovery, "web", map[string]string{
			"k8s.edge.exposes.inferred": dep,
		}),
		testEdgeNode(dep, node.BgKindWorkload, "web", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("tag + heuristic should dedupe, len=%d %+v", len(out), out)
	}
}

func TestResolveEdgesMetaNilNoPanic(t *testing.T) {
	res := NewEdgeResolver()
	nodes := []node.Data{
		{
			ID:            "c/k8s/namespaces/ns/services/x",
			Label:         "x",
			Floor:         0,
			BgKind:        node.BgKindServiceDiscovery,
			InfraProvider: node.InfraProviderKubernetes,
			Meta:          nil,
		},
	}
	_, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
}

func TestK8sRestPathFromNodeID(t *testing.T) {
	got, ok := k8sRestPathFromNodeID("cluster/k8s/namespaces/ns/deployments/web")
	if !ok || got != "namespaces/ns/deployments/web" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	_, ok = k8sRestPathFromNodeID("no-k8s-marker")
	if ok {
		t.Fatal("expected false")
	}
}

func TestNamespaceFromNamespacedRest(t *testing.T) {
	ns, ok := namespaceFromNamespacedRest("namespaces/prod/services/api")
	if !ok || ns != "prod" {
		t.Fatalf("got %q ok=%v", ns, ok)
	}
	_, ok = namespaceFromNamespacedRest("persistentvolumes/pv1")
	if ok {
		t.Fatal("expected false for cluster-scoped rest path")
	}
}

func TestParseEdgeTagKey(t *testing.T) {
	rel, inf, ok := parseEdgeTagKey("k8s.edge.routes")
	if !ok || rel != edge.BgRelationRoutes || inf {
		t.Fatalf("routes explicit: rel=%q inf=%v ok=%v", rel, inf, ok)
	}
	rel, inf, ok = parseEdgeTagKey("k8s.edge.routes.inferred")
	if !ok || rel != edge.BgRelationRoutes || !inf {
		t.Fatalf("routes inferred: rel=%q inf=%v ok=%v", rel, inf, ok)
	}
	_, _, ok = parseEdgeTagKey("k8s.edge.nope")
	if ok {
		t.Fatal("unknown tail")
	}
	_, _, ok = parseEdgeTagKey("other.prefix")
	if ok {
		t.Fatal("wrong prefix")
	}
}

func TestWorkloadRestPath(t *testing.T) {
	if !workloadRestPath("namespaces/ns/deployments/web") {
		t.Fatal("deployments")
	}
	if workloadRestPath("namespaces/ns/horizontalpodautoscalers/web") {
		t.Fatal("hpa should not count as workload path heuristic segment")
	}
}

func TestResolveEdgesFloorsCopiedFromSource(t *testing.T) {
	res := NewEdgeResolver()
	svc := "c/k8s/namespaces/ns/services/s"
	dep := "c/k8s/namespaces/ns/deployments/s"
	s := testEdgeNode(svc, node.BgKindServiceDiscovery, "s", nil)
	s.Floor = 3
	d := testEdgeNode(dep, node.BgKindWorkload, "s", nil)
	d.Floor = 0
	out, err := res.ResolveEdges(context.Background(), []node.Data{s, d})
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 1 || out[0].Floor != 3 {
		t.Fatalf("got %+v", out[0])
	}
}

func TestResolveEdgesTagExplicitVsInferredSamePairProducesTwoEdges(t *testing.T) {
	res := NewEdgeResolver()
	a := "c/k8s/namespaces/ns/deployments/a"
	b := "c/k8s/namespaces/ns/deployments/b"
	nodes := []node.Data{
		testEdgeNode(a, node.BgKindWorkload, "a", map[string]string{
			"k8s.edge.calls":          b,
			"k8s.edge.calls.inferred": b,
		}),
		testEdgeNode(b, node.BgKindWorkload, "b", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("want two edges (differing inferred), len=%d %+v", len(out), out)
	}
	seenFalse, seenTrue := false, false
	for idx := range out {
		if out[idx].Inferred {
			seenTrue = true
		} else {
			seenFalse = true
		}
	}
	if !seenFalse || !seenTrue {
		t.Fatalf("inferred flags: %+v", out)
	}
}

func TestResolveEdgesEdgeIDContainsRelationForUniqueness(t *testing.T) {
	res := NewEdgeResolver()
	src := "c/k8s/namespaces/ns/services/s"
	tgt := "c/k8s/namespaces/ns/deployments/d"
	nodes := []node.Data{
		testEdgeNode(src, node.BgKindServiceDiscovery, "s", map[string]string{
			"k8s.edge.exposes": tgt,
			"k8s.edge.calls":   tgt,
		}),
		testEdgeNode(tgt, node.BgKindWorkload, "d", nil),
	}
	out, err := res.ResolveEdges(context.Background(), nodes)
	if err != nil {
		t.Fatalf("ResolveEdges: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	ids := make(map[string]struct{})
	for idx := range out {
		if _, dup := ids[out[idx].ID]; dup {
			t.Fatalf("duplicate edge id %q", out[idx].ID)
		}
		ids[out[idx].ID] = struct{}{}
		if !strings.Contains(out[idx].ID, string(out[idx].BGRelation)) {
			t.Fatalf("id %q should encode relation", out[idx].ID)
		}
	}
}
