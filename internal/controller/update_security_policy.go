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

	// Add PolicyList and PolicyAddresses to slices
	var sliceAnnotationSecurityPolicyLists []string
	var sliceAnnotationSecurityPolicyAddresses []string

	if _, ok := annotations[AnnotationSecurityPolicyLists]; ok {
		sliceAnnotationSecurityPolicyLists = utils.FilterSliceFromString(strings.Split(annotations[AnnotationSecurityPolicyLists], ","))
	}

	if _, ok := annotations[AnnotationSecurityPolicyAddresses]; ok {
		sliceAnnotationSecurityPolicyAddresses = utils.FilterSliceFromString(strings.Split(annotations[AnnotationSecurityPolicyAddresses], ","))
	}

	// Get addresses
	cidrs := getAddresses(ctx, r, sliceAnnotationSecurityPolicyLists, sliceAnnotationSecurityPolicyAddresses)

	// Remove SecurityPolicy Rules if no CIDRs found
	if len(cidrs) == 0 {
		securitypolicy.Spec.Authorization = nil
		if err := r.Update(ctx, &securitypolicy); err != nil {
			return fmt.Errorf("failed to update SecurityPolicy: %w", err)
		}
		return fmt.Errorf("removed all security policy rules")
	}

	// Convert string slice to CIDR slice
	cidrSlice := make([]envoyv1.CIDR, len(cidrs))
	for i, cidr := range cidrs {
		cidrSlice[i] = envoyv1.CIDR(cidr)
	}

	// Add cidrs to SecurityPolicy rules
	defaultActionValue := envoyv1.AuthorizationAction(defaultAction)
	securitypolicy.Spec.Authorization = &envoyv1.Authorization{
		DefaultAction: &defaultActionValue,
		Rules: []envoyv1.AuthorizationRule{
			{
				Action: envoyv1.AuthorizationAction(ruleAction),
				Principal: envoyv1.Principal{
					ClientCIDRs: cidrSlice,
				},
			},
		},
	}

	// Update SecurityPolicy
	if err := r.Update(ctx, &securitypolicy); err != nil {
		return fmt.Errorf("failed to update SecurityPolicy: %w", err)
	}

	return nil

}
