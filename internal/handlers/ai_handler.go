package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"wikilivee/internal/models"

	"github.com/go-chi/chi/v5"
)

const (
	mwsGPTURL    = "https://api.gpt.mws.ru/v1/chat/completions"
	defaultModel = "mws-gpt-alpha"
)

type AICompleteRequest struct {
	Text string `json:"text"`
}

type AISummarizeRequest struct {
	MaxLength int `json:"maxLength"`
}

type AIBlockSuggestion struct {
	Type    string         `json:"type"`
	Content string         `json:"content"`
	Props   map[string]any `json:"props"`
}

type aiBlockModelResponse struct {
	Type           string         `json:"type"`
	Content        string         `json:"content"`
	SuggestionText string         `json:"suggestionText"`
	Reasoning      string         `json:"reasoning"`
	Props          map[string]any `json:"props"`
}

type AIBlocksRequest struct {
	LastBlock models.Block `json:"lastBlock"`
}

type AIResponse struct {
	Result string `json:"result"`
	Usage  struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type mwsGPTRequest struct {
	Model       string                   `json:"model"`
	Messages    []mwsGPTMessage          `json:"messages"`
	Temperature float64                  `json:"temperature"`
	MaxTokens   int                      `json:"max_tokens"`
}

type mwsGPTMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mwsGPTResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (h *Handler) CompleteTextHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid page id"})
		return
	}

	var req AICompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text cannot be empty"})
		return
	}

	page, err := h.db.GetPage(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
		return
	}

	context := buildPageContext(page)
	prompt := fmt.Sprintf(`You are a helpful assistant. Complete this text naturally.

Context from page "%s":
%s

Text to complete:
%s

Continue the text with 1-2 sentences:`, page.Title, context, req.Text)

	completion, tokens, err := callMWSGPT(r.Context(), prompt, 100, h.cfg.MWSGPTAPIKey)
	if err != nil {
		fallback := fallbackCompletion(req.Text, context)
		writeJSON(w, http.StatusOK, map[string]any{
			"result": fallback,
			"usage": map[string]int{
				"tokens": 0,
			},
			"fallback": true,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result": strings.TrimSpace(completion),
		"usage": map[string]int{
			"tokens": tokens,
		},
	})
}

func (h *Handler) SummarizePageHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid page id"})
		return
	}

	var req AISummarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if req.MaxLength <= 0 {
		req.MaxLength = 200
	}

	page, err := h.db.GetPage(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
		return
	}

	context := buildPageContext(page)

	prompt := fmt.Sprintf(`Summarize the following page content in %d characters or less.

Page title: "%s"

Content:
%s

Summary:`, req.MaxLength, page.Title, context)

	summary, _, err := callMWSGPT(r.Context(), prompt, req.MaxLength+50, h.cfg.MWSGPTAPIKey)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"summary":  fallbackSummary(page.Title, context, req.MaxLength),
			"fallback": true,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"summary": strings.TrimSpace(summary),
	})
}

func (h *Handler) SuggestBlocksHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid page id"})
		return
	}

	var req AIBlocksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	page, err := h.db.GetPage(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
		return
	}

	context := buildPageContext(page)

	blockTypeHint := ""
	blockContentHint := ""
	if req.LastBlock.Type != "" {
		blockTypeHint = req.LastBlock.Type
	}
	if req.LastBlock.Content != nil {
		blockContentHint = *req.LastBlock.Content
	}

	prompt := fmt.Sprintf(`You are a smart editor assistant. Suggest what type of block should come next.

Page context:
%s

Last block type: %s
Last block content: %s

Suggest the next block by responding with JSON:
{
  "type": "paragraph|heading|list|page_link",
  "suggestionText": "suggested content here",
  "reasoning": "why this block makes sense"
}

Response:`, context, blockTypeHint, blockContentHint)

	response, _, err := callMWSGPT(r.Context(), prompt, 500, h.cfg.MWSGPTAPIKey)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"type":       "paragraph",
			"suggestion": fallbackSuggestion(blockTypeHint, blockContentHint, context),
			"fallback":   true,
		})
		return
	}

	parsed := aiBlockModelResponse{}
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		writeJSON(w, http.StatusOK, map[string]string{
			"suggestion": strings.TrimSpace(response),
		})
		return
	}

	suggestionText := strings.TrimSpace(parsed.SuggestionText)
	if suggestionText == "" {
		suggestionText = strings.TrimSpace(parsed.Content)
	}
	if suggestionText == "" {
		suggestionText = strings.TrimSpace(parsed.Reasoning)
	}
	if suggestionText == "" {
		suggestionText = strings.TrimSpace(response)
	}

	blockType := strings.TrimSpace(parsed.Type)
	if blockType == "" {
		blockType = "paragraph"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"type":       blockType,
		"content":    suggestionText,
		"suggestion": suggestionText,
		"props":      parsed.Props,
	})
}

func callMWSGPT(ctx context.Context, prompt string, maxTokens int, apiKey string) (string, int, error) {
	if strings.TrimSpace(apiKey) == "" {
		return "", 0, fmt.Errorf("ai api key is not configured")
	}

	reqBody := mwsGPTRequest{
		Model: defaultModel,
		Messages: []mwsGPTMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   maxTokens,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", mwsGPTURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("mws gpt error: %d - %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var gptResp mwsGPTResponse
	if err := json.NewDecoder(resp.Body).Decode(&gptResp); err != nil {
		return "", 0, err
	}

	if len(gptResp.Choices) == 0 {
		return "", 0, fmt.Errorf("no choices in response")
	}

	totalTokens := gptResp.Usage.PromptTokens + gptResp.Usage.CompletionTokens
	return gptResp.Choices[0].Message.Content, totalTokens, nil
}

func buildPageContext(page models.Page) string {
	var parts []string

	for _, block := range page.Content {
		if block.Content != nil && *block.Content != "" {
			parts = append(parts, *block.Content)
		}
	}

	return strings.Join(parts, "\n")
}

func fallbackCompletion(seed, context string) string {
	base := strings.TrimSpace(seed)
	if base == "" {
		base = "Продолжите текст"
	}

	ctx := strings.TrimSpace(context)
	if ctx == "" {
		return base + ". Добавьте конкретный пример и короткий вывод."
	}

	return base + ". " + fallbackSummary("", ctx, 180)
}

func fallbackSummary(title, context string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 200
	}

	text := strings.TrimSpace(context)
	if text == "" {
		if strings.TrimSpace(title) != "" {
			text = "Страница \"" + strings.TrimSpace(title) + "\" пока содержит мало информации."
		} else {
			text = "На странице пока мало информации для сводки."
		}
	}
	text = strings.Join(strings.Fields(text), " ")
	return trimWithEllipsis(text, maxLen)
}

func fallbackSuggestion(lastType, lastContent, context string) string {
	if strings.TrimSpace(lastContent) != "" {
		return "Продолжите мысль из предыдущего блока: добавьте 1-2 предложения с фактом или примером."
	}
	if strings.TrimSpace(context) != "" {
		return "Сформулируйте следующий абзац как краткий вывод по уже написанному."
	}
	if strings.TrimSpace(lastType) == "heading" {
		return "Добавьте абзац, который раскрывает заголовок конкретными деталями."
	}
	return "Добавьте короткий абзац с ключевой идеей и одним практическим примером."
}

func trimWithEllipsis(text string, maxLen int) string {
	if maxLen <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	if maxLen <= 1 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-1]) + "…"
}
