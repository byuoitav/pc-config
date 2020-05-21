package couch

import pcconfig "github.com/byuoitav/pc-config"

type pcMapping struct {
	UIConfig     string `json:"uiConfig"`
	ControlGroup string `json:"controlGroup"`
}

type uiConfig struct {
	ControlGroups []struct {
		ID      string            `json:"name"`
		Cameras []pcconfig.Camera `json:"cameras"`
	} `json:"presets"`
}
