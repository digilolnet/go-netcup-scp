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
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"

	"github.com/coder/websocket"
	"github.com/spf13/cobra"

	"github.com/digilolnet/go-netcup-scp/pkg/scp"
)

// The stock noVNC client (https://github.com/novnc/noVNC), vendored unmodified.
//
//go:embed all:novnc
var novncFS embed.FS

func newServersVNCCmd() *cobra.Command {
	var tcpAddr, webAddr string
	var noTCP, noWeb, open bool
	cmd := &cobra.Command{
		Use:   "vnc <id>",
		Short: "Open the server's VNC console (native VNC port and/or browser)",
		Long: "Bridge the server's VNC console to a local native VNC port and/or a\n" +
			"browser-based noVNC page. Each incoming connection opens its own console\n" +
			"session. Runs until interrupted with Ctrl+C.",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}

			if noTCP && noWeb {
				return fmt.Errorf("nothing to do: both --no-tcp and --no-web set")
			}

			ctx, stop := signal.NotifyContext(cc.ctx, os.Interrupt)
			defer stop()

			var wg sync.WaitGroup

			if !noTCP {
				ln, err := net.Listen("tcp", tcpAddr)
				if err != nil {
					return fmt.Errorf("listen on %s: %w", tcpAddr, err)
				}
				defer ln.Close()
				fmt.Printf("Native VNC:  vnc://%s   (connect a VNC client here)\n", ln.Addr())
				wg.Add(1)
				go func() {
					defer wg.Done()
					serveVNCTCP(ctx, cc.client, id, ln)
				}()
			}

			if !noWeb {
				httpLn, err := net.Listen("tcp", webAddr)
				if err != nil {
					return fmt.Errorf("listen on %s: %w", webAddr, err)
				}
				webURL := fmt.Sprintf("http://%s/", httpLn.Addr())
				fmt.Printf("Browser:     %s\n", webURL)

				srv := &http.Server{Handler: vncWebHandler(ctx, cc.client, id)}
				wg.Add(1)
				go func() {
					defer wg.Done()
					_ = srv.Serve(httpLn)
				}()
				go func() {
					<-ctx.Done()
					_ = srv.Close()
				}()
				if open {
					if err := openBrowser(webURL); err != nil {
						fmt.Fprintf(os.Stderr, "could not open browser: %v\n", err)
					}
				}
			}

			fmt.Println("Press Ctrl+C to stop.")
			<-ctx.Done()
			wg.Wait()
			return nil
		},
	}
	cmd.Flags().StringVar(&tcpAddr, "listen", "127.0.0.1:5900", "address for the native VNC TCP bridge")
	cmd.Flags().StringVar(&webAddr, "web-listen", "127.0.0.1:0", "address for the browser noVNC server (port 0 = random)")
	cmd.Flags().BoolVar(&noTCP, "no-tcp", false, "disable the native VNC port")
	cmd.Flags().BoolVar(&noWeb, "no-web", false, "disable the browser noVNC server")
	cmd.Flags().BoolVar(&open, "open", false, "open the browser console automatically")
	return cmd
}

// serveVNCTCP accepts native VNC clients and bridges each to its own console
// WebSocket session.
func serveVNCTCP(ctx context.Context, client *scp.Client, id int32, ln net.Listener) {
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed
		}
		go func() {
			defer conn.Close()
			remote, err := client.DialVNC(ctx, id)
			if err != nil {
				fmt.Fprintf(os.Stderr, "vnc: %v\n", err)
				return
			}
			defer remote.Close()
			bridgeConns(conn, remote)
		}()
	}
}

// vncWebHandler serves the noVNC page and a websockify endpoint that bridges the
// browser to a console WebSocket session.
func vncWebHandler(ctx context.Context, client *scp.Client, id int32) http.Handler {
	mux := http.NewServeMux()
	novnc, err := fs.Sub(novncFS, "novnc")
	if err != nil {
		panic(err) // embedded tree is fixed at build time
	}
	fileServer := http.FileServer(http.FS(novnc))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/vnc.html?autoconnect=true&resize=scale&reconnect=true&reconnect_delay=2000", http.StatusFound)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
	mux.HandleFunc("/websockify", func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols: []string{"binary"},
		})
		if err != nil {
			return
		}
		ws.SetReadLimit(-1)
		browser := websocket.NetConn(r.Context(), ws, websocket.MessageBinary)

		remote, err := client.DialVNC(ctx, id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "console dial failed: %v\n", err)
			ws.Close(websocket.StatusInternalError, "console dial failed")
			return
		}
		defer remote.Close()

		bridgeConns(browser, remote)
		ws.Close(websocket.StatusNormalClosure, "")
	})
	return mux
}

// bridgeConns copies bytes in both directions until either side closes.
func bridgeConns(a, b net.Conn) {
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(a, b); done <- struct{}{} }()
	go func() { _, _ = io.Copy(b, a); done <- struct{}{} }()
	<-done
}

// openBrowser launches the platform's default browser for url.
func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd, args = "cmd", []string{"/c", "start"}
	default:
		cmd = "xdg-open"
	}
	return exec.Command(cmd, append(args, url)...).Start()
}
