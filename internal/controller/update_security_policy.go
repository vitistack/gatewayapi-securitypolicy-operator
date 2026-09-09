package controller

import (
	"context"
	"fmt"
	"strings"

	envoyv1 "github.com/envoyproxy/gateway/api/v1alpha1"

	utils "github.com/vitistack/gatewayapi-securitypolicy-operator/internal/utils"
)

func updateSecurityPolicy(ctx context.Context, r Client, securitypolicy envoyv1.SecurityPolicy, annotations map[string]string) error {

	// Declare variables
	var defaultAction string
	var ruleAction string

	// Check if defaultAction is a valid value
	if _, ok := annotations[AnnotationSecurityPolicyDefaultAction]; ok {
		switch annotations[AnnotationSecurityPolicyDefaultAction] {
		case "allow":
			defaultAction = string(envoyv1.AuthorizationActionAllow)
		case "deny":
			defaultAction = string(envoyv1.AuthorizationActionDeny)
		default:
			return fmt.Errorf("defaultAction not valid. Valid values: %s || %s", "allow", "deny")
		}
	}

	// Set defaultAction if not present in annotations
	if _, ok := annotations[AnnotationSecurityPolicyDefaultAction]; !ok {
		defaultAction = string(envoyv1.AuthorizationActionDeny)
	}

	// set ruleAction opposite of defaultAction
	if defaultAction == string(envoyv1.AuthorizationActionAllow) {
		ruleAction = string(envoyv1.AuthorizationActionDeny)
	} else {
		ruleAction = string(envoyv1.AuthorizationActionAllow)
	}

	// Add PolicyList, PolicyAddresses, and PolicyCountries to slices
	var sliceAnnotationSecurityPolicyLists []string
	var sliceAnnotationSecurityPolicyAddresses []string
	var sliceAnnotationSecurityPolicyCountries []string

	if _, ok := annotations[AnnotationSecurityPolicyLists]; ok {
		sliceAnnotationSecurityPolicyLists = utils.FilterSliceFromString(strings.Split(annotations[AnnotationSecurityPolicyLists], ","))
	}

	if _, ok := annotations[AnnotationSecurityPolicyAddresses]; ok {
		sliceAnnotationSecurityPolicyAddresses = utils.FilterSliceFromString(strings.Split(annotations[AnnotationSecurityPolicyAddresses], ","))
	}

	if _, ok := annotations[AnnotationSecurityPolicyCountries]; ok {
		sliceAnnotationSecurityPolicyCountries = utils.FilterSliceFromString(strings.Split(annotations[AnnotationSecurityPolicyCountries], ","))
		sliceAnnotationSecurityPolicyCountries = filterValidCountries(sliceAnnotationSecurityPolicyCountries)
	}

	// Get addresses
	cidrs, err := getAddresses(ctx, r, sliceAnnotationSecurityPolicyLists, sliceAnnotationSecurityPolicyAddresses)
	if err != nil {
		return err
	}

	// Remove SecurityPolicy Rules if no CIDRs and Countries found
	if len(cidrs) == 0 && len(sliceAnnotationSecurityPolicyCountries) == 0 {
		defaultActionValue := envoyv1.AuthorizationAction(defaultAction)
		securitypolicy.Spec.Authorization = &envoyv1.Authorization{
			DefaultAction: &defaultActionValue,
			Rules:         []envoyv1.AuthorizationRule{},
		}
		if err := r.Update(ctx, &securitypolicy); err != nil {
			return fmt.Errorf("failed to update SecurityPolicy: %w", err)
		}
		return nil
	}

	// Convert string slice to CIDR slice
	cidrSlice := make([]envoyv1.CIDR, len(cidrs))
	for i, cidr := range cidrs {
		cidrSlice[i] = envoyv1.CIDR(cidr)
	}

	// Build SecurityPolicy rules. CIDRs and countries are matched independently
	// (OR), so each principal type gets its own rule.
	rules := []envoyv1.AuthorizationRule{}

	if len(cidrSlice) > 0 {
		rules = append(rules, envoyv1.AuthorizationRule{
			Action: envoyv1.AuthorizationAction(ruleAction),
			Principal: &envoyv1.Principal{
				ClientCIDRs: cidrSlice,
			},
		})
	}

	if len(sliceAnnotationSecurityPolicyCountries) > 0 {
		geoLocations := make([]envoyv1.ClientIPGeoLocation, len(sliceAnnotationSecurityPolicyCountries))
		for i, country := range sliceAnnotationSecurityPolicyCountries {
			geoLocations[i] = envoyv1.ClientIPGeoLocation{
				Country: &country,
			}
		}
		rules = append(rules, envoyv1.AuthorizationRule{
			Action: envoyv1.AuthorizationAction(ruleAction),
			Principal: &envoyv1.Principal{
				ClientIPGeoLocations: geoLocations,
			},
		})
	}

	defaultActionValue := envoyv1.AuthorizationAction(defaultAction)
	securitypolicy.Spec.Authorization = &envoyv1.Authorization{
		DefaultAction: &defaultActionValue,
		Rules:         rules,
	}

	// Update SecurityPolicy
	if err := r.Update(ctx, &securitypolicy); err != nil {
		return fmt.Errorf("failed to update SecurityPolicy: %w", err)
	}

	return nil

}
