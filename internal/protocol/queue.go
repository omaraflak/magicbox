package protocol

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/magicbox/core/internal/db"
	"github.com/magicbox/core/internal/logging"
	"github.com/magicbox/core/internal/p2p"
)

// nextRetryDelay returns the delay before the next retry attempt using
// exponential backoff: 1m, 2m, 4m, 8m, 16m, 32m, 64m, 128m, ..., capped at 24h.
func nextRetryDelay(attempts int) time.Duration {
	const maxDelay = 24 * time.Hour
	if attempts > 20 {
		return maxDelay
	}
	d := time.Minute * time.Duration(1<<uint(attempts))
	if d > maxDelay {
		d = maxDelay
	}
	return d
}

// EnqueueForContacts queues a system message for delivery to all given contacts.
func EnqueueForContacts(database *db.DB, contacts []db.Contact, appID string, payload []byte) error {
	for _, contact := range contacts {
		id := uuid.NewString()
		if err := database.EnqueueMessage(id, contact.ID, appID, payload, 10); err != nil {
			return fmt.Errorf("failed to enqueue message for contact %s: %w", contact.ID, err)
		}
	}
	return nil
}

// ProcessQueue attempts to deliver all pending messages in the queue.
// Successfully delivered messages are removed; failed ones are rescheduled
// with exponential backoff. Messages for deleted contacts are silently
// excluded by the database JOIN.
func ProcessQueue(database *db.DB, p2pService p2p.Service, logger *logging.Logger) {
	msgs, err := database.GetPendingMessages()
	if err != nil {
		logger.Error("Failed to get pending messages from queue", logging.F("error", err.Error()))
		return
	}

	if len(msgs) == 0 {
		return
	}

	logger.Info("Processing message queue", logging.F("pending", len(msgs)))

	for _, m := range msgs {
		p2pMsg := &p2p.Message{
			AppID:        m.AppID,
			TargetUserID: m.TargetUserID,
			Payload:      m.Payload,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := p2pService.SendTo(ctx, m.Multiaddr, m.EncPubKey, p2pMsg)
		cancel()

		if err != nil {
			delay := nextRetryDelay(m.Attempts)
			nextRetry := time.Now().Add(delay).UTC().Format(time.RFC3339)
			if dbErr := database.IncrementMessageAttempts(m.ID, nextRetry); dbErr != nil {
				logger.Error("Failed to update message retry",
					logging.F("msg_id", m.ID),
					logging.F("error", dbErr.Error()))
			}
			logger.Warn("Failed to deliver queued message, will retry",
				logging.F("msg_id", m.ID),
				logging.F("app_id", m.AppID),
				logging.F("contact_id", m.ContactID),
				logging.F("attempt", m.Attempts+1),
				logging.F("next_retry", nextRetry),
				logging.F("error", err.Error()))
			continue
		}

		if err := database.DeleteMessage(m.ID); err != nil {
			logger.Error("Failed to delete delivered message from queue",
				logging.F("msg_id", m.ID),
				logging.F("error", err.Error()))
		}

		logger.Info("Delivered queued message",
			logging.F("msg_id", m.ID),
			logging.F("app_id", m.AppID),
			logging.F("contact_id", m.ContactID))
	}

	// Clean up any messages that exhausted their retries.
	cleaned, err := database.CleanExpiredMessages()
	if err != nil {
		logger.Error("Failed to clean expired messages", logging.F("error", err.Error()))
	}
	if cleaned > 0 {
		logger.Warn("Dropped undeliverable messages", logging.F("count", cleaned))
	}
}

// StartQueueProcessor starts a background goroutine that processes the message
// queue at the given interval. Returns a stop function.
func StartQueueProcessor(database *db.DB, p2pService p2p.Service, logger *logging.Logger, interval time.Duration) func() {
	ticker := time.NewTicker(interval)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				ProcessQueue(database, p2pService, logger)
			case <-done:
				return
			}
		}
	}()

	return func() {
		ticker.Stop()
		close(done)
	}
}
