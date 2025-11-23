package controller

import (
	networkingv1 "k8s.io/api/networking/v1"
)

// extractCIDRsFromNetworkPolicy extracts all unique CIDRs from the given NetworkPolicy's ingress rules.
// It appends any new CIDRs found to the provided cidrs slice and returns the updated slice.

func extractCIDRsFromNetworkPolicy(np *networkingv1.NetworkPolicy, cidrs []string) []string {
	seen := make(map[string]struct{}, len(cidrs))
	for _, c := range cidrs {
		seen[c] = struct{}{}
	}

	for _, ingress := range np.Spec.Ingress {
		for _, from := range ingress.From {
			if from.IPBlock != nil && from.IPBlock.CIDR != "" {
				c := from.IPBlock.CIDR
				if _, exists := seen[c]; !exists {
					cidrs = append(cidrs, c)
					seen[c] = struct{}{}
				}
			}
		}
	}

	return cidrs
}
