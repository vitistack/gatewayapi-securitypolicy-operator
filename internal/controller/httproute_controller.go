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

// HTTPRouteReconciler reconciles a HTTPRoute object
type HTTPRouteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=gateway.networking.k8s.io/v1,resources=httproutes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=gateway.envoyproxy.io/v1alpha1,resources=securitypolicies,verbs=get;list;watch;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io/v1,resources=networkpolicies,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HTTPRoute object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *HTTPRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	log.Info("Reconciling HttpRoute", "HttpRoute.Namespace", req.Namespace, "HttpRoute.Name", req.Name)

	// Fetch the HTTPRoute that triggered this reconciliation
	var httproute gatewayv1.HTTPRoute
	if err := r.Get(ctx, req.NamespacedName, &httproute); err != nil {
		log.Error(err, "unable to fetch HTTPRoute")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Define gatewayApiResource for use in get/create/update SecurityPolicy functions
	gatewayApiResource := gatewayApiResource{
		Name:      strings.ToLower(httproute.GetObjectKind().GroupVersionKind().Kind) + "-" + httproute.Name,
		Namespace: httproute.Namespace,
		Kind:      httproute.GetObjectKind().GroupVersionKind().Kind,
	}

	// Examine DeletionTimestamp to determine if object is under deletion
	if httproute.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then let's add the finalizer and update the object. This is equivalent
		// to registering our finalizer.
		if !controllerutil.ContainsFinalizer(&httproute, FinalizerSecurityPolicy) {
			log.Info("Add Finalizer", "HttpRoute.Namespace", req.Namespace, "HttpRoute.Name", req.Name)
			controllerutil.AddFinalizer(&httproute, FinalizerSecurityPolicy)
			if err := r.Update(ctx, &httproute); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		log.Info("HTTPRoute deletion in progress", "HttpRoute.Namespace", req.Namespace, "HttpRoute.Name", req.Name)
		// our finalizer is present, so let's handle any external dependency
		securityPolicy, err := getSecurityPolicy(ctx, r.Client, gatewayApiResource)
		if err == nil {
			if err := deleteSecurityPolicy(ctx, r.Client, gatewayApiResource); err != nil {
				log.Info("Failed to delete SecurityPolicy", "SecurityPolicy.Name", securityPolicy.Name)
				return ctrl.Result{}, err
			}
		}
		// remove our finalizer from the list and update it.
		controllerutil.RemoveFinalizer(&httproute, FinalizerSecurityPolicy)
		if err := r.Update(ctx, &httproute); err != nil {
			return ctrl.Result{}, err
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Get SecurityPolicy associated with this HTTPRoute
	securityPolicy, err := getSecurityPolicy(ctx, r.Client, gatewayApiResource)
	if err != nil {
		log.Info("Unable to fetch SecurityPolicy for HTTPRoute", "HttpRoute.Namespace", req.Namespace, "HttpRoute.Name", req.Name, "Error", err)
	}

	// Create SecurityPolicy if it does not exist
	if securityPolicy.Name == "" {
		securityPolicy, err = createSecurityPolicy(ctx, r.Client, gatewayApiResource)
		if err != nil {
			log.Info("Unable to create SecurityPolicy for HTTPRoute", "HttpRoute.Namespace", req.Namespace, "HttpRoute.Name", req.Name, "Error", err)
			return ctrl.Result{}, err
		}
		log.Info("Created SecurityPolicy for HTTPRoute", "HttpRoute.Namespace", req.Namespace, "HttpRoute.Name", req.Name)
	}

	// Get annotations from HTTPRoute
	annotations := httproute.Annotations

	// Update SecurityPolicy based on annotations
	if err := updateSecurityPolicy(ctx, r.Client, securityPolicy, annotations); err != nil {
		log.Info("Update SecurityPolicy for HTTPRoute", "HttpRoute.Namespace", req.Namespace, "HttpRoute.Name", req.Name, "Error", err)
		return ctrl.Result{}, err
	}

	// Create a patch to update mandatory annotations
	deepCopyHttpRoute := httproute.DeepCopy()
	httproute.Annotations[AnnotationSecurityPolicyLastUpdated] = time.Now().Format(time.RFC3339)
	httproute.Annotations[AnnotationSecurityPolicyManagedBy] = AnnotationSecurityPolicyOwner
	// Apply the patch
	if err := r.Patch(ctx, &httproute, client.MergeFrom(deepCopyHttpRoute)); err != nil {
		log.Error(err, "unable to patch HTTPRoute with mandatory annotations", "HttpRoute.Namespace", req.Namespace, "HttpRoute.Name", req.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HTTPRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {

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
		For(&gatewayv1.HTTPRoute{}).
		Named("httproute").
		WithEventFilter(annotationChangedPredicate).
		Complete(r)
}
