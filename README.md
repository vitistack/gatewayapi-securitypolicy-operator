# GatewayApi-SecurityPolicy-Operator

## Description
This operator enables dynamic integration of Kubernetes network policies with Gateway API resources. It monitors network policy objects and automatically applies IP-based allow/deny lists to Gateway API SecurityPolicies through annotations, ensuring network security rules are consistently enforced at the ingress layer.

**Note**: This operator currently supports `HTTPRoute`, `GRPCRoute`, and `Gateway` resources. Support for additional resources such as `TCPRoute` will be added once they reach the v1 stability channel.

**Valid Annotations**:
- `securitypolicies.vitistack.io/default-action`: Specifies default action for the security policy. Valid values: `deny` || `allow`. It defaults to `deny` if omitted.
- `securitypolicies.vitistack.io/lists`: Specifies the name of the `NetworkPolicy`. The Controller watches `networkpolicies.networking.k8s` in namespace `network-policies`. It supports multiple lists separated by comma.
- `securitypolicies.vitistack.io/addresses`: Specifies a list of CIDR blocks to be manually included, e.g., `10.20.30.40/32,172.16.12.1/32`.

## Getting Started

### Prerequisites

- Install Gateway API Stable Channel Resources for v1.4.0 in the cluster
```bash
kubectl apply --server-side -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.0/standard-install.yaml
```

- Install Envoy Gateway, a Kubernetes-native API Gateway controller that manages Envoy Proxy deployments using the Kubernetes Gateway API.
```bash
kubectl apply --server-side -f https://github.com/envoyproxy/gateway/releases/download/v1.6.0/install.yaml
```

- Network Policies
The operator watches for standard `networkpolicies.networking.k8s` resources in namespace `network-policies`.
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
