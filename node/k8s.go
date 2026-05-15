package node

type K8sMetadata struct {
	Kind            string            `json:"kind"`
	Name            string            `json:"name"`
	Namespace       *string           `json:"namespace"`
	APIVersion      string            `json:"apiVersion,omitempty"`
	UID             string            `json:"uid,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	Replicas        *int              `json:"replicas,omitempty"`
	Image           *string           `json:"image,omitempty"`
	Resources       *K8sResources     `json:"resources,omitempty"`
	ServiceType     *string           `json:"serviceType,omitempty"`
	Ports           []K8sPort         `json:"ports,omitempty"`
	StorageClass    *string           `json:"storageClass,omitempty"`
	StorageCapacity *string           `json:"storageCapacity,omitempty"`
	Schedule        *string           `json:"schedule,omitempty"`
}

type K8sPort struct {
	Name       string `json:"name,omitempty"`
	Port       int    `json:"port"`
	TargetPort any    `json:"targetPort,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
}

type K8sResources struct {
	Requests *K8sResourceAmounts `json:"requests,omitempty"`
	Limits   *K8sResourceAmounts `json:"limits,omitempty"`
}

type K8sResourceAmounts struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}
