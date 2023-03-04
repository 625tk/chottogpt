package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type ModerationRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type ModerationCategories string

const (
	Hate            ModerationCategories = "hate"
	HateThreatening ModerationCategories = "hate/threatening"
	SelfHarm        ModerationCategories = "self-harm"
	Sexual          ModerationCategories = "sexual"
	SexualMinors    ModerationCategories = "sexual/minors"
	Violence        ModerationCategories = "violence"
	ViolenceGraphic ModerationCategories = "violence/graphic"
)

var (
	Categories     = []ModerationCategories{Hate, HateThreatening, SelfHarm, Sexual, SexualMinors, Violence, ViolenceGraphic}
	OPENAI_API_KEY = os.Getenv("OPENAI_API_KEY")
)

type ModerationResult struct {
	Categories     map[ModerationCategories]bool    `json:"categories"`
	CategoryScores map[ModerationCategories]float64 `json:"category_scores"`
	Flagged        bool                             `json:"flagged"`
}

type ModerationResponse struct {
	Id      string             `json:"id"`
	Model   string             `json:"model"`
	Results []ModerationResult `json:"results"`
}

type Chat struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model            string            `json:"model"`
	Messages         []Chat            `json:"messages"`
	Temperature      float64           `json:"temperature"`          // Default to 1
	TopP             float64           `json:"top_p"`                // Deafault to 1
	N                int64             `json:"n"`                    // Defaults to 1
	Stream           bool              `json:"stream"`               // Defaults to false
	Stop             []string          `json:"stop,omitempty"`       // Defaults to null
	MaxTokens        int64             `json:"max_tokens"`           // Defaults to inf
	PresencePenalty  int64             `json:"presence_penalty"`     // Defaults to 0
	FrequencyPenalty int64             `json:"frequency_penalty"`    // Defaults to 0
	LogitBias        map[string]string `json:"logit_bias,omitempty"` // Defaults to null
	User             string            `json:"user"`                 // Defaults to
}

type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type CompletionChoice struct {
	Message      Chat   `json:"message"`
	FinishReason string `json:"finish_reason"`
	Index        int    `json:"index"`
}

type ChatCompletionResponse struct {
	Id      string             `json:"id"`
	Object  string             `json:"object"`
	Created int                `json:"created"`
	Model   string             `json:"model"`
	Usage   CompletionUsage    `json:"usage"`
	Choices []CompletionChoice `json:"choices"`
}

func main() {

	ctx := context.Background()
	prompt := "会話に困ったときのおすすめの話題を教えて下さい"

	err := Moderation(ctx, []string{prompt})
	if err != nil {
		log.Fatal("moderation failed:", err)
	}

	res, err := chat(ctx, prompt, "test-1")
	if err != nil {
		log.Fatal("chat Failed:", err)
	}
	log.Println("gpt response is:", res)
}

func chat(ctx context.Context, prompt, userID string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	req := ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Chat{
			{
				Role:    "system",
				Content: "あなたは猫です。猫になったつもりで語尾ににゃなどをつけて答えてください。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature:      0.6,
		TopP:             1,
		N:                1,
		Stream:           false,
		Stop:             nil,
		MaxTokens:        3500,
		PresencePenalty:  0,
		FrequencyPenalty: 0,
		LogitBias:        nil,
		User:             userID,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return "", err
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", OPENAI_API_KEY))

	res, err := http.DefaultClient.Do(r)

	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	a, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", errors.New(res.Status)
	}

	var resp ChatCompletionResponse
	err = json.Unmarshal(a, &resp)
	if err != nil {
		return "", err
	}

	ret := ""

	for _, c := range resp.Choices {
		ret = c.Message.Content
		log.Println("prompt: ", prompt)
		log.Println("response: ", ret)
		log.Println("stop reason: ", c.FinishReason)
		break
	}

	return ret, nil
}

func Moderation(ctx context.Context, prompt []string) error {
	url := "https://api.openai.com/v1/moderations"
	req := ModerationRequest{
		Input: prompt,
		Model: "text-moderation-latest",
	}
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", OPENAI_API_KEY))

	res, err := http.DefaultClient.Do(r)

	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New(res.Status)
	}

	a, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var resp ModerationResponse
	err = json.Unmarshal(a, &resp)
	if err != nil {
		return err
	}

	for _, v := range resp.Results {
		if v.Flagged {
			for _, c := range Categories {
				log.Println(v, v.Categories[c], v.CategoryScores[c])
			}

			return errors.New("MODERATION ERROR")
		}
	}

	return nil
}
