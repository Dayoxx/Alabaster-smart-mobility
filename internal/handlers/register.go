package handlers

import (
	"github.com/google/uuid"
	"net/http"
	"time"
	"transportation/internal/models"

	"github.com/gin-gonic/gin"
)

func (app *App) RegisterHandler(c *gin.Context) {
	var input struct {
		Phone    string `json:"phone" binding:"required"`
		FullName string `json:"full_name" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error check your regihand logic": err.Error()})
		return
	}

	newUser := models.User{
		ID: uuid.New(), Phone: input.Phone, FullName: input.FullName,
		Role: input.Role, CreatedAt: time.Now(),
	}
	saved, err := app.Users.Create(c.Request.Context(), newUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create user"})
		return
	}
	c.JSON(http.StatusCreated, saved)
}

func (app *App) linkAccountHandler(c *gin.Context) {
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

	existingCount, err := app.Accounts.CountByPassenger(
		c.Request.Context(),
		passengerID,
	)
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

	saved, err := app.Accounts.Create(
		c.Request.Context(),
		newAccount,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not link account internal error detected",
		})
		return
	}

	c.JSON(http.StatusCreated, saved)
}
