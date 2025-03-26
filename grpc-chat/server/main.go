package main

import (
	"fmt"
	"log"
	"net"

	"grpc-chat/common/repository"
	"grpc-chat/common/services"
	pb "grpc-chat/proto"

	"google.golang.org/grpc"
)

func main() {
	// Inisialisasi repository dan service
	userRepo := repository.NewUserRepository()
	messageRepo := repository.NewMessageRepository()
	chatService := service.NewChatService(userRepo, messageRepo)

	// Setup server
	server := grpc.NewServer()
	pb.RegisterChatServiceServer(server, chatService)

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	fmt.Println("Server is running on port 50051...")
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
