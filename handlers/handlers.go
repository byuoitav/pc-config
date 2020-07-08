package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	pcconfig "github.com/byuoitav/pc-config"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

type Handlers struct {
	ConfigService     pcconfig.ConfigService
	ControlKeyService pcconfig.ControlKeyService
}

func (h *Handlers) ConfigForPC(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	room, preset, err := h.ConfigService.RoomAndPreset(ctx, c.Param("hostname"))
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("unable to get room/preset: %s", err))
		return
	}

	g, ctx := errgroup.WithContext(ctx)
	var config pcconfig.Config

	// get the cameras
	g.Go(func() error {
		var err error
		config.Cameras, err = h.ConfigService.Cameras(ctx, room, preset)
		return err
	})

	// get the control key
	g.Go(func() error {
		var err error
		config.ControlKey, err = h.ControlKeyService.ControlKey(ctx, room, preset)
		return err
	})

	if err := g.Wait(); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, config)
}
