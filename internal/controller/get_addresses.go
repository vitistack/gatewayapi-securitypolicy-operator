package controller

import (
	"context"
	"fmt"

	securitypolicyv1alpha1 "github.com/vitistack/gatewayapi-securitypolicy-operator/api/v1alpha1"
	utils "github.com/vitistack/gatewayapi-securitypolicy-operator/internal/utils"
	v1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func getAddresses(ctx context.Context, r Client, securityPolicyList []string, addressList []string) ([]string, error) {

	var cidrs []string

	// Get each list as a CIDRList, falling back to a NetworkPolicy, and extract CIDRs

	for _, listName := range securityPolicyList {

		cidrListCIDRs, found, err := getCIDRListCIDRs(ctx, r, listName)
		if err != nil {
			return nil, err
		}

		if found {
			cidrs = append(cidrs, cidrListCIDRs...)
			continue
		}

		processNetworkPolicy := v1.NetworkPolicy{}

		err = r.Get(ctx, client.ObjectKey{
			Namespace: NetworkPoliciesNamespace,
			Name:      listName,
		}, &processNetworkPolicy)

		if err != nil {
			return nil, fmt.Errorf("unable to fetch NetworkPolicy %q: %w", listName, err)
		}

		// Extract CIDRs from NetworkPolicy and append to list
		cidrs = append(cidrs, extractCIDRsFromNetworkPolicy(&processNetworkPolicy, cidrs)...)
	}

	// Append valid CIDRs from customList, adding /32 or /128 to plain IP addresses
	for _, address := range addressList {
		cidr := utils.NormalizeCIDR(address)
		if utils.CheckValidCIDR(cidr) {
			cidrs = append(cidrs, cidr)
		} else {
			return nil, fmt.Errorf("Invalid CIDR %q in custom list, skipping", address)
		}
	}

	// Remove duplicates and sort
	compactSortedCIDRs := utils.SortSlice(cidrs)

	return compactSortedCIDRs, nil
}

// getCIDRListCIDRs returns the CIDRs of the cluster-scoped CIDRList with the given name.
// found is false if the CIDRList does not exist or the CIDRList CRD is not installed.
func getCIDRListCIDRs(ctx context.Context, r Client, name string) (cidrs []string, found bool, err error) {
	log := logf.FromContext(ctx)

	cidrList := securitypolicyv1alpha1.CIDRList{}

	err = r.Get(ctx, client.ObjectKey{Name: name}, &cidrList)

	switch {
	case err == nil:
		return cidrList.Spec.CIDRs, true, nil
	case meta.IsNoMatchError(err):
		log.Info("CIDRList CRD not installed, falling back to NetworkPolicy", "Name", name)
		return nil, false, nil
	case apierrors.IsNotFound(err):
		log.Info("CIDRList not found, falling back to NetworkPolicy", "Name", name)
		return nil, false, nil
	default:
		return nil, false, fmt.Errorf("unable to fetch CIDRList %q: %w", name, err)
	}
}
