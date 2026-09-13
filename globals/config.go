package globals

// NEX configuration for "Art of Balance" (Wii U, Shin'en Multimedia, 2013).
//
// Confirmed by static analysis of the decompressed retail RPX (RTTI /
// typeinfo symbol enumeration + debug strings) - this title has no
// kinnay.github.io or Pretendo Network entry, so there was no public
// database to cross-reference. See RECON.md for the full trail.
const (
	// GameServerID is CONFIRMED 2026-09-13 from a real EUR console's actual
	// nex_token request (packet capture on the deployed VPS token service):
	// the EUR build (title ID 0005000010149400) requests game server ID
	// 10135000 -- NOT 10149400. This disproves the earlier "game server ID =
	// lower 32 bits of the title ID" hypothesis for this title: 10135000 is
	// instead the USA release's title-ID-derived value (title
	// 0005000010135000), reused as a single shared game server ID across all
	// regions -- the same "one game server, multiple title IDs" pattern
	// already seen on Trine 2. USA/JPN builds are assumed (not yet directly
	// observed) to request this same ID.
	GameServerID = "10135000"

	// AccessKey CONFIRMED 2026-09-13 by bruteforcing all 2^32 candidates
	// against a real PRUDPv1 SYN packet captured from an actual Wii U console
	// (tcpdump on the deployed VPS while the user tried connecting). Exactly
	// one candidate validated -- PRUDPv1 uses the access key directly in its
	// signature, so a match against a real packet is definitive, unlike the
	// ROM-scan's 5 unverified guesses this replaced (d4f027b8 and others,
	// none of which validated against this packet).
	AccessKey = "96900116"

	// PRUDP library version reported by both endpoints. Confirmed via RPX
	// debug strings: [SDK+Nintendo:NEX_3_7_1].
	NEXMajor = 3
	NEXMinor = 7
	NEXPatch = 1
)
