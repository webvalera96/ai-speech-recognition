package telegram

import (
	"context"
	"strings"

	tele "gopkg.in/telebot.v3"

	"github.com/webvalera96/ai-speech-recognition/internal/services"
)

// Notifier sends Telegram messages using telebot.
type Notifier struct {
	bot *tele.Bot
}

// NewNotifier constructs Notifier.
func NewNotifier(bot *tele.Bot) *Notifier {
	return &Notifier{bot: bot}
}

// SendText implements services.Notifier.
func (n *Notifier) SendText(ctx context.Context, chatID int64, text string) error {
	chat := &tele.Chat{ID: chatID}
	for _, chunk := range splitChunks(text, 4000) {
		if _, err := n.bot.Send(chat, chunk); err != nil {
			return err
		}
	}
	return nil
}

var _ services.Notifier = (*Notifier)(nil)

func splitChunks(s string, max int) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{""}
	}
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
