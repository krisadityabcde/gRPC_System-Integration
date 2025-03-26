package service

import (
	"context"
	"fmt"
	"sync"

	"grpc-chat/common/repository"
	pb "grpc-chat/proto"
)

type ChatService struct {
	pb.UnimplementedChatServiceServer
	userRepo    *repository.UserRepository
	messageRepo *repository.MessageRepository
	mu          sync.Mutex
	clients     map[string]chan *pb.OnlineStatusResponse
}

func NewChatService(userRepo *repository.UserRepository, messageRepo *repository.MessageRepository) *ChatService {
	return &ChatService{
		userRepo:    userRepo,
		messageRepo: messageRepo,
		clients:     make(map[string]chan *pb.OnlineStatusResponse),
	}
}

func (s *ChatService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	username := req.Username

	s.userRepo.SetOnline(username)

	s.mu.Lock()
	s.clients[username] = make(chan *pb.OnlineStatusResponse, 1)
	s.mu.Unlock()

	go s.broadcastOnlineStatus(username, true)

	return &pb.LoginResponse{Message: fmt.Sprintf("Welcome, %s!", username)}, nil
}

func (s *ChatService) OnlineStatus(req *pb.OnlineStatusRequest, stream pb.ChatService_OnlineStatusServer) error {
	username := fmt.Sprintf("User-%d", len(s.clients)+1)

	clientChan := make(chan *pb.OnlineStatusResponse)
	s.mu.Lock()
	s.clients[username] = clientChan
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, username)
		s.mu.Unlock()
	}()

	for status := range clientChan {
		if err := stream.Send(status); err != nil {
			return err
		}
	}
	return nil
}

func (s *ChatService) broadcastOnlineStatus(username string, online bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, client := range s.clients {
		client <- &pb.OnlineStatusResponse{
			Username: username,
			Online:   online,
		}
	}
}
