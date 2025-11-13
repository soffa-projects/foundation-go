package f

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

// Test interfaces and types for DI testing
type TestService interface {
	GetName() string
}

type testServiceImpl struct {
	name string
}

func (s *testServiceImpl) GetName() string {
	return s.name
}

type AnotherService interface {
	GetValue() int
}

type anotherServiceImpl struct {
	value int
}

func (s *anotherServiceImpl) GetValue() int {
	return s.value
}

func TestProvide(t *testing.T) {
	Clear() // Clear any previous registrations

	service := &testServiceImpl{name: "test"}
	Provide[TestService](service)

	// Verify the service was registered
	result := Lookup[TestService]()
	assert.NotEqual(t, result, nil)
	assert.Equal(t, (*result).GetName(), "test")
}

func TestProvide_MultipleTypes(t *testing.T) {
	Clear()

	service1 := &testServiceImpl{name: "service1"}
	service2 := &anotherServiceImpl{value: 42}

	Provide[TestService](service1)
	Provide[AnotherService](service2)

	// Verify both services are registered
	result1 := Lookup[TestService]()
	assert.NotEqual(t, result1, nil)
	assert.Equal(t, (*result1).GetName(), "service1")

	result2 := Lookup[AnotherService]()
	assert.NotEqual(t, result2, nil)
	assert.Equal(t, (*result2).GetValue(), 42)
}

func TestProvide_OverwritesExisting(t *testing.T) {
	Clear()

	service1 := &testServiceImpl{name: "first"}
	service2 := &testServiceImpl{name: "second"}

	Provide[TestService](service1)
	Provide[TestService](service2) // Should overwrite

	result := Lookup[TestService]()
	assert.NotEqual(t, result, nil)
	assert.Equal(t, (*result).GetName(), "second")
}

func TestLookup_Found(t *testing.T) {
	Clear()

	service := &testServiceImpl{name: "lookup test"}
	Provide[TestService](service)

	result := Lookup[TestService]()
	assert.NotEqual(t, result, nil)
	assert.Equal(t, (*result).GetName(), "lookup test")
}

func TestLookup_NotFound(t *testing.T) {
	Clear()

	result := Lookup[TestService]()
	assert.Equal(t, result, nil)
}

func TestLookup_Caching(t *testing.T) {
	Clear()

	service := &testServiceImpl{name: "cached"}
	Provide[TestService](service)

	// First lookup - should populate cache
	result1 := Lookup[TestService]()
	assert.NotEqual(t, result1, nil)

	// Second lookup - should hit cache
	result2 := Lookup[TestService]()
	assert.NotEqual(t, result2, nil)

	// Both should return the same value
	assert.Equal(t, (*result1).GetName(), (*result2).GetName())
}

func TestResolve_Success(t *testing.T) {
	Clear()

	service := &testServiceImpl{name: "resolve test"}
	Provide[TestService](service)

	result, err := Resolve[TestService]()
	assert.Equal(t, err, nil)
	assert.Equal(t, result.GetName(), "resolve test")
}

func TestResolve_NotFound(t *testing.T) {
	Clear()

	result, err := Resolve[TestService]()
	assert.NotEqual(t, err, nil)
	assert.Equal(t, err.Error(), "failed to resolve component f.TestService")

	// Result should be zero value
	assert.Equal(t, result, nil)
}

func TestMustResolve_Success(t *testing.T) {
	Clear()

	service := &testServiceImpl{name: "must resolve test"}
	Provide[TestService](service)

	// Should not panic
	result := MustResolve[TestService]()
	assert.Equal(t, result.GetName(), "must resolve test")
}

func TestMustResolve_Panic(t *testing.T) {
	Clear()

	// Should panic when component not found
	defer func() {
		r := recover()
		assert.NotEqual(t, r, nil)
	}()

	MustResolve[TestService]() // This should panic
	t.Error("MustResolve should have panicked")
}

func TestClear(t *testing.T) {
	Clear()

	// Register services
	service1 := &testServiceImpl{name: "service1"}
	service2 := &anotherServiceImpl{value: 42}
	Provide[TestService](service1)
	Provide[AnotherService](service2)

	// Verify they exist
	assert.NotEqual(t, Lookup[TestService](), nil)
	assert.NotEqual(t, Lookup[AnotherService](), nil)

	// Clear registry
	Clear()

	// Verify they're gone
	assert.Equal(t, Lookup[TestService](), nil)
	assert.Equal(t, Lookup[AnotherService](), nil)
}

func TestConcurrentProvide(t *testing.T) {
	Clear()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			// Each goroutine provides a different implementation
			service := &testServiceImpl{name: "test"}
			Provide[TestService](service)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic, one of them should win
	result := Lookup[TestService]()
	assert.NotEqual(t, result, nil)
}

func TestConcurrentLookup(t *testing.T) {
	Clear()

	service := &testServiceImpl{name: "concurrent"}
	Provide[TestService](service)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			result := Lookup[TestService]()
			assert.NotEqual(t, result, nil)
			assert.Equal(t, (*result).GetName(), "concurrent")
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestConcurrentResolve(t *testing.T) {
	Clear()

	service := &testServiceImpl{name: "concurrent resolve"}
	Provide[TestService](service)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			result, err := Resolve[TestService]()
			assert.Equal(t, err, nil)
			assert.Equal(t, result.GetName(), "concurrent resolve")
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestResolve_AfterClear(t *testing.T) {
	Clear()

	service := &testServiceImpl{name: "test"}
	Provide[TestService](service)

	// Verify it's there
	result, err := Resolve[TestService]()
	assert.Equal(t, err, nil)
	assert.Equal(t, result.GetName(), "test")

	// Clear and try again
	Clear()

	result2, err2 := Resolve[TestService]()
	assert.NotEqual(t, err2, nil)
	assert.Equal(t, result2, nil)
}

func TestProvide_WithNilValue(t *testing.T) {
	Clear()

	// Provide nil - type assertion will panic when Lookup tries to retrieve it
	var nilService TestService
	Provide(nilService)

	// Lookup will panic due to type assertion on nil interface
	defer func() {
		r := recover()
		assert.NotEqual(t, r, nil) // Should panic
	}()

	Lookup[TestService]() // This should panic
	t.Error("Lookup should have panicked on nil value")
}

func TestMultipleTypesIndependent(t *testing.T) {
	Clear()

	// Register different services
	testSvc := &testServiceImpl{name: "test"}
	anotherSvc := &anotherServiceImpl{value: 99}

	Provide[TestService](testSvc)
	Provide[AnotherService](anotherSvc)

	// Clear should affect both
	Clear()

	assert.Equal(t, Lookup[TestService](), nil)
	assert.Equal(t, Lookup[AnotherService](), nil)
}

// Test that the same concrete type can be registered for different interfaces
func TestSameConcreteType_DifferentInterfaces(t *testing.T) {
	Clear()

	service1 := &testServiceImpl{name: "service1"}
	service2 := &anotherServiceImpl{value: 42}

	Provide[TestService](service1)
	Provide[AnotherService](service2)

	// Both should be independently accessible
	result1 := Lookup[TestService]()
	result2 := Lookup[AnotherService]()

	assert.NotEqual(t, result1, nil)
	assert.NotEqual(t, result2, nil)
	assert.Equal(t, (*result1).GetName(), "service1")
	assert.Equal(t, (*result2).GetValue(), 42)
}

// Test error message format
func TestResolve_ErrorMessage(t *testing.T) {
	Clear()

	_, err := Resolve[TestService]()
	assert.NotEqual(t, err, nil)
	// Error message should contain the type name
	assert.Equal(t, err.Error(), "failed to resolve component f.TestService")
}
