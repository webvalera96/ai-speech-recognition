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
	runes := []rune(s)
	if len(runes) <= max {
		return []string{s}
	}
	var out []string
	for len(runes) > 0 {
		if len(runes) <= max {
			out = append(out, string(runes))
			break
		}
		out = append(out, string(runes[:max]))
		runes = runes[max:]
	}
	return out
}
