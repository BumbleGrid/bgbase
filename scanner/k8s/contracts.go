package k8s

import (
	"context"

	"github.com/BumbleGrid/bgbase/edge"
	"github.com/BumbleGrid/bgbase/node"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

type K8sTranslateContext struct {
	Floor                 int
	Meta                  node.Meta
	ClusterNodeID         string
	NamespaceName         string
	NamespaceParentNodeID string
}

type K8sLister interface {
	ListNamespaces(ctx context.Context) ([]corev1.Namespace, error)
	ListPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error)
	ListIngressClasses(ctx context.Context) ([]networkingv1.IngressClass, error)

	ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error)
	ListStatefulSets(ctx context.Context, namespace string) ([]appsv1.StatefulSet, error)
	ListDaemonSets(ctx context.Context, namespace string) ([]appsv1.DaemonSet, error)
	ListReplicaSets(ctx context.Context, namespace string) ([]appsv1.ReplicaSet, error)

	ListCronJobs(ctx context.Context, namespace string) ([]batchv1.CronJob, error)
	ListJobs(ctx context.Context, namespace string) ([]batchv1.Job, error)

	ListServices(ctx context.Context, namespace string) ([]corev1.Service, error)
	ListIngresses(ctx context.Context, namespace string) ([]networkingv1.Ingress, error)

	ListConfigMaps(ctx context.Context, namespace string) ([]corev1.ConfigMap, error)
	ListSecrets(ctx context.Context, namespace string) ([]corev1.Secret, error)
	ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error)

	ListNetworkPolicies(ctx context.Context, namespace string) ([]networkingv1.NetworkPolicy, error)

	ListHorizontalPodAutoscalersV2(ctx context.Context, namespace string) ([]autoscalingv2.HorizontalPodAutoscaler, error)
}

type K8sNodeTranslator interface {
	TranslateCluster(ctx context.Context, tctx K8sTranslateContext) (node.Data, error)

	TranslateNamespaces(ctx context.Context, tctx K8sTranslateContext, items []corev1.Namespace) ([]node.Data, error)
	TranslatePersistentVolumes(ctx context.Context, tctx K8sTranslateContext, items []corev1.PersistentVolume) ([]node.Data, error)
	TranslateIngressClasses(ctx context.Context, tctx K8sTranslateContext, items []networkingv1.IngressClass) ([]node.Data, error)

	TranslateDeployments(ctx context.Context, tctx K8sTranslateContext, items []appsv1.Deployment) ([]node.Data, error)
	TranslateStatefulSets(ctx context.Context, tctx K8sTranslateContext, items []appsv1.StatefulSet) ([]node.Data, error)
	TranslateDaemonSets(ctx context.Context, tctx K8sTranslateContext, items []appsv1.DaemonSet) ([]node.Data, error)
	TranslateReplicaSets(ctx context.Context, tctx K8sTranslateContext, items []appsv1.ReplicaSet) ([]node.Data, error)

	TranslateCronJobs(ctx context.Context, tctx K8sTranslateContext, items []batchv1.CronJob) ([]node.Data, error)
	TranslateJobs(ctx context.Context, tctx K8sTranslateContext, items []batchv1.Job) ([]node.Data, error)

	TranslateServices(ctx context.Context, tctx K8sTranslateContext, items []corev1.Service) ([]node.Data, error)
	TranslateIngresses(ctx context.Context, tctx K8sTranslateContext, items []networkingv1.Ingress) ([]node.Data, error)

	TranslateConfigMaps(ctx context.Context, tctx K8sTranslateContext, items []corev1.ConfigMap) ([]node.Data, error)
	TranslateSecrets(ctx context.Context, tctx K8sTranslateContext, items []corev1.Secret) ([]node.Data, error)
	TranslatePersistentVolumeClaims(ctx context.Context, tctx K8sTranslateContext, items []corev1.PersistentVolumeClaim) ([]node.Data, error)

	TranslateNetworkPolicies(ctx context.Context, tctx K8sTranslateContext, items []networkingv1.NetworkPolicy) ([]node.Data, error)

	TranslateHorizontalPodAutoscalersV2(ctx context.Context, tctx K8sTranslateContext, items []autoscalingv2.HorizontalPodAutoscaler) ([]node.Data, error)
}

type K8sEdgeResolver interface {
	// ResolveEdges emits edge.Data values from the assembled nodes. Set
	// Inferred to true for edges derived from label selector matching,
	// and to false for edges grounded in explicit manifest references
	// such as an Ingress rule backend or a PVC volumeName.
	ResolveEdges(ctx context.Context, nodes []node.Data) ([]edge.Data, error)
}
