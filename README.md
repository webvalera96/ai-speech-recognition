# Meeting assistant (Telegram)

Telegram bot that transcribes meeting audio with **SaluteSpeech**, summarizes text with **GigaChat**, stores transcripts in **PostgreSQL**, and supports keyword search and ad-hoc questions via `/chat`.

## Architecture

- **Handlers** (`internal/handlers`) — thin Telegram and HTTP entrypoints.
- **Services** (`internal/services`) — business logic and port interfaces.
- **Repository** (`internal/repository/postgres`) — SQL via `pgx`.
- **Adapters** (`internal/adapter`) — SaluteSpeech, GigaChat, telebot, chi HTTP, in-process job queue.
- **Composition** — [Uber Fx](https://uber-go.github.io/fx/) (`internal/app`).

On startup the binary runs **embedded SQL migrations** (`internal/migrate/migrations`) against `DATABASE_URL` before opening the connection pool.

## Requirements

- Go **1.24+**
- PostgreSQL 14+ (or compatible)
- Telegram bot token ([BotFather](https://core.telegram.org/bots/tutorial))
- SaluteSpeech and GigaChat API credentials from [Sber developer portal](https://developers.sber.ru/)

## Environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | yes | Telegram bot token |
| `DATABASE_URL` | yes | PostgreSQL URL (e.g. `postgres://user:pass@localhost:5432/meetings?sslmode=disable`) |
| `HTTP_ADDR` | no | HTTP listen address (default `:8080`) |
| `WORKER_POOL` | no | Concurrent job workers (default `3`) |
| `GIGACHAT_CLIENT_ID` | yes | GigaChat OAuth client id |
| `GIGACHAT_CLIENT_SECRET` | yes | GigaChat OAuth client secret |
| `GIGACHAT_AUTH_URL` | no | Token endpoint (default Sber `.../api/v2/oauth`) |
| `GIGACHAT_API_URL` | no | Chat completions URL |
| `GIGACHAT_SCOPE` | no | OAuth scope (default `GIGACHAT_API_PERS`) |
| `SALUTESPEECH_CLIENT_ID` | yes | SaluteSpeech OAuth client id |
| `SALUTESPEECH_CLIENT_SECRET` | yes | SaluteSpeech OAuth client secret |
| `SALUTESPEECH_REST_URL` | no | SmartSpeech REST base (default `https://smartspeech.sber.ru/rest/v1`) |
| `SALUTESPEECH_AUTH_URL` | no | OAuth token URL |
| `SALUTESPEECH_SCOPE` | no | OAuth scope (default `SALUTE_SPEECH_PERS`) |

Copy `.env.example` and fill in secrets. **Do not commit real credentials.**

## Run

```bash
go run ./cmd/bot
```

Build:

```bash
go build -o bot ./cmd/bot
```

### Task (optional)

With [Task](https://taskfile.dev/) installed:

| Command | Description |
|---------|-------------|
| `task build` | Compile the bot into `bin/` (`bot` or `bot.exe`) |
| `task run` | `go run ./cmd/bot` (loads `.env` from the repo root if it exists) |
| `task docker:up` | Start PostgreSQL via `docker/docker-compose.yaml` |
| `task docker:down` | Stop containers from that compose file |
| `task docker:logs` | Tail Postgres logs |

Default database URL for the bundled compose stack:

`postgres://meetings:meetings@localhost:5432/meetings?sslmode=disable`

Health checks (after start):

- `GET /health` — process is up.
- `GET /ready` — PostgreSQL ping succeeds.

## Bot commands

| Command | Description |
|---------|-------------|
| `/start` | Register user |
| `/list` | List saved meetings |
| `/get <id>` | Full transcript |
| `/find <keyword>` | Search transcripts (Russian full-text) |
| `/chat <text>` | Ask GigaChat |
| Voice / audio file | Transcribe + summarize + save |

## SaluteSpeech REST notes

The client uses async flow: upload → `speech:asyncRecognize` → `task:get` polling → `data:download`. If Sber changes paths or JSON field names, adjust `internal/adapter/salutespeech/client.go` to match the current [documentation](https://developers.sber.ru/docs/ru/salutespeech/rest/salutespeech-rest-api).

## Migrations

Add versioned files under `internal/migrate/migrations/` (`NNNNNN_name.up.sql` / `.down.sql`). They are applied automatically on each application start.

## License

MIT (or your choice — update this line for your course/project).
