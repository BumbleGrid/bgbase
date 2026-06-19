package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/BumbleGrid/bgbase/edge"
	"github.com/BumbleGrid/bgbase/node"
)

type istioVSReference struct {
	namespace       string
	sourceHost      string
	destinationHost string
}

type nodeIndex struct {
	byID               map[string]node.Data
	serviceByNSName    map[string]string
	externalByHost     map[string]string
	workloadsByService map[string][]string
}

func buildNodeIndex(nodes []node.Data) nodeIndex {
	idx := nodeIndex{
		byID:               make(map[string]node.Data, len(nodes)),
		serviceByNSName:    make(map[string]string),
		externalByHost:     make(map[string]string),
		workloadsByService: make(map[string][]string),
	}
	for nodeIdx := range nodes {
		item := nodes[nodeIdx]
		idx.byID[item.ID] = item
		if item.K8s == nil || item.K8s.Namespace == nil {
			continue
		}
		namespace := *item.K8s.Namespace
		switch item.BgKind {
		case node.BgKindServiceDiscovery, node.BgKindLoadBalancer:
			if item.K8s.Kind == "Service" {
				key := serviceIndexKey(namespace, item.K8s.Name)
				idx.serviceByNSName[key] = item.ID
			}
		case node.BgKindExternalService:
			host := externalServiceHost(item)
			if host != "" {
				idx.externalByHost[host] = item.ID
			}
		case node.BgKindWorkload:
			continue
		}
	}
	for nodeIdx := range nodes {
		item := nodes[nodeIdx]
		if item.BgKind != node.BgKindWorkload {
			continue
		}
		workloadRest, ok := k8sRestPathFromNodeID(item.ID)
		if !ok {
			continue
		}
		workloadNS, inNS := namespaceFromNamespacedRest(workloadRest)
		if !inNS {
			continue
		}
		for serviceKey, serviceID := range idx.serviceByNSName {
			serviceNode, exists := idx.byID[serviceID]
			if !exists {
				continue
			}
			serviceNS, serviceName := splitServiceIndexKey(serviceKey)
			if serviceNS != workloadNS || serviceNode.Label != item.Label {
				continue
			}
			idx.workloadsByService[serviceID] = append(idx.workloadsByService[serviceID], item.ID)
			_ = serviceName
		}
	}
	return idx
}

func externalServiceHost(item node.Data) string {
	if item.Label != "" {
		return item.Label
	}
	if item.K8s != nil {
		return item.K8s.Name
	}
	return ""
}

func serviceIndexKey(namespace, name string) string {
	return namespace + "/" + name
}

func splitServiceIndexKey(key string) (namespace, name string) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func parseIstioHost(host, defaultNamespace string) (name, namespace string) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", defaultNamespace
	}
	parts := strings.Split(host, ".")
	name = parts[0]
	if len(parts) >= 2 && parts[1] != "svc" {
		namespace = parts[1]
		return name, namespace
	}
	return name, defaultNamespace
}

func (idx nodeIndex) serviceNodeID(host, defaultNamespace string) (string, bool) {
	name, namespace := parseIstioHost(host, defaultNamespace)
	if name == "" {
		return "", false
	}
	nodeID, ok := idx.serviceByNSName[serviceIndexKey(namespace, name)]
	return nodeID, ok
}

func (idx nodeIndex) externalNodeID(host string) (string, bool) {
	nodeID, ok := idx.externalByHost[host]
	return nodeID, ok
}

func relationOnlyKey(sourceID, targetID string, rel edge.BgRelation) string {
	return fmt.Sprintf("%s|%s|%s", sourceID, targetID, rel)
}

func indexRelationKeys(edges []edge.Data) map[string]struct{} {
	out := make(map[string]struct{}, len(edges))
	for idx := range edges {
		key := relationOnlyKey(edges[idx].Source, edges[idx].Target, edges[idx].BGRelation)
		out[key] = struct{}{}
	}
	return out
}

func istioEdgeID(sourceID, targetID string, rel edge.BgRelation) string {
	return fmt.Sprintf("%s--%s--%s", sourceID, strings.ToLower(string(rel)), targetID)
}

func istioEdgeFromNodes(sourceID, targetID string, rel edge.BgRelation, inferred bool, src node.Data, label *string, description string) edge.Data {
	em := edge.Meta{Description: description}
	if src.Meta != nil {
		em.ExtractedAt = src.Meta.ExtractedAt
		em.ExtractorVersion = src.Meta.ExtractorVersion
	}
	return edge.Data{
		ID:               istioEdgeID(sourceID, targetID, rel),
		Source:           sourceID,
		Target:           targetID,
		Floor:            src.Floor,
		BGRelation:       rel,
		ExtractionSource: edge.ExtractionSourceIstioManifest,
		Inferred:         inferred,
		Label:            label,
		Meta:             em,
	}
}

func namespacesFromNodes(nodes []node.Data) []string {
	seen := make(map[string]struct{})
	for idx := range nodes {
		item := nodes[idx]
		if item.K8s == nil || item.K8s.Namespace == nil {
			continue
		}
		seen[*item.K8s.Namespace] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for namespace := range seen {
		out = append(out, namespace)
	}
	return out
}

func (resolver *EdgeResolver) resolveIstioEdges(ctx context.Context, nodes []node.Data, existing []edge.Data) []edge.Data {
	if resolver.istioLister == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return nil
	}
	if !resolver.istioLister.Present(ctx) {
		resolver.debugf("istio API group not present; skipping istio edge pass")
		return nil
	}
	idx := buildNodeIndex(nodes)
	relations := indexRelationKeys(existing)
	out := make([]edge.Data, 0)
	vsRefs := make([]istioVSReference, 0)

	for _, namespace := range namespacesFromNodes(nodes) {
		virtualServices, err := resolver.istioLister.ListVirtualServices(ctx, namespace)
		if err != nil {
			resolver.debugf("istio virtualservices list failed for namespace %q: %v", namespace, err)
			continue
		}
		for vsIdx := range virtualServices {
			vs := virtualServices[vsIdx]
			sourceHost := vs.Hosts[0]
			sourceID, ok := idx.serviceNodeID(sourceHost, vs.Namespace)
			if !ok {
				continue
			}
			sourceNode := idx.byID[sourceID]
			description := fmt.Sprintf("Istio VirtualService: %s", vs.Name)
			for httpIdx := range vs.HTTP {
				httpRoute := vs.HTTP[httpIdx]
				for destIdx := range httpRoute.Destinations {
					destHost := httpRoute.Destinations[destIdx]
					targetID, ok := idx.serviceNodeID(destHost, vs.Namespace)
					if !ok {
						continue
					}
					relKey := relationOnlyKey(sourceID, targetID, edge.BgRelationRoutes)
					if _, exists := relations[relKey]; exists {
						continue
					}
					relations[relKey] = struct{}{}
					out = append(out, istioEdgeFromNodes(sourceID, targetID, edge.BgRelationRoutes, false, sourceNode, httpRoute.MatchPrefix, description))
					vsRefs = append(vsRefs, istioVSReference{
						namespace:       vs.Namespace,
						sourceHost:      sourceHost,
						destinationHost: destHost,
					})
				}
			}
		}

		destinationRules, err := resolver.istioLister.ListDestinationRules(ctx, namespace)
		if err != nil {
			resolver.debugf("istio destinationrules list failed for namespace %q: %v", namespace, err)
			continue
		}
		applyDestinationRuleMetadata(out, destinationRules, idx)

		serviceEntries, err := resolver.istioLister.ListServiceEntries(ctx, namespace)
		if err != nil {
			resolver.debugf("istio serviceentries list failed for namespace %q: %v", namespace, err)
			continue
		}
		for seIdx := range serviceEntries {
			entry := serviceEntries[seIdx]
			for hostIdx := range entry.Hosts {
				host := entry.Hosts[hostIdx]
				externalID, ok := idx.externalNodeID(host)
				if !ok {
					resolver.warnf("istio ServiceEntry %q host %q has no matching ExternalService node; skipping", entry.Name, host)
					continue
				}
				externalNode, ok := idx.byID[externalID]
				if !ok {
					continue
				}
				description := fmt.Sprintf("Istio ServiceEntry: %s", entry.Name)
				for refIdx := range vsRefs {
					ref := vsRefs[refIdx]
					if ref.destinationHost != host {
						continue
					}
					sourceID, ok := idx.serviceNodeID(ref.sourceHost, ref.namespace)
					if !ok {
						continue
					}
					workloads := idx.workloadsByService[sourceID]
					for workloadIdx := range workloads {
						workloadID := workloads[workloadIdx]
						workloadNode, ok := idx.byID[workloadID]
						if !ok {
							continue
						}
						relKey := relationOnlyKey(workloadID, externalID, edge.BgRelationCalls)
						if _, exists := relations[relKey]; exists {
							continue
						}
						relations[relKey] = struct{}{}
						out = append(out, istioEdgeFromNodes(workloadID, externalID, edge.BgRelationCalls, true, workloadNode, nil, description))
					}
				}
				_ = externalNode
			}
		}
	}
	return out
}

func applyDestinationRuleMetadata(edges []edge.Data, rules []IstioDestinationRule, idx nodeIndex) {
	for ruleIdx := range rules {
		rule := rules[ruleIdx]
		targetID, ok := idx.serviceNodeID(rule.Host, rule.Namespace)
		if !ok {
			continue
		}
		targetNode, ok := idx.byID[targetID]
		if !ok {
			continue
		}
		_ = targetNode
		for edgeIdx := range edges {
			item := &edges[edgeIdx]
			if item.BGRelation != edge.BgRelationRoutes || item.ExtractionSource != edge.ExtractionSourceIstioManifest {
				continue
			}
			if item.Target != targetID {
				continue
			}
			sourceNode, ok := idx.byID[item.Source]
			if !ok || sourceNode.K8s == nil {
				continue
			}
			if sourceNode.K8s.Namespace == nil || *sourceNode.K8s.Namespace != rule.Namespace {
				continue
			}
			if len(rule.SubsetNames) > 0 {
				item.Meta.Description += "; DestinationRule subsets: " + strings.Join(rule.SubsetNames, ", ")
			}
			if rule.HasTrafficPolicy {
				item.Meta.Description += "; DestinationRule trafficPolicy attached"
			}
		}
	}
}
