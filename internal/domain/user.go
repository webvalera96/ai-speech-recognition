package domain

import "time"

// User is a registered Telegram user.
type User struct {
	TelegramUserID int64
	CreatedAt      time.Time
}
