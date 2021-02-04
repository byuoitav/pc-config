package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	pcconfig "github.com/byuoitav/pc-config"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	ConfigService     pcconfig.ConfigService
	ControlKeyService pcconfig.ControlKeyService
}

func (h *Handlers) ConfigForPC(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	room, cg, err := h.ConfigService.RoomAndControlGroup(ctx, c.Param("hostname"))
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("unable to get room/controlGroup: %s", err))
		return
	}

	var config pcconfig.Config

	cameras, err := h.ConfigService.Cameras(ctx, room, cg)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("unable to get cameras: %s", err))
		return
	}

	config.Cameras = cameras

	key, err := h.ControlKeyService.ControlKey(ctx, room, cg)
	if err != nil {
		// ignore this error, just don't set the key
		c.JSON(http.StatusOK, config)
		return
	}

	config.ControlKey = key
	c.JSON(http.StatusOK, config)
}
