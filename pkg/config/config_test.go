package config

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var (
		config *Config
	)

	It("has parseable default config", func() {
		var config Config
		err := json.Unmarshal([]byte(defaultConfig), &config)
		Expect(err).ToNot(HaveOccurred())
		Expect(config.LogLevel).To(Equal("INFO"))
	})

	Describe("LoadConfig", func() {
		It("loads a valid config file", func() {
			_, err := LoadConfig("./fixtures/valid.json")
			Expect(err).ToNot(HaveOccurred())
		})
		It("fails loading a invalid config file", func() {
			_, err := LoadConfig("./fixtures/invalid.json")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Validate", func() {
		BeforeEach(func() {
			var err error
			config, err = LoadConfig("./fixtures/valid.json")
			Expect(err).ToNot(HaveOccurred())
		})

		It("does not return error if all sections are valid", func() {
			err := config.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if LogLevel is not valid", func() {
			config.LogLevel = ""

			err := config.Validate()
			Expect(err).To(HaveOccurred())
		})

		It("returns error if the scheduler intervals are not valid", func() {
			config.Scheduler.InstanceRefreshInterval = 20000

			err := config.Validate()
			Expect(err).To(HaveOccurred())
		})
	})
})
