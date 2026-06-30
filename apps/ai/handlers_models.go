package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/api/iterator"
)

type ModelInfo struct {
	Name                       string   `json:"name"`
	DisplayName                string   `json:"display_name"`
	Description                string   `json:"description"`
	SupportedGenerationMethods []string `json:"supported_methods"`
	InputTokenLimit            int32    `json:"input_token_limit"`
	OutputTokenLimit           int32    `json:"output_token_limit"`
	Temperature                float32  `json:"temperature"`
	TopP                       float32  `json:"top_p"`
	TopK                       int32    `json:"top_k"`
}

var getModels = func(ctx context.Context) ([]ModelInfo, error) {
	client, err := getGeminiClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Gemini client: %w", err)
	}
	defer client.Close()

	var models []ModelInfo
	iter := client.ListModels(ctx)
	for {
		m, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to list models: %w", err)
		}

		// Filter: must support generateContent (chat-capable)
		supportsChat := false
		for _, method := range m.SupportedGenerationMethods {
			if method == "generateContent" {
				supportsChat = true
				break
			}
		}
		if !supportsChat {
			continue
		}

		// Filter: skip experimental/preview models
		nameLower := strings.ToLower(m.Name)
		if strings.Contains(nameLower, "-exp") || strings.Contains(nameLower, "-preview") {
			continue
		}

		info := ModelInfo{
			Name:                       m.Name,
			DisplayName:                m.DisplayName,
			Description:                m.Description,
			SupportedGenerationMethods: m.SupportedGenerationMethods,
			InputTokenLimit:            m.InputTokenLimit,
			OutputTokenLimit:           m.OutputTokenLimit,
			Temperature:                m.Temperature,
			TopP:                       m.TopP,
			TopK:                       m.TopK,
		}

		models = append(models, info)
	}
	return models, nil
}

func handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	models, err := getModels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models)
}
