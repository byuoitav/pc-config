package handlers

import (
	"context"
	"net/http"
	"time"

	pcconfig "github.com/byuoitav/pc-config"
	"github.com/labstack/echo"
)

type Handlers struct {
	ConfigService pcconfig.ConfigService
}

func (h *Handlers) ConfigForPC(c echo.Context) error {
	hostname := c.Param("hostname")
	if len(hostname) == 0 {
		return c.String(http.StatusBadRequest, "hostname must not be empty")
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	cameras, err := h.ConfigService.CamerasForPC(ctx, hostname)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, pcconfig.Config{
		Cameras: cameras,
	})
}
