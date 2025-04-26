package types

// Config holds the configuration for the API helper
type Config struct {
	APIKey            string
	SkipCategories    []string
	ChannelWhitelist  []ChannelInfo
	SkipCountTracking bool
	Devices           []string
	MuteAds           bool
	SkipAds           bool
	AutoPlay          bool
	YouTube           YouTubeConfig
	SponsorBlock      SponsorBlockConfig
}

// YouTubeConfig holds YouTube-specific configuration
type YouTubeConfig struct {
	APIKey string
}

// SponsorBlockConfig holds SponsorBlock-specific configuration
type SponsorBlockConfig struct {
	Categories        []string
	SkipCountTracking bool
}

// ChannelInfo represents a channel in the whitelist
type ChannelInfo struct {
	ID string `json:"id"`
}
