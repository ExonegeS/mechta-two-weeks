run:
	@clear; go run cmd/worker-pool/main.go

proto:
	protoc   -I=pkg/grpc/  --go_out=paths=source_relative:pkg/grpc/   --go-grpc_out=paths=source_relative:pkg/grpc/   pkg/grpc/mindbox.proto