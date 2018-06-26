package proxy

import (
	"encoding/hex"
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/webchain-network/cryptonight"
)

var hasher = cryptonight.New()

var (
	big0 = big.NewInt(0)
	maxUint256  = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
)

func (s *ProxyServer) checkHash(hash *big.Int, difficulty *big.Int) bool {
	/* Cannot happen if block header diff is validated prior to PoW, but can
		 happen if PoW is checked first due to parallel PoW checking.
	*/
	if difficulty.Cmp(big0) == 0 {
		return false
	}

	target := new(big.Int).Div(maxUint256, difficulty)
	return hash.Cmp(target) <= 0
}

func (s *ProxyServer) processShare(login, id, ip string, t *BlockTemplate, params []string, shareDiff int64) (bool, bool) {
	nonceHex := params[0]
	hashNoNonce := params[1]
	nonce, _ := strconv.ParseUint(strings.Replace(nonceHex, "0x", "", -1), 16, 64)

	h, ok := t.headers[hashNoNonce]
	if !ok {
		log.Printf("Stale share from %v@%v", login, ip)
		return false, false
	}

	header, err := hex.DecodeString(t.Seed)

	hash := hasher.CalcHash(header, nonce)

	if err != nil || !s.checkHash(hash, big.NewInt(shareDiff)) {
		return false, false
	}

	if s.checkHash(hash, h.diff) {
		ok, err := s.rpc().SubmitBlock(params)
		if err != nil {
			log.Printf("Block submission failure at height %v for %v: %v", h.height, t.Header, err)
		} else if !ok {
			log.Printf("Block rejected at height %v for %v", h.height, t.Header)
			return false, false
		} else {
			s.fetchBlockTemplate()
			exist, err := s.backend.WriteBlock(login, id, params, shareDiff, h.diff.Int64(), h.height, s.hashrateExpiration)
			if exist {
				return true, false
			}
			if err != nil {
				log.Println("Failed to insert block candidate into backend:", err)
			} else {
				log.Printf("Inserted block %v to backend", h.height)
			}
			log.Printf("Block found by miner %v@%v at height %d", login, ip, h.height)
		}
	} else {
		exist, err := s.backend.WriteShare(login, id, params, shareDiff, h.height, s.hashrateExpiration)
		if exist {
			return true, false
		}
		if err != nil {
			log.Println("Failed to insert share data into backend:", err)
		}
	}
	return false, true
}
