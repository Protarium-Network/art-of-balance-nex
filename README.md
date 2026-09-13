# Art of Balance — NEX server

A preservation-oriented NEX server for the Wii U title **Art of Balance**
(Shin'en Multimedia, 2013). It speaks the game's PRUDP authentication and
secure protocols so its **online leaderboards** (and matchmaking, if the
game genuinely uses it online — unconfirmed, see below) work again after the
official servers went away.

Built on the [Pretendo Network](https://github.com/PretendoNetwork) NEX
libraries (`nex-go`, `nex-protocols-go`, `nex-protocols-common-go`), the same
stack as this org's Sonic & All-Stars Racing Transformed, Mario Tennis,
Hyrule Warriors and Trine 2 servers. Ranking uses the same vendored fork as
Hyrule Warriors (upstream v2.4.0 has no mode-aware ranking query or
PID-scoped common data — see [NOTICE](NOTICE)); everything else, including
matchmaking, is unmodified upstream, and `nex-protocols-common-go` provisions
its own matchmaking schema.

> **Login and secure-connection handshake are console-confirmed** (2026-09-13
> debugging session, see [RECON.md](RECON.md)). Unlike most sibling servers
> in this project, Art of Balance had no entry in kinnay.github.io's public
> NEX database to start from — the access key and game server ID were both
> wrong on the first console test and had to be corrected using a real
> packet capture. Ranking and the full matchmaking flow (finding/joining a
> session) are not yet exercised. See RECON.md for the full evidence trail
> and remaining open questions, and [PROTOCOL_COVERAGE.md](PROTOCOL_COVERAGE.md)
> for per-protocol status.

## Recovered configuration

| Field | Value | Source |
|---|---|---|
| Title ID (EUR) | `0005000010149400` | this dump's own `meta.xml` (product code `WUP-P-WABP`) |
| Title ID (USA / JPN) | `0005000010135000` / `000500001017cb00` | local `wiiu.json` community game-info cache |
| Game server ID | `10135000` | **console-confirmed** — the real EUR console requests this exact ID, shared across all regions (not the EUR-specific value the title-ID pattern predicted) |
| Access key | `96900116` | **console-confirmed** — bruteforced against a real captured PRUDPv1 SYN packet, cryptographically unique for PRUDPv1 |
| NEX version | `3.7.1` | RPX debug strings (`[SDK+Nintendo:NEX_3_7_1]`) |
| PRUDPv1 connection signature | modern (`LegacyConnectionSignature = false`) | **console-confirmed** — the legacy scheme sent an all-zero signature a real console rejected |
| Structure headers | on (`UseStructureHeader = true`) | pattern-matched, never implicated in an observed error, **not independently proven** |
| MatchMaking lib | default (3.7.1, no override) | NEX 3.7.1 is above the 3.4.0 serialization break; matchmaking itself not yet exercised |

The game's own executable (`project.rpx`) statically links NEX and drives it
through `nn::act::AcquireNexServiceToken` — the standard Wii U NASC →
nex_token → PRUDP path. RTTI/typeinfo symbol enumeration confirms it links
Ranking, MatchMaking, MatchmakeExtension, NAT Traversal, Secure Connection,
Account Management and Utility, and confirms it does **not** link DataStore,
Friends or Miiverse.

### Multi-region note

PRUDP itself carries no "game server ID" concept — that only matters to
whatever account/token server issues `nex_token`s. This server needs just
ONE access key, ONE NEX version and (console-confirmed) ONE game server ID
(`10135000`) to serve every region's client — Art of Balance shares a single
game server across all three region title IDs, the same pattern already
documented on Trine 2.

## Scope

- **Ticket Granting** — login / secure-server handoff
- **Secure Connection**, **Utility** — baseline secure-endpoint handshake
- **Ranking** — leaderboards (score upload, category/mode queries, per-player
  common data)
- **NAT Traversal + MatchMaking + MatchMakingExt + MatchmakeExtension** —
  matchmaking, wired the same way as every other sibling server; whether Art
  of Balance's online mode is genuine live multiplayer or mostly idle (like
  Hyrule Warriors) is unconfirmed

No DataStore, no S3 — the game links neither.

## Database

One PostgreSQL database. This server creates its own `artofbalance_*`
ranking tables in `database/init_postgres.go`. **The matchmaking schema is
created by `nex-protocols-common-go` itself** — `CommonProtocol.SetManager`
(both the `match-making` and `matchmake-extension` protocols) runs
`CREATE SCHEMA IF NOT EXISTS` + `CREATE TABLE IF NOT EXISTS` for
`matchmaking.gatherings` / `matchmake_sessions` / `persistent_gatherings` /
`community_participations` / `notifications` and every `tracking.*` table.
This server does not hand-author that schema itself — see
[docs/matchmaking-schema.md](docs/matchmaking-schema.md).

## Running

### Local preservation mode

```bash
cp .env.example .env                     # PN_ARTOFBALANCE_LOCAL_MODE=1 by default
cp settings.example.json settings.json   # add your console's PID + NEX password
docker compose up --build
```

In local mode there is no account server: player NEX passwords come from
`settings.json` and the login token is accepted unconditionally. Use it only
on an isolated network. Set `PN_ARTOFBALANCE_SECURE_HOST` to the LAN IP the
console can reach this machine on (not `localhost`, unless the client runs
here too).

### Shared mode

Set `PN_ARTOFBALANCE_LOCAL_MODE` to anything but `1` and provide
`PN_ARTOFBALANCE_NEX_TOKEN_AES_KEY` (64 hex chars) and
`PN_ARTOFBALANCE_NEX_PASSWORD_SECRET` (≥32 bytes hex), both matching your
account server. Login tokens are then decrypted and validated, and each
player's NEX password is derived as `HMAC-SHA256(secret, pid)` — the same
scheme every other Pretendo-style Wii U NEX server in this project uses.

### Without Docker

```bash
export PN_ARTOFBALANCE_AUTH_PORT=26500 PN_ARTOFBALANCE_SECURE_PORT=26501
export PN_ARTOFBALANCE_SECURE_HOST=<LAN-IP-of-this-machine>
export PN_ARTOFBALANCE_POSTGRES_URI='postgres://artofbalance:artofbalance@localhost:5432/artofbalance?sslmode=disable'
export PN_ARTOFBALANCE_LOCAL_MODE=1
go build -o art-of-balance-nex . && ./art-of-balance-nex
```

## Deployed instance (Protarium VPS)

Running in shared mode on the Protarium estate's VPS, behind a dedicated
token issuer (`account-server`, built from
[Protarium-Network/account-server](https://github.com/Protarium-Network/account-server))
with its own freshly generated AES key / password secret (not the shared
estate secret — this game's token issuer and NEX server only need to agree
with each other). The estate's nginx `game_server_id` routing map sends the
confirmed ID (`10135000`) to this token issuer, which in turn points
consoles at this server. Postgres + the Go server run via `docker compose`,
same pattern as the other sibling servers' VPS deployments.

## What is verified

Locally: the server builds cleanly (`go build`/`go vet`/`gofmt`), its one
non-trivial unit of logic (HMAC password derivation + local-settings
fallback) has a test, and it has been smoke-tested end to end via
`docker compose up --build`.

Against a real Wii U (2026-09-13, see RECON.md's "Console debugging
session"): `LoginEx`, `RequestTicket`, `SecureConnection::Register` and
`Utility::AcquireNexUniqueID` all succeed. The access key, game server ID
and PRUDPv1 connection-signature scheme are console-confirmed correct.
**Not yet confirmed:** `UseStructureHeader`, the MatchMaking wire version,
Ranking, and whether a full matchmaking session (finding/joining another
player) works end to end.

## License

AGPL-3.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE). No proprietary
Nintendo or Shin'en code or assets are included.
