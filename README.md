## CK-NGINX

### How to use?

### FileLog server - fast, safe, async, nginx logs parser

**With native binary:**

```shell
TODO
```

**With Docker container:**

```shell
TODO
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
go run ./cmd/syslog/  \
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
   -e PARALLELISM=5 \
   -e RUN_HTTP_SERVER=true \
   -e DEBUG=true \
   --name ck-nginx-syslog qwx1337/ck-nginx-syslog:latest
```

**Local build Docker image:**

```shell
#!/bin/bash

docker build -t your_image_name:latest -f ./cmd/syslog/Dockerfile .
```

**For more information:**

`$ go run ./cmd/syslog/main.go --help`