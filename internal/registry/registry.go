package registry

import (
	_ "embed"
	"encoding/json"
)

//go:embed frameworks.json
var embeddedData []byte

type Framework struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Ecosystem        string   `json:"ecosystem"`
	Type             string   `json:"type,omitempty"`
	Description      string   `json:"description"`
	Tags             []string `json:"tags"`
	NpmPackage       string   `json:"npm_package,omitempty"`
	PackagistPackage string   `json:"packagist_package,omitempty"`
	GithubRepo       string   `json:"github_repo,omitempty"`
	Interactive      bool     `json:"interactive,omitempty"`
}

type registryPayload struct {
	Frameworks []Framework `json:"frameworks"`
}

func Load() ([]Framework, error) {
	var payload registryPayload
	if err := json.Unmarshal(embeddedData, &payload); err != nil {
		return nil, err
	}
	return payload.Frameworks, nil
}
