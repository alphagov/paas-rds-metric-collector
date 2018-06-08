package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPaasPostgresMetricCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RDS Metric Collector Suite")
}
