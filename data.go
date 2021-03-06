package pcconfig

import "context"

// ConfigService talks the a datastore to get configuration information.
type ConfigService interface {
	// RoomAndControlGroup returns the room and control group associated with a PC's hostname
	RoomAndControlGroup(ctx context.Context, hostname string) (string, string, error)

	// Cameras returns the camera configurations for the given room and control group
	Cameras(ctx context.Context, room, controlGroup string) ([]Camera, error)
}

// ControlKeyService gets the control key for a room
type ControlKeyService interface {
	ControlKey(ctx context.Context, room, controlGroup string) (string, error)
}

type Camera struct {
	DisplayName string `json:"displayName"`

	TiltUp      string `json:"tiltUp"`
	TiltDown    string `json:"tiltDown"`
	PanLeft     string `json:"panLeft"`
	PanRight    string `json:"panRight"`
	PanTiltStop string `json:"panTiltStop"`

	ZoomIn   string `json:"zoomIn"`
	ZoomOut  string `json:"zoomOut"`
	ZoomStop string `json:"zoomStop"`

	Stream string `json:"stream"`

	Presets []CameraPreset `json:"presets"`
}

type CameraPreset struct {
	DisplayName string `json:"displayName"`
	SetPreset   string `json:"setPreset"`
}
