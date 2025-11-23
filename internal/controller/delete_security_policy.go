package controller

import (
	"context"

	envoyv1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func deleteSecurityPolicy(ctx context.Context, r client.Client, gatewayApiResource gatewayApiResource) error {

	// Get a list of SecurityPolicies in the same namespace as the HTTPRoute
	securityPolicyList := &envoyv1.SecurityPolicyList{}
	err := r.List(ctx, securityPolicyList, client.InNamespace(gatewayApiResource.Namespace))
	if err != nil {
		return err
	}

	// Find the SecurityPolicy that matches the HTTPRoute's name in targetRefs and append to a list
	filterSecurityPolicyList := []envoyv1.SecurityPolicy{}
	if len(securityPolicyList.Items) > 0 {
		for _, securityPolicy := range securityPolicyList.Items {
			for _, targetRef := range securityPolicy.Spec.TargetRefs {
				if string(targetRef.Name) == gatewayApiResource.Name {
					filterSecurityPolicyList = append(filterSecurityPolicyList, securityPolicy)
				}
			}
		}
	}

	// Return if no SecurityPolicies found
	if len(filterSecurityPolicyList) == 0 {
		return nil
	}

	// Delete all SecurityPolicies that match the HTTPRoute`s name in targetRefes if length is 1
	if len(filterSecurityPolicyList) > 0 {
		for _, securityPolicy := range filterSecurityPolicyList {
			if len(securityPolicy.Spec.TargetRefs) == 1 {
				if err := r.Delete(ctx, &securityPolicy); err != nil {
					return err
				}
			} else {
				// Remove TargetRef that matches the HTTPRoute's name
				newTargetRefs := []gatewayv1.LocalPolicyTargetReferenceWithSectionName{}
				for _, targetRef := range securityPolicy.Spec.TargetRefs {
					if string(targetRef.Name) != gatewayApiResource.Name {
						newTargetRefs = append(newTargetRefs, targetRef)
					}
				}
				securityPolicy.Spec.TargetRefs = newTargetRefs
				// Update the SecurityPolicy
				if err := r.Update(ctx, &securityPolicy); err != nil {
					return err
				}
			}
		}
	}

	// Return the created SecurityPolicy
	return nil

}
