# Bumble Grid — Floor Architecture

## Core Mental Model

Each floor answers a different question about the same system, for a different audience.

| Floor | Question | Primary Audience |
|---|---|---|
| **0** | What is running? | Platform & DevOps engineers |
| **1** | What does it do? | Product & backend engineers |
| **2** | How do we operate it? | Team leads & engineering managers |
| **3** | How does the business work? | Directors, VPs, C-suite |
| **N** | *(company-defined)* | *(company-defined)* |

The floor count is open-ended. Four floors is the minimum. Companies may define additional floors above Floor 3 to represent portfolio layers, business units, or any abstraction that serves their organizational structure.

## Bumble Grid — Floor Definitions

| Floor | Name | Description | Vocabulary | Authoring |
|---|---|---|---|---|
| **0** | Physical / Logical | The foundational infrastructure layer. Represents everything defined by machine-readable infrastructure code (k8s, Terraform, Pulumi, etc.). Nodes map directly to real, provisionable resources. | **Closed** — node types are strictly defined by the supported extractor (currently k8s). No custom node types allowed. | Automated — extracted from infra tooling. Human edits limited to style and metadata overrides. |
| **1** | System / Feature | Groups of infrastructure resources that together form a meaningful technical unit — a microservice domain, a product feature, a third-party integration. May introduce node types that have no infrastructure backing (e.g. external SaaS, workspace tools, CRMs) when those systems exist outside the company's own infrastructure. | **Open** — can reference Floor 0 nodes or introduce new node types not representable at Floor 0. | Human-authored. Floor 0 nodes are referenced, not duplicated. |
| **2** | Workflow / Process | A set of systems or features that together execute a meaningful business process. Examples: User Onboarding, Billing Cycle, Acquisition Pipeline. Nodes at this level represent process stages or capabilities, not technical components. | **Open** — composes Floor 1 nodes. New abstract node types allowed. | Human-authored. |
| **3** | Core Business | The highest level of abstraction. Represents the company's core operational domains as seen by leadership. Examples: Sales Flow, Marketing Operations, Customer Success. | **Open** — composes Floor 2 nodes. Vocabulary is entirely business-domain driven. | Human-authored. Intended audience is non-technical stakeholders. |

---

## Key Rules (to include in README)

| Rule | Description |
|---|---|
| **Composition** | Every Floor N node either references Floor N-1 nodes, or introduces a concept not expressible at lower floors. |
| **No skipping** | A Floor 2 node cannot directly reference a Floor 0 node, bypassing Floor 1. |
| **Leaf nodes** | Floor 1 and above may contain leaf nodes with no lower-floor backing. This is valid and expected for external tools and third-party services. |
| **Automation boundary** | Floor 0 is the only floor eligible for automated extraction. All other floors require human authoring. |
| **Closed vs Open** | Floor 0 has a strictly validated, closed node vocabulary. Floors 1–3 are progressively more permissive in the node types they accept. |


---

## The Traceability Chain

The central invariant of Bumble Grid is that **every node at any floor must be traceable, through references, to at least one Floor 0 node**.

```
Floor 3: "Order Fulfillment"              ← business process
    └── Floor 2: "Place an Order"         ← product capability
            └── Floor 1: "Payments API"   ← logical service
                    └── Floor 0: k8s-payments-api-deployment  ← real infrastructure
```

This chain is what separates Bumble Grid from static architecture diagrams. A VP looking at a Floor 3 node can drill all the way down to the actual running infrastructure behind it. If that chain breaks anywhere, the graph loses its value as a living document.

---

## Floor 0 — Physical / Logical Infrastructure

**What it represents:** The raw, running system. Everything that actually exists in infrastructure-as-code or can be observed in a cluster.

**Defined by:** Automated extraction from IaC sources (Kubernetes manifests, Terraform, etc.). Human authoring is restricted to `style` and `meta` overrides only. Manual edges are permitted exclusively to cover runtime relationships invisible to static manifests (e.g. cross-namespace service calls).

**Nodes:** Kubernetes resources — `Deployment`, `StatefulSet`, `Service`, `Ingress`, `CronJob`, `Secret`, `ConfigMap`, `PVC`, `Namespace`, and others. Also Terraform-provisioned resources where applicable.

**Edges:** `Routes`, `Exposes`, `Mounts`, `ScheduledBy`, `Calls`.

**Source of truth:** The actual cluster and IaC repository. If it is not there, it is not here.

**Key invariant:** Every node at Floor 1 and above must reference at least one Floor 0 node.

---

## Floor 1 — Application / Service Layer

**What it represents:** Logical software components as engineers think about them — coherent services, data stores, and integrations that teams own and develop. Not individual Kubernetes primitives, but the named things that appear in architecture docs and team wikis.

**Defined by:** Human-authored, constrained to reference Floor 0 nodes. A Floor 1 node is a named logical component that wraps or references one or more Floor 0 nodes.

**Nodes:** `Payments API`, `User Auth Service`, `Order Database`, `Notification Service`, `Email Gateway`. Third-party SaaS integrations (Stripe, SendGrid, Salesforce) also live at this floor.

**Edges:** `DependsOn`, `PublishesTo`, `ConsumesFrom`, `AuthorizedBy`.

**Node references:** A Floor 1 node references Floor 0 nodes. References are non-exclusive — a Floor 0 node (e.g. a shared auth service) may be referenced by multiple Floor 1 nodes.

**Key invariant:** Every Floor 1 node must reference at least one Floor 0 node.

---

## Floor 2 — Feature / Capability Layer

**What it represents:** Discrete product capabilities or operational sub-workflows. Things a Product Manager or Engineering Manager would place on a roadmap or a runbook. This is where business intent becomes visible for the first time.

**Defined by:** Product and engineering leadership, collaboratively. Each node describes something the system *enables a user or operator to do*, composed from Floor 1 services.

**Nodes:** `User Login Flow`, `Place an Order`, `Process a Refund`, `Send Invoice Email`, `Nightly Sales Report`, `Fraud Detection Pipeline`.

**Edges:** `Triggers`, `Requires`, `Produces`, `BlockedBy`.

**Why this floor matters operationally:** This is where incident impact becomes legible to non-engineers. Floor 0 shows what infrastructure is broken. Floor 2 shows which user-facing capabilities are affected — which is what support, operations, and leadership actually need.

**Key invariant:** Every Floor 2 node must reference at least one Floor 1 node.

---

## Floor 3 — Business Workflow Layer

**What it represents:** The company's core operational and commercial processes, expressed in terms that a non-technical executive understands immediately. Nodes at this floor map to departments, P&L lines, and KPIs.

**Defined by:** Business leadership, operations, and finance. Informed by Floor 2, but authored entirely in business language.

**Nodes:** `Customer Acquisition`, `Order Fulfillment`, `Accounts Receivable`, `Supplier Onboarding`, `Customer Support`, `Monthly Close`.

**Edges:** `Feeds`, `Triggers`, `Owns`, `ReportsTo`.

**The payoff:** A CFO can look at `Accounts Receivable` and — if they need to — drill down through Floor 2 (`Invoice Generation`, `Payment Collection`), through Floor 1 (`Payments API`, `Billing Service`), all the way to the Floor 0 Kubernetes deployment behind it. That traceability is what no org chart, Confluence page, or static architecture diagram currently provides.

**Key invariant:** Every Floor 3 node must reference at least one Floor 2 node.

---

## Floors 4 and Above — Company-Defined

Companies may define additional floors above Floor 3. Typical use cases include portfolio layers, business unit groupings, or regulatory / compliance domains. The structure, node vocabulary, and edge taxonomy at these floors are defined by the company and fall outside the BGSpec minimum specification.

The same traceability invariant applies: every node at Floor N must reference at least one node at Floor N−1.

---

## Authoring Rules Summary

| Floor | Who authors nodes | Who authors edges | Human override allowed |
|---|---|---|---|
| 0 | Automated extractor only | Extractor + manual (runtime only) | `style`, `meta` only |
| 1 | Human | Human | Full authorship |
| 2 | Human | Human | Full authorship |
| 3 | Human | Human | Full authorship |
| N | Human | Human | Full authorship |

---

## Drift and Validation

The Bumble Grid application is responsible for detecting and surfacing drift — cases where a higher-floor node references a Floor 0 node that no longer exists, or where an extracted Floor 0 update invalidates an assumption made at Floor 1 or above. The schema itself does not enforce this at write time; validation is a runtime concern handled by the application layer.