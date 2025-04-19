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

	// Add tracking for active users and user streams
	activeUsers       map[string]bool                                   // Track active users by username
	activeUsersMutex  sync.RWMutex                                      // Mutex for thread-safe access
	userUpdateStreams map[string]pb.ChatService_ActiveUsersStreamServer // Maps client ID to active users stream
}

// Login hanya menyimpan username
func (s *server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	log.Printf("User login: %s", req.Username)

	// Add user to active users list
	s.activeUsersMutex.Lock()
	// Check if user already exists before adding them
	isNewUser := !s.activeUsers[req.Username]
	s.activeUsers[req.Username] = true
	s.activeUsersMutex.Unlock()

	// Always broadcast the full user list after a new login
	go func() {
		// Small delay to ensure all initial setup is complete
		time.Sleep(200 * time.Millisecond)
		s.broadcastAllActiveUsers()
	}()

	// Additionally broadcast the join event if this is a new user
	if isNewUser {
		go s.broadcastUserJoin(req.Username)
	}

	return &pb.LoginResponse{
		Username: req.Username,
		Message:  "Login sukses",
	}, nil
}

// New helper function to broadcast the complete active users list to all clients
func (s *server) broadcastAllActiveUsers() {
	// Get a snapshot of current active users with proper locking
	s.activeUsersMutex.RLock()
	activeUsersList := make([]string, 0, len(s.activeUsers))
	for user := range s.activeUsers {
		activeUsersList = append(activeUsersList, user)
	}
	s.activeUsersMutex.RUnlock()

	log.Printf("Broadcasting full active users list to all clients: %v", activeUsersList)

	// Create the update message
	update := &pb.ActiveUsersUpdate{
		UpdateType: pb.ActiveUsersUpdate_FULL_LIST,
		Users:      activeUsersList,
	}

	// Send to all active streams
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, userStream := range s.userUpdateStreams {
		if err := userStream.Send(update); err != nil {
			log.Printf("Error sending full users list to stream %s: %v", id, err)
		} else {
			log.Printf("Successfully sent active users list to stream %s", id)
		}
	}
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

	// Remove from active users
	s.activeUsersMutex.Lock()
	delete(s.activeUsers, username)
	s.activeUsersMutex.Unlock()

	// Broadcast that user has left
	go s.broadcastUserLeave(username)

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

		// If this is a "joined the chat" message, add user to active users
		if msg.Message == "joined the chat" {
			s.activeUsersMutex.Lock()
			s.activeUsers[msg.Sender] = true
			s.activeUsersMutex.Unlock()

			// Broadcast to all active user streams
			go s.broadcastUserJoin(msg.Sender)
		} else if msg.Message == "left the chat" || msg.Message == "left the chat (client shutdown)" {
			// If this is a leave message, remove user from active users
			s.activeUsersMutex.Lock()
			delete(s.activeUsers, msg.Sender)
			s.activeUsersMutex.Unlock()

			// Broadcast to all active user streams
			go s.broadcastUserLeave(msg.Sender)
		}

		// Broadcast to all connected streams
		s.broadcastMessage(msg)
	}
}

// New method for streaming active users
func (s *server) ActiveUsersStream(req *pb.ActiveUsersRequest, stream pb.ChatService_ActiveUsersStreamServer) error {
	// Generate a unique ID for this stream
	streamID := fmt.Sprintf("active_%p", stream)
	log.Printf("New active users stream connected for user %s: %s", req.Username, streamID)

	// Register this stream
	s.mu.Lock()
	s.userUpdateStreams[streamID] = stream
	s.mu.Unlock()

	// Clean up on disconnect
	defer func() {
		s.mu.Lock()
		delete(s.userUpdateStreams, streamID)
		s.mu.Unlock()
		log.Printf("Active users stream disconnected: %s", streamID)
	}()

	// Get the current list of active users with proper locking
	s.activeUsersMutex.RLock()
	activeUsersList := make([]string, 0, len(s.activeUsers))
	for user := range s.activeUsers {
		activeUsersList = append(activeUsersList, user)
	}
	s.activeUsersMutex.RUnlock()

	log.Printf("Sending initial active users list to %s: %v", req.Username, activeUsersList)

	// Send initial full list
	err := stream.Send(&pb.ActiveUsersUpdate{
		UpdateType: pb.ActiveUsersUpdate_FULL_LIST,
		Users:      activeUsersList,
	})

	if err != nil {
		log.Printf("Error sending initial active users list: %v", err)
		return err
	}

	// Keep connection open until client disconnects
	<-stream.Context().Done()
	return nil
}

// Modified to ensure all clients get a full user list after any change
func (s *server) broadcastUserJoin(username string) {
	s.mu.Lock()

	// First send the JOIN notification
	update := &pb.ActiveUsersUpdate{
		UpdateType: pb.ActiveUsersUpdate_JOIN,
		Username:   username,
	}

	for id, userStream := range s.userUpdateStreams {
		if err := userStream.Send(update); err != nil {
			log.Printf("Error sending user join update to stream %s: %v", id, err)
		}
	}
	s.mu.Unlock()

	// Then broadcast the full list to ensure all clients are in sync
	go s.broadcastAllActiveUsers()
}

// Modified to ensure all clients get a full user list after any change
func (s *server) broadcastUserLeave(username string) {
	s.mu.Lock()

	// First send the LEAVE notification
	update := &pb.ActiveUsersUpdate{
		UpdateType: pb.ActiveUsersUpdate_LEAVE,
		Username:   username,
	}

	for id, userStream := range s.userUpdateStreams {
		if err := userStream.Send(update); err != nil {
			log.Printf("Error sending user leave update to stream %s: %v", id, err)
		}
	}
	s.mu.Unlock()

	// Then broadcast the full list to ensure all clients are in sync
	go s.broadcastAllActiveUsers()
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
		streams:           make(map[string]pb.ChatService_ChatStreamServer),
		userStreams:       make(map[string]string),
		messageCache:      make([]*pb.ChatMessage, 0, 100),
		activeUsers:       make(map[string]bool),
		userUpdateStreams: make(map[string]pb.ChatService_ActiveUsersStreamServer),
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
