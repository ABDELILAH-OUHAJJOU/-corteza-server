package crm

import (
	"strconv"
)

// parseInt64 parses an string to int64
func parseInt64(s string) int64 {
	if s == "" {
		return 0
	}
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

// parseUInt64 parses an string to uint64
func parseUInt64(s string) uint64 {
	if s == "" {
		return 0
	}
	i, _ := strconv.ParseUint(s, 10, 64)
	return i
}

// is checks if string s is contained in matches
func is(s string, matches ...string) bool {
	for _, v := range matches {
		if s == v {
			return true
		}
	}
	return false
}
