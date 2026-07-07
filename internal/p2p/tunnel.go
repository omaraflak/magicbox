package p2p

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
)

type HandshakeRequest struct {
	OTP   string `json:"otp"`
	Token string `json:"token"`
}

type HandshakeResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Error   string `json:"error,omitempty"`
}

type AuthHeader struct {
	Token string `json:"token"`
}

func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func sendHandshakeResponse(stream network.Stream, resp HandshakeResponse) error {
	bytes, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	
	lenHeader := make([]byte, 4)
	binary.BigEndian.PutUint32(lenHeader, uint32(len(bytes)))
	
	if _, err := stream.Write(lenHeader); err != nil {
		return err
	}
	if _, err := stream.Write(bytes); err != nil {
		return err
	}
	return nil
}

// StartTunnelHandler registers the /magicbox/tunnel/1.0.0 protocol stream handler on the libp2p service.
func StartTunnelHandler(p2pService *Libp2pService, database *db.DB, localURL string, logger *logging.Logger) {
	p2pService.SetStreamHandler("/magicbox/tunnel/1.0.0", func(stream network.Stream) {
		defer stream.Close()

		remotePeer := stream.Conn().RemotePeer().String()
		logger.Info("Incoming P2P tunnel stream connection", logging.F("remote", remotePeer))

		// 1. Read handshake request length (4 bytes)
		lenHeader := make([]byte, 4)
		if _, err := io.ReadFull(stream, lenHeader); err != nil {
			logger.Error("P2P tunnel handshake: failed to read length header", logging.F("error", err.Error()))
			return
		}
		length := binary.BigEndian.Uint32(lenHeader)

		// 2. Read handshake JSON body
		bodyBytes := make([]byte, length)
		if _, err := io.ReadFull(stream, bodyBytes); err != nil {
			logger.Error("P2P tunnel handshake: failed to read JSON body", logging.F("error", err.Error()))
			return
		}

		var req HandshakeRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			logger.Error("P2P tunnel handshake: invalid JSON", logging.F("error", err.Error()))
			return
		}

		// 3. Authenticate OTP pairing or client token
		authorized := false
		var generatedToken string
		var err error

		if req.Token != "" {
			authorized, err = database.IsValidP2PPairingToken(req.Token)
			if err != nil {
				logger.Error("P2P tunnel DB verification failed", logging.F("error", err.Error()))
				_ = sendHandshakeResponse(stream, HandshakeResponse{Success: false, Error: "Internal server error"})
				return
			}
		} else if req.OTP != "" {
			if VerifyAndConsumeOTP(req.OTP) {
				generatedToken, err = generateSecureToken()
				if err != nil {
					logger.Error("P2P tunnel token generation failed", logging.F("error", err.Error()))
					_ = sendHandshakeResponse(stream, HandshakeResponse{Success: false, Error: "Internal server error"})
					return
				}

				if err := database.InsertP2PPairingToken(generatedToken); err != nil {
					logger.Error("P2P tunnel failed to save token to database", logging.F("error", err.Error()))
					_ = sendHandshakeResponse(stream, HandshakeResponse{Success: false, Error: "Internal server error"})
					return
				}
				authorized = true
			}
		}

		if !authorized {
			logger.Warn("P2P tunnel handshake denied: unauthorized pairing request", logging.F("remote", remotePeer))
			_ = sendHandshakeResponse(stream, HandshakeResponse{Success: false, Error: "Unauthorized"})
			return
		}

		// 4. Send successful handshake response
		if err := sendHandshakeResponse(stream, HandshakeResponse{Success: true, Token: generatedToken}); err != nil {
			logger.Error("P2P tunnel failed to send handshake response", logging.F("error", err.Error()))
			return
		}

		logger.Info("P2P tunnel handshake accepted", logging.F("remote", remotePeer))

		// 5. Read request authorization header length (4 bytes)
		if _, err := io.ReadFull(stream, lenHeader); err != nil {
			logger.Error("P2P tunnel: failed to read request auth header length", logging.F("error", err.Error()))
			return
		}
		length = binary.BigEndian.Uint32(lenHeader)

		// 6. Read request authorization header body
		bodyBytes = make([]byte, length)
		if _, err := io.ReadFull(stream, bodyBytes); err != nil {
			logger.Error("P2P tunnel: failed to read request auth header body", logging.F("error", err.Error()))
			return
		}

		var auth AuthHeader
		if err := json.Unmarshal(bodyBytes, &auth); err != nil {
			logger.Error("P2P tunnel: invalid auth header JSON", logging.F("error", err.Error()))
			return
		}

		// Verify token for the request
		tokenToVerify := auth.Token
		if tokenToVerify == "" {
			tokenToVerify = req.Token
		}
		authorized, err = database.IsValidP2PPairingToken(tokenToVerify)
		if err != nil || !authorized {
			logger.Warn("P2P tunnel request blocked: invalid authorization token", logging.F("remote", remotePeer))
			return
		}

		// 7. Parse raw HTTP request from P2P stream
		reader := bufio.NewReader(stream)
		httpReq, err := http.ReadRequest(reader)
		if err != nil {
			logger.Error("P2P tunnel: failed to parse HTTP request", logging.F("error", err.Error()))
			return
		}

		// 8. Proxy request locally to local HTTP server
		targetURL, err := url.Parse(localURL + httpReq.URL.Path)
		if err != nil {
			logger.Error("P2P tunnel: invalid local URL path parsing", logging.F("error", err.Error()))
			return
		}
		targetURL.RawQuery = httpReq.URL.RawQuery

		localReq, err := http.NewRequest(httpReq.Method, targetURL.String(), httpReq.Body)
		if err != nil {
			logger.Error("P2P tunnel: failed to create local proxy request", logging.F("error", err.Error()))
			return
		}

		// Transfer request headers
		for key, values := range httpReq.Header {
			for _, val := range values {
				localReq.Header.Add(key, val)
			}
		}

		// Perform local proxy dispatch
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // return redirect responses to client
			},
		}
		resp, err := client.Do(localReq)
		if err != nil {
			logger.Error("P2P tunnel: local target server connection refused", logging.F("error", err.Error()))
			errResp := &http.Response{
				StatusCode: http.StatusBadGateway,
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Body:       io.NopCloser(strings.NewReader("Local Magicbox host is unreachable: " + err.Error())),
			}
			_ = errResp.Write(stream)
			return
		}
		defer resp.Body.Close()

		// 9. Write raw response wire bytes back into P2P stream
		if err := resp.Write(stream); err != nil {
			logger.Error("P2P tunnel: failed to write response wire bytes back to stream", logging.F("error", err.Error()))
		}
	})
}
