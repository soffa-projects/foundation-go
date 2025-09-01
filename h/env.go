package h

import "strings"

func IsProduction(input string) bool {
	return strings.HasPrefix(strings.ToLower(input), "prod")
}
