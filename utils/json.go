package utils

import (
	"encoding/json"
)

func ToJsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func FromJsonString(source string, target any) error {
	err := json.Unmarshal([]byte(source), target)
	if err != nil {
		return err
	}
	return nil
}
