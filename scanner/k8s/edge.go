package k8s

type EdgeResolver struct {
	istioLister IstioCRDLister
	logDebug    func(format string, args ...any)
	logWarn     func(format string, args ...any)
}

type EdgeResolverOption func(*EdgeResolver)

func EdgeResolverWithIstioLister(lister IstioCRDLister) EdgeResolverOption {
	return func(resolver *EdgeResolver) {
		resolver.istioLister = lister
	}
}

func EdgeResolverWithLogger(logDebug, logWarn func(format string, args ...any)) EdgeResolverOption {
	return func(resolver *EdgeResolver) {
		resolver.logDebug = logDebug
		resolver.logWarn = logWarn
	}
}

func NewEdgeResolver(opts ...EdgeResolverOption) *EdgeResolver {
	resolver := &EdgeResolver{}
	for idx := range opts {
		opts[idx](resolver)
	}
	return resolver
}

func (resolver *EdgeResolver) debugf(format string, args ...any) {
	if resolver != nil && resolver.logDebug != nil {
		resolver.logDebug(format, args...)
	}
}

func (resolver *EdgeResolver) warnf(format string, args ...any) {
	if resolver != nil && resolver.logWarn != nil {
		resolver.logWarn(format, args...)
	}
}
