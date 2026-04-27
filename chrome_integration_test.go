package htmlpdf_test

import (
	"os"
	"testing"

	"github.com/HeartBeat1608/htmlpdf"
)

func TestChromeBackendIntegration(t *testing.T) {
	chromeBin := os.Getenv("HTMLPDF_CHROME_BIN")
	if chromeBin == "" {
		t.Skip("set HTMLPDF_CHROME_BIN to run Chrome integration tests")
	}

	html := []byte(`<html><body><h1>Chrome backend</h1><p>integration</p></body></html>`)
	pdf, err := htmlpdf.Generate(html, htmlpdf.Options{
		Backend:    htmlpdf.BackendChrome,
		ChromePath: chromeBin,
	})
	if err != nil {
		t.Fatalf("chrome render failed: %v", err)
	}
	if len(pdf) == 0 {
		t.Fatal("chrome render returned empty PDF")
	}
}
