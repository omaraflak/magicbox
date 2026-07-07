package p2p

import (
	"crypto/rand"
	"math/big"
	"sync"
	"time"
)

var (
	pairingCodes   = make(map[string]time.Time)
	pairingCodesMu sync.Mutex
)

// GenerateOTP generates a random 6-digit OTP code, registers it with a 5-minute expiry, and returns it.
func GenerateOTP() (string, error) {
	otp := ""
	for i := 0; i < 6; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		otp += num.String()
	}

	pairingCodesMu.Lock()
	pairingCodes[otp] = time.Now().Add(5 * time.Minute)
	pairingCodesMu.Unlock()

	return otp, nil
}

// VerifyAndConsumeOTP checks if the OTP is valid and not expired, consuming it in the process.
func VerifyAndConsumeOTP(otp string) bool {
	pairingCodesMu.Lock()
	defer pairingCodesMu.Unlock()

	expiry, exists := pairingCodes[otp]
	if !exists {
		return false
	}
	delete(pairingCodes, otp) // single-use OTP

	return time.Now().Before(expiry)
}
