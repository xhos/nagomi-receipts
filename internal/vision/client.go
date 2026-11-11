// Package vision is a minimal client for OpenAI-compatible chat completion
// APIs with image input: Ollama, Gemini, OpenAI, vLLM, llama.cpp and friends.
package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	baseURL string
	apiKey  string
	Model   string
	http    *http.Client
}

func New(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		Model:   model,
		http:    &http.Client{},
	}
}

// Ask sends a text prompt together with one image and returns the reply text.
func (c *Client) Ask(ctx context.Context, prompt string, image []byte, mimeType string) (string, error) {
	dataURL := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(image)

	req := chatRequest{
		Model: c.Model,
		Messages: []message{{
			Role: "user",
			Content: []part{
				{Type: "text", Text: prompt},
				{Type: "image_url", ImageURL: &imageURL{URL: dataURL}},
			},
		}},
	}

	var resp chatResponse
	if err := c.do(ctx, http.MethodPost, "/chat/completions", req, &resp); err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("model returned no choices")
	}

	return resp.Choices[0].Message.Content, nil
}

// Ping checks that the API is reachable and the key is accepted.
func (c *Client) Ping(ctx context.Context) error {
	return c.do(ctx, http.MethodGet, "/models", nil, nil)
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, bytes.TrimSpace(msg))
	}
	if out == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content []part `json:"content"`
}

type part struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
