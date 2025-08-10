package test

import (
	"github.com/go-faker/faker/v4"
)

func FakeEmail() string {
	return faker.Email()
}

func FakeName() string {
	return faker.Name()
}

func FakePassword() string {
	return faker.Password()
}

func FakeUrl() string {
	return faker.URL()
}

func FakePhone() string {
	return faker.Phonenumber()
}
