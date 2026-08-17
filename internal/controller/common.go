package controller

import (
	"context"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client interface {
	Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
	Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error
	Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
}

type gatewayApiResource struct {
	Name      string
	Namespace string
	Kind      string
}

// securityPolicyName returns the kind-prefixed name used for the SecurityPolicy
// resource, keeping it distinct from the targetRef name (the real object name).
func (g gatewayApiResource) securityPolicyName() string {
	return strings.ToLower(g.Kind) + "-" + g.Name
}
