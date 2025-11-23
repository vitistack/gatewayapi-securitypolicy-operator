/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"slices"
	"strings"

	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// NetworkPolicyReconciler reconciles a NetworkPolicy object
type NetworkPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.k8s.io/v1,resources=networkpolicies,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NetworkPolicyRoute object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *NetworkPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	log.Info("Reconciling NetworkPolicy", "NetworkPolicy.Namespace", req.Namespace, "NetworkPolicy.Name", req.Name)

	// Fetch the NetworkPolicy instance
	var networkPolicy v1.NetworkPolicy
	if err := r.Get(ctx, req.NamespacedName, &networkPolicy); err != nil {
		log.Error(err, "Failed to get NetworkPolicy")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Fetch all HttpRoutes in the cluster
	var httpRouteList gatewayv1.HTTPRouteList
	if err := r.List(ctx, &httpRouteList); err != nil {
		log.Error(err, "Failed to list HTTPRoutes")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Fetch all GRPCRoutes in the cluster
	var grpcRouteList gatewayv1.GRPCRouteList
	if err := r.List(ctx, &grpcRouteList); err != nil {
		log.Error(err, "Failed to list GRPCRoutes")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Fetch all Gateways in the cluster
	var gatewayList gatewayv1.GatewayList
	if err := r.List(ctx, &gatewayList); err != nil {
		log.Error(err, "Failed to list Gateways")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Loop through all HttpRoutes to find those that reference NetworkPolicy Name
	for _, httpRoute := range httpRouteList.Items {
		if _, ok := httpRoute.Annotations[AnnotationSecurityPolicyLists]; ok {
			// Create slice of AnnotationSecurityPolicyLists entries
			annotationLists := httpRoute.Annotations[AnnotationSecurityPolicyLists]
			listEntries := strings.Split(annotationLists, ",")
			if slices.Contains(listEntries, networkPolicy.Name) {
				// Update the HttpRoute to trigger reconciliation
				// Use a merge patch instead of full Update
				deepCopyHttpRoute := httpRoute.DeepCopy()
				httpRoute.Annotations[AnnotationSecurityPolicyLastUpdated] = ""
				if err := r.Patch(ctx, &httpRoute, client.MergeFrom(deepCopyHttpRoute)); err != nil {
					log.Error(err, "Failed to patch HttpRoute", "HttpRoute.Namespace", httpRoute.Namespace, "HttpRoute.Name", httpRoute.Name)
					return ctrl.Result{}, err
				}
				log.Info("Patched HttpRoute due to NetworkPolicy change", "HttpRoute.Namespace", httpRoute.Namespace, "HttpRoute.Name", httpRoute.Name)
			}
		}
	}

	// Loop through all GRPCRoutes to find those that reference NetworkPolicy Name
	for _, grpcRoute := range grpcRouteList.Items {
		if _, ok := grpcRoute.Annotations[AnnotationSecurityPolicyLists]; ok {
			// Create slice of AnnotationSecurityPolicyLists entries
			annotationLists := grpcRoute.Annotations[AnnotationSecurityPolicyLists]
			listEntries := strings.Split(annotationLists, ",")
			if slices.Contains(listEntries, networkPolicy.Name) {
				// Update the grpcRoute to trigger reconciliation
				// Use a merge patch instead of full Update
				deepCopyGRPCRoute := grpcRoute.DeepCopy()
				grpcRoute.Annotations[AnnotationSecurityPolicyLastUpdated] = ""
				if err := r.Patch(ctx, &grpcRoute, client.MergeFrom(deepCopyGRPCRoute)); err != nil {
					log.Error(err, "Failed to patch GRPCRoute", "GRPCRoute.Namespace", grpcRoute.Namespace, "GRPCRoute.Name", grpcRoute.Name)
					return ctrl.Result{}, err
				}
				log.Info("Patched GRPCRoute due to NetworkPolicy change", "GRPCRoute.Namespace", grpcRoute.Namespace, "GRPCRoute.Name", grpcRoute.Name)
			}
		}
	}

	// Loop through all Gateways to find those that reference NetworkPolicy Name
	for _, gateway := range gatewayList.Items {
		if _, ok := gateway.Annotations[AnnotationSecurityPolicyLists]; ok {
			// Create slice of AnnotationSecurityPolicyLists entries
			annotationLists := gateway.Annotations[AnnotationSecurityPolicyLists]
			listEntries := strings.Split(annotationLists, ",")
			if slices.Contains(listEntries, networkPolicy.Name) {
				// Update the gateway to trigger reconciliation
				// Use a merge patch instead of full Update
				deepCopyGateway := gateway.DeepCopy()
				gateway.Annotations[AnnotationSecurityPolicyLastUpdated] = ""
				if err := r.Patch(ctx, &gateway, client.MergeFrom(deepCopyGateway)); err != nil {
					log.Error(err, "Failed to patch Gateway", "Gateway.Namespace", gateway.Namespace, "Gateway.Name", gateway.Name)
					return ctrl.Result{}, err
				}
				log.Info("Patched Gateway due to NetworkPolicy change", "Gateway.Namespace", gateway.Namespace, "Gateway.Name", gateway.Name)
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Predicate that filters updates where only annotations changed
	annotationChangedPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectNew.GetNamespace() == NetworkPoliciesNamespace
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Object.GetNamespace() == NetworkPoliciesNamespace
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.NetworkPolicy{}).
		Named("networkpolicy").
		WithEventFilter(annotationChangedPredicate).
		Complete(r)
}
