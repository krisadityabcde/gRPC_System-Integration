1. protoc --go_out=proto/ --go-grpc_out=proto/ proto/chat.proto
2. go mod tidy
3. go run server/main.go
4. go run client/main.go
