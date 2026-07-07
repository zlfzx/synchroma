package config

import (
	"encoding/json"
	"fmt"
	"os"
	"synchroma/pkg/models"
)

type Config struct {
	Profiles map[string]Profile `json:"profiles"`
}

type Profile struct {
	Source models.DataSource `json:"source"`
	Target models.DataSource `json:"target"`
}

// LoadConfig reads the config file and returns the specified profile.
func LoadConfig(configPath, profileName string) (models.DataSource, models.DataSource, error) {
	file, err := os.ReadFile(configPath)
	if err != nil {
		return models.DataSource{}, models.DataSource{}, fmt.Errorf("could not read config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		return models.DataSource{}, models.DataSource{}, fmt.Errorf("error parsing config file: %v", err)
	}

	if profileName == "" {
		profileName = "default"
	}

	profile, ok := config.Profiles[profileName]
	if !ok {
		return models.DataSource{}, models.DataSource{}, fmt.Errorf("profile '%s' not found in config", profileName)
	}

	return profile.Source, profile.Target, nil
}

// SaveConfig saves a profile to the config file.
func SaveConfig(configPath, profileName string, sourceCfg, targetCfg models.DataSource) error {
	var config Config
	file, err := os.ReadFile(configPath)
	if err == nil {
		json.Unmarshal(file, &config)
	}
	
	if config.Profiles == nil {
		config.Profiles = make(map[string]Profile)
	}

	if profileName == "" {
		profileName = "default"
	}

	config.Profiles[profileName] = Profile{
		Source: sourceCfg,
		Target: targetCfg,
	}

	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode config: %v", err)
	}

	if err := os.WriteFile(configPath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}
