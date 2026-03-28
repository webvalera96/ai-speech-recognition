// Package app defines Module, the Uber Fx graph for the meeting assistant: config, PostgreSQL
// (migrations then pool), services, SaluteSpeech/GigaChat, queue workers, Telegram, chi HTTP.
// Run from main: fx.New(app.Module).Run().
package app

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/webvalera96/ai-speech-recognition/internal/adapter/gigachat"
	"github.com/webvalera96/ai-speech-recognition/internal/adapter/httpadapter"
	"github.com/webvalera96/ai-speech-recognition/internal/adapter/queue"
	"github.com/webvalera96/ai-speech-recognition/internal/adapter/salutespeech"
	"github.com/webvalera96/ai-speech-recognition/internal/adapter/telegram"
	"github.com/webvalera96/ai-speech-recognition/internal/config"
	bothandler "github.com/webvalera96/ai-speech-recognition/internal/handlers/bot"
	"github.com/webvalera96/ai-speech-recognition/internal/migrate"
	"github.com/webvalera96/ai-speech-recognition/internal/repository/postgres"
	"github.com/webvalera96/ai-speech-recognition/internal/services"
)

// Module is the root fx.Module: wires Provide constructors and Invoke lifecycle hooks
// (workers, Telegram, HTTP). Pass it to fx.New from main.
var Module = fx.Module("app",
	fx.Provide(
		config.Load,
		newPool,
		postgres.NewUserRepository,
		postgres.NewMeetingRepository,
		asUserRepository,
		asMeetingRepository,
		services.NewUserService,
		services.NewMeetingService,
		salutespeech.NewClient,
		gigachat.NewClient,
		asTranscriber,
		asSummarizer,
		asChatCompleter,
		queue.NewQueue,
		asJobQueue,
		telegram.NewBot,
		telegram.NewNotifier,
		asNotifier,
		services.NewWorkflowService,
		newBotHandlers,
		httpadapter.NewRouter,
	),
	fx.Invoke(queue.RegisterWorkers),
	fx.Invoke(telegram.RegisterHandlers),
	fx.Invoke(telegram.RegisterLifecycle),
	fx.Invoke(httpadapter.RegisterHTTPServer),
)

// newPool runs embedded SQL migrations, opens a pgx pool for cfg.DatabaseURL, and
// registers OnStop to close the pool.
func newPool(lc fx.Lifecycle, cfg *config.Config) (*pgxpool.Pool, error) {
	if err := migrate.Up(cfg.DatabaseURL); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			pool.Close()
			return nil
		},
	})
	return pool, nil
}

// asUserRepository binds the concrete postgres user repo to services.UserRepository.
func asUserRepository(r *postgres.UserRepository) services.UserRepository { return r }

// asMeetingRepository binds the concrete postgres meeting repo to services.MeetingRepository.
func asMeetingRepository(r *postgres.MeetingRepository) services.MeetingRepository { return r }

// asTranscriber binds the SaluteSpeech client to services.Transcriber.
func asTranscriber(c *salutespeech.Client) services.Transcriber { return c }

// asSummarizer binds the GigaChat client to services.Summarizer.
func asSummarizer(c *gigachat.Client) services.Summarizer { return c }

// asChatCompleter binds the GigaChat client to services.ChatCompleter.
func asChatCompleter(c *gigachat.Client) services.ChatCompleter { return c }

// asJobQueue binds the in-process queue to services.JobQueue.
func asJobQueue(q *queue.Queue) services.JobQueue { return q }

// asNotifier binds the Telegram adapter to services.Notifier.
func asNotifier(n *telegram.Notifier) services.Notifier { return n }

// newBotHandlers returns bothandler.Handlers wired with user/meeting services and the job queue
// for Telegram command and media handling.
func newBotHandlers(
	users *services.UserService,
	meetings *services.MeetingService,
	jq services.JobQueue,
) *bothandler.Handlers {
	return &bothandler.Handlers{
		Users:    users,
		Meetings: meetings,
		Jobs:     jq,
	}
}
