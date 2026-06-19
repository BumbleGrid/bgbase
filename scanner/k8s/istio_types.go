package k8s

type IstioVirtualService struct {
	Namespace string
	Name      string
	Hosts     []string
	HTTP      []IstioHTTPRoute
}

type IstioHTTPRoute struct {
	MatchPrefix  *string
	Destinations []string
}

type IstioDestinationRule struct {
	Namespace        string
	Name             string
	Host             string
	SubsetNames      []string
	HasTrafficPolicy bool
}

type IstioServiceEntry struct {
	Namespace string
	Name      string
	Hosts     []string
}
