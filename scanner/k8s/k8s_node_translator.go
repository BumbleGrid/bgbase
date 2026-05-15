package k8s

import (
	"context"
	"fmt"

	"github.com/BumbleGrid/bgbase/node"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func k8sRESTID(clusterNodeID, restPath string) string {
	return fmt.Sprintf("%s/k8s/%s", clusterNodeID, restPath)
}

func metaWithObjectLabels(base node.Meta, objectLabels map[string]string) *node.Meta {
	if len(objectLabels) == 0 {
		copyMeta := base
		return &copyMeta
	}
	merged := base
	tags := make(map[string]string, len(base.Tags)+len(objectLabels))
	for key, val := range base.Tags {
		tags[key] = val
	}
	for key, val := range objectLabels {
		tags[key] = val
	}
	merged.Tags = tags
	return &merged
}

func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for key, val := range src {
		out[key] = val
	}
	return out
}

func baseK8s(typeMeta metav1.TypeMeta, objMeta metav1.ObjectMeta, kind string) *node.K8sMetadata {
	km := &node.K8sMetadata{
		Kind: kind,
		Name: objMeta.Name,
	}
	if typeMeta.APIVersion != "" {
		km.APIVersion = typeMeta.APIVersion
	}
	if objMeta.Namespace != "" {
		namespace := objMeta.Namespace
		km.Namespace = &namespace
	}
	if objMeta.UID != "" {
		km.UID = string(objMeta.UID)
	}
	if labels := copyStringMap(objMeta.Labels); labels != nil {
		km.Labels = labels
	}
	if ann := copyStringMap(objMeta.Annotations); ann != nil {
		km.Annotations = ann
	}
	return km
}

func firstContainerImage(spec *corev1.PodSpec) string {
	if spec == nil || len(spec.Containers) == 0 {
		return ""
	}
	return spec.Containers[0].Image
}

func resourceAmountsFromList(rl corev1.ResourceList) *node.K8sResourceAmounts {
	if len(rl) == 0 {
		return nil
	}
	var amounts node.K8sResourceAmounts
	if qty, ok := rl[corev1.ResourceCPU]; ok {
		amounts.CPU = qty.String()
	}
	if qty, ok := rl[corev1.ResourceMemory]; ok {
		amounts.Memory = qty.String()
	}
	if amounts.CPU == "" && amounts.Memory == "" {
		return nil
	}
	return &amounts
}

func k8sResourcesFromRequirements(req corev1.ResourceRequirements) *node.K8sResources {
	reqs := resourceAmountsFromList(req.Requests)
	limits := resourceAmountsFromList(req.Limits)
	if reqs == nil && limits == nil {
		return nil
	}
	return &node.K8sResources{Requests: reqs, Limits: limits}
}

func firstContainerResources(spec *corev1.PodSpec) *node.K8sResources {
	if spec == nil || len(spec.Containers) == 0 {
		return nil
	}
	return k8sResourcesFromRequirements(spec.Containers[0].Resources)
}

func podTemplateWorkloadPatch(template *corev1.PodTemplateSpec, replicas *int32) func(*node.K8sMetadata) {
	return func(km *node.K8sMetadata) {
		if replicas != nil {
			replicaCount := int(*replicas)
			km.Replicas = &replicaCount
		}
		if template == nil {
			return
		}
		if image := firstContainerImage(&template.Spec); image != "" {
			imageCopy := image
			km.Image = &imageCopy
		}
		if res := firstContainerResources(&template.Spec); res != nil {
			km.Resources = res
		}
	}
}

func k8sServicePortsFrom(ports []corev1.ServicePort) []node.K8sPort {
	out := make([]node.K8sPort, 0, len(ports))
	for idx := range ports {
		port := ports[idx]
		entry := node.K8sPort{Port: int(port.Port)}
		if port.Name != "" {
			entry.Name = port.Name
		}
		switch port.TargetPort.Type {
		case intstr.Int:
			if port.TargetPort.IntVal != 0 {
				entry.TargetPort = port.TargetPort.IntVal
			}
		case intstr.String:
			if port.TargetPort.StrVal != "" {
				entry.TargetPort = port.TargetPort.StrVal
			}
		}
		if port.Protocol != "" {
			entry.Protocol = string(port.Protocol)
		} else {
			entry.Protocol = string(corev1.ProtocolTCP)
		}
		out = append(out, entry)
	}
	return out
}

func serviceSpecPatch(spec *corev1.ServiceSpec) func(*node.K8sMetadata) {
	return func(km *node.K8sMetadata) {
		serviceType := string(spec.Type)
		km.ServiceType = &serviceType
		if len(spec.Ports) > 0 {
			km.Ports = k8sServicePortsFrom(spec.Ports)
		}
	}
}

func pvcSpecPatch(spec *corev1.PersistentVolumeClaimSpec) func(*node.K8sMetadata) {
	return func(km *node.K8sMetadata) {
		if spec.StorageClassName != nil && *spec.StorageClassName != "" {
			className := *spec.StorageClassName
			km.StorageClass = &className
		}
		if qty, ok := spec.Resources.Requests[corev1.ResourceStorage]; ok {
			capacity := qty.String()
			km.StorageCapacity = &capacity
		}
	}
}

func pvSpecPatch(spec *corev1.PersistentVolumeSpec) func(*node.K8sMetadata) {
	return func(km *node.K8sMetadata) {
		if spec.StorageClassName != "" {
			className := spec.StorageClassName
			km.StorageClass = &className
		}
		if qty, ok := spec.Capacity[corev1.ResourceStorage]; ok {
			capacity := qty.String()
			km.StorageCapacity = &capacity
		}
	}
}

func cronJobSpecPatch(spec *batchv1.CronJobSpec) func(*node.K8sMetadata) {
	return func(km *node.K8sMetadata) {
		if spec.Schedule != "" {
			schedule := spec.Schedule
			km.Schedule = &schedule
		}
	}
}

func nodeFromK8s(
	tctx K8sTranslateContext,
	restPath, graphLabel string,
	bgKind node.BgKind,
	parent *string,
	typeMeta metav1.TypeMeta,
	objMeta metav1.ObjectMeta,
	k8sKind string,
	patch func(*node.K8sMetadata),
) node.Data {
	k8s := baseK8s(typeMeta, objMeta, k8sKind)
	if patch != nil {
		patch(k8s)
	}
	return node.Data{
		ID:            k8sRESTID(tctx.ClusterNodeID, restPath),
		Label:         graphLabel,
		Floor:         tctx.Floor,
		BgKind:        bgKind,
		Parent:        parent,
		InfraProvider: node.InfraProviderKubernetes,
		K8s:           k8s,
		Meta:          metaWithObjectLabels(tctx.Meta, objMeta.Labels),
	}
}

func (*NodeTranslator) TranslateNamespaces(ctx context.Context, tctx K8sTranslateContext, items []corev1.Namespace) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.ClusterNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s", item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindNamespace, &parent, item.TypeMeta, item.ObjectMeta, "Namespace", nil))
	}
	return out, nil
}

func (*NodeTranslator) TranslatePersistentVolumes(ctx context.Context, tctx K8sTranslateContext, items []corev1.PersistentVolume) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.ClusterNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("persistentvolumes/%s", item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindStorage, &parent, item.TypeMeta, item.ObjectMeta, "PersistentVolume", pvSpecPatch(&item.Spec)))
	}
	return out, nil
}

func (*NodeTranslator) TranslateIngressClasses(ctx context.Context, tctx K8sTranslateContext, items []networkingv1.IngressClass) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.ClusterNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("ingressclasses/%s", item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindGateway, &parent, item.TypeMeta, item.ObjectMeta, "IngressClass", nil))
	}
	return out, nil
}

func (*NodeTranslator) TranslateDeployments(ctx context.Context, tctx K8sTranslateContext, items []appsv1.Deployment) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/deployments/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindWorkload, &parent, item.TypeMeta, item.ObjectMeta, "Deployment", podTemplateWorkloadPatch(&item.Spec.Template, item.Spec.Replicas)))
	}
	return out, nil
}

func (*NodeTranslator) TranslateStatefulSets(ctx context.Context, tctx K8sTranslateContext, items []appsv1.StatefulSet) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/statefulsets/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindWorkload, &parent, item.TypeMeta, item.ObjectMeta, "StatefulSet", podTemplateWorkloadPatch(&item.Spec.Template, item.Spec.Replicas)))
	}
	return out, nil
}

func (*NodeTranslator) TranslateDaemonSets(ctx context.Context, tctx K8sTranslateContext, items []appsv1.DaemonSet) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/daemonsets/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindWorkload, &parent, item.TypeMeta, item.ObjectMeta, "DaemonSet", podTemplateWorkloadPatch(&item.Spec.Template, nil)))
	}
	return out, nil
}

func (*NodeTranslator) TranslateReplicaSets(ctx context.Context, tctx K8sTranslateContext, items []appsv1.ReplicaSet) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/replicasets/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindWorkload, &parent, item.TypeMeta, item.ObjectMeta, "ReplicaSet", podTemplateWorkloadPatch(&item.Spec.Template, item.Spec.Replicas)))
	}
	return out, nil
}

func (*NodeTranslator) TranslateCronJobs(ctx context.Context, tctx K8sTranslateContext, items []batchv1.CronJob) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/cronjobs/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindJobRunner, &parent, item.TypeMeta, item.ObjectMeta, "CronJob", cronJobSpecPatch(&item.Spec)))
	}
	return out, nil
}

func (*NodeTranslator) TranslateJobs(ctx context.Context, tctx K8sTranslateContext, items []batchv1.Job) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/jobs/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindJobRunner, &parent, item.TypeMeta, item.ObjectMeta, "Job", podTemplateWorkloadPatch(&item.Spec.Template, nil)))
	}
	return out, nil
}

func serviceBgKind(svc *corev1.Service) node.BgKind {
	switch svc.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		return node.BgKindLoadBalancer
	case corev1.ServiceTypeExternalName:
		return node.BgKindExternalService
	default:
		return node.BgKindServiceDiscovery
	}
}

func (*NodeTranslator) TranslateServices(ctx context.Context, tctx K8sTranslateContext, items []corev1.Service) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/services/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, serviceBgKind(&item), &parent, item.TypeMeta, item.ObjectMeta, "Service", serviceSpecPatch(&item.Spec)))
	}
	return out, nil
}

func (*NodeTranslator) TranslateIngresses(ctx context.Context, tctx K8sTranslateContext, items []networkingv1.Ingress) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/ingresses/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindGateway, &parent, item.TypeMeta, item.ObjectMeta, "Ingress", nil))
	}
	return out, nil
}

func (*NodeTranslator) TranslateConfigMaps(ctx context.Context, tctx K8sTranslateContext, items []corev1.ConfigMap) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/configmaps/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindConfigSource, &parent, item.TypeMeta, item.ObjectMeta, "ConfigMap", nil))
	}
	return out, nil
}

func (*NodeTranslator) TranslateSecrets(ctx context.Context, tctx K8sTranslateContext, items []corev1.Secret) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/secrets/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindSecretSource, &parent, item.TypeMeta, item.ObjectMeta, "Secret", nil))
	}
	return out, nil
}

func (*NodeTranslator) TranslatePersistentVolumeClaims(ctx context.Context, tctx K8sTranslateContext, items []corev1.PersistentVolumeClaim) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/persistentvolumeclaims/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindStorage, &parent, item.TypeMeta, item.ObjectMeta, "PersistentVolumeClaim", pvcSpecPatch(&item.Spec)))
	}
	return out, nil
}

func (*NodeTranslator) TranslateNetworkPolicies(ctx context.Context, tctx K8sTranslateContext, items []networkingv1.NetworkPolicy) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/networkpolicies/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindNetworkPolicy, &parent, item.TypeMeta, item.ObjectMeta, "NetworkPolicy", nil))
	}
	return out, nil
}

func (*NodeTranslator) TranslateHorizontalPodAutoscalersV2(ctx context.Context, tctx K8sTranslateContext, items []autoscalingv2.HorizontalPodAutoscaler) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/horizontalpodautoscalers/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindWorkload, &parent, item.TypeMeta, item.ObjectMeta, "HorizontalPodAutoscaler", nil))
	}
	return out, nil
}
