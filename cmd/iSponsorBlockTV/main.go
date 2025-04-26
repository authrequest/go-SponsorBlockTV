package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/api"
	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/config"
	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/ytlounge"
	"github.com/sirupsen/logrus"
)

// DeviceListener handles communication with a YouTube device
type DeviceListener struct {
	apiHelper        *api.APIHelper
	config           *config.Config
	device           *Device
	debug            bool
	httpClient       *http.Client
	logger           *logrus.Logger
	loungeController *ytlounge.YtLoungeApi
	task             *Task
	cancelled        bool
}

// Device represents a YouTube device configuration
type Device struct {
	Name     string
	Offset   float64
	ScreenID string
}

// Task represents an asynchronous task
type Task struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewDeviceListener creates a new DeviceListener instance
func NewDeviceListener(apiHelper *api.APIHelper, config *config.Config, device *Device, debug bool, httpClient *http.Client) *DeviceListener {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	client, err := ytlounge.NewClient(config)
	if err != nil {
		logger.Fatalf("Failed to create client: %v", err)
	}
	loungeController := ytlounge.NewYtLoungeApi(client, apiHelper, logger)

	return &DeviceListener{
		apiHelper:        apiHelper,
		config:           config,
		device:           device,
		debug:            debug,
		httpClient:       httpClient,
		logger:           logger,
		loungeController: loungeController,
	}
}

// Loop handles the main device connection and monitoring loop
func (d *DeviceListener) Loop(ctx context.Context) {
	for !d.cancelled {
		// Subscribe to device events
		if err := d.loungeController.SubscribeMonitored(ctx, d.handleEvent); err != nil && d.debug {
			d.logger.Errorf("Error subscribing to device: %v", err)
		}

		// Wait a bit before retrying
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
			continue
		}
	}
}

// handleEvent processes events from the YouTube Lounge
func (d *DeviceListener) handleEvent(eventType string, args []interface{}) {
	switch eventType {
	case "onStateChange":
		if data, ok := args[0].(map[string]interface{}); ok {
			state := &ytlounge.PlaybackState{
				VideoID: data["videoId"].(string),
				State:   ytlounge.StatePlaying,
			}
			if currentTime, ok := data["currentTime"].(float64); ok {
				state.CurrentTime = currentTime
			}
			d.HandlePlaybackStateChange(state)
		}
	}
}

// HandlePlaybackStateChange processes playback state changes
func (d *DeviceListener) HandlePlaybackStateChange(state *ytlounge.PlaybackState) {
	if d.task != nil {
		d.task.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.task = &Task{ctx: ctx, cancel: cancel}

	go d.processPlaybackState(state, time.Now())
}

// processPlaybackState processes the playback state
func (d *DeviceListener) processPlaybackState(state *ytlounge.PlaybackState, startTime time.Time) {
	segments := []api.Segment{}
	if state.VideoID != "" {
		var err error
		segments, _, err = d.apiHelper.GetSegments(context.Background(), state.VideoID)
		if err != nil && d.debug {
			d.logger.Errorf("Error getting segments: %v", err)
		}
	}

	if state.State == ytlounge.StatePlaying {
		d.logger.Infof("Playing video %s with %d segments", state.VideoID, len(segments))
		if len(segments) > 0 {
			d.timeToSegment(segments, state.CurrentTime, startTime)
		}
	}
}

// timeToSegment finds the next segment to skip to
func (d *DeviceListener) timeToSegment(segments []api.Segment, position float64, startTime time.Time) {
	var nextSegment *api.Segment
	var startNextSegment float64

	for _, segment := range segments {
		isWithinStartRange := position < 1 && segment.End > 1 && segment.Start <= position && position < segment.End
		isBeyondCurrentPosition := segment.Start > position

		if isWithinStartRange || isBeyondCurrentPosition {
			nextSegment = &segment
			if isWithinStartRange {
				startNextSegment = position
			} else {
				startNextSegment = segment.Start
			}
			break
		}
	}

	if nextSegment != nil {
		timeToNext := (startNextSegment - position - time.Since(startTime).Seconds()) - d.device.Offset
		d.skip(timeToNext, nextSegment.End, nextSegment.UUIDs)
	}
}

// skip handles segment skipping
func (d *DeviceListener) skip(timeTo float64, position float64, uuids []string) {
	time.Sleep(time.Duration(timeTo * float64(time.Second)))

	d.logger.Infof("Skipping segment: seeking to %f", position)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := d.apiHelper.MarkViewedSegments(context.Background(), uuids); err != nil && d.debug {
			d.logger.Errorf("Error marking segments as viewed: %v", err)
		}
	}()

	wg.Wait()
}

// Cancel stops the device listener
func (d *DeviceListener) Cancel() {
	d.cancelled = true
	if d.task != nil {
		d.task.cancel()
	}
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create API helper
	apiHelper := api.NewAPIHelper(cfg, &http.Client{
		Timeout: 10 * time.Second,
	})

	// Create device listeners
	listeners := make([]*DeviceListener, len(cfg.Devices))
	for i, deviceConfig := range cfg.Devices {
		device := &Device{
			Name:     deviceConfig.Name,
			Offset:   deviceConfig.Offset,
			ScreenID: deviceConfig.ScreenID,
		}
		listeners[i] = NewDeviceListener(apiHelper, cfg, device, cfg.Debug, &http.Client{
			Timeout: 10 * time.Second,
		})
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Start device listeners
	var wg sync.WaitGroup
	for _, device := range listeners {
		wg.Add(1)
		go func(d *DeviceListener) {
			defer wg.Done()
			d.Loop(ctx)
		}(device)
	}

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for signal
	<-sigChan
	log.Println("Cancelling tasks and exiting...")

	// Cancel context and wait for tasks to complete
	cancel()
	for _, device := range listeners {
		device.Cancel()
	}
	wg.Wait()

	// Close HTTP client
	for _, device := range listeners {
		device.httpClient.CloseIdleConnections()
	}
}
