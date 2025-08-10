package test

import (
	"testing"

	"github.com/onsi/gomega"
)

type Assertions struct {
	internal *gomega.WithT
}

func NewAssertions(t *testing.T) Assertions {
	return Assertions{internal: gomega.NewWithT(t)}
}

func (a Assertions) Nil(err error, msg ...any) {
	a.internal.Expect(err).To(gomega.BeNil(), msg...)
}

func (a Assertions) NotNil(values ...any) {
	for _, value := range values {
		a.internal.Expect(value).To(gomega.Not(gomega.BeNil()))
	}
}

func (a Assertions) NotEmpty(value string) {
	a.internal.Expect(value).To(gomega.Not(gomega.BeEmpty()))
}

func (a Assertions) True(value bool) {
	a.internal.Expect(value).To(gomega.BeTrue())
}

func (a Assertions) False(value bool) {
	a.internal.Expect(value).To(gomega.BeFalse())
}

func (a Assertions) Equals(value any, expected any) {
	a.internal.Expect(value).To(gomega.Equal(expected))
}

func (a Assertions) MatchJson(value string, pattern string) {
	a.internal.Expect(value).To(gomega.MatchJSON(pattern))
}
