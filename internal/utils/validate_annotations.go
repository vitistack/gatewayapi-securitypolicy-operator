package utils

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// validateAnnotations checks that annotation keys and values conform to Kubernetes rules.
func ValidateAnnotations(annotations map[string]string) error {
	// `field.NewPath("metadata", "annotations")` is used to build validation error paths
	fldPath := field.NewPath("metadata", "annotations")

	// The core validator lives in `k8s.io/apimachinery/pkg/api/validation`
	errs := validation.ValidateAnnotations(annotations, fldPath)

	if len(errs) > 0 {
		return fmt.Errorf("invalid annotations: %v", errs.ToAggregate().Error())
	}
	return nil
}
