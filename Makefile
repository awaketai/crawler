.PHONY: proto clean gen
proto:
	protoc -I proto/ \
		-I $(GOPATH)/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.16.0/ \
		-I $(GOPATH)/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.16.0/third_party/googleapis \
		--micro_out=./goout \
		--go_out=./goout  \
		--go-grpc_out=./goout  \
		--grpc-gateway_out=logtostderr=true,register_func_suffix=Gw:./goout \
		./proto/*

clean:
	rm -rf goout/*

gen:
	make clean
	make proto

VERSION := v1.0.0
BUILD_TARGET = crawler

LDFLAGS = -X "main.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "main.GitHash=$(shell git rev-parse HEAD)"
LDFLAGS += -X "main.GitBranch=$(shell git rev-parse --abbrev-ref HEAD)"
LDFLAGS += -X "main.Version=${VERSION}"

ifeq ($(gorace), 1)
  BUILD_FLAGS=-race
endif

build:
	go build -ldflags '$(LDFLAGS)' -o $(BUILD_TARGET) main/main.go

lint:
	golangci-lint run ./...


master1: build
	./$(BUILD_TARGET) master --id=1 --http=:8081 --grpc=:9091

master2: build
	./$(BUILD_TARGET) master --id=2 --http=:8082 --grpc=:9092

master3: build
	./$(BUILD_TARGET) master --id=3 --http=:8083 --grpc=:9093

worker1: build
	./$(BUILD_TARGET) worker --id=1 --http=:6081 --grpc=:7091

worker2: build
	./$(BUILD_TARGET) worker --id=2 --http=:6082 --grpc=:7092

