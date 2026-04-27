package chrome

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
)

// FindBinary returns the path of the first usable browser binary.
// It checks override first, then PATH candidates, then platform paths.
func FindBinary(override string) (string, error) {
	if override != "" {
		if _, err := os.Stat(override); err == nil {
			return override, nil
		}
		return "", nil
	}

	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	for _, path := range platformCandidates() {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", nil // not found — caller checks empty string
}

// Renderer renders HTML to PDF using headless Chrome.
type Renderer struct {
	BinaryPath     string // resolved browser binary
	DisableSandbox bool   // adds --no-sandbox when the host requires it
}

// Render serves html on a local HTTP port, launches Chrome with
// --print-to-pdf, reads the output file, and returns the bytes.
func (r *Renderer) Render(ctx context.Context, html []byte) ([]byte, error) {
	// 1. Start an ephemeral local HTTP server to serve the HTML content.
	//    Chrome requires an http:// URL to enable all layout features.
	ln, err := net.Listen("tcp", "127.0.0.1:0") // OS-assigned port
	if err != nil {
		return nil, fmt.Errorf("chrome: listen: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(html)
	})
	srv := &http.Server{Handler: mux}

	go func() {
		_ = srv.Serve(ln)
	}()
	defer ln.Close()
	defer srv.Close()

	url := fmt.Sprintf("http://%s/", ln.Addr())

	// 2. Write PDF to a temp file (Chrome requires a file path, not stdout).
	tmpFile, err := os.CreateTemp("", "htmlpdf-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("chrome: create temp file: %w", err)
	}
	outFile := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("chrome: close temp file: %w", err)
	}
	defer os.Remove(outFile)

	// 3. Build the Chrome argument list.
	args := []string{
		"--headless",
		"--disable-gpu",
		"--disable-software-rasterizer",
		"--disable-dev-shm-usage",
		"--run-all-compositor-stages-before-draw",
		"--print-to-pdf-no-header",
		"--print-to-pdf=" + outFile,
		url,
	}
	if r.DisableSandbox {
		args = append(args[:2], append([]string{"--no-sandbox"}, args[2:]...)...)
	}

	cmd := exec.CommandContext(ctx, r.BinaryPath, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil // suppress Chrome's verbose stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("chrome: process failed: %w", err)
	}

	// 4. Read and return the PDF.
	data, err := os.ReadFile(outFile)
	if err != nil {
		return nil, fmt.Errorf("chrome: reading output: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("chrome: produced empty PDF")
	}
	return data, nil
}
