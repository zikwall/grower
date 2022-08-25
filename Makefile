PROJECT_NAME=$(shell basename "$(PWD)")
SCRIPT_AUTHOR=Andrey Kapitonov <andrey.kapitonov.96@gmail.com>
SCRIPT_VERSION=0.0.1
SERVICES=\
	filebuf

$(SERVICES):
	protoc -I ./protobuf/$@/ -I . \
		--go_out=./protobuf/$@/ \
		--go_opt=paths=source_relative \
		--go-grpc_out=./protobuf/$@/ \
		--go-grpc_opt=paths=source_relative \
		$@.proto;

default: $(SERVICES)