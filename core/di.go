package f

import (
	"reflect"

	log "github.com/sirupsen/logrus"
)

type Component interface{}

var registry = make(map[string]any)

func Register(name string, provider interface{}) {
	registry[name] = provider
}

func Resolve[T Component](typ T) *T {
	rtype := reflect.TypeOf(typ)
	for _, component := range registry {
		cr := reflect.TypeOf(component)
		if cr == rtype {
			return component.(*T)
		}
		if cr.Kind() == reflect.Ptr && cr.Elem() == rtype {
			return component.(*T)
		}
	}
	log.Fatalf("failed to resolve component %v", typ)
	return nil
}

func ResolveByName[T interface{}](name string) T {
	if component, ok := registry[name]; ok {
		return component.(T)
	}
	log.Fatalf("failed to resolve component %s", name)
	panic("failed to resolve component")
}

func Clear() {
	registry = make(map[string]any)
}
