package htmlpdf_test

import (
	"bytes"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/HeartBeat1608/htmlpdf"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------
const (
	FixtureInvoiceHTML   = "fixtures/invoice.html"
	FixtureReportHTML    = "fixtures/report.html"
	FixtureEdgeCasesHTML = "fixtures/edge_cases.html"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read fixture %q: %v", name, err)
	}
	return data
}

// pdfPageCount extracts /Count N from the PDF's Pages dictionary.
func pdfPageCount(data []byte) int {
	m := regexp.MustCompile(`/Count (\d+)`).FindSubmatch(data)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(string(m[1]))
	return n
}

// pdfHasText reports whether the literal string appears anywhere in the PDF
// content streams (escaped as a PDF string token).
func pdfHasText(data []byte, text string) bool {
	return bytes.Contains(data, []byte("("+text)) ||
		bytes.Contains(data, []byte(text))
}

// validatePDFStructure checks the minimum required structural elements.
func validatePDFStructure(t *testing.T, data []byte, label string) {
	t.Helper()
	checks := map[string]bool{
		"header":    bytes.HasPrefix(data, []byte("%PDF-1.4")),
		"xref":      bytes.Contains(data, []byte("\nxref\n")),
		"trailer":   bytes.Contains(data, []byte("\ntrailer\n")),
		"startxref": bytes.Contains(data, []byte("\nstartxref\n")),
		"EOF":       bytes.Contains(data, []byte("%%EOF")),
		"Catalog":   bytes.Contains(data, []byte("/Type /Catalog")),
		"Pages":     bytes.Contains(data, []byte("/Type /Pages")),
		"Page":      bytes.Contains(data, []byte("/Type /Page")),
		"BT":        bytes.Contains(data, []byte("BT\n")),
	}
	for name, ok := range checks {
		if !ok {
			t.Errorf("[%s] PDF structural check failed: %s", label, name)
		}
	}
}

// validatePDFEnvelope checks portable PDF structure without assuming any
// backend-specific content stream representation.
func validatePDFEnvelope(t *testing.T, data []byte, label string) {
	t.Helper()
	checks := map[string]bool{
		"header":    bytes.HasPrefix(data, []byte("%PDF-")),
		"xref":      bytes.Contains(data, []byte("\nxref\n")),
		"trailer":   bytes.Contains(data, []byte("\ntrailer\n")),
		"startxref": bytes.Contains(data, []byte("\nstartxref\n")),
		"EOF":       bytes.Contains(data, []byte("%%EOF")),
		"Catalog":   bytes.Contains(data, []byte("/Type /Catalog")),
		"Pages":     bytes.Contains(data, []byte("/Type /Pages")),
		"Page":      bytes.Contains(data, []byte("/Type /Page")),
	}
	for name, ok := range checks {
		if !ok {
			t.Errorf("[%s] PDF envelope check failed: %s", label, name)
		}
	}
}

// ---------------------------------------------------------------------------
// Options tests
// ---------------------------------------------------------------------------

func TestOptionsDefaults(t *testing.T) {
	html := []byte("<p>test</p>")
	data, err := htmlpdf.Generate(html, htmlpdf.Options{})
	if err != nil {
		t.Fatalf("Generate with zero Options: %v", err)
	}
	validatePDFEnvelope(t, data, "defaults")
}

func TestOptionsNativeBackendExplicit(t *testing.T) {
	html := []byte("<h1>Hello</h1><p>World</p>")
	data, err := htmlpdf.Generate(html, htmlpdf.Options{Backend: htmlpdf.BackendNative})
	if err != nil {
		t.Fatalf("BackendNative: %v", err)
	}
	validatePDFStructure(t, data, "native-explicit")
}

func TestOptionsLetterSize(t *testing.T) {
	html := []byte("<p>Letter page</p>")
	data, err := htmlpdf.Generate(html, htmlpdf.Options{
		Backend:  htmlpdf.BackendNative,
		PageSize: htmlpdf.PageLetter,
	})
	if err != nil {
		t.Fatalf("Letter size: %v", err)
	}
	// Letter MediaBox: 612.00 x 792.00
	if !bytes.Contains(data, []byte("612.00 792.00")) {
		t.Error("Letter page dimensions not found in MediaBox")
	}
}

func TestOptionsLandscape(t *testing.T) {
	html := []byte("<p>Landscape page</p>")
	data, err := htmlpdf.Generate(html, htmlpdf.Options{
		Backend:     htmlpdf.BackendNative,
		Orientation: htmlpdf.Landscape,
	})
	if err != nil {
		t.Fatalf("Landscape: %v", err)
	}
	// A4 landscape: 841.89 x 595.28 — width > height in the MediaBox
	if !bytes.Contains(data, []byte("841.89 595.28")) {
		t.Error("A4 Landscape dimensions not found in MediaBox")
	}
}

func TestOptionsCustomMargins(t *testing.T) {
	html := []byte("<p>Narrow margins</p>")
	_, err := htmlpdf.Generate(html, htmlpdf.Options{
		Backend:        htmlpdf.BackendNative,
		MarginTopPt:    36,
		MarginRightPt:  36,
		MarginBottomPt: 36,
		MarginLeftPt:   36,
	})
	if err != nil {
		t.Fatalf("custom margins: %v", err)
	}
}

func TestOptionsTitleOverride(t *testing.T) {
	html := []byte("<html><head><title>HTML Title</title></head><body><p>text</p></body></html>")
	// Title override wins over the HTML <title>
	_, err := htmlpdf.Generate(html, htmlpdf.Options{
		Backend: htmlpdf.BackendNative,
		Title:   "Override Title",
	})
	if err != nil {
		t.Fatalf("title override: %v", err)
	}
}

func TestChromeBackendMissingBrowser(t *testing.T) {
	html := []byte("<p>test</p>")
	_, err := htmlpdf.Generate(html, htmlpdf.Options{
		Backend:    htmlpdf.BackendChrome,
		ChromePath: "/nonexistent/chrome",
	})
	if err == nil {
		t.Error("expected error for missing chrome binary, got nil")
	}
}

func TestAutoBackendFallsBackToNative(t *testing.T) {
	// Force chrome path to something that doesn't exist so auto falls back.
	html := []byte("<p>auto fallback</p>")
	data, err := htmlpdf.Generate(html, htmlpdf.Options{
		Backend:    htmlpdf.BackendAuto,
		ChromePath: "/nonexistent/chrome",
	})
	if err != nil {
		t.Fatalf("auto fallback to native failed: %v", err)
	}
	validatePDFStructure(t, data, "auto-fallback")
}

// ---------------------------------------------------------------------------
// Fixture: invoice
// ---------------------------------------------------------------------------

func TestInvoiceFixture(t *testing.T) {
	html := readFixture(t, FixtureInvoiceHTML)
	data, err := htmlpdf.Generate(html, htmlpdf.Options{Backend: htmlpdf.BackendNative})
	if err != nil {
		t.Fatalf("invoice render: %v", err)
	}

	validatePDFStructure(t, data, "invoice")

	if len(data) < 5000 {
		t.Errorf("invoice PDF suspiciously small: %d bytes", len(data))
	}

	// Key content checks
	for _, want := range []string{"INVOICE", "Acme", "1042", "7,800"} {
		if !pdfHasText(data, want) {
			t.Errorf("invoice: expected %q in PDF output", want)
		}
	}

	t.Logf("invoice: %d bytes, %d pages", len(data), pdfPageCount(data))
}

func TestInvoiceHasTableStructure(t *testing.T) {
	html := readFixture(t, FixtureInvoiceHTML)
	data, err := htmlpdf.Generate(html, htmlpdf.Options{Backend: htmlpdf.BackendNative})
	if err != nil {
		t.Fatalf("%v", err)
	}
	// Tables emit stroked lines for borders
	if !bytes.Contains(data, []byte("\nS\n")) {
		t.Error("invoice: no stroke operators — table borders missing")
	}
}

func TestListMarkersDoNotRenderRawUTF8Bytes(t *testing.T) {
	html := []byte("<ul><li>alpha</li><li>beta</li></ul>")
	data, err := htmlpdf.Generate(html, htmlpdf.Options{Backend: htmlpdf.BackendNative})
	if err != nil {
		t.Fatalf("list render: %v", err)
	}
	if bytes.Contains(data, []byte("342200242")) {
		t.Error("list marker rendered as raw UTF-8 byte digits instead of a visible bullet shape")
	}
}

// ---------------------------------------------------------------------------
// Fixture: technical report
// ---------------------------------------------------------------------------

func TestReportFixture(t *testing.T) {
	html := readFixture(t, FixtureReportHTML)
	data, err := htmlpdf.Generate(html, htmlpdf.Options{Backend: htmlpdf.BackendNative})
	if err != nil {
		t.Fatalf("report render: %v", err)
	}

	validatePDFStructure(t, data, "report")

	pages := pdfPageCount(data)
	if pages < 2 {
		t.Errorf("report: expected ≥2 pages for long document, got %d", pages)
	}

	for _, want := range []string{"Cache", "Redis", "PostgreSQL"} {
		if !pdfHasText(data, want) {
			t.Errorf("report: expected %q in output", want)
		}
	}

	t.Logf("report: %d bytes, %d pages", len(data), pages)
}

func TestReportHasPreBackground(t *testing.T) {
	html := readFixture(t, FixtureReportHTML)
	data, err := htmlpdf.Generate(html, htmlpdf.Options{Backend: htmlpdf.BackendNative})
	if err != nil {
		t.Fatalf("%v", err)
	}
	// Pre blocks draw a filled rect — 'f' operator
	if !bytes.Contains(data, []byte("\nf\n")) {
		t.Error("report: no fill operators — pre block backgrounds missing")
	}
}

func TestReportHasMonoFont(t *testing.T) {
	html := readFixture(t, FixtureReportHTML)
	data, err := htmlpdf.Generate(html, htmlpdf.Options{Backend: htmlpdf.BackendNative})
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !bytes.Contains(data, []byte("/Courier")) {
		t.Error("report: Courier font not referenced — code blocks not monospaced")
	}
}

// ---------------------------------------------------------------------------
// Fixture: edge cases
// ---------------------------------------------------------------------------

func TestEdgeCasesFixture(t *testing.T) {
	html := readFixture(t, FixtureEdgeCasesHTML)
	data, err := htmlpdf.Generate(html, htmlpdf.Options{Backend: htmlpdf.BackendNative})
	if err != nil {
		t.Fatalf("edge_cases render: %v", err)
	}

	validatePDFStructure(t, data, "edge_cases")
	t.Logf("edge_cases: %d bytes, %d pages", len(data), pdfPageCount(data))
}

func TestEdgeCasesScriptIgnored(t *testing.T) {
	html := readFixture(t, FixtureEdgeCasesHTML)
	data, err := htmlpdf.Generate(html, htmlpdf.Options{Backend: htmlpdf.BackendNative})
	if err != nil {
		t.Fatalf("%v", err)
	}
	if bytes.Contains(data, []byte("must not appear")) {
		t.Error("edge_cases: script/style content leaked into PDF")
	}
}

func TestEdgeCasesMultiPage(t *testing.T) {
	html := readFixture(t, FixtureEdgeCasesHTML)
	data, err := htmlpdf.Generate(html, htmlpdf.Options{Backend: htmlpdf.BackendNative})
	if err != nil {
		t.Fatalf("%v", err)
	}
	pages := pdfPageCount(data)
	if pages < 2 {
		t.Errorf("edge_cases: expected ≥2 pages (section 12 overflows), got %d", pages)
	}
}

func TestEdgeCasesUnicodeDoesNotCrash(t *testing.T) {
	html := []byte(`<p>café naïve résumé £ € ¥ em—dash ellipsis… "smart quotes"</p>`)
	_, err := htmlpdf.Generate(html, htmlpdf.Options{Backend: htmlpdf.BackendNative})
	if err != nil {
		t.Fatalf("unicode content caused error: %v", err)
	}
}

func TestEdgeCasesLongWordDoesNotCrash(t *testing.T) {
	longWord := strings.Repeat("abcdefghij", 20) // 200 chars, wider than any page
	html := []byte("<p>Before " + longWord + " after</p>")
	_, err := htmlpdf.Generate(html, htmlpdf.Options{Backend: htmlpdf.BackendNative})
	if err != nil {
		t.Fatalf("long word caused error: %v", err)
	}
}

func TestEdgeCasesEmptyBody(t *testing.T) {
	for _, html := range []string{
		"",
		"<html><body></body></html>",
		"<p></p><div></div>",
	} {
		data, err := htmlpdf.Generate([]byte(html), htmlpdf.Options{Backend: htmlpdf.BackendNative})
		if err != nil {
			t.Errorf("empty body %q: unexpected error: %v", html, err)
			continue
		}
		if !bytes.HasPrefix(data, []byte("%PDF")) {
			t.Errorf("empty body %q: output is not a PDF", html)
		}
	}
}

// ---------------------------------------------------------------------------
// Size and performance smoke tests
// ---------------------------------------------------------------------------

func TestOutputSizeReasonable(t *testing.T) {
	fixtures := []struct {
		name    string
		minSize int
		maxSize int
	}{
		{FixtureInvoiceHTML, 3_000, 200_000},
		{FixtureReportHTML, 5_000, 500_000},
		{FixtureEdgeCasesHTML, 3_000, 500_000},
	}
	for _, f := range fixtures {
		html := readFixture(t, f.name)
		data, err := htmlpdf.Generate(html, htmlpdf.Options{Backend: htmlpdf.BackendNative})
		if err != nil {
			t.Errorf("%s: render error: %v", f.name, err)
			continue
		}
		if len(data) < f.minSize {
			t.Errorf("%s: PDF too small (%d bytes, want ≥%d)", f.name, len(data), f.minSize)
		}
		if len(data) > f.maxSize {
			t.Errorf("%s: PDF too large (%d bytes, want ≤%d)", f.name, len(data), f.maxSize)
		}
	}
}
