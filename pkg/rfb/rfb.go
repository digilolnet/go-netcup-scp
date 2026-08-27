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
	"fmt"
	"image"
	"image/color"
	"io"
	"net"
	"os"
	"time"
)

// pixelFormat mirrors the RFB PIXEL_FORMAT structure (RFC 6143 §7.4).
type pixelFormat struct {
	bitsPerPixel uint8
	depth        uint8
	bigEndian    uint8
	trueColor    uint8
	redMax       uint16
	greenMax     uint16
	blueMax      uint16
	redShift     uint8
	greenShift   uint8
	blueShift    uint8
}

// rfbConn is a minimal RFB (VNC) client over an already-connected stream: it
// performs the RFB 3.8 handshake, advertises Raw encoding only, and decodes
// FramebufferUpdate messages into a persistent image.
type rfbConn struct {
	conn   net.Conn
	width  int
	height int
	pf     pixelFormat
	bpp    int
	order  binary.ByteOrder
	img    *image.RGBA
}

// connect runs the RFB 3.8 handshake (ProtocolVersion, None security,
// ClientInit, ServerInit) and selects Raw encoding.
func connect(conn net.Conn) (*rfbConn, error) {
	ver := make([]byte, 12)
	if _, err := io.ReadFull(conn, ver); err != nil {
		return nil, fmt.Errorf("rfb: read protocol version: %w", err)
	}
	if _, err := conn.Write([]byte("RFB 003.008\n")); err != nil {
		return nil, fmt.Errorf("rfb: write protocol version: %w", err)
	}

	nSec := make([]byte, 1)
	if _, err := io.ReadFull(conn, nSec); err != nil {
		return nil, fmt.Errorf("rfb: read security count: %w", err)
	}
	if nSec[0] == 0 {
		return nil, fmt.Errorf("rfb: server rejected connection during security handshake")
	}
	secTypes := make([]byte, nSec[0])
	if _, err := io.ReadFull(conn, secTypes); err != nil {
		return nil, fmt.Errorf("rfb: read security types: %w", err)
	}
	hasNone := false
	for _, t := range secTypes {
		if t == 1 {
			hasNone = true
		}
	}
	if !hasNone {
		return nil, fmt.Errorf("rfb: server requires auth (types %v); only None supported", secTypes)
	}
	if _, err := conn.Write([]byte{1}); err != nil {
		return nil, fmt.Errorf("rfb: select security type: %w", err)
	}
	secResult := make([]byte, 4)
	if _, err := io.ReadFull(conn, secResult); err != nil {
		return nil, fmt.Errorf("rfb: read security result: %w", err)
	}
	if binary.BigEndian.Uint32(secResult) != 0 {
		return nil, fmt.Errorf("rfb: security handshake failed")
	}

	if _, err := conn.Write([]byte{1}); err != nil { // ClientInit (shared)
		return nil, fmt.Errorf("rfb: client init: %w", err)
	}

	head := make([]byte, 24)
	if _, err := io.ReadFull(conn, head); err != nil {
		return nil, fmt.Errorf("rfb: read server init: %w", err)
	}
	width := int(binary.BigEndian.Uint16(head[0:2]))
	height := int(binary.BigEndian.Uint16(head[2:4]))
	pf := pixelFormat{
		bitsPerPixel: head[4], depth: head[5], bigEndian: head[6], trueColor: head[7],
		redMax:   binary.BigEndian.Uint16(head[8:10]),
		greenMax: binary.BigEndian.Uint16(head[10:12]),
		blueMax:  binary.BigEndian.Uint16(head[12:14]),
		redShift: head[14], greenShift: head[15], blueShift: head[16],
	}
	nameLen := binary.BigEndian.Uint32(head[20:24])
	if nameLen > 0 {
		if _, err := io.CopyN(io.Discard, conn, int64(nameLen)); err != nil {
			return nil, fmt.Errorf("rfb: read server name: %w", err)
		}
	}
	if pf.trueColor == 0 || (pf.bitsPerPixel != 32 && pf.bitsPerPixel != 16 && pf.bitsPerPixel != 8) {
		return nil, fmt.Errorf("rfb: unsupported pixel format (bpp=%d trueColor=%d)", pf.bitsPerPixel, pf.trueColor)
	}
	if width == 0 || height == 0 || width > 8192 || height > 8192 {
		return nil, fmt.Errorf("rfb: implausible framebuffer size %dx%d", width, height)
	}

	if _, err := conn.Write([]byte{2, 0, 0, 1, 0, 0, 0, 0}); err != nil { // SetEncodings [Raw]
		return nil, fmt.Errorf("rfb: set encodings: %w", err)
	}

	order := binary.ByteOrder(binary.LittleEndian)
	if pf.bigEndian != 0 {
		order = binary.BigEndian
	}
	return &rfbConn{
		conn: conn, width: width, height: height, pf: pf,
		bpp: int(pf.bitsPerPixel) / 8, order: order,
		img: image.NewRGBA(image.Rect(0, 0, width, height)),
	}, nil
}

// requestUpdate sends a FramebufferUpdateRequest for the whole screen.
func (r *rfbConn) requestUpdate(incremental bool) error {
	req := make([]byte, 10)
	req[0] = 3
	if incremental {
		req[1] = 1
	}
	binary.BigEndian.PutUint16(req[6:8], uint16(r.width))
	binary.BigEndian.PutUint16(req[8:10], uint16(r.height))
	_, err := r.conn.Write(req)
	return err
}

// readUpdate reads one FramebufferUpdate (skipping other server messages),
// applies its Raw rectangles to the image, and returns the pixels covered.
func (r *rfbConn) readUpdate() (int, error) {
	for {
		msgType := make([]byte, 1)
		if _, err := io.ReadFull(r.conn, msgType); err != nil {
			return 0, fmt.Errorf("rfb: read message type: %w", err)
		}
		if msgType[0] != 0 {
			if err := skipServerMessage(r.conn, msgType[0]); err != nil {
				return 0, err
			}
			continue
		}
		hdr := make([]byte, 3)
		if _, err := io.ReadFull(r.conn, hdr); err != nil {
			return 0, fmt.Errorf("rfb: read update header: %w", err)
		}
		nRects := int(binary.BigEndian.Uint16(hdr[1:3]))
		covered := 0
		for i := 0; i < nRects; i++ {
			rh := make([]byte, 12)
			if _, err := io.ReadFull(r.conn, rh); err != nil {
				return covered, fmt.Errorf("rfb: read rect header: %w", err)
			}
			rx := int(binary.BigEndian.Uint16(rh[0:2]))
			ry := int(binary.BigEndian.Uint16(rh[2:4]))
			rw := int(binary.BigEndian.Uint16(rh[4:6]))
			rhh := int(binary.BigEndian.Uint16(rh[6:8]))
			enc := int32(binary.BigEndian.Uint32(rh[8:12]))
			if enc != 0 {
				return covered, fmt.Errorf("rfb: server used unsupported encoding %d", enc)
			}
			buf := make([]byte, rw*rhh*r.bpp)
			if _, err := io.ReadFull(r.conn, buf); err != nil {
				return covered, fmt.Errorf("rfb: read rect pixels: %w", err)
			}
			// rx/ry/rw/rhh are one rectangle-geometry tuple; kept inline.
			r.decodeRaw(rx, ry, rw, rhh, buf)
			covered += rw * rhh
		}
		return covered, nil
	}
}

// Screenshot runs the RFB handshake on conn and returns a single full
// framebuffer as an image. The caller retains ownership of conn.
func Screenshot(ctx context.Context, conn net.Conn) (image.Image, error) {
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	} else {
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	}
	r, err := connect(conn)
	if err != nil {
		return nil, err
	}
	if err := r.requestUpdate(false); err != nil {
		return nil, fmt.Errorf("rfb: framebuffer request: %w", err)
	}
	target := r.width * r.height
	for covered := 0; covered < target; {
		n, err := r.readUpdate()
		if err != nil {
			return nil, err
		}
		covered += n
	}
	return r.img, nil
}

// WatchFrames streams the framebuffer: it runs the handshake on conn, keeps
// requesting incremental updates and, no more often than minInterval, invokes
// onFrame with the current image. It returns nil when onFrame reports done, or
// ctx's error on timeout. This reliably captures transient content (e.g. a log
// line that scrolls past) that a single screenshot would miss. The caller
// retains ownership of conn.
func WatchFrames(
	ctx context.Context,
	conn net.Conn,
	minInterval time.Duration,
	onFrame func(image.Image) (done bool),
) error {
	dbg := os.Getenv("FP_DEBUG") != ""
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	r, err := connect(conn)
	if err != nil {
		return err
	}
	if dbg {
		fmt.Fprintf(os.Stderr, "[vnc] connected %dx%d bpp=%d\n", r.width, r.height, r.pf.bitsPerPixel)
	}
	if err := r.requestUpdate(false); err != nil {
		return err
	}

	var lastFrame time.Time
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if dl, ok := ctx.Deadline(); ok {
			_ = conn.SetDeadline(dl)
		} else {
			_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
		}
		n, err := r.readUpdate()
		if err != nil {
			return err
		}
		if dbg {
			fmt.Fprintf(os.Stderr, "[vnc] readUpdate covered=%d\n", n)
		}
		if err := r.requestUpdate(true); err != nil {
			return err
		}
		if time.Since(lastFrame) >= minInterval {
			snap := *r.img // shallow copy of header; pixels shared but consumed synchronously
			if onFrame(&snap) {
				return nil
			}
			lastFrame = time.Now()
		}
	}
}

func skipServerMessage(conn io.Reader, msgType byte) error {
	switch msgType {
	case 1: // SetColourMapEntries
		h := make([]byte, 5)
		if _, err := io.ReadFull(conn, h); err != nil {
			return err
		}
		n := int(binary.BigEndian.Uint16(h[3:5]))
		_, err := io.CopyN(io.Discard, conn, int64(n)*6)
		return err
	case 2: // Bell
		return nil
	case 3: // ServerCutText
		h := make([]byte, 7)
		if _, err := io.ReadFull(conn, h); err != nil {
			return err
		}
		n := int(binary.BigEndian.Uint32(h[3:7]))
		_, err := io.CopyN(io.Discard, conn, int64(n))
		return err
	default:
		return fmt.Errorf("rfb: unknown server message type %d", msgType)
	}
}

func (r *rfbConn) decodeRaw(x0, y0, w, h int, buf []byte) {
	scale := func(v uint32, max uint16) uint8 {
		if max == 0 {
			return 0
		}
		return uint8(uint32(v) * 255 / uint32(max))
	}
	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var raw uint32
			switch r.bpp {
			case 4:
				raw = r.order.Uint32(buf[i : i+4])
			case 2:
				raw = uint32(r.order.Uint16(buf[i : i+2]))
			case 1:
				raw = uint32(buf[i])
			}
			i += r.bpp
			red := scale((raw>>r.pf.redShift)&uint32(r.pf.redMax), r.pf.redMax)
			green := scale((raw>>r.pf.greenShift)&uint32(r.pf.greenMax), r.pf.greenMax)
			blue := scale((raw>>r.pf.blueShift)&uint32(r.pf.blueMax), r.pf.blueMax)
			r.img.Set(x0+x, y0+y, color.RGBA{R: red, G: green, B: blue, A: 255})
		}
	}
}
