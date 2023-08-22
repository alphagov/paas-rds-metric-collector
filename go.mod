module github.com/alphagov/paas-rds-metric-collector

go 1.20

require (
	code.cloudfoundry.org/clock v0.0.0-20180518195852-02e53af36e6c
	code.cloudfoundry.org/go-loggregator v7.0.0+incompatible
	code.cloudfoundry.org/lager v2.0.0+incompatible
	code.cloudfoundry.org/locket v0.0.0-20180713150409-cd6f53abfd14
	github.com/Kount/pq-timeouts v1.0.0
	github.com/alphagov/paas-go v0.0.0-20181008075827-0065fc706364
	github.com/alphagov/paas-rds-broker v1.48.0
	github.com/aws/aws-sdk-go v1.42.50
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/lib/pq v1.10.4
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.19.0
	github.com/phayes/freeport v0.0.0-20220201140144-74d24b5ae9f5
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.5.1
	github.com/tedsuo/ifrit v0.0.0-20180622163835-2a37a9eb7c3a
	golang.org/x/net v0.14.0
	gopkg.in/go-playground/validator.v9 v9.20.2
)

replace github.com/Kount/pq-timeouts v1.0.0 => ./fork/pq-timeouts

require (
	code.cloudfoundry.org/cfhttp v1.0.0 // indirect
	code.cloudfoundry.org/consuladapter v0.0.0-20170912000402-c6d9ccbe0f83 // indirect
	code.cloudfoundry.org/diego-logging-client v0.0.0-20180713150051-67e71e13e3da // indirect
	code.cloudfoundry.org/go-diodes v0.0.0-20180717154652-3385e722aaa0 // indirect
	code.cloudfoundry.org/inigo v0.0.0-20210929170650-c842b4924e10 // indirect
	code.cloudfoundry.org/rfc5424 v0.0.0-20170822183049-769e2ed6887e // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/armon/go-metrics v0.3.9 // indirect
	github.com/cloudfoundry/sonde-go v0.0.0-20171206171820-b33733203bb4 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-playground/locales v0.12.1 // indirect
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/gogo/protobuf v1.1.1 // indirect
	github.com/golang/glog v1.0.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/hashicorp/consul v1.2.1 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.0 // indirect
	github.com/hashicorp/go-msgpack v1.1.5 // indirect
	github.com/hashicorp/go-rootcerts v0.0.0-20160503143440-6bb64b370b90 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/memberlist v0.2.4 // indirect
	github.com/hashicorp/serf v0.8.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mailru/easyjson v0.0.0-20180723221831-d5012789d665 // indirect
	github.com/mitchellh/go-homedir v0.0.0-20180523094522-3864e76763d9 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/mapstructure v0.0.0-20180715050151-f15292f7a699 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/pascaldekloe/goe v0.1.0 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pivotal-cf/brokerapi/v8 v8.2.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.1.1 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/text v0.12.0 // indirect
	golang.org/x/tools v0.12.0 // indirect
	google.golang.org/genproto v0.0.0-20180722052100-02b4e9547331 // indirect
	google.golang.org/grpc v1.13.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
