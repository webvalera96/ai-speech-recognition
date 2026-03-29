package services

import "context"

// UserService handles /start registration.
type UserService struct {
	users UserRepository
}

// NewUserService constructs UserService.
func NewUserService(users UserRepository) *UserService {
	return &UserService{users: users}
}

// Register ensures the user exists in the database.
func (s *UserService) Register(ctx context.Context, telegramUserID int64) error {
	return s.users.Upsert(ctx, telegramUserID)
}
