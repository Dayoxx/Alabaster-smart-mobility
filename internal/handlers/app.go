package handlers

import "transportation/internal/repository"

type App struct {
	Users     repository.UserRepository
	Accounts  repository.PassengerAccountRepository
	Terminals repository.TerminalRepository
	Trips     repository.TripRepository
}
