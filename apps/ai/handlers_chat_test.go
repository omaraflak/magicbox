package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleChat_Success(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Chat", `{"temp":0.5}`)

	oldRunChatStream := runChatStream
	defer func() { runChatStream = oldRunChatStream }()

	runChatStream = func(ctx context.Context, conversationID string, newMsgContent string, ch chan<- string, errCh chan<- error) {
		// Mock streaming of tokens
		ch <- "Hello"
		ch <- " world!"
		close(ch)
		close(errCh)
	}

	body := strings.NewReader(`{"message":"Hello"}`)
	req := httptest.NewRequest("POST", "/api/conversations/"+c.ID+"/chat", body)
	req.SetPathValue("id", c.ID)

	w := httptest.NewRecorder()
	handleChat(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, `data: {"content":"Hello"}`) || !strings.Contains(bodyStr, `data: {"content":" world!"}`) {
		t.Errorf("Expected SSE stream data, got:\n%s", bodyStr)
	}
}

func TestHandleChat_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/conversations/123/chat", nil)
	w := httptest.NewRecorder()
	handleChat(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleChat_MissingID(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/conversations//chat", nil)
	w := httptest.NewRecorder()
	handleChat(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleRegenerate_Success(t *testing.T) {
	setupTestDB(t)
	defer dbConn.Close()

	c, _ := createConversation("Chat", `{"temp":0.5}`)
	_, _ = addMessage(c.ID, "user", "Original User Message")
	_, _ = addMessage(c.ID, "model", "Original Model Response")

	oldRunChatStream := runChatStream
	defer func() { runChatStream = oldRunChatStream }()

	runChatStream = func(ctx context.Context, conversationID string, newMsgContent string, ch chan<- string, errCh chan<- error) {
		if newMsgContent != "Original User Message" {
			errCh <- errors.New("Unexpected user message: " + newMsgContent)
			close(ch)
			close(errCh)
			return
		}
		ch <- "Regenerated Response"
		close(ch)
		close(errCh)
	}

	req := httptest.NewRequest("POST", "/api/conversations/"+c.ID+"/regenerate", nil)
	req.SetPathValue("id", c.ID)

	w := httptest.NewRecorder()
	handleRegenerate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, `data: {"content":"Regenerated Response"}`) {
		t.Errorf("Expected SSE stream data, got:\n%s", bodyStr)
	}
}

func TestHandleRegenerate_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/conversations/123/regenerate", nil)
	w := httptest.NewRecorder()
	handleRegenerate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHandleRegenerate_MissingID(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/conversations//regenerate", nil)
	w := httptest.NewRecorder()
	handleRegenerate(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}
