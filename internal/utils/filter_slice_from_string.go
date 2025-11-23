package utils

import (
	"strings"
)

func FilterSliceFromString(slice []string) []string {

	var filteredSlice []string

	for _, item := range slice {
		if item != "" {
			filteredSlice = append(filteredSlice, strings.TrimSpace(item))
		}
	}

	return filteredSlice
}
