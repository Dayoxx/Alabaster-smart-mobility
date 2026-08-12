package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"transportation/internal/db"
	"transportation/internal/models"
	"transportation/internal/repository"
)

type App struct {
	Users     repository.UserRepository
	Accounts  repository.PassengerAccountRepository
	Terminals repository.TerminalRepository
	Trips     repository.TripRepository
}

func main() {
	conn, err := db.NewPostgresDB()
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer conn.Close()

	app := &App{
		Users:     repository.NewPostgresUserRepository(conn),
		Accounts:  repository.NewPostgresPassengerAccountRepository(conn),
		Terminals: repository.NewPostgresTerminalRepository(conn),
		Trips:     repository.NewPostgresTripRepository(conn),
	}

	router := gin.Default()
	router.POST("/register", app.registerHandler)
	router.POST("/link-account", app.linkAccountHandler)
	router.POST("/tap", app.tapHandler)
	router.GET("/trips", app.tripsHandler)
	router.POST("/terminals", app.terminalHandler)

	mux := http.NewServeMux()
	mux.Handle("/", router)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}

	log.Printf("listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}

// Registerhandler
func (app *App) registerHandler(c *gin.Context) {
	var input struct {
		Phone    string `json:"phone" binding:"required"`
		FullName string `json:"full_name" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			"error": "could not check existing accounts",
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
			"error": "could not link account",
		})
		return
	}

	c.JSON(http.StatusCreated, saved)
}

// Tap handler
func (app *App) tapHandler(c *gin.Context) {
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

	terminal, err := app.Terminals.FindByID(c.Request.Context(), input.TerminalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not look up terminal"})
		return
	}
	if terminal == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "terminal not found"})
		return
	}

	accounts, err := app.Accounts.FindActiveByPassenger(c.Request.Context(), input.PassengerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not look up passenger accounts"})
		return
	}
	if len(accounts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passenger has no linked accounts"})
		return
	}
	usedAccount := accounts[0]

	platformFee := input.Amount * 0.05
	driverPayout := input.Amount - platformFee

	newTrip := models.Trip{
		ID: uuid.New(), PassengerID: input.PassengerID, DriverID: terminal.DriverID,
		TerminalID: input.TerminalID, FareAmount: input.Amount, DriverPayout: driverPayout,
		PlatformFee: platformFee, PickupLat: input.Lat, PickupLng: input.Lng,
		Status: "success", TappedAt: time.Now(),
	}
	saved, err := app.Trips.Create(c.Request.Context(), newTrip)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save trip"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"trip": saved, "paid_from": usedAccount.BankName, "driver_payout": driverPayout})
}

func (app *App) tripsHandler(c *gin.Context) {
	trips, err := app.Trips.FindByPassengerOrDriver(c.Request.Context(), c.Query("passenger_id"), c.Query("driver_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch trips"})
		return
	}
	c.JSON(http.StatusOK, trips)
}

func (app *App) terminalHandler(c *gin.Context) {
	var input struct {
		DriverID     string  `json:"driver_id" binding:"required"`
		DeviceSerial string  `json:"device_serial" binding:"required"`
		VehiclePlate string  `json:"vehicle_plate" binding:"required"`
		TerminalType string  `json:"terminal_type" binding:"required"`
		LastLat      float64 `json:"last_lat"`
		LastLng      float64 `json:"last_lng"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validTypes := map[string]bool{"keke": true, "bus": true}
	if !validTypes[input.TerminalType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "terminal_type must be keke or bus"})
		return
	}

	existing, err := app.Terminals.FindByDeviceSerial(c.Request.Context(), input.DeviceSerial)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not check existing terminals"})
		return
	}
	if existing != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_serial already registered"})
		return
	}

	newTerminal := models.Terminal{
		ID: uuid.New(), DriverID: input.DriverID, DeviceSerial: input.DeviceSerial,
		VehiclePlate: input.VehiclePlate, TerminalType: input.TerminalType,
		LastLat: input.LastLat, LastLng: input.LastLng, LastSeenAt: time.Now(), IsActive: true,
	}
	saved, err := app.Terminals.Create(c.Request.Context(), newTerminal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create terminal"})
		return
	}
	c.JSON(http.StatusCreated, saved)
}
