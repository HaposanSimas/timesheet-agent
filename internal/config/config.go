// Package config memuat definisi sumber commit dari file JSON.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config adalah isi config.json.
type Config struct {
	Timezone string       `json:"timezone"`
	Author   Author       `json:"author"`
	GitHub   GitHubSource `json:"github"`
	GitLab   GitLabSource `json:"gitlab"`
}

// Author menentukan identitas commit milik kamu di tiap sumber.
type Author struct {
	GitHubLogin string   `json:"github_login"` // filter author di GitHub
	Names       []string `json:"names"`        // untuk cocokkan author_name GitLab
	Emails      []string `json:"emails"`       // untuk cocokkan author_email GitLab
}

// GitHubSource: banyak repo di bawah satu owner/org.
type GitHubSource struct {
	Owner string   `json:"owner"`
	Repos []string `json:"repos"`
}

// GitLabSource: banyak project di satu instance self-hosted.
type GitLabSource struct {
	BaseURL     string   `json:"base_url"`
	Projects    []string `json:"projects"`     // "group/project"
	InsecureTLS bool     `json:"insecure_tls"` // set true kalau CA internal
}

// Load membaca & memvalidasi config JSON.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if c.Timezone == "" {
		c.Timezone = "Asia/Jakarta"
	}
	return &c, nil
}
