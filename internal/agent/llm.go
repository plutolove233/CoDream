package agent

import (
	"os"

	"github.com/sashabaranov/go-openai"
)

func NewClient() *openai.Client {
	cfg := openai.DefaultConfig(os.Getenv("API_KEY"))
	cfg.BaseURL = os.Getenv("BASE_URL")
	return openai.NewClientWithConfig(cfg)
}
