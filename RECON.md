# Art of Balance (Wii U) — online-stack recon

This title has **no entry in kinnay.github.io's public Wii U NEX database
and no Pretendo Network repo** — unlike every other server in this org, all
of the config below started from **static analysis of the retail executable**
(`project.rpx`) plus a local community game-info cache (`wiiu.json`) and the
dump's own `meta.xml`, then was corrected/confirmed against a real Wii U
console and packet captures on 2026-09-13 (see "Console debugging session"
at the end of this document).

## The executable

| | |
|---|---|
| File | `project.rpx`, 2,419,456 bytes, RPX (ELF32 BE PowerPC, Cafe OS), zlib section compression |
| Build path (string in `.rodata`) | `C:\shinen\aob_wiiu\cafe\project\Release\project.rpx` |
| Developer | Shin'en Multimedia (confirmed by the build path and `meta.xml`'s `publisher_en` field) |
| RPL sections | 25 of 34 sections carry the Wii U compression flag (`sh_flags & 0x08000000`); each is a 4-byte big-endian decompressed-size prefix followed by a raw zlib stream — not a standard zlib-in-section layout, decompressed manually (also decompressable with `wiiurpxtool -d`) |

Decompressed debug strings confirm the NEX SDK build:

```
[SDK+Nintendo:NEX_3_7_1]
[SDK+Nintendo:NEX_MM_3_7_1]
[SDK+Nintendo:NEX_RK_3_7_1]
[SDK+Nintendo:NEX_UT_3_7_1]
```

i.e. **NEX 3.7.1**, with the MatchMaking (`MM`), Ranking (`RK`) and Utility
(`UT`) sub-libraries all at the same version.

## Linked protocols (RTTI / typeinfo evidence)

RTTI/typeinfo symbol names recovered from the decompressed `.rodata` /
`.symtab` confirm which `nn::nex::*` client classes are actually compiled
in:

| Protocol | Confirmed by |
|---|---|
| Ranking | `RankingClient`, `RankingProtocolClient`, `RankingOrderParam`, … |
| MatchMaking + MatchmakeExtension | `MatchMakingClient`, `MatchMakingProtocolExtClient`, `MatchmakeExtensionClient` |
| NAT Traversal | `NATTraversalClient`, `NATTraversalEngine`, `NATTraversalRelayClient` |
| Secure Connection | `SecureConnectionClient` |
| Account Management | `AccountManagementClient` |
| Utility | `UtilityClient`, `UtilitySubsystem` |

**Confirmed NOT linked:** DataStore (no `nn::nex::DataStore*` symbol anywhere),
Friends, a distinct Messaging service (only the generic internal
`nn::nex::Message`/`MessageBundle` transport classes exist, shared by every
protocol), Miiverse/`nn::olv` (also confirmed by `meta.xml`'s
`olv_accesskey` field being `0`). No Quazal PC-backend debug strings, IPs,
`nasc`, `.nintendowifi.net` or GameSpy strings were found either — this is a
Wii U-only build with no dead PC-era backend leftovers to rule out.

## Authentication path

`nn::act::AcquireNexServiceToken` is present as the CodeWarrior-mangled
symbol `AcquireNexServiceToken__Q2_2nn3actFP26ACTNexAuthenticationResultUi`,
plus the usual log format string:

```
GameID:%08x Success:%d AcquireNexServiceTokenResult:%08x
```

— the standard Wii U NASC → nex_token → PRUDP flow every other title in this
project uses.

## Access key and game server ID: NOT in the executable

Unlike every other title in this project, the access key and game server ID
are **not literal strings anywhere in the decompressed RPX**. Two relevant
findings explain why:

- `"gameServerID"` only appears inside a NEX DDL/RMC schema (a NATTraversal
  report field, type `uint32`) — a protocol field name, not a config value.
- `"CNex.gameId"` / `"CNex.accessKey"` appear as **scriptable property
  bindings** in Shin'en's internal engine VM (`vm.cpp` referenced
  repeatedly). A log string confirms the concept: `" Make sure using the
  Game Server ID+Access Key from OMAS website (and not access key by
  Nintendo game code)"`. Shin'en's engine reads these two values from
  external script/config data at runtime rather than compiling them in as
  constants — the same engine architecture is shared across their other Wii
  U eShop titles.

### Access key: CONFIRMED `96900116` via bruteforce against a real packet

Initial pipeline (ROM-scan mode, already used successfully once in this
project for a different title, "Lost Reavers"):

1. `wiiurpxtool -d project.rpx project_decompressed.elf` — full RPL→ELF
   decompression (a clean superset of the manual section-by-section
   decompression above).
2. `access-key-extractor -rom=project_decompressed.elf` (PretendoNetwork's
   tool, scans for 8-lowercase-hex-character strings in UTF-8, UTF-16BE and
   UTF-16LE — the plain-ASCII scan above found nothing, so the value is
   presumably UTF-16-encoded somewhere in engine string tables) returned 5
   unverified candidates (`d4f027b8`, `20070622`, `0005001b`, `10052000`,
   `1005c000`). **None of them validated** once a real packet was available
   to check against (see below) — the correct key was never in this list.

**Resolved 2026-09-13** once a real Wii U attempted to connect to the
deployed server: a `tcpdump` capture on the server isolated the console's
actual PRUDPv1 SYN packet, and `access-key-extractor -bruteforce -packet=<the
real SYN's hex payload>` swept all 2^32 candidates against it. PRUDPv1 uses
the access key directly in its packet signature, so exactly one candidate
can ever validate against a real packet (unlike PRUDPv0's byte-sum checksum,
which has many collisions) — this is a **definitive, cryptographically
verified** result, not a guess:

```
96900116
```

### Game server ID: CONFIRMED `10135000`, single ID shared across regions

The initial hypothesis (a pattern confirmed elsewhere in this project —
Trine 2: title ID `...10112200` → game server ID `10112200`, exact match —
that the game server ID equals the lower 32 bits of the 16-hex-digit title
ID) predicted three distinct region IDs. Title IDs for all three regions
came from a local community game-info database (`wiiu.json`) cross-referenced
against this dump's own `meta.xml` (EUR title ID matches exactly):

| Region | Product code | Title ID | Publisher |
|---|---|---|---|
| USA | WUP-P-WABE | `0005000010135000` | Shin'en |
| EUR | WUP-P-WABP | `0005000010149400` | Shin'en (this dump: RPX + `meta.xml` on disk) |
| JPN | WUP-P-WABJ | `000500001017cb00` | Arc System Works |

`meta.xml` for the EUR dump confirms: `title_id=0005000010149400`,
`product_code=WUP-P-WABP`, `region=00000004` (EUR-only build),
`publisher_en=Shin'en`.

**Game server ID: CONFIRMED `10135000` 2026-09-13**, directly observed in a
real EUR console's `nex_token` request (packet capture on the deployed token
service) — **not** `10149400` as the "lower 32 bits of the title ID" pattern
(confirmed exact on a sibling project, Trine 2) would have predicted for the
EUR release. `10135000` is instead the USA release's title-ID-derived value,
reused as a single game server ID shared across all regions — the same
"one game server, multiple title IDs" pattern already documented on Trine 2.
USA/JPN builds are assumed (not directly observed) to request this same ID.
PRUDP itself does not carry a game-server-ID concept — it only matters to
whatever account/token server maps a requested game_server_id to
host:port:access_key; this repo's NEX server needs only ONE access key and
ONE NEX version to run.

## What the server implements

Auth endpoint: **Ticket Granting**.
Secure endpoint: **Secure Connection**, **Utility**, **Ranking**, **NAT
Traversal**, **MatchMaking**, **MatchMakingExt**, **MatchmakeExtension**.

Not implemented: DataStore, Friends, Account Management (linked by the RPX,
but Pretendo's common libraries have no server implementation and the
account already exists by the time the game reaches NEX).

## Console debugging session (2026-09-13)

A real Wii U was pointed at the deployed server. Each fix below cleared one
observed failure, in order:

1. **106-0502 (`Transport::ConnectionFailure`)**, immediately, with the
   originally-guessed access key (`d4f027b8`) and `LegacyConnectionSignature
   = false`. Flipping to `true` (legacy/empty) did **not** fix it — still
   106-0502, identical retransmit pattern. This attempt still had the wrong
   access key, so it wasn't a clean test of the signature setting alone.
2. Captured the console's real PRUDPv1 SYN packet with `tcpdump` on the
   server and bruteforced the access key against it (see above) →
   `96900116`. Redeployed with the correct key, **still** `LegacyConnectionSignature
   = true` — 106-0502 persisted. Hand-decoding the capture against nex-go's
   PRUDPv1 header format (magic/version/lengths/ports/type-flags/session/
   sequence/16-byte signature/TLV options) showed our CONNECT-ACK sending an
   **all-zero `connectionSignature`** (confirming the legacy path was
   active) while the client's own CONNECT carried a real non-zero one — the
   opposite of what the legacy scheme expects. This was the first clean
   signal that `false` (modern), not `true`, was the right setting for this
   title.
3. Reverted to `LegacyConnectionSignature = false` (now paired with the
   correct access key for the first time) → **new error, 106-0103**,
   replacing 106-0502 entirely. This confirmed the PRUDP transport layer
   (access key + modern connection-signature scheme) is correct: `LoginEx`,
   `RequestTicket` and `SecureConnection::Register` all succeeded per server
   logs.
4. Server logs pinpointed 106-0103's cause exactly:
   `Utility::AcquireNexUniqueID missing GenerateNEXUniqueID!` — the Utility
   protocol was registered but its `GenerateNEXUniqueID` callback (an
   in-memory atomic counter is sufficient; these IDs only need to be unique
   for the process lifetime) was never set. None of this project's other
   servers needed this callback, because none of their games call
   `AcquireNexUniqueID` — Art of Balance does, during login. Fixed in
   `nex/secure.go`.

**Confirmed by this session:** access key `96900116`, game server ID
`10135000`, `LegacyConnectionSignature = false`, `AcquireNexUniqueID` needs a
real `GenerateNEXUniqueID` implementation. **Still open:** `UseStructureHeader`
was left at `true` throughout (never implicated in either error) and the
MatchMaking wire version was never exercised far enough to test — both
remain pattern-matched guesses, not console-proven. Whether a full
matchmaking session (finding/joining another player) works end to end is
also not yet confirmed.
