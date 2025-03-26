package repository

import (
	"sync"
	"time"

	"grpc-chat/common/models"
)

type MessageRepository struct {
	mu       sync.Mutex
	messages []models.Message
}

func NewMessageRepository() *MessageRepository {
	return &MessageRepository{
		messages: []models.Message{},
	}
}

func (r *MessageRepository) SaveMessage(username, content string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = append(r.messages, models.Message{
		Username:  username,
		Content:   content,
		Timestamp: time.Now(),
	})
}

func (r *MessageRepository) GetMessages() []models.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.messages
}
