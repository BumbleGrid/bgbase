package k8s

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Reader struct {
	cs kubernetes.Interface
}

func NewReader(client *Client) *Reader {
	if client == nil {
		return &Reader{}
	}
	return &Reader{cs: client.cs}
}

func NewReaderWithClientset(cs kubernetes.Interface) *Reader {
	if cs == nil {
		return &Reader{}
	}
	return &Reader{cs: cs}
}

func (r *Reader) requireClientset() (kubernetes.Interface, error) {
	if r == nil || r.cs == nil {
		return nil, fmt.Errorf("k8s reader: kubernetes clientset is not configured")
	}
	return r.cs, nil
}

func (r *Reader) ListNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListIngressClasses(ctx context.Context) ([]networkingv1.IngressClass, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.NetworkingV1().IngressClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListStatefulSets(ctx context.Context, namespace string) ([]appsv1.StatefulSet, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListDaemonSets(ctx context.Context, namespace string) ([]appsv1.DaemonSet, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListReplicaSets(ctx context.Context, namespace string) ([]appsv1.ReplicaSet, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListCronJobs(ctx context.Context, namespace string) ([]batchv1.CronJob, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListJobs(ctx context.Context, namespace string) ([]batchv1.Job, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListServices(ctx context.Context, namespace string) ([]corev1.Service, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListIngresses(ctx context.Context, namespace string) ([]networkingv1.Ingress, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListConfigMaps(ctx context.Context, namespace string) ([]corev1.ConfigMap, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListSecrets(ctx context.Context, namespace string) ([]corev1.Secret, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListNetworkPolicies(ctx context.Context, namespace string) ([]networkingv1.NetworkPolicy, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *Reader) ListHorizontalPodAutoscalersV2(ctx context.Context, namespace string) ([]autoscalingv2.HorizontalPodAutoscaler, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

type namespacePickLister struct {
	*Reader
	allowed map[string]struct{}
}

// NewListerForNamespaces returns lister unchanged when namespaces is empty.
// Otherwise ListNamespaces is restricted to those names (cluster-scoped lists
// are unchanged).
func NewListerForNamespaces(reader *Reader, namespaces []string) K8sLister {
	if reader == nil || len(namespaces) == 0 {
		return reader
	}
	allowed := make(map[string]struct{}, len(namespaces))
	for idx := range namespaces {
		allowed[namespaces[idx]] = struct{}{}
	}
	return &namespacePickLister{Reader: reader, allowed: allowed}
}

func (pick *namespacePickLister) ListNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	all, err := pick.Reader.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	var out []corev1.Namespace
	for idx := range all {
		if _, ok := pick.allowed[all[idx].Name]; ok {
			out = append(out, all[idx])
		}
	}
	return out, nil
}
