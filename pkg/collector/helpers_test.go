package collector

import (
	"database/sql"
	"fmt"
	"regexp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
)

func openMultipleDBConns(count int, driver, url string) func() {
	var dbConns []*sql.DB
	success := false

	closeDBConns := func() {
		for _, c := range dbConns {
			c.Close()
		}
	}
	defer func() {
		if success != true {
			closeDBConns()
		}
	}()

	for i := 0; i < count; i++ {
		dbConn, err := sql.Open(driver, url)
		Expect(err).ToNot(HaveOccurred())
		err = dbConn.Ping()
		Expect(err).ToNot(HaveOccurred())
		dbConns = append(dbConns, dbConn)
	}
	success = true
	return closeDBConns
}

func getMetricByKey(collectedMetrics []metrics.Metric, key string) *metrics.Metric {
	for _, metric := range collectedMetrics {
		if metric.Key == key {
			return &metric
		}
	}
	return nil
}

// Replaces the DB name in a postgres DB connection string
func injectDBName(connectionString, newDBName string) string {
	re := regexp.MustCompile("(.*:[0-9()]+)[^?]*([?].*)?$")
	return re.ReplaceAllString(connectionString, fmt.Sprintf("$1/%s$2", newDBName))
}

var _ = Describe("injectDBName", func() {
	It("replaces the db name", func() {
		Expect(
			injectDBName("postgresql://postgres@localhost:5432/foo?sslmode=disable", "mydb"),
		).To(Equal(
			"postgresql://postgres@localhost:5432/mydb?sslmode=disable",
		))
		Expect(
			injectDBName("postgresql://postgres@localhost:5432?sslmode=disable", "mydb"),
		).To(Equal(
			"postgresql://postgres@localhost:5432/mydb?sslmode=disable",
		))
		Expect(
			injectDBName("user:pass@tcp(localhost:3306)?something=false", "mydb"),
		).To(Equal(
			"user:pass@tcp(localhost:3306)/mydb?something=false",
		))
		Expect(
			injectDBName("user:pass@tcp(localhost:3306)/foo?something=false", "mydb"),
		).To(Equal(
			"user:pass@tcp(localhost:3306)/mydb?something=false",
		))
	})
})
