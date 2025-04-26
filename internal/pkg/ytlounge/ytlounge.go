package ytlounge

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/api"
	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/config"
	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/constants"
	"github.com/sirupsen/logrus"
)

// PlaybackState represents the current playback state
type PlaybackState struct {
	VideoID     string
	CurrentTime float64
	State       PlaybackStateType
}

// PlaybackStateType represents the type of playback state
type PlaybackStateType int

const (
	StateUnknown PlaybackStateType = iota
	StatePlaying
	StatePaused
	StateBuffering
)

// VolumeState represents the current volume state
type VolumeState struct {
	Volume int
	Muted  bool
}

// YtLoungeApi represents a YouTube Lounge API client
type YtLoungeApi struct {
	client             *Client
	config             *config.Config
	apiHelper          *api.APIHelper
	logger             *logrus.Logger
	volumeState        map[string]interface{}
	playbackSpeed      float64
	subscribeTask      context.CancelFunc
	watchdogTask       context.CancelFunc
	callback           func(eventType string, args []interface{})
	shortsDisconnected bool
	autoPlay           bool
	muteAds            bool
	skipAds            bool
	commandMutex       sync.Mutex
}

// NewYtLoungeApi creates a new YtLoungeApi instance
func NewYtLoungeApi(client *Client, apiHelper *api.APIHelper, logger *logrus.Logger) *YtLoungeApi {
	return &YtLoungeApi{
		client:        client,
		apiHelper:     apiHelper,
		logger:        logger,
		volumeState:   make(map[string]interface{}),
		playbackSpeed: 1.0,
	}
}

// SetMuteAds sets whether to mute ads
func (y *YtLoungeApi) SetMuteAds(mute bool) {
	y.muteAds = mute
}

// SetSkipAds sets whether to skip ads
func (y *YtLoungeApi) SetSkipAds(skip bool) {
	y.skipAds = skip
}

// SetAutoPlay sets whether to enable autoplay
func (y *YtLoungeApi) SetAutoPlay(autoPlay bool) {
	y.autoPlay = autoPlay
}

// SubscribeMonitored starts a monitored subscription to the lounge
func (y *YtLoungeApi) SubscribeMonitored(ctx context.Context, callback func(eventType string, args []interface{})) error {
	y.callback = callback

	// Cancel existing tasks if any
	if y.watchdogTask != nil {
		y.watchdogTask()
	}
	if y.subscribeTask != nil {
		y.subscribeTask()
	}

	// Create new context for subscription
	subCtx, subCancel := context.WithCancel(ctx)
	y.subscribeTask = subCancel

	// Start subscription
	go y.subscribe(subCtx)

	// Start watchdog
	watchCtx, watchCancel := context.WithCancel(ctx)
	y.watchdogTask = watchCancel
	go y.watchdog(watchCtx)

	return nil
}

func (y *YtLoungeApi) watchdog(ctx context.Context) {
	ticker := time.NewTicker(35 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if y.subscribeTask != nil {
				y.subscribeTask()
				y.subscribeTask = nil
			}
		}
	}
}

func (y *YtLoungeApi) subscribe(ctx context.Context) {
	// Implementation of subscription logic here
	// This would involve setting up a websocket or long-polling connection
	// to receive events from the YouTube Lounge API
}

// ProcessEvent processes events from the YouTube Lounge API
func (y *YtLoungeApi) ProcessEvent(eventType string, args []interface{}) {
	y.logger.Debugf("process_event(%s, %v)", eventType, args)

	// Restart watchdog
	if y.watchdogTask != nil {
		y.watchdogTask()
	}

	switch eventType {
	case "onStateChange":
		if data, ok := args[0].(map[string]interface{}); ok {
			if y.muteAds && data["state"] == "1" {
				go y.Mute(false, true)
			}
		}

	case "nowPlaying":
		if data, ok := args[0].(map[string]interface{}); ok {
			if y.muteAds && data["state"] == "1" {
				y.logger.Info("Ad has ended, unmuting")
				go y.Mute(false, true)
			}
		}

	case "onAdStateChange":
		if data, ok := args[0].(map[string]interface{}); ok {
			if data["adState"] == "0" {
				y.logger.Info("Ad has ended, unmuting")
				go y.Mute(false, true)
			} else if y.skipAds && data["isSkipEnabled"] == "true" {
				y.logger.Info("Ad can be skipped, skipping")
				go y.SkipAd()
				go y.Mute(false, true)
			} else if y.muteAds {
				y.logger.Info("Ad has started, muting")
				go y.Mute(true, true)
			}
		}

	case "onVolumeChanged":
		if len(args) > 0 {
			y.volumeState = args[0].(map[string]interface{})
		}

	case "autoplayUpNext":
		if len(args) > 0 {
			if data, ok := args[0].(map[string]interface{}); ok {
				if videoID, ok := data["videoId"].(string); ok && videoID != "" {
					y.logger.Infof("Getting segments for next video: %s", videoID)
					go y.apiHelper.GetSegments(context.Background(), videoID)
				}
			}
		}

	case "adPlaying":
		if data, ok := args[0].(map[string]interface{}); ok {
			if videoID, ok := data["contentVideoId"].(string); ok && videoID != "" {
				y.logger.Infof("Getting segments for next video: %s", videoID)
				go y.apiHelper.GetSegments(context.Background(), videoID)
			}
			if y.skipAds && data["isSkipEnabled"] == "true" {
				y.logger.Info("Ad can be skipped, skipping")
				go y.SkipAd()
				go y.Mute(false, true)
			} else if y.muteAds {
				y.logger.Info("Ad has started, muting")
				go y.Mute(true, true)
			}
		}

	case "loungeStatus":
		if data, ok := args[0].(map[string]interface{}); ok {
			if devices, ok := data["devices"].(string); ok {
				var devicesData []map[string]interface{}
				if err := json.Unmarshal([]byte(devices), &devicesData); err == nil {
					for _, device := range devicesData {
						if device["type"] == "LOUNGE_SCREEN" {
							if deviceInfo, ok := device["deviceInfo"].(string); ok {
								var info map[string]interface{}
								if err := json.Unmarshal([]byte(deviceInfo), &info); err == nil {
									if clientName, ok := info["clientName"].(string); ok {
										for _, blacklisted := range constants.YouTubeClientBlacklist {
											if clientName == blacklisted {
												// Force disconnect
												y.subscribeTask()
												return
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}

	case "onSubtitlesTrackChanged":
		if y.shortsDisconnected {
			if data, ok := args[0].(map[string]interface{}); ok {
				if videoID, ok := data["videoId"].(string); ok {
					y.shortsDisconnected = false
					go y.PlayVideo(videoID)
				}
			}
		}

	case "loungeScreenDisconnected":
		if len(args) > 0 {
			if data, ok := args[0].(map[string]interface{}); ok {
				if data["reason"] == "disconnectedByUserScreenInitiated" {
					y.shortsDisconnected = true
				}
			}
		}

	case "onAutoplayModeChanged":
		go y.SetAutoPlayMode(y.autoPlay)

	case "onPlaybackSpeedChanged":
		if data, ok := args[0].(map[string]interface{}); ok {
			if speed, ok := data["playbackSpeed"].(string); ok {
				if parsedSpeed, err := strconv.ParseFloat(speed, 64); err == nil {
					y.playbackSpeed = parsedSpeed
				}
			}
			go y.GetNowPlaying()
		}
	}

	if y.callback != nil {
		y.callback(eventType, args)
	}
}

// SetVolume sets the volume to a specific value (0-100)
func (y *YtLoungeApi) SetVolume(volume int) error {
	y.commandMutex.Lock()
	defer y.commandMutex.Unlock()

	return y.client.SendCommand(context.Background(), y.client.ScreenID, map[string]interface{}{
		"command": "setVolume",
		"volume":  volume,
	})
}

// Mute mutes or unmutes the device
func (y *YtLoungeApi) Mute(mute bool, override bool) error {
	y.commandMutex.Lock()
	defer y.commandMutex.Unlock()

	muteStr := "false"
	if mute {
		muteStr = "true"
	}

	if override || y.volumeState["muted"] != muteStr {
		y.volumeState["muted"] = muteStr
		volume := 100
		if vol, ok := y.volumeState["volume"].(float64); ok {
			volume = int(vol)
		}

		return y.client.SendCommand(context.Background(), y.client.ScreenID, map[string]interface{}{
			"command": "setVolume",
			"volume":  volume,
			"muted":   muteStr,
		})
	}

	return nil
}

// PlayVideo plays a video by its ID
func (y *YtLoungeApi) PlayVideo(videoID string) error {
	y.commandMutex.Lock()
	defer y.commandMutex.Unlock()

	return y.client.SendCommand(context.Background(), y.client.ScreenID, map[string]interface{}{
		"command": "setPlaylist",
		"videoId": videoID,
	})
}

// GetNowPlaying gets the currently playing video information
func (y *YtLoungeApi) GetNowPlaying() error {
	y.commandMutex.Lock()
	defer y.commandMutex.Unlock()

	return y.client.SendCommand(context.Background(), y.client.ScreenID, map[string]interface{}{
		"command": "getNowPlaying",
	})
}

// SkipAd skips the current advertisement if possible
func (y *YtLoungeApi) SkipAd() error {
	y.commandMutex.Lock()
	defer y.commandMutex.Unlock()

	return y.client.SendCommand(context.Background(), y.client.ScreenID, map[string]interface{}{
		"command": "skipAd",
	})
}

// SetAutoPlayMode sets the autoplay mode
func (y *YtLoungeApi) SetAutoPlayMode(enabled bool) error {
	y.commandMutex.Lock()
	defer y.commandMutex.Unlock()

	return y.client.SendCommand(context.Background(), y.client.ScreenID, map[string]interface{}{
		"command":      "setAutoplayMode",
		"autoplayMode": enabled,
	})
}

// PlaybackSpeed returns the current playback speed
func (y *YtLoungeApi) PlaybackSpeed() float64 {
	return y.playbackSpeed
}

func (y *YtLoungeApi) handleEvent(event string, args []interface{}) {
	y.logger.Debugf("handle_event(%s, %v)", event, args)

	switch event {
	case "nowPlaying":
		if len(args) > 0 {
			if data, ok := args[0].(map[string]interface{}); ok {
				if videoID, ok := data["videoId"].(string); ok && videoID != "" {
					y.logger.Infof("Getting segments for video: %s", videoID)
					go y.apiHelper.GetSegments(context.Background(), videoID)
				}
				if y.muteAds && data["state"] == "1" {
					y.logger.Info("Ad has ended, unmuting")
					go y.Mute(false, true)
				}
			}
		}

	case "onStateChange":
		if len(args) > 0 {
			if data, ok := args[0].(map[string]interface{}); ok {
				if state, ok := data["state"].(string); ok {
					if state == "3" && y.shortsDisconnected {
						y.shortsDisconnected = false
						go y.GetNowPlaying()
					} else if state == "1" && y.muteAds {
						go y.Mute(true, true)
					}
				}
			}
		}

	case "onAdStateChange":
		if len(args) > 0 {
			if data, ok := args[0].(map[string]interface{}); ok {
				if adState, ok := data["adState"].(string); ok {
					if adState == "0" {
						y.logger.Info("Ad has ended, unmuting")
						go y.Mute(false, true)
					} else if y.skipAds {
						if isSkipEnabled, ok := data["isSkipEnabled"].(string); ok && isSkipEnabled == "true" {
							y.logger.Info("Ad can be skipped, skipping")
							go y.SkipAd()
							go y.Mute(false, true)
						}
					} else if y.muteAds {
						y.logger.Info("Ad has started, muting")
						go y.Mute(true, true)
					}
				}
			}
		}

	case "onVolumeChanged":
		if len(args) > 0 {
			if data, ok := args[0].(map[string]interface{}); ok {
				y.volumeState = data
			}
		}

	case "onSubtitlesTrackChanged", "loungeScreenDisconnected":
		y.shortsDisconnected = true

	case "onAutoplayModeChanged":
		if len(args) > 0 {
			if data, ok := args[0].(map[string]interface{}); ok {
				if enabled, ok := data["autoplayMode"].(string); ok {
					y.autoPlay = enabled == "true"
				}
			}
		}

	case "onPlaybackSpeedChanged":
		if len(args) > 0 {
			if data, ok := args[0].(map[string]interface{}); ok {
				if speed, ok := data["playbackRate"].(string); ok {
					if parsedSpeed, err := strconv.ParseFloat(speed, 64); err == nil {
						y.playbackSpeed = parsedSpeed
					}
				}
			}
		}
	}

	if y.callback != nil {
		y.callback(event, args)
	}
}
