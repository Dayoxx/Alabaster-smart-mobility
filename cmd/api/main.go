package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"transportation/internal/db"
	"transportation/internal/handlers"
	"transportation/internal/repository"
)

func main() {
	conn, err := db.NewPostgresDB()
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer conn.Close()

	app := &handlers.App{
		Users:     repository.NewPostgresUserRepository(conn),
		Accounts:  repository.NewPostgresPassengerAccountRepository(conn),
		Terminals: repository.NewPostgresTerminalRepository(conn),
		Trips:     repository.NewPostgresTripRepository(conn),
	}

	router := gin.Default()
	router.POST("/register", app.RegisterHandler)
	router.POST("/link-account", app.LinkAccountHandler)
	router.POST("/tap", app.TapHandler)
	router.GET("/trips", app.TripsHandler)
	router.POST("/terminals", app.TerminalHandler)

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
