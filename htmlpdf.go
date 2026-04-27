// Package htmlpdf converts HTML to PDF.
//
// # Quick start
//
//	pdf, err := htmlpdf.Generate(htmlBytes, htmlpdf.Options{})
//
// # Backend selection
//
// Two backends are available:
//
//   - BackendNative — pure Go, no external process, ships with this module.
//     Handles a well-defined subset of HTML/CSS suitable for documents,
//     reports, and data exports.
//
//   - BackendChrome — headless Chrome/Chromium via os/exec.
//     Full HTML5/CSS3 fidelity. Requires a browser binary on the host.
//
// With BackendAuto (the default), Chrome is tried first and Native is used
// as the fallback if no browser is found. This gives you the best output
// when Chrome is available and a safe fallback everywhere else.
//
// # Supported HTML (native backend)
//
// Block elements: div, section, article, main, header, footer, p, h1–h6,
// ul, ol, li, blockquote, pre, hr, table, thead, tbody, tr, td, th, figure.
//
// Inline elements: span, strong, b, em, i, code, a, br, img (alt text only).
//
// CSS (inline style= only): font-size, font-weight, font-style, font-family,
// color, text-align. All other properties are silently ignored.
package htmlpdf

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/HeartBeat1608/htmlpdf/chrome"
	"github.com/HeartBeat1608/htmlpdf/document"
	"github.com/HeartBeat1608/htmlpdf/native/layout"
	"github.com/HeartBeat1608/htmlpdf/native/parse"
)

// Generate converts html to a PDF and returns the raw bytes.
//
// opts may be a zero value; all fields have documented defaults.
// Pass a context to enforce a timeout, especially with BackendChrome.
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	pdf, err := htmlpdf.GenerateContext(ctx, html, opts)
func Generate(html []byte, opts Options) ([]byte, error) {
	return GenerateContext(context.Background(), html, opts)
}

// GenerateContext is like Generate but accepts a context for cancellation and
// timeout. The context is forwarded to the Chrome process when that backend
// is active; the native backend does not block on I/O so context cancellation
// has no effect mid-render (but the call returns quickly regardless).
func GenerateContext(ctx context.Context, html []byte, opts Options) ([]byte, error) {
	opts = opts.defaults()

	switch opts.Backend {
	case BackendChrome:
		return renderChrome(ctx, html, opts)
	case BackendNative:
		return renderNative(html, opts)
	case BackendAuto:
		data, err := renderChrome(ctx, html, opts)
		if err != nil {
			// Only fall back on ErrNoBrowser — other Chrome errors surface as-is.
			log.Printf("error in GenerateContent(...): %v", err)
			if errors.Is(err, ErrNoBrowser) {
				return renderNative(html, opts)
			}
			return nil, err
		}
		return data, nil
	default:
		return nil, fmt.Errorf("htmlpdf: unknown backend")
	}
}

// ---------------------------------------------------------------------------
// Chrome backend
// ---------------------------------------------------------------------------

func renderChrome(ctx context.Context, html []byte, opts Options) ([]byte, error) {
	bin, err := chrome.FindBinary(opts.ChromePath)
	if err != nil {
		return nil, err
	}
	if bin == "" {
		return nil, ErrNoBrowser
	}

	r := &chrome.Renderer{BinaryPath: bin}
	data, err := r.Render(ctx, html)
	if err != nil {
		return nil, fmt.Errorf("htmlpdf chrome: %w", err)
	}
	return data, nil
}

// ---------------------------------------------------------------------------
// Native backend
// ---------------------------------------------------------------------------

func renderNative(html []byte, opts Options) ([]byte, error) {
	doc, err := parse.Parse(html)
	if err != nil {
		return nil, fmt.Errorf("htmlpdf native parse: %w", err)
	}

	applyOptions(doc, opts)

	data, err := layout.Render(doc)
	if err != nil {
		return nil, fmt.Errorf("htmlpdf native render: %w", err)
	}
	return data, nil
}

// applyOptions wires Options fields into the document model.
func applyOptions(doc *document.Document, opts Options) {
	w, h := opts.pageDimensions()
	doc.PageWidth = w
	doc.PageHeight = h

	doc.Margins = document.Margins{
		Top:    opts.MarginTopPt,
		Right:  opts.MarginRightPt,
		Bottom: opts.MarginBottomPt,
		Left:   opts.MarginLeftPt,
	}

	if opts.Title != "" {
		doc.Title = opts.Title
	}
}
