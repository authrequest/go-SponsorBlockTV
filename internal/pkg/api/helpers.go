package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/config"
	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/constants"
	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/dial"
)

// Segment represents a sponsor segment
type Segment struct {
	Start float64  `json:"start"`
	End   float64  `json:"end"`
	UUIDs []string `json:"uuids"`
}

// APIHelper handles all API calls and caching
type APIHelper struct {
	cfg              *config.Config
	httpClient       *http.Client
	cache            *Cache
	channelWhitelist []string
}

// NewAPIHelper creates a new API helper
func NewAPIHelper(cfg *config.Config, httpClient *http.Client) *APIHelper {
	return &APIHelper{
		cfg:        cfg,
		httpClient: httpClient,
		cache:      NewCache(100, 5*time.Minute),
	}
}

// GetSegments retrieves sponsor segments for a video
func (a *APIHelper) GetSegments(ctx context.Context, videoID string) ([]Segment, bool, error) {
	// Check if channel is whitelisted
	if a.cfg.YouTube.APIKey != "" && len(a.channelWhitelist) > 0 {
		channelID, err := a.getChannelID(ctx, videoID)
		if err != nil {
			return nil, false, err
		}

		for _, whitelistedID := range a.channelWhitelist {
			if whitelistedID == channelID {
				return []Segment{}, true, nil
			}
		}
	}

	// Hash video ID
	hash := sha256.Sum256([]byte(videoID))
	videoIDHashed := hex.EncodeToString(hash[:])[:4]

	// Build request
	params := url.Values{}
	params.Add("category", strings.Join(a.cfg.SponsorBlock.Categories, ","))
	params.Add("actionType", constants.SponsorBlockActionType)
	params.Add("service", constants.SponsorBlockService)

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%sskipSegments/%s", constants.SponsorBlockAPI, videoIDHashed), nil)
	if err != nil {
		return nil, false, err
	}

	req.URL.RawQuery = params.Encode()
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", constants.UserAgent)

	// Send request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body := map[string]interface{}{}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return nil, false, fmt.Errorf("failed to get segments: %d - %v", resp.StatusCode, err)
		}
		return nil, false, fmt.Errorf("failed to get segments: %d - %v", resp.StatusCode, body)
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, false, err
	}

	// Find matching video
	var segmentsData map[string]interface{}
	for _, item := range response {
		if item["videoID"] == videoID {
			segmentsData = item
			break
		}
	}

	if segmentsData == nil {
		return []Segment{}, true, nil
	}

	return a.processSegments(segmentsData)
}

// processSegments processes the segments data
func (a *APIHelper) processSegments(data map[string]interface{}) ([]Segment, bool, error) {
	segments := make([]Segment, 0)
	ignoreTTL := true

	rawSegments, ok := data["segments"].([]interface{})
	if !ok {
		return segments, ignoreTTL, nil
	}

	// Convert to typed segments
	typedSegments := make([]struct {
		Segment []float64 `json:"segment"`
		UUID    string    `json:"UUID"`
		Locked  int       `json:"locked"`
	}, len(rawSegments))

	for i, s := range rawSegments {
		segmentData, _ := json.Marshal(s)
		json.Unmarshal(segmentData, &typedSegments[i])
	}

	// Sort by end time
	sort.Slice(typedSegments, func(i, j int) bool {
		return typedSegments[i].Segment[1] < typedSegments[j].Segment[1]
	})

	// Extend ends of overlapping segments
	for i := range typedSegments {
		for j := range typedSegments {
			if typedSegments[j].Segment[0] <= typedSegments[i].Segment[1] &&
				typedSegments[i].Segment[1] <= typedSegments[j].Segment[1] {
				typedSegments[i].Segment[1] = typedSegments[j].Segment[1]
			}
		}
	}

	// Sort by start time
	sort.Slice(typedSegments, func(i, j int) bool {
		return typedSegments[i].Segment[0] < typedSegments[j].Segment[0]
	})

	// Extend starts of overlapping segments
	for i := len(typedSegments) - 1; i >= 0; i-- {
		for j := len(typedSegments) - 1; j >= 0; j-- {
			if typedSegments[j].Segment[0] <= typedSegments[i].Segment[0] &&
				typedSegments[i].Segment[0] <= typedSegments[j].Segment[1] {
				typedSegments[i].Segment[0] = typedSegments[j].Segment[0]
			}
		}
	}

	// Combine close segments
	for _, s := range typedSegments {
		ignoreTTL = ignoreTTL && s.Locked == 1

		segment := Segment{
			Start: s.Segment[0],
			End:   s.Segment[1],
			UUIDs: []string{s.UUID},
		}

		if len(segments) > 0 {
			last := &segments[len(segments)-1]
			if segment.Start-last.End < 1 {
				// Less than 1 second apart, combine them
				segment.Start = last.Start
				segment.UUIDs = append(segment.UUIDs, last.UUIDs...)
				segments = segments[:len(segments)-1]
			}
		}

		segments = append(segments, segment)
	}

	return segments, ignoreTTL, nil
}

// MarkViewedSegments marks segments as viewed in SponsorBlock
func (a *APIHelper) MarkViewedSegments(ctx context.Context, uuids []string) error {
	if !a.cfg.SponsorBlock.SkipCountTracking {
		return nil
	}

	for _, uuid := range uuids {
		params := url.Values{}
		params.Add("UUID", uuid)

		req, err := http.NewRequestWithContext(ctx, "POST",
			constants.SponsorBlockAPI+"viewedVideoSponsorTime/", nil)
		if err != nil {
			return err
		}

		req.URL.RawQuery = params.Encode()
		req.Header.Set("User-Agent", constants.UserAgent)

		resp, err := a.httpClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
	}

	return nil
}

// getChannelID retrieves the channel ID for a video
func (a *APIHelper) getChannelID(ctx context.Context, videoID string) (string, error) {
	params := url.Values{}
	params.Add("id", videoID)
	params.Add("key", a.cfg.YouTube.APIKey)
	params.Add("part", "snippet")

	req, err := http.NewRequestWithContext(ctx, "GET",
		constants.YouTubeAPI+"videos", nil)
	if err != nil {
		return "", err
	}

	req.URL.RawQuery = params.Encode()
	req.Header.Set("User-Agent", constants.UserAgent)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var response struct {
		Items []struct {
			Snippet struct {
				ChannelID string `json:"channelId"`
			} `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Items) == 0 {
		return "", fmt.Errorf("no video found")
	}

	return response.Items[0].Snippet.ChannelID, nil
}

// DiscoverYouTubeDevices discovers YouTube devices using DIAL
func (a *APIHelper) DiscoverYouTubeDevices(ctx context.Context) ([]dial.Device, error) {
	return dial.Discover(ctx, a.httpClient)
}
