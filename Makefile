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

start_mock_locket_server:
	go run ./testhelpers/mock_locket_server/main.go -listenAddress=127.0.0.1:8891 -fixturesPath=./fixtures -mode=alwaysGrantLock \
		> tmp/mock_locket_server.stdout \
		2> tmp/mock_locket_server.stderr \
		& \
		ps -o pgid= -p "$$!" > tmp/mock_locket_server.pgid

stop_mock_locket_server:
	@if [ -f tmp/mock_locket_server.pgid ];\
	then\
		kill -- "-$$(cat tmp/mock_locket_server.pgid)";\
		rm tmp/mock_locket_server.pgid;\
	fi

