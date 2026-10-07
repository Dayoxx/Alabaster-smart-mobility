package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"transportation/internal/models"
)

func (app *App) TerminalHandler(c *gin.Context) {
	var input struct {
		DriverID     string   `json:"driver_id" binding:"required"`
		DeviceSerial string   `json:"device_serial" binding:"required"`
		VehiclePlate string   `json:"vehicle_plate" binding:"required"`
		TerminalType string   `json:"terminal_type" binding:"required"`
		VehicleType  string   `json:"vehicle_type" binding:"required"`
		LastLat      *float64 `json:"last_lat"`
		LastLng      *float64 `json:"last_lng"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	driverID, err := uuid.Parse(input.DriverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid driver_id",
		})
		return
	}

	validTerminalTypes := map[string]bool{
		"nfc_reader": true,
	}

	if !validTerminalTypes[input.TerminalType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "terminal_type must be nfc_reader",
		})
		return
	}

	validVehicleTypes := map[string]bool{
		"keke": true,
		"bus":  true,

		// Future driver-controlled vehicle types:
		// "taxi": true,

		// Future company/operator-controlled transport:
		// "train": true,
	}

	if !validVehicleTypes[input.VehicleType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "vehicle_type must be keke or bus",
		})
		return
	}

	existing, err := app.Terminals.FindByDeviceSerial(
		c.Request.Context(),
		input.DeviceSerial,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not check existing terminals",
		})
		return
	}

	if existing != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_serial already registered",
		})
		return
	}

	lastSeenAt := time.Now()

	newTerminal := models.Terminal{
		ID:           uuid.New(),
		DriverID:     driverID,
		DeviceSerial: input.DeviceSerial,
		VehiclePlate: input.VehiclePlate,
		TerminalType: input.TerminalType,
		VehicleType:  input.VehicleType,
		LastLat:      input.LastLat,
		LastLng:      input.LastLng,
		LastSeenAt:   &lastSeenAt,
		IsActive:     true,
	}

	saved, err := app.Terminals.Create(
		c.Request.Context(),
		newTerminal,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not create terminal",
		})
		return
	}

	c.JSON(http.StatusCreated, saved)
}
