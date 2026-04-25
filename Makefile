run:
	@go run cmd/main.go

cli:
	@go run cmd/cli/main.go @

test:
	@go test -count=1 ./...

.PHONY: proto
proto:
	@protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/*.proto

