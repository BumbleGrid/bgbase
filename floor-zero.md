# Floor zero

**Floor zero** is the first slice of data we pull from a Kubernetes cluster when building a service-interaction diagram. It focuses on workloads and anything that defines or constrains how traffic flows between them.

## What we fetch first

### Workloads and identity

- **Pods** — what is actually running.
- **Deployments**, **StatefulSets**, **DaemonSets** — how replicas are owned and rolled.
- **Jobs** / **CronJobs** — ephemeral or scheduled actors that still call Services or APIs.

Use labels and selectors on these objects to correlate workloads with Services and policies.

### In-cluster routing

- **Services** (`ClusterIP`, `NodePort`, `LoadBalancer`) — stable DNS names and ports that workloads target.
- **Endpoints** / **EndpointSlices** — concrete backends behind a Service when that is not obvious from selectors alone.
- **Headless Services** (`clusterIP: None`) — discovery patterns typical of StatefulSets and peer meshes.

### North–south and L7 ingress

- **Ingress** — host/path rules mapping to backend Services.
- **Gateway API** resources — **`Gateway`**, **`HTTPRoute`**, **`GRPCRoute`**, etc., when the cluster uses them instead of classic Ingress.

### East–west and validation

- **NetworkPolicy** — allowed/denied pod-to-pod or namespace flows; useful to cross-check or annotate diagrams.

### Optional but high-signal (when installed)

- **Service mesh** custom resources (for example Istio `VirtualService` / `DestinationRule`, or Linkerd equivalents) — explicit routing, subsets, and dependencies.
- **ExternalName Services** — DNS aliases to systems outside the cluster; easy to omit from diagrams otherwise.

## Out of scope for floor zero (unless we extend the goal)

These matter for operations diagrams but not for a minimal **service topology** sketch:

- **ConfigMaps** / **Secrets**
- **PersistentVolumeClaims** / storage classes
- **HorizontalPodAutoscaler** / **VerticalPodAutoscaler**
- **RBAC** (**Roles**, **ClusterRoles**, **Bindings**)

We can add a later “floor” if config, storage, scaling, or auth edges become part of the documentation.


## Cytoscape shape for floor 0

---

| bgKind | Cytoscape shape | Notes |
|---|---|---|
| Workload | `roundrectangle` | Standard service/pod |
| Database | `cylinder` | Universal DB icon |
| Cache | `ellipse` | Fast/ephemeral feel |
| MessageBroker | `hexagon` | Hub/routing semantics |
| Storage | `barrel` | Persistent volume |
| Gateway | `diamond` | Entry/decision point |
| LoadBalancer | `triangle` | Directional distribution |
| ExternalService | `rectangle` + dashed border | "Outside the box" |
| ConfigSource | `roundrectangle` + dotted border | Similar to Workload but softer |
| SecretSource | `pentagon` | Distinct from ConfigSource |
| ServiceDiscovery | `ellipse` + amber border | Lookup/broadcast role |
| NetworkPolicy | `rectangle` + red/coral border | Constraint, not a service |
| JobRunner | `rhomboid` | Async/temporal slant |
| Namespace | compound parent node | Cytoscape parent grouping |
| Cluster | compound parent (top-level) | Wraps Namespaces |