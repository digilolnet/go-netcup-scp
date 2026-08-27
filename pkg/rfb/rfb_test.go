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

package rfb

import (
	"context"
	"encoding/binary"
	"image"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeServer speaks just enough RFB 3.8 (security None, Raw encoding, 2x2
// 32bpp framebuffer) to drive the client through the handshake, then hands
// control to script for the update phase.
func fakeServer(t *testing.T, script func(c net.Conn)) net.Conn {
	t.Helper()
	client, server := net.Pipe()
	go func() {
		defer server.Close()
		c := server
		c.Write([]byte("RFB 003.008\n"))
		io.ReadFull(c, make([]byte, 12)) // client version
		c.Write([]byte{1, 1})            // one security type: None
		io.ReadFull(c, make([]byte, 1))  // client choice
		c.Write([]byte{0, 0, 0, 0})      // security result OK
		io.ReadFull(c, make([]byte, 1))  // ClientInit
		init := make([]byte, 24)
		binary.BigEndian.PutUint16(init[0:2], 2) // width
		binary.BigEndian.PutUint16(init[2:4], 2) // height
		init[4], init[5], init[7] = 32, 24, 1    // bpp, depth, trueColor
		binary.BigEndian.PutUint16(init[8:10], 255)
		binary.BigEndian.PutUint16(init[10:12], 255)
		binary.BigEndian.PutUint16(init[12:14], 255)
		init[14], init[15], init[16] = 16, 8, 0
		c.Write(init)
		io.ReadFull(c, make([]byte, 8))  // SetEncodings [Raw]
		io.ReadFull(c, make([]byte, 10)) // first FramebufferUpdateRequest
		script(c)
	}()
	return client
}

// writeUpdate sends one FramebufferUpdate with a single Raw rect.
func writeUpdate(c net.Conn, x, y, w, h int) {
	hdr := make([]byte, 16)
	hdr[0] = 0 // FramebufferUpdate
	binary.BigEndian.PutUint16(hdr[2:4], 1)
	binary.BigEndian.PutUint16(hdr[4:6], uint16(x))
	binary.BigEndian.PutUint16(hdr[6:8], uint16(y))
	binary.BigEndian.PutUint16(hdr[8:10], uint16(w))
	binary.BigEndian.PutUint16(hdr[10:12], uint16(h))
	c.Write(hdr)
	c.Write(make([]byte, w*h*4))
}

func TestWatchFramesSurvivesStaticScreen(t *testing.T) {
	old := watchIdleDeadline
	watchIdleDeadline = 150 * time.Millisecond
	defer func() { watchIdleDeadline = old }()

	conn := fakeServer(t, func(c net.Conn) {
		writeUpdate(c, 0, 0, 2, 2)
		// Static screen: keep draining the client's update requests (net.Pipe
		// writes block until read, unlike buffered TCP) but never answer.
		// The client's rolling read deadline fires repeatedly; WatchFrames
		// must keep waiting instead of failing, until ctx ends it.
		io.Copy(io.Discard, c)
	})
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background()) // no deadline
	go func() {
		time.Sleep(600 * time.Millisecond) // ≈4 idle-deadline periods
		cancel()
	}()

	frames := 0
	err := WatchFrames(ctx, conn, 0, func(image.Image) bool {
		frames++
		return false
	})
	if err != context.Canceled {
		t.Fatalf("want context.Canceled after quiet watch, got %v", err)
	}
	if frames != 1 {
		t.Errorf("want the initial frame only, got %d", frames)
	}
}

func TestReadUpdateRejectsOversizedRect(t *testing.T) {
	conn := fakeServer(t, func(c net.Conn) {
		// 60000x60000 rect on a 2x2 framebuffer: must be rejected before
		// the client allocates the ~14 GiB pixel buffer.
		writeUpdate(c, 0, 0, 60000, 60000)
		time.Sleep(time.Second)
	})
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Screenshot(ctx, conn)
	if err == nil || !strings.Contains(err.Error(), "exceeds framebuffer") {
		t.Fatalf("want oversized-rect rejection, got %v", err)
	}
}
