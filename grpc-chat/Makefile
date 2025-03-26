all: proto server client

generate_proto:
	@protoc --go_out=proto/ --go-grpc_out=proto/ proto/chat.proto

server:
	@go run server/main.go $(profile)

client:
	@go run client/main.go $(profile)
