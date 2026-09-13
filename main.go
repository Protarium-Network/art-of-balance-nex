// Command art-of-balance-nex is a preservation NEX server for the Wii U
// title Art of Balance (Shin'en Multimedia, 2013).
//
// It runs the two PRUDP servers the game expects: an authentication server
// that hands out Kerberos tickets, and a secure server that does online
// leaderboards (Ranking) and matchmaking (NAT Traversal + MatchMaking +
// MatchMakingExt + MatchmakeExtension). The game links no DataStore, so
// this server implements none.
package main

import (
	"sync"

	"github.com/Protarium-Network/art-of-balance-nex/nex"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		nex.StartAuthenticationServer()
	}()
	go func() {
		defer wg.Done()
		nex.StartSecureServer()
	}()

	wg.Wait()
}
