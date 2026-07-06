package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var dbConn *sql.DB

func initDB() {
	dbPath := "/data/app_state/chat.db"
	if err := os.MkdirAll("/data/app_state", 0755); err != nil {
		log.Fatalf("Failed to create app state directory: %v", err)
	}

	var err error
	dbConn, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open sqlite database: %v", err)
	}

	queries := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS conversation_participants (
			conversation_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			invite_link TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (conversation_id, user_id),
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			sender_name TEXT NOT NULL,
			text TEXT NOT NULL,
			attachment_name TEXT NOT NULL DEFAULT '',
			attachment_type TEXT NOT NULL DEFAULT '',
			attachment_path TEXT NOT NULL DEFAULT '',
			sent_at TEXT NOT NULL,
			is_read BOOLEAN NOT NULL DEFAULT 0,
			is_system BOOLEAN NOT NULL DEFAULT 0,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
		)`,
	}

	for _, q := range queries {
		if _, err := dbConn.Exec(q); err != nil {
			log.Fatalf("Failed to run schema query %q: %v", q, err)
		}
	}

	// Migrate existing database schemas if invite_link is missing.
	var hasInviteLink bool
	rows, err := dbConn.Query("PRAGMA table_info(conversation_participants)")
	if err == nil {
		for rows.Next() {
			var cid int
			var name, typeStr string
			var notnull int
			var dfltValue interface{}
			var pk int
			if err := rows.Scan(&cid, &name, &typeStr, &notnull, &dfltValue, &pk); err == nil {
				if name == "invite_link" {
					hasInviteLink = true
					break
				}
			}
		}
		rows.Close()
	}
	if !hasInviteLink {
		log.Println("Migrating database: adding invite_link column to conversation_participants table")
		_, err = dbConn.Exec("ALTER TABLE conversation_participants ADD COLUMN invite_link TEXT NOT NULL DEFAULT ''")
		if err != nil {
			log.Printf("Warning: failed to add invite_link column: %v", err)
		}
	}

	// Try to dynamically add is_system column to messages table if not present
	var hasIsSystem bool
	mRows, err := dbConn.Query("PRAGMA table_info(messages)")
	if err == nil {
		for mRows.Next() {
			var cid int
			var name, typeStr string
			var notnull int
			var dfltValue interface{}
			var pk int
			if err := mRows.Scan(&cid, &name, &typeStr, &notnull, &dfltValue, &pk); err == nil {
				if name == "is_system" {
					hasIsSystem = true
					break
				}
			}
		}
		mRows.Close()
	}
	if !hasIsSystem {
		log.Println("Migrating database: adding is_system column to messages table")
		_, err = dbConn.Exec("ALTER TABLE messages ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT 0")
		if err != nil {
			log.Printf("Warning: failed to add is_system column: %v", err)
		}
	}
}

// Structs

type Conversation struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	CreatedAt    string        `json:"created_at"`
	Participants []Participant `json:"participants"`
	LastMessage  *Message      `json:"last_message,omitempty"`
	UnreadCount  int           `json:"unread_count"`
}

type Participant struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	InviteLink  string `json:"invite_link,omitempty"`
}

type Message struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	SenderName     string `json:"sender_name"`
	Text           string `json:"text"`
	AttachmentName string `json:"attachment_name"`
	AttachmentType string `json:"attachment_type"`
	AttachmentPath string `json:"attachment_path"`
	SentAt         string `json:"sent_at"`
	IsRead         bool   `json:"is_read"`
	IsSystem       bool   `json:"is_system"`
}

// DB functions

func createConversation(id, name string, createdAt string) error {
	_, err := dbConn.Exec("INSERT INTO conversations (id, name, created_at) VALUES (?, ?, ?)", id, name, createdAt)
	return err
}

func addParticipant(conversationID, userID, displayName, inviteLink string) error {
	_, err := dbConn.Exec("INSERT OR REPLACE INTO conversation_participants (conversation_id, user_id, display_name, invite_link) VALUES (?, ?, ?, ?)", conversationID, userID, displayName, inviteLink)
	return err
}

func getConversation(id string) (*Conversation, error) {
	var c Conversation
	err := dbConn.QueryRow("SELECT id, name, created_at FROM conversations WHERE id = ?", id).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	parts, err := getParticipants(c.ID)
	if err != nil {
		return nil, err
	}
	c.Participants = parts

	return &c, nil
}

func getParticipants(conversationID string) ([]Participant, error) {
	rows, err := dbConn.Query("SELECT user_id, display_name, invite_link FROM conversation_participants WHERE conversation_id = ?", conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []Participant
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.UserID, &p.DisplayName, &p.InviteLink); err != nil {
			return nil, err
		}
		parts = append(parts, p)
	}
	return parts, nil
}

func listConversations() ([]Conversation, error) {
	rows, err := dbConn.Query("SELECT id, name, created_at FROM conversations ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		convs = append(convs, c)
	}
	rows.Close() // Release connection back to pool before performing sub-queries

	for i := range convs {
		c := &convs[i]
		parts, err := getParticipants(c.ID)
		if err != nil {
			return nil, err
		}
		c.Participants = parts

		// Get last message
		lastMsg, err := getLastMessage(c.ID)
		if err != nil {
			return nil, err
		}
		c.LastMessage = lastMsg

		// Get unread count
		var unread int
		err = dbConn.QueryRow("SELECT COUNT(*) FROM messages WHERE conversation_id = ? AND is_read = 0", c.ID).Scan(&unread)
		if err != nil {
			unread = 0
		}
		c.UnreadCount = unread
	}

	return convs, nil
}

func insertMessage(m *Message) error {
	_, err := dbConn.Exec(`
		INSERT INTO messages (id, conversation_id, sender_id, sender_name, text, attachment_name, attachment_type, attachment_path, sent_at, is_read, is_system)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.ConversationID, m.SenderID, m.SenderName, m.Text, m.AttachmentName, m.AttachmentType, m.AttachmentPath, m.SentAt, m.IsRead, m.IsSystem,
	)
	return err
}

func getMessages(conversationID string, beforeSentAt string, limit int) ([]Message, error) {
	var rows *sql.Rows
	var err error
	if beforeSentAt == "" {
		rows, err = dbConn.Query(`
			SELECT id, conversation_id, sender_id, sender_name, text, attachment_name, attachment_type, attachment_path, sent_at, is_read, is_system
			FROM messages WHERE conversation_id = ? ORDER BY sent_at DESC LIMIT ?`, conversationID, limit)
	} else {
		rows, err = dbConn.Query(`
			SELECT id, conversation_id, sender_id, sender_name, text, attachment_name, attachment_type, attachment_path, sent_at, is_read, is_system
			FROM messages WHERE conversation_id = ? AND sent_at < ? ORDER BY sent_at DESC LIMIT ?`, conversationID, beforeSentAt, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.SenderName, &m.Text, &m.AttachmentName, &m.AttachmentType, &m.AttachmentPath, &m.SentAt, &m.IsRead, &m.IsSystem); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}

	// Reverse messages so they are returned in ascending chronological order
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	return msgs, nil
}

func searchMessages(conversationID string, query string) ([]Message, error) {
	rows, err := dbConn.Query(`
		SELECT id, conversation_id, sender_id, sender_name, text, attachment_name, attachment_type, attachment_path, sent_at, is_read, is_system
		FROM messages WHERE conversation_id = ? AND text LIKE ? ORDER BY sent_at ASC`, conversationID, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.SenderName, &m.Text, &m.AttachmentName, &m.AttachmentType, &m.AttachmentPath, &m.SentAt, &m.IsRead, &m.IsSystem); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func getLastMessage(conversationID string) (*Message, error) {
	var m Message
	err := dbConn.QueryRow(`
		SELECT id, conversation_id, sender_id, sender_name, text, attachment_name, attachment_type, attachment_path, sent_at, is_read, is_system
		FROM messages WHERE conversation_id = ? ORDER BY sent_at DESC LIMIT 1`, conversationID).
		Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.SenderName, &m.Text, &m.AttachmentName, &m.AttachmentType, &m.AttachmentPath, &m.SentAt, &m.IsRead, &m.IsSystem)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func markMessagesAsRead(conversationID string) error {
	_, err := dbConn.Exec("UPDATE messages SET is_read = 1 WHERE conversation_id = ? AND is_read = 0", conversationID)
	return err
}

func deleteConversation(id string) error {
	_, err := dbConn.Exec("DELETE FROM conversations WHERE id = ?", id)
	return err
}

func renameConversation(id, name string) error {
	_, err := dbConn.Exec("UPDATE conversations SET name = ? WHERE id = ?", name, id)
	return err
}

func getSharedMedia(conversationID string, beforeSentAt string, limit int) ([]Message, error) {
	var rows *sql.Rows
	var err error
	if beforeSentAt == "" {
		rows, err = dbConn.Query(`
			SELECT id, conversation_id, sender_id, sender_name, text, attachment_name, attachment_type, attachment_path, sent_at, is_read, is_system
			FROM messages 
			WHERE conversation_id = ? AND attachment_name != '' 
			ORDER BY sent_at DESC LIMIT ?`, conversationID, limit)
	} else {
		rows, err = dbConn.Query(`
			SELECT id, conversation_id, sender_id, sender_name, text, attachment_name, attachment_type, attachment_path, sent_at, is_read, is_system
			FROM messages 
			WHERE conversation_id = ? AND attachment_name != '' AND sent_at < ? 
			ORDER BY sent_at DESC LIMIT ?`, conversationID, beforeSentAt, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.SenderName, &m.Text, &m.AttachmentName, &m.AttachmentType, &m.AttachmentPath, &m.SentAt, &m.IsRead, &m.IsSystem); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}
