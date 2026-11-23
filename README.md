# GatewayApi-SecurityPolicy-Operator

## Description
This operator enables dynamic integration of IP allow/deny lists with Gateway API resources. It monitors `CidrList` and network policy objects and automatically applies IP-based allow/deny lists to Gateway API SecurityPolicies through annotations, ensuring network security rules are consistently enforced at the ingress layer.

**Note**: This operator currently supports `HTTPRoute`, `GRPCRoute`, and `Gateway` resources. Support for additional resources such as `TCPRoute` will be added once they reach the v1 stability channel.

**Valid Annotations**:
- `securitypolicies.vitistack.io/default-action`: Specifies default action for the security policy. Valid values: `deny` || `allow`. It defaults to `deny` if omitted.
- `securitypolicies.vitistack.io/lists`: Specifies the name of a `CidrList` or `NetworkPolicy`. For each name, the controller first looks for a cluster-scoped `CidrList`. If no `CidrList` is found, or the `CidrList` CRD is not installed, it falls back to the `NetworkPolicy` with that name in namespace `network-policies`. It supports multiple lists separated by comma.
- `securitypolicies.vitistack.io/addresses`: Specifies a list of IPv4/IPv6 CIDR blocks to be manually included, e.g., `10.20.30.40/32,172.16.12.0/24,2001:db8::/32`. Plain IP addresses without a prefix get `/32` (IPv4) or `/128` (IPv6) added automatically, e.g., `10.20.30.40` becomes `10.20.30.40/32`.
- `securitypolicies.vitistack.io/countries`: Specifies a comma-separated list of ISO 3166-1 alpha-2 country codes to match on the client's geolocation, e.g., `US,GB,DE`. Codes are case-insensitive; any code not recognized as a valid Envoy country is silently omitted. CIDRs and countries are matched independently (OR). Country matching relies on Envoy Gateway geolocation (GeoIP) and is supported for `HTTPRoute` and `GRPCRoute` only.

**Example**:
```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: example
  annotations:
    securitypolicies.vitistack.io/default-action: deny
    securitypolicies.vitistack.io/lists: office-networks,example-securitypolicy-list
    securitypolicies.vitistack.io/addresses: 10.20.30.40,2001:db8:1234::/48
```
- `securitypolicies.vitistack.io/countries`: Specifies a comma-separated list of ISO 3166-1 alpha-2 country codes to match on the client's geolocation, e.g., `US,GB,DE`. Codes are case-insensitive; any code not recognized as a valid Envoy country is silently omitted. CIDRs and countries are matched independently (OR). Country matching relies on Envoy Gateway geolocation (GeoIP) and is supported for `HTTPRoute` and `GRPCRoute` only.

## Getting Started

### Prerequisites

- Install Gateway API Stable Channel Resources for v1.4.0 in the cluster
```bash
kubectl apply --server-side -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.0/standard-install.yaml
```

- Install Envoy Gateway, a Kubernetes-native API Gateway controller that manages Envoy Proxy deployments using the Kubernetes Gateway API. Version `v1.9` or later is required for country-based (geolocation) matching.
```bash
kubectl apply --server-side -f https://github.com/envoyproxy/gateway/releases/download/v1.9.1/install.yaml
```

- CidrLists

`CidrList` (`securitypolicy.vitistack.io/v1alpha1`) is a cluster-scoped resource holding a list of IPv4/IPv6 CIDR blocks. The CRD is installed by the Helm chart. Each entry must be a valid CIDR without host bits set (e.g. `10.0.0.0/8`, not `10.0.0.1/8`), duplicates are rejected and the list may be empty. Use `/32` or `/128` for a single host.
```yaml
apiVersion: securitypolicy.vitistack.io/v1alpha1
kind: CidrList
metadata:
  name: office-networks
spec:
  cidrs:
    - 10.0.0.0/8
    - 192.168.10.0/24
    - 203.0.113.42/32        # single IPv4 host
    - 2001:db8::/32
    - 2001:db8:abcd::1/128   # single IPv6 host
```
When a `CidrList` is created, updated or deleted, all resources referencing it in `securitypolicies.vitistack.io/lists` are reconciled. Requires Kubernetes v1.31 or later.

`status.lastSynced` is reserved for the creator of the `CidrList`, e.g. a tool that syncs CIDRs from an external source, to record when it last synced the list. The operator never sets or reads it. As `status` is a subresource, it is ignored on create/apply and must be written with a separate status update, e.g.:
```bash
kubectl patch cidrlist office-networks --subresource=status --type=merge \
  -p "{\"status\":{\"lastSynced\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}}"
```

- Network Policies

The operator watches for standard `networkpolicies.networking.k8s` resources in namespace `network-policies`. These are used when no `CidrList` with the same name exists.
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: example-securitypolicy-list
  namespace: network-policies
spec:
  ingress:
  - from:
    - ipBlock:
        cidr: 10.20.25.0/26
    - ipBlock:
        cidr: 172.16.1.0/24
  podSelector:
    matchLabels:
      network-policies: example-securitypolicy-list
  policyTypes:
  - Ingress
```

### Cluster Deployment

**ArgoCD application definition**:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: gatewayapi-securitypolicy-operator
  namespace: argocd
spec:
  project: default
  source:
    path: .
    repoURL: oci://ncr.sky.nhn.no/ghcr/vitistack/helm/gatewayapi-securitypolicy-operator
    targetRevision: 1.*
    helm:
      valueFiles:
          - values.yaml
  destination:
    server: "https://kubernetes.default.svc"
    namespace: gatewayapi-securitypolicy-system
  syncPolicy:
      automated:
          selfHeal: true
          prune: true
      syncOptions:
      - CreateNamespace=true
```

**CRD Helm values**:
- `crd.enable` (default `true`): Installs the `CidrList` CRD. The CRD is part of the chart templates, so it is also updated on `helm upgrade`. If disabled, the `CidrList` controller is skipped and only `NetworkPolicy` lists are used.
- `crd.keep` (default `true`): Adds `helm.sh/resource-policy: keep`, so the CRD and all `CidrList` resources are kept when the release is uninstalled.

# License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
