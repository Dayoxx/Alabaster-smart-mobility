# Transportation Model

Transportation Model is a transportation infrastructure platform being built
to support digital transportation services, payments, and trip management.

The project is being developed incrementally with a Go backend and PostgreSQL
as its persistent data store.

---

## Overview

Transportation Model connects the core components involved in a
transportation transaction:

```text
Passenger
    ↓
NFC Terminal
    ↓
Vehicle / Driver
    ↓
Trip

The platform is currently being developed around passenger and driver
transportation services.

Current Development

The current backend provides the foundation for:

User registration
Passenger account linking
Driver registration
NFC terminal registration
Vehicle identification
Trip records
PostgreSQL persistence
Repository-based data access

Current transportation support includes:

Keke
Bus
NFC readers

The project is actively under development.

Architecture

The backend follows a simple layered structure:

HTTP API
    │
    ▼
Handlers
    │
    ▼
Repository Interfaces
    │
    ▼
PostgreSQL

The application separates HTTP handling, application logic, data models,
and persistence so that each layer can evolve independently.

Project Structure
transportation-model/
│
├── cmd/
│   └── api/
│       └── main.go
│
├── internal/
│   ├── db/
│   │   └── db.go
│   │
│   ├── models/
│   │   └── ...
│   │
│   └── repository/
│       ├── repository.go
│       └── postgres.go
│
├── migrations/
│   └── 0001_init.sql
│
├── go.mod
├── go.sum
└── README.md
API

Current endpoints include:

POST /register
POST /link-account
POST /terminals
POST /tap
GET  /trips

The API is being expanded as development continues.

Database

PostgreSQL is used for persistent application data.

The database schema is managed through SQL migrations located in:

migrations/

The application receives its database connection through the
DATABASE_URL environment variable.

Database credentials are not stored in source code.

Local Development
Requirements
Go
PostgreSQL
Git
Configure the Database

Set the database connection:

$env:DATABASE_URL="postgres://USER:PASSWORD@localhost:5433/transportation_model?sslmode=disable"
Install Dependencies
go mod tidy
Build
go build ./...
Run
go run ./cmd/api

The development server runs on:

http://localhost:8080
Development Status

Active Development

The project is currently progressing from its foundational transportation
and persistence layer toward a complete transportation transaction system.

The implementation will continue to evolve as development progresses.

License

This project is currently under development.

This is the level of detail I'd keep **public**. It tells someone what Transportation Model is and how to run it, but doesn't expose the internal roadmap, transaction architecture, security decisions, future infrastructure, or detailed business logic.