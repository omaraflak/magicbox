package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Params struct {
	Model        string  `json:"model"`
	SystemPrompt string  `json:"system_prompt"`
	Temperature  float32 `json:"temperature"`
	TopK         int32   `json:"top_k"`
	TopP         float32 `json:"top_p"`
}

func getGeminiClient(ctx context.Context) (*genai.Client, error) {
	apiKey := getSetting("api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("API key not set")
	}
	return genai.NewClient(ctx, option.WithAPIKey(apiKey))
}

func getGeminiModel(client *genai.Client, params Params) *genai.GenerativeModel {
	modelName := params.Model
	if modelName == "" {
		modelName = "gemini-3.1-flash-lite"
	}
	modelName = strings.TrimPrefix(modelName, "models/")
	model := client.GenerativeModel(modelName)
	
	if params.Temperature != 0 {
		model.SetTemperature(params.Temperature)
	}
	if params.TopK != 0 {
		model.SetTopK(params.TopK)
	}
	if params.TopP != 0 {
		model.SetTopP(params.TopP)
	}
	if params.SystemPrompt != "" {
		model.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(params.SystemPrompt)},
		}
	}
	
	// No tools configured for now
	
	return model
}

func chatStream(ctx context.Context, conversationID string, newMsgContent string, ch chan<- string, errCh chan<- error) {
	defer close(ch)
	defer close(errCh)

	client, err := getGeminiClient(ctx)
	if err != nil {
		errCh <- err
		return
	}
	defer client.Close()

	conv, err := getConversation(conversationID)
	if err != nil {
		errCh <- err
		return
	}

	var params Params
	if conv.Params != "" {
		if err := json.Unmarshal([]byte(conv.Params), &params); err != nil {
			log.Printf("Failed to parse params: %v", err)
		}
	}

	model := getGeminiModel(client, params)
	
	msgs, err := getMessages(conversationID)
	if err != nil {
		errCh <- err
		return
	}

	cs := model.StartChat()
	for _, m := range msgs {
		role := "user"
		if m.Role == "model" {
			role = "model"
		}
		cs.History = append(cs.History, &genai.Content{
			Role:  role,
			Parts: []genai.Part{genai.Text(m.Content)},
		})
	}

	// Add user msg
	_, err = addMessage(conversationID, "user", newMsgContent)
	if err != nil {
		errCh <- err
		return
	}

	iter := cs.SendMessageStream(ctx, genai.Text(newMsgContent))
	
	var fullResponse string
	
	for {
		resp, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			errCh <- err
			return
		}
		
		for _, cand := range resp.Candidates {
			if cand.Content != nil {
				for _, part := range cand.Content.Parts {
					if text, ok := part.(genai.Text); ok {
						ch <- string(text)
						fullResponse += string(text)
					}
				}
			}
		}
	}
	
	_, err = addMessage(conversationID, "model", fullResponse)
	if err != nil {
		log.Printf("Failed to save model message: %v", err)
	}
}
