package repository

import (
	"sync"

	"grpc-chat/common/models"
)

type UserRepository struct {
	mu    sync.Mutex
	users map[string]*models.User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		users: make(map[string]*models.User),
	}
}

func (r *UserRepository) SetOnline(username string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[username] = &models.User{Username: username, Online: true}
}

func (r *UserRepository) SetOffline(username string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if user, exists := r.users[username]; exists {
		user.Online = false
	}
}

func (r *UserRepository) GetAllUsers() []*models.User {
	r.mu.Lock()
	defer r.mu.Unlock()

	var users []*models.User
	for _, user := range r.users {
		users = append(users, user)
	}
	return users
}
