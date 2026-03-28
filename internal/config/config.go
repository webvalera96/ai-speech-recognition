package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds runtime configuration loaded from the environment.
type Config struct {
	TelegramBotToken string
	DatabaseURL      string
	HTTPAddr         string
	WorkerPool       int

	// GigaChat
	GigaChatAuthURL      string
	GigaChatAPIURL       string
	GigaChatClientID     string
	GigaChatClientSecret string
	GigaChatScope        string

	// SaluteSpeech / SmartSpeech
	SaluteSpeechRESTURL  string
	SaluteSpeechAuthURL  string
	SaluteSpeechClientID string
	SaluteSpeechSecret   string
	SaluteSpeechScope    string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	tg := os.Getenv("TELEGRAM_BOT_TOKEN")
	if tg == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	db := os.Getenv("DATABASE_URL")
	if db == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}
	workers := 3
	if v := os.Getenv("WORKER_POOL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			workers = n
		}
	}

	c := &Config{
		TelegramBotToken:     tg,
		DatabaseURL:          db,
		HTTPAddr:             httpAddr,
		WorkerPool:           workers,
		GigaChatAuthURL:      getenvDefault("GIGACHAT_AUTH_URL", "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"),
		GigaChatAPIURL:       getenvDefault("GIGACHAT_API_URL", "https://gigachat.devices.sberbank.ru/api/v1/chat/completions"),
		GigaChatClientID:     os.Getenv("GIGACHAT_CLIENT_ID"),
		GigaChatClientSecret: os.Getenv("GIGACHAT_CLIENT_SECRET"),
		GigaChatScope:        getenvDefault("GIGACHAT_SCOPE", "GIGACHAT_API_PERS"),
		SaluteSpeechRESTURL:  getenvDefault("SALUTESPEECH_REST_URL", "https://smartspeech.sber.ru/rest/v1"),
		SaluteSpeechAuthURL:  getenvDefault("SALUTESPEECH_AUTH_URL", "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"),
		SaluteSpeechClientID: os.Getenv("SALUTESPEECH_CLIENT_ID"),
		SaluteSpeechSecret:   os.Getenv("SALUTESPEECH_CLIENT_SECRET"),
		SaluteSpeechScope:    getenvDefault("SALUTESPEECH_SCOPE", "SALUTE_SPEECH_PERS"),
	}
	if c.GigaChatClientID == "" || c.GigaChatClientSecret == "" {
		return nil, fmt.Errorf("GIGACHAT_CLIENT_ID and GIGACHAT_CLIENT_SECRET are required")
	}
	if c.SaluteSpeechClientID == "" || c.SaluteSpeechSecret == "" {
		return nil, fmt.Errorf("SALUTESPEECH_CLIENT_ID and SALUTESPEECH_CLIENT_SECRET are required")
	}
	return c, nil
}

// getenvDefault returns the environment variable value if set, otherwise returns the default value.
func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
