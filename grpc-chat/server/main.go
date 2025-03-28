package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	pb "grpc-chat/proto"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedChatServiceServer
	mu           sync.Mutex
	streams      map[string]pb.ChatService_ChatStreamServer
	userStreams  map[string]string // Maps username to stream ID
	messageCache []*pb.ChatMessage // Cache of recent messages
}

// Login hanya menyimpan username
func (s *server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	log.Printf("User login: %s", req.Username)
	return &pb.LoginResponse{
		Username: req.Username,
		Message:  "Login sukses",
	}, nil
}

// Add a function to handle user leaving
func (s *server) userLeft(username string) {
	// Create a message to inform all clients this user has left
	leaveMsg := &pb.ChatMessage{
		Sender:    username,
		Message:   "left the chat",
		Timestamp: time.Now().Format("15:04:05"),
	}

	// Broadcast the leave message to all clients
	s.broadcastMessage(leaveMsg)

	log.Printf("User %s left the chat", username)
}

// ChatStream implements bidirectional streaming RPC
func (s *server) ChatStream(stream pb.ChatService_ChatStreamServer) error {
	// Get a unique ID for this stream
	streamID := fmt.Sprintf("%p", stream)
	log.Printf("New chat stream connected: %s", streamID)

	// Store this stream
	s.mu.Lock()
	s.streams[streamID] = stream
	s.mu.Unlock()

	var currentUser string // Track the username for this stream

	// Clean up on disconnect
	defer func() {
		s.mu.Lock()
		// Find and remove username mapping to this stream
		for username, sid := range s.userStreams {
			if sid == streamID {
				currentUser = username
				delete(s.userStreams, username)
				log.Printf("Removed user %s mapping to stream %s", username, streamID)
			}
		}
		delete(s.streams, streamID)
		s.mu.Unlock()

		// If we have a username, inform other clients this user has left
		if currentUser != "" {
			s.userLeft(currentUser)
		}

		log.Printf("Stream disconnected: %s", streamID)
	}()

	// Handle messages from the client
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil // Client closed normally
			}
			return err // Some other error
		}

		// Log received message
		log.Printf("Received message from %s: %s", msg.Sender, msg.Message)

		// Associate this username with the stream ID
		s.mu.Lock()
		// If this username already has a different stream, use the new one
		if existingStreamID, ok := s.userStreams[msg.Sender]; ok && existingStreamID != streamID {
			log.Printf("User %s reconnected. Old stream: %s, New stream: %s",
				msg.Sender, existingStreamID, streamID)
		}
		s.userStreams[msg.Sender] = streamID

		// Add message to cache
		s.messageCache = append(s.messageCache, msg)
		if len(s.messageCache) > 100 { // Limit cache size
			s.messageCache = s.messageCache[1:]
		}
		s.mu.Unlock()

		// Broadcast to all connected streams
		s.broadcastMessage(msg)
	}
}

// Helper function to broadcast message to all streams
func (s *server) broadcastMessage(msg *pb.ChatMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Broadcasting message from %s to %d clients", msg.Sender, len(s.streams))

	for id, clientStream := range s.streams {
		if err := clientStream.Send(msg); err != nil {
			log.Printf("Error sending to stream %s: %v", id, err)
		}
	}
}

func main() {
	// Create and configure server
	s := &server{
		streams:      make(map[string]pb.ChatService_ChatStreamServer),
		userStreams:  make(map[string]string),
		messageCache: make([]*pb.ChatMessage, 0, 100),
	}

	// Set up gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Configure and start server
	opts := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(100),
		grpc.ConnectionTimeout(30 * time.Second),
	}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterChatServiceServer(grpcServer, s)

	log.Println("gRPC Server running on port 50051")
	log.Println("Ready to handle chat connections")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
