package utils

import (
	"os"
)

type Secrets struct {
	OpenAIAPIKey string
	OpenAIModel  string
}

func LoadSecrets() *Secrets {
	return &Secrets{
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:  os.Getenv("OPENAI_MODEL"),
	}
}
