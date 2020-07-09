package pcconfig

type Config struct {
	ControlKey string   `json:"controlKey"`
	Cameras    []Camera `json:"cameras"`
}
