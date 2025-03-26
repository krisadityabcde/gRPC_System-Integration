package main

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	pb "github.com/krisadityabcde/gRPC_System-Integration/chat-app/proto"
)

type chatServer struct {
	pb.UnimplementedChatServiceServer
	activeUsers sync.Map
	client      *firestore.Client
}

func main() {
	// Firebase Setup
	ctx := context.Background()
	sa := option.WithCredentialsFile("firebase/grpc-system-integration-firebase-adminsdk-fbsvc-8be5e34ac5.json")
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalf("firebase init error: %v", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalf("firestore init error: %v", err)
	}
	defer client.Close()

	// gRPC Server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterChatServiceServer(s, &chatServer{client: client})
	log.Printf("server listening at %v", lis.Addr())
	
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// Implementasi Unary RPC untuk Login
func (s *chatServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	doc, err := s.client.Collection("users").Doc(req.Username).Get(ctx)
	if err != nil || !doc.Exists() {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	return &pb.LoginResponse{Token: req.Username}, nil
}

// Implementasi Server Streaming untuk Daftar Pengguna
func (s *chatServer) StreamUsers(_ *pb.Empty, stream pb.ChatService_StreamUsersServer) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var users []string
			s.activeUsers.Range(func(key, _ interface{}) bool {
				users = append(users, key.(string))
				return true
			})
			
			if err := stream.Send(&pb.UserList{Users: users}); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

// Implementasi Bidirectional Streaming untuk Chat
func (s *chatServer) ChatStream(stream pb.ChatService_ChatStreamServer) error {
	// Handle koneksi masuk
	msg, err := stream.Recv()
	if err != nil {
		return err
	}

	s.activeUsers.Store(msg.From, stream)
	defer s.activeUsers.Delete(msg.From)

	// Broadcast pesan
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}

		// Simpan ke Firestore
		_, _, err = s.client.Collection("messages").Add(stream.Context(), map[string]interface{}{
			"from":    msg.From,
			"to":      msg.To,
			"message": msg.Message,
			"time":    time.Now(),
		})
		if err != nil {
			log.Printf("Failed to save message: %v", err)
		}

		// Broadcast ke semua pengguna
		s.activeUsers.Range(func(_, value interface{}) bool {
			userStream := value.(pb.ChatService_ChatStreamServer)
			userStream.Send(msg)
			return true
		})
	}
}

// Implementasi Client Streaming untuk Mengirim Banyak Pesan
func (s *chatServer) SendMultipleMessages(stream pb.ChatService_SendMultipleMessagesServer) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.Empty{})
		}
		if err != nil {
			return err
		}

		// Proses pesan
		_, _, err = s.client.Collection("messages").Add(stream.Context(), map[string]interface{}{
			"from":    msg.From,
			"to":      msg.To,
			"message": msg.Message,
			"time":    time.Now(),
		})
	}
}