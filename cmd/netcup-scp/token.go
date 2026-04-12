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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/digilolnet/go-netcup-scp/pkg/scp/auth"
)

func defaultTokenFilePath() string {
	if envFile := os.Getenv("NETCUP_SCP_TOKEN_FILE"); envFile != "" {
		return envFile
	}
	// Use XDG-style ~/.config on all platforms for predictable, portable placement.
	configDir := filepath.Join(os.Getenv("HOME"), ".config")
	return filepath.Join(configDir, "netcup-scp", "token.json")
}

func loadToken(path string) (*auth.TokenResponse, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read token file %s: %w", path, err)
	}
	var tok auth.TokenResponse
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("parse token file: %w", err)
	}
	return &tok, nil
}

// jwtUserID extracts the numeric user ID from the "id" claim in a JWT access token.
func jwtUserID(accessToken string) (int32, error) {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid JWT format")
	}
	// Add padding so base64 decoding works regardless of token length.
	payload := parts[1]
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}
	data, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return 0, fmt.Errorf("decode JWT payload: %w", err)
	}
	var claims struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &claims); err != nil {
		return 0, fmt.Errorf("parse JWT claims: %w", err)
	}
	if claims.ID == "" {
		return 0, fmt.Errorf("JWT missing 'id' claim")
	}
	id, err := strconv.ParseInt(claims.ID, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse user ID %q: %w", claims.ID, err)
	}
	return int32(id), nil
}

func saveToken(path string, tok *auth.TokenResponse) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
