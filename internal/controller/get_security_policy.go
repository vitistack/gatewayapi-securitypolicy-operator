package controller

import (
	"context"
	"fmt"

	envoyv1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getSecurityPolicy(ctx context.Context, r client.Client, gatewayApiResource gatewayApiResource) (envoyv1.SecurityPolicy, error) {

	// Get a list of SecurityPolicies in the same namespace as the HTTPRoute
	securityPolicyList := &envoyv1.SecurityPolicyList{}
	err := r.List(ctx, securityPolicyList, client.InNamespace(gatewayApiResource.Namespace))
	if err != nil {
		return envoyv1.SecurityPolicy{}, err
	}

	// Find the SecurityPolicy that matches the HTTPRoute's name in targetRefs and append to a list
	processedSecurityPolicyList := []envoyv1.SecurityPolicy{}
	if len(securityPolicyList.Items) > 0 {
		for _, securityPolicy := range securityPolicyList.Items {
			for _, targetRef := range securityPolicy.Spec.TargetRefs {
				if string(targetRef.Name) == gatewayApiResource.Name {
					processedSecurityPolicyList = append(processedSecurityPolicyList, securityPolicy)
				}
			}
		}
	}

	// Return error if no SecurityPolicies found
	if len(processedSecurityPolicyList) == 0 {
		return envoyv1.SecurityPolicy{}, fmt.Errorf("No SecurityPolicies found for HTTPRoute %s/%s", gatewayApiResource.Namespace, gatewayApiResource.Name)
	}

	// Return SecurityPolicy if count is 1
	if len(processedSecurityPolicyList) == 1 {
		return processedSecurityPolicyList[0], nil
	}

	// Find the oldest SecurityPolicy in the same namespace as the HTTPRoute by time of creation if multiple are found
	// vitistack will only process one SecurityPolicy per HTTPRoute, and it uses the oldest one if multiple are found
	oldestSecurityPolicy := processedSecurityPolicyList[0]
	if len(processedSecurityPolicyList) > 1 {
		for _, securityPolicy := range processedSecurityPolicyList {
			if securityPolicy.CreationTimestamp.Time.Before(oldestSecurityPolicy.CreationTimestamp.Time) {
				oldestSecurityPolicy = securityPolicy
			}
		}
	}

	return oldestSecurityPolicy, nil

}
