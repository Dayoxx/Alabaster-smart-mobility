package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)
//user structure
type User struct {
	ID        string  `json:"id"`
	Phone	  string  `json:"phone"`
	FullName  string  `json:"full_name"`
	Role	  string  `json:"role"`
	CreatedAt  time.Time `json:"created_at"`
}
//passenger structure
type Passenger struct {
	ID 	  string  `json:"id"`
	PassengerID string  `json:"passenger_id"`
	AccountNumber string  `json:"account_number"`
	BankCode string  `json:"bank_code"`
	PriorityOrder int  `json:"priority_order"`
	IsActive bool  `json:"is_active"`
}
//trip structure
type Trip struct {
	ID 	  string  `json:"id"`
	PassengerID string  `json:"passenger_id"`
	DriverID string  `json:"driver_id"`
	TerminalID string  `json:"terminal_id"`
	FareAmount float64  `json:"fare_amount"`
	Status string  `json:"status"`
	TappedAt time.Time `json:"tapped_at"`
}
//user value, passenger value, trip value
var Users = []User 
var Passengers = []Passenger
var Trips = []Trip
//main function
func main() {
	router := gin.Default()

	router.POST("/register", registerHandler )
	router.POST("/link-account", linkAccountHandler )
	router.POST("/tap", tapHandler )
	router.GET("/trips", tripsHandler )

	router.Run(":8080")
}
//register handler function
func registerHandler(c *gin.Context) {
	var input struct {
		Phone string `json:"phone" binding:"required"`
		FullName string `json:"full-name" binding:"required"`
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
//create a new user and append to the Users slice
	eachUser := User{
		ID: uuid.New().String(),
		Phone: input.Phone,
		FullName: input.FullName,
		Role: input.Role,
		CreatedAt: time.Now(),
	}

	Users = append(Users, eachUser)
	c.JSON(http.StatusCreated, eachUser)