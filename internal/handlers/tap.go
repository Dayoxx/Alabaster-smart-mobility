package handlers

import (
	"github.com/google/uuid"
	"net/http"
	"time"
	"transportation/internal/models"

	"github.com/gin-gonic/gin"
)

func (app *App) TapHandler(c *gin.Context) {
	var input struct {
		PassengerID string  `json:"passenger_id" binding:"required"`
		TerminalID  string  `json:"terminal_id" binding:"required"`
		AmountKobo  int64   `json:"amount_kobo" binding:"required"`
		Lat         float64 `json:"lat"`
		Lng         float64 `json:"lng"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	passengerID, err := uuid.Parse(input.PassengerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid passenger_id",
		})
		return
	}

	terminalID, err := uuid.Parse(input.TerminalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid terminal_id",
		})
		return
	}

	terminal, err := app.Terminals.FindByID(
		c.Request.Context(),
		terminalID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not look up terminal",
		})
		return
	}

	if terminal == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "terminal not found",
		})
		return
	}

	if !terminal.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "terminal is inactive",
		})
		return
	}

	accounts, err := app.Accounts.FindActiveByPassenger(
		c.Request.Context(),
		passengerID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not look up passenger accounts",
		})
		return
	}

	if len(accounts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "passenger has no linked accounts",
		})
		return
	}

	usedAccount := accounts[0]

	const (
		PassengerPlatformFeeKobo int64 = 1000
		DriverPlatformFeeKobo    int64 = 1000
	)

	if input.AmountKobo <= PassengerPlatformFeeKobo+DriverPlatformFeeKobo {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid fare amount",
		})
		return
	}

	passengerTotal := input.AmountKobo
	fareAmount := passengerTotal - PassengerPlatformFeeKobo
	driverPayout := fareAmount - DriverPlatformFeeKobo
	platformTotal := PassengerPlatformFeeKobo + DriverPlatformFeeKobo

	lat := input.Lat
	lng := input.Lng

	newTrip := models.Trip{
		ID:                   uuid.New(),
		PassengerID:          passengerID,
		DriverID:             terminal.DriverID,
		TerminalID:           terminalID,
		FareAmount:           fareAmount,
		PassengerPlatformFee: PassengerPlatformFeeKobo,
		DriverPlatformFee:    DriverPlatformFeeKobo,
		PassengerTotal:       passengerTotal,
		DriverPayout:         driverPayout,
		PlatformTotal:        platformTotal,
		PickupLat:            &lat,
		PickupLng:            &lng,
		Status:               "success",
		TappedAt:             time.Now(),
	}

	saved, err := app.Trips.Create(
		c.Request.Context(),
		newTrip,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not save trip",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"trip":            saved,
		"paid_from":       usedAccount.BankName,
		"passenger_total": saved.PassengerTotal,
		"driver_payout":   saved.DriverPayout,
	})
}
