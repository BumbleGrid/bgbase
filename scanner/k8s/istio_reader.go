package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

const istioNetworkingAPIGroup = "networking.istio.io"

func IsIstioPresent(ctx context.Context, discoveryClient discovery.DiscoveryInterface) bool {
	if err := ctx.Err(); err != nil {
		return false
	}
	if discoveryClient == nil {
		return false
	}
	groups, err := discoveryClient.ServerGroups()
	if err != nil {
		return false
	}
	for idx := range groups.Groups {
		if groups.Groups[idx].Name == istioNetworkingAPIGroup {
			return true
		}
	}
	return false
}

var (
	istioVirtualServiceGVR = schema.GroupVersionResource{
		Group:    istioNetworkingAPIGroup,
		Version:  "v1beta1",
		Resource: "virtualservices",
	}
	istioDestinationRuleGVR = schema.GroupVersionResource{
		Group:    istioNetworkingAPIGroup,
		Version:  "v1beta1",
		Resource: "destinationrules",
	}
	istioServiceEntryGVR = schema.GroupVersionResource{
		Group:    istioNetworkingAPIGroup,
		Version:  "v1beta1",
		Resource: "serviceentries",
	}
)

type IstioCRDLister interface {
	Present(ctx context.Context) bool
	ListVirtualServices(ctx context.Context, namespace string) ([]IstioVirtualService, error)
	ListDestinationRules(ctx context.Context, namespace string) ([]IstioDestinationRule, error)
	ListServiceEntries(ctx context.Context, namespace string) ([]IstioServiceEntry, error)
}

type istioCRDReader struct {
	discovery discovery.DiscoveryInterface
	dynamic   dynamic.Interface
}

func NewIstioCRDLister(client *Client) IstioCRDLister {
	if client == nil {
		return nil
	}
	return &istioCRDReader{
		discovery: client.discovery,
		dynamic:   client.dynamic,
	}
}

func (reader *istioCRDReader) Present(ctx context.Context) bool {
	return IsIstioPresent(ctx, reader.discovery)
}

func (reader *istioCRDReader) ListVirtualServices(ctx context.Context, namespace string) ([]IstioVirtualService, error) {
	list, err := reader.dynamic.Resource(istioVirtualServiceGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]IstioVirtualService, 0, len(list.Items))
	for idx := range list.Items {
		parsed, ok := parseVirtualService(&list.Items[idx])
		if ok {
			out = append(out, parsed)
		}
	}
	return out, nil
}

func (reader *istioCRDReader) ListDestinationRules(ctx context.Context, namespace string) ([]IstioDestinationRule, error) {
	list, err := reader.dynamic.Resource(istioDestinationRuleGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]IstioDestinationRule, 0, len(list.Items))
	for idx := range list.Items {
		parsed, ok := parseDestinationRule(&list.Items[idx])
		if ok {
			out = append(out, parsed)
		}
	}
	return out, nil
}

func (reader *istioCRDReader) ListServiceEntries(ctx context.Context, namespace string) ([]IstioServiceEntry, error) {
	list, err := reader.dynamic.Resource(istioServiceEntryGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]IstioServiceEntry, 0, len(list.Items))
	for idx := range list.Items {
		parsed, ok := parseServiceEntry(&list.Items[idx])
		if ok {
			out = append(out, parsed)
		}
	}
	return out, nil
}

func parseVirtualService(obj *unstructured.Unstructured) (IstioVirtualService, bool) {
	if obj == nil {
		return IstioVirtualService{}, false
	}
	hosts, _, err := unstructured.NestedStringSlice(obj.Object, "spec", "hosts")
	if err != nil || len(hosts) == 0 {
		return IstioVirtualService{}, false
	}
	httpSlice, _, err := unstructured.NestedSlice(obj.Object, "spec", "http")
	if err != nil {
		return IstioVirtualService{}, false
	}
	routes := make([]IstioHTTPRoute, 0, len(httpSlice))
	for _, entry := range httpSlice {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		var matchPrefix *string
		matchSlice, _, _ := unstructured.NestedSlice(entryMap, "match")
		if len(matchSlice) > 0 {
			if matchMap, ok := matchSlice[0].(map[string]any); ok {
				prefix, found, _ := unstructured.NestedString(matchMap, "uri", "prefix")
				if found && prefix != "" {
					matchPrefix = &prefix
				}
			}
		}
		routeSlice, _, _ := unstructured.NestedSlice(entryMap, "route")
		destinations := make([]string, 0, len(routeSlice))
		for _, routeEntry := range routeSlice {
			routeMap, ok := routeEntry.(map[string]any)
			if !ok {
				continue
			}
			host, found, _ := unstructured.NestedString(routeMap, "destination", "host")
			if found && host != "" {
				destinations = append(destinations, host)
			}
		}
		if len(destinations) > 0 {
			routes = append(routes, IstioHTTPRoute{
				MatchPrefix:  matchPrefix,
				Destinations: destinations,
			})
		}
	}
	return IstioVirtualService{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
		Hosts:     hosts,
		HTTP:      routes,
	}, true
}

func parseDestinationRule(obj *unstructured.Unstructured) (IstioDestinationRule, bool) {
	if obj == nil {
		return IstioDestinationRule{}, false
	}
	host, found, err := unstructured.NestedString(obj.Object, "spec", "host")
	if err != nil || !found || host == "" {
		return IstioDestinationRule{}, false
	}
	subsetSlice, _, _ := unstructured.NestedSlice(obj.Object, "spec", "subsets")
	subsetNames := make([]string, 0, len(subsetSlice))
	for _, entry := range subsetSlice {
		subsetMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		name, found, _ := unstructured.NestedString(subsetMap, "name")
		if found && name != "" {
			subsetNames = append(subsetNames, name)
		}
	}
	_, hasTrafficPolicy, _ := unstructured.NestedMap(obj.Object, "spec", "trafficPolicy")
	return IstioDestinationRule{
		Namespace:        obj.GetNamespace(),
		Name:             obj.GetName(),
		Host:             host,
		SubsetNames:      subsetNames,
		HasTrafficPolicy: hasTrafficPolicy,
	}, true
}

func parseServiceEntry(obj *unstructured.Unstructured) (IstioServiceEntry, bool) {
	if obj == nil {
		return IstioServiceEntry{}, false
	}
	hosts, _, err := unstructured.NestedStringSlice(obj.Object, "spec", "hosts")
	if err != nil || len(hosts) == 0 {
		return IstioServiceEntry{}, false
	}
	return IstioServiceEntry{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
		Hosts:     hosts,
	}, true
}
