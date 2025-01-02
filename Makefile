.PHONY: proto clean gen
proto:
	protoc -I proto/ \
		-I $(GOPATH)/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.16.0/ \
		-I $(GOPATH)/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.16.0/third_party/googleapis \
		--micro_out=./goout \
		--go_out=./goout  \
		--go-grpc_out=./goout  \
		--grpc-gateway_out=logtostderr=true,register_func_suffix=Gw:./goout \
		./proto/hello.proto

clean:
	rm -rf goout/*

gen:
	make clean
	make proto


LDFLAGS = -X "main.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "main.GitHash=$(shell git rev-parse HEAD)"
LDFLAGS += -X "main.GitBranch=$(shell git rev-parse --abbrev-ref HEAD)"
LDFLAGS += -X "main.Version=${VERSION}"

ifeq ($(gorace), 1)
  BUILD_FLAGS=-race
endif

build:
  go build -ldflags '$(LDFLAGS)' $(BUILD_FLAGS) main.go

lint:
  golangci-lint run ./...