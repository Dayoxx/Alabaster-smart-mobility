package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"transportation/internal/models"
)

func (app *App) LinkAccountHandler(c *gin.Context) {
	var input struct {
		PassengerID   string `json:"passenger_id" binding:"required"`
		AccountNumber string `json:"account_number" binding:"required"`
		BankCode      string `json:"bank_code" binding:"required"`
		BankName      string `json:"bank_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error check your linkacc logic": err.Error()})
		return
	}

	passengerID, err := uuid.Parse(input.PassengerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passenger_id"})
		return
	}

	existingCount, err := app.Accounts.CountByPassenger(c.Request.Context(), passengerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not check existing accounts internal error detected",
		})
		return
	}

	newAccount := models.PassengerAccount{
		ID:            uuid.New(),
		PassengerID:   passengerID,
		AccountNumber: input.AccountNumber,
		BankCode:      input.BankCode,
		BankName:      input.BankName,
		PriorityOrder: existingCount + 1,
		IsActive:      true,
	}

	saved, err := app.Accounts.Create(c.Request.Context(), newAccount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not link account internal error detected",
		})
		return
	}

	c.JSON(http.StatusCreated, saved)
}
