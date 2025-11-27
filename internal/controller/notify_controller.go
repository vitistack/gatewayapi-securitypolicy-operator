package controller

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ControllerInterface interface {
	client.Object
}

// notifyController updates the annotation to trigger reprocessing of the controller
func notifyController(ctx context.Context, r Client, controller ControllerInterface) error {
	deepCopyObject, ok := controller.DeepCopyObject().(client.Object)
	if !ok {
		return fmt.Errorf("failed to deep copy controller object")
	}
	controller.GetAnnotations()[AnnotationSecurityPolicyLastUpdated] = ""
	if err := r.Patch(ctx, controller, client.MergeFrom(deepCopyObject)); err != nil {
		return err
	}
	return nil
}
