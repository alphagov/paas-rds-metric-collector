.PHONY: test unit integration start_docker_dbs stop_docker_dbs

TEST_POSTGRES_URL ?= postgres://postgres:@localhost:5432/?sslmode=disable
TETS_MYSQL_URL ?= root:@tcp(localhost:3306)/mysql?tls=false

test: unit

unit:
	ginkgo -r --skipPackage=ci --nodes=8

integration:
	ginkgo -v -r ci/blackbox

start_docker_dbs:
	docker run --rm -p 5432:5432 --name postgres -e POSTGRES_PASSWORD= -d postgres:9.5
	docker run --rm -p 3306:3306 --name mysql -e MYSQL_ALLOW_EMPTY_PASSWORD=yes -d mysql:5.7
	until docker exec mysql mysqladmin ping --silent; do \
		printf "."; sleep 1;                             \
	done

stop_docker_dbs:
	docker stop postgres
	docker stop mysql
