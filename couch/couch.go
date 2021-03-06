package couch

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	pcconfig "github.com/byuoitav/pc-config"
	_ "github.com/go-kivik/couchdb/v3"
	"github.com/go-kivik/kivik/v3"
)

type configService struct {
	client      *kivik.Client
	uiConfigDB  string
	pcMappingDB string
}

// New creates a new ConfigService, created a couchdb client pointed at url.
func New(ctx context.Context, url string, opts ...Option) (pcconfig.ConfigService, error) {
	client, err := kivik.New("couch", url)
	if err != nil {
		return nil, fmt.Errorf("unable to build client: %w", err)
	}

	return NewWithClient(ctx, client, opts...)
}

// NewWithClient creates a new ConfigService using the given client.
func NewWithClient(ctx context.Context, client *kivik.Client, opts ...Option) (pcconfig.ConfigService, error) {
	options := options{
		uiConfigDB:  _defaultUIConfigDB,
		pcMappingDB: _defaultPCMappingDB,
	}

	for _, o := range opts {
		o.apply(&options)
	}

	if options.authFunc != nil {
		if err := client.Authenticate(ctx, options.authFunc); err != nil {
			return nil, fmt.Errorf("unable to authenticate: %w", err)
		}
	}

	return &configService{
		client:      client,
		uiConfigDB:  options.uiConfigDB,
		pcMappingDB: options.pcMappingDB,
	}, nil
}

func (c *configService) RoomAndControlGroup(ctx context.Context, hostname string) (string, string, error) {
	var mapping pcMapping
	var err error
	db := c.client.DB(ctx, c.pcMappingDB)

	for {
		err = db.Get(ctx, hostname).ScanDoc(&mapping)
		switch kivik.StatusCode(err) {
		case 0:
			return mapping.UIConfig, mapping.ControlGroup, nil
		case http.StatusNotFound:
			// try again
		default:
			return "", "", fmt.Errorf("unable to get/scan pc mapping: %w", err)
		}

		if len(hostname) < 4 {
			break
		}

		hostname = hostname[:len(hostname)-1]
	}

	return "", "", fmt.Errorf("unable to get/scan pc mapping: %w", err)
}

func (c *configService) Cameras(ctx context.Context, room, controlGroup string) ([]pcconfig.Camera, error) {
	var config uiConfig

	db := c.client.DB(ctx, c.uiConfigDB)
	if err := db.Get(ctx, room).ScanDoc(&config); err != nil {
		return []pcconfig.Camera{}, fmt.Errorf("unable to get/scan ui config: %w", err)
	}

	for _, cg := range config.ControlGroups {
		if cg.ID == controlGroup {
			return cg.Cameras, nil
		}
	}

	return []pcconfig.Camera{}, errors.New("no matching control group found")
}
