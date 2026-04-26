package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	webview "github.com/webview/webview_go"
)

//go:embed assets/*
var embeddedAssets embed.FS

func main() {
	var (
		assetsDir string
		debug     bool
		width     int
		height    int
	)

	flag.StringVar(&assetsDir, "assets", "", "directory containing index.html, wasm_exec.js and main.wasm")
	flag.BoolVar(&debug, "debug", false, "enable webview developer tools when supported")
	flag.IntVar(&width, "width", 1440, "window width")
	flag.IntVar(&height, "height", 960, "window height")
	flag.Parse()

	assetFS, err := resolveAssetsFS(assetsDir)
	if err != nil {
		log.Fatal(err)
	}

	baseURL, shutdown, err := startServer(assetFS)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = shutdown(ctx)
	}()

	w := webview.New(debug)
	defer w.Destroy()
	w.SetTitle("gu Node Graph Desktop")
	w.SetSize(width, height, webview.HintNone)
	w.Navigate(baseURL)
	w.Run()
}

func resolveAssetsFS(assetsDir string) (fs.FS, error) {
	if assetsDir == "" {
		sub, err := fs.Sub(embeddedAssets, "assets")
		if err != nil {
			return nil, err
		}
		if err := ensureAssetsFS(sub, "embedded assets"); err != nil {
			return nil, err
		}
		return sub, nil
	}

	assetsDir, err := filepath.Abs(assetsDir)
	if err != nil {
		return nil, err
	}
	assetFS := os.DirFS(assetsDir)
	if err := ensureAssetsFS(assetFS, assetsDir); err != nil {
		return nil, err
	}
	return assetFS, nil
}

func ensureAssetsFS(assetFS fs.FS, label string) error {
	required := []string{"index.html", "main.wasm", "wasm_exec.js"}
	var missing []string
	for _, name := range required {
		if _, err := fs.Stat(assetFS, name); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				missing = append(missing, name)
				continue
			}
			return err
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("missing desktop assets in %s: %s", label, strings.Join(missing, ", "))
}

func startServer(assetFS fs.FS) (string, func(context.Context) error, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}

	mux := http.NewServeMux()
	fileServer := http.FileServer(http.FS(assetFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".wasm") {
			w.Header().Set("Content-Type", "application/wasm")
		}
		if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Cache-Control", "no-cache")
		}
		fileServer.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("desktop asset server error: %v", err)
		}
	}()

	return fmt.Sprintf("http://%s/index.html", ln.Addr().String()), srv.Shutdown, nil
}

func init() {
	if runtime.GOOS == "windows" {
		log.SetFlags(0)
	}
}
