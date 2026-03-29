package services

// JobType identifies async work processed by workers.
type JobType int

const (
	JobTranscribe JobType = iota
	JobChat
)

// Job carries async work for a specific Telegram user/chat.
type Job struct {
	Type           JobType
	TelegramUserID int64
	ChatID         int64
	// Transcribe
	Audio    []byte
	FileName string
	// Chat
	ChatPrompt string
}
