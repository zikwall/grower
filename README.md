[![build](https://github.com/zikwall/grower/workflows/build_and_tests/badge.svg)](https://github.com/zikwall/clickhouse-buffer/v4/actions)
[![build](https://github.com/zikwall/grower/workflows/golangci_lint/badge.svg)](https://github.com/zikwall/clickhouse-buffer/v4/actions)

<div align="center">
  <h1>Grower</h1>
  <h5>An easy-to-use, powerful and productive tool that allows you to write Nginx logs to a Clickhouse columnar database.</h5>
</div>

**Features:**

- **No dependencies**: it works like a regular binary file or in docker
- **Syslog** protocol (tcp, udp, unix) support
- **File Log** read/write/rotate support
- **Fully configurable**: timeouts, buffer sizes, flush intervals, retries, schema & log formats
- **Type Safe**: native support for protection types
- Support **all nginx attributes** and ability to add your **own fields**
- **Multithreading** support

**TODO:**

- prometheus metrics and dashboard configuration
- saving corrupted files for manual processing
- possibility of log native compression
- native support for more data types
- native support for complex data types such as:
  - Geo: `GeoIPRegion(ip)`, `GeoIPCity(ip)`, `GeoIPAS(ip)`
  - JSON: `JSONStringField(field_name, json_string_field)`, `JSONUInt64Field(field_name, json_string_field)`
  - RegExp: `RegExp('/\(?([0-9]{3})\)?([ .-]?)([0-9]{3})\2([0-9]{4})/', target_field)` - example get phone number from string
  - Cast: `toUInt32(GeoIPAS(ip))`

### How to use?

### Configuration

is very simple and clear, see [sample.yaml configuration file](./sample_test.yaml)

### FileLog server - fast, safe, async, nginx logs parser

**With native binary:**

```shell
go run ./cmd/filelog/main.go  \
    --config-file ./sample_test.yaml \
    --bind-address 0.0.0.0:3000 \
    --logs-dir /var/log/nginx \
    --source-log-file access.log \
    --scrape-interval '10s' \
    --backup-files 5 \
    --backup-file-max-age '1m0s' \
    --clickhouse-host 'xxx.xx.xx.xx:9000' \
    --clickhouse-host 'xxx.xx.xx.xx:9001' \
    --clickhouse-user default \
    --clickhouse-database default \
    --clickhouse-password '' \
    --buffer-size 10000 \
    --buffer-flush-interval 5000 \
    --write-timeout '0m30s' \
    --parallelism 5 \
    --debug \
    --auto-create-target-from-scratch \
    --enable-rotating \
    --skip-nginx-reopen \
    --run-rotating-at-startup \
    --rewrite-nginx-local-time
```

**With Docker container:**

```shell
docker run -d --net=host \
   -v /usr/share/config/:/config/ \
   -e CONFIG_FILE='/config/sample_test.yaml'
   -e BIND_ADDRESS='0.0.0.0:3004' \
   -e SOURCE_LOG_FILE='access.log' \
   -e LOGS_DIR='/var/log/nginx' \
   -e SCRAPE_INTERVAL='1m0s' \
   -e BACKUP_FILES=5 \
   -e BACKUP_FILE_MAX_AGE='5m0s' \
   -e CLICKHOUSE_HOST='xxx.xx.xx.xx:9000,xxx.xx.xx.xx:9001' \
   -e CLICKHOUSE_USER='default' \
   -e CLICKHOUSE_PASSWORD = '' \
   -e CLICKHOUSE_DATABASE='default' \
   -e BUFFER_FLUSH_INTERVAL=2000 \
   -e BUFFER_SIZE=5000 \
   -e WRITE_TIMEOUT='0m30s' \
   -e PARALLELISM=5 \
   -e RUN_HTTP_SERVER=true \
   -e AUTO_CREATE_TARGET_FROM_SCRATCH \
   -e ENABLE_ROTATING \
   -e SKIP_NGINX_REOPEN \
   -e RUN_ROTATING_AT_STARTUP \
   -e DEBUG=true \
   --name grower-syslog qwx1337/grower-filelog:latest
```

**Local build Docker image:**

```shell
#!/bin/bash

docker build -t your_image_name:latest -f ./cmd/filelog/Dockerfile .
```

**For more information:**

`$ go run ./cmd/filelog/main.go --help`

### Syslog server - system log standard

**With native binary:**

```shell
go run ./cmd/syslog/main.go  \
    --config-file ./sample_test.yaml \
    --bind-address 0.0.0.0:3000 \
    --syslog-unix-socket /tmp/syslog.sock \
    --syslog-udp-address 0.0.0.0:3011 \
    --syslog-tcp-address 0.0.0.0:3012 \
    --listeners 'unix' \
    --listeners 'tcp' \
    --listeners 'udp' \
    --clickhouse-host 'xxx.xx.xx.xx:9000' \
    --clickhouse-host 'xxx.xx.xx.xx:9001' \
    --clickhouse-user default \
    --clickhouse-database default \
    --clickhouse-password '' \
    --buffer-size 5000 \
    --buffer-flush-interval 2000 \
    --write-timeout '0m30s' \
    --parallelism 5 \
    --run-http-server \
    --debug
```

**With Docker container:**

```shell
docker run -d --net=host \
   -v /usr/share/config/:/config/ \
   -e CONFIG_FILE='/config/sample_test.yaml'
   -e BIND_ADDRESS='0.0.0.0:3004' \
   -e SYSLOG_UNIX_SOCKET='/tmp/syslog.sock' \
   -e SYSLOG_UDP_ADDRESS='0.0.0.0:3011' \
   -e SYSLOG_TCP_ADDRESS='0.0.0.0:3012' \
   -e LISTENERS='unix,tcp,udp' \
   -e CLICKHOUSE_HOST='xxx.xx.xx.xx:9000,xxx.xx.xx.xx:9001' \
   -e CLICKHOUSE_USER='default' \
   -e CLICKHOUSE_PASSWORD = '' \
   -e CLICKHOUSE_DATABASE='default' \
   -e BUFFER_FLUSH_INTERVAL=2000 \
   -e BUFFER_SIZE=5000 \
   -e WRITE_TIMEOUT='0m30s' \
   -e PARALLELISM=5 \
   -e RUN_HTTP_SERVER=true \
   -e DEBUG=true \
   --name grower-syslog qwx1337/grower-syslog:latest
```

**Local build Docker image:**

```shell
#!/bin/bash

docker build -t your_image_name:latest -f ./cmd/syslog/Dockerfile .
```

**For more information:**

`$ go run ./cmd/syslog/main.go --help`