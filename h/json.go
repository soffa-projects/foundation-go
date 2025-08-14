package h

import (
	"encoding/json"

	"github.com/tidwall/gjson"
)

type JsonValue struct {
	value string
}

func NewJsonValue(value string) JsonValue {
	return JsonValue{value: value}
}

func (j JsonValue) Get(path string) any {
	value := gjson.Get(j.value, path)
	if value.Exists() {
		return value.Value()
	}
	return nil
}

func ToJsonString(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func FromJsonString(source string, target any) error {
	err := json.Unmarshal([]byte(source), target)
	if err != nil {
		return err
	}
	return nil
}
