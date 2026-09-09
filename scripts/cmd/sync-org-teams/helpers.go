package main

import (
	"sort"
	"strings"
)

// lowerSet returns the set of lower-cased handles for case-insensitive
// comparison.
func lowerSet(handles []string) map[string]bool {
	s := make(map[string]bool, len(handles))
	for _, h := range handles {
		s[strings.ToLower(h)] = true
	}
	return s
}

// diff returns the sorted keys present in a but not in b.
func diff(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
