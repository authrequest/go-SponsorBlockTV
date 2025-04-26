package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/cache"
	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/constants"
	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/types"
)

// Config holds the configuration for the API helper
type Config = types.Config

// ChannelInfo represents a channel in the whitelist
type ChannelInfo = types.ChannelInfo

// SponsorSegment represents a sponsor segment
type SponsorSegment struct {
	Start float64  `json:"start"`
	End   float64  `json:"end"`
	UUIDs []string `json:"uuids"`
}

// VideoInfo represents information about a YouTube video
type VideoInfo struct {
	ID          string
	Title       string
	ChannelID   string
	ChannelName string
}

// ApiHelper handles all API calls and their caching
type ApiHelper struct {
	config         *Config
	httpClient     *http.Client
	numDevices     int
	cache          *cache.Decorator
	whitelistCache *cache.Decorator
	channelCache   *cache.Decorator
	searchCache    *cache.Decorator
}

// NewApiHelper creates a new ApiHelper instance
func NewApiHelper(config *Config) *ApiHelper {
	helper := &ApiHelper{
		config:     config,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		numDevices: len(config.Devices),
	}

	// Initialize caches
	helper.cache = cache.NewDecorator(5*time.Minute, 10, 0) // 5 minutes TTL, max 10 items
	helper.whitelistCache = cache.NewDecorator(1*time.Hour, 100, 0)
	helper.channelCache = cache.NewDecorator(1*time.Hour, 10, 0)
	helper.searchCache = cache.NewDecorator(1*time.Hour, 10, 0)

	return helper
}

// IsWhitelisted checks if a video's channel is in the whitelist
func (a *ApiHelper) IsWhitelisted(vidID string) (bool, error) {
	if a.config.APIKey == "" || len(a.config.ChannelWhitelist) == 0 {
		return false, nil
	}

	channelID, err := a.getChannelID(vidID)
	if err != nil {
		return false, err
	}

	for _, channel := range a.config.ChannelWhitelist {
		if channel.ID == channelID {
			return true, nil
		}
	}

	return false, nil
}

// GetSegments retrieves sponsor segments for a video
func (a *ApiHelper) GetSegments(vidID string) ([]SponsorSegment, bool, error) {
	// Check if video is whitelisted
	if whitelisted, err := a.IsWhitelisted(vidID); err != nil {
		return nil, false, err
	} else if whitelisted {
		return []SponsorSegment{}, true, nil
	}

	// Hash video ID
	hash := sha256.Sum256([]byte(vidID))
	vidIDHashed := hex.EncodeToString(hash[:])[:4]

	// Build request
	url := fmt.Sprintf("%s/skipSegments/%s", constants.SponsorBlockAPI, vidIDHashed)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, false, err
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("category", strings.Join(a.config.SkipCategories, ","))
	q.Add("actionType", constants.SponsorBlockActionType)
	q.Add("service", constants.SponsorBlockService)
	req.URL.RawQuery = q.Encode()

	// Add headers
	req.Header.Add("Accept", "application/json")

	// Make request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("error getting segments for video %s (hashed as %s): %s", vidID, vidIDHashed, string(body))
	}

	// Parse response
	var segments []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&segments); err != nil {
		return nil, false, err
	}

	// Find matching video
	var response map[string]interface{}
	for _, segment := range segments {
		if segment["videoID"] == vidID {
			response = segment
			break
		}
	}

	return a.processSegments(response)
}

// processSegments processes the raw segments into a more usable format
func (a *ApiHelper) processSegments(response map[string]interface{}) ([]SponsorSegment, bool, error) {
	if response == nil {
		return []SponsorSegment{}, true, nil
	}

	segments := []SponsorSegment{}
	ignoreTTL := true

	rawSegments, ok := response["segments"].([]interface{})
	if !ok {
		return segments, ignoreTTL, nil
	}

	// Sort by end time
	sort.Slice(rawSegments, func(i, j int) bool {
		seg1 := rawSegments[i].(map[string]interface{})["segment"].([]interface{})
		seg2 := rawSegments[j].(map[string]interface{})["segment"].([]interface{})
		return seg1[1].(float64) < seg2[1].(float64)
	})

	// Process segments
	for _, rawSeg := range rawSegments {
		seg := rawSeg.(map[string]interface{})
		segment := seg["segment"].([]interface{})
		uuid := seg["UUID"].(string)
		locked := seg["locked"].(float64) == 1

		ignoreTTL = ignoreTTL && locked

		// Check if this segment can be merged with the previous one
		if len(segments) > 0 {
			lastSeg := &segments[len(segments)-1]
			if segment[0].(float64)-lastSeg.End < 1.0 {
				// Merge segments
				lastSeg.End = segment[1].(float64)
				lastSeg.UUIDs = append(lastSeg.UUIDs, uuid)
				continue
			}
		}

		// Add new segment
		segments = append(segments, SponsorSegment{
			Start: segment[0].(float64),
			End:   segment[1].(float64),
			UUIDs: []string{uuid},
		})
	}

	return segments, ignoreTTL, nil
}

// MarkViewedSegments marks segments as viewed in the SponsorBlock API
func (a *ApiHelper) MarkViewedSegments(uuids []string) error {
	if !a.config.SkipCountTracking {
		return nil
	}

	for _, uuid := range uuids {
		url := fmt.Sprintf("%s/viewedVideoSponsorTime/", constants.SponsorBlockAPI)
		req, err := http.NewRequest("POST", url, nil)
		if err != nil {
			return err
		}

		q := req.URL.Query()
		q.Add("UUID", uuid)
		req.URL.RawQuery = q.Encode()

		resp, err := a.httpClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
	}

	return nil
}

// getChannelID retrieves the channel ID for a video
func (a *ApiHelper) getChannelID(vidID string) (string, error) {
	url := fmt.Sprintf("%s/videos", constants.YouTubeAPI)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Add("id", vidID)
	q.Add("key", a.config.APIKey)
	q.Add("part", "snippet")
	req.URL.RawQuery = q.Encode()

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		Items []struct {
			Snippet struct {
				ChannelID string `json:"channelId"`
			} `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if len(data.Items) == 0 {
		return "", fmt.Errorf("no video found with ID %s", vidID)
	}

	return data.Items[0].Snippet.ChannelID, nil
}

// GetVideoID searches for a video by title and artist
func (a *ApiHelper) GetVideoID(ctx context.Context, title, artist string) (*VideoInfo, error) {
	params := url.Values{}
	params.Add("q", title+" "+artist)
	params.Add("key", a.config.APIKey)
	params.Add("part", "snippet")

	req, err := http.NewRequestWithContext(ctx, "GET", constants.YouTubeAPI+"search", nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.Encode()

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Items []struct {
			ID struct {
				VideoID string `json:"videoId"`
			} `json:"id"`
			Snippet struct {
				Title       string `json:"title"`
				ChannelID   string `json:"channelId"`
				ChannelName string `json:"channelTitle"`
			} `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	for _, item := range data.Items {
		if item.ID.VideoID == "" {
			continue
		}
		return &VideoInfo{
			ID:          item.ID.VideoID,
			Title:       item.Snippet.Title,
			ChannelID:   item.Snippet.ChannelID,
			ChannelName: item.Snippet.ChannelName,
		}, nil
	}

	return nil, fmt.Errorf("video not found")
}

// SearchChannels searches for YouTube channels
func (a *ApiHelper) SearchChannels(ctx context.Context, query string) ([]struct {
	ID              string
	Title           string
	SubscriberCount string
}, error) {
	params := url.Values{}
	params.Add("q", query)
	params.Add("key", a.config.APIKey)
	params.Add("part", "snippet")
	params.Add("type", "channel")
	params.Add("maxResults", "5")

	req, err := http.NewRequestWithContext(ctx, "GET", constants.YouTubeAPI+"search", nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.Encode()

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Items []struct {
			Snippet struct {
				ChannelID   string `json:"channelId"`
				ChannelName string `json:"channelTitle"`
			} `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	channels := make([]struct {
		ID              string
		Title           string
		SubscriberCount string
	}, 0, len(data.Items))
	for _, item := range data.Items {
		// Get channel statistics
		stats, err := a.getChannelStats(ctx, item.Snippet.ChannelID)
		if err != nil {
			continue
		}

		channels = append(channels, struct {
			ID              string
			Title           string
			SubscriberCount string
		}{
			ID:              item.Snippet.ChannelID,
			Title:           item.Snippet.ChannelName,
			SubscriberCount: stats,
		})
	}

	return channels, nil
}

// getChannelStats gets the subscriber count for a channel
func (a *ApiHelper) getChannelStats(ctx context.Context, channelID string) (string, error) {
	params := url.Values{}
	params.Add("id", channelID)
	params.Add("key", a.config.APIKey)
	params.Add("part", "statistics")

	req, err := http.NewRequestWithContext(ctx, "GET", constants.YouTubeAPI+"channels", nil)
	if err != nil {
		return "", err
	}
	req.URL.RawQuery = params.Encode()

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		Items []struct {
			Statistics struct {
				HiddenSubscriberCount bool   `json:"hiddenSubscriberCount"`
				SubscriberCount       string `json:"subscriberCount"`
			} `json:"statistics"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if len(data.Items) == 0 {
		return "", fmt.Errorf("channel not found")
	}

	stats := data.Items[0].Statistics
	if stats.HiddenSubscriberCount {
		return "Hidden", nil
	}

	return stats.SubscriberCount, nil
}
