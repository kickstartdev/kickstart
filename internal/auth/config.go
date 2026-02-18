package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	
)

type Config struct {
	Token		string `json:"token"`
	Username	string `json:"username"`
}



func configPath() string {
	
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kickstart", "config.json")
}

func SaveConfig(cfg Config) error {
	err := os.MkdirAll(filepath.Dir(configPath()), 0700)
	if err != nil {
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath(), data, 0600)
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}