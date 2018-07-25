# PaaS RDS metric collector

Small application connecting to all RDS instances hosted on the GOV.UK
PaaS and gathering metrics. Pushing them to loggregator.


## Testing

In order to run the tests, you will be required to run the DB instances on
your machine.

We usually use `docker` to achieve that, either by using the make target:

```
make start_docker_dbs
```

Or manually by:

```
docker run --rm -p 5432:5432 --name postgres -e POSTGRES_PASSWORD= -d postgres:9.5
docker run --rm -p 3306:3306 --name mysql -e MYSQL_ALLOW_EMPTY_PASSWORD=yes -d mysql:5.7
```

You can tear it down after with:

```
make stop_docker_dbs

```
Or manually by:

```
docker stop postgres
docker stop mysql
```

installing the dependencies can be achieved by running:

```
go get ./...
```

Ginkgo is setup with the suite, which allows you to run:

```
make test
```

or

```
go test ./...
```

or

```
ginkgo ./...
```

Without further configuring your stack.

We provide a `Makefile` for the different tasks to run and test the application.

## How does it work?

The metrics collector queries AWS RDS for instances. Once these have been found the metrics collector generated the master password for each instance in order to spawn a worker process that connects to the instance. There is one process per instance. This runs a series of queries against the instance and pushes the results to loggregator.

From Loggregator the metics can be collected by our tenants in the same manner as any other metrics. This is now through the plugin for log-cache.
