package telegram

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	tele "gopkg.in/telebot.v3"
	"go.uber.org/fx"

	"github.com/webvalera96/ai-speech-recognition/internal/config"
	bothandler "github.com/webvalera96/ai-speech-recognition/internal/handlers/bot"
)

// NewBot creates a telebot instance (polling not started).
func NewBot(cfg *config.Config) (*tele.Bot, error) {
	pref := tele.Settings{
		Token:  cfg.TelegramBotToken,
		Poller: &tele.LongPoller{Timeout: 30 * time.Second},
	}
	return tele.NewBot(pref)
}

// RegisterHandlers attaches OnText, OnVoice, OnAudio.
func RegisterHandlers(bot *tele.Bot, h *bothandler.Handlers) {
	bot.Handle(tele.OnText, func(c tele.Context) error {
		ctx := context.Background()
		msgs, err := h.HandleText(ctx, c.Sender().ID, c.Chat().ID, c.Text())
		if err != nil {
			slog.Error("handle text", "err", err)
			_ = c.Send("Something went wrong. Please try again.")
			return nil
		}
		for _, m := range msgs {
			if err := c.Send(m); err != nil {
				return err
			}
		}
		return nil
	})

	bot.Handle(tele.OnVoice, func(c tele.Context) error {
		return handleMedia(h, c, c.Message().Voice.File, "voice.ogg")
	})

	bot.Handle(tele.OnAudio, func(c tele.Context) error {
		f := c.Message().Audio.File
		name := c.Message().Audio.FileName
		if name == "" {
			name = "audio.mp3"
		}
		return handleMedia(h, c, f, name)
	})
}

// RegisterLifecycle starts and stops telebot polling.
func RegisterLifecycle(lc fx.Lifecycle, bot *tele.Bot) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go bot.Start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			bot.Stop()
			return nil
		},
	})
}

func handleMedia(h *bothandler.Handlers, c tele.Context, f tele.File, name string) error {
	reader, err := c.Bot().File(&f)
	if err != nil {
		_ = c.Send("Could not download the file.")
		return nil
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		_ = c.Send("Could not read the file.")
		return nil
	}
	ctx := context.Background()
	msgs, err := h.HandleAudio(ctx, c.Sender().ID, c.Chat().ID, data, name)
	if err != nil {
		slog.Error("handle audio", "err", err)
		_ = c.Send("Failed to queue audio.")
		return nil
	}
	for _, m := range msgs {
		if err := c.Send(m); err != nil {
			return fmt.Errorf("send: %w", err)
		}
	}
	return nil
}
