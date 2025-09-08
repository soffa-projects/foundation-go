package adapters

import (
	"reflect"
	"sync"

	log "github.com/sirupsen/logrus"
)

var (
	registry = make(map[reflect.Type]any)
	cache    = make(map[reflect.Type]any) // resolved singletons
	mu       sync.RWMutex
)

// Provide registers a provider instance for type T
func Provide[T any](provider T) {
	t := reflect.TypeOf((*T)(nil)).Elem() // reflect type for interface or struct
	mu.Lock()
	defer mu.Unlock()
	registry[t] = provider
	log.Infof("[di] component registered %s", t.String())
}

// Resolve returns the component of type T
func Resolve[T any]() T {
	t := reflect.TypeOf((*T)(nil)).Elem()

	mu.RLock()
	if c, ok := cache[t]; ok { // fast path (cached instance)
		mu.RUnlock()
		return c.(T)
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()

	// check cache again in case another goroutine populated it
	if c, ok := cache[t]; ok {
		return c.(T)
	}

	if component, ok := registry[t]; ok {
		cache[t] = component
		return component.(T)
	}

	log.Fatalf("failed to resolve component %s", t.String())
	panic("failed to resolve component " + t.String())
}

// Clear wipes out all registrations and cache
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[reflect.Type]any)
	cache = make(map[reflect.Type]any)
}
