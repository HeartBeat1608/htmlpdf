# htmlpdf

`htmlpdf` is a Go package for converting HTML into PDF.

It ships with two rendering backends:

- `BackendChrome` uses headless Chrome or Chromium for high-fidelity HTML/CSS output.
- `BackendNative` uses a pure-Go renderer for document-style HTML without external binaries.

By default, `BackendAuto` tries Chrome first and falls back to the native renderer when no supported browser is available or Chrome cannot render successfully in the current environment.

## Status

This project is pre-`v1.0.0`. The exported API is usable, but minor option and renderer behavior changes may still happen while the package settles into open-source development.

## Install

```bash
go get github.com/HeartBeat1608/htmlpdf
```

## Quick Start

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/HeartBeat1608/htmlpdf"
)

func main() {
	html := []byte(`<html><body><h1>Hello</h1><p>PDF output</p></body></html>`)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pdf, err := htmlpdf.GenerateContext(ctx, html, htmlpdf.Options{
		Backend: htmlpdf.BackendAuto,
		Title:   "Example",
	})
	if err != nil {
		log.Fatal(err)
	}

	_ = pdf
}
```

## Backend Notes

### Chrome backend

- Best output fidelity.
- Requires a Chrome or Chromium binary on the host.
- Does not add `--no-sandbox` by default. If your container or CI environment requires it, set `Options.ChromeDisableSandbox = true`.

### Native backend

- No external process required.
- Designed for reports, invoices, and other document-style HTML.
- Supports a focused subset of HTML and inline CSS.

Supported block elements:

- `div`, `section`, `article`, `main`, `header`, `footer`, `p`
- `h1` to `h6`
- `ul`, `ol`, `li`
- `blockquote`, `pre`, `hr`
- `table`, `thead`, `tbody`, `tr`, `td`, `th`
- `figure`

Supported inline elements:

- `span`, `strong`, `b`, `em`, `i`, `code`, `a`, `br`
- `img` as inline/block fallback content using `alt` text

Supported inline CSS properties:

- `font-size`
- `font-weight`
- `font-style`
- `font-family`
- `color`
- `text-align`
- `background-color`

Unsupported properties are ignored.

## Development

```bash
go test ./...
go vet ./...
```

Chrome integration tests are opt-in and run only when `HTMLPDF_CHROME_BIN` points to a local Chrome or Chromium executable.
