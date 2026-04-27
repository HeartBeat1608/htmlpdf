package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/HeartBeat1608/htmlpdf"
)

func main() {
	var (
		outputFile = flag.String("out", "example.pdf", "output PDF file")
		backend    = flag.String("backend", "auto", "backend to use: auto, chrome, native")
		title      = flag.String("title", "htmlpdf Example", "PDF document title")
	)
	flag.Parse()

	selectedBackend, err := parseBackend(*backend)
	if err != nil {
		log.Fatal(err)
	}

	html := []byte(sampleHTML(*title))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pdf, err := htmlpdf.GenerateContext(ctx, html, htmlpdf.Options{
		Backend: selectedBackend,
		Title:   *title,
	})
	if err != nil {
		log.Fatalf("render PDF: %v", err)
	}

	if err := os.WriteFile(*outputFile, pdf, 0o644); err != nil {
		log.Fatalf("write %s: %v", *outputFile, err)
	}

	fmt.Printf("Wrote %s using backend=%s (%d bytes)\n", *outputFile, *backend, len(pdf))
}

func parseBackend(raw string) (htmlpdf.Backend, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "auto":
		return htmlpdf.BackendAuto, nil
	case "chrome":
		return htmlpdf.BackendChrome, nil
	case "native":
		return htmlpdf.BackendNative, nil
	default:
		return 0, fmt.Errorf("unknown backend %q; use auto, chrome, or native", raw)
	}
}

func sampleHTML(title string) string {
	return fmt.Sprintf(`<!doctype html>
<html>
  <head>
    <meta charset="utf-8">
    <title>%s</title>
  </head>
  <body style="font-family: Helvetica; color: #222;">
    <h1 style="font-size: 24px;">htmlpdf Example</h1>
    <p>This PDF was generated from a runnable example application.</p>
    <p>Use this as a starting point for your own HTML templates, reports, or invoices.</p>

    <h2 style="font-size: 18px;">Highlights</h2>
    <ul>
      <li>Works with <strong>BackendAuto</strong>, <strong>BackendChrome</strong>, or <strong>BackendNative</strong>.</li>
      <li>Writes the generated PDF to a file.</li>
      <li>Uses a context timeout to keep rendering bounded.</li>
    </ul>

    <h2 style="font-size: 18px;">Example Table</h2>
    <table>
      <thead>
        <tr style="background-color: #eeeeee;">
          <th>Item</th>
          <th>Value</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>Document title</td>
          <td>%s</td>
        </tr>
        <tr>
          <td>Generated at</td>
          <td>%s</td>
        </tr>
      </tbody>
    </table>

    <pre>go run ./examples/basic -backend=native -out=example.pdf</pre>
  </body>
</html>`, title, title, time.Now().Format("2006-01-02 15:04:05"))
}
