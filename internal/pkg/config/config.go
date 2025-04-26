package config

import (
	"encoding/json"
	"os"

	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/types"
)

// Config represents the application configuration
type Config struct {
	APIKey            string                   `json:"apikey"`
	SkipCategories    []string                 `json:"skip_categories"`
	ChannelWhitelist  []types.ChannelInfo      `json:"channel_whitelist"`
	SkipCountTracking bool                     `json:"skip_count_tracking"`
	Devices           []DeviceConfig           `json:"devices"`
	Debug             bool                     `json:"debug"`
	MuteAds           bool                     `json:"mute_ads"`
	SkipAds           bool                     `json:"skip_ads"`
	AutoPlay          bool                     `json:"auto_play"`
	YouTube           types.YouTubeConfig      `json:"youtube"`
	SponsorBlock      types.SponsorBlockConfig `json:"sponsorblock"`
	JoinName          string                   `json:"join_name"`
}

// DeviceConfig represents a device configuration
type DeviceConfig struct {
	Name     string  `json:"name"`
	Offset   float64 `json:"offset"`
	ScreenID string  `json:"screen_id"`
}

// LoadConfig loads the configuration from config.json
func LoadConfig() (*Config, error) {
	// Read config file
	data, err := os.ReadFile("config.json")
	if err != nil {
		return nil, err
	}

	// Parse config
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
