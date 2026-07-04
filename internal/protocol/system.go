// Package protocol implements Magicbox system-level P2P message handlers.
// Each system:* protocol (key rotation, succession, contact requests, etc.)
// is registered as a named handler on the P2P service, keeping main.go clean
// and making it trivial to add new system protocols.
package protocol

import (
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
)

// System protocol AppIDs for P2P messages.
const (
	AppIDKeyUpdate      = "system:key-update"
	AppIDKeySuccession  = "system:key-succession"
	AppIDContactRequest = "system:contact-request"
	AppIDContactAccept  = "system:contact-accept"
)

// RegisterSystemHandlers registers all system:* P2P message handlers on the given service.
// Each handler is registered by its AppID so the P2P layer dispatches directly
// without a manual switch/if-else chain.
func RegisterSystemHandlers(service p2p.Service, database *db.DB, logger *logging.Logger) {
	service.RegisterHandler(AppIDKeyUpdate, newKeyUpdateHandler(database, logger))
	service.RegisterHandler(AppIDContactRequest, newContactRequestHandler(database, logger))
	service.RegisterHandler(AppIDContactAccept, newContactAcceptHandler(database, logger))
	service.RegisterHandler(AppIDKeySuccession, newKeySuccessionHandler(database, logger))
}

