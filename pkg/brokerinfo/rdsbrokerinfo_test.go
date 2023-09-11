package brokerinfo_test

import (
	"fmt"

	rdsfake "github.com/alphagov/paas-rds-broker/awsrds/fakes"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

var _ = Describe("RDSBrokerInfo", func() {
	var (
		brokerInfo     *brokerinfo.RDSBrokerInfo
		fakeDBInstance *rdsfake.FakeRDSInstance
	)

	BeforeEach(func() {
		fakeDBInstance = &rdsfake.FakeRDSInstance{}
		brokerInfo = brokerinfo.NewRDSBrokerInfo(
			config.RDSBrokerInfoConfig{
				BrokerName:         "broker_name",
				DBPrefix:           "dbprefix",
				MasterPasswordSeed: "12345",
			},
			fakeDBInstance,
			logger,
		)
	})

	Context("ListInstances()", func() {
		BeforeEach(func() {
			fakeDBInstance.DescribeByTagReturns(
				[]*rds.DBInstance{
					{
						DBInstanceIdentifier: aws.String("dbprefix-instance-id-1"),
						Engine:               aws.String("postgres"),
						Endpoint: &rds.Endpoint{
							Address: aws.String("endpoint-address-1.example.com"),
							Port:    aws.Int64(5432),
						},
						DBName:         aws.String("dbprefix-db"),
						MasterUsername: aws.String("master-username"),
					},
					{
						DBInstanceIdentifier: aws.String("dbprefix-instance-id-2"),
						Engine:               aws.String("postgres"),
						Endpoint: &rds.Endpoint{
							Address: aws.String("endpoint-address-2.example.com"),
							Port:    aws.Int64(5432),
						},
						DBName:         aws.String("dbprefix-db"),
						MasterUsername: aws.String("master-username"),
					},
					{
						DBInstanceIdentifier: aws.String("dbprefix-instance-id-3"),
						Engine:               aws.String("mysql"),
						Endpoint: &rds.Endpoint{
							Address: aws.String("endpoint-address-3.example.com"),
							Port:    aws.Int64(3306),
						},
						DBName:         aws.String("dbprefix-db"),
						MasterUsername: aws.String("master-username"),
					},
				},
				nil,
			)
		})

		It("returns error if it fails retrieving existing instances in AWS", func() {
			fakeDBInstance.DescribeByTagReturns(nil, fmt.Errorf("error calling rds.DescribeByTag(...)"))

			_, err := brokerInfo.ListInstances()
			Expect(err).To(HaveOccurred())
		})
		It("lists the instances for the right tag", func() {
			_, err := brokerInfo.ListInstances()
			Expect(err).NotTo(HaveOccurred())
			describeByTagTagNameArg, describeByTagTagValueArg, _ := fakeDBInstance.DescribeByTagArgsForCall(0)
			Expect(describeByTagTagNameArg).To(Equal("Broker Name"))
			Expect(describeByTagTagValueArg).To(Equal("broker_name"))
		})
		It("returns the list of instances", func() {
			instances, err := brokerInfo.ListInstances()
			Expect(err).NotTo(HaveOccurred())
			Expect(instances).To(ConsistOf(
				brokerinfo.InstanceInfo{GUID: "instance-id-1", Type: "postgres"},
				brokerinfo.InstanceInfo{GUID: "instance-id-2", Type: "postgres"},
				brokerinfo.InstanceInfo{GUID: "instance-id-3", Type: "mysql"},
			))
		})
	})

	Context("GetInstanceConnectionDetails()", func() {
		BeforeEach(func() {
			fakeDBInstance.DescribeReturns(
				&rds.DBInstance{
					DBInstanceIdentifier: aws.String("dbprefix-instance-id"),
					Endpoint: &rds.Endpoint{
						Address: aws.String("endpoint-address.example.com"),
						Port:    aws.Int64(5432),
					},
					DBName:         aws.String("dbprefix-db"),
					MasterUsername: aws.String("master-username"),
				},
				nil,
			)
		})

		It("returns error if it fails retrieving existing instances in AWS", func() {
			fakeDBInstance.DescribeReturns(nil, fmt.Errorf("error calling rds.Describe(...)"))

			_, err := brokerInfo.GetInstanceConnectionDetails(brokerinfo.InstanceInfo{GUID: "instance-id", Type: "postgres"})
			Expect(err).To(HaveOccurred())
		})
		It("returns error if we query the wrong instance type", func() {
			_, err := brokerInfo.GetInstanceConnectionDetails(brokerinfo.InstanceInfo{GUID: "instance-id", Type: "foo"})
			Expect(err).To(HaveOccurred())
		})
		It("retrieves information of the right AWS RDS instance", func() {
			_, err := brokerInfo.GetInstanceConnectionDetails(brokerinfo.InstanceInfo{GUID: "instance-id", Type: "postgres"})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeDBInstance.DescribeCallCount()).To(BeNumerically(">=", 1))
			describeIdArg := fakeDBInstance.DescribeArgsForCall(0)
			Expect(describeIdArg).To(Equal("dbprefix-instance-id"))
		})
		It("returns the proper information of the instance", func() {
			details, err := brokerInfo.GetInstanceConnectionDetails(brokerinfo.InstanceInfo{GUID: "instance-id", Type: "postgres"})
			Expect(err).ToNot(HaveOccurred())
			Expect(details.DBAddress).To(Equal("endpoint-address.example.com"))
			Expect(details.DBPort).To(BeNumerically("==", 5432))
			Expect(details.DBName).To(Equal("dbprefix-db"))
			Expect(details.MasterUsername).To(Equal("master-username"))
			Expect(details.MasterPassword).To(Equal("9Fs6CWnuwf0BAY3rDFAels3OXANSo0-M"))
		})
		It("fails if the type is invalid", func() {
			_, err := brokerInfo.GetInstanceConnectionDetails(brokerinfo.InstanceInfo{GUID: "instance-id", Type: "foo"})
			Expect(err).To(HaveOccurred())
		})
	})

})
