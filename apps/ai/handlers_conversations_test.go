package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleConversations_GET_Empty(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	req := httptest.NewRequest("GET", "/api/conversations", nil)
	w := httptest.NewRecorder()
	handleConversations(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var list []Conversation
	json.NewDecoder(resp.Body).Decode(&list)
	if len(list) != 0 {
		t.Errorf("Expected 0 conversations, got %d", len(list))
	}
}

func TestHandleConversations_POST_Create(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	body := strings.NewReader(`{"title":"My Test Conversation","params":"{\"temp\":0.5}"}`)
	req := httptest.NewRequest("POST", "/api/conversations", body)
	w := httptest.NewRecorder()
	handleConversations(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var c Conversation
	json.NewDecoder(resp.Body).Decode(&c)
	if c.Title != "My Test Conversation" || c.Params != `{"temp":0.5}` {
		t.Errorf("Unexpected conversation details in response: %+v", c)
	}
}

func TestHandleConversations_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("PUT", "/api/conversations", nil)
	w := httptest.NewRecorder()
	handleConversations(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleConversationDetail_GET_Exists(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Chat Title", `{"temp":0.5}`)
	_, _ = addMessage(c.ID, "user", "Hello there")

	req := httptest.NewRequest("GET", "/api/conversations/"+c.ID, nil)
	req.SetPathValue("id", c.ID)

	w := httptest.NewRecorder()
	handleConversationDetail(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result struct {
		Conversation
		Messages []Message `json:"messages"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.ID != c.ID || len(result.Messages) != 1 || result.Messages[0].Content != "Hello there" {
		t.Errorf("Unexpected details in response: %+v", result)
	}
}

func TestHandleConversationDetail_GET_NotFound(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	req := httptest.NewRequest("GET", "/api/conversations/non-existent", nil)
	req.SetPathValue("id", "non-existent")

	w := httptest.NewRecorder()
	handleConversationDetail(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestHandleConversationDetail_DELETE(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Delete Me", `{"temp":0.5}`)

	req := httptest.NewRequest("DELETE", "/api/conversations/"+c.ID, nil)
	req.SetPathValue("id", c.ID)

	w := httptest.NewRecorder()
	handleConversationDetail(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify deletion in DB
	cDeleted, _ := getConversation(c.ID)
	if cDeleted != nil {
		t.Errorf("Conversation still exists in DB after DELETE request")
	}
}

func TestHandleConversationDetail_MissingID(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/conversations/", nil)
	w := httptest.NewRecorder()
	handleConversationDetail(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleConversationDetail_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/conversations/123", nil)
	req.SetPathValue("id", "123")
	w := httptest.NewRecorder()
	handleConversationDetail(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleConversationParams_PUT(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Chat Title", `{"temp":0.5}`)

	body := strings.NewReader(`{"params":"{\"temp\":0.9}"}`)
	req := httptest.NewRequest("PUT", "/api/conversations/"+c.ID+"/params", body)
	req.SetPathValue("id", c.ID)

	w := httptest.NewRecorder()
	handleConversationParams(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	cUpdated, _ := getConversation(c.ID)
	if cUpdated.Params != `{"temp":0.9}` {
		t.Errorf("Expected params updated to '{\"temp\":0.9}', got %s", cUpdated.Params)
	}
}

func TestHandleConversationParams_MissingID(t *testing.T) {
	body := strings.NewReader(`{"params":"{\"temp\":0.9}"}`)
	req := httptest.NewRequest("PUT", "/api/conversations//params", body)
	w := httptest.NewRecorder()
	handleConversationParams(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleConversationParams_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/conversations/123/params", nil)
	req.SetPathValue("id", "123")
	w := httptest.NewRecorder()
	handleConversationParams(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleConversationTitle_PUT(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Chat Title", `{"temp":0.5}`)

	body := strings.NewReader(`{"title":"New Custom Title"}`)
	req := httptest.NewRequest("PUT", "/api/conversations/"+c.ID+"/title", body)
	req.SetPathValue("id", c.ID)

	w := httptest.NewRecorder()
	handleConversationTitle(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	cUpdated, _ := getConversation(c.ID)
	if cUpdated.Title != "New Custom Title" {
		t.Errorf("Expected title updated to 'New Custom Title', got %s", cUpdated.Title)
	}
}

func TestHandleConversationTitle_MissingID(t *testing.T) {
	body := strings.NewReader(`{"title":"New Custom Title"}`)
	req := httptest.NewRequest("PUT", "/api/conversations//title", body)
	w := httptest.NewRecorder()
	handleConversationTitle(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleConversationTitle_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/conversations/123/title", nil)
	req.SetPathValue("id", "123")
	w := httptest.NewRecorder()
	handleConversationTitle(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleConversationFork_POST(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	parent, _ := createConversation("Parent Title", `{"temp":0.5}`)
	m, _ := addMessage(parent.ID, "user", "Message 1")

	body := strings.NewReader(`{"message_id":"` + m.ID + `"}`)
	req := httptest.NewRequest("POST", "/api/conversations/"+parent.ID+"/fork", body)
	req.SetPathValue("id", parent.ID)

	w := httptest.NewRecorder()
	handleConversationFork(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var fork Conversation
	json.NewDecoder(resp.Body).Decode(&fork)
	if fork.ParentID == nil || *fork.ParentID != parent.ID {
		t.Errorf("Fork parent ID does not match parent: %+v", fork)
	}
}

func TestHandleConversationFork_MissingID(t *testing.T) {
	body := strings.NewReader(`{"message_id":"123"}`)
	req := httptest.NewRequest("POST", "/api/conversations//fork", body)
	w := httptest.NewRecorder()
	handleConversationFork(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleConversationFork_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/conversations/123/fork", nil)
	req.SetPathValue("id", "123")
	w := httptest.NewRecorder()
	handleConversationFork(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleConversationForks_GET(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	parent, _ := createConversation("Parent Title", `{"temp":0.5}`)
	m, _ := addMessage(parent.ID, "user", "Message 1")
	fork, _ := forkConversation(parent.ID, m.ID)

	req := httptest.NewRequest("GET", "/api/conversations/"+parent.ID+"/forks", nil)
	req.SetPathValue("id", parent.ID)

	w := httptest.NewRecorder()
	handleConversationForks(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var forks []Conversation
	json.NewDecoder(resp.Body).Decode(&forks)
	if len(forks) != 1 || forks[0].ID != fork.ID {
		t.Errorf("Forks response does not match created fork: %+v", forks)
	}
}

func TestHandleConversationForks_MissingID(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/conversations//forks", nil)
	w := httptest.NewRecorder()
	handleConversationForks(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleConversationForks_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/conversations/123/forks", nil)
	req.SetPathValue("id", "123")
	w := httptest.NewRecorder()
	handleConversationForks(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleConversationTreeContext_GET(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	parent, _ := createConversation("Parent Title", `{"temp":0.5}`)
	m, _ := addMessage(parent.ID, "user", "Message 1")
	fork, _ := forkConversation(parent.ID, m.ID)

	req := httptest.NewRequest("GET", "/api/conversations/"+fork.ID+"/tree-context", nil)
	req.SetPathValue("id", fork.ID)

	w := httptest.NewRecorder()
	handleConversationTreeContext(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var tree []Conversation
	json.NewDecoder(resp.Body).Decode(&tree)
	if len(tree) != 2 {
		t.Errorf("Expected 2 conversations in tree context, got %d", len(tree))
	}
}

func TestHandleConversationTreeContext_MissingID(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/conversations//tree-context", nil)
	w := httptest.NewRecorder()
	handleConversationTreeContext(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleConversationTreeContext_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/conversations/123/tree-context", nil)
	req.SetPathValue("id", "123")
	w := httptest.NewRecorder()
	handleConversationTreeContext(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleUndoLastTurn_POST(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Test Chat", `{"temp":0.7}`)
	_, _ = addMessage(c.ID, "user", "User Message")
	_, _ = addMessage(c.ID, "model", "Model Response")

	req := httptest.NewRequest("POST", "/api/conversations/"+c.ID+"/undo-last-turn", nil)
	req.SetPathValue("id", c.ID)

	w := httptest.NewRecorder()
	handleUndoLastTurn(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var data map[string]string
	json.NewDecoder(resp.Body).Decode(&data)
	if data["content"] != "User Message" {
		t.Errorf("Expected returned content to be 'User Message', got '%s'", data["content"])
	}

	// Verify message deletion in DB
	msgs, _ := getMessages(c.ID)
	if len(msgs) != 0 {
		t.Errorf("Expected 0 messages in DB after undo, got %d", len(msgs))
	}
}

func TestHandleUndoLastTurn_MissingID(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/conversations//undo-last-turn", nil)
	w := httptest.NewRecorder()
	handleUndoLastTurn(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleUndoLastTurn_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/conversations/123/undo-last-turn", nil)
	req.SetPathValue("id", "123")
	w := httptest.NewRecorder()
	handleUndoLastTurn(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}
