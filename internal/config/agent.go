package config

import "time"


type AgentConfig struct {
	PollInterval time.Duration
	ReportInterval time.Duration
	ServerAddress string
}

func NewAgentConfig() *AgentConfig {
	return &AgentConfig{
		PollInterval:   2 * time.Second, 
		ReportInterval: 10 * time.Second, 
		ServerAddress:  "http://localhost:8080",
	}
}
