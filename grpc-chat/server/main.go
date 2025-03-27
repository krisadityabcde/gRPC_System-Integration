package main

import (
	"context"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
	pb "grpc-chat/proto"
)

type server struct {
	pb.UnimplementedChatServiceServer
	mu      sync.Mutex
	clients map[string]pb.ChatService_ChatStreamServer
}

// Login hanya menyimpan username
func (s *server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	log.Printf("User login: %s", req.Username)
	return &pb.LoginResponse{
		Username: req.Username,
		Message:  "Login sukses",
	}, nil
}

// Streaming Chat (Bidirectional)
func (s *server) ChatStream(stream pb.ChatService_ChatStreamServer) error {
	var username string

	// Terima pesan pertama untuk mendapatkan username
	msg, err := stream.Recv()
	if err != nil {
		return err
	}
	username = msg.Sender

	// Simpan client dalam daftar
	s.mu.Lock()
	s.clients[username] = stream
	s.mu.Unlock()

	// Kirim pesan ke semua client saat ada pesan masuk
	for {
		msg, err := stream.Recv()
		if err != nil {
			// Hapus client saat disconnect
			s.mu.Lock()
			delete(s.clients, username)
			s.mu.Unlock()
			return err
		}

		log.Printf("[%s] %s: %s", msg.Timestamp, msg.Sender, msg.Message)

		// Kirim ke semua client
		s.mu.Lock()
		for _, client := range s.clients {
			client.Send(msg)
		}
		s.mu.Unlock()
	}
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterChatServiceServer(grpcServer, &server{
		clients: make(map[string]pb.ChatService_ChatStreamServer),
	})

	log.Println("gRPC Server berjalan di port 50051")
	grpcServer.Serve(lis)
}
