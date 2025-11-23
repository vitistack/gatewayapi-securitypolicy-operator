# GatewayApi-SecurityPolicy-Operator

## Description
The whole purpose of the operator is to make sure that k8s network-policies can be used as a dynamic input to ingress objects.

**Note**: The Ingress object reflects any changes in network policies.

**Valid Annotations**:
1. ``networking.k8s.io/whitelist-policy`` || ``networking.k8s.io/denylist-policy``
   - the value should point to the name of the ``networkpolicies.networking.k8s.io`` object from namespace ``network-policies``.
2. ``networkpolicies.networking.k8s.io/whitelist`` || ``networkpolicies.networking.k8s.io/denylist``
   - gives you the ability to add custom ip-addresses by choice in addition to applied network policies.
   - require valid prefix, f.ex ``10.0.0.1/32``.
  
  
**Note**: Both annotations supports multiple values by comma separation.

## Getting Started

### Prerequisites

Install Gateway API Stable Channel Resources for v1.4.0 in the cluster
```bash
kubectl apply --server-side -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.0/standard-install.yaml
```

### To Deploy on the cluster

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
