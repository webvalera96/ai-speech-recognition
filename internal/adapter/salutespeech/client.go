package salutespeech

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
	"time"

	"github.com/google/uuid"

	"github.com/webvalera96/ai-speech-recognition/internal/config"
	"github.com/webvalera96/ai-speech-recognition/internal/services"
)

// Client implements services.Transcriber using SaluteSpeech REST (async).
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
	token      string
	tokenExp   time.Time
}

// NewClient constructs a SaluteSpeech client.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

var _ services.Transcriber = (*Client)(nil)

// Transcribe uploads audio, runs async recognition, polls until done, returns transcript text.
func (c *Client) Transcribe(ctx context.Context, audio []byte, filename string) (string, error) {
	if err := c.ensureToken(ctx); err != nil {
		return "", err
	}
	fileID, err := c.upload(ctx, audio)
	if err != nil {
		return "", fmt.Errorf("upload: %w", err)
	}
	taskID, err := c.createRecognizeTask(ctx, fileID, filename)
	if err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}
	respFileID, err := c.waitTask(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("wait task: %w", err)
	}
	raw, err := c.download(ctx, respFileID)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	return parseTranscriptJSON(raw)
}

func (c *Client) ensureToken(ctx context.Context) error {
	if c.token != "" && time.Now().Before(c.tokenExp.Add(-2*time.Minute)) {
		return nil
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("scope", c.cfg.SaluteSpeechScope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.SaluteSpeechAuthURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("RqUID", uuid.NewString())
	key := base64.StdEncoding.EncodeToString([]byte(c.cfg.SaluteSpeechClientID + ":" + c.cfg.SaluteSpeechSecret))
	req.Header.Set("Authorization", "Basic "+key)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("salutespeech token: status %d: %s", resp.StatusCode, truncate(body, 500))
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return fmt.Errorf("token json: %w", err)
	}
	if tr.AccessToken == "" {
		return fmt.Errorf("empty access_token")
	}
	c.token = tr.AccessToken
	if tr.ExpiresIn > 0 {
		c.tokenExp = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	} else {
		c.tokenExp = time.Now().Add(25 * time.Minute)
	}
	return nil
}

func (c *Client) upload(ctx context.Context, audio []byte) (string, error) {
	u := strings.TrimSuffix(c.cfg.SaluteSpeechRESTURL, "/") + "/data:upload"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(audio))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("RqUID", uuid.NewString())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, truncate(body, 500))
	}
	var wrap struct {
		Result struct {
			RequestFileID string `json:"request_file_id"`
		} `json:"result"`
		RequestFileID string `json:"request_file_id"`
	}
	_ = json.Unmarshal(body, &wrap)
	id := wrap.Result.RequestFileID
	if id == "" {
		id = wrap.RequestFileID
	}
	if id == "" {
		return "", fmt.Errorf("no request_file_id in response: %s", truncate(body, 300))
	}
	return id, nil
}

func (c *Client) createRecognizeTask(ctx context.Context, requestFileID, filename string) (string, error) {
	u := strings.TrimSuffix(c.cfg.SaluteSpeechRESTURL, "/") + "/speech:asyncRecognize"
	enc, rate := guessEncoding(filename)
	payload := map[string]any{
		"requestFileId": requestFileID,
		"options": map[string]any{
			"audioEncoding":  enc,
			"sampleRate":     rate,
			"language":       "ru-RU",
			"model":          "general",
			"channelsCount":  1,
			"hypothesesCount": 1,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(raw))
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
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, truncate(body, 500))
	}
	var wrap struct {
		Result struct {
			ID string `json:"id"`
		} `json:"result"`
		ID string `json:"id"`
	}
	_ = json.Unmarshal(body, &wrap)
	id := wrap.Result.ID
	if id == "" {
		id = wrap.ID
	}
	if id == "" {
		return "", fmt.Errorf("no task id in response: %s", truncate(body, 300))
	}
	return id, nil
}

func (c *Client) waitTask(ctx context.Context, taskID string) (string, error) {
	u := strings.TrimSuffix(c.cfg.SaluteSpeechRESTURL, "/") + "/task:get"
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	deadline := time.After(15 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-deadline:
			return "", fmt.Errorf("task timeout")
		case <-ticker.C:
			payload, _ := json.Marshal(map[string]string{"taskId": taskID})
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
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
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return "", fmt.Errorf("task get status %d: %s", resp.StatusCode, truncate(body, 400))
			}
			var wrap struct {
				Result taskPayload `json:"result"`
				taskPayload
			}
			_ = json.Unmarshal(body, &wrap)
			tp := wrap.Result
			if tp.ID == "" {
				tp = wrap.taskPayload
			}
			switch strings.ToUpper(tp.Status) {
			case "DONE", "STATUS_DONE":
				if tp.ResponseFileID != "" {
					return tp.ResponseFileID, nil
				}
				if tp.Result.ResponseFileID != "" {
					return tp.Result.ResponseFileID, nil
				}
				return "", fmt.Errorf("done but no response_file_id")
			case "ERROR", "STATUS_ERROR":
				return "", fmt.Errorf("task error: %s", tp.Error)
			case "CANCELED", "STATUS_CANCELED":
				return "", fmt.Errorf("task canceled")
			}
		}
	}
}

type taskPayload struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	Error            string `json:"error"`
	ResponseFileID   string `json:"responseFileId"`
	Result           struct {
		Error            string `json:"error"`
		ResponseFileID   string `json:"responseFileId"`
	} `json:"result"`
}

func (c *Client) download(ctx context.Context, responseFileID string) ([]byte, error) {
	u := strings.TrimSuffix(c.cfg.SaluteSpeechRESTURL, "/") + "/data:download"
	payload, _ := json.Marshal(map[string]string{"responseFileId": responseFileID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("RqUID", uuid.NewString())
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(body, 400))
	}
	return body, nil
}

func guessEncoding(filename string) (enc string, sampleRate int) {
	low := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(low, ".opus"), strings.HasSuffix(low, ".ogg"):
		return "OPUS", 48000
	case strings.HasSuffix(low, ".wav"):
		return "PCM_S16LE", 16000
	case strings.HasSuffix(low, ".mp3"):
		return "MP3", 44100
	default:
		return "MP3", 44100
	}
}

func parseTranscriptJSON(raw []byte) (string, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err == nil {
		if v, ok := root["text"]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil && s != "" {
				return s, nil
			}
		}
		if v, ok := root["result"]; ok {
			if txt, err := parseTranscriptJSON(v); err == nil && txt != "" {
				return txt, nil
			}
		}
		if v, ok := root["hypotheses"]; ok {
			var hyps []struct {
				Text            string `json:"text"`
				NormalizedText  string `json:"normalizedText"`
			}
			if json.Unmarshal(v, &hyps) == nil && len(hyps) > 0 {
				t := hyps[0].NormalizedText
				if t == "" {
					t = hyps[0].Text
				}
				if t != "" {
					return t, nil
				}
			}
		}
	}
	var hyps []struct {
		Text           string `json:"text"`
		NormalizedText string `json:"normalizedText"`
	}
	if err := json.Unmarshal(raw, &hyps); err == nil && len(hyps) > 0 {
		t := hyps[0].NormalizedText
		if t == "" {
			t = hyps[0].Text
		}
		if t != "" {
			return t, nil
		}
	}
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return "", fmt.Errorf("could not parse transcript from response")
	}
	return s, nil
}

func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
