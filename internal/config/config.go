package config

import (
	"cmp"
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

type Config struct {
	ListenAddress string
	LogLevel      log.Level
	LogFormat     string // "json" | "text"

	BaseURL string // any OpenAI-compatible API, e.g. Ollama's http://127.0.0.1:11434/v1
	APIKey  string // optional; local servers usually don't need one
	Model   string
}

func Load() Config {
	logLevel, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = log.InfoLevel
	}

	logFormat := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT")))
	if logFormat != "json" {
		logFormat = "text"
	}

	return Config{
		ListenAddress: parseAddress(cmp.Or(os.Getenv("LISTEN_ADDRESS"), "127.0.0.1:55556")),
		LogLevel:      logLevel,
		LogFormat:     logFormat,
		BaseURL:       cmp.Or(os.Getenv("OPENAI_BASE_URL"), "http://127.0.0.1:11434/v1"),
		APIKey:        os.Getenv("OPENAI_API_KEY"),
		Model:         cmp.Or(os.Getenv("OPENAI_MODEL"), "qwen2.5vl:3b"),
	}
}

// parseAddress accepts "8080", ":8080" or "127.0.0.1:8080".
func parseAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if strings.Contains(addr, ":") {
		return addr
	}
	return ":" + addr
}
