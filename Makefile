.PHONY: test unit integration start_postgres_docker stop_postgres_docker

TEST_DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable

test: unit

unit:
	ginkgo -r --skipPackage=ci --nodes=8

integration:
	ginkgo -v -r ci/blackbox

start_postgres_docker:
	docker run --rm -p 5432:5432 --name postgres -e POSTGRES_PASSWORD= -d postgres:9.5

stop_postgres_docker:
	docker stop postgres
	docker rm postgres
