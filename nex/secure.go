package nex

import (
	"fmt"
	"os"
	"strconv"
	"sync/atomic"

	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	common_matchmaking "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making"
	common_matchmaking_ext "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making-ext"
	common_matchmake_extension "github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension"
	common_nat_traversal "github.com/PretendoNetwork/nex-protocols-common-go/v2/nat-traversal"
	common_ranking "github.com/PretendoNetwork/nex-protocols-common-go/v2/ranking"
	common_secure "github.com/PretendoNetwork/nex-protocols-common-go/v2/secure-connection"
	common_utility "github.com/PretendoNetwork/nex-protocols-common-go/v2/utility"
	matchmaking "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	matchmaking_ext "github.com/PretendoNetwork/nex-protocols-go/v2/match-making-ext"
	matchmaking_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/v2/nat-traversal"
	ranking "github.com/PretendoNetwork/nex-protocols-go/v2/ranking"
	secure "github.com/PretendoNetwork/nex-protocols-go/v2/secure-connection"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"

	"github.com/Protarium-Network/art-of-balance-nex/database"
	"github.com/Protarium-Network/art-of-balance-nex/globals"
)

// nexUniqueIDCounter backs Utility::AcquireNexUniqueID -- only needs to be
// unique for the life of the process, not persisted.
var nexUniqueIDCounter atomic.Uint64

func StartSecureServer() {
	globals.SecureServer = nex.NewPRUDPServer()

	// See authentication.go: reverted to false (modern) 2026-09-13 after a
	// packet capture showed the legacy/empty scheme wasn't what the client
	// wanted either, in a test that also had the wrong access key.
	globals.SecureServer.PRUDPV1Settings.LegacyConnectionSignature = false

	globals.SecureEndpoint = nex.NewPRUDPEndPoint(1)
	globals.SecureEndpoint.IsSecureEndPoint = true
	globals.SecureEndpoint.ServerAccount = globals.SecureServerAccount
	globals.SecureEndpoint.AccountDetailsByPID = globals.AccountDetailsByPID
	globals.SecureEndpoint.AccountDetailsByUsername = globals.AccountDetailsByUsername
	globals.SecureServer.BindPRUDPEndPoint(globals.SecureEndpoint)

	globals.SecureServer.LibraryVersions.SetDefault(nex.NewLibraryVersion(globals.NEXMajor, globals.NEXMinor, globals.NEXPatch))
	// No per-library version override: NEX 3.7.1 is comfortably above the
	// MatchMaking 3.4.0 serialisation break that forced Trine 2 / Sochi 2014
	// to pin MatchMaking to 3.3.0 (those titles are NEX 3.4.x). If a real
	// console capture ever shows CreateMatchmakeSession decoding garbled or
	// SessionKey reading out of bounds, add a MatchMakingMajor/Minor/Patch
	// override here (globals/config.go) and set
	// SecureServer.LibraryVersions.MatchMaking below, same as trine2-wiiu-nex.
	globals.SecureServer.AccessKey = globals.AccessKey
	globals.SecureServer.ByteStreamSettings.UseStructureHeader = true

	globals.SecureEndpoint.OnData(func(packet nex.PacketInterface) {
		request := packet.RMCMessage()
		if request == nil {
			return
		}
		pid := uint64(packet.Sender().PID())
		fmt.Printf("[ArtOfBalance Secure] PID=%d protocol=0x%02X method=0x%02X\n", pid, request.ProtocolID, request.MethodID)
	})

	globals.SecureEndpoint.OnConnectionEnded(func(connection *nex.PRUDPConnection) {
		fmt.Printf("[ArtOfBalance Secure] PID=%d disconnected\n", uint64(connection.PID()))
	})

	// The matchmaking manager needs the secure endpoint and the database.
	globals.MatchmakingManager = common_globals.NewMatchmakingManager(globals.SecureEndpoint, database.Postgres)
	globals.MatchmakingManager.GetUserFriendPIDs = globals.GetUserFriendPIDs

	registerSecureServerProtocols()

	port, _ := strconv.Atoi(os.Getenv("PN_ARTOFBALANCE_SECURE_PORT"))
	globals.Logger.Successf("[ArtOfBalance] Secure server listening on UDP %d", port)
	globals.SecureServer.Listen(port)
}

// registerSecureServerProtocols wires up the secure-connection handshake,
// Utility, Ranking (leaderboards) and matchmaking (NAT Traversal +
// MatchMaking + MatchMakingExt + MatchmakeExtension) -- the exact set the
// retail RPX proves linked via RTTI/typeinfo symbol enumeration (see
// RECON.md). No DataStore, no Friends: the game links neither.
func registerSecureServerProtocols() {
	// Secure Connection (0x0B) -- Register / RegisterEx: the console tells us
	// the station URLs other party members should use to reach it.
	secureProtocol := secure.NewProtocol()
	globals.SecureEndpoint.RegisterServiceProtocol(secureProtocol)
	secureCommon := common_secure.NewCommonProtocol(secureProtocol)
	secureCommon.EnableInsecureRegister()
	secureCommon.CreateReportDBRecord = func(pid types.PID, reportID types.UInt32, reportData types.QBuffer) error {
		globals.Logger.Warningf("[ArtOfBalance] Player report from PID %d (report ID %d, %d bytes) discarded: this server does not store reports",
			pid, reportID, len(reportData))
		return nil
	}

	// Utility (0x6E) -- NEX unique IDs. Confirmed linked (UtilityClient /
	// UtilitySubsystem RTTI symbols in the RPX), and console-confirmed
	// 2026-09-13 that the game actually calls AcquireNexUniqueID during
	// login (missing GenerateNEXUniqueID caused a 106-0103 disconnect).
	// A simple in-memory atomic counter is enough: these IDs only need to be
	// unique for the life of the server process.
	utilityProtocol := utility.NewProtocol()
	globals.SecureEndpoint.RegisterServiceProtocol(utilityProtocol)
	utilityCommon := common_utility.NewCommonProtocol(utilityProtocol)
	utilityCommon.GenerateNEXUniqueID = func() uint64 {
		return nexUniqueIDCounter.Add(1)
	}

	// Ranking (0x70) -- leaderboards. Confirmed linked (RankingClient /
	// RankingProtocolClient / RankingOrderParam RTTI symbols in the RPX).
	rankingProtocol := ranking.NewProtocol()
	globals.SecureEndpoint.RegisterServiceProtocol(rankingProtocol)
	rankingCommon := common_ranking.NewCommonProtocol(rankingProtocol)
	rankingCommon.GetRankingsAndCountByCategoryAndRankingOrderParam = database.ArtOfBalanceGetRankingsAndCountByCategoryAndRankingOrderParam
	rankingCommon.GetRankingsByMode = database.ArtOfBalanceGetRankings
	rankingCommon.GetCommonData = database.ArtOfBalanceGetCommonData
	rankingCommon.UploadCommonData = database.ArtOfBalanceUploadCommonData
	rankingCommon.InsertRankingByPIDAndRankingScoreData = database.ArtOfBalanceInsertRankingByPIDAndRankingScoreData

	// NAT Traversal (0x03) -- hole punching between consoles in a
	// matchmaking session.
	natTraversalProtocol := nat_traversal.NewProtocol()
	globals.SecureEndpoint.RegisterServiceProtocol(natTraversalProtocol)
	common_nat_traversal.NewCommonProtocol(natTraversalProtocol)

	// MatchMaking (0x15) + MatchMakingExt (0x32) -- gathering lifecycle,
	// participation, session URLs, host migration. Confirmed linked
	// (MatchMakingClient / MatchMakingProtocolExtClient RTTI symbols).
	matchMakingProtocol := matchmaking.NewProtocol()
	globals.SecureEndpoint.RegisterServiceProtocol(matchMakingProtocol)
	common_matchmaking.NewCommonProtocol(matchMakingProtocol).SetManager(globals.MatchmakingManager)

	matchMakingExtProtocol := matchmaking_ext.NewProtocol()
	globals.SecureEndpoint.RegisterServiceProtocol(matchMakingExtProtocol)
	common_matchmaking_ext.NewCommonProtocol(matchMakingExtProtocol).SetManager(globals.MatchmakingManager)

	// MatchmakeExtension (0x6D) -- create / browse / join / auto-matchmake a
	// session. Confirmed linked (MatchmakeExtensionClient RTTI symbol).
	matchmakeExtensionProtocol := matchmake_extension.NewProtocol()
	globals.SecureEndpoint.RegisterServiceProtocol(matchmakeExtensionProtocol)
	commonMatchmakeExtensionProtocol := common_matchmake_extension.NewCommonProtocol(matchmakeExtensionProtocol)
	commonMatchmakeExtensionProtocol.SetManager(globals.MatchmakingManager)

	// AutoMatchmakePostpone / AutoMatchmakeWithSearchCriteriaPostpone
	// hard-fail with Core::NotImplemented if these two cleanup callbacks are
	// nil, even though there is nothing to clean up. Pretendo's own reference
	// server sets the same pair to no-ops.
	commonMatchmakeExtensionProtocol.CleanupMatchmakeSessionSearchCriterias = func(searchCriterias types.List[matchmaking_types.MatchmakeSessionSearchCriteria]) {}
	commonMatchmakeExtensionProtocol.CleanupSearchMatchmakeSession = func(matchmakeSession *matchmaking_types.MatchmakeSession) {}

	// No Art of Balance matchmaking capture exists, so log the decoded
	// request server side for the calls a host makes.
	commonMatchmakeExtensionProtocol.OnAfterCreateMatchmakeSession = func(packet nex.PacketInterface, anyGathering matchmaking_types.GatheringHolder, message types.String, participationCount types.UInt16) {
		if ms, ok := anyGathering.Object.(matchmaking_types.MatchmakeSession); ok {
			fmt.Printf("[MM] CreateMatchmakeSession OK: gameMode=%d attribs=%v min=%d max=%d open=%v sysType=%d appBuf=%dB msg=%q pc=%d\n",
				uint32(ms.GameMode), ms.Attributes, uint16(ms.Gathering.MinimumParticipants), uint16(ms.Gathering.MaximumParticipants),
				bool(ms.OpenParticipation), uint32(ms.MatchmakeSystemType), len(ms.ApplicationBuffer), string(message), uint16(participationCount))
		}
	}
	commonMatchmakeExtensionProtocol.OnAfterOpenParticipation = func(packet nex.PacketInterface, gid types.UInt32) {
		fmt.Printf("[MM] OpenParticipation OK: gid=%d\n", uint32(gid))
	}
}
