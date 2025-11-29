package controller

const (
	NetworkPoliciesNamespace              = "network-policies"
	AnnotationSecurityPolicyDefaultAction = "securitypolicies.vitistack.io/default-action"
	AnnotationSecurityPolicyLists         = "securitypolicies.vitistack.io/lists"
	AnnotationSecurityPolicyAddresses     = "securitypolicies.vitistack.io/addresses"
	AnnotationSecurityPolicyLastUpdated   = "securitypolicies.vitistack.io/last-updated"
	AnnotationSecurityPolicyManagedBy     = "securitypolicies.vitistack.io/managed-by"
	SecurityPolicyOwner                   = "gatewayapi-securitypolicy-operator"
	AnnotationSecurityPolicyGateway       = "securitypolicies.vitistack.io/gateway"
	DefaultAPIGatewayName                 = "envoy-proxy"
	FinalizerNetworkPolicy                = "networkpolicies.vitistack.io/finalizer"
	FinalizerSecurityPolicy               = "securitypolicies.vitistack.io/finalizer"
)
