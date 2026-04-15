package services

import (
	"context"

	"github.com/webvalera96/ai-speech-recognition/internal/domain"
)

// UserRepository persists Telegram users.
type UserRepository interface {
	Upsert(ctx context.Context, telegramUserID int64) error
	Exists(ctx context.Context, telegramUserID int64) (bool, error)
}

// MeetingRepository stores meeting transcripts and summaries.
type MeetingRepository interface {
	Create(ctx context.Context, userID int64, transcript, summary string) (*domain.Meeting, error)
	ListByUser(ctx context.Context, userID int64, limit int) ([]domain.Meeting, error)
	GetByIDForUser(ctx context.Context, id, userID int64) (*domain.Meeting, error)
	SearchByUser(ctx context.Context, userID int64, keyword string, limit int) ([]domain.Meeting, error)
}

// Transcriber converts audio bytes to plain text (SaluteSpeech).
type Transcriber interface {
	Transcribe(ctx context.Context, audio []byte, filename string) (text string, err error)
}

// Summarizer produces a short summary from transcript text (GigaChat).
type Summarizer interface {
	SummarizeMeeting(ctx context.Context, transcript string) (summary string, err error)
}

// ChatCompleter answers a free-form user question (GigaChat).
type ChatCompleter interface {
	Complete(ctx context.Context, userMessage string) (reply string, err error)
}

// Notifier sends a text message to a Telegram chat.
type Notifier interface {
	SendText(ctx context.Context, chatID int64, text string) error
}

// JobQueue accepts background jobs (transcription, chat).
type JobQueue interface {
	Enqueue(ctx context.Context, job Job) error
}
