package handlers

import (
	"github.com/google/uuid"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (app *App) TripsHandler(c *gin.Context) {
	passengerIDStr := c.Query("passenger_id")
	driverIDStr := c.Query("driver_id")

	if passengerIDStr == "" && driverIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "passenger_id or driver_id is required",
		})
		return
	}

	if passengerIDStr != "" {
		passengerID, err := uuid.Parse(passengerIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid passenger_id",
			})
			return
		}

		trips, err := app.Trips.FindByPassenger(
			c.Request.Context(),
			passengerID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "could not fetch passenger trips",
			})
			return
		}

		c.JSON(http.StatusOK, trips)
		return
	}

	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid driver_id",
		})
		return
	}

	trips, err := app.Trips.FindByDriver(
		c.Request.Context(),
		driverID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not fetch driver trips",
		})
		return
	}

	c.JSON(http.StatusOK, trips)
}
