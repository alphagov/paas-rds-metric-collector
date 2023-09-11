package utils

import (
	"time"

	_ "github.com/lib/pq"

	. "github.com/onsi/ginkgo/v2"
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
	Context("WithTimeout", func() {
		It("runs a function and return false if does not timeout", func() {
			executed := false
			startTime := time.Now()
			timeout := 1 * time.Second
			ret := WithTimeout(timeout, func() { executed = true })
			endTime := time.Now()

			Expect(startTime.Add(timeout)).To(BeTemporally(">", endTime))
			Expect(executed).To(BeTrue())
			Expect(ret).To(BeFalse())
		})

		It("runs a function and return true if does timeout", func() {
			executed := false
			startTime := time.Now()
			timeout := 1 * time.Second

			stopIt := make(chan bool, 1)
			defer func() { stopIt <- true }()

			ret := WithTimeout(timeout, func() { executed = true; <-stopIt })
			endTime := time.Now()

			Expect(startTime.Add(timeout)).To(BeTemporally("~", endTime, 100*time.Millisecond))
			Expect(executed).To(BeTrue())
			Expect(ret).To(BeTrue())
		})
	})

	Context("RandomString", func() {
		It("Returns random strings", func() {
			str1 := RandomString(10)
			str2 := RandomString(10)
			Expect(str1).ToNot(BeEmpty())
			Expect(str2).ToNot(BeEmpty())
			Expect(str1).ToNot(Equal(str2))
		})
	})
})
