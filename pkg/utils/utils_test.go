package utils

import (
	_ "github.com/lib/pq"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("utils", func() {
	Context("SliceContainsString", func() {
		It("should find a string in slice", func() {
			Expect(SliceContainsString([]string{"a"}, "a")).To(BeTrue())
		})

		It("should fail to find a string in slice", func() {
			Expect(SliceContainsString([]string{"a"}, "b")).To(BeFalse())
		})
	})
})
