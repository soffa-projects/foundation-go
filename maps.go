package micro

import "encoding/json"

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

func NewMap(input string) Map {
	values := map[string]any{}
	if err := json.Unmarshal([]byte(input), &values); err != nil {
		return Map{values: values}
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

func (m Map) Set(key string, value any) Map {
	m.values[key] = value
	return m
}
