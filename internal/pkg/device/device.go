package device

import (
	"sync"
	"time"
)

// Device represents a connected device with its properties
type Device struct {
	ID           string                 // Unique identifier for the device
	Name         string                 // Device name
	Model        string                 // Device model
	LastSeen     time.Time              // Last time the device was seen
	IsConnected  bool                   // Current connection status
	Capabilities []string               // List of device capabilities
	CustomData   map[string]interface{} // Additional device-specific data
}

// DeviceManager handles device registration and management
type DeviceManager struct {
	devices map[string]*Device
	mu      sync.RWMutex
}

// NewDeviceManager creates a new instance of DeviceManager
func NewDeviceManager() *DeviceManager {
	return &DeviceManager{
		devices: make(map[string]*Device),
	}
}

// RegisterDevice adds or updates a device in the manager
func (dm *DeviceManager) RegisterDevice(device *Device) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	device.LastSeen = time.Now()
	device.IsConnected = true
	dm.devices[device.ID] = device
}

// GetDevice retrieves a device by its ID
func (dm *DeviceManager) GetDevice(id string) (*Device, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	device, exists := dm.devices[id]
	return device, exists
}

// UpdateDeviceStatus updates the connection status of a device
func (dm *DeviceManager) UpdateDeviceStatus(id string, connected bool) bool {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if device, exists := dm.devices[id]; exists {
		device.IsConnected = connected
		if connected {
			device.LastSeen = time.Now()
		}
		return true
	}
	return false
}

// ListDevices returns a list of all registered devices
func (dm *DeviceManager) ListDevices() []*Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	devices := make([]*Device, 0, len(dm.devices))
	for _, device := range dm.devices {
		devices = append(devices, device)
	}
	return devices
}

// RemoveDevice removes a device from the manager
func (dm *DeviceManager) RemoveDevice(id string) bool {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.devices[id]; exists {
		delete(dm.devices, id)
		return true
	}
	return false
}

// GetConnectedDevices returns a list of currently connected devices
func (dm *DeviceManager) GetConnectedDevices() []*Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	devices := make([]*Device, 0)
	for _, device := range dm.devices {
		if device.IsConnected {
			devices = append(devices, device)
		}
	}
	return devices
}
