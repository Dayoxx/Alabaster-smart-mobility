package main

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// user structure
type User struct {
	ID        string    `json:"id"`
	Phone     string    `json:"phone"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// passenger structure
type PassengerAccount struct {
	ID            string `json:"id"`
	PassengerID   string `json:"passenger_id"`
	AccountNumber string `json:"account_number"`
	BankCode      string `json:"bank_code"`
	BankName      string `json:"bank_name"`
	PriorityOrder int    `json:"priority_order"`
	IsActive      bool   `json:"is_active"`
}

// trip structure
type Trip struct {
	ID           string    `json:"id"`
	PassengerID  string    `json:"passenger_id"`
	DriverID     string    `json:"driver_id"`
	TerminalID   string    `json:"terminal_id"`
	FareAmount   float64   `json:"fare_amount"`
	DriverPayout float64   `json:"driver_payout"`
	PlatformFee  float64   `json:"platform_fee"`
	PickupLat    float64   `json:"pickup_lat"`
	PickupLng    float64   `json:"pickup_lng"`
	Status       string    `json:"status"`
	TappedAt     time.Time `json:"tapped_at"`
}

type Terminal struct {
	ID           string    `json:"id"`
	DriverID     string    `json:"driver_id"`
	DeviceSerial string    `json:"device_serial"`
	VehiclePlate string    `json:"vehicle_plate"`
	TerminalType string    `json:"terminal_type"` // "keke", "bus", "ferry", "train", "plane", "tram", "monorail"
	LastLat      float64   `json:"last_lat"`
	LastLng      float64   `json:"last_lng"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	IsActive     bool      `json:"is_active"`
}

// user value, passenger value, trip value
var Users = []User{}
var Passengers = []PassengerAccount{}
var Trips = []Trip{}
var Terminals = []Terminal{}

// main function server
func main() {
	router := gin.Default()

	router.POST("/register", registerHandler)        // filled
	router.POST("/link-account", linkAccountHandler) // filled
	router.POST("/tap", tapHandler)
	router.GET("/trips", tripsHandler)         //filled
	router.POST("/terminals", TerminalHandler) //filled
	router.Run(":8080")
}

// register handler function
func registerHandler(c *gin.Context) {
	var input struct {
		Phone    string `json:"phone" binding:"required"`
		FullName string `json:"full_name" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//create a new user and append to the Users slice
	eachUser := User{
		ID:        uuid.New().String(),
		Phone:     input.Phone,
		FullName:  input.FullName,
		Role:      input.Role,
		CreatedAt: time.Now(),
	}

	Users = append(Users, eachUser)
	c.JSON(http.StatusCreated, eachUser)
}

func linkAccountHandler(c *gin.Context) {
	var input struct {
		PassengerID   string `json:"passenger_id" binding:"required"`
		AccountNumber string `json:"account_number" binding:"required"`
		BankCode      string `json:"bank_code" binding:"required"`
		BankName      string `json:"bank_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existingCount := 0
	for _, passenger := range Passengers {
		if passenger.PassengerID == input.PassengerID {
			existingCount++
		}
	}
	//create a new passenger and append to the Passengers slice
	newAccount := PassengerAccount{
		ID:            uuid.New().String(),
		PassengerID:   input.PassengerID,
		AccountNumber: input.AccountNumber,
		BankCode:      input.BankCode,
		BankName:      input.BankName,
		PriorityOrder: existingCount + 1, // next in line
		IsActive:      true,
	}

	Passengers = append(Passengers, newAccount)
	c.JSON(http.StatusCreated, newAccount)
}

func tapHandler(c *gin.Context) {
	var input struct {
		PassengerID string  `json:"passenger_id" binding:"required"`
		TerminalID  string  `json:"terminal_id" binding:"required"`
		Amount      float64 `json:"amount" binding:"required"`
		Lat         float64 `json:"lat"`
		Lng         float64 `json:"lng"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//find the terminal to get the driver ID and terminal type
	var terminal *Terminal
	for i := range Terminals {
		if Terminals[i].ID == input.TerminalID {
			terminal = &Terminals[i]
			break
		}
	}

	if terminal == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "terminal not found"})
		return
	}
	//
	var PassengerAccounts []PassengerAccount
	for _, acc := range Passengers {
		if acc.PassengerID == input.PassengerID && acc.IsActive {
			PassengerAccounts = append(PassengerAccounts, acc)
		}
	}
	sort.Slice(PassengerAccounts, func(i, j int) bool {
		return PassengerAccounts[i].PriorityOrder < PassengerAccounts[j].PriorityOrder
	})

	if len(PassengerAccounts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passenger has no linked accounts"})
		return
	}
	// 3. Simulate trying each account until one "succeeds"
	//    (Moniepoint call goes here later — for now, first account always works)
	usedAccount := PassengerAccounts[0]

	// 4. Calculate the split
	platformFee := input.Amount * 0.05
	driverPayout := input.Amount - platformFee

	// 5. Build and save the trip
	newTrip := Trip{
		ID:           uuid.New().String(),
		PassengerID:  input.PassengerID,
		DriverID:     terminal.DriverID,
		TerminalID:   input.TerminalID,
		FareAmount:   input.Amount,
		DriverPayout: driverPayout,
		PlatformFee:  platformFee,
		PickupLat:    input.Lat,
		PickupLng:    input.Lng,
		Status:       "success",
		TappedAt:     time.Now(),
	}

	Trips = append(Trips, newTrip)

	c.JSON(http.StatusCreated, gin.H{
		"trip":          newTrip,
		"paid_from":     usedAccount.BankName,
		"driver_payout": driverPayout,
	})
}

// tripsHandler handles the GET /trips endpoint to retrieve all trips.
func tripsHandler(c *gin.Context) {
	passengerID := c.Query("passenger_id")
	driverID := c.Query("driver_id")

	var result []Trip
	for _, t := range Trips {
		if (passengerID != "" && t.PassengerID == passengerID) ||
			(driverID != "" && t.DriverID == driverID) {
			result = append(result, t)
		}
	}

	c.JSON(http.StatusOK, result)
}
func TerminalHandler(c *gin.Context) {
	var input struct {
		ID           string    `json:"id"`
		DriverID     string    `json:"driver_id"`
		DeviceSerial string    `json:"device_serial"`
		VehiclePlate string    `json:"vehicle_plate"`
		TerminalType string    `json:"terminal_type"` // "keke", "bus", "ferry", "train", "plane", "tram", "monorail"
		LastLat      float64   `json:"last_lat"`
		LastLng      float64   `json:"last_lng"`
		LastSeenAt   time.Time `json:"last_seen_at"`
		IsActive     bool      `json:"is_active"`
	}
	// Bind the JSON input to the struct
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	validTypes := map[string]bool{"keke": true, "bus": true, "ferry": false, "train": false, "plane": false, "tram": false, "monorail": false}
	if !validTypes[input.TerminalType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "terminal_type must be keke or bus"})
		return
	}

	for _, terminal := range Terminals {
		if terminal.TerminalType == input.TerminalType {
			c.JSON(http.StatusBadRequest, gin.H{"error": "device_serial already registered"})
			return
		}
	}

	Terminalcheck := Terminal{
		ID:           uuid.New().String(),
		DriverID:     input.DriverID,
		DeviceSerial: input.DeviceSerial,
		VehiclePlate: input.VehiclePlate,
		TerminalType: input.TerminalType,
		LastLat:      input.LastLat,
		LastLng:      input.LastLng,
		LastSeenAt:   time.Now(),
		IsActive:     true,
	}

	Terminals = append(Terminals, Terminalcheck)
	c.JSON(http.StatusCreated, Terminalcheck)
}
