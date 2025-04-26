package convert

import (
	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/api"
	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/config"
)

// ToApiConfig converts a config.Config to an api.Config
func ToApiConfig(cfg *config.Config) *api.Config {
	apiConfig := &api.Config{
		APIKey:            cfg.APIKey,
		SkipCategories:    cfg.SkipCategories,
		ChannelWhitelist:  cfg.ChannelWhitelist,
		SkipCountTracking: cfg.SkipCountTracking,
		Devices:           make([]string, len(cfg.Devices)),
		MuteAds:           cfg.MuteAds,
		SkipAds:           cfg.SkipAds,
		AutoPlay:          cfg.AutoPlay,
	}

	for i, device := range cfg.Devices {
		apiConfig.Devices[i] = device.ScreenID
	}

	return apiConfig
}
