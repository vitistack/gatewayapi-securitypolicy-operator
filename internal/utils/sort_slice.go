package utils

import "slices"

func SortSlice(slice []string) []string {
	slices.Sort(slice)
	return slices.Compact(slice)
}
