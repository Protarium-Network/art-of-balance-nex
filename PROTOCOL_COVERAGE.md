# Protocol coverage

What this server implements for Art of Balance, and how each was confirmed.
Evidence trail: [RECON.md](RECON.md), including the 2026-09-13 console
debugging session that confirmed the transport-layer settings below.

## Transport (PRUDPv1)

`PRUDPV1Settings.LegacyConnectionSignature = false` — **console-confirmed**
2026-09-13: a hand-decoded packet capture showed the legacy/empty scheme
sending an all-zero `connectionSignature` that a real console rejected;
`false` (modern) paired with the correct access key got the console past
`LoginEx` / `RequestTicket` / `SecureConnection::Register`.
`ByteStreamSettings.UseStructureHeader = true` was never implicated in any
observed error, but is still only pattern-matched from the other NEX 3.5+
titles in this project (hyrule-warriors-nex @ 3.8.13,
xenoblade-chronicles-x-nex @ 3.5.16), not independently proven.

## Authentication endpoint

| Protocol | Coverage | Confirmed by |
|---|---|---|
| Ticket Granting | `LoginEx` / `RequestTicket` | **console** — both succeed |

## Secure endpoint

| Protocol | Coverage | Confirmed by |
|---|---|---|
| Secure Connection | insecure `Register` | **console** — succeeds |
| Utility | `AcquireNexUniqueID` (atomic counter) | **console** — the game calls this during login; missing the `GenerateNEXUniqueID` callback caused a 106-0103 disconnect, fixed 2026-09-13 |
| Ranking | leaderboards — 5 callbacks via `nex-protocols-common-go`'s manager | RTTI-confirmed linked (`RankingClient`, `RankingProtocolClient`) |
| NAT Traversal | hole punching | RTTI-confirmed linked (`NATTraversalClient`, `NATTraversalEngine`, `NATTraversalRelayClient`) |
| MatchMaking / MatchMakingExt | gathering lifecycle, participation, session URLs — via the common manager | RTTI-confirmed linked (`MatchMakingClient`, `MatchMakingProtocolExtClient`) |
| MatchmakeExtension | create / browse / join / auto-matchmake | RTTI-confirmed linked (`MatchmakeExtensionClient`) |

### MatchMaking library version

Left at the default (3.7.1, no override) in `globals/config.go` — see
RECON.md for why. If a real console capture ever shows
`CreateMatchmakeSession` decoding garbled or `SessionKey` reading out of
bounds (106-0105), add a `MatchMakingMajor/Minor/Patch` override, the same
mechanism used by trine2-wiiu-nex / mario-sonic-sochi-2014-nex.

`CleanupMatchmakeSessionSearchCriterias` / `CleanupSearchMatchmakeSession`
are set to no-ops (they otherwise hard-fail `AutoMatchmake*_Postpone` with
`Core::NotImplemented`).

### Not yet exercised

Login and the secure-connection handshake are console-confirmed (see
RECON.md's debugging session). Ranking and the matchmaking flow itself
(creating/finding/joining a session) have not — every callback there is
still the stock `nex-protocols-common-go` implementation, with no
title-specific decoder patch, because no capture has yet shown one is
needed.

## Not implemented

| | Why |
|---|---|
| DataStore | not linked into the executable at all — no DDL, no protocol, no client symbols (confirmed absent by RTTI scan). |
| Friends | not linked — confirmed absent by RTTI scan. |
| Account Management | `AccountManagementClient` RTTI symbol is linked, but the common libraries have no server implementation and the NNID account already exists before NEX is reached. Returns `NotImplemented` (logged) if ever called. |
| Miiverse / `nn::olv` | not linked — confirmed absent by RTTI scan and by `meta.xml`'s `olv_accesskey=0`. |
