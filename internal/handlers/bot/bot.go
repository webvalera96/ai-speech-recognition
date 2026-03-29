package bothandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/webvalera96/ai-speech-recognition/internal/services"
)

// Handlers wires Telegram events to application services.
type Handlers struct {
	Users    *services.UserService
	Meetings *services.MeetingService
	Jobs     services.JobQueue
}

// HandleText processes plain text and bot commands. Returns one or more message chunks to send.
func (h *Handlers) HandleText(ctx context.Context, telegramUserID, chatID int64, text string) ([]string, error) {
	line := strings.TrimSpace(text)
	if line == "" {
		return nil, nil
	}

	switch {
	case strings.HasPrefix(line, "/start"):
		if err := h.Users.Register(ctx, telegramUserID); err != nil {
			return nil, err
		}
		return []string{"Welcome! Send voice or audio to transcribe and summarize.\nCommands: /list, /get <id>, /find <keyword>, /chat <question>."}, nil

	case strings.HasPrefix(line, "/list"):
		meetings, err := h.Meetings.List(ctx, telegramUserID, 20)
		if err != nil {
			return nil, err
		}
		if len(meetings) == 0 {
			return []string{"No saved meetings yet."}, nil
		}
		var b strings.Builder
		b.WriteString("Your meetings:\n")
		for _, m := range meetings {
			preview := m.Summary
			if len(preview) > 80 {
				preview = preview[:80] + "…"
			}
			b.WriteString(fmt.Sprintf("#%d — %s — %s\n", m.ID, m.CreatedAt.UTC().Format(time.RFC3339), preview))
		}
		return []string{b.String()}, nil

	case strings.HasPrefix(line, "/get"):
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return []string{"Usage: /get <id>"}, nil
		}
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return []string{"Invalid meeting id."}, nil
		}
		m, err := h.Meetings.Get(ctx, telegramUserID, id)
		if err != nil {
			return []string{"Meeting not found."}, nil
		}
		return splitTelegramMessages("Transcript:\n\n" + m.Transcript), nil

	case strings.HasPrefix(line, "/find"):
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			return []string{"Usage: /find <keyword>"}, nil
		}
		keyword := strings.TrimSpace(parts[1])
		meetings, err := h.Meetings.Find(ctx, telegramUserID, keyword, 15)
		if err != nil {
			return nil, err
		}
		if len(meetings) == 0 {
			return []string{"No meetings matched."}, nil
		}
		var b strings.Builder
		b.WriteString("Matches:\n")
		for _, m := range meetings {
			b.WriteString(fmt.Sprintf("#%d (%s)\n", m.ID, m.CreatedAt.UTC().Format(time.RFC3339)))
		}
		return []string{b.String()}, nil

	case strings.HasPrefix(line, "/chat"):
		prompt := strings.TrimSpace(strings.TrimPrefix(line, "/chat"))
		if prompt == "" {
			return []string{"Usage: /chat <your question>"}, nil
		}
		job := services.Job{
			Type:           services.JobChat,
			TelegramUserID: telegramUserID,
			ChatID:         chatID,
			ChatPrompt:     prompt,
		}
		if err := h.Jobs.Enqueue(ctx, job); err != nil {
			return nil, err
		}
		return []string{"Thinking…"}, nil
	}

	return []string{"Unknown command. Try /start for help."}, nil
}

// HandleAudio enqueues transcription for voice or audio attachment.
func (h *Handlers) HandleAudio(ctx context.Context, telegramUserID, chatID int64, audio []byte, fileName string) ([]string, error) {
	if len(audio) == 0 {
		return []string{"Empty audio file."}, nil
	}
	job := services.Job{
		Type:           services.JobTranscribe,
		TelegramUserID: telegramUserID,
		ChatID:         chatID,
		Audio:          audio,
		FileName:       fileName,
	}
	if err := h.Jobs.Enqueue(ctx, job); err != nil {
		return nil, err
	}
	return []string{"Processing audio, please wait…"}, nil
}

func splitTelegramMessages(s string) []string {
	const max = 4000
	if len(s) <= max {
		return []string{s}
	}
	var out []string
	for len(s) > 0 {
		if len(s) <= max {
			out = append(out, s)
			break
		}
		out = append(out, s[:max])
		s = s[max:]
	}
	return out
}
