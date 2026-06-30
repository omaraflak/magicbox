package main

import (
	"net/http"

	"google.golang.org/api/iterator"
)

type ModelInfo struct {
	Name                       string   `json:"name"`
	DisplayName                string   `json:"display_name"`
	Description                string   `json:"description"`
	SupportedGenerationMethods []string `json:"supported_generation_methods"`
	InputTokenLimit            int32    `json:"input_token_limit"`
	OutputTokenLimit           int32    `json:"output_token_limit"`
	Temperature                float32  `json:"temperature"`
	TopP                       float32  `json:"top_p"`
	TopK                       int32    `json:"top_k"`
}

func handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	client, err := getGeminiClient(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create Gemini client: "+err.Error())
		return
	}
	defer client.Close()

	var models []ModelInfo
	iter := client.ListModels(r.Context())
	for {
		m, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to list models: "+err.Error())
			return
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

	writeJSON(w, http.StatusOK, models)
}
