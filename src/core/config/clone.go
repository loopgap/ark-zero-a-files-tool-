package config

import "encoding/json"

// CloneAppConfig returns a detached snapshot that can safely be used across goroutines.
func CloneAppConfig(cfg *AppConfig) *AppConfig {
	if cfg == nil {
		return DefaultConfig()
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return NormalizeAppConfig(cfg)
	}

	var cloned AppConfig
	if err := json.Unmarshal(data, &cloned); err != nil {
		return NormalizeAppConfig(cfg)
	}

	return NormalizeAppConfig(&cloned)
}
