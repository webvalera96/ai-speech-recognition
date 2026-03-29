package gigachat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/webvalera96/ai-speech-recognition/internal/config"
	"github.com/webvalera96/ai-speech-recognition/internal/services"
)

// Client implements Summarizer and ChatCompleter for GigaChat API.
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
	mu         sync.Mutex
	token      string
	expiresAt  time.Time
}

// NewClient constructs a GigaChat HTTP client.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

var (
	_ services.Summarizer    = (*Client)(nil)
	_ services.ChatCompleter = (*Client)(nil)
)

// SummarizeMeeting implements services.Summarizer.
func (c *Client) SummarizeMeeting(ctx context.Context, transcript string) (string, error) {
	system := "Ты помощник для распознавания аудио встреч. Дай краткую выжимку на русском: ключевые темы, решения, action items. Не превышай 2000 символов."
	user := "Транскрипт встречи:\n\n" + transcript
	return c.chat(ctx, system, user)
}

// Complete implements services.ChatCompleter.
func (c *Client) Complete(ctx context.Context, userMessage string) (string, error) {
	system := "Ты полезный ассистент. Отвечай по-русски кратко и по делу."
	return c.chat(ctx, system, userMessage)
}

func (c *Client) chat(ctx context.Context, system, user string) (string, error) {
	if err := c.ensureToken(ctx); err != nil {
		return "", err
	}
	body := map[string]any{
		"model": "GigaChat",
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.9,
		"max_tokens":  2000,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.GigaChatAPIURL, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("RqUID", uuid.NewString())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gigachat chat: status %d: %s", resp.StatusCode, truncate(respBody, 500))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("gigachat json: %w", err)
	}
	if len(out.Choices) == 0 || out.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty gigachat response")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

func (c *Client) ensureToken(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expiresAt.Add(-2*time.Minute)) {
		return nil
	}
	form := url.Values{}
	form.Set("scope", c.cfg.GigaChatScope)
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.GigaChatAuthURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("RqUID", uuid.NewString())
	key := base64.StdEncoding.EncodeToString([]byte(c.cfg.GigaChatClientID + ":" + c.cfg.GigaChatClientSecret))
	req.Header.Set("Authorization", "Basic "+key)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gigachat token: status %d: %s", resp.StatusCode, truncate(b, 500))
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresAt   int64  `json:"expires_at"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(b, &tr); err != nil {
		return fmt.Errorf("token json: %w", err)
	}
	if tr.AccessToken == "" {
		return fmt.Errorf("empty access_token")
	}
	c.token = tr.AccessToken
	if tr.ExpiresAt > 0 {
		c.expiresAt = time.UnixMilli(tr.ExpiresAt)
	} else if tr.ExpiresIn > 0 {
		c.expiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	} else {
		c.expiresAt = time.Now().Add(25 * time.Minute)
	}
	return nil
}

func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
