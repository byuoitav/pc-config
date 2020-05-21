package couch

import (
	"context"
	"fmt"

	pcconfig "github.com/byuoitav/pc-config"
	_ "github.com/go-kivik/couchdb/v4"
	"github.com/go-kivik/kivik/v4"
)

type ConfigService struct {
	Address  string
	Username string
	Password string
}

const (
	_dbUIConfig  = "ui-configuration"
	_dbPCMapping = "pc-mapping"
)

func (c *ConfigService) CamerasForPC(ctx context.Context, hostname string) ([]pcconfig.Camera, error) {
	addr := fmt.Sprintf("https://%s:%s@%s", c.Username, c.Password, c.Address)
	client, err := kivik.New("couch", addr)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to couch: %w", err)
	}

	mappingDB := client.DB(ctx, _dbPCMapping)
	row := mappingDB.Get(ctx, hostname)

	var pcMapping pcMapping
	if err := row.ScanDoc(&pcMapping); err != nil {
		return nil, fmt.Errorf("unable to get pc mapping for %q: %w", hostname, err)
	}

	uiConfigDB := client.DB(ctx, _dbUIConfig)
	row = uiConfigDB.Get(ctx, pcMapping.UIConfig)

	var uiConfig uiConfig
	if err := row.ScanDoc(&uiConfig); err != nil {
		return nil, fmt.Errorf("unable to get pc mapping: %w", err)
	}

	for _, cg := range uiConfig.ControlGroups {
		if cg.ID == pcMapping.ControlGroup {
			return cg.Cameras, nil
		}
	}

	return nil, fmt.Errorf("no control group matching %q was found in %q", pcMapping.ControlGroup, pcMapping.UIConfig)
}
