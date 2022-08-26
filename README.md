[![build](https://github.com/zikwall/grower/workflows/build_and_tests/badge.svg)](https://github.com/zikwall/clickhouse-buffer/v4/actions)
[![build](https://github.com/zikwall/grower/workflows/golangci_lint/badge.svg)](https://github.com/zikwall/clickhouse-buffer/v4/actions)

<div align="center">
  <h1>Grower</h1>
  <h5>An easy-to-use, powerful and productive tool that allows you to write Nginx logs to a Clickhouse columnar database.</h5>
</div>

**Features:**

- **No dependencies**: it works like a regular binary file or in docker
- Supports **three versions** of content delivery:
  - **FileLog** read/write/rotate support
  - **SysLog** protocol (tcp, udp, unix) support
  - **FileBuf** gRPC client and server
- **Fully customizable**: 
  - timeouts and runtime limitations (buffer sizes, flush intervals, retries configuration),
  - schema & log formats
  - support **all native nginx attributes** and ability to add your **own fields**
  - **multithreading** support and customizable
- **Completely Type Safe**: native support for protection types

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

### How to use it?

### Configuration

is very simple and clear, see [sample.yaml configuration file](./sample_test.yaml)

<details>
  <summary><b>Example:</b></summary>

```yaml
nginx:
  log_type: csv
  log_time_format: '02/Jan/2006:15:04:05 -0700'
  log_time_rewrite: true
  log_custom_casts_enable: true
  log_custom_casts:
    custom_field: Integer
    custom_time_field: Datetime
  log_format: '$remote_addr - $remote_user [$time_local] "$request" $status $bytes_sent $request_time "$request_method" "$http_referer" "$http_user_agent" $https $custom_field <$custom_time_field>'
  log_remove_hyphen: true
scheme:
  logs_table: only_tests.access_log
  columns:
    remote_addr: remote_addr
    remote_user: remote_user
    time_local: time_local
    request: request
    status: status
    bytes_sent: bytes_sent
    request_time: request_time
    request_method: request_method
    http_referer: http_referer
    http_user_agent: http_user_agent
    https: https
    custom_field: custom_field
    custom_time_field: custom_time_field
```
</details>

### FileLog

<details>
  <summary>Run Go native binary:</summary>

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
    --run-http-server \
    --run-rotating-at-startup
```
</details>


<details>
  <summary>Run container in Docker:</summary>

```shell
docker run -d --net=host \
   -v /usr/share/config/:/config/ \
   -e CONFIG_FILE='/config/sample_test.yaml' \
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
</details>


<details>
  <summary>Run build Docker image:</summary>

```shell
#!/bin/bash

docker build -t qwx1337/grower-filelog:latest -f ./cmd/filelog/Dockerfile .
```
</details>

**For more information:**

`$ go run ./cmd/filelog/main.go --help`

### SysLog

<details>
  <summary>Run Go native binary:</summary>

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
</details>


<details>
  <summary>Run container in Docker:</summary>

```shell
docker run -d --net=host \
   -v /usr/share/config/:/config/ \
   -e CONFIG_FILE='/config/sample_test.yaml' \
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
</details>


<details>
  <summary>Run build Docker image:</summary>

```shell
#!/bin/bash

docker build -t qwx1337/grower-syslog:latest -f ./cmd/syslog/Dockerfile .
```
</details>

**For more information:**

`$ go run ./cmd/syslog/main.go --help`

### FileBuf: gRPC client and server

**Client side:**

<details>
  <summary>Run <b>Client</b> Go native binary:</summary>

```shell
go run ./cmd/filecleint/main.go  \
    --bind-address 0.0.0.0:3000 \
    --grpc-conn-address 0.0.0.0:3003 \
    --logs-dir /var/log/nginx \
    --source-log-file access.log \
    --scrape-interval '10s' \
    --backup-files 5 \
    --backup-file-max-age '1m0s' \
    --parallelism 5 \
    --debug \
    --auto-create-target-from-scratch \
    --enable-rotating \
    --skip-nginx-reopen \
    --run-rotating-at-startup \
    --run-http-server
```
</details>


<details>
  <summary>Run <b>Client</b> container in Docker:</summary>

```shell
docker run -d --net=host \
   -e BIND_ADDRESS='0.0.0.0:3004' \
   -e GRPC_BIND_ADDRESS='0.0.0.0:3003' \
   -e SOURCE_LOG_FILE='access.log' \
   -e LOGS_DIR='/var/log/nginx' \
   -e SCRAPE_INTERVAL='1m0s' \
   -e BACKUP_FILES=5 \
   -e BACKUP_FILE_MAX_AGE='5m0s' \
   -e PARALLELISM=5 \
   -e RUN_HTTP_SERVER=true \
   -e AUTO_CREATE_TARGET_FROM_SCRATCH \
   -e ENABLE_ROTATING \
   -e SKIP_NGINX_REOPEN \
   -e RUN_ROTATING_AT_STARTUP \
   -e DEBUG=true \
   -e RUN_HTTP_SERVER=true \
   --name grower-filebuf-client qwx1337/grower-filebuf-client:latest
```
</details>


<details>
  <summary>Run build <b>Client</b> Docker image:</summary>

```shell
#!/bin/bash

docker build -t qwx1337/grower-filebuf-client:latest -f ./cmd/fileclient/Dockerfile .
```
</details>

**For more information:**

`$ go run ./cmd/fileclient/main.go --help`

**Server side::**

<details>
  <summary>Run <b>Server</b> Go native binary:</summary>

```shell
go run ./cmd/fileserver/main.go  \
    --config-file ./sample_test.yaml \
    --bind-address 0.0.0.0:3000 \
    --grpc-bind-address 0.0.0.0:3003 \
    --clickhouse-host 'xxx.xx.xx.xx:9000' \
    --clickhouse-host 'xxx.xx.xx.xx:9001' \
    --clickhouse-user default \
    --clickhouse-database default \
    --clickhouse-password '' \
    --buffer-size 10000 \
    --buffer-flush-interval 5000 \
    --write-timeout '0m30s' \
    --parallelism 5 \
    --run-http-server \
    --debug
```
</details>


<details>
  <summary>Run <b>Server</b> container in Docker:</summary>

```shell
docker run -d --net=host \
   -v /usr/share/config/:/config/ \
   -e CONFIG_FILE='/config/sample_test.yaml' \
   -e BIND_ADDRESS='0.0.0.0:3004' \
   -e GRPC_BIND_ADDRESS='0.0.0.0:3003' \
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
   --name grower-filebuf-server qwx1337/grower-filebuf-server:latest
```
</details>


<details>
  <summary>Run build <b>Server</b> Docker image:</summary>

```shell
#!/bin/bash

docker build -t qwx1337/grower-filebuf-server:latest -f ./cmd/fileserver/Dockerfile .
```
</details>

**For more information:**

`$ go run ./cmd/filserver/main.go --help`

### Recommendations and Notes

1. Only for big data (200k>) or fast line-by-line processing (10k/s>):
   - Use a buffer size between 1000 and 5000 if you want less CPU utilization, but this **will increase processing time**.
   - Use a buffer size in the range of 5000, 10000 and more if you want to process data faster, but this **will increase CPU and memory utilization**.
2. You can find the most advantageous option in your case yourself by simply configuring the following arguments: (`--buffer-size`, `--buffer-flush-interval`)
3. Easiest way to use it is SysLog, at the same time optimal, because there are no sharp spikes in load when reading a file, as in the case of FileLog
4. Remember that delivery to **SysLog via UDP is not guaranteed**
5. FileLog is convenient, because there is no need to raise separate servers, as in the case of SysLog and FileBuf
6. FileBuf is as load-efficient as SysLog, but with reliable content delivery.
7. At the moment FileBuf **does not support automatic reconnection** of streams
8. You can configure an acceptable multithreading for yourself through the argument `--parallelism`
