package ytlounge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/config"
)

// Client represents a YouTube Lounge client
type Client struct {
	cfg      *config.Config
	http     *http.Client
	baseURL  string
	ScreenID string
}

// NewClient creates a new YouTube Lounge client
func NewClient(cfg *config.Config) (*Client, error) {
	client := &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://www.youtube.com/api/lounge",
	}

	// Get screen ID
	screenID, err := client.GetScreenID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get screen ID: %w", err)
	}
	client.ScreenID = screenID

	return client, nil
}

// GetScreenID retrieves the screen ID for the YouTube Lounge
func (c *Client) GetScreenID(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/bc/bind", c.baseURL), nil)
	if err != nil {
		return "", err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ScreenID string `json:"screenId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ScreenID, nil
}

// SendCommand sends a command to the YouTube Lounge
func (c *Client) SendCommand(ctx context.Context, screenID string, command interface{}) error {
	data, err := json.Marshal(command)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/bc/bind?screen_id=%s", c.baseURL, screenID),
		bytes.NewReader(data))
	if err != nil {
		return err
	}

	_, err = c.http.Do(req)
	return err
}
