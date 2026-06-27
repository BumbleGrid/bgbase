package k8s

import (
	"context"
	"fmt"
	"path"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

var floor0KindSegments = map[string]struct{}{
	"deployments":  {},
	"statefulsets": {},
	"daemonsets":   {},
	"cronjobs":     {},
	"jobs":         {},
	"services":     {},
	"ingresses":    {},
}

type ScanFilter struct {
	denyNamespaces   map[string]struct{}
	workloadPatterns []workloadDenyPattern
}

type workloadDenyPattern struct {
	namespace string
	kind      string
	name      string
}

func ParseScanFilter(denyNamespaces []string, denyWorkloadPatterns []string) (ScanFilter, error) {
	filter := ScanFilter{}
	if len(denyNamespaces) > 0 {
		filter.denyNamespaces = make(map[string]struct{}, len(denyNamespaces))
		for idx := range denyNamespaces {
			name := denyNamespaces[idx]
			if name == "" {
				return ScanFilter{}, fmt.Errorf("scan filter: deny namespace entry must not be empty")
			}
			filter.denyNamespaces[name] = struct{}{}
		}
	}
	if len(denyWorkloadPatterns) > 0 {
		patterns := make([]workloadDenyPattern, 0, len(denyWorkloadPatterns))
		for idx := range denyWorkloadPatterns {
			parsed, err := parseWorkloadDenyPattern(denyWorkloadPatterns[idx])
			if err != nil {
				return ScanFilter{}, err
			}
			patterns = append(patterns, parsed)
		}
		filter.workloadPatterns = patterns
	}
	return filter, nil
}

func parseWorkloadDenyPattern(raw string) (workloadDenyPattern, error) {
	segments := strings.Split(raw, "/")
	if len(segments) != 3 {
		return workloadDenyPattern{}, fmt.Errorf("scan filter: workload pattern %q must be namespace/kind/name", raw)
	}
	for segIdx, segment := range segments {
		if segment == "" {
			return workloadDenyPattern{}, fmt.Errorf("scan filter: workload pattern %q has empty segment", raw)
		}
		if err := validateGlobSegment(segment); err != nil {
			return workloadDenyPattern{}, fmt.Errorf("scan filter: workload pattern %q segment %d: %w", raw, segIdx+1, err)
		}
	}
	kind := segments[1]
	if kind != "*" && !hasGlobMeta(kind) {
		if _, ok := floor0KindSegments[kind]; !ok {
			return workloadDenyPattern{}, fmt.Errorf("scan filter: workload pattern %q kind %q must be one of deployments, statefulsets, daemonsets, cronjobs, jobs, services, ingresses, or *", raw, kind)
		}
	}
	return workloadDenyPattern{
		namespace: segments[0],
		kind:      kind,
		name:      segments[2],
	}, nil
}

func validateGlobSegment(segment string) error {
	if _, err := path.Match(segment, "x"); err != nil {
		return fmt.Errorf("invalid glob %q: %w", segment, err)
	}
	return nil
}

func hasGlobMeta(segment string) bool {
	return strings.ContainsAny(segment, "*?[")
}

func (pattern workloadDenyPattern) matches(namespace, kind, name string) bool {
	if !globMatch(pattern.namespace, namespace) {
		return false
	}
	if !globMatch(pattern.kind, kind) {
		return false
	}
	return globMatch(pattern.name, name)
}

func globMatch(pattern, value string) bool {
	ok, err := path.Match(pattern, value)
	return err == nil && ok
}

func (filter ScanFilter) namespaceDenied(name string) bool {
	if len(filter.denyNamespaces) == 0 {
		return false
	}
	_, ok := filter.denyNamespaces[name]
	return ok
}

func (filter ScanFilter) workloadDenied(namespace, kind, name string) bool {
	for idx := range filter.workloadPatterns {
		if filter.workloadPatterns[idx].matches(namespace, kind, name) {
			return true
		}
	}
	return false
}

func (filter ScanFilter) empty() bool {
	return len(filter.denyNamespaces) == 0 && len(filter.workloadPatterns) == 0
}

type scanFilterLister struct {
	inner  K8sLister
	filter ScanFilter
}

func NewListerWithScanFilter(base K8sLister, filter ScanFilter) K8sLister {
	if base == nil || filter.empty() {
		return base
	}
	return &scanFilterLister{inner: base, filter: filter}
}

func (filtered *scanFilterLister) ListNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	all, err := filtered.inner.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	if len(filtered.filter.denyNamespaces) == 0 {
		return all, nil
	}
	out := make([]corev1.Namespace, 0, len(all))
	for idx := range all {
		if filtered.filter.namespaceDenied(all[idx].Name) {
			continue
		}
		out = append(out, all[idx])
	}
	return out, nil
}

func (filtered *scanFilterLister) ListPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	return filtered.inner.ListPersistentVolumes(ctx)
}

func (filtered *scanFilterLister) ListIngressClasses(ctx context.Context) ([]networkingv1.IngressClass, error) {
	return filtered.inner.ListIngressClasses(ctx)
}

func (filtered *scanFilterLister) ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	items, err := filtered.inner.ListDeployments(ctx, namespace)
	if err != nil {
		return nil, err
	}
	return filterNamedItems(items, namespace, "deployments", filtered.filter, func(item appsv1.Deployment) string {
		return item.Name
	}), nil
}

func (filtered *scanFilterLister) ListStatefulSets(ctx context.Context, namespace string) ([]appsv1.StatefulSet, error) {
	items, err := filtered.inner.ListStatefulSets(ctx, namespace)
	if err != nil {
		return nil, err
	}
	return filterNamedItems(items, namespace, "statefulsets", filtered.filter, func(item appsv1.StatefulSet) string {
		return item.Name
	}), nil
}

func (filtered *scanFilterLister) ListDaemonSets(ctx context.Context, namespace string) ([]appsv1.DaemonSet, error) {
	items, err := filtered.inner.ListDaemonSets(ctx, namespace)
	if err != nil {
		return nil, err
	}
	return filterNamedItems(items, namespace, "daemonsets", filtered.filter, func(item appsv1.DaemonSet) string {
		return item.Name
	}), nil
}

func (filtered *scanFilterLister) ListReplicaSets(ctx context.Context, namespace string) ([]appsv1.ReplicaSet, error) {
	return filtered.inner.ListReplicaSets(ctx, namespace)
}

func (filtered *scanFilterLister) ListCronJobs(ctx context.Context, namespace string) ([]batchv1.CronJob, error) {
	items, err := filtered.inner.ListCronJobs(ctx, namespace)
	if err != nil {
		return nil, err
	}
	return filterNamedItems(items, namespace, "cronjobs", filtered.filter, func(item batchv1.CronJob) string {
		return item.Name
	}), nil
}

func (filtered *scanFilterLister) ListJobs(ctx context.Context, namespace string) ([]batchv1.Job, error) {
	items, err := filtered.inner.ListJobs(ctx, namespace)
	if err != nil {
		return nil, err
	}
	return filterNamedItems(items, namespace, "jobs", filtered.filter, func(item batchv1.Job) string {
		return item.Name
	}), nil
}

func (filtered *scanFilterLister) ListServices(ctx context.Context, namespace string) ([]corev1.Service, error) {
	items, err := filtered.inner.ListServices(ctx, namespace)
	if err != nil {
		return nil, err
	}
	return filterNamedItems(items, namespace, "services", filtered.filter, func(item corev1.Service) string {
		return item.Name
	}), nil
}

func (filtered *scanFilterLister) ListIngresses(ctx context.Context, namespace string) ([]networkingv1.Ingress, error) {
	items, err := filtered.inner.ListIngresses(ctx, namespace)
	if err != nil {
		return nil, err
	}
	return filterNamedItems(items, namespace, "ingresses", filtered.filter, func(item networkingv1.Ingress) string {
		return item.Name
	}), nil
}

func (filtered *scanFilterLister) ListConfigMaps(ctx context.Context, namespace string) ([]corev1.ConfigMap, error) {
	return filtered.inner.ListConfigMaps(ctx, namespace)
}

func (filtered *scanFilterLister) ListSecrets(ctx context.Context, namespace string) ([]corev1.Secret, error) {
	return filtered.inner.ListSecrets(ctx, namespace)
}

func (filtered *scanFilterLister) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	return filtered.inner.ListPersistentVolumeClaims(ctx, namespace)
}

func (filtered *scanFilterLister) ListNetworkPolicies(ctx context.Context, namespace string) ([]networkingv1.NetworkPolicy, error) {
	return filtered.inner.ListNetworkPolicies(ctx, namespace)
}

func (filtered *scanFilterLister) ListHorizontalPodAutoscalersV2(ctx context.Context, namespace string) ([]autoscalingv2.HorizontalPodAutoscaler, error) {
	return filtered.inner.ListHorizontalPodAutoscalersV2(ctx, namespace)
}

func filterNamedItems[T any](items []T, namespace, kind string, filter ScanFilter, nameOf func(T) string) []T {
	if len(filter.workloadPatterns) == 0 {
		return items
	}
	out := make([]T, 0, len(items))
	for idx := range items {
		if filter.workloadDenied(namespace, kind, nameOf(items[idx])) {
			continue
		}
		out = append(out, items[idx])
	}
	return out
}
