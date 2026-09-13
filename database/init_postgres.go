package database

import (
	"os"

	"github.com/Protarium-Network/art-of-balance-nex/globals"
)

// initPostgres creates the tables this server owns directly: the
// artofbalance_* leaderboard tables. Every statement is idempotent, so it is
// safe to run on every start.
//
// The matchmaking.* / tracking.* schema is NOT created here -
// nex-protocols-common-go v2.4.0 self-creates it inside
// CommonProtocol.SetManager (see nex/secure.go). Hand-authoring it first only
// risks a column-shape mismatch that no-ops the library's own CREATE TABLE
// IF NOT EXISTS -- exactly what happened on an earlier sibling server
// (mario-sonic-sochi-2014-nex) and was fixed by trine2-wiiu-nex leaving this
// to the library. See docs/matchmaking-schema.md for the reference layout.
func initPostgres() {
	mustExec := func(label, query string) {
		if _, err := Postgres.Exec(query); err != nil {
			globals.Logger.Criticalf("%s: %s", label, err.Error())
			os.Exit(1)
		}
	}

	// --- Ranking (leaderboards) --------------------------------------------
	mustExec("artofbalance_rankings", `CREATE TABLE IF NOT EXISTS artofbalance_rankings (
		owner_pid   bigint,
		unique_id   bigint,
		category    bigint,
		score       bigint,
		order_by    smallint,
		update_mode smallint,
		groups      bytea,
		param       bigint,
		updated_at  bigint,
		PRIMARY KEY (unique_id, owner_pid, category)
	)`)

	mustExec("artofbalance_ranking_categories", `CREATE TABLE IF NOT EXISTS artofbalance_ranking_categories (
		category   bigint PRIMARY KEY,
		order_by   smallint NOT NULL CHECK (order_by IN (0, 1)),
		created_at bigint NOT NULL
	)`)

	mustExec("artofbalance_common_datas", `CREATE TABLE IF NOT EXISTS artofbalance_common_datas (
		unique_id   bigint,
		owner_pid   bigint,
		common_data bytea,
		updated_at  bigint,
		PRIMARY KEY (unique_id, owner_pid)
	)`)

	mustExec("artofbalance ranking indexes", `
		CREATE INDEX IF NOT EXISTS artofbalance_rankings_category_score_idx
			ON artofbalance_rankings (category, score, updated_at);
		CREATE INDEX IF NOT EXISTS artofbalance_rankings_owner_category_idx
			ON artofbalance_rankings (owner_pid, category);
		CREATE INDEX IF NOT EXISTS artofbalance_common_datas_owner_updated_idx
			ON artofbalance_common_datas (owner_pid, updated_at DESC)
	`)

	globals.Logger.Success("Postgres schema ready (ranking; matchmaking.* is library-created)")
}
