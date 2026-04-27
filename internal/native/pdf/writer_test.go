package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// parsePDFObjects is a minimal PDF parser that extracts:
// - the PDF version comment
// - all indirect object IDs present
// - the xref offset from startxref
// - whether %%EOF is present
//
// It is deliberately simple - good enough to validate structure, not full render.
func parsePDFObjects(data []byte) (version string, objIDs []int, hasEOF bool, err error) {
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return "", nil, false, fmt.Errorf("empty PDF")
	}

	// Version line
	if !strings.HasPrefix(lines[0], "%PDF-") {
		return "", nil, false, fmt.Errorf("missing PDF header, got %q", lines[0])
	}
	version = strings.TrimPrefix(lines[0], "%PDF-")

	// Scan for object definitions and %%EOF
	for _, line := range lines {
		line = strings.TrimSpace(line)
		var id, gen int
		if n, _ := fmt.Sscanf(line, "%d %d obj", &id, &gen); n == 2 {
			objIDs = append(objIDs, id)
		}
		if line == "%%EOF" {
			hasEOF = true
		}
	}
	return version, objIDs, hasEOF, nil
}

func TestWriterEmptyDocument(t *testing.T) {
	w := NewWriter()
	w.AddPage(A4Width, A4Height)

	data, err := w.Write()
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Write() returned empty bytes")
	}

	version, objIDs, hasEOF, err := parsePDFObjects(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if version != "1.4" {
		t.Errorf("expected PDF 1.4, got %q", version)
	}
	if !hasEOF {
		t.Error("PDF missing EOF marker")
	}
	// Expect at least: fonts (7) + content (1) + page (1) + pages + catalog = 12 minimum
	if len(objIDs) < 12 {
		t.Errorf("expected ≥12 objects, got %d: %v", len(objIDs), objIDs)
	}

	// Verify xref table is present
	if !bytes.Contains(data, []byte("xref")) {
		t.Error("PDF missing xref table")
	}
	if !bytes.Contains(data, []byte("trailer")) {
		t.Error("PDF missing trailer")
	}
	if !bytes.Contains(data, []byte("startxref")) {
		t.Error("PDF missing startxref")
	}
}

func TestWriterMultiPage(t *testing.T) {
	w := NewWriter()
	for i := range 3 {
		pg := w.AddPage(A4Width, A4Height)
		pg.SetFont("F1", 12)
		pg.DrawText(72, 100, fmt.Sprintf("Page %d", i+1))
	}

	data, err := w.Write()
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	// /Count 3 must appear in the Pages dictionary
	if !bytes.Contains(data, []byte("/Count 3")) {
		t.Error("expected /Count 3 in Pages dictionary")
	}
}

func TestWriterTextContent(t *testing.T) {
	w := NewWriter()
	pg := w.AddPage(A4Width, A4Height)
	pg.SetFont("F1", 14)
	pg.DrawText(72, 72, "Hello, PDF!")

	data, err := w.Write()
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	// The escaped text must appear in the content stream
	if !bytes.Contains(data, []byte("(Hello, PDF!)")) {
		t.Error("text 'Hello, PDF!' not found in PDF content stream")
	}
	// Font operator must be present
	if !bytes.Contains(data, []byte("Tf")) {
		t.Error("Tf (font) operator not found")
	}
	// Text-show operator must be used for visible text rendering
	if !bytes.Contains(data, []byte("Tj")) {
		t.Error("Tj (show text) operator not found")
	}
}

func TestWriterRect(t *testing.T) {
	w := NewWriter()
	pg := w.AddPage(A4Width, A4Height)
	pg.DrawRect(50, 50, 200, 100, 0.9, 0.9, 0.9)

	data, err := w.Write()
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if !bytes.Contains(data, []byte("re")) {
		t.Error("rect operator 're' not found in content stream")
	}
	if !bytes.Contains(data, []byte("\nf\n")) {
		t.Error("fill operator 'f' not found in content stream")
	}
}

func TestWriterLine(t *testing.T) {
	w := NewWriter()
	pg := w.AddPage(A4Width, A4Height)
	pg.DrawLine(50, 100, 545, 100, 0.5, 0, 0, 0)

	data, err := w.Write()
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if !bytes.Contains(data, []byte(" m\n")) {
		t.Error("moveto operator 'm' not found")
	}
	if !bytes.Contains(data, []byte(" l\n")) {
		t.Error("lineto operator 'l' not found")
	}
	if !bytes.Contains(data, []byte("\nS\n")) {
		t.Error("stroke operator 'S' not found")
	}
}

func TestWriterFontResources(t *testing.T) {
	w := NewWriter()
	w.AddPage(A4Width, A4Height)

	data, err := w.Write()
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	// All standard font keys must be present in resources
	for _, key := range []string{"F1", "F2", "F3", "F4", "F5", "F6", "F7"} {
		if !bytes.Contains(data, []byte("/"+key)) {
			t.Errorf("font key %q not found in PDF", key)
		}
	}

	// Standard font base names
	for _, name := range []string{"Helvetica", "Times-Roman", "Courier"} {
		if !bytes.Contains(data, []byte("/"+name)) {
			t.Errorf("base font name %q not found in PDF", name)
		}
	}
}

func TestEscapeText(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"hello", "(hello)"},
		{"say (hi)", `(say \(hi\))`},
		{"back\\slash", `(back\\slash)`},
		{"line\nnewline", `(line\nnewline)`},
	}
	for _, c := range cases {
		got := escapeText(c.input)
		if got != c.want {
			t.Errorf("escapeText(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestPageCoordinateFlip(t *testing.T) {
	// In A4 (height=841.89), a doc-space Y of 100 should become PDF Y of 741.89
	pg := newPage(A4Width, A4Height)
	got := pg.toY(100)
	want := A4Height - 100
	if got != want {
		t.Errorf("toY(100) = %.2f, want %.2f", got, want)
	}
}

func TestLetterPageSize(t *testing.T) {
	w := NewWriter()
	w.AddPage(LetterWidth, LetterHeight)
	data, err := w.Write()
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	want := fmt.Sprintf("%.2f %.2f", LetterWidth, LetterHeight)
	if !bytes.Contains(data, []byte(want)) {
		t.Errorf("Letter page dimensions %q not found in MediaBox", want)
	}
}
