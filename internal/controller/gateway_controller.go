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
	"reflect"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// GatewayReconciler reconciles a gateway object
type GatewayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=gateway.envoyproxy.io,resources=securitypolicies,verbs=get;list;watch;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Gateway object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	log.Info("Reconciling gateway", "Gateway.Namespace", req.Namespace, "Gateway.Name", req.Name)

	// Fetch the gateway that triggered this reconciliation
	var gateway gatewayv1.Gateway
	if err := r.Get(ctx, req.NamespacedName, &gateway); err != nil {
		log.Error(err, "unable to fetch gateway")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Define gatewayApiResource for use in get/create/update SecurityPolicy functions
	gatewayApiResource := gatewayApiResource{
		Name:      strings.ToLower(gateway.GetObjectKind().GroupVersionKind().Kind) + "-" + gateway.Name,
		Namespace: gateway.Namespace,
		Kind:      gateway.GetObjectKind().GroupVersionKind().Kind,
	}

	// Examine DeletionTimestamp to determine if object is under deletion
	if gateway.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then let's add the finalizer and update the object. This is equivalent
		// to registering our finalizer.
		if !controllerutil.ContainsFinalizer(&gateway, FinalizerSecurityPolicy) {
			log.Info("Add Finalizer", "Gateway.Namespace", req.Namespace, "Gateway.Name", req.Name)
			controllerutil.AddFinalizer(&gateway, FinalizerSecurityPolicy)
			if err := r.Update(ctx, &gateway); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		log.Info("gateway deletion in progress", "Gateway.Namespace", req.Namespace, "Gateway.Name", req.Name)
		// our finalizer is present, so let's handle any external dependency
		securityPolicy, err := getSecurityPolicy(ctx, r.Client, gatewayApiResource)
		if err == nil {
			if err := deleteSecurityPolicy(ctx, r.Client, gatewayApiResource); err != nil {
				log.Info("Failed to delete SecurityPolicy", "SecurityPolicy.Name", securityPolicy.Name)
				return ctrl.Result{}, err
			}
		}
		// remove our finalizer from the list and update it.
		controllerutil.RemoveFinalizer(&gateway, FinalizerSecurityPolicy)
		if err := r.Update(ctx, &gateway); err != nil {
			return ctrl.Result{}, err
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Get SecurityPolicy associated with this gateway
	securityPolicy, err := getSecurityPolicy(ctx, r.Client, gatewayApiResource)
	if err != nil {
		log.Info("Unable to fetch SecurityPolicy for Gateway", "Gateway.Namespace", req.Namespace, "Gateway.Name", req.Name, "Error", err)
	}

	// Create SecurityPolicy if it does not exist
	if securityPolicy.Name == "" {
		securityPolicy, err = createSecurityPolicy(ctx, r.Client, gatewayApiResource)
		if err != nil {
			log.Info("Unable to create SecurityPolicy for Gateway", "Gateway.Namespace", req.Namespace, "Gateway.Name", req.Name, "Error", err)
			return ctrl.Result{}, err
		}
		log.Info("Created SecurityPolicy for Gateway", "Gateway.Namespace", req.Namespace, "Gateway.Name", req.Name)
	}

	// Get annotations from gateway
	annotations := gateway.Annotations

	// Update SecurityPolicy based on annotations
	if err := updateSecurityPolicy(ctx, r.Client, securityPolicy, annotations); err != nil {
		log.Info("Update SecurityPolicy for Gateway", "Gateway.Namespace", req.Namespace, "Gateway.Name", req.Name, "Error", err)
		return ctrl.Result{}, err
	}

	// Create a patch to update mandatory annotations
	deepCopygateway := gateway.DeepCopy()
	gateway.Annotations[AnnotationSecurityPolicyLastUpdated] = time.Now().Format(time.RFC3339)
	gateway.Annotations[AnnotationSecurityPolicyManagedBy] = AnnotationSecurityPolicyOwner
	// Apply the patch
	if err := r.Patch(ctx, &gateway, client.MergeFrom(deepCopygateway)); err != nil {
		log.Error(err, "unable to patch gateway with mandatory annotations", "Gateway.Namespace", req.Namespace, "Gateway.Name", req.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Predicate that filters updates where only annotations changed
	annotationChangedPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			oldObjAnnotationSecurityPolicyDefaultAction := e.ObjectOld.GetAnnotations()[AnnotationSecurityPolicyDefaultAction]
			newObjAnnotationSecurityPolicyDefaultAction := e.ObjectNew.GetAnnotations()[AnnotationSecurityPolicyDefaultAction]

			oldObjAnnotationSecurityPolicyLists := e.ObjectOld.GetAnnotations()[AnnotationSecurityPolicyLists]
			newObjAnnotationSecurityPolicyLists := e.ObjectNew.GetAnnotations()[AnnotationSecurityPolicyLists]

			oldObjAnnotationSecurityPolicyAddresses := e.ObjectOld.GetAnnotations()[AnnotationSecurityPolicyAddresses]
			newObjAnnotationSecurityPolicyAddresses := e.ObjectNew.GetAnnotations()[AnnotationSecurityPolicyAddresses]

			oldObjAnnotationSecurityPolicyLastUpdated := e.ObjectOld.GetAnnotations()[AnnotationSecurityPolicyLastUpdated]
			newdObjAnnotationSecurityPolicyLastUpdated := e.ObjectNew.GetAnnotations()[AnnotationSecurityPolicyLastUpdated]

			// Trigger reconciliation if relevant annotations have changed
			return !reflect.DeepEqual(oldObjAnnotationSecurityPolicyDefaultAction, newObjAnnotationSecurityPolicyDefaultAction) ||
				!reflect.DeepEqual(oldObjAnnotationSecurityPolicyLists, newObjAnnotationSecurityPolicyLists) ||
				!reflect.DeepEqual(oldObjAnnotationSecurityPolicyAddresses, newObjAnnotationSecurityPolicyAddresses) ||
				!reflect.DeepEqual(oldObjAnnotationSecurityPolicyLastUpdated, newdObjAnnotationSecurityPolicyLastUpdated) ||
				!reflect.DeepEqual(e.ObjectOld.GetDeletionTimestamp(), e.ObjectNew.GetDeletionTimestamp())
		},
		CreateFunc: func(e event.CreateEvent) bool {
			// Trigger reconciliation if relevant annotations are present
			return e.Object.GetAnnotations()[AnnotationSecurityPolicyDefaultAction] != "" ||
				e.Object.GetAnnotations()[AnnotationSecurityPolicyLists] != "" ||
				e.Object.GetAnnotations()[AnnotationSecurityPolicyAddresses] != ""
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1.Gateway{}).
		Named("gateway").
		WithEventFilter(annotationChangedPredicate).
		Complete(r)
}
