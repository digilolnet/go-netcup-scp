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

// Package rfb is a minimal RFB (VNC) client for driving machine consoles
// over any net.Conn: RFB 3.8 handshake, security type None, Raw-encoding
// framebuffer capture and keyboard input. It is transport-agnostic — pair it
// with scp.Client.DialVNC for netcup consoles or any plain VNC server socket.
package rfb

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

// X11 keysyms for common non-printable keys used to drive firmware/boot menus.
// Printable ASCII characters use their ASCII value directly as the keysym.
const (
	KeyEnter     uint32 = 0xFF0D
	KeyEscape    uint32 = 0xFF1B
	KeyBackspace uint32 = 0xFF08
	KeyTab       uint32 = 0xFF09
	KeyUp        uint32 = 0xFF52
	KeyDown      uint32 = 0xFF54
	KeyLeft      uint32 = 0xFF51
	KeyRight     uint32 = 0xFF53
	KeyF1        uint32 = 0xFFBE
	KeyF2        uint32 = 0xFFBF
	KeyF10       uint32 = 0xFFC7
	KeySpace     uint32 = 0x0020
	KeyCtrlLeft  uint32 = 0xFFE3
	KeyAltLeft   uint32 = 0xFFE9
	KeyShiftLeft uint32 = 0xFFE1
)

// SendChord runs the RFB handshake on conn and sends a single chord: it holds
// the modifier keysyms down, presses+releases key, then releases the modifiers.
// e.g. Ctrl+B = SendChord(conn, key='b', KeyCtrlLeft). The caller retains
// ownership of conn.
func SendChord(conn net.Conn, key uint32, modifiers ...uint32) error {
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	if _, err := connect(conn); err != nil {
		return err
	}
	for _, m := range modifiers {
		if err := writeKeyEvent(conn, m, true); err != nil {
			return err
		}
	}
	if err := writeKeyEvent(conn, key, true); err != nil {
		return err
	}
	if err := writeKeyEvent(conn, key, false); err != nil {
		return err
	}
	for i := len(modifiers) - 1; i >= 0; i-- {
		if err := writeKeyEvent(conn, modifiers[i], false); err != nil {
			return err
		}
	}
	return nil
}

// SendKeys runs the RFB handshake on conn and sends the given keysyms as
// press+release KeyEvent messages (RFC 6143 §7.5.4), pausing delay between each.
// A delay of ~120ms is a safe default for firmware/boot menus. Printable ASCII
// runes can be passed as their code point; use the Key* constants for the rest.
// The caller retains ownership of conn.
func SendKeys(ctx context.Context, conn net.Conn, delay time.Duration, keys ...uint32) error {
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	} else {
		_ = conn.SetDeadline(time.Now().Add(60 * time.Second))
	}
	if _, err := connect(conn); err != nil {
		return err
	}
	if delay <= 0 {
		delay = 120 * time.Millisecond
	}
	for _, k := range keys {
		if err := writeKeyEvent(conn, k, true); err != nil {
			return err
		}
		if err := writeKeyEvent(conn, k, false); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil
}

func writeKeyEvent(w io.Writer, keysym uint32, down bool) error {
	msg := make([]byte, 8)
	msg[0] = 4 // KeyEvent
	if down {
		msg[1] = 1
	}
	binary.BigEndian.PutUint32(msg[4:8], keysym)
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("rfb: write key event: %w", err)
	}
	return nil
}
