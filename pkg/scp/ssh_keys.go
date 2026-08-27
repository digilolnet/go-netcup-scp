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

package scp

import (
	"context"
	"fmt"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

// SSHKey represents an SSH public key stored in a user account.
type SSHKey = generated.SSHKey

// ListSSHKeys retrieves all SSH public keys stored for a user.
func (c *Client) ListSSHKeys(ctx context.Context, userID int32) ([]SSHKey, error) {
	resp, err := c.api.GetApiV1UsersUserIdSshKeysWithResponse(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list ssh keys: %w", err)
	}

	return pickBodyVal("list ssh keys", resp, resp.JSON200, resp.HALJSON200, 200)
}

// CreateSSHKey adds a new SSH public key to a user's account.
// The key field must contain a valid OpenSSH public key string.
func (c *Client) CreateSSHKey(ctx context.Context, userID int32, key SSHKey) (*SSHKey, error) {
	resp, err := c.api.PostApiV1UsersUserIdSshKeysWithResponse(ctx, userID, key)
	if err != nil {
		return nil, fmt.Errorf("create ssh key: %w", err)
	}

	return pickBody("create ssh key", resp, resp.JSON201, resp.HALJSON201, 201)
}

// DeleteSSHKey removes an SSH public key from a user's account.
func (c *Client) DeleteSSHKey(ctx context.Context, userID, keyID int32) error {
	resp, err := c.api.DeleteApiV1UsersUserIdSshKeysIdWithResponse(ctx, userID, keyID)
	if err != nil {
		return fmt.Errorf("delete ssh key: %w", err)
	}

	if err := checkResponse(resp, 204); err != nil {
		return fmt.Errorf("delete ssh key: %w", err)
	}

	return nil
}
