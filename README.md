# PaaS RDS metric collector

Small application connecting to all RDS instances hosted on the GOV.UK
PaaS and gathering metrics. Pushing them to loggregator.

## Exported metrics

### Common metrics

The metrics are queried from CloudWatch.

| Metric             | Type  | Description                                                                                                     |
| ------------------ | ----- | --------------------------------------------------------------------------------------------------------------- |
| free_storage_space | gauge | The amount of available storage space, in bytes                                                                 |
| freeable_memory    | gauge | The amount of available random access memory, in bytes                                                          |
| swap_usage         | gauge | The amount of swap space used on the DB instance, in bytes                                                      |
| read_iops          | gauge | The average number of disk read I/O operations per second                                                       |
| write_iops         | gauge | The average number of disk write I/O operations per second                                                      |
| cpu                | gauge | The percentage of CPU utilization                                                                               |
| cpu_credit_usage   | gauge | The number of CPU credits spent by the instance for CPU utilization (t2.* instances)                            |
| cpu_credit_balance | gauge | The number of earned CPU credits that an instance has accrued since it was launched or started (t2.* instances) |

### MySQL-specific metrics

The metrics are queried from various MySQL statistics tables.

| Metric                                | Type  | Description                           |
| ------------------------------------- | ----- | ------------------------------------- |
| threads_connected                     | gauge | [1]                                   |
| threads_running                       | gauge | [1]                                   |
| threads_created                       | gauge | [1]                                   |
| queries                               | gauge | [1]                                   |
| questions                             | gauge | [1]                                   |
| aborted_clients                       | gauge | [1]                                   |
| aborted_connects                      | gauge | [1]                                   |
| innodb_row_lock_waits                 | gauge | [1]                                   |
| innodb_row_lock_time                  | gauge | [1]                                   |
| innodb_num_open_files                 | gauge | [1]                                   |
| innodb_log_waits                      | gauge | [1]                                   |
| innodb_buffer_pool_bytes_data         | gauge | [1]                                   |
| innodb_buffer_pool_bytes_dirty        | gauge | [1]                                   |
| innodb_buffer_pool_pages_data         | gauge | [1]                                   |
| innodb_buffer_pool_pages_dirty        | gauge | [1]                                   |
| innodb_buffer_pool_pages_flushed      | gauge | [1]                                   |
| innodb_buffer_pool_pages_free         | gauge | [1]                                   |
| innodb_buffer_pool_pages_misc         | gauge | [1]                                   |
| innodb_buffer_pool_pages_total        | gauge | [1]                                   |
| innodb_buffer_pool_read_ahead         | gauge | [1]                                   |
| innodb_buffer_pool_read_ahead_evicted | gauge | [1]                                   |
| innodb_buffer_pool_read_ahead_rnd     | gauge | [1]                                   |
| innodb_buffer_pool_read_requests      | gauge | [1]                                   |
| innodb_buffer_pool_reads              | gauge | [1]                                   |
| innodb_buffer_pool_wait_free          | gauge | [1]                                   |
| innodb_buffer_pool_write_requests     | gauge | [1]                                   |
| max_connections                       | gauge | Maximum number of backend connections |
| connection_errors                     | gauge | [1] Sum of all Connection_errors_xxx  |

[1] See https://dev.mysql.com/doc/refman/5.7/en/server-status-variables.html

### PostgreSQL-specific metrics

The metrics are queried from various PostgreSQL statistics tables.

| Metric              | Type  | Description                                                                                          |
| ------------------- | ----- | ---------------------------------------------------------------------------------------------------- |
| connections         | gauge | Number of backends currently connected to the database                                               |
| max_connections     | gauge | Maximum number of connections allowed                                                                |
| dbsize              | gauge | Database storage used in bytes                                                                       |
| deadlocks           | gauge | Number of deadlocks detected in the database                                                         |
| commits             | gauge | Number of transactions in the database that have been committed                                      |
| rollbacks           | gauge | Number of transactions in the database that have been rolled back                                    |
| blocks_read         | gauge | Number of disk blocks read in the database                                                           |
| blocks_hit          | gauge | Number of times disk blocks were found already in the buffer cache, so that a read was not necessary |
| read_time           | gauge | Time spent reading data file blocks by backends in the database, in milliseconds                     |
| write_time          | gauge | Time spent writing data file blocks by backends in the database, in milliseconds                     |
| temp_bytes          | gauge | Total amount of data written to temporary files by queries in the database                           |
| seq_scan            | gauge | Number of index scans initiated on all indexes                                                       |
| idx_scan            | gauge | Number of sequential scans initiated on all tables                                                   |
| blocked_connections | gauge | Number of backends currently waiting for a lock to be released                                       |
| max_tx_age          | gauge | The longest running transaction's age excluding system queries, in seconds                           |
| max_system_tx_age   | gauge | The longest running system transaction's age, in seconds                                             |

## Testing

The tests require [ginkgo](https://onsi.github.io/ginkgo/) which can be installed
by various methods including via `go install github.com/onsi/ginkgo/v2/ginkgo@latest` and
ensuring the resulting binary is on your `$PATH`.

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
