package dial

import (
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	multicastAddress = "239.255.255.250"
	port             = 1900
	searchTarget     = "urn:dial-multiscreen-org:service:dial:1"
)

// Device represents a discovered YouTube TV device
type Device struct {
	ScreenID string
	Name     string
	Offset   int
}

// Handler handles SSDP responses
type Handler struct {
	devices []string
}

// NewHandler creates a new SSDP handler
func NewHandler() *Handler {
	return &Handler{
		devices: make([]string, 0),
	}
}

// Clear clears the list of discovered devices
func (h *Handler) Clear() {
	h.devices = h.devices[:0]
}

// HandleResponse handles an SSDP response
func (h *Handler) HandleResponse(headers map[string]string) {
	if location, ok := headers["location"]; ok {
		h.devices = append(h.devices, location)
	}
}

// getLocalIP returns the local IP address
func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// Discover searches for YouTube TV devices on the network
func Discover(ctx context.Context, client *http.Client) ([]Device, error) {
	handler := NewHandler()
	handler.Clear()

	// Get local IP address
	localIP, err := getLocalIP()
	if err != nil {
		return nil, fmt.Errorf("failed to get local IP: %w", err)
	}

	// Create UDP connection
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", localIP, 0))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen UDP: %w", err)
	}
	defer conn.Close()

	// Create M-SEARCH request
	searchRequest := fmt.Sprintf(
		"M-SEARCH * HTTP/1.1\r\n"+
			"HOST: %s:%d\r\n"+
			"MAN: \"ssdp:discover\"\r\n"+
			"MX: 10\r\n"+
			"ST: %s\r\n"+
			"\r\n",
		multicastAddress, port, searchTarget)

	// Send M-SEARCH request
	multicastAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", multicastAddress, port))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve multicast address: %w", err)
	}

	_, err = conn.WriteToUDP([]byte(searchRequest), multicastAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to send M-SEARCH request: %w", err)
	}

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(4 * time.Second))

	// Read responses
	buffer := make([]byte, 1500)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		response := string(buffer[:n])
		headers := parseSSDPHeaders(response)
		handler.HandleResponse(headers)
	}

	// Process discovered devices
	var devices []Device
	for _, location := range handler.devices {
		device, err := findYouTubeApp(ctx, client, location)
		if err != nil {
			continue
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// findYouTubeApp finds YouTube app information from a device location
func findYouTubeApp(ctx context.Context, client *http.Client, location string) (Device, error) {
	// Get device description
	req, err := http.NewRequestWithContext(ctx, "GET", location, nil)
	if err != nil {
		return Device{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return Device{}, fmt.Errorf("failed to get device description: %w", err)
	}
	defer resp.Body.Close()

	var deviceDesc struct {
		Root struct {
			Device struct {
				FriendlyName string `xml:"friendlyName"`
			} `xml:"device"`
		} `xml:"root"`
	}

	if err := xml.NewDecoder(resp.Body).Decode(&deviceDesc); err != nil {
		return Device{}, fmt.Errorf("failed to decode device description: %w", err)
	}

	// Get YouTube app URL
	appURL := resp.Header.Get("application-url")
	if appURL == "" {
		return Device{}, fmt.Errorf("no application URL found")
	}

	youtubeURL := appURL + "YouTube"
	req, err = http.NewRequestWithContext(ctx, "GET", youtubeURL, nil)
	if err != nil {
		return Device{}, fmt.Errorf("failed to create YouTube request: %w", err)
	}

	resp, err = client.Do(req)
	if err != nil {
		return Device{}, fmt.Errorf("failed to get YouTube app info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Device{}, fmt.Errorf("YouTube app not found")
	}

	var youtubeInfo struct {
		Service struct {
			AdditionalData struct {
				ScreenID string `xml:"screenId"`
			} `xml:"additionalData"`
		} `xml:"service"`
	}

	if err := xml.NewDecoder(resp.Body).Decode(&youtubeInfo); err != nil {
		return Device{}, fmt.Errorf("failed to decode YouTube info: %w", err)
	}

	return Device{
		ScreenID: youtubeInfo.Service.AdditionalData.ScreenID,
		Name:     deviceDesc.Root.Device.FriendlyName,
		Offset:   0,
	}, nil
}

// parseSSDPHeaders parses SSDP response headers
func parseSSDPHeaders(response string) map[string]string {
	headers := make(map[string]string)
	lines := strings.Split(response, "\r\n")
	for _, line := range lines {
		if idx := strings.Index(line, ":"); idx != -1 {
			key := strings.ToLower(strings.TrimSpace(line[:idx]))
			value := strings.TrimSpace(line[idx+1:])
			headers[key] = value
		}
	}
	return headers
}
