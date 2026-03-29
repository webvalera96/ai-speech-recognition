# Помощник для конспектирования встреч (Telegram)

Telegram-бот: транскрибация аудио встреч через **SaluteSpeech**, краткие выжимки текста через **GigaChat**, хранение в **PostgreSQL**, поиск по ключевым словам и вопросы к модели командой `/chat`.

## Архитектура

- **Handlers** (`internal/handlers`) — тонкие точки входа: Telegram и HTTP.
- **Services** (`internal/services`) — бизнес-логика и интерфейсы-порты.
- **Repository** (`internal/repository/postgres`) — SQL через `pgx`.
- **Adapters** (`internal/adapter`) — SaluteSpeech, GigaChat, telebot, chi HTTP, очередь задач в памяти.

При старте бинарник применяет **встроенные SQL-миграции** (`internal/migrate/migrations`) к базе из `DATABASE_URL`

## Технологический стек

- Go **1.24+**
- PostgreSQL 14+ (или совместимая СУБД)
- Telegram бот @ai_speech_recognition_bot ([BotFather](https://core.telegram.org/bots/tutorial))
- API SaluteSpeech и GigaChat в [ссылка](https://developers.sber.ru/)

## Переменные окружения (конфигурация)

| Переменная | Обязательно | Описание |
|------------|-------------|----------|
| `TELEGRAM_BOT_TOKEN` | да | Токен Telegram-бота |
| `DATABASE_URL` | да | Строка подключения PostgreSQL (например `postgres://user:pass@localhost:5432/meetings?sslmode=disable`) |
| `HTTP_ADDR` | нет | Адрес HTTP-сервера (по умолчанию `:8080`) |
| `WORKER_POOL` | нет | Число воркеров фоновых задач (по умолчанию `3`) |
| `FX_LOG` | нет | Лог старта Uber Fx в stdout: `debug`, `true`, `1`, `yes` |
| `LOG_LEVEL` | нет | Если `debug` — то же, что включённый лог Fx |
| `FX_DEBUG` | нет | `1` — включить лог Fx в stdout |
| `GIGACHAT_CLIENT_ID` | да | OAuth client id GigaChat |
| `GIGACHAT_CLIENT_SECRET` | да | OAuth client secret GigaChat |
| `GIGACHAT_AUTH_URL` | нет | Эндпоинт токена (по умолчанию Sber `.../api/v2/oauth`) |
| `GIGACHAT_API_URL` | нет | URL chat completions |
| `GIGACHAT_SCOPE` | нет | OAuth scope (по умолчанию `GIGACHAT_API_PERS`) |
| `SALUTESPEECH_CLIENT_ID` | да | OAuth client id SaluteSpeech |
| `SALUTESPEECH_CLIENT_SECRET` | да | OAuth client secret SaluteSpeech |
| `SALUTESPEECH_REST_URL` | нет | Базовый URL SmartSpeech REST (по умолчанию `https://smartspeech.sber.ru/rest/v1`) |
| `SALUTESPEECH_AUTH_URL` | нет | URL получения OAuth-токена |
| `SALUTESPEECH_SCOPE` | нет | OAuth scope (по умолчанию `SALUTE_SPEECH_PERS`) |

Пример `.env.example`

Для запуска скопировать и заполнить `.env`

## Запуск

```bash
go run ./cmd/bot
```

Сборка:

```bash
go build -o bot ./cmd/bot
```

### Task 

| Команда | Описание |
|---------|----------|
| `task build` | Сборка бота в `bin/` (`bot` или `bot.exe`) |
| `task run` | `go run ./cmd/bot` (подхватывает `.env` в корне репозитория, если есть) |
| `task vet` | `go vet ./...` |
| `task docker:up` | Запуск PostgreSQL через `docker/docker-compose.yaml` |
| `task docker:down` | Остановка контейнеров из compose |
| `task docker:logs` | Логи PostgreSQL в режиме follow |

Строка подключения к БД для базы докер

`postgres://meetings:meetings@localhost:5432/meetings?sslmode=disable`

Проверки работоспособности после старта:

- `GET /health` — процесс жив.
- `GET /ready` — успешный ping PostgreSQL.

## Команды бота

| Команда | Описание |
|---------|----------|
| `/start` | Регистрация пользователя |
| `/list` | Список сохранённых встреч |
| `/get <id>` | Полный транскрипт |
| `/find <keyword>` | Поиск по транскриптам (полнотекстовый, русский) |
| `/chat <текст>` | Вопрос к GigaChat |
| Голос / аудиофайл | Транскрибация + выжимка + сохранение |
