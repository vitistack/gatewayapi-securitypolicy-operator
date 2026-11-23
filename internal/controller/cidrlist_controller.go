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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	securitypolicyv1alpha1 "github.com/vitistack/gatewayapi-securitypolicy-operator/api/v1alpha1"
	utils "github.com/vitistack/gatewayapi-securitypolicy-operator/internal/utils"
)

// CIDRListReconciler reconciles a CIDRList object
type CIDRListReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=securitypolicy.vitistack.io,resources=cidrlists,verbs=get;list;watch

// Reconcile notifies all HTTPRoutes, GRPCRoutes and Gateways that reference the
// CIDRList in their lists annotation, so their SecurityPolicy is rebuilt.
// This also runs when the CIDRList is deleted, so they fall back to NetworkPolicy.
// The CIDRList status is owned by its creator and is never written here.
func (r *CIDRListReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	log.Info("Reconciling CIDRList", "CIDRList.Name", req.Name)

	// Check if the CIDRList still exists
	var cidrList securitypolicyv1alpha1.CIDRList
	if err := r.Get(ctx, req.NamespacedName, &cidrList); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to get CIDRList")
			return ctrl.Result{}, err
		}
		log.Info("CIDRList deleted, notifying referencing resources", "CIDRList.Name", req.Name)
	}

	// Fetch all HttpRoutes in the cluster
	var httpRouteList gatewayv1.HTTPRouteList
	if err := r.List(ctx, &httpRouteList); err != nil {
		log.Error(err, "Failed to list HTTPRoutes")
		return ctrl.Result{}, err
	}

	// Fetch all GRPCRoutes in the cluster
	var grpcRouteList gatewayv1.GRPCRouteList
	if err := r.List(ctx, &grpcRouteList); err != nil {
		log.Error(err, "Failed to list GRPCRoutes")
		return ctrl.Result{}, err
	}

	// Fetch all Gateways in the cluster
	var gatewayList gatewayv1.GatewayList
	if err := r.List(ctx, &gatewayList); err != nil {
		log.Error(err, "Failed to list Gateways")
		return ctrl.Result{}, err
	}

	type referencingObject struct {
		kind string
		obj  client.Object
	}

	var objects []referencingObject
	for i := range httpRouteList.Items {
		objects = append(objects, referencingObject{"HTTPRoute", &httpRouteList.Items[i]})
	}
	for i := range grpcRouteList.Items {
		objects = append(objects, referencingObject{"GRPCRoute", &grpcRouteList.Items[i]})
	}
	for i := range gatewayList.Items {
		objects = append(objects, referencingObject{"Gateway", &gatewayList.Items[i]})
	}

	// Notify all objects that reference the CIDRList Name
	for _, o := range objects {
		annotationLists, ok := o.obj.GetAnnotations()[AnnotationSecurityPolicyLists]
		if !ok {
			continue
		}
		listEntries := utils.FilterSliceFromString(strings.Split(annotationLists, ","))
		if !slices.Contains(listEntries, req.Name) {
			continue
		}
		if err := notifyController(ctx, r.Client, o.obj); err != nil {
			log.Error(err, "Failed to notify "+o.kind, "Namespace", o.obj.GetNamespace(), "Name", o.obj.GetName())
			return ctrl.Result{}, err
		}
		log.Info("Patched "+o.kind+" due to CIDRList change", "Namespace", o.obj.GetNamespace(), "Name", o.obj.GetName())
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
// The controller is skipped if the CIDRList CRD is not installed.
func (r *CIDRListReconciler) SetupWithManager(mgr ctrl.Manager) error {
	gvk := securitypolicyv1alpha1.SchemeGroupVersion.WithKind("CIDRList")
	if _, err := mgr.GetRESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version); err != nil {
		if meta.IsNoMatchError(err) {
			mgr.GetLogger().Info("CIDRList CRD not installed, skipping CIDRList controller")
			return nil
		}
		return err
	}

	// Only react to spec changes, so status updates do not trigger a new reconcile
	return ctrl.NewControllerManagedBy(mgr).
		For(&securitypolicyv1alpha1.CIDRList{}).
		Named("cidrlist").
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
