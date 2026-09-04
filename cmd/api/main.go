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

	svr := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}

	log.Printf("listening on %s", svr.Addr)
	log.Fatal(svr.ListenAndServe())
}

// Registerhandler
func (app *App) registerHandler(c *gin.Context) {
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

// Tap handler
func (app *App) tapHandler(c *gin.Context) {
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

// Trips
func (app *App) tripsHandler(c *gin.Context) {
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

// TerminalHandler
func (app *App) terminalHandler(c *gin.Context) {
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
