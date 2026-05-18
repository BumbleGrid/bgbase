package k8s

import (
	"context"
	"fmt"
	"sort"

	"github.com/BumbleGrid/bgbase/edge"
	"github.com/BumbleGrid/bgbase/floor"
	"github.com/BumbleGrid/bgbase/node"
)

func namespaceParentNodeID(clusterNodeID, namespace string) string {
	return fmt.Sprintf("%s/k8s/namespaces/%s", clusterNodeID, namespace)
}

func wrapNodes(data []node.Data) []node.Wrapper {
	out := make([]node.Wrapper, len(data))
	for idx := range data {
		out[idx] = node.Wrapper{Data: data[idx]}
	}
	return out
}

func wrapEdges(data []edge.Data) []edge.Wrapper {
	out := make([]edge.Wrapper, len(data))
	for idx := range data {
		out[idx] = edge.Wrapper{Data: data[idx]}
	}
	return out
}

func Floor0Extractor(
	ctx context.Context,
	lister K8sLister,
	translator K8sNodeTranslator,
	resolver K8sEdgeResolver,
	tctx K8sTranslateContext,
) (floor.Content, error) {
	var content floor.Content
	content.Floor = tctx.Floor

	clusterTctx := tctx
	clusterTctx.NamespaceName = ""
	clusterTctx.NamespaceParentNodeID = ""

	var flat []node.Data

	clusterNode, err := translator.TranslateCluster(ctx, clusterTctx)
	if err != nil {
		return content, fmt.Errorf("translate cluster: %w", err)
	}
	flat = append(flat, clusterNode)

	namespaces, err := lister.ListNamespaces(ctx)
	if err != nil {
		return content, fmt.Errorf("list namespaces: %w", err)
	}
	nsNodes, err := translator.TranslateNamespaces(ctx, clusterTctx, namespaces)
	if err != nil {
		return content, fmt.Errorf("translate namespaces: %w", err)
	}
	flat = append(flat, nsNodes...)

	pvs, err := lister.ListPersistentVolumes(ctx)
	if err != nil {
		return content, fmt.Errorf("list persistent volumes: %w", err)
	}
	pvNodes, err := translator.TranslatePersistentVolumes(ctx, clusterTctx, pvs)
	if err != nil {
		return content, fmt.Errorf("translate persistent volumes: %w", err)
	}
	flat = append(flat, pvNodes...)

	ingClasses, err := lister.ListIngressClasses(ctx)
	if err != nil {
		return content, fmt.Errorf("list ingress classes: %w", err)
	}
	ingClassNodes, err := translator.TranslateIngressClasses(ctx, clusterTctx, ingClasses)
	if err != nil {
		return content, fmt.Errorf("translate ingress classes: %w", err)
	}
	flat = append(flat, ingClassNodes...)

	nsNames := make([]string, 0, len(namespaces))
	for idx := range namespaces {
		nsNames = append(nsNames, namespaces[idx].Name)
	}
	sort.Strings(nsNames)

	for _, nsName := range nsNames {
		nsTctx := tctx
		nsTctx.NamespaceName = nsName
		nsTctx.NamespaceParentNodeID = namespaceParentNodeID(tctx.ClusterNodeID, nsName)

		deploys, err := lister.ListDeployments(ctx, nsName)
		if err != nil {
			return content, fmt.Errorf("namespace %q list deployments: %w", nsName, err)
		}
		deployNodes, err := translator.TranslateDeployments(ctx, nsTctx, deploys)
		if err != nil {
			return content, fmt.Errorf("namespace %q translate deployments: %w", nsName, err)
		}
		flat = append(flat, deployNodes...)

		stsItems, err := lister.ListStatefulSets(ctx, nsName)
		if err != nil {
			return content, fmt.Errorf("namespace %q list statefulsets: %w", nsName, err)
		}
		stsNodes, err := translator.TranslateStatefulSets(ctx, nsTctx, stsItems)
		if err != nil {
			return content, fmt.Errorf("namespace %q translate statefulsets: %w", nsName, err)
		}
		flat = append(flat, stsNodes...)

		dsItems, err := lister.ListDaemonSets(ctx, nsName)
		if err != nil {
			return content, fmt.Errorf("namespace %q list daemonsets: %w", nsName, err)
		}
		dsNodes, err := translator.TranslateDaemonSets(ctx, nsTctx, dsItems)
		if err != nil {
			return content, fmt.Errorf("namespace %q translate daemonsets: %w", nsName, err)
		}
		flat = append(flat, dsNodes...)

		rsItems, err := lister.ListReplicaSets(ctx, nsName)
		if err != nil {
			return content, fmt.Errorf("namespace %q list replicasets: %w", nsName, err)
		}
		rsNodes, err := translator.TranslateReplicaSets(ctx, nsTctx, rsItems)
		if err != nil {
			return content, fmt.Errorf("namespace %q translate replicasets: %w", nsName, err)
		}
		flat = append(flat, rsNodes...)

		cjItems, err := lister.ListCronJobs(ctx, nsName)
		if err != nil {
			return content, fmt.Errorf("namespace %q list cronjobs: %w", nsName, err)
		}
		cjNodes, err := translator.TranslateCronJobs(ctx, nsTctx, cjItems)
		if err != nil {
			return content, fmt.Errorf("namespace %q translate cronjobs: %w", nsName, err)
		}
		flat = append(flat, cjNodes...)

		jobItems, err := lister.ListJobs(ctx, nsName)
		if err != nil {
			return content, fmt.Errorf("namespace %q list jobs: %w", nsName, err)
		}
		jobNodes, err := translator.TranslateJobs(ctx, nsTctx, jobItems)
		if err != nil {
			return content, fmt.Errorf("namespace %q translate jobs: %w", nsName, err)
		}
		flat = append(flat, jobNodes...)

		svcItems, err := lister.ListServices(ctx, nsName)
		if err != nil {
			return content, fmt.Errorf("namespace %q list services: %w", nsName, err)
		}
		svcNodes, err := translator.TranslateServices(ctx, nsTctx, svcItems)
		if err != nil {
			return content, fmt.Errorf("namespace %q translate services: %w", nsName, err)
		}
		flat = append(flat, svcNodes...)

		ingItems, err := lister.ListIngresses(ctx, nsName)
		if err != nil {
			return content, fmt.Errorf("namespace %q list ingresses: %w", nsName, err)
		}
		ingNodes, err := translator.TranslateIngresses(ctx, nsTctx, ingItems)
		if err != nil {
			return content, fmt.Errorf("namespace %q translate ingresses: %w", nsName, err)
		}
		flat = append(flat, ingNodes...)

		cmItems, err := lister.ListConfigMaps(ctx, nsName)
		if err != nil {
			return content, fmt.Errorf("namespace %q list configmaps: %w", nsName, err)
		}
		cmNodes, err := translator.TranslateConfigMaps(ctx, nsTctx, cmItems)
		if err != nil {
			return content, fmt.Errorf("namespace %q translate configmaps: %w", nsName, err)
		}
		flat = append(flat, cmNodes...)

		secItems, err := lister.ListSecrets(ctx, nsName)
		if err != nil {
			return content, fmt.Errorf("namespace %q list secrets: %w", nsName, err)
		}
		secNodes, err := translator.TranslateSecrets(ctx, nsTctx, secItems)
		if err != nil {
			return content, fmt.Errorf("namespace %q translate secrets: %w", nsName, err)
		}
		flat = append(flat, secNodes...)

		pvcItems, err := lister.ListPersistentVolumeClaims(ctx, nsName)
		if err != nil {
			return content, fmt.Errorf("namespace %q list pvcs: %w", nsName, err)
		}
		pvcNodes, err := translator.TranslatePersistentVolumeClaims(ctx, nsTctx, pvcItems)
		if err != nil {
			return content, fmt.Errorf("namespace %q translate pvcs: %w", nsName, err)
		}
		flat = append(flat, pvcNodes...)

		npItems, err := lister.ListNetworkPolicies(ctx, nsName)
		if err != nil {
			return content, fmt.Errorf("namespace %q list networkpolicies: %w", nsName, err)
		}
		npNodes, err := translator.TranslateNetworkPolicies(ctx, nsTctx, npItems)
		if err != nil {
			return content, fmt.Errorf("namespace %q translate networkpolicies: %w", nsName, err)
		}
		flat = append(flat, npNodes...)

		hpaItems, err := lister.ListHorizontalPodAutoscalersV2(ctx, nsName)
		if err != nil {
			return content, fmt.Errorf("namespace %q list horizontalpodautoscalers: %w", nsName, err)
		}
		hpaNodes, err := translator.TranslateHorizontalPodAutoscalersV2(ctx, nsTctx, hpaItems)
		if err != nil {
			return content, fmt.Errorf("namespace %q translate horizontalpodautoscalers: %w", nsName, err)
		}
		flat = append(flat, hpaNodes...)
	}

	edgeData, err := resolver.ResolveEdges(ctx, flat)
	if err != nil {
		return content, fmt.Errorf("resolve edges: %w", err)
	}

	content.Nodes = wrapNodes(flat)
	content.Edges = wrapEdges(edgeData)
	return content, nil
}
