# PaaS RDS metric collector

Small application connecting to all RDS instances hosted on the GOV.UK
PaaS and gathering metrics. Pushing them to loggregator.

## Getting started

Install the dependencies with [dep](https://github.com/golang/dep):

```
dep ensure
```

## Testing

To run the tests you need to run the databases on your machine:

```
make start_docker_dbs
```

Then run the tests with:

```
make test
```

You can stop the databases with:

```
make stop_docker_dbs
```

### Fixtures

Certificates in the test fixtures are generated with a script. To generate them run:

```bash
./scripts/generate-cert-fixtures.sh
```

(See the [Makefile](Makefile) for more details)

## Running locally

The application will not do very much running locally, but it is possible to
start it.

First you will need the databases and locket (or a mock) running. These can be
started with the following make tasks:

```
make start_docker_dbs
make start_mock_locket_server
```

And stopped with:

```
make stop_docker_dbs
make stop_mock_locket_server
```

(See the [Makefile](Makefile) for more details)

You can then start the application with:

```
go run main.go -config=./fixtures/collector_config.json
```

## How does it work?

The metrics collector queries AWS RDS for instances. Once these have been found the metrics collector generated the master password for each instance in order to spawn a worker process that connects to the instance. There is one process per instance. This runs a series of queries against the instance and pushes the results to loggregator.

From Loggregator the metics can be collected by our tenants in the same manner as any other metrics. This is now through the plugin for log-cache.
