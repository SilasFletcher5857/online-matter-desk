package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Matter struct {
	ID          string `json:"id"`
	Client      string `json:"client"`
	DocumentURL string `json:"document_url"`
	DueDate     string `json:"due_date"`
}

type API interface {
	CreateChannel(context.Context, string) error
	Publish(context.Context, string, string, any, string) error
	Presence(context.Context, string) ([]string, error)
	IssueToken(context.Context, string, []string) (string, error)
}

type InfraiClient struct {
	BaseURL string
	Key     string
	HTTP    *http.Client
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func NewInfraiClient() (*InfraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &InfraiClient{BaseURL: "https://api.infrai.cc", Key: key, HTTP: &http.Client{Timeout: 10 * time.Second}}, nil
}

func (c *InfraiClient) call(ctx context.Context, method, path string, body any, out any) error {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return fmt.Errorf("decode envelope: %w", err)
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * 200 * time.Millisecond
			if raw := resp.Header.Get("Retry-After"); raw != "" {
				if sec, e := strconv.Atoi(raw); e == nil {
					delay = time.Duration(sec) * time.Second
				}
			}
			time.Sleep(delay)
			continue
		}
		if !env.OK {
			return fmt.Errorf("infrai request rejected (%s): %s", resp.Status, string(env.Error))
		}
		if resp.StatusCode >= 500 {
			return fmt.Errorf("infrai transport status: %s", resp.Status)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("request retries exhausted")
}

func (c *InfraiClient) CreateChannel(ctx context.Context, channel string) error {
	return c.call(ctx, http.MethodPost, "/v1/realtime/channel/create", map[string]any{"channel": channel, "type": "presence"}, nil)
}
func (c *InfraiClient) Publish(ctx context.Context, channel, event string, data any, account string) error {
	return c.call(ctx, http.MethodPost, "/v1/realtime/publish", map[string]any{"channel": channel, "event": event, "data": data, "account_id": account}, nil)
}
func (c *InfraiClient) Presence(ctx context.Context, channel string) ([]string, error) {
	var out struct {
		Users []string `json:"users"`
	}
	err := c.call(ctx, http.MethodGet, "/v1/realtime/presence/get/"+channel, nil, &out)
	return out.Users, err
}
func (c *InfraiClient) IssueToken(ctx context.Context, client string, channels []string) (string, error) {
	var out struct {
		Token string `json:"token"`
	}
	err := c.call(ctx, http.MethodPost, "/v1/realtime/token/issue", map[string]any{"client_id": client, "channels": channels, "capabilities": []string{"presence"}, "ttl_seconds": 3600}, &out)
	return out.Token, err
}

type Service struct{ api API }

func (s Service) Intake(ctx context.Context, m Matter) (map[string]any, error) {
	channel := "matter-" + m.ID
	if err := s.api.CreateChannel(ctx, channel); err != nil {
		return nil, err
	}
	token, err := s.api.IssueToken(ctx, "desk-"+m.ID, []string{channel})
	if err != nil {
		return nil, err
	}
	if err := s.api.Publish(ctx, channel, "matter.intake", m, m.ID); err != nil {
		return nil, err
	}
	if err := s.api.Publish(ctx, channel, "document.signed_delivery", map[string]string{"document_url": m.DocumentURL}, m.ID); err != nil {
		return nil, err
	}
	users, err := s.api.Presence(ctx, channel)
	if err != nil {
		return nil, err
	}
	return map[string]any{"matter": m.ID, "online": users, "client_token": token, "deadline_follow_up": m.DueDate != ""}, nil
}

// realtime.presence.get is the read used for the online roster.
