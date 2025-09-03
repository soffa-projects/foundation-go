package h

import "github.com/lestrrat-go/jwx/v3/jwt"

func GetClaimValues(token jwt.Token, keys ...string) []string {
	values := []string{}
	for _, key := range keys {
		var value string
		_ = token.Get(key, &value)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
