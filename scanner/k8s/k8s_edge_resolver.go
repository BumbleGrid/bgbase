package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/BumbleGrid/bgbase/edge"
	"github.com/BumbleGrid/bgbase/node"
)

const k8sEdgeTagPrefix = "k8s.edge."

func (*EdgeResolver) ResolveEdges(ctx context.Context, nodes []node.Data) ([]edge.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	byID := make(map[string]node.Data, len(nodes))
	for idx := range nodes {
		byID[nodes[idx].ID] = nodes[idx]
	}
	dedupe := make(map[string]struct{})
	out := make([]edge.Data, 0)

	add := func(sourceID, targetID string, rel edge.BgRelation, inferred bool, src node.Data) {
		if sourceID == targetID {
			return
		}
		if _, ok := byID[targetID]; !ok {
			return
		}
		key := edgeDedupeKey(sourceID, targetID, rel, inferred)
		if _, exists := dedupe[key]; exists {
			return
		}
		dedupe[key] = struct{}{}
		out = append(out, edgeFromNodes(sourceID, targetID, rel, inferred, src))
	}

	for idx := range nodes {
		src := nodes[idx]
		edgesFromMetaTags(&src, byID, add)
	}
	applyNameHeuristicEdges(nodes, add)

	sort.Slice(out, func(a, b int) bool {
		return out[a].ID < out[b].ID
	})
	return out, nil
}

func edgeDedupeKey(sourceID, targetID string, rel edge.BgRelation, inferred bool) string {
	return fmt.Sprintf("%s|%s|%s|%t", sourceID, targetID, rel, inferred)
}

func edgeFromNodes(sourceID, targetID string, rel edge.BgRelation, inferred bool, src node.Data) edge.Data {
	em := edge.Meta{}
	if src.Meta != nil {
		em.Description = src.Meta.Description
		em.ExtractedAt = src.Meta.ExtractedAt
		em.ExtractorVersion = src.Meta.ExtractorVersion
	}
	return edge.Data{
		ID:               edgeDedupeKey(sourceID, targetID, rel, inferred),
		Source:           sourceID,
		Target:           targetID,
		Floor:            src.Floor,
		BGRelation:       rel,
		ExtractionSource: edge.ExtractionSourceK8sManifest,
		Inferred:         inferred,
		Meta:             em,
	}
}

func edgesFromMetaTags(src *node.Data, byID map[string]node.Data, add func(sourceID, targetID string, rel edge.BgRelation, inferred bool, src node.Data)) {
	if src == nil || src.Meta == nil {
		return
	}
	for key, targetID := range src.Meta.Tags {
		if key == "" || targetID == "" {
			continue
		}
		if !strings.HasPrefix(key, k8sEdgeTagPrefix) {
			continue
		}
		rel, inferred, ok := parseEdgeTagKey(key)
		if !ok {
			continue
		}
		if _, exists := byID[targetID]; !exists {
			continue
		}
		add(src.ID, targetID, rel, inferred, *src)
	}
}

func parseEdgeTagKey(key string) (rel edge.BgRelation, inferred bool, ok bool) {
	if !strings.HasPrefix(key, k8sEdgeTagPrefix) {
		return "", false, false
	}
	tail := strings.TrimPrefix(key, k8sEdgeTagPrefix)
	if strings.HasSuffix(tail, ".inferred") {
		inferred = true
		tail = strings.TrimSuffix(tail, ".inferred")
	}
	switch tail {
	case "routes":
		return edge.BgRelationRoutes, inferred, true
	case "exposes":
		return edge.BgRelationExposes, inferred, true
	case "mounts":
		return edge.BgRelationMounts, inferred, true
	case "scheduled-by":
		return edge.BgRelationScheduledBy, inferred, true
	case "calls":
		return edge.BgRelationCalls, inferred, true
	default:
		return "", false, false
	}
}

func k8sRestPathFromNodeID(nodeID string) (rest string, ok bool) {
	const marker = "/k8s/"
	idx := strings.Index(nodeID, marker)
	if idx < 0 {
		return "", false
	}
	return nodeID[idx+len(marker):], true
}

func namespaceFromNamespacedRest(rest string) (ns string, ok bool) {
	const prefix = "namespaces/"
	if !strings.HasPrefix(rest, prefix) {
		return "", false
	}
	rest = strings.TrimPrefix(rest, prefix)
	slash := strings.Index(rest, "/")
	if slash <= 0 {
		return "", false
	}
	return rest[:slash], true
}

func restContainsSegment(rest, pluralSegment string) bool {
	return strings.Contains(rest, "/"+pluralSegment+"/")
}

func tagsDeclareRelation(src node.Data, targetID string, wantRel edge.BgRelation) bool {
	if src.Meta == nil {
		return false
	}
	for key, val := range src.Meta.Tags {
		if val != targetID {
			continue
		}
		if !strings.HasPrefix(key, k8sEdgeTagPrefix) {
			continue
		}
		gotRel, _, ok := parseEdgeTagKey(key)
		if !ok || gotRel != wantRel {
			continue
		}
		return true
	}
	return false
}

func applyNameHeuristicEdges(nodes []node.Data, add func(sourceID, targetID string, rel edge.BgRelation, inferred bool, src node.Data)) {
	for idx := range nodes {
		src := nodes[idx]
		srcRest, ok := k8sRestPathFromNodeID(src.ID)
		if !ok {
			continue
		}
		srcNS, srcInNS := namespaceFromNamespacedRest(srcRest)
		if !srcInNS {
			continue
		}
		switch src.BgKind {
		case node.BgKindServiceDiscovery, node.BgKindLoadBalancer:
			if !restContainsSegment(srcRest, "services") {
				continue
			}
			for jdx := range nodes {
				dst := nodes[jdx]
				if dst.ID == src.ID || dst.BgKind != node.BgKindWorkload {
					continue
				}
				dstRest, ok := k8sRestPathFromNodeID(dst.ID)
				if !ok {
					continue
				}
				dstNS, dstInNS := namespaceFromNamespacedRest(dstRest)
				if !dstInNS || dstNS != srcNS {
					continue
				}
				if !workloadRestPath(dstRest) {
					continue
				}
				if src.Label != dst.Label {
					continue
				}
				if tagsDeclareRelation(src, dst.ID, edge.BgRelationExposes) {
					continue
				}
				add(src.ID, dst.ID, edge.BgRelationExposes, true, src)
			}
		case node.BgKindGateway:
			if !restContainsSegment(srcRest, "ingresses") {
				continue
			}
			for jdx := range nodes {
				dst := nodes[jdx]
				if dst.ID == src.ID {
					continue
				}
				if dst.BgKind != node.BgKindServiceDiscovery && dst.BgKind != node.BgKindLoadBalancer {
					continue
				}
				dstRest, ok := k8sRestPathFromNodeID(dst.ID)
				if !ok {
					continue
				}
				dstNS, dstInNS := namespaceFromNamespacedRest(dstRest)
				if !dstInNS || dstNS != srcNS {
					continue
				}
				if !restContainsSegment(dstRest, "services") {
					continue
				}
				if src.Label != dst.Label {
					continue
				}
				if tagsDeclareRelation(src, dst.ID, edge.BgRelationRoutes) {
					continue
				}
				add(src.ID, dst.ID, edge.BgRelationRoutes, true, src)
			}
		case node.BgKindJobRunner:
			if !restContainsSegment(srcRest, "jobs") {
				continue
			}
			for jdx := range nodes {
				dst := nodes[jdx]
				if dst.ID == src.ID || dst.BgKind != node.BgKindJobRunner {
					continue
				}
				dstRest, ok := k8sRestPathFromNodeID(dst.ID)
				if !ok {
					continue
				}
				dstNS, dstInNS := namespaceFromNamespacedRest(dstRest)
				if !dstInNS || dstNS != srcNS {
					continue
				}
				if !restContainsSegment(dstRest, "cronjobs") {
					continue
				}
				if strings.HasPrefix(src.Label, dst.Label+"-") {
					if tagsDeclareRelation(src, dst.ID, edge.BgRelationScheduledBy) {
						continue
					}
					add(src.ID, dst.ID, edge.BgRelationScheduledBy, true, src)
				}
			}
		default:
			continue
		}
	}
}

func workloadRestPath(rest string) bool {
	return restContainsSegment(rest, "deployments") ||
		restContainsSegment(rest, "statefulsets") ||
		restContainsSegment(rest, "daemonsets") ||
		restContainsSegment(rest, "replicasets")
}
