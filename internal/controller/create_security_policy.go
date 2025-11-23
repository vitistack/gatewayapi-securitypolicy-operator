package controller

import (
	"context"
	"time"

	envoyv1 "github.com/envoyproxy/gateway/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func createSecurityPolicy(ctx context.Context, r client.Client, gatewayApiResource gatewayApiResource) (envoyv1.SecurityPolicy, error) {

	// Create a new targetRef for the SecurityPolicy
	targetRefs := []gatewayv1.LocalPolicyTargetReferenceWithSectionName{
		{
			LocalPolicyTargetReference: gatewayv1.LocalPolicyTargetReference{
				Group: gatewayv1.Group("gateway.networking.k8s.io"),
				Kind:  gatewayv1.Kind(gatewayApiResource.Kind),
				Name:  gatewayv1.ObjectName(gatewayApiResource.Name),
			},
		},
	}

	// Check if SecurityPolicy already exists with the given name and overwrite TargetRefs
	var existingSecurityPolicy envoyv1.SecurityPolicy
	err := r.Get(ctx, client.ObjectKey{Name: gatewayApiResource.Name, Namespace: gatewayApiResource.Namespace}, &existingSecurityPolicy)
	if err == nil {
		// Overwrite TargetRefs if SecurityPolicy already exists
		existingSecurityPolicy.Spec.TargetRefs = targetRefs
		if err := r.Update(ctx, &existingSecurityPolicy); err != nil {
			return envoyv1.SecurityPolicy{}, err
		}
		return existingSecurityPolicy, nil
	}

	// Create a SecurityPolicy object
	securityPolicy := envoyv1.SecurityPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayApiResource.Name,
			Namespace: gatewayApiResource.Namespace,
		},
		Spec: envoyv1.SecurityPolicySpec{
			PolicyTargetReferences: envoyv1.PolicyTargetReferences{
				TargetRefs: targetRefs,
			},
		},
	}

	// Create the SecurityPolicy in the cluster
	if err := r.Create(ctx, &securityPolicy); err != nil {
		return envoyv1.SecurityPolicy{}, err
	}

	// Short Sleep to ensure the resource is created before returning
	time.Sleep(2 * time.Second)

	// Return the created SecurityPolicy
	return securityPolicy, nil

}
