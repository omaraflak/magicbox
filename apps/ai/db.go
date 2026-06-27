package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var dbConn *sql.DB

func initDB() {
	dbPath := "/data/app_state/ai.db"
	// Create directory if it doesn't exist
	os.MkdirAll("/data/app_state", 0755)
	
	// Fallback to local if not in docker
	if _, err := os.Stat("/data/app_state"); os.IsNotExist(err) {
		dbPath = "ai.db"
	}

	var err error
	dbConn, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open sqlite database: %v", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			params TEXT NOT NULL,
			parent_id TEXT REFERENCES conversations(id)
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(conversation_id) REFERENCES conversations(id)
		)`,
		`CREATE TABLE IF NOT EXISTS presets (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			params TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, q := range queries {
		if _, err := dbConn.Exec(q); err != nil {
			log.Fatalf("Failed to run schema queries: %v", err)
		}
	}

	// Migrate existing database
	_, _ = dbConn.Exec("ALTER TABLE conversations ADD COLUMN parent_id TEXT REFERENCES conversations(id)")
}

func getSetting(key string) string {
	var value string
	err := dbConn.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return ""
	}
	return value
}

func setSetting(key, value string) error {
	_, err := dbConn.Exec("INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
	return err
}

type Conversation struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	Params    string  `json:"params"`
	ParentID  *string `json:"parent_id"`
}

type Message struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	CreatedAt      string `json:"created_at"`
}

func scanConversations(rows *sql.Rows) ([]Conversation, error) {
	var convs []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.Title, &c.CreatedAt, &c.UpdatedAt, &c.Params, &c.ParentID); err != nil {
			log.Printf("Failed to scan conversation row: %v", err)
			continue
		}
		convs = append(convs, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return convs, nil
}

func removeLastTurn(conversationID string) (string, error) {
	msgs, err := getMessages(conversationID)
	if err != nil {
		return "", fmt.Errorf("failed to get messages: %w", err)
	}

	if len(msgs) == 0 {
		return "", fmt.Errorf("no messages in conversation")
	}

	if msgs[len(msgs)-1].Role == "model" {
		if err := deleteMessage(msgs[len(msgs)-1].ID); err != nil {
			return "", fmt.Errorf("failed to delete model message: %w", err)
		}
		msgs = msgs[:len(msgs)-1]
	}

	if len(msgs) == 0 || msgs[len(msgs)-1].Role != "user" {
		return "", fmt.Errorf("last message is not from user")
	}

	lastUserContent := msgs[len(msgs)-1].Content
	if err := deleteMessage(msgs[len(msgs)-1].ID); err != nil {
		return "", fmt.Errorf("failed to delete user message: %w", err)
	}

	return lastUserContent, nil
}

func getConversationsPaginated(limit, offset int) ([]Conversation, error) {
	rows, err := dbConn.Query("SELECT id, title, created_at, updated_at, params, parent_id FROM conversations WHERE parent_id IS NULL ORDER BY updated_at DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanConversations(rows)
}

func getConversationForks(parentID string) ([]Conversation, error) {
	rows, err := dbConn.Query("SELECT id, title, created_at, updated_at, params, parent_id FROM conversations WHERE parent_id = ? ORDER BY created_at ASC", parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanConversations(rows)
}

func getConversationTreeContext(id string) ([]Conversation, error) {
	query := `
		WITH RECURSIVE path(id, parent_id) AS (
			SELECT id, parent_id FROM conversations WHERE id = ?
			UNION ALL
			SELECT c.id, c.parent_id FROM conversations c
			JOIN path p ON c.id = p.parent_id
		)
		SELECT id, title, created_at, updated_at, params, parent_id 
		FROM conversations 
		WHERE id IN (SELECT id FROM path) 
		   OR parent_id IN (SELECT id FROM path)
	`
	rows, err := dbConn.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanConversations(rows)
}

func getConversation(id string) (*Conversation, error) {
	var c Conversation
	err := dbConn.QueryRow("SELECT id, title, created_at, updated_at, params, parent_id FROM conversations WHERE id = ?", id).Scan(&c.ID, &c.Title, &c.CreatedAt, &c.UpdatedAt, &c.Params, &c.ParentID)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func createConversation(title, params string) (*Conversation, error) {
	id := uuid.New().String()
	_, err := dbConn.Exec("INSERT INTO conversations (id, title, params) VALUES (?, ?, ?)", id, title, params)
	if err != nil {
		return nil, err
	}
	return getConversation(id)
}

func forkConversation(parentID, upToMessageID string) (*Conversation, error) {
	parent, err := getConversation(parentID)
	if err != nil {
		return nil, err
	}

	parentMsgs, err := getMessages(parentID)
	if err != nil {
		return nil, err
	}

	childID := uuid.New().String()
	title := parent.Title + " (Fork)"
	_, err = dbConn.Exec("INSERT INTO conversations (id, title, params, parent_id) VALUES (?, ?, ?, ?)", childID, title, parent.Params, parentID)
	if err != nil {
		return nil, err
	}

	for _, m := range parentMsgs {
		newMsgID := uuid.New().String()
		_, err = dbConn.Exec("INSERT INTO messages (id, conversation_id, role, content, created_at) VALUES (?, ?, ?, ?, ?)", newMsgID, childID, m.Role, m.Content, m.CreatedAt)
		if err != nil {
			return nil, err
		}
		if m.ID == upToMessageID {
			break
		}
	}

	return getConversation(childID)
}

func updateConversationParams(id, params string) error {
	_, err := dbConn.Exec("UPDATE conversations SET params = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", params, id)
	return err
}

func updateConversationTitle(id, title string) error {
	_, err := dbConn.Exec("UPDATE conversations SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", title, id)
	return err
}

func getMessages(conversationID string) ([]Message, error) {
	rows, err := dbConn.Query("SELECT id, conversation_id, role, content, created_at FROM messages WHERE conversation_id = ? ORDER BY created_at ASC", conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			log.Printf("Failed to scan message row: %v", err)
			continue
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return msgs, nil
}

func addMessage(conversationID, role, content string) (*Message, error) {
	id := uuid.New().String()
	_, err := dbConn.Exec("INSERT INTO messages (id, conversation_id, role, content) VALUES (?, ?, ?, ?)", id, conversationID, role, content)
	if err != nil {
		return nil, err
	}
	_, err = dbConn.Exec("UPDATE conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = ?", conversationID)
	if err != nil {
		return nil, err
	}
	
	var m Message
	err = dbConn.QueryRow("SELECT id, conversation_id, role, content, created_at FROM messages WHERE id = ?", id).Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func deleteConversation(id string) error {
	_, err := dbConn.Exec("DELETE FROM messages WHERE conversation_id = ?", id)
	if err != nil {
		return err
	}
	_, err = dbConn.Exec("DELETE FROM conversations WHERE id = ?", id)
	return err
}

func deleteMessage(id string) error {
	_, err := dbConn.Exec("DELETE FROM messages WHERE id = ?", id)
	return err
}

type Preset struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Params    string `json:"params"`
	CreatedAt string `json:"created_at"`
}

func getPresets() ([]Preset, error) {
	rows, err := dbConn.Query("SELECT id, name, params, created_at FROM presets ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ps []Preset
	for rows.Next() {
		var p Preset
		if err := rows.Scan(&p.ID, &p.Name, &p.Params, &p.CreatedAt); err != nil {
			log.Printf("Failed to scan preset row: %v", err)
			continue
		}
		ps = append(ps, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ps, nil
}

func createPreset(name, params string) (*Preset, error) {
	id := uuid.New().String()
	_, err := dbConn.Exec("INSERT INTO presets (id, name, params) VALUES (?, ?, ?)", id, name, params)
	if err != nil {
		return nil, err
	}
	var p Preset
	err = dbConn.QueryRow("SELECT id, name, params, created_at FROM presets WHERE id = ?", id).Scan(&p.ID, &p.Name, &p.Params, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func deletePreset(id string) error {
	_, err := dbConn.Exec("DELETE FROM presets WHERE id = ?", id)
	return err
}

func updatePreset(id, name, params string) error {
	_, err := dbConn.Exec("UPDATE presets SET name = ?, params = ? WHERE id = ?", name, params, id)
	return err
}
