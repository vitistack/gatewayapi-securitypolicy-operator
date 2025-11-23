package controller

import (
	"context"

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
