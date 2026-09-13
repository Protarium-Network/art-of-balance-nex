package database

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq"

	"github.com/Protarium-Network/art-of-balance-nex/globals"
)

// Postgres backs both the Ranking leaderboards and the matchmaking manager
// (gatherings, matchmake sessions, participants, tracking logs).
var Postgres *sql.DB

func ConnectPostgres() {
	var err error

	Postgres, err = sql.Open("postgres", os.Getenv("PN_ARTOFBALANCE_POSTGRES_URI"))
	if err != nil {
		globals.Logger.Critical(err.Error())
		os.Exit(1)
	}

	if err = Postgres.Ping(); err != nil {
		globals.Logger.Critical(err.Error())
		os.Exit(1)
	}

	globals.Logger.Success("Connected to Postgres!")

	initPostgres()
}
