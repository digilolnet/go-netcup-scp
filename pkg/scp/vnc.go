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
	"image"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"

	"github.com/digilolnet/go-netcup-scp/pkg/rfb"
)

// VNCWebSocketURL builds the wss:// URL for a server's VNC console, embedding a
// fresh access token as the `token` query parameter.
//
// The endpoint is not part of the OpenAPI spec: it is a WebSocket upgrade route
// that tunnels the raw RFB (VNC) protocol over binary WebSocket frames and
// authenticates via the standard SCP access token passed in the query string.
func (c *Client) VNCWebSocketURL(serverID int32) (string, error) {
	token, err := c.validAccessToken()
	if err != nil {
		return "", fmt.Errorf("vnc: %w", err)
	}

	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("vnc: parse base url: %w", err)
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	}
	u.Path = strings.TrimRight(u.Path, "/") + fmt.Sprintf("/api/v1/servers/%d/vnc", serverID)
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// DialVNC opens the VNC console WebSocket for a server and returns a net.Conn
// carrying the raw RFB (VNC) byte stream. Callers bridge this to a native VNC
// client (over TCP) or to a browser noVNC session.
//
// The returned connection stays open until the context is cancelled or either
// side closes; the access token is only checked during the initial handshake,
// so token expiry does not interrupt an established session.
func (c *Client) DialVNC(ctx context.Context, serverID int32) (net.Conn, error) {
	wsURL, err := c.VNCWebSocketURL(serverID)
	if err != nil {
		return nil, err
	}

	ws, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		// A long-lived stream must not run on a client with a request timeout.
		HTTPClient:   &http.Client{},
		Subprotocols: []string{"binary"},
		HTTPHeader: http.Header{
			// The token's allowed-origins names the panel host; the gateway
			// may reject a mismatched Origin.
			"Origin": {c.originURL()},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("vnc: dial websocket: %w", err)
	}
	// RFB framebuffer-update messages can be far larger than the default limit.
	ws.SetReadLimit(-1)

	return websocket.NetConn(ctx, ws, websocket.MessageBinary), nil
}

// validAccessToken returns a currently-valid access token, refreshing once if
// the cached token has expired.
func (c *Client) validAccessToken() (string, error) {
	if tok, err := c.auth.GetAccessToken(); err == nil {
		return tok, nil
	}
	refresh, err := c.auth.GetRefreshToken()
	if err != nil {
		return "", fmt.Errorf("no valid access token: %w", err)
	}
	tok, err := c.auth.RefreshToken(context.Background(), refresh)
	if err != nil {
		return "", fmt.Errorf("refresh access token: %w", err)
	}
	return tok.AccessToken, nil
}

// originURL returns the scheme://host of the API base URL, for use as an Origin
// header on the VNC WebSocket handshake.
func (c *Client) originURL() string {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return c.baseURL
	}
	return u.Scheme + "://" + u.Host
}

// VNCScreenshot connects to the server's VNC console and returns a single full
// framebuffer as an image.
func (c *Client) VNCScreenshot(ctx context.Context, serverID int32) (image.Image, error) {
	conn, err := c.DialVNC(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return rfb.Screenshot(ctx, conn)
}

// VNCWatchFrames streams the server's console via rfb.WatchFrames: it invokes
// onFrame with the current framebuffer no more often than minInterval until
// onFrame reports done or ctx expires.
func (c *Client) VNCWatchFrames(ctx context.Context, serverID int32, minInterval time.Duration, onFrame func(image.Image) (done bool)) error {
	conn, err := c.DialVNC(ctx, serverID)
	if err != nil {
		return err
	}
	defer conn.Close()
	return rfb.WatchFrames(ctx, conn, minInterval, onFrame)
}

// SendVNCChord opens the server's VNC console and sends a single chord: it
// holds the modifier keysyms down, presses+releases key, then releases the
// modifiers. e.g. Ctrl+B = SendVNCChord(ctx, id, key='b', rfb.KeyCtrlLeft).
func (c *Client) SendVNCChord(ctx context.Context, serverID int32, key uint32, modifiers ...uint32) error {
	conn, err := c.DialVNC(ctx, serverID)
	if err != nil {
		return err
	}
	defer conn.Close()
	return rfb.SendChord(conn, key, modifiers...)
}

// SendVNCKeys opens the server's VNC console and sends the given keysyms as
// press+release KeyEvent messages, pausing delay (default ~120ms) between
// each. Printable ASCII runes can be passed as their code point; use the
// rfb.Key* constants for the rest.
func (c *Client) SendVNCKeys(ctx context.Context, serverID int32, delay time.Duration, keys ...uint32) error {
	conn, err := c.DialVNC(ctx, serverID)
	if err != nil {
		return err
	}
	defer conn.Close()
	return rfb.SendKeys(ctx, conn, delay, keys...)
}
