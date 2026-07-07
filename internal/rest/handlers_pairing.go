package rest

import (
	"encoding/json"
	"net/http"

	"github.com/magicbox/core/internal/p2p"
)

func (s *Server) handleGeneratePairingCode(w http.ResponseWriter, r *http.Request) {
	otp, err := p2p.GenerateOTP()
	if err != nil {
		http.Error(w, "Failed to generate pairing OTP: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"pairing_code":    otp,
		"peer_id":         s.p2pService.HostID(),
		"relay_multiaddr": s.p2pService.GetRelayMultiaddr(),
	})
}
