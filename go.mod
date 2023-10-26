module github.com/alphagov/paas-rds-metric-collector

go 1.20

require (
	code.cloudfoundry.org/clock v0.0.0-20180518195852-02e53af36e6c
	code.cloudfoundry.org/go-loggregator/v9 v9.0.8
	code.cloudfoundry.org/lager/v3 v3.0.2
	code.cloudfoundry.org/locket v0.0.0-20230612151453-08e003863044
	github.com/Kount/pq-timeouts v1.0.0
	github.com/alphagov/paas-rds-broker v1.54.0
	github.com/aws/aws-sdk-go v1.42.50
	github.com/go-sql-driver/mysql v1.6.0
	github.com/lib/pq v1.10.4
	github.com/onsi/ginkgo/v2 v2.13.0
	github.com/onsi/gomega v1.27.10
	github.com/phayes/freeport v0.0.0-20220201140144-74d24b5ae9f5
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.8.1
	github.com/tedsuo/ifrit v0.0.0-20230516164442-7862c310ad26
	golang.org/x/net v0.15.0
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.0
	gopkg.in/go-playground/validator.v9 v9.20.2
)

replace github.com/Kount/pq-timeouts v1.0.0 => ./fork/pq-timeouts

require (
	code.cloudfoundry.org/go-diodes v0.0.0-20180905200951-72629b5276e3 // indirect
	code.cloudfoundry.org/inigo v0.0.0-20210929170650-c842b4924e10 // indirect
	code.cloudfoundry.org/tlsconfig v0.0.0-20231017135636-f0e44068c22f // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-playground/locales v0.12.1 // indirect
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/gogo/protobuf v1.1.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/pprof v0.0.0-20210720184732-4bb14d4b1be1 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/openzipkin/zipkin-go v0.4.2 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pivotal-cf/brokerapi/v9 v9.0.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	golang.org/x/tools v0.12.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
