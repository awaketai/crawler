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