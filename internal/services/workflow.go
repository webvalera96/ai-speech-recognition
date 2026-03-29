package services

import (
	"context"
	"fmt"
	"strings"
)

// WorkflowService runs long-running pipelines invoked from queue workers.
type WorkflowService struct {
	transcribe Transcriber
	summarize  Summarizer
	chat       ChatCompleter
	meetings   MeetingRepository
	notify     Notifier
}

// NewWorkflowService constructs WorkflowService.
func NewWorkflowService(
	t Transcriber,
	s Summarizer,
	c ChatCompleter,
	m MeetingRepository,
	n Notifier,
) *WorkflowService {
	return &WorkflowService{
		transcribe: t,
		summarize:  s,
		chat:       c,
		meetings:   m,
		notify:     n,
	}
}

// ProcessTranscribe transcribes audio, summarizes, saves a meeting, notifies the user.
func (w *WorkflowService) ProcessTranscribe(ctx context.Context, job Job) error {
	if len(job.Audio) == 0 {
		return fmt.Errorf("empty audio")
	}
	text, err := w.transcribe.Transcribe(ctx, job.Audio, job.FileName)
	if err != nil {
		return fmt.Errorf("transcribe: %w", err)
	}
	summary, err := w.summarize.SummarizeMeeting(ctx, text)
	if err != nil {
		return fmt.Errorf("summarize: %w", err)
	}
	m, err := w.meetings.Create(ctx, job.TelegramUserID, text, summary)
	if err != nil {
		return fmt.Errorf("save meeting: %w", err)
	}
	msg := fmt.Sprintf("Summary:\n%s\n\nSaved as meeting #%d.", summary, m.ID)
	return w.notify.SendText(ctx, job.ChatID, msg)
}

// ProcessChat sends a prompt to GigaChat and replies in Telegram.
func (w *WorkflowService) ProcessChat(ctx context.Context, job Job) error {
	prompt := strings.TrimSpace(job.ChatPrompt)
	if prompt == "" {
		return fmt.Errorf("empty chat prompt")
	}
	reply, err := w.chat.Complete(ctx, prompt)
	if err != nil {
		return fmt.Errorf("chat: %w", err)
	}
	return w.notify.SendText(ctx, job.ChatID, reply)
}

// NotifyError sends an error message to the user's Telegram chat (used by workers).
func (w *WorkflowService) NotifyError(ctx context.Context, chatID int64, msg string) error {
	return w.notify.SendText(ctx, chatID, msg)
}
