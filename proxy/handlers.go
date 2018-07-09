package proxy

import (
	"log"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/webchain-network/webchain-pool/rpc"
	"github.com/webchain-network/webchain-pool/util"
)

// Allow only lowercase hexadecimal with 0x prefix
var noncePattern = regexp.MustCompile("^0x[0-9a-f]{16}$")
var hashPattern = regexp.MustCompile("^0x[0-9a-f]{64}$")
var workerPattern = regexp.MustCompile("^[0-9a-zA-Z-_.]{1,192}$")

// Stratum
func (s *ProxyServer) handleLoginRPC(cs *Session, params map[string]string, id string) (bool, *ErrorReply) {
	if len(params) == 0 {
		return false, &ErrorReply{Code: -1, Message: "Invalid params"}
	}

    login := strings.ToLower(params["login"])
	if !util.IsValidHexAddress(login) {
		return false, &ErrorReply{Code: -1, Message: "Invalid login"}
	}
	if !s.policy.ApplyLoginPolicy(login, cs.ip) {
		return false, &ErrorReply{Code: -1, Message: "You are blacklisted"}
	}
	cs.login = login
	cs.diff = s.config.Proxy.Difficulty
	cs.nextDiff = cs.diff
	s.registerSession(cs)
	log.Printf("Stratum miner connected %v@%v", login, cs.ip)
	return true, nil
}

func (s *ProxyServer) handleGetWorkRPC(cs *Session) ([]string, *ErrorReply) {
	t := s.currentBlockTemplate()
	if t == nil || len(t.Header) == 0 || s.isSick() {
		return nil, &ErrorReply{Code: 0, Message: "Work not ready"}
	}
	cs.diff = cs.nextDiff
	return []string{t.Header, t.Seed, util.GetTargetHex(cs.diff)}, nil
}

// Stratum
func (s *ProxyServer) handleTCPSubmitRPC(cs *Session, id string, params []string) (bool, *ErrorReply) {
	s.sessionsMu.RLock()
	_, ok := s.sessions[cs]
	s.sessionsMu.RUnlock()

	if !ok {
		return false, &ErrorReply{Code: 25, Message: "Not subscribed"}
	}
	return s.handleSubmitRPC(cs, cs.login, id, params)
}

func (s *ProxyServer) calcNewDiff(cs *Session) int64 {
	config := &s.config.Proxy.VarDiff

	now := time.Now()

	if cs.lastShareTime.IsZero() {
		cs.lastShareTime = now
		return cs.diff
	}

	sinceLast := now.Sub(cs.lastShareTime)
	cs.lastShareTime = now

	cs.lastShareDurations = append(cs.lastShareDurations, sinceLast)

	if len(cs.lastShareDurations) > 5 {
		cs.lastShareDurations = cs.lastShareDurations[1:]
	}

	var avg float64
	for i := 0; i<len(cs.lastShareDurations); i++ {
		avg += cs.lastShareDurations[i].Seconds()
	}
	avg /= float64(len(cs.lastShareDurations))

	variance := float64(config.VariancePercent) / 100.0 * config.TargetTime
	tMin := config.TargetTime - variance
	tMax := config.TargetTime + variance

	var direction float64
	var newDiff int64

	if avg > tMax && cs.diff > config.MinDiff {
		newDiff = int64(config.TargetTime / avg * float64(cs.diff))
        newDiff = util.Max(newDiff, config.MinDiff)
		direction = -1
	} else if avg < tMin && cs.diff < config.MaxDiff {
		newDiff = int64(config.TargetTime / avg * float64(cs.diff))
		newDiff = util.Min(newDiff, config.MaxDiff)
		direction = 1
	} else {
		return cs.diff
	}

	if math.Abs(float64(newDiff - cs.diff)) / float64(cs.diff) * 100 > float64(config.MaxJump) {
		change := int64(float64(config.MaxJump) / 100 * float64(cs.diff) * direction);
		newDiff = cs.diff + change;
	}
	cs.lastShareDurations = nil
	return newDiff
}

func (s *ProxyServer) handleSubmitRPC(cs *Session, login, id string, params []string) (bool, *ErrorReply) {
	if !workerPattern.MatchString(id) {
		id = "0"
	}
	if len(params) != 3 {
		s.policy.ApplyMalformedPolicy(cs.ip)
		log.Printf("Malformed params from %s@%s %v", login, cs.ip, params)
		return false, &ErrorReply{Code: -1, Message: "Invalid params"}
	}

	/*if !noncePattern.MatchString(params[0]) || !hashPattern.MatchString(params[1]) || !hashPattern.MatchString(params[2])  {
		s.policy.ApplyMalformedPolicy(cs.ip)
		log.Printf("Malformed PoW result from %s@%s %v", login, cs.ip, params)
		return false, &ErrorReply{Code: -1, Message: "Malformed PoW result"}
	}*/
	t := s.currentBlockTemplate()
	exist, validShare := s.processShare(login, id, cs.ip, t, params, cs.diff)
	ok := s.policy.ApplySharePolicy(cs.ip, !exist && validShare)

	if exist {
		log.Printf("Duplicate share from %s@%s %v", login, cs.ip, params)
		return false, &ErrorReply{Code: 22, Message: "Duplicate share"}
	}

	if !validShare {
		log.Printf("Invalid share from %s@%s", login, cs.ip)
		// Bad shares limit reached, return error and close
		if !ok {
			return false, &ErrorReply{Code: 23, Message: "Invalid share"}
		}
		return false, nil
	}
	log.Printf("Valid share from %s@%s", login, cs.ip)

	cs.nextDiff = s.calcNewDiff(cs)

	if !ok {
		return true, &ErrorReply{Code: -1, Message: "High rate of invalid shares"}
	}
	return true, nil
}

func (s *ProxyServer) handleGetBlockByNumberRPC() *rpc.GetBlockReplyPart {
	t := s.currentBlockTemplate()
	var reply *rpc.GetBlockReplyPart
	if t != nil {
		reply = t.GetPendingBlockCache
	}
	return reply
}

func (s *ProxyServer) handleUnknownRPC(cs *Session, m string) *ErrorReply {
	log.Printf("Unknown request method %s from %s", m, cs.ip)
	s.policy.ApplyMalformedPolicy(cs.ip)
	return &ErrorReply{Code: -3, Message: "Method not found"}
}
