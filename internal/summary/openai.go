package summary

import (
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"log"
	"strings"
	"sync"
)

// Имплементация интерфейса summarizer, который уже ранее был объявлен
type OpenAISummarizer struct {
	// sdk для openai
	client *openai.Client
	// С его помощью будем просить gpt генерить summary
	promt string
	// Флаг вкл/выкл summarizer
	enabled bool
	mu      sync.Mutex
}

func NewOpenAISummarizer(apiKey string, promt string) *OpenAISummarizer {
	s := &OpenAISummarizer{
		client: openai.NewClient(apiKey),
		promt:  promt,
	}

	log.Printf("openai summarizer enabled: %v", apiKey != "")

	if apiKey != "" {
		s.enabled = true
	}

	return s
}

func (s *OpenAISummarizer) Summarize(ctx context.Context, text string) (string, error) {
	// Обкладываем мьютексами, т.к. конкурентный доступ может вызывать сюрпризы
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled {
		return "", nil
	}

	// Составляем запрос к openai
	request := openai.ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: fmt.Sprintf("%s%s", text, s.promt),
			},
		},
		MaxTokens:   256,
		Temperature: 0.7,
		TopP:        1,
	}

	// Отправляем запрос
	resp, err := s.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", err
	}

	// openai отправляем нам несколько вариантов, мы выбираем самый первый
	rawSummary := strings.TrimSpace(resp.Choices[0].Message.Content)
	if strings.HasSuffix(rawSummary, ".") {
		return rawSummary, nil
	}

	sentences := strings.Split(rawSummary, ".")

	// Берем все предложения кроме последнего. добавляем между ними точку. и к последнему предложению добавляем точку.
	return strings.Join(sentences[:len(sentences)-1], ".") + ".", nil
}
