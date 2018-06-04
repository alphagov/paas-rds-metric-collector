# PaaS RDS metric collector

Small application connecting to all RDS instances hosted on the GOV.UK
PaaS and gathering metrics. Pushing them to loggregator.


## Testing

In order to run the tests, you will be required to run postgres instance on
your machine.

We usually use `docker` to achieve that:

```
docker run -p 5432:5432 --name postgres -e POSTGRES_PASSWORD= -d postgres:9.5
```

You can tear it down after with:

```
docker rm -f postgres
```

Ginkgo is setup with the suite, which allows you to run:

```
go get ./...
go test ./...
```

or

```
ginkgo ./...
```

Without further configuring your stack.

We provide a `Makefile` for the different tasks to run and test the application.
