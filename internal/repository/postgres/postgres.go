package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/webvalera96/ai-speech-recognition/internal/domain"
	"github.com/webvalera96/ai-speech-recognition/internal/services"
)

// UserRepository implements services.UserRepository.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository constructs UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Upsert implements services.UserRepository.
func (r *UserRepository) Upsert(ctx context.Context, telegramUserID int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (telegram_user_id) VALUES ($1)
		ON CONFLICT (telegram_user_id) DO NOTHING
	`, telegramUserID)
	return err
}

// Exists implements services.UserRepository.
func (r *UserRepository) Exists(ctx context.Context, telegramUserID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE telegram_user_id = $1)
	`, telegramUserID).Scan(&exists)
	return exists, err
}

// MeetingRepository implements services.MeetingRepository.
type MeetingRepository struct {
	pool *pgxpool.Pool
}

// NewMeetingRepository constructs MeetingRepository.
func NewMeetingRepository(pool *pgxpool.Pool) *MeetingRepository {
	return &MeetingRepository{pool: pool}
}

// Create implements services.MeetingRepository.
func (r *MeetingRepository) Create(ctx context.Context, userID int64, transcript, summary string) (*domain.Meeting, error) {
	var m domain.Meeting
	err := r.pool.QueryRow(ctx, `
		INSERT INTO meetings (user_id, transcript, summary)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, transcript, summary, created_at
	`, userID, transcript, summary).Scan(&m.ID, &m.UserID, &m.Transcript, &m.Summary, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// ListByUser implements services.MeetingRepository.
func (r *MeetingRepository) ListByUser(ctx context.Context, userID int64, limit int) ([]domain.Meeting, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, transcript, summary, created_at
		FROM meetings
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMeetings(rows)
}

// GetByIDForUser implements services.MeetingRepository.
func (r *MeetingRepository) GetByIDForUser(ctx context.Context, id, userID int64) (*domain.Meeting, error) {
	var m domain.Meeting
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, transcript, summary, created_at
		FROM meetings
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&m.ID, &m.UserID, &m.Transcript, &m.Summary, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// SearchByUser implements services.MeetingRepository using full-text search.
func (r *MeetingRepository) SearchByUser(ctx context.Context, userID int64, keyword string, limit int) ([]domain.Meeting, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, transcript, summary, created_at
		FROM meetings
		WHERE user_id = $1
		  AND to_tsvector('russian', transcript) @@ plainto_tsquery('russian', $2)
		ORDER BY created_at DESC
		LIMIT $3
	`, userID, keyword, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMeetings(rows)
}

type meetingRows interface {
	Next() bool
	Scan(dest ...any) error
	Close()
	Err() error
}

func scanMeetings(rows meetingRows) ([]domain.Meeting, error) {
	var list []domain.Meeting
	for rows.Next() {
		var m domain.Meeting
		if err := rows.Scan(&m.ID, &m.UserID, &m.Transcript, &m.Summary, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

var (
	_ services.UserRepository    = (*UserRepository)(nil)
	_ services.MeetingRepository = (*MeetingRepository)(nil)
)
