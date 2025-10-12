package h

import (
	"encoding/json"
	"strings"

	"github.com/thoas/go-funk"
)

func TrimToNull(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func TrimToEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func IsEmpty(s interface{}) bool {
	return funk.IsEmpty(s)
}

func IsNotEmpty(s interface{}) bool {
	return !funk.IsEmpty(s)
}
func StrPtr(s string) *string {
	return &s
}
func PtrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ToMap(input string) map[string]any {
	values := map[string]any{}
	if err := json.Unmarshal([]byte(input), &values); err != nil {
		return nil
	}
	return values
}

func StrPtrToLower(s *string) *string {
	if s == nil {
		return nil
	}
	res := strings.ToLower(*s)
	return &res
}
