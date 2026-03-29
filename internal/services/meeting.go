package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/webvalera96/ai-speech-recognition/internal/domain"
)

// MeetingService handles listing, fetching, and searching meetings.
type MeetingService struct {
	meetings MeetingRepository
}

// NewMeetingService constructs MeetingService.
func NewMeetingService(meetings MeetingRepository) *MeetingService {
	return &MeetingService{meetings: meetings}
}

// List returns recent meetings for a user, newest first.
func (s *MeetingService) List(ctx context.Context, telegramUserID int64, limit int) ([]domain.Meeting, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.meetings.ListByUser(ctx, telegramUserID, limit)
}

// Get returns a meeting transcript if it belongs to the user.
func (s *MeetingService) Get(ctx context.Context, telegramUserID, meetingID int64) (*domain.Meeting, error) {
	return s.meetings.GetByIDForUser(ctx, meetingID, telegramUserID)
}

// Find searches transcripts by keyword for the user.
func (s *MeetingService) Find(ctx context.Context, telegramUserID int64, keyword string, limit int) ([]domain.Meeting, error) {
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return nil, fmt.Errorf("empty keyword")
	}
	if limit <= 0 {
		limit = 20
	}
	return s.meetings.SearchByUser(ctx, telegramUserID, kw, limit)
}
