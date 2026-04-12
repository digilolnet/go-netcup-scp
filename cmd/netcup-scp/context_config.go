// Copyright 2026 Laurynas Četyrkinas <laurynas@digilol.net>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ContextEntry describes a named netcup account context.
type ContextEntry struct {
	Name      string `json:"name"`
	TokenFile string `json:"token_file"`
}

// Config is the top-level CLI configuration file (multi-account contexts).
type Config struct {
	CurrentContext string         `json:"current_context,omitempty"`
	Contexts       []ContextEntry `json:"contexts,omitempty"`
}

func defaultConfigFilePath() string {
	if v := os.Getenv("NETCUP_SCP_CONFIG"); v != "" {
		return v
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "netcup-scp", "config.json")
}

// defaultContextTokenFile returns the default token file path for a named context.
func defaultContextTokenFile(name string) string {
	return filepath.Join(os.Getenv("HOME"), ".config", "netcup-scp", "contexts", name+".json")
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func saveConfig(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (cfg *Config) findContext(name string) *ContextEntry {
	for i := range cfg.Contexts {
		if cfg.Contexts[i].Name == name {
			return &cfg.Contexts[i]
		}
	}
	return nil
}
