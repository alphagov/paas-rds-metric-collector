.PHONY: test unit start_postgres_docker stop_postgres_docker

TEST_DATABASE_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable

test: unit

unit:
	ginkgo -r --nodes=8  ./...

start_postgres_docker:
	docker run -p 5432:5432 --name postgres -e POSTGRES_PASSWORD= -d postgres:9.5

stop_postgres_docker:
	docker stop postgres
	docker rm postgres
