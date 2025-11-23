package controller

import (
	"context"

	utils "github.com/vitistack/gatewayapi-securitypolicy-operator/internal/utils"
	v1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func getAddresses(ctx context.Context, r Client, securityPolicyList []string, addressList []string) []string {
	log := logf.FromContext(ctx)

	var cidrs []string

	// Get each NetworkPolicy and extract CIDRs

	for _, networkPolicy := range securityPolicyList {

		processNetworkPolicy := v1.NetworkPolicy{}

		err := r.Get(ctx, client.ObjectKey{
			Namespace: NetworkPoliciesNamespace,
			Name:      networkPolicy,
		}, &processNetworkPolicy)

		if err != nil {
			log.Error(err, "Unable to fetch NetworkPolicy", "NetworkPolicy.Name", networkPolicy)
			continue
		}

		// Extract CIDRs from NetworkPolicy and append to list
		cidrs = append(cidrs, extractCIDRsFromNetworkPolicy(&processNetworkPolicy, cidrs)...)
	}

	// Append valid CIDRs from customList
	for _, cidr := range addressList {
		if utils.CheckValidCIDR(cidr) {
			cidrs = append(cidrs, cidr)
		}
	}

	// Remove duplicates and sort
	compactSortedCIDRs := utils.SortSlice(cidrs)

	return compactSortedCIDRs
}
