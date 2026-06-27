package utils

import (
	"os"
)

type Secrets struct {
	OpenAIAPIKey     string
	OpenAIModel      string
	MetaAppID        string
	MetaAppSecret    string
	MetaRedirectURI  string
	MetaGraphVersion string
	AppBaseURL       string
}

func LoadSecrets() *Secrets {
	graphVersion := os.Getenv("META_GRAPH_VERSION")
	if graphVersion == "" {
		graphVersion = "v21.0"
	}
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &Secrets{
		OpenAIAPIKey:     os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:      os.Getenv("OPENAI_MODEL"),
		MetaAppID:        os.Getenv("META_APP_ID"),
		MetaAppSecret:    os.Getenv("META_APP_SECRET"),
		MetaRedirectURI:  os.Getenv("META_REDIRECT_URI"),
		MetaGraphVersion: graphVersion,
		AppBaseURL:       baseURL,
	}
}
