// Package nex is the Art of Balance (Wii U, Shin'en Multimedia) NEX server.
//
// Two PRUDP endpoints run as separate goroutines: an authentication server
// (Ticket Granting only) and a secure server (Secure Connection, Utility,
// Ranking, NAT Traversal and the three matchmaking protocols). The game
// links no DataStore -- see RECON.md and PROTOCOL_COVERAGE.md.
package nex

import (
	"fmt"
	"os"
	"strconv"

	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/constants"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_ticket_granting "github.com/PretendoNetwork/nex-protocols-common-go/v2/ticket-granting"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting"

	"github.com/Protarium-Network/art-of-balance-nex/globals"
)

func StartAuthenticationServer() {
	globals.AuthenticationServer = nex.NewPRUDPServer()

	// PRUDPv1 CONNECT-ACK signature scheme. 2026-09-13: flipping this to true
	// (legacy/empty) did NOT fix a real console's 106-0502 -- a packet capture
	// on the deployed server, decoded by hand against nex-go's PRUDPv1 header
	// format, showed our CONNECT-ACK sending an all-zero connectionSignature
	// (confirming the legacy path was active) while the client's own CONNECT
	// carries a real non-zero connectionSignature -- the opposite of what the
	// legacy/Trine-2-style scheme expects. That attempt also happened to still
	// have the wrong access key, so it wasn't a clean test. Reverted to false
	// (modern), matching hyrule-warriors-nex (3.8.13) / xenoblade-chronicles-x-nex
	// (3.5.16), now paired with the console-confirmed-correct access key
	// (96900116) for a real test of this setting alone.
	globals.AuthenticationServer.PRUDPV1Settings.LegacyConnectionSignature = false

	globals.AuthenticationEndpoint = nex.NewPRUDPEndPoint(1)
	globals.AuthenticationEndpoint.ServerAccount = globals.AuthenticationServerAccount
	globals.AuthenticationEndpoint.AccountDetailsByPID = globals.AccountDetailsByPID
	globals.AuthenticationEndpoint.AccountDetailsByUsername = globals.AccountDetailsByUsername
	globals.AuthenticationServer.BindPRUDPEndPoint(globals.AuthenticationEndpoint)

	globals.AuthenticationServer.LibraryVersions.SetDefault(nex.NewLibraryVersion(globals.NEXMajor, globals.NEXMinor, globals.NEXPatch))
	globals.AuthenticationServer.AccessKey = globals.AccessKey
	// Same pattern-matched reasoning as LegacyConnectionSignature above: at
	// NEX 3.5+ titles write the structure-header version byte. Flip both
	// endpoints to false together if a real LoginEx capture ever shows
	// "Structure content length longer than data size".
	globals.AuthenticationServer.ByteStreamSettings.UseStructureHeader = true

	globals.AuthenticationEndpoint.OnData(func(packet nex.PacketInterface) {
		request := packet.RMCMessage()
		if request == nil {
			return
		}
		pid := uint64(packet.Sender().PID())
		fmt.Printf("[ArtOfBalance Auth] PID=%d protocol=0x%02X method=0x%02X\n", pid, request.ProtocolID, request.MethodID)
		// One-off wire-format diagnostic: dump the raw RMC parameter bytes for
		// LoginEx (0x0A/0x02) so the AuthenticationInfo layout can be checked
		// by hand against a real capture instead of guessed.
		if request.ProtocolID == 0x0A && request.MethodID == 0x02 {
			fmt.Printf("[ArtOfBalance Auth] RAW LoginEx params (%d bytes): %x\n", len(request.Parameters), request.Parameters)
		}
	})

	registerAuthenticationServerProtocols()

	port, _ := strconv.Atoi(os.Getenv("PN_ARTOFBALANCE_AUTH_PORT"))
	globals.Logger.Successf("[ArtOfBalance] Authentication server listening on UDP %d", port)
	globals.AuthenticationServer.Listen(port)
}

func registerAuthenticationServerProtocols() {
	ticketGrantingProtocol := ticket_granting.NewProtocol()
	globals.AuthenticationEndpoint.RegisterServiceProtocol(ticketGrantingProtocol)
	commonTicketGrantingProtocol := common_ticket_granting.NewCommonProtocol(ticketGrantingProtocol)

	securePort, _ := strconv.Atoi(os.Getenv("PN_ARTOFBALANCE_SECURE_PORT"))

	// Must stay short (~15 chars): the retail binary truncates this field into
	// a small fixed-size buffer, so use a bare host, not a subdomain.
	secureHost := os.Getenv("PN_ARTOFBALANCE_SECURE_HOST")
	if secureHost == "" {
		secureHost = "localhost"
	}

	secureStationURL := types.NewStationURL("")
	secureStationURL.SetURLType(constants.StationURLPRUDPS)
	secureStationURL.SetAddress(secureHost)
	secureStationURL.SetPortNumber(uint16(securePort))
	secureStationURL.SetConnectionID(1)
	secureStationURL.SetPrincipalID(types.NewPID(2))
	secureStationURL.SetStreamID(1)
	secureStationURL.SetStreamType(constants.StreamTypeRVSecure)
	secureStationURL.SetType(uint8(constants.StationURLFlagPublic))

	commonTicketGrantingProtocol.ValidateLoginData = globals.ValidateLoginData
	commonTicketGrantingProtocol.SecureStationURL = secureStationURL
	commonTicketGrantingProtocol.BuildName = types.NewString("")
	commonTicketGrantingProtocol.SecureServerAccount = globals.SecureServerAccount
}
