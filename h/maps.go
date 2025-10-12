package h

import (
	"encoding/json"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	"github.com/soffa-projects/foundation-go/log"
)

func NonEmptyValuesMaps(input map[string]any) map[string]any {
	values := map[string]any{}
	for key, value := range input {
		if value != nil && value != "" {
			values[key] = value
		}
	}
	return values
}

type Map struct {
	values map[string]any
}

func NewMapWithValues(values map[string]any) Map {
	return Map{values: values}
}

func DecodeMap(input any, output any) error {
	return mapstructure.Decode(input, output)
}

func NewMap(input string) Map {
	var values map[string]any
	if err := json.Unmarshal([]byte(input), &values); err != nil {
		log.Error("failed to unmarshal map: %s", err)
	}
	return Map{values: values}
}

func (m Map) Has(key string) bool {
	_, ok := m.values[key]
	return ok
}

func (m Map) Get(key string) any {
	if value, ok := m.values[key]; ok {
		return value
	}
	return nil
}

func (m Map) GetString(key string) string {
	if value, ok := m.values[key]; ok {
		return value.(string)
	}
	return ""
}

func (m Map) GetBool(key string) bool {
	if value, ok := m.values[key]; ok {
		return value.(bool)
	}
	return false
}

func (m Map) GetInt(key string) int {
	if value, ok := m.values[key]; ok {
		return value.(int)
	}
	return 0
}

func (m Map) Set(key string, value any) Map {
	m.values[key] = value
	return m
}

func IsMap(value any) bool {
	switch value.(type) {
	case map[string]string, map[string]any, map[int]string:
		return true
	default:
		if reflect.TypeOf(value).Kind() == reflect.Map {
			return true
		}
		return false
	}
}
